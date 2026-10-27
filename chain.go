package tig

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Chain is the project's documents as one whole: the main document, the
// detail documents its blocks link to (and theirs, and so on), every
// document's text blocks, and the project files the blocks link to.
type Chain struct {
	Tree     string              `json:"tree,omitempty"` // "" when read from the working files
	Root     string              `json:"root"`
	Docs     []string            `json:"docs"` // in the order the chain reaches them
	Blocks   map[string]*Block   `json:"-"`
	Order    []string            `json:"-"` // every block, documents in chain order
	DocBlock map[string]string   `json:"-"` // document → its own block (level 0)
	Children map[string][]string `json:"-"`
	Details  map[string][]string `json:"-"` // block → detail documents its own text links to
	Refs     map[string][]string `json:"-"` // block → project files (and folders, ending in "/") its own text links to
	Orphans  []string            `json:"orphans,omitempty"`
	Missing  []Finding           `json:"missing,omitempty"` // links to documents or files that do not exist

	outside []string
	src     source
}

// Finding is one problem the closure check found.
type Finding struct {
	Block   string `json:"block,omitempty"`
	Path    string `json:"path,omitempty"`
	Title   string `json:"title,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

// source is where the chain reads files: a tree, or the working files.
type source interface {
	list(ctx context.Context) ([]string, error)
	read(ctx context.Context, rel string) ([]byte, error)
	hash(ctx context.Context, rel string) (string, bool)
}

type treeSource struct {
	r       *Repo
	tree    string
	entries map[string]Entry
}

func (s *treeSource) load(ctx context.Context) error {
	if s.entries != nil {
		return nil
	}
	var err error
	s.entries, err = ReadTree(ctx, s.r.Objects, s.tree)
	return err
}

func (s *treeSource) list(ctx context.Context) ([]string, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(s.entries))
	for p := range s.entries {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

func (s *treeSource) read(ctx context.Context, rel string) ([]byte, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	e, ok := s.entries[rel]
	if !ok {
		return nil, fs.ErrNotExist
	}
	if e.Size > maxDocument {
		return nil, fmt.Errorf("tig: %s is too large to read as a document", rel)
	}
	return s.r.Objects.GetObject(ctx, e.Hash)
}

func (s *treeSource) hash(ctx context.Context, rel string) (string, bool) {
	if s.load(ctx) != nil {
		return "", false
	}
	e, ok := s.entries[rel]
	return e.Hash, ok
}

type diskSource struct {
	r     *Repo
	files []string
}

func (s *diskSource) list(ctx context.Context) ([]string, error) {
	if s.files == nil {
		var err error
		if s.files, err = Files(ctx, s.r.Root, s.r.Skip); err != nil {
			return nil, err
		}
	}
	return s.files, nil
}

func (s *diskSource) read(_ context.Context, rel string) ([]byte, error) {
	full := filepath.Join(s.r.Root, filepath.FromSlash(rel))
	fi, err := os.Lstat(full)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fs.ErrNotExist
	}
	if fi.Size() > maxDocument {
		return nil, fmt.Errorf("tig: %s is too large to read as a document", rel)
	}
	return os.ReadFile(full)
}

func (s *diskSource) hash(_ context.Context, rel string) (string, bool) {
	full := filepath.Join(s.r.Root, filepath.FromSlash(rel))
	if fi, err := os.Lstat(full); err != nil || !fi.Mode().IsRegular() {
		return "", false
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", false
	}
	return HashOf(data), true
}

// Chain reads the chain from a tree, or from the working files when tree is
// "".
func (r *Repo) Chain(ctx context.Context, tree string) (*Chain, error) {
	cfg, err := r.Config()
	if err != nil {
		return nil, err
	}
	var src source = &diskSource{r: r}
	if tree != "" {
		src = &treeSource{r: r, tree: tree}
	}
	files, err := src.list(ctx)
	if err != nil {
		return nil, err
	}
	c := &Chain{Tree: tree, Root: cfg.Root, Blocks: map[string]*Block{}, DocBlock: map[string]string{},
		Children: map[string][]string{}, Details: map[string][]string{}, Refs: map[string][]string{},
		outside: cfg.Outside, src: src}
	exists := map[string]bool{}
	for _, f := range files {
		exists[f] = true
	}
	isFolder := func(p string) bool {
		i := sort.SearchStrings(files, p+"/")
		return i < len(files) && strings.HasPrefix(files[i], p+"/")
	}
	if !exists[cfg.Root] {
		c.Missing = append(c.Missing, Finding{Path: cfg.Root, Message: "the main document does not exist"})
	}
	queue := []string{cfg.Root}
	queued := map[string]bool{cfg.Root: true}
	for len(queue) > 0 {
		doc := queue[0]
		queue = queue[1:]
		if !exists[doc] {
			continue
		}
		data, err := src.read(ctx, doc)
		if err != nil {
			c.Missing = append(c.Missing, Finding{Path: doc, Message: err.Error()})
			continue
		}
		c.Docs = append(c.Docs, doc)
		blocks := ParseBlocks(doc, data)
		for i := range blocks {
			b := &blocks[i]
			c.Blocks[b.Hash] = b
			c.Order = append(c.Order, b.Hash)
			if b.Level == 0 {
				c.DocBlock[doc] = b.Hash
			} else {
				c.Children[b.Parent] = append(c.Children[b.Parent], b.Hash)
			}
			for _, target := range b.Links {
				p, ok := resolveLink(doc, target)
				if !ok || p == doc {
					continue
				}
				switch {
				case exists[p] && IsMarkdown(p) && !c.isOutside(p):
					c.Details[b.Hash] = appendNew(c.Details[b.Hash], p)
					if !queued[p] {
						queued[p] = true
						queue = append(queue, p)
					}
				case exists[p]:
					c.Refs[b.Hash] = appendNew(c.Refs[b.Hash], p)
				case isFolder(p):
					c.Refs[b.Hash] = appendNew(c.Refs[b.Hash], p+"/")
				default:
					c.Missing = append(c.Missing, Finding{Block: b.Hash, Path: doc, Title: b.Title, Line: b.Start,
						Message: fmt.Sprintf("links to %s, which does not exist", target)})
				}
			}
		}
	}
	for _, f := range files {
		if IsMarkdown(f) && !queued[f] && !c.isOutside(f) {
			c.Orphans = append(c.Orphans, f)
		}
	}
	return c, nil
}

func (c *Chain) isOutside(p string) bool {
	for _, o := range c.outside {
		o = strings.TrimSuffix(o, "/")
		if p == o || strings.HasPrefix(p, o+"/") {
			return true
		}
	}
	return false
}

func appendNew(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

// resolveLink is the project path a link in doc points to; false for a link
// that leaves the project (a URL, a path above the project) or only points
// into the same document.
func resolveLink(doc, target string) (string, bool) {
	if external(target) || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "//") {
		return "", false
	}
	target, _, _ = strings.Cut(target, "#")
	target, _, _ = strings.Cut(target, "?")
	if t, err := url.PathUnescape(target); err == nil {
		target = t
	}
	target = strings.ReplaceAll(target, "\\", "/")
	if target == "" {
		return "", false
	}
	var p string
	if strings.HasPrefix(target, "/") {
		p = path.Clean(strings.TrimPrefix(target, "/"))
	} else {
		p = path.Clean(path.Join(path.Dir(doc), target))
	}
	if p == "." || p == ".." || strings.HasPrefix(p, "../") || !listable(p) {
		return "", false
	}
	return p, true
}

// Block finds a block by its hash or a unique prefix of it (at least 8
// characters).
func (c *Chain) Block(ref string) (*Block, error) {
	if b, ok := c.Blocks[ref]; ok {
		return b, nil
	}
	if len(ref) < 8 {
		return nil, fmt.Errorf("tig: no block %q (give at least 8 characters of its hash)", ref)
	}
	var found *Block
	for h, b := range c.Blocks {
		if strings.HasPrefix(h, ref) {
			if found != nil {
				return nil, fmt.Errorf("tig: %q matches more than one block", ref)
			}
			found = b
		}
	}
	if found == nil {
		return nil, fmt.Errorf("tig: no block %q in the chain", ref)
	}
	return found, nil
}

// inAcceptance reports whether a block is an acceptance section or inside
// one.
func (c *Chain) inAcceptance(h string) bool {
	for b := c.Blocks[h]; b != nil; b = c.Blocks[b.Parent] {
		if b.IsAcceptance() {
			return true
		}
	}
	return false
}

// isContent reports whether a block holds design that needs accepting: any
// section that is not acceptance, and a document's own text when it has
// some.
func (c *Chain) isContent(h string) bool {
	b := c.Blocks[h]
	if b == nil || c.inAcceptance(h) {
		return false
	}
	return b.Level > 0 || b.Text != ""
}

// cover is what accepts a block. Walking up from the block, the first that
// comes wins: an acceptance section directly under a block (acc), or a block
// whose own text hands on to detail documents (by). With neither, the block
// is covered by its sections, if it has any.
func (c *Chain) cover(h string) (acc, by string) {
	for b := c.Blocks[h]; b != nil; b = c.Blocks[b.Parent] {
		for _, k := range c.Children[b.Hash] {
			if c.Blocks[k].IsAcceptance() {
				return k, ""
			}
		}
		if len(c.Details[b.Hash]) > 0 {
			return "", b.Hash
		}
	}
	return "", ""
}

// delegated reports whether a block hands its acceptance to detail
// documents.
func (c *Chain) delegated(h string) bool {
	_, by := c.cover(h)
	return by != ""
}

// sections are the blocks directly under a block that hold design.
func (c *Chain) sections(h string) []string {
	var out []string
	for _, k := range c.Children[h] {
		if c.isContent(k) {
			out = append(out, k)
		}
	}
	return out
}

// The footprint as the chain sees it.

type history struct {
	works      []Work // oldest first
	steps      map[string]Step
	successors map[string][]string // block → the blocks it became
	links      []Link
}

func (r *Repo) history(ctx context.Context) (*history, error) {
	h := &history{steps: map[string]Step{}, successors: map[string][]string{}}
	works, err := r.Works(ctx, 0)
	if err != nil {
		return nil, err
	}
	for i := len(works) - 1; i >= 0; i-- {
		h.works = append(h.works, works[i])
		for _, id := range works[i].Steps {
			s, err := LoadStep(ctx, r.Objects, id)
			if err != nil {
				return nil, err
			}
			h.steps[id] = s
		}
	}
	if cur, err := r.Current(); err == nil && cur != nil {
		// The work in progress counts too, as an unfinished work.
		w := Work{Basis: cur.Basis, Intent: cur.Intent, Before: cur.Before, Steps: cur.Steps, Actor: cur.Actor, Started: cur.Started, Result: "in-progress"}
		for _, id := range cur.Steps {
			if s, err := LoadStep(ctx, r.Objects, id); err == nil {
				h.steps[id] = s
			}
		}
		h.works = append(h.works, w)
	}
	snaps, err := r.Log(ctx, 0)
	if err != nil {
		return nil, err
	}
	for _, s := range snaps {
		if s.Lineage == "" {
			continue
		}
		l, err := LoadLineage(ctx, r.Objects, s.Lineage)
		if err != nil {
			return nil, err
		}
		for _, m := range l.Matches {
			h.successors[m.Old] = appendNew(h.successors[m.Old], m.New)
		}
	}
	if h.links, err = r.Links(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

// workingLineage matches the documents changed since the last snapshot
// against the working files, so a block edited a moment ago still has its
// footprint. It is not recorded; the next snapshot records it.
func (r *Repo) workingLineage(ctx context.Context, c *Chain, h *history) {
	head, err := r.Head(ctx)
	if err != nil || head.ID == "" {
		return
	}
	before, err := ReadTree(ctx, r.Objects, head.Tree)
	if err != nil {
		return
	}
	for _, doc := range c.Docs {
		e, ok := before[doc]
		if now, _ := c.src.hash(ctx, doc); !ok || now == e.Hash {
			continue
		}
		var after []Block
		for _, hash := range c.Order {
			if b := c.Blocks[hash]; b.Path == doc {
				after = append(after, *b)
			}
		}
		matches, _, _ := MatchBlocks(blocksAt(ctx, r.Objects, head.Tree, doc), after)
		for _, m := range matches {
			h.successors[m.Old] = appendNew(h.successors[m.Old], m.New)
		}
	}
}

// current is what a block hash is now: itself if the chain has it, or the
// blocks it became (following its lineage forward).
func (h *history) current(c *Chain, hash string) []string {
	if _, ok := c.Blocks[hash]; ok {
		return []string{hash}
	}
	var out []string
	seen := map[string]bool{hash: true}
	queue := []string{hash}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		for _, y := range h.successors[x] {
			if seen[y] {
				continue
			}
			seen[y] = true
			if _, ok := c.Blocks[y]; ok {
				out = append(out, y)
			} else {
				queue = append(queue, y)
			}
		}
	}
	return out
}

// Analysis is the chain together with its footprint: what is closed, what
// is a guideline (built and accepted) or a direction (designed with an
// acceptance, not accepted yet), and what was built for each block.
type Analysis struct {
	Chain  *Chain
	Report Report
	// Status of each block with design: guideline, direction or open.
	Status map[string]string
	// Cover is the acceptance section that accepts each block.
	Cover map[string]string
	// Built is the files the footprint built for each block.
	Built map[string][]string

	hist     *history
	verify   string
	accepted map[string]bool // acceptance section → its commands last passed
}

// Block statuses.
const (
	StatusGuideline = "guideline"
	StatusDirection = "direction"
	StatusOpen      = "open"
)

// Report is the closure check.
type Report struct {
	Root       string       `json:"root"`
	Closed     bool         `json:"closed"`
	Verify     string       `json:"verify"`
	Docs       int          `json:"docs"`
	Blocks     int          `json:"blocks"`
	Guidelines int          `json:"guidelines"`
	Directions int          `json:"directions"`
	Broken     []Finding    `json:"broken,omitempty"`
	Open       []Finding    `json:"open,omitempty"`
	Stale      []Finding    `json:"stale,omitempty"`
	Undesigned []string     `json:"undesigned,omitempty"`
	Acceptance []Acceptance `json:"acceptance,omitempty"`
}

// Acceptance is one acceptance section and how its commands last went.
type Acceptance struct {
	Block    string          `json:"block"`
	Path     string          `json:"path"`
	Accepts  string          `json:"accepts"` // the title of the section it accepts
	Passed   bool            `json:"passed"`
	Covers   int             `json:"covers"`
	Commands []CommandResult `json:"commands"`
}

// CommandResult is how an acceptance command last went.
type CommandResult struct {
	Command string    `json:"command"`
	Result  string    `json:"result"` // passed, failed, never
	Runner  string    `json:"runner,omitempty"`
	At      time.Time `json:"at,omitzero"`
	Step    string    `json:"step,omitempty"`
}

// Analyze reads the chain (from a tree, or the working files when tree is
// "") and checks it against the footprint.
func (r *Repo) Analyze(ctx context.Context, tree string) (*Analysis, error) {
	c, err := r.Chain(ctx, tree)
	if err != nil {
		return nil, err
	}
	cfg, err := r.Config()
	if err != nil {
		return nil, err
	}
	h, err := r.history(ctx)
	if err != nil {
		return nil, err
	}
	if tree == "" {
		r.workingLineage(ctx, c, h)
	}
	a := &Analysis{Chain: c, Status: map[string]string{}, Cover: map[string]string{}, Built: map[string][]string{}, hist: h, verify: cfg.Verify}
	rep := &a.Report
	rep.Root, rep.Verify, rep.Docs, rep.Blocks = c.Root, cfg.Verify, len(c.Docs), len(c.Blocks)
	rep.Broken = append(rep.Broken, c.Missing...)
	for _, o := range c.Orphans {
		rep.Broken = append(rep.Broken, Finding{Path: o, Message: "the main document does not reach this document"})
	}

	// Acceptance: how each section's commands last went.
	accepted := map[string]bool{}
	for _, hash := range c.Order {
		b := c.Blocks[hash]
		if !b.IsAcceptance() {
			continue
		}
		acc := Acceptance{Block: hash, Path: b.Path, Accepts: a.accepts(b)}
		last := map[string]CommandResult{}
		for _, w := range h.works {
			for _, id := range w.Steps {
				s := h.steps[id]
				if s.Kind != StepCheck || (cfg.Verify == VerifyStrict && s.Runner != RunnerTig) || !h.reaches(c, s.Basis, hash) {
					continue
				}
				res := CommandResult{Command: s.Command, Result: "failed", Runner: s.Runner, At: s.At, Step: s.ID}
				if s.Passed() {
					res.Result = "passed"
				}
				last[s.Command] = res
			}
		}
		acc.Passed = len(b.Accept) > 0
		for _, cmd := range b.Accept {
			res, ok := last[cmd]
			if !ok {
				res, ok = last[""] // a check reported without its command counts for every command
			}
			if !ok {
				res = CommandResult{Command: cmd, Result: "never"}
			}
			res.Command = cmd
			acc.Passed = acc.Passed && res.Result == "passed"
			acc.Commands = append(acc.Commands, res)
		}
		if len(b.Accept) == 0 {
			rep.Open = append(rep.Open, Finding{Block: hash, Path: b.Path, Title: b.Title, Line: b.Start, Message: "the acceptance section has no commands"})
		}
		accepted[hash] = acc.Passed
		rep.Acceptance = append(rep.Acceptance, acc)
	}

	// Every block with design is accepted by an acceptance section, handed on
	// to detail documents, or covered by its sections.
	a.accepted = accepted
	covers := map[string]int{}
	for _, hash := range c.Order {
		if !c.isContent(hash) {
			continue
		}
		a.status(hash, map[string]bool{})
		if acc := a.Cover[hash]; acc != "" {
			covers[acc]++
		}
		if acc, by := c.cover(hash); acc == "" && by == "" && len(c.sections(hash)) == 0 {
			b := c.Blocks[hash]
			rep.Open = append(rep.Open, Finding{Block: hash, Path: b.Path, Title: b.Title, Line: b.Start, Message: "no acceptance covers this block"})
		}
	}
	for i := range rep.Acceptance {
		rep.Acceptance[i].Covers = covers[rep.Acceptance[i].Block]
	}
	for _, doc := range c.Docs {
		if doc == c.Root {
			continue
		}
		if !a.hasAcceptance(doc) && a.hasOwnDesign(doc) {
			rep.Open = append(rep.Open, Finding{Block: c.DocBlock[doc], Path: doc, Message: "the detail document has no acceptance"})
		}
		if !a.hasAcceptance(doc) && !a.hasOwnDesign(doc) && !a.handsOn(doc) {
			rep.Open = append(rep.Open, Finding{Block: c.DocBlock[doc], Path: doc, Message: "the detail document is empty"})
		}
	}
	for _, s := range a.Status {
		switch s {
		case StatusGuideline:
			rep.Guidelines++
		case StatusDirection:
			rep.Directions++
		}
	}

	a.footprint(ctx)
	rep.Closed = len(rep.Broken) == 0 && len(rep.Open) == 0 && len(rep.Stale) == 0 && len(rep.Undesigned) == 0
	return a, nil
}

// accepts is the title of what an acceptance section accepts.
func (a *Analysis) accepts(b *Block) string {
	if p := a.Chain.Blocks[b.Parent]; p != nil {
		if p.Level == 0 {
			return p.Path
		}
		return p.Title
	}
	return b.Path
}

func (a *Analysis) hasAcceptance(doc string) bool {
	for _, h := range a.Chain.Order {
		if b := a.Chain.Blocks[h]; b.Path == doc && b.IsAcceptance() {
			return true
		}
	}
	return false
}

// hasOwnDesign reports whether a document has design it does not hand on.
func (a *Analysis) hasOwnDesign(doc string) bool {
	for _, h := range a.Chain.Order {
		if b := a.Chain.Blocks[h]; b.Path == doc && a.Chain.isContent(h) && !a.Chain.delegated(h) {
			return true
		}
	}
	return false
}

func (a *Analysis) handsOn(doc string) bool {
	for _, h := range a.Chain.Order {
		if b := a.Chain.Blocks[h]; b.Path == doc && len(a.Chain.Details[h]) > 0 {
			return true
		}
	}
	return false
}

// status is a block's status, worked out once: from the acceptance section
// that accepts it, from the detail documents it is handed on to, or from its
// sections. Open when nothing covers it.
func (a *Analysis) status(h string, seen map[string]bool) string {
	if s, ok := a.Status[h]; ok {
		return s
	}
	if seen[h] {
		return StatusGuideline // documents that hand on to each other: the rest decides
	}
	seen[h] = true
	c := a.Chain
	s := StatusOpen
	acc, by := c.cover(h)
	switch {
	case acc != "":
		a.Cover[h] = acc
		s = StatusDirection
		if a.accepted[acc] {
			s = StatusGuideline
		}
	case by != "":
		s = a.detailStatus(by, seen)
	default:
		if kids := c.sections(h); len(kids) > 0 {
			s = StatusGuideline
			for _, k := range kids {
				s = worse(s, a.status(k, seen))
			}
		}
	}
	a.Status[h] = s
	return s
}

func worse(x, y string) string {
	switch {
	case x == StatusOpen || y == StatusOpen:
		return StatusOpen
	case x == StatusDirection || y == StatusDirection:
		return StatusDirection
	}
	return StatusGuideline
}

// detailStatus is the status of what a block hands on: the worst status of
// the design in the detail documents its own text links to.
func (a *Analysis) detailStatus(by string, seen map[string]bool) string {
	s := StatusGuideline
	for _, doc := range a.Chain.Details[by] {
		for _, h := range a.Chain.Order {
			if a.Chain.Blocks[h].Path == doc && a.Chain.isContent(h) {
				s = worse(s, a.status(h, seen))
			}
		}
		if !a.hasAcceptance(doc) && !a.handsOn(doc) {
			s = StatusOpen
		}
	}
	return s
}

// reaches reports whether a step done for basis was done for block (or what
// the block was before it changed).
func (h *history) reaches(c *Chain, basis []string, block string) bool {
	for _, b := range basis {
		if b == block {
			return true
		}
		for _, x := range h.current(c, b) {
			if x == block {
				return true
			}
		}
	}
	return false
}

// atEnd is what the blocks a work changed became by the time it ended (for
// the work in progress, by its last step), from the project before it and
// after it. Blocks it did not change are not in it.
func (a *Analysis) atEnd(ctx context.Context, w Work) map[string][]string {
	objects := a.objects()
	treeOf := func(snapshot string) string {
		s, err := Load(ctx, objects, snapshot)
		if err != nil {
			return ""
		}
		return s.Tree
	}
	start, end := treeOf(w.Before), ""
	if w.After != "" {
		end = treeOf(w.After)
	} else {
		for _, id := range w.Steps {
			if s := a.hist.steps[id]; s.After != "" {
				end = s.After
			}
		}
	}
	out := map[string][]string{}
	if start == "" || end == "" || start == end {
		return out
	}
	changes, err := DiffTrees(ctx, objects, start, end)
	if err != nil {
		return out
	}
	for _, ch := range changes {
		if !IsMarkdown(ch.Path) {
			continue
		}
		matches, _, _ := MatchBlocks(blocksAt(ctx, objects, start, ch.Path), blocksAt(ctx, objects, end, ch.Path))
		for _, m := range matches {
			out[m.Old] = appendNew(out[m.Old], m.New)
		}
	}
	return out
}

// footprint finds what was built for each block, what in the project no
// design reaches, and which links went stale.
func (a *Analysis) footprint(ctx context.Context) {
	c, h, rep := a.Chain, a.hist, &a.Report
	files, _ := c.src.list(ctx)
	type built struct {
		basis string // the block hash the step was done for
		hash  string // the file's content after the step ("" when deleted)
		at    time.Time
	}
	latest := map[[2]string]built{}  // (current block, file) → the last step that built it
	lastTouch := map[string]string{} // file → its content after the last step that changed it
	exact := map[string]bool{}       // block hashes some step was done for exactly
	treeCache := map[string]map[string]Entry{}
	readTree := func(t string) map[string]Entry {
		if m, ok := treeCache[t]; ok {
			return m
		}
		m, _ := ReadTree(ctx, a.objects(), t)
		treeCache[t] = m
		return m
	}
	for _, w := range h.works {
		// A work that changes its own design builds for the design it leaves.
		end := a.atEnd(ctx, w)
		ended := func(b string) []string {
			if news, ok := end[b]; ok {
				return news
			}
			return []string{b}
		}
		for _, b := range w.Basis {
			for _, e := range ended(b) {
				exact[e] = true
			}
		}
		for _, id := range w.Steps {
			s := h.steps[id]
			for _, b := range s.Basis {
				for _, e := range ended(b) {
					exact[e] = true
				}
			}
			if s.Before == "" || s.After == "" {
				continue
			}
			changes, err := DiffTrees(ctx, a.objects(), s.Before, s.After)
			if err != nil {
				continue
			}
			after := readTree(s.After)
			for _, ch := range changes {
				if IsMarkdown(ch.Path) {
					continue
				}
				content := after[ch.Path].Hash
				lastTouch[ch.Path] = content
				for _, b := range s.Basis {
					for _, e := range ended(b) {
						for _, cur := range h.current(c, e) {
							latest[[2]string{cur, ch.Path}] = built{basis: e, hash: content, at: s.At}
						}
					}
				}
			}
		}
	}
	designed := map[string]bool{}
	for key, bt := range latest {
		block, file := key[0], key[1]
		if bt.hash != "" {
			a.Built[block] = appendNew(a.Built[block], file)
			designed[file] = true
		}
		now, ok := c.src.hash(ctx, file)
		if !ok {
			now = ""
		}
		b := c.Blocks[block]
		switch {
		case bt.basis != block && now == bt.hash && !exact[block]:
			rep.Stale = append(rep.Stale, Finding{Block: block, Path: file, Title: b.Title,
				Message: "the design changed after this file was built for it, and the file did not follow"})
		case bt.basis == block && now != bt.hash && lastTouch[file] != now:
			msg := "the file changed outside the footprint after it was built for this design"
			if now == "" {
				msg = "the file built for this design is gone"
			}
			rep.Stale = append(rep.Stale, Finding{Block: block, Path: file, Title: b.Title, Message: msg})
		}
	}
	for _, l := range h.links {
		cur := h.current(c, l.From)
		if len(cur) == 0 {
			continue
		}
		if l.ToFile != "" {
			for _, x := range cur {
				a.Built[x] = appendNew(a.Built[x], l.ToFile)
			}
			designed[l.ToFile] = true
			if now, _ := c.src.hash(ctx, l.ToFile); l.ToHash != "" && now != l.ToHash {
				rep.Stale = append(rep.Stale, Finding{Block: cur[0], Path: l.ToFile, Title: c.Blocks[cur[0]].Title,
					Message: "a proposed link: the file changed since the link was made"})
			}
		}
	}
	sort.Slice(rep.Stale, func(i, j int) bool {
		return rep.Stale[i].Path+rep.Stale[i].Block < rep.Stale[j].Path+rep.Stale[j].Block
	})
	for _, refs := range c.Refs {
		for _, ref := range refs {
			designed[ref] = true
		}
	}
	isDesigned := func(f string) bool {
		if designed[f] {
			return true
		}
		for d := path.Dir(f); d != "."; d = path.Dir(d) {
			if designed[d+"/"] {
				return true
			}
		}
		return false
	}
	for _, f := range files {
		if IsMarkdown(f) || c.isOutside(f) || isDesigned(f) {
			continue
		}
		rep.Undesigned = append(rep.Undesigned, f)
	}
}

func (a *Analysis) objects() Objects { return a.Chain.src.(interface{ objects() Objects }).objects() }

func (s *treeSource) objects() Objects { return s.r.Objects }
func (s *diskSource) objects() Objects { return s.r.Objects }

// Scope is a block, everything under it, and the detail documents it hands
// on (and theirs).
func (a *Analysis) Scope(hash string) map[string]bool {
	c := a.Chain
	out := map[string]bool{}
	var add func(h string)
	add = func(h string) {
		if out[h] {
			return
		}
		out[h] = true
		for _, k := range c.Children[h] {
			add(k)
		}
		for _, doc := range c.Details[h] {
			if root, ok := c.DocBlock[doc]; ok {
				add(root)
			}
		}
	}
	add(hash)
	return out
}

// Trail is the works done for a part of the chain, newest first: the main
// record of that part. Each work's steps are its details.
func (a *Analysis) Trail(hash string) []Work {
	scope := a.Scope(hash)
	var out []Work
	for i := len(a.hist.works) - 1; i >= 0; i-- {
		w := a.hist.works[i]
		if a.touches(w, scope) {
			out = append(out, w)
		}
	}
	return out
}

func (a *Analysis) touches(w Work, scope map[string]bool) bool {
	bases := append([]string{}, w.Basis...)
	for _, id := range w.Steps {
		bases = append(bases, a.hist.steps[id].Basis...)
	}
	for _, b := range bases {
		if scope[b] {
			return true
		}
		for _, x := range a.hist.current(a.Chain, b) {
			if scope[x] {
				return true
			}
		}
	}
	return false
}

// FileTrail is the works whose steps changed a file, newest first.
func (a *Analysis) FileTrail(ctx context.Context, file string) []Work {
	var out []Work
	for i := len(a.hist.works) - 1; i >= 0; i-- {
		w := a.hist.works[i]
		for _, id := range w.Steps {
			s := a.hist.steps[id]
			if s.Before == "" || s.After == "" {
				continue
			}
			changes, err := DiffTrees(ctx, a.objects(), s.Before, s.After)
			if err == nil && containsPath(changes, file) {
				out = append(out, w)
				break
			}
		}
	}
	return out
}

func containsPath(cs []Change, p string) bool {
	for _, c := range cs {
		if c.Path == p {
			return true
		}
	}
	return false
}

// Steps is a work's steps, in order.
func (a *Analysis) Steps(w Work) []Step {
	out := make([]Step, 0, len(w.Steps))
	for _, id := range w.Steps {
		out = append(out, a.hist.steps[id])
	}
	return out
}

// Before is what a block was before it changed: the blocks lineage says it
// came from (direct predecessors).
func (a *Analysis) Before(hash string) []Match {
	var out []Match
	for old, news := range a.hist.successors {
		for _, n := range news {
			if n == hash {
				out = append(out, Match{Old: old, New: n})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Old < out[j].Old })
	return out
}

// Works is every work, newest first (the one in progress included).
func (a *Analysis) Works() []Work {
	out := make([]Work, len(a.hist.works))
	for i, w := range a.hist.works {
		out[len(out)-1-i] = w
	}
	return out
}
