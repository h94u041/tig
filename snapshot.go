package tig

import (
	"context"
	"fmt"
	"time"
)

// Snapshot is a snapshot record: the tree, the one before, when, why, who,
// and how the documents' text blocks changed since the one before.
type Snapshot struct {
	ID      string    `json:"id"`
	Tree    string    `json:"tree"`
	Parents []string  `json:"parents,omitempty"`
	At      time.Time `json:"at"`
	Reason  string    `json:"reason"`
	Actor   string    `json:"actor,omitempty"`
	Lineage string    `json:"lineage,omitempty"`
	// Extra is what a later feature added to the record ("x-…" fields).
	Extra map[string][]string `json:"extra,omitempty"`
}

// Parent is the snapshot before this one ("" for the first).
func (s Snapshot) Parent() string {
	if len(s.Parents) == 0 {
		return ""
	}
	return s.Parents[0]
}

const snapshotHeader = "tig-snapshot 1"

// Reasons for a snapshot (key moments).
const (
	ReasonOpen   = "open"
	ReasonBefore = "before-construction"
	ReasonAfter  = "after-construction"
	ReasonScan   = "scan"
	ReasonUpdate = "baseline" // a tool moved its baseline over files it changed
)

// Take records the project as it is now. When nothing changed since the head
// it returns the head and records nothing (changed=false).
func (r *Repo) Take(ctx context.Context, reason, actor string) (s Snapshot, changed bool, err error) {
	files, err := r.Scan(ctx)
	if err != nil {
		return Snapshot{}, false, err
	}
	tree, err := WriteTree(ctx, r.Objects, files)
	if err != nil {
		return Snapshot{}, false, err
	}
	return r.Record(ctx, tree, reason, actor)
}

// Record adds a snapshot record for a tree on top of the head (unless the
// head already has that tree).
func (r *Repo) Record(ctx context.Context, tree, reason, actor string) (Snapshot, bool, error) {
	if r.Dir != "" {
		unlock, err := lockRepo(ctx, r.Dir)
		if err != nil {
			return Snapshot{}, false, err
		}
		defer unlock()
	}
	head, err := r.Head(ctx)
	if err != nil {
		return Snapshot{}, false, err
	}
	if head.ID != "" && head.Tree == tree {
		return head, false, nil
	}
	s := Snapshot{Tree: tree, At: time.Now().UTC(), Reason: reason, Actor: actor}
	if head.ID != "" {
		s.Parents = []string{head.ID}
		// The documents' blocks are matched only between consecutive
		// snapshots, so a block keeps its line of predecessors.
		if s.Lineage, err = r.recordLineage(ctx, head.Tree, tree); err != nil {
			return Snapshot{}, false, err
		}
	}
	rec := newRecord(snapshotHeader)
	rec.set("tree", s.Tree)
	rec.setAll("parent", s.Parents)
	rec.setTime("at", s.At)
	rec.set("reason", reason)
	rec.set("actor", actor)
	rec.set("lineage", s.Lineage)
	id, err := r.Objects.PutObject(ctx, rec.bytes())
	if err != nil {
		return Snapshot{}, false, err
	}
	s.ID = id
	return s, true, r.writeHead(headSnapshots, id)
}

// Load reads a snapshot record.
func Load(ctx context.Context, objects Objects, id string) (Snapshot, error) {
	raw, err := objects.GetObject(ctx, id)
	if err != nil {
		return Snapshot{}, err
	}
	rec, err := parseRecord(raw, snapshotHeader)
	if err != nil {
		return Snapshot{}, fmt.Errorf("tig: %s is not a snapshot", id)
	}
	s := Snapshot{ID: id, Tree: rec.get("tree"), Parents: rec.all("parent"), At: rec.time("at"),
		Reason: rec.get("reason"), Actor: rec.get("actor"), Lineage: rec.get("lineage"), Extra: rec.extra()}
	for _, h := range append([]string{s.Tree}, s.Parents...) {
		if !ValidHash(h) {
			return Snapshot{}, fmt.Errorf("tig: snapshot %s has an invalid hash %q", id, h)
		}
	}
	if s.Lineage != "" && !ValidHash(s.Lineage) {
		return Snapshot{}, fmt.Errorf("tig: snapshot %s has an invalid lineage %q", id, s.Lineage)
	}
	return s, nil
}

// Head is the latest snapshot (zero when there is none yet).
func (r *Repo) Head(ctx context.Context) (Snapshot, error) {
	id, err := r.readHead(headSnapshots)
	if err != nil || id == "" {
		return Snapshot{}, err
	}
	return Load(ctx, r.Objects, id)
}

// Log walks the chain from the head, newest first, at most n (n<=0: all).
func (r *Repo) Log(ctx context.Context, n int) ([]Snapshot, error) {
	s, err := r.Head(ctx)
	if err != nil || s.ID == "" {
		return nil, err
	}
	var out []Snapshot
	seen := map[string]bool{}
	for {
		out = append(out, s)
		seen[s.ID] = true
		if s.Parent() == "" || seen[s.Parent()] || (n > 0 && len(out) >= n) {
			return out, nil
		}
		if s, err = Load(ctx, r.Objects, s.Parent()); err != nil {
			return out, err
		}
	}
}
