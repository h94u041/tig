package tig

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Marking says what a line means and who followed it: the document gets the
// line's mark, the ledger gets the intent and the links. A mark is not design:
// the block's text, and so its hash, holds the plain line.

// Mark is a line of a document that has work on it.
type Mark struct {
	Path  string   `json:"path"`
	Block string   `json:"block"`
	Line  int      `json:"line"` // 1-based
	Text  string   `json:"text"` // the line's text
	Hash  string   `json:"hash"` // the line's hash
	N     int      `json:"n"`    // which of the lines with this text
	IDs   []string `json:"ids"`  // commits, by short ID; a fold stands for what it folds
}

// legacyMark is the format before line marks, **sentence *a1b2***: read only to
// convert it.
var legacyMark = regexp.MustCompile(`\*\*((?:[^*\n]|\*[^*\n])+?)[ \t]+\*([0-9a-f]{4,64}(?:[ \t]+[0-9a-f]{4,64})*)\*\*\*`)

// maskCode blanks a line's inline code without moving anything in it, so a
// mark quoted as code is not read as a mark.
func maskCode(line string) string {
	return codeSpan.ReplaceAllStringFunc(line, func(s string) string { return strings.Repeat(" ", len(s)) })
}

func stripLegacyMarks(line string) string {
	locs := legacyMark.FindAllStringSubmatchIndex(maskCode(line), -1)
	if locs == nil {
		return line
	}
	var b strings.Builder
	end := 0
	for _, l := range locs {
		b.WriteString(line[end:l[0]])
		b.WriteString(line[l[2]:l[3]])
		end = l[1]
	}
	b.WriteString(line[end:])
	return b.String()
}

// sentenceKey names a sentence: the same words in the same order, however
// they were spaced or wrapped.
func sentenceKey(text string) string { return HashOf([]byte(strings.Join(strings.Fields(text), " "))) }

// MarkInput says what to mark.
type MarkInput struct {
	// Path is the document; the main document when empty.
	Path string
	// Sentence is one line of text. A line saying just that is the one marked;
	// otherwise the first line holding it, for that range. A document with
	// neither gets it as a new line.
	Sentence string
	// Under is the title of the section a new line goes under; a section is
	// added if the document has none by that title, and with no title the line
	// goes at the end.
	Under string
	// IDs are the commits to link, by short ID, hash or unique start. There may
	// be none: an intent nothing carries out yet.
	IDs []string
	// LinkKind is how they follow it; implements when empty.
	LinkKind string
	// Inferred: the sentence is someone's guess. If the document does not say it
	// already, it goes into the inferred document until a person agrees. If it
	// does, the words are the document's and the guess is only that these commits
	// follow them: nobody is taken to have confirmed anything.
	Inferred bool
	// State is what a new intent starts as: IntentConfirmed, or proposed when empty.
	State string
	// Link is what the links start as: LinkConfirmed or LinkProposed. Empty means
	// confirmed when a person says it (By is ByHuman), proposed otherwise.
	Link  string
	By    string
	Via   string
	Actor string
	// Reason says why, on the links.
	Reason string
	// Line, when above 0, is the 1-based line meant, and Sentence is what it says
	// or part of it. It tells apart lines that say the same thing. If the line no
	// longer says that, the sentence is looked for as usual.
	Line int
}

func fencedLines(lines []string) []bool {
	fenced := make([]bool, len(lines))
	var fenceMark string
	for i, line := range lines {
		m := fenceLine.FindStringSubmatch(strings.TrimRight(line, "\r"))
		switch {
		case fenceMark != "":
			fenced[i] = true
			rest := ""
			if m != nil {
				rest = strings.TrimRight(line, "\r")[strings.Index(line, m[1])+len(m[1]):]
			}
			if m != nil && m[1][0] == fenceMark[0] && len(m[1]) >= len(fenceMark) && strings.TrimSpace(rest) == "" {
				fenceMark = ""
			}
		case m != nil:
			fenced[i], fenceMark = true, m[1]
		}
	}
	return fenced
}

