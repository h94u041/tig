package tig

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The ledger is what is known about intents and whose work follows them. Its
// records never change and name themselves by their hash; each is one file in
// .tig/ledger. There is no head: an intent is a chain of its own events, so
// two machines merge by putting their files together.
//
// What is true now is worked out from the events, never stored. The documents
// hold no state: a line's intents are found by the line's hash, and that index
// is worked out too.

const (
	intentEventHeader = "tig-intent-event 1"
	lineEventHeader   = "tig-line-event 1"
	linkEventHeader   = "tig-link-event 1"
	acceptanceHeader  = "tig-acceptance 1"
	ledgerStartHeader = "tig-ledger-start 1"
)

// Kinds of intent event.
const (
	IntentProposed   = "proposed"
	IntentConfirmed  = "confirmed"
	IntentRevised    = "revised"    // someone changed what it says
	IntentTouched    = "touched"    // an edit of its line reached its range
	IntentSplit      = "split"      // made out of part of another
	IntentComposed   = "composed"   // made out of others
	IntentSuperseded = "superseded" // another took its place
	IntentDeprecated = "deprecated"
	IntentRejected   = "rejected"
	IntentSpans      = "spans" // its text moved and did not change
)

// Who decided, and how it came to them.
const (
	ByHuman = "human"
	ByAI    = "ai"

	ViaDirect   = "direct"
	ViaSmooth   = "smooth"
	ViaAnalysis = "analysis"
	ViaBuilder  = "builder"
	ViaPlanner  = "planner"
	ViaMigrated = "migrated"
)

// Kinds of link between a work and an intent.
const (
	LinkImplements = "implements"
	LinkFixes      = "fixes"
	LinkRefactors  = "refactors"
	LinkTests      = "tests"
	LinkReverts    = "reverts"
	LinkUnlinked   = "unlinked" // an earlier link is withdrawn
)

// States of a work's acceptance.
const (
	WorkCandidate = "candidate"
	WorkAccepted  = "accepted"
	WorkRejected  = "rejected"
)

// Span is a range of a line, in code points from 0, leaving out its end. A span
// belongs to the line with that text; the same text elsewhere is told apart by N.
type Span struct {
	Line string `json:"line"` // the line's hash
	N    int    `json:"n"`    // which of the lines with this text, from 1
	From int    `json:"from"`
	To   int    `json:"to"`
}

func (s Span) encode() string {
	return s.Line + " " + strconv.Itoa(s.N) + " " + strconv.Itoa(s.From) + " " + strconv.Itoa(s.To)
}

func parseSpan(v string) (Span, error) {
	f := strings.Fields(v)
	if len(f) != 4 || !ValidHash(f[0]) {
		return Span{}, fmt.Errorf("tig: a span is not %q", v)
	}
	n, err1 := strconv.Atoi(f[1])
	from, err2 := strconv.Atoi(f[2])
	to, err3 := strconv.Atoi(f[3])
	if err := errors.Join(err1, err2, err3); err != nil || n < 1 || from < 0 || to < from {
		return Span{}, fmt.Errorf("tig: a span is not %q", v)
	}
	return Span{f[0], n, from, to}, nil
}

