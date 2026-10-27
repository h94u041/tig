// Package hub sends a project's documents and chain to a hub, and is the hub:
// it keeps the publications projects push to it and shows them.
//
// A push carries the documents of the chain and the chain itself, and nothing
// else: never the project's other files, nor its snapshots, works and steps.
package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/h94u041/tig"
)

// Publication is one push: the documents, the chain, and the publication
// before it.
type Publication struct {
	ID       string    `json:"id"`
	Docs     string    `json:"docs"`  // a tree of Markdown files only
	Chain    string    `json:"chain"` // a ChainView, as JSON
	Snapshot string    `json:"snapshot,omitempty"`
	Parent   string    `json:"parent,omitempty"`
	At       time.Time `json:"at"`
	Actor    string    `json:"actor,omitempty"`
	Note     string    `json:"note,omitempty"`
}

const publicationHeader = "tig-publication 1"

func (p Publication) bytes() []byte {
	var b strings.Builder
	b.WriteString(publicationHeader + "\n")
	set := func(key, value string) {
		if value != "" {
			value = strings.NewReplacer("\n", " ", "\r", " ").Replace(value)
			b.WriteString(key + " " + value + "\n")
		}
	}
	set("docs", p.Docs)
	set("chain", p.Chain)
	set("snapshot", p.Snapshot)
	set("parent", p.Parent)
	set("at", p.At.UTC().Format(time.RFC3339Nano))
	set("actor", p.Actor)
	set("note", p.Note)
	return []byte(b.String())
}

// LoadPublication reads a publication record.
func LoadPublication(ctx context.Context, objects tig.Objects, id string) (Publication, error) {
	raw, err := objects.GetObject(ctx, id)
	if err != nil {
		return Publication{}, err
	}
	text, ok := strings.CutPrefix(string(raw), publicationHeader+"\n")
	if !ok {
		return Publication{}, fmt.Errorf("hub: %s is not a publication", id)
	}
	p := Publication{ID: id}
	for _, line := range strings.Split(text, "\n") {
		k, v, _ := strings.Cut(line, " ")
		switch k {
		case "docs":
			p.Docs = v
		case "chain":
			p.Chain = v
		case "snapshot":
			p.Snapshot = v
		case "parent":
			p.Parent = v
		case "at":
			p.At, _ = time.Parse(time.RFC3339Nano, v)
		case "actor":
			p.Actor = v
		case "note":
			p.Note = v
		}
	}
	for _, h := range []string{p.Docs, p.Chain} {
		if !tig.ValidHash(h) {
			return Publication{}, fmt.Errorf("hub: publication %s has an invalid hash %q", id, h)
		}
	}
	for _, h := range []string{p.Snapshot, p.Parent} {
		if h != "" && !tig.ValidHash(h) {
			return Publication{}, fmt.Errorf("hub: publication %s has an invalid hash %q", id, h)
		}
	}
	return p, nil
}

// ChainView is the chain as a hub keeps it: the blocks and what the
// footprint concluded about them, without the footprint itself.
type ChainView struct {
	Root   string      `json:"root"`
	Docs   []string    `json:"docs"` // in the order the chain reaches them
	Blocks []BlockView `json:"blocks"`
	Report tig.Report  `json:"report"`
}

// BlockView is one block of the chain. Its text is in the documents.
type BlockView struct {
	Hash    string   `json:"hash"`
	Path    string   `json:"path"`
	Parent  string   `json:"parent,omitempty"`
	Level   int      `json:"level"`
	Title   string   `json:"title"`
	Start   int      `json:"start"`
	Status  string   `json:"status,omitempty"` // guideline, direction or open; "" for blocks without design
	Cover   string   `json:"cover,omitempty"`  // the acceptance section that accepts it
	Accept  []string `json:"accept,omitempty"`
	Details []string `json:"details,omitempty"` // detail documents its own text links to
	Refs    []string `json:"refs,omitempty"`    // project files its own text links to
	Built   []string `json:"built,omitempty"`   // files the footprint built for it
}

// Block finds a block of the view by its hash.
func (v *ChainView) Block(hash string) *BlockView {
	for i := range v.Blocks {
		if v.Blocks[i].Hash == hash {
			return &v.Blocks[i]
		}
	}
	return nil
}

func viewOf(a *tig.Analysis) ChainView {
	c := a.Chain
	v := ChainView{Root: c.Root, Docs: c.Docs, Report: a.Report}
	for _, h := range c.Order {
		b := c.Blocks[h]
		built := slices.Clone(a.Built[h])
		sort.Strings(built)
		v.Blocks = append(v.Blocks, BlockView{Hash: h, Path: b.Path, Parent: b.Parent, Level: b.Level, Title: b.Title,
			Start: b.Start, Status: a.Status[h], Cover: a.Cover[h], Accept: b.Accept, Details: c.Details[h],
			Refs: c.Refs[h], Built: built})
	}
	return v
}

// memory is an object store in memory.
type memory map[string][]byte

func (m memory) PutObject(_ context.Context, data []byte) (string, error) {
	h := tig.HashOf(data)
	m[h] = data
	return h, nil
}

func (m memory) GetObject(_ context.Context, h string) ([]byte, error) {
	if data, ok := m[h]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("%w: %s", tig.ErrNotFound, h)
}

// Build makes the publication of a project's latest snapshot on top of
// parent: the documents of its chain (and README.md), and the chain. It
// returns the publication and every object it is made of.
func Build(ctx context.Context, r *tig.Repo, parent, actor, note string) (Publication, map[string][]byte, error) {
	head, err := r.Head(ctx)
	if err != nil {
		return Publication{}, nil, err
	}
	if head.ID == "" {
		return Publication{}, nil, fmt.Errorf("hub: the project has no snapshot yet")
	}
	a, err := r.Analyze(ctx, head.Tree)
	if err != nil {
		return Publication{}, nil, err
	}
	files, err := tig.ReadTree(ctx, r.Objects, head.Tree)
	if err != nil {
		return Publication{}, nil, err
	}
	objects := memory{}
	docs := map[string]tig.Entry{}
	paths := slices.Clone(a.Chain.Docs)
	if _, ok := files["README.md"]; ok {
		paths = append(paths, "README.md")
	}
	for _, p := range paths {
		e, ok := files[p]
		if !ok || !tig.IsMarkdown(p) {
			continue
		}
		data, err := r.Objects.GetObject(ctx, e.Hash)
		if err != nil {
			return Publication{}, nil, err
		}
		objects[e.Hash] = data
		docs[p] = tig.Entry{Hash: e.Hash, Size: e.Size}
	}
	tree, err := tig.WriteTree(ctx, objects, docs)
	if err != nil {
		return Publication{}, nil, err
	}
	chain, err := json.Marshal(viewOf(a))
	if err != nil {
		return Publication{}, nil, err
	}
	p := Publication{Docs: tree, Snapshot: head.ID, Parent: parent, At: time.Now().UTC(), Actor: actor, Note: note}
	if p.Chain, err = objects.PutObject(ctx, chain); err != nil {
		return Publication{}, nil, err
	}
	p.ID, _ = objects.PutObject(ctx, p.bytes())
	return p, objects, nil
}