// isHeading reports whether a line is a heading, which is a block's name and
// not a sentence of it.
func isHeading(line string) bool { return headingLine.MatchString(strings.TrimRight(line, "\r")) }

// locate finds a sentence in a document: the line saying just that, or else the
// first holding it, with the range it is in the line's text.
func locate(lines []string, fenced []bool, sentence string) (at, from, to int, ok bool) {
	want := LineText(sentence)
	for i, line := range lines {
		if !fenced[i] && !isHeading(line) && isTrackedLine(line) && LineText(line) == want {
			return i, 0, len([]rune(want)), true
		}
	}
	for i, line := range lines {
		if fenced[i] || isHeading(line) || !isTrackedLine(line) {
			continue
		}
		text := LineText(line)
		if byteAt := strings.Index(text, want); byteAt >= 0 {
			from = len([]rune(text[:byteAt]))
			return i, from, from + len([]rune(want)), true
		}
	}
	return 0, 0, 0, false
}

// locateAt is locate, except that a line asked for by number is the one found if
// it still says that.
func locateAt(lines []string, fenced []bool, sentence string, line int) (at, from, to int, ok bool) {
	if i := line - 1; i >= 0 && i < len(lines) && !fenced[i] && !isHeading(lines[i]) && isTrackedLine(lines[i]) {
		text, want := LineText(lines[i]), LineText(sentence)
		if text == want {
			return i, 0, len([]rune(want)), true
		}
		if byteAt := strings.Index(text, want); byteAt >= 0 {
			from = len([]rune(text[:byteAt]))
			return i, from, from + len([]rune(want)), true
		}
	}
	return locate(lines, fenced, sentence)
}

func checkSentence(sentence string) (string, error) {
	s := strings.TrimSpace(stripMarks(sentence))
	if s == "" || strings.ContainsAny(s, "\r\n") {
		return "", fmt.Errorf("tig: a sentence is one line of text: %q", sentence)
	}
	return s, nil
}

// AddSentence writes a sentence with no commit yet: design, not built. A
// sentence the document already has stays as it is. It goes under the section
// titled under, made at the end when there is none.
func AddSentence(path string, content []byte, sentence, under string) ([]byte, error) {
	sentence, err := checkSentence(sentence)
	if err != nil {
		return nil, err
	}
	for _, b := range ParseBlocks(path, content) {
		for _, s := range b.Sentences {
			if sentenceKey(s.Text) == sentenceKey(sentence) {
				return content, nil
			}
		}
	}
	return insertLine(path, content, sentence, under), nil
}