// IntentEvent is one thing that happened to an intent. The first of an intent
// has no Intent: its own hash is the intent's ID.
type IntentEvent struct {
	ID       string    `json:"id"`
	Intent   string    `json:"intent,omitempty"`
	Parent   string    `json:"parent,omitempty"`
	Kind     string    `json:"kind"`
	Revision int       `json:"revision"` // the intent's revision after this event
	Text     string    `json:"text,omitempty"`
	Spans    []Span    `json:"spans,omitempty"`
	From     []string  `json:"from,omitempty"` // the intents it came from, or what took its place
	By       string    `json:"by,omitempty"`
	Via      string    `json:"via,omitempty"`
	Actor    string    `json:"actor,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	At       time.Time `json:"at"`
}

func (e IntentEvent) encode() []byte {
	rec := newRecord(intentEventHeader)
	rec.set("intent", e.Intent)
	rec.set("parent", e.Parent)
	rec.set("kind", e.Kind)
	rec.set("revision", strconv.Itoa(e.Revision))
	rec.set("text", e.Text)
	for _, s := range e.Spans {
		rec.set("span", s.encode())
	}
	rec.setAll("from", e.From)
	rec.set("by", e.By)
	rec.set("via", e.Via)
	rec.set("actor", e.Actor)
	rec.set("reason", e.Reason)
	rec.setTime("at", e.At)
	return rec.bytes()
}

// LineEvent says a line came from another: the mark held the old hash when the
// text had become the new one.
type LineEvent struct {
	ID     string    `json:"id"`
	Line   string    `json:"line"`
	Parent string    `json:"parent,omitempty"`
	At     time.Time `json:"at"`
}

// LinkEvent says a work carries out, or stopped carrying out, a revision of an
// intent. Work is the commit's hash; a mark converted from the old format can
// name a commit that was never there, which the check reports.
type LinkEvent struct {
	ID       string    `json:"id"`
	Work     string    `json:"work"`
	Intent   string    `json:"intent"`
	Revision int       `json:"revision"`
	Kind     string    `json:"kind"`
	Actor    string    `json:"actor,omitempty"`
	At       time.Time `json:"at"`
}

// Acceptance is someone's decision on a work. tig only records it: whoever
// takes the work judges it, by whatever means they have.
type Acceptance struct {
	ID     string    `json:"id"`
	Work   string    `json:"work"`
	State  string    `json:"state"` // accepted or rejected
	By     string    `json:"by,omitempty"`
	Via    string    `json:"via,omitempty"`
	Actor  string    `json:"actor,omitempty"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

// Intent is what its events add up to.
type Intent struct {
	ID       string `json:"id"`
	Short    string `json:"short"` // with its "i"
	Text     string `json:"text"`
	State    string `json:"state"`    // proposed, confirmed, superseded, deprecated or rejected
	Revision int    `json:"revision"` // goes up when what it says changes
	// Review: its text changed after it was confirmed, or by someone other than
	// a person, and nobody has confirmed it since.
	Review bool   `json:"review,omitempty"`
	Spans  []Span `json:"spans,omitempty"`
	// By, Via and Actor are who last decided on it.
	By    string        `json:"by,omitempty"`
	Via   string        `json:"via,omitempty"`
	Actor string        `json:"actor,omitempty"`
	From  []string      `json:"from,omitempty"`
	Log   []IntentEvent `json:"log,omitempty"` // oldest first
}

// Active reports whether the intent still stands.
func (i *Intent) Active() bool { return i.State == IntentProposed || i.State == IntentConfirmed }

// RevisionEvent is the event that left the intent at a revision.
func (i *Intent) RevisionEvent(n int) (IntentEvent, bool) {
	var out IntentEvent
	found := false
	for _, e := range i.Log {
		if e.Revision == n && e.Text != "" {
			out, found = e, true
		}
	}
	return out, found
}

func (i *Intent) apply(e IntentEvent) {
	i.Log = append(i.Log, e)
	if e.Revision > i.Revision {
		i.Revision = e.Revision
	}
	if e.Text != "" {
		i.Text = e.Text
	}
	if e.Spans != nil {
		i.Spans = e.Spans
	}
	switch e.Kind {
	case IntentProposed, IntentSplit, IntentComposed:
		i.State, i.By, i.Via, i.Actor = IntentProposed, e.By, e.Via, e.Actor
		i.From = e.From
	case IntentConfirmed:
		i.State, i.By, i.Via, i.Actor, i.Review = IntentConfirmed, e.By, e.Via, e.Actor, false
	case IntentRevised:
		i.Review = i.State == IntentConfirmed && e.By != ByHuman
	case IntentTouched:
		i.Review = i.State == IntentConfirmed
	case IntentSuperseded, IntentDeprecated, IntentRejected:
		i.State, i.By, i.Via, i.Actor = e.Kind, e.By, e.Via, e.Actor
		if e.Kind == IntentSuperseded {
			i.From = e.From
		}
		i.Review = false
	}
}

// Ledger is every record, read.
type Ledger struct {
	intents map[string]*Intent
	order   []*Intent // oldest first
	byShort map[string]*Intent
	events  map[string]IntentEvent
	links   []LinkEvent // oldest first
	lines   map[string]bool
	parents map[string]string // line → the line it came from
	accepts map[string]Acceptance
	start   time.Time // when the ledger began; zero when it has not
}

func (r *Repo) ledgerDir() string {
	if r.Dir == "" {
		return ""
	}
	if r.LedgerDir != "" {
		return r.LedgerDir
	}
	return filepath.Join(r.Dir, "ledger")
}

// Ledger reads the ledger. A .tig can come from someone else, so a record whose
// name is not its hash, or which does not read, is an error.
func (r *Repo) Ledger(ctx context.Context) (*Ledger, error) {
	l := &Ledger{intents: map[string]*Intent{}, byShort: map[string]*Intent{}, events: map[string]IntentEvent{},
		lines: map[string]bool{}, parents: map[string]string{}, accepts: map[string]Acceptance{}}
	dir := r.ledgerDir()
	if dir == "" {
		return l, nil
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return nil, err
	}
	accepts := []Acceptance{}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		id := entry.Name()
		if entry.IsDir() || !ValidHash(id) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, id))
		if err != nil {
			return nil, err
		}
		if HashOf(raw) != id {
			return nil, fmt.Errorf("tig: the ledger record %s is not what its name says", id)
		}
		header, _, _ := strings.Cut(string(raw), "\n")
		switch header {
		case intentEventHeader:
			e, err := parseIntentEvent(id, raw)
			if err != nil {
				return nil, err
			}
			l.events[id] = e
			for _, s := range e.Spans {
				l.lines[s.Line] = true
			}
		case lineEventHeader:
			rec, err := parseRecord(raw, header)
			if err != nil {
				return nil, err
			}
			if err := hashesValid(id, nil, rec.get("line"), rec.get("parent")); err != nil {
				return nil, err
			}
			l.lines[rec.get("line")] = true
			if p := rec.get("parent"); p != "" {
				l.lines[p] = true
				l.parents[rec.get("line")] = p
			}
		case linkEventHeader:
			rec, err := parseRecord(raw, header)
			if err != nil {
				return nil, err
			}
			rev, _ := strconv.Atoi(rec.get("revision"))
			l.links = append(l.links, LinkEvent{ID: id, Work: rec.get("work"), Intent: rec.get("intent"), Revision: rev, Kind: rec.get("kind"), Actor: rec.get("actor"), At: rec.time("at")})
		case acceptanceHeader:
			rec, err := parseRecord(raw, header)
			if err != nil {
				return nil, err
			}
			accepts = append(accepts, Acceptance{ID: id, Work: rec.get("work"), State: rec.get("state"), By: rec.get("by"), Via: rec.get("via"), Actor: rec.get("actor"), Reason: rec.get("reason"), At: rec.time("at")})
		case ledgerStartHeader:
			rec, err := parseRecord(raw, header)
			if err != nil {
				return nil, err
			}
			if at := rec.time("at"); l.start.IsZero() || at.Before(l.start) {
				l.start = at
			}
		}
	}
	slices.SortFunc(l.links, func(a, b LinkEvent) int { return laterFirst(a.At, b.At, a.ID, b.ID) })
	slices.SortFunc(accepts, func(a, b Acceptance) int { return laterFirst(a.At, b.At, a.ID, b.ID) })
	for _, a := range accepts {
		l.accepts[a.Work] = a
	}
	l.fold()
	return l, nil
}

