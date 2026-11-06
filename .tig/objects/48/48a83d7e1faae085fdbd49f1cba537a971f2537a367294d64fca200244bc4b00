package tig

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// What one intent has to do with another. A shared file says the code depends on
// it; only someone can say the intents do. tig keeps what it is told and acts on
// none of it: whoever reads a set decides what a relation means for the work.

// The kinds of relation tig's own answers know. Any other word may be written and
// is kept.
const (
	RelationDependsOn = "depends_on" // it does not hold without the other
	RelationPartOf    = "part_of"    // the other is what it is a part of
	RelationParallel  = "parallel"   // they stand apart, whatever code they share
)

// RelationEvent is one thing said about two intents. The last one said about a
// pair and a kind is what stands; one that was rejected is remembered, so nobody
// proposes it again. A parallel relation reads the same from both sides.
type RelationEvent struct {
	ID     string    `json:"id"`
	From   string    `json:"from"`
	To     string    `json:"to"`
	Kind   string    `json:"kind"`
	State  string    `json:"state"` // LinkProposed, LinkConfirmed or LinkRejected
	By     string    `json:"by,omitempty"`
	Via    string    `json:"via,omitempty"`
	Actor  string    `json:"actor,omitempty"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

func (e RelationEvent) encode() []byte {
	rec := newRecord(relationHeader)
	rec.set("from", e.From)
	rec.set("to", e.To)
	rec.set("kind", e.Kind)
	rec.set("state", e.State)
	rec.set("by", e.By)
	rec.set("via", e.Via)
	rec.set("actor", e.Actor)
	rec.set("reason", e.Reason)
	rec.setTime("at", e.At)
	return rec.bytes()
}

func parseRelationEvent(id string, raw []byte) (RelationEvent, error) {
	rec, err := parseRecord(raw, relationHeader)
	if err != nil {
		return RelationEvent{}, err
	}
	e := RelationEvent{ID: id, From: rec.get("from"), To: rec.get("to"), Kind: rec.get("kind"), State: rec.get("state"),
		By: rec.get("by"), Via: rec.get("via"), Actor: rec.get("actor"), Reason: rec.get("reason"), At: rec.time("at")}
	if err := hashesValid(id, nil, e.From, e.To); err != nil {
		return RelationEvent{}, err
	}
	return e, nil
}

// key is what makes two events about the same relation.
func (e RelationEvent) key() [3]string {
	a, b := e.From, e.To
	if e.Kind == RelationParallel && b < a {
		a, b = b, a
	}
	return [3]string{a, b, e.Kind}
}

// lastRelations is the last thing said about each relation an intent is in, in
// the order they first came up.
func (l *Ledger) lastRelations(intent string) []RelationEvent {
	latest := map[[3]string]RelationEvent{}
	var keys [][3]string
	for _, e := range l.relations {
		if e.From != intent && e.To != intent {
			continue
		}
		k := e.key()
		if _, ok := latest[k]; !ok {
			keys = append(keys, k)
		}
		latest[k] = e
	}
	out := make([]RelationEvent, 0, len(keys))
	for _, k := range keys {
		out = append(out, latest[k])
	}
	return out
}

// IntentRelation is a relation as one of its two intents sees it.
type IntentRelation struct {
	Kind string `json:"kind"`
	With string `json:"with"` // the other intent's short ID
	// Incoming: the relation was said of this intent, from the other one.
	Incoming bool      `json:"incoming,omitempty"`
	State    string    `json:"state"`
	By       string    `json:"by,omitempty"`
	Via      string    `json:"via,omitempty"`
	Actor    string    `json:"actor,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	At       time.Time `json:"at"`
}

func (l *Ledger) viewRelations(intent string, keep func(RelationEvent) bool) []IntentRelation {
	var out []IntentRelation
	for _, e := range l.lastRelations(intent) {
		if !keep(e) {
			continue
		}
		other, incoming := e.To, false
		if e.To == intent {
			other, incoming = e.From, e.Kind != RelationParallel
		}
		with := other
		if i := l.intents[other]; i != nil {
			with = i.Short
		} else if len(with) > minLineShort {
			with = with[:minLineShort]
		}
		out = append(out, IntentRelation{Kind: e.Kind, With: with, Incoming: incoming, State: e.State, By: e.By, Via: e.Via, Actor: e.Actor, Reason: e.Reason, At: e.At})
	}
	return out
}

// RelationsOf is the relations standing on an intent: oldest first, the rejected
// ones left out.
func (l *Ledger) RelationsOf(intent string) []IntentRelation {
	return l.viewRelations(intent, func(e RelationEvent) bool { return e.State != LinkRejected })
}

// RejectedRelationsOf is the relations someone said do not hold, and nobody has
// said otherwise since.
func (l *Ledger) RejectedRelationsOf(intent string) []IntentRelation {
	return l.viewRelations(intent, func(e RelationEvent) bool { return e.State == LinkRejected })
}

// ProposedRelations counts the relations nobody has decided on yet.
func (l *Ledger) ProposedRelations() int {
	latest := map[[3]string]RelationEvent{}
	for _, e := range l.relations {
		latest[e.key()] = e
	}
	n := 0
	for _, e := range latest {
		if e.State == LinkProposed {
			n++
		}
	}
	return n
}

// RelateInput says one intent has something to do with another, or does not.
type RelateInput struct {
	From, To string // references to the intents
	Kind     string // RelationDependsOn, RelationPartOf, RelationParallel, or a word of your own
	// State is LinkProposed, LinkConfirmed or LinkRejected. Empty means confirmed
	// when a person says it (By is ByHuman), proposed otherwise.
	State  string
	By     string
	Via    string
	Actor  string
	Reason string
}

// Relate records what is said about two intents. It judges nothing and changes
// neither intent: a relation is not what either of them says.
func (r *Repo) Relate(ctx context.Context, in RelateInput) (RelationEvent, error) {
	if r.Dir == "" {
		return RelationEvent{}, errors.New("tig: a relation needs a .tig folder")
	}
	if in.Kind == "" || strings.ContainsAny(in.Kind, " \t\r\n") {
		return RelationEvent{}, fmt.Errorf("tig: a relation's kind is one word, not %q", in.Kind)
	}
	state := in.State
	if state == "" {
		state = LinkProposed
		if in.By == ByHuman {
			state = LinkConfirmed
		}
	}
	if !slices.Contains([]string{LinkProposed, LinkConfirmed, LinkRejected}, state) {
		return RelationEvent{}, fmt.Errorf("tig: a relation is proposed, confirmed or rejected, not %q", state)
	}
	unlock, err := lockRepo(ctx, r.Dir)
	if err != nil {
		return RelationEvent{}, err
	}
	defer unlock()
	l, err := r.Ledger(ctx)
	if err != nil {
		return RelationEvent{}, err
	}
	from, err := l.Intent(in.From)
	if err != nil {
		return RelationEvent{}, err
	}
	to, err := l.Intent(in.To)
	if err != nil {
		return RelationEvent{}, err
	}
	if from.ID == to.ID {
		return RelationEvent{}, fmt.Errorf("tig: %s is the same intent on both sides", from.Short)
	}
	if err := r.startLedger(l); err != nil {
		return RelationEvent{}, err
	}
	e := RelationEvent{From: from.ID, To: to.ID, Kind: in.Kind, State: state, By: in.By, Via: in.Via, Actor: in.Actor, Reason: in.Reason, At: time.Now().UTC()}
	if e.ID, err = r.putLedger(e.encode()); err != nil {
		return RelationEvent{}, err
	}
	return e, nil
}