func eolOf(content []byte) string {
	if bytes.Contains(content, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func insertLine(path string, content []byte, text, under string) []byte {
	eol := eolOf(content)
	lines := strings.Split(string(content), "\n")
	if under != "" {
		for _, b := range ParseBlocks(path, content) {
			if b.Level == 0 || b.Title != under {
				continue
			}
			at := min(b.OwnEnd, len(lines))
			for at > b.Start && strings.TrimSpace(lines[at-1]) == "" {
				at--
			}
			add := []string{"" + strings.TrimSuffix(eol, "\n"), text + strings.TrimSuffix(eol, "\n")}
			if at == len(lines) {
				add = append(add, "")
			} else if strings.TrimSpace(lines[at]) != "" {
				add = append(add, strings.TrimSuffix(eol, "\n"))
			}
			return []byte(strings.Join(slices.Insert(slices.Clone(lines), at, add...), "\n"))
		}
	}
	out := strings.TrimRight(string(content), "\r\n \t")
	if out != "" {
		out += eol + eol
	}
	if under != "" {
		out += "## " + under + eol + eol
	}
	return []byte(out + text + eol)
}

// trackedDocs are the documents whose lines the ledger follows: the chain, the
// inferred document, and any named.
func (r *Repo) trackedDocs(ctx context.Context, also ...string) ([]string, error) {
	chain, err := r.Chain(ctx, "")
	if err != nil {
		return nil, err
	}
	docs := slices.Clone(chain.Docs)
	if chain.Inferred != "" {
		docs = appendNew(docs, chain.Inferred)
	}
	for _, d := range also {
		docs = appendNew(docs, d)
	}
	return docs, nil
}

func (r *Repo) readDoc(rel string) (string, []byte, error) {
	full, err := r.safePath(rel)
	if err != nil {
		return "", nil, err
	}
	data, err := os.ReadFile(full)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return full, nil, err
	}
	return full, data, nil
}

func writeDoc(full string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return writeAtomic(full, data, 0o644)
}

// spanKey is one line with its text told apart from the same text elsewhere.
type spanKey struct {
	line string
	n    int
}

// usedLines is what the marks in the documents hold, for a new mark to keep clear of.
func (r *Repo) usedLines(docs []string) map[spanKey]bool {
	used := map[spanKey]bool{}
	for _, doc := range docs {
		_, data, err := r.readDoc(doc)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		fenced := fencedLines(lines)
		for i, line := range lines {
			if fenced[i] {
				continue
			}
			if m, ok := readLineMark(line); ok {
				if h := LineHash(LineText(line)); strings.HasPrefix(h, m.Hash) {
					used[spanKey{h, m.N}] = true
				}
			}
		}
	}
	return used
}

func nextFree(used map[spanKey]bool, hash string) int {
	n := 1
	for used[spanKey{hash, n}] {
		n++
	}
	used[spanKey{hash, n}] = true
	return n
}

// SyncResult says what syncing found.
type SyncResult struct {
	// Docs are the documents whose marks were rewritten.
	Docs []string `json:"docs,omitempty"`
	// Touched are the intents an edit reached, by short ID.
	Touched []string `json:"touched,omitempty"`
	// Moved are the intents whose text moved and did not change.
	Moved []string `json:"moved,omitempty"`
	// Migrated is how many marks in the old format became line marks.
	Migrated int `json:"migrated,omitempty"`
}

// Sync brings the ledger up to the documents: lines edited since the last time
// get their new hash, and the intents on them follow. It writes.
func (r *Repo) Sync(ctx context.Context, actor string) (SyncResult, error) {
	if r.Dir == "" {
		return SyncResult{}, errors.New("tig: syncing needs a .tig folder")
	}
	unlock, err := lockRepo(ctx, r.Dir)
	if err != nil {
		return SyncResult{}, err
	}
	defer unlock()
	res, _, err := r.syncLocked(ctx, actor)
	return res, err
}

// lineMove is a line whose text changed.
type lineMove struct {
	to               spanKey
	oldText, newText string
}

func (r *Repo) syncLocked(ctx context.Context, actor string, also ...string) (SyncResult, *Ledger, error) {
	var res SyncResult
	l, err := r.Ledger(ctx)
	if err != nil {
		return res, nil, err
	}
	docs, err := r.trackedDocs(ctx, also...)
	if err != nil {
		return res, nil, err
	}
	used := r.usedLines(docs)
	seen := map[spanKey]bool{}
	moves := map[spanKey]lineMove{}
	for _, doc := range docs {
		full, data, err := r.readDoc(doc)
		if err != nil {
			return res, nil, err
		}
		if len(data) == 0 {
			continue
		}
		lines := strings.Split(string(data), "\n")
		fenced := fencedLines(lines)
		changed := false
		for i := range lines {
			if fenced[i] {
				continue
			}
			line := lines[i]
			if legacyMark.MatchString(maskCode(line)) {
				n, next, err := r.migrateLine(ctx, l, used, line, actor)
				if err != nil {
					return res, nil, err
				}
				res.Migrated += n
				lines[i], changed = next, true
				seen[spanKey{LineHash(LineText(next)), mustMark(next).N}] = true
				continue
			}
			text := LineText(line)
			hash := LineHash(text)
			m, marked := readLineMark(line)
			switch {
			case !marked:
				// The mark was removed by hand: put it back if the text is known.
				key := spanKey{hash, 1}
				if isTrackedLine(line) && !isHeading(line) && !seen[key] && len(l.on(hash, 1)) > 0 {
					seen[key], used[key] = true, true
					lines[i], changed = setLineMark(line, LineMark{Hash: l.shortLine(hash), N: 1}), true
				}
			case text == "":
				lines[i], changed = stripLineMarks(line), true
			case strings.HasPrefix(hash, m.Hash):
				key := spanKey{hash, m.N}
				if seen[key] {
					// The line was copied: the copy is a line of its own.
					n := nextFree(used, hash)
					seen[spanKey{hash, n}] = true
					lines[i], changed = setLineMark(line, LineMark{Hash: m.Hash, N: n}), true
					continue
				}
				seen[key] = true
			default:
				old, ok := l.resolveLine(m.Hash)
				if !ok {
					continue // the check reports it: nothing can be carried
				}
				if _, err := r.Objects.PutObject(ctx, []byte(text)); err != nil {
					return res, nil, err
				}
				n := nextFree(used, hash)
				if err := r.addLineEvent(l, hash, old); err != nil {
					return res, nil, err
				}
				oldText, _ := r.Objects.GetObject(ctx, old)
				moves[spanKey{old, m.N}] = lineMove{to: spanKey{hash, n}, oldText: string(oldText), newText: text}
				seen[spanKey{hash, n}] = true
				lines[i], changed = setLineMark(line, LineMark{Hash: l.shortLine(hash), N: n}), true
			}
		}
		if changed {
			if err := writeDoc(full, []byte(strings.Join(lines, "\n"))); err != nil {
				return res, nil, err
			}
			res.Docs = append(res.Docs, doc)
		}
	}
	if err := r.followMoves(ctx, l, moves, actor, &res); err != nil {
		return res, nil, err
	}
	return res, l, nil
}

func mustMark(line string) LineMark {
	m, _ := readLineMark(line)
	return m
}

// followMoves carries the intents on lines whose text changed.
func (r *Repo) followMoves(ctx context.Context, l *Ledger, moves map[spanKey]lineMove, actor string, res *SyncResult) error {
	if len(moves) == 0 {
		return nil
	}
	for _, i := range l.Intents() {
		if !i.Active() {
			continue
		}
		spans := slices.Clone(i.Spans)
		moved, touched := false, false
		for j, s := range spans {
			mv, ok := moves[spanKey{s.Line, s.N}]
			if !ok {
				continue
			}
			moved = true
			var t bool
			var from, to int
			if mv.oldText == "" {
				from, to, t = 0, len([]rune(mv.newText)), true
			} else {
				from, to, t = carry(mv.oldText, mv.newText, s.From, s.To)
			}
			spans[j] = Span{Line: mv.to.line, N: mv.to.n, From: from, To: to}
			touched = touched || t
		}
		if !moved {
			continue
		}
		e := IntentEvent{Intent: i.ID, Kind: IntentSpans, Spans: spans, Actor: actor}
		if touched {
			e.Kind, e.Revision, e.Text = IntentTouched, i.Revision+1, r.textOf(ctx, spans)
			res.Touched = append(res.Touched, i.Short)
		} else {
			res.Moved = append(res.Moved, i.Short)
		}
		if _, err := r.addEvent(l, e); err != nil {
			return err
		}
	}
	return nil
}

// textOf is what spans say, joined when they are on several lines.
func (r *Repo) textOf(ctx context.Context, spans []Span) string {
	var parts []string
	for _, s := range spans {
		data, err := r.Objects.GetObject(ctx, s.Line)
		if err != nil {
			continue
		}
		if t := sliceRunes(string(data), s.From, s.To); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

// migrateLine turns the old marks of a line into a line mark and intents. Where
// they came from is not known, so they are not taken as confirmed: whoever runs
// the project is asked.
func (r *Repo) migrateLine(ctx context.Context, l *Ledger, used map[spanKey]bool, line, actor string) (int, string, error) {
	commits, err := r.Commits(ctx)
	if err != nil {
		return 0, line, err
	}
	locs := legacyMark.FindAllStringSubmatchIndex(maskCode(line), -1)
	plain := stripLegacyMarks(line)
	text := LineText(plain)
	hash := LineHash(text)
	if _, err := r.Objects.PutObject(ctx, []byte(text)); err != nil {
		return 0, line, err
	}
	if err := r.addLineEvent(l, hash, ""); err != nil {
		return 0, line, err
	}
	n := nextFree(used, hash)
	runes := []rune(text)
	cursor := 0
	for _, loc := range locs {
		phrase := line[loc[2]:loc[3]]
		from := runeIndex(runes, []rune(phrase), cursor)
		to := from + len([]rune(phrase))
		if from < 0 {
			from, to = 0, len(runes)
		}
		cursor = to
		// The same old mark put back later is the intent it already became.
		span := Span{hash, n, from, to}
		var i *Intent
		for _, have := range l.on(hash, n) {
			if len(have.Spans) == 1 && have.Spans[0] == span {
				i = have
				break
			}
		}
		if i == nil {
			if i, err = r.addEvent(l, IntentEvent{Kind: IntentProposed, Text: phrase, Spans: []Span{span}, Via: ViaMigrated, Actor: actor}); err != nil {
				return 0, line, err
			}
		}
		// The old mark was how a commit was bound: the binding stands as it did.
		for _, id := range strings.Fields(line[loc[4]:loc[5]]) {
			work := id
			if w, err := commits.Resolve(id); err == nil {
				work = w.ID
			}
			if err := r.addLink(l, LinkEvent{Work: work, Intent: i.ID, Revision: i.Revision, Kind: LinkImplements, State: LinkConfirmed, Via: ViaMigrated, Actor: actor}); err != nil {
				return 0, line, err
			}
		}
	}
	return len(locs), setLineMark(plain, LineMark{Hash: l.shortLine(hash), N: n}), nil
}

func runeIndex(text, part []rune, from int) int {
	for i := from; i+len(part) <= len(text); i++ {
		if slices.Equal(text[i:i+len(part)], part) {
			return i
		}
	}
	return -1
}

// Mark writes down what a sentence means and which commits follow it, and
// returns the line as it stands. Several writers can do this at once: it holds
// the lock.
func (r *Repo) Mark(ctx context.Context, in MarkInput) (Mark, error) {
	if r.Dir == "" {
		return Mark{}, errors.New("tig: marking needs a .tig folder")
	}
	sentence, err := checkSentence(in.Sentence)
	if err != nil {
		return Mark{}, err
	}
	commits, err := r.Commits(ctx)
	if err != nil {
		return Mark{}, err
	}
	var works []string
	for _, ref := range in.IDs {
		w, err := commits.Resolve(ref)
		if err != nil {
			return Mark{}, err
		}
		works = appendNew(works, w.ID)
	}
	cfg, err := r.Config()
	if err != nil {
		return Mark{}, err
	}
	if in.Path == "" {
		in.Path = cfg.Root
	}
	if !IsMarkdown(in.Path) {
		return Mark{}, fmt.Errorf("tig: %s is not a Markdown document", in.Path)
	}
	if _, err := r.safePath(in.Path); err != nil {
		return Mark{}, err
	}
	unlock, err := lockRepo(ctx, r.Dir)
	if err != nil {
		return Mark{}, err
	}
	defer unlock()
	_, l, err := r.syncLocked(ctx, in.Actor, in.Path)
	if err != nil {
		return Mark{}, err
	}
	if in.Inferred {
		_, data, err := r.readDoc(in.Path)
		if err != nil {
			return Mark{}, err
		}
		lines := strings.Split(string(data), "\n")
		if _, _, _, found := locate(lines, fencedLines(lines), sentence); !found {
			in.Path = cfg.InferredDoc()
		}
	}
	full, data, err := r.readDoc(in.Path)
	if err != nil {
		return Mark{}, err
	}
	lines := strings.Split(string(data), "\n")
	at, from, to, found := locateAt(lines, fencedLines(lines), sentence, in.Line)
	if !found {
		data = insertLine(in.Path, data, sentence, in.Under)
		lines = strings.Split(string(data), "\n")
		if at, from, to, found = locate(lines, fencedLines(lines), sentence); !found {
			return Mark{}, fmt.Errorf("tig: the line %q did not read back", sentence)
		}
	}
	text := LineText(lines[at])
	hash := LineHash(text)
	if _, err := r.Objects.PutObject(ctx, []byte(text)); err != nil {
		return Mark{}, err
	}
	if !l.lines[hash] {
		if err := r.addLineEvent(l, hash, ""); err != nil {
			return Mark{}, err
		}
	}
	m, marked := readLineMark(lines[at])
	n := m.N
	if !marked {
		docs, err := r.trackedDocs(ctx, in.Path)
		if err != nil {
			return Mark{}, err
		}
		n = nextFree(r.usedLines(docs), hash)
		m = LineMark{Hash: l.shortLine(hash), N: n}
		lines[at] = setLineMark(lines[at], m)
		data = []byte(strings.Join(lines, "\n"))
	}
	if !marked || !found {
		if err := writeDoc(full, data); err != nil {
			return Mark{}, err
		}
	} else if _, err := os.Stat(full); err != nil {
		if err := writeDoc(full, data); err != nil {
			return Mark{}, err
		}
	}
	span := Span{Line: hash, N: n, From: from, To: to}
	var i *Intent
	for _, have := range l.on(hash, n) {
		if len(have.Spans) == 1 && have.Spans[0] == span {
			i = have
			break
		}
	}
	if i == nil {
		if i, err = r.addEvent(l, IntentEvent{Kind: IntentProposed, Text: sliceRunes(text, from, to), Spans: []Span{span}, By: in.By, Via: in.Via, Actor: in.Actor}); err != nil {
			return Mark{}, err
		}
	}
	if in.State == IntentConfirmed && (i.State != IntentConfirmed || i.Review) {
		if i, err = r.addEvent(l, IntentEvent{Intent: i.ID, Kind: IntentConfirmed, By: in.By, Via: in.Via, Actor: in.Actor}); err != nil {
			return Mark{}, err
		}
	}
	kind := in.LinkKind
	if kind == "" {
		kind = LinkImplements
	}
	state := in.Link
	if state == "" {
		state = LinkProposed
		if in.By == ByHuman {
			state = LinkConfirmed
		}
	}
	have := l.LinksOf(i.ID)
	rejected := l.RejectedLinks(i.ID)
	for _, w := range works {
		if slices.ContainsFunc(have, func(x LinkEvent) bool {
			return x.Work == w && x.Revision == i.Revision && x.Kind == kind && (state != LinkConfirmed || x.State == LinkConfirmed)
		}) {
			continue
		}
		// What someone turned down is not proposed again; only a decision brings it back.
		if state == LinkProposed && slices.ContainsFunc(rejected, func(x LinkEvent) bool { return x.Work == w }) {
			continue
		}
		if err := r.addLink(l, LinkEvent{Work: w, Intent: i.ID, Revision: i.Revision, Kind: kind, State: state, By: in.By, Via: in.Via, Actor: in.Actor, Reason: in.Reason}); err != nil {
			return Mark{}, err
		}
	}
	out := Mark{Path: in.Path, Line: at + 1, Text: text, Hash: hash, N: n, IDs: l.lineCommits(commits, hash, n)}
	for _, b := range ParseBlocks(in.Path, data) {
		for _, s := range b.Sentences {
			if s.Line == at+1 {
				out.Block = s.Block
			}
		}
	}
	return out, nil
}

// lineCommits is the commits on a line's intents, by short ID, oldest link
// first. A folded commit shows as the commit folding it.
func (l *Ledger) lineCommits(commits *Commits, hash string, n int) []string {
	var out []string
	for _, i := range l.on(hash, n) {
		for _, e := range l.LinksOf(i.ID) {
			out = appendNew(out, commits.ShowAs(e.Work))
		}
	}
	return out
}

// Unmark withdraws the links of commits whose work was undone. The documents
// stay: the design is still there, the code is gone.
func (r *Repo) Unmark(ctx context.Context, refs []string) error {
	if r.Dir == "" {
		return errors.New("tig: unmarking needs a .tig folder")
	}
	unlock, err := lockRepo(ctx, r.Dir)
	if err != nil {
		return err
	}
	defer unlock()
	commits, err := r.Commits(ctx)
	if err != nil {
		return err
	}
	l, err := r.Ledger(ctx)
	if err != nil {
		return err
	}
	undone := map[string]bool{}
	for _, ref := range refs {
		w, err := commits.Resolve(ref)
		if err != nil {
			return err
		}
		undone[w.ID] = true
		for _, leaf := range commits.Leaves(w.ID) {
			undone[leaf] = true
		}
	}
	for _, i := range l.Intents() {
		for _, e := range l.LinksOf(i.ID) {
			if undone[e.Work] {
				if err := r.addLink(l, LinkEvent{Work: e.Work, Intent: i.ID, Revision: e.Revision, Kind: LinkUnlinked}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// DecideInput is someone's decision on an intent.
type DecideInput struct {
	Intent string // its reference
	// Under is the section a line confirmed out of the inferred document goes
	// under in the main document; with none it stays under the title it had.
	Under  string
	From   []string // what took its place, for a superseded intent
	By     string
	Via    string
	Actor  string
	Reason string
}

// lineIn finds the line of a document with this hash and number.
func lineIn(lines []string, fenced []bool, hash string, n int) int {
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		if m, ok := readLineMark(line); ok && m.N == n && LineHash(LineText(line)) == hash {
			return i
		}
	}
	return -1
}

// ConfirmIntent says an intent is what was meant. Lines of it still in the
// inferred document move to the main document; their text does not change, so
// their ranges stay good.
func (r *Repo) ConfirmIntent(ctx context.Context, in DecideInput) (*Intent, error) {
	return r.decide(ctx, in, func(l *Ledger, i *Intent, cfg Config) error {
		if err := r.moveOutOfInferred(ctx, l, i, cfg, in); err != nil {
			return err
		}
		_, err := r.addEvent(l, IntentEvent{Intent: i.ID, Kind: IntentConfirmed, By: in.By, Via: in.Via, Actor: in.Actor, Reason: in.Reason})
		return err
	})
}

// RejectIntent says an intent is not what was meant. Its line leaves the inferred
// document, unless another intent still stands on it; a line in the main document
// stays, because that is the document's owner's to change.
func (r *Repo) RejectIntent(ctx context.Context, in DecideInput) (*Intent, error) {
	return r.decide(ctx, in, func(l *Ledger, i *Intent, cfg Config) error {
		if _, err := r.addEvent(l, IntentEvent{Intent: i.ID, Kind: IntentRejected, By: in.By, Via: in.Via, Actor: in.Actor, Reason: in.Reason}); err != nil {
			return err
		}
		full, data, err := r.readDoc(cfg.InferredDoc())
		if err != nil || len(data) == 0 {
			return err
		}
		lines := strings.Split(string(data), "\n")
		fenced := fencedLines(lines)
		var drop []int
		for _, s := range i.Spans {
			if at := lineIn(lines, fenced, s.Line, s.N); at >= 0 && len(l.on(s.Line, s.N)) == 0 {
				drop = append(drop, at)
			}
		}
		if len(drop) == 0 {
			return nil
		}
		slices.Sort(drop)
		for j := len(drop) - 1; j >= 0; j-- {
			lines = slices.Delete(lines, drop[j], drop[j]+1)
		}
		return writeDoc(full, []byte(strings.Join(lines, "\n")))
	})
}

// RetireIntent says an intent no longer stands: deprecated, or, with From,
// superseded by others.
func (r *Repo) RetireIntent(ctx context.Context, in DecideInput) (*Intent, error) {
	kind := IntentDeprecated
	if len(in.From) > 0 {
		kind = IntentSuperseded
	}
	return r.decide(ctx, in, func(l *Ledger, i *Intent, _ Config) error {
		_, err := r.addEvent(l, IntentEvent{Intent: i.ID, Kind: kind, From: in.From, By: in.By, Via: in.Via, Actor: in.Actor, Reason: in.Reason})
		return err
	})
}

func (r *Repo) decide(ctx context.Context, in DecideInput, do func(l *Ledger, i *Intent, cfg Config) error) (*Intent, error) {
	if r.Dir == "" {
		return nil, errors.New("tig: deciding needs a .tig folder")
	}
	cfg, err := r.Config()
	if err != nil {
		return nil, err
	}
	unlock, err := lockRepo(ctx, r.Dir)
	if err != nil {
		return nil, err
	}
	defer unlock()
	_, l, err := r.syncLocked(ctx, in.Actor)
	if err != nil {
		return nil, err
	}
	i, err := l.Intent(in.Intent)
	if err != nil {
		return nil, err
	}
	if err := do(l, i, cfg); err != nil {
		return nil, err
	}
	return l.intents[i.ID], nil
}

// moveOutOfInferred moves an intent's lines from the inferred document to the
// main one. A line the main document already has is not written twice.
func (r *Repo) moveOutOfInferred(ctx context.Context, l *Ledger, i *Intent, cfg Config, in DecideInput) error {
	fromDoc := cfg.InferredDoc()
	full, data, err := r.readDoc(fromDoc)
	if err != nil || len(data) == 0 {
		return err
	}
	lines := strings.Split(string(data), "\n")
	fenced := fencedLines(lines)
	mainFull, mainData, err := r.readDoc(cfg.Root)
	if err != nil {
		return err
	}
	spans := slices.Clone(i.Spans)
	respanned := false
	moved := false
	blocks := ParseBlocks(fromDoc, data)
	for j, s := range spans {
		at := lineIn(lines, fenced, s.Line, s.N)
		if at < 0 {
			continue
		}
		mainLines := strings.Split(string(mainData), "\n")
		mainFenced := fencedLines(mainLines)
		text := LineText(lines[at])
		if have, _, _, found := locate(mainLines, mainFenced, text); found && LineText(mainLines[have]) == text {
			m, marked := readLineMark(mainLines[have])
			if !marked {
				m = LineMark{Hash: l.shortLine(s.Line), N: s.N}
				mainLines[have] = setLineMark(mainLines[have], m)
				mainData = []byte(strings.Join(mainLines, "\n"))
			} else if m.N != s.N {
				spans[j].N = m.N
				respanned = true
			}
		} else {
			under := in.Under
			if under == "" {
				under = headingOfLine(blocks, at+1)
			}
			mainData = insertLine(cfg.Root, mainData, lines[at], under)
		}
		lines[at] = "\x00"
		moved = true
	}
	if !moved {
		return nil
	}
	kept := slices.DeleteFunc(lines, func(s string) bool { return s == "\x00" })
	if err := writeDoc(mainFull, mainData); err != nil {
		return err
	}
	if err := writeDoc(full, []byte(strings.Join(kept, "\n"))); err != nil {
		return err
	}
	if respanned {
		if _, err := r.addEvent(l, IntentEvent{Intent: i.ID, Kind: IntentSpans, Spans: spans, Actor: in.Actor}); err != nil {
			return err
		}
	}
	return nil
}

func headingOfLine(blocks []Block, line int) string {
	for _, b := range blocks {
		if b.Level > 0 && line > b.Start && line <= b.OwnEnd {
			return b.Title
		}
	}
	return ""
}

// InferredDoc is where sentences go until a person agrees with them.
func (c Config) InferredDoc() string {
	if c.Inferred != "" {
		return c.Inferred
	}
	return path.Join(path.Dir(c.Root), "inferred.md")
}