func laterFirst(a, b time.Time, x, y string) int {
	if c := a.Compare(b); c != 0 {
		return c
	}
	return strings.Compare(x, y)
}

func parseIntentEvent(id string, raw []byte) (IntentEvent, error) {
	rec, err := parseRecord(raw, intentEventHeader)
	if err != nil {
		return IntentEvent{}, err
	}
	rev, _ := strconv.Atoi(rec.get("revision"))
	e := IntentEvent{ID: id, Intent: rec.get("intent"), Parent: rec.get("parent"), Kind: rec.get("kind"), Revision: rev, Text: rec.get("text"),
		From: rec.all("from"), By: rec.get("by"), Via: rec.get("via"), Actor: rec.get("actor"), Reason: rec.get("reason"), At: rec.time("at")}
	if err := hashesValid(id, nil, e.Intent, e.Parent); err != nil {
		return IntentEvent{}, err
	}
	for _, v := range rec.all("span") {
		s, err := parseSpan(v)
		if err != nil {
			return IntentEvent{}, fmt.Errorf("%w (in %s)", err, id)
		}
		e.Spans = append(e.Spans, s)
	}
	return e, nil
}

// fold adds up the events. Within an intent an event follows its parent; two
// events on one parent (two machines) go in the order of their time, then
// their hash, so everyone gets the same answer.
func (l *Ledger) fold() {
	children := map[string][]IntentEvent{}
	var firsts []IntentEvent
	byIntent := map[string][]IntentEvent{}
	for _, e := range l.events {
		if e.Intent == "" {
			firsts = append(firsts, e)
			continue
		}
		byIntent[e.Intent] = append(byIntent[e.Intent], e)
		children[e.Parent] = append(children[e.Parent], e)
	}
	byTime := func(a, b IntentEvent) int { return laterFirst(a.At, b.At, a.ID, b.ID) }
	slices.SortFunc(firsts, byTime)
	for k := range children {
		slices.SortFunc(children[k], byTime)
	}
	for _, first := range firsts {
		i := &Intent{ID: first.ID}
		seen := map[string]bool{}
		var walk func(e IntentEvent)
		walk = func(e IntentEvent) {
			if seen[e.ID] {
				return
			}
			seen[e.ID] = true
			i.apply(e)
			for _, k := range children[e.ID] {
				walk(k)
			}
		}
		walk(first)
		// An event whose parent has not arrived yet still counts, last.
		rest := byIntent[first.ID]
		slices.SortFunc(rest, byTime)
		for _, e := range rest {
			walk(e)
		}
		n := minIntentShort
		for n < len(i.ID) && l.byShort["i"+i.ID[:n]] != nil {
			n++
		}
		i.Short = "i" + i.ID[:n]
		l.byShort[i.Short] = i
		l.intents[i.ID] = i
		l.order = append(l.order, i)
	}
}

