package tig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Link is a link someone proposed between a block and a file or another
// block: a model or a tool saw that they belong together. It is always
// inferred. The links the documents write and the ones construction makes
// are not records: they are read from the documents and the steps.
type Link struct {
	ID      string    `json:"id"`
	Parent  string    `json:"parent,omitempty"`
	From    string    `json:"from"`              // a block
	ToBlock string    `json:"toBlock,omitempty"` // another block, or
	ToFile  string    `json:"toFile,omitempty"`  // a project file
	ToHash  string    `json:"toHash,omitempty"`  // the file's content when the link was made
	Kind    string    `json:"kind"`              // implements, tests, refines, references
	Note    string    `json:"note,omitempty"`
	Actor   string    `json:"actor,omitempty"`
	At      time.Time `json:"at"`
}

const linkHeader = "tig-link 1"

// ProposeLink records a proposed link on top of the latest one.
func (r *Repo) ProposeLink(ctx context.Context, l Link) (Link, error) {
	if !ValidHash(l.From) || (l.ToBlock != "" && !ValidHash(l.ToBlock)) || (l.ToBlock == "") == (l.ToFile == "") {
		return Link{}, fmt.Errorf("tig: a link goes from a block to one block or one file")
	}
	if l.ToFile != "" {
		if !listable(l.ToFile) {
			return Link{}, fmt.Errorf("tig: %q is not a project file", l.ToFile)
		}
		if l.ToHash == "" {
			if data, err := os.ReadFile(filepath.Join(r.Root, filepath.FromSlash(l.ToFile))); err == nil {
				l.ToHash = HashOf(data)
			}
		}
	}
	if r.Dir != "" {
		unlock, err := lockRepo(ctx, r.Dir)
		if err != nil {
			return Link{}, err
		}
		defer unlock()
	}
	parent, err := r.readHead(headLinks)
	if err != nil {
		return Link{}, err
	}
	l.Parent = parent
	if l.At.IsZero() {
		l.At = time.Now().UTC()
	}
	rec := newRecord(linkHeader)
	rec.set("parent", l.Parent)
	rec.set("from", l.From)
	rec.set("to-block", l.ToBlock)
	rec.set("to-file", l.ToFile)
	rec.set("to-hash", l.ToHash)
	rec.set("kind", l.Kind)
	rec.set("note", l.Note)
	rec.set("actor", l.Actor)
	rec.setTime("at", l.At)
	if l.ID, err = r.Objects.PutObject(ctx, rec.bytes()); err != nil {
		return Link{}, err
	}
	return l, r.writeHead(headLinks, l.ID)
}

// Links is every proposed link, oldest first.
func (r *Repo) Links(ctx context.Context) ([]Link, error) {
	id, err := r.readHead(headLinks)
	if err != nil {
		return nil, err
	}
	var out []Link
	seen := map[string]bool{}
	for id != "" && !seen[id] {
		seen[id] = true
		raw, err := r.Objects.GetObject(ctx, id)
		if err != nil {
			return nil, err
		}
		rec, err := parseRecord(raw, linkHeader)
		if err != nil {
			return nil, err
		}
		l := Link{ID: id, Parent: rec.get("parent"), From: rec.get("from"), ToBlock: rec.get("to-block"), ToFile: rec.get("to-file"),
			ToHash: rec.get("to-hash"), Kind: rec.get("kind"), Note: rec.get("note"), Actor: rec.get("actor"), At: rec.time("at")}
		if err := hashesValid(id, nil, l.Parent, l.From, l.ToBlock, l.ToHash); err != nil {
			return nil, err
		}
		if l.ToFile != "" && !listable(l.ToFile) {
			return nil, fmt.Errorf("tig: link %s points outside the project", id)
		}
		out = append(out, l)
		id = l.Parent
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}