// Intents is every intent, oldest first.
func (l *Ledger) Intents() []*Intent { return slices.Clone(l.order) }

// Intent finds an intent by its short ID, its hash, or a unique start of it.
func (l *Ledger) Intent(ref string) (*Intent, error) {
	code := strings.TrimPrefix(ref, "i")
	if r, err := ParseRef("i" + code); err == nil {
		code = r.Code
	}
	if i := l.byShort["i"+code]; i != nil {
		return i, nil
	}
	var found *Intent
	for id, i := range l.intents {
		if strings.HasPrefix(id, code) {
			if found != nil {
				return nil, fmt.Errorf("tig: %q matches more than one intent", ref)
			}
			found = i
		}
	}
	if found == nil {
		return nil, fmt.Errorf("tig: no intent %q", ref)
	}
	return found, nil
}

// on is the standing intents with a span on a line.
func (l *Ledger) on(line string, n int) []*Intent {
	var out []*Intent
	for _, i := range l.order {
		if !i.Active() {
			continue
		}
		for _, s := range i.Spans {
			if s.Line == line && s.N == n {
				out = append(out, i)
				break
			}
		}
	}
	return out
}

// resolveLine finds the whole hash of a line the ledger knows by the start of it.
func (l *Ledger) resolveLine(short string) (string, bool) {
	if l.lines[short] {
		return short, true
	}
	found := ""
	for h := range l.lines {
		if strings.HasPrefix(h, short) {
			if found != "" {
				return "", false
			}
			found = h
		}
	}
	return found, found != ""
}

// shortLine is what a mark writes for a line: at least 6 characters, as many as
// it takes not to be another line the ledger knows.
func (l *Ledger) shortLine(hash string) string {
	n := minLineShort
	for n < len(hash) {
		clash := false
		for h := range l.lines {
			if h != hash && strings.HasPrefix(h, hash[:n]) {
				clash = true
				break
			}
		}
		if !clash {
			break
		}
		n++
	}
	return hash[:n]
}

// LinksOf is the links standing on an intent: per work, the last one said,
// unless that withdrew it. Oldest first.
func (l *Ledger) LinksOf(intent string) []LinkEvent {
	latest := map[string]LinkEvent{}
	var works []string
	for _, e := range l.links {
		if e.Intent != intent {
			continue
		}
		if _, ok := latest[e.Work]; !ok {
			works = append(works, e.Work)
		}
		latest[e.Work] = e
	}
	var out []LinkEvent
	for _, w := range works {
		if latest[w].Kind != LinkUnlinked {
			out = append(out, latest[w])
		}
	}
	return out
}

// AcceptanceOf says where a work stands. A work with no decision is a candidate,
// except for what never needs one: a work older than the ledger, which nobody
// could have decided on, and the records that build nothing of their own.
func (l *Ledger) AcceptanceOf(w Work) string {
	if a, ok := l.accepts[w.ID]; ok {
		return a.State
	}
	switch {
	case l.start.IsZero(), w.At.Before(l.start), w.Result == ResultBaseline, w.Result == ResultFolded:
		return WorkAccepted
	}
	return WorkCandidate
}

// Acceptance is the decision on a work, if there is one.
func (l *Ledger) Acceptance(work string) (Acceptance, bool) {
	a, ok := l.accepts[work]
	return a, ok
}

// put writes a ledger record and returns its hash.
func (r *Repo) putLedger(data []byte) (string, error) {
	dir := r.ledgerDir()
	if dir == "" {
		return "", errors.New("tig: the ledger needs a .tig folder")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	id := HashOf(data)
	full := filepath.Join(dir, id)
	if _, err := os.Stat(full); err == nil {
		return id, nil
	}
	return id, writeAtomic(full, data, 0o644)
}

// startLedger marks when the ledger began, once: works older than this are
// taken as accepted.
func (r *Repo) startLedger(l *Ledger) error {
	if l != nil && !l.start.IsZero() {
		return nil
	}
	rec := newRecord(ledgerStartHeader)
	at := time.Now().UTC()
	rec.setTime("at", at)
	if _, err := r.putLedger(rec.bytes()); err != nil {
		return err
	}
	if l != nil {
		l.start = at
	}
	return nil
}

// addEvent writes an intent event on top of the intent's last one and folds it in.
func (r *Repo) addEvent(l *Ledger, e IntentEvent) (*Intent, error) {
	if err := r.startLedger(l); err != nil {
		return nil, err
	}
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	var i *Intent
	if e.Intent != "" {
		if i = l.intents[e.Intent]; i == nil {
			return nil, fmt.Errorf("tig: no intent %s", e.Intent)
		}
		e.Parent = i.Log[len(i.Log)-1].ID
		if e.Revision == 0 {
			e.Revision = i.Revision
		}
	} else if e.Revision == 0 {
		e.Revision = 1
	}
	id, err := r.putLedger(e.encode())
	if err != nil {
		return nil, err
	}
	e.ID = id
	l.events[id] = e
	for _, s := range e.Spans {
		l.lines[s.Line] = true
	}
	if i == nil {
		i = &Intent{ID: id}
		n := minIntentShort
		for n < len(id) && l.byShort["i"+id[:n]] != nil {
			n++
		}
		i.Short = "i" + id[:n]
		l.byShort[i.Short] = i
		l.intents[id] = i
		l.order = append(l.order, i)
	}
	i.apply(e)
	return i, nil
}

func (r *Repo) addLineEvent(l *Ledger, line, parent string) error {
	if err := r.startLedger(l); err != nil {
		return err
	}
	rec := newRecord(lineEventHeader)
	rec.set("line", line)
	rec.set("parent", parent)
	rec.setTime("at", time.Now().UTC())
	if _, err := r.putLedger(rec.bytes()); err != nil {
		return err
	}
	l.lines[line] = true
	if parent != "" {
		l.parents[line] = parent
	}
	return nil
}

func (r *Repo) addLink(l *Ledger, e LinkEvent) error {
	if err := r.startLedger(l); err != nil {
		return err
	}
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	rec := newRecord(linkEventHeader)
	rec.set("work", e.Work)
	rec.set("intent", e.Intent)
	rec.set("revision", strconv.Itoa(e.Revision))
	rec.set("kind", e.Kind)
	rec.set("actor", e.Actor)
	rec.setTime("at", e.At)
	id, err := r.putLedger(rec.bytes())
	if err != nil {
		return err
	}
	e.ID = id
	l.links = append(l.links, e)
	return nil
}

// AcceptInput is a decision on a work.
type AcceptInput struct {
	Work   string // short ID, hash or unique prefix
	State  string // WorkAccepted or WorkRejected
	By     string
	Via    string
	Actor  string
	Reason string
}

// Accept records a decision on a work. It judges nothing.
func (r *Repo) Accept(ctx context.Context, in AcceptInput) (Acceptance, error) {
	if in.State != WorkAccepted && in.State != WorkRejected {
		return Acceptance{}, fmt.Errorf("tig: a work is accepted or rejected, not %q", in.State)
	}
	commits, err := r.Commits(ctx)
	if err != nil {
		return Acceptance{}, err
	}
	w, err := commits.Resolve(in.Work)
	if err != nil {
		return Acceptance{}, err
	}
	l, err := r.Ledger(ctx)
	if err != nil {
		return Acceptance{}, err
	}
	if err := r.startLedger(l); err != nil {
		return Acceptance{}, err
	}
	a := Acceptance{Work: w.ID, State: in.State, By: in.By, Via: in.Via, Actor: in.Actor, Reason: in.Reason, At: time.Now().UTC()}
	rec := newRecord(acceptanceHeader)
	rec.set("work", a.Work)
	rec.set("state", a.State)
	rec.set("by", a.By)
	rec.set("via", a.Via)
	rec.set("actor", a.Actor)
	rec.set("reason", a.Reason)
	rec.setTime("at", a.At)
	a.ID, err = r.putLedger(rec.bytes())
	return a, err
}
