package tig

import (
	"context"
	"regexp"
	"slices"
	"strings"
)

// Relations is what the documents and the code say about each other: every
// sentence of the design, the commits it names and the files those changed,
// all many-to-many. Whoever works on the project asks for a part of it (a Set)
// instead of reading all of it, and the parts that share commits or most of
// their files are Groups.

// ExtraUndoes names, on a commit that undid others, the commits it undid. An
// undone commit is history: it is in no set and no group.
const ExtraUndoes = "x-undoes"

// Sentence is one line of the design, marked or not.
type Sentence struct {
	Path    string   `json:"path"`
	Block   string   `json:"block"`
	Heading []string `json:"heading"`
	Line    int      `json:"line"` // 1-based
	Text    string   `json:"text"`
	IDs     []string `json:"ids,omitempty"` // commits, by short ID
	// Hash is the hash of the line's text now; Marked is the hash its mark holds
	// (the start of one) and N which of the lines with this text it is.
	Hash   string `json:"hash,omitempty"`
	Marked string `json:"marked,omitempty"`
	N      int    `json:"n,omitempty"`
	// Unsynced: its text is not what its mark holds. Nothing has followed the
	// edit yet.
	Unsynced bool `json:"unsynced,omitempty"`
	// Inferred: it is in the inferred document. Nobody has agreed with it yet.
	Inferred bool `json:"inferred,omitempty"`
	// Intents are what the line means, each with its range. Any number of them
	// may cover the same text.
	Intents []SentenceIntent `json:"intents,omitempty"`
	// Why is, in a set, how it got there: see Set.
	Why string `json:"why,omitempty"`
}

// SentenceIntent is an intent on a line.
type SentenceIntent struct {
	Ref      string `json:"ref"`   // the whole reference: intent, revision, line and range
	Short    string `json:"short"` // the intent's short ID
	Text     string `json:"text"`
	State    string `json:"state"`
	Revision int    `json:"revision"`
	Review   bool   `json:"review,omitempty"`
	By       string `json:"by,omitempty"`
	Via      string `json:"via,omitempty"`
	From     int    `json:"from"`
	To       int    `json:"to"`
	// Commits are the commits that stand on it. A description nobody has agreed
	// with is not one of them: it is only in Links.
	Commits []string `json:"commits,omitempty"`
	// Links are all that is said about code going with it, decided or not.
	Links []SentenceLink `json:"links,omitempty"`
}

// SentenceLink is one work said to go with an intent.
type SentenceLink struct {
	Commit     string   `json:"commit"` // short ID
	Kind       string   `json:"kind"`
	State      string   `json:"state"` // LinkProposed or LinkConfirmed
	Acceptance string   `json:"acceptance"`
	By         string   `json:"by,omitempty"`
	Via        string   `json:"via,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	Files      []string `json:"files,omitempty"`
}

var tableLine = regexp.MustCompile(`^\s*\|`)

// sentencesIn lists a block's own sentences; first is the 1-based line number
// of the first of its lines. Code is not a sentence, nor is a table row unless
// someone marked it.
func sentencesIn(path string, lines []string, fenced []bool, first int) []Sentence {
	var out []Sentence
	for i, line := range lines {
		if i < len(fenced) && fenced[i] || !isTrackedLine(line) {
			continue
		}
		m, marked := readLineMark(line)
		if tableLine.MatchString(line) && !marked {
			continue
		}
		text := LineText(line)
		s := Sentence{Path: path, Line: first + i, Text: text, Hash: LineHash(text), N: 1}
		s.Unsynced = legacyMark.MatchString(maskCode(line))
		if marked {
			s.Marked, s.N = m.Hash, m.N
			s.Unsynced = !strings.HasPrefix(s.Hash, m.Hash)
		}
		out = append(out, s)
	}
	return out
}

// CommitLink is a commit with the sentences and files on either side of it.
type CommitLink struct {
	ID     string `json:"id"`
	Short  string `json:"short"`
	Intent string `json:"intent"`
	Result string `json:"result"`
	// Files are its project files that still exist; a fold's are its children's.
	Files []string `json:"files,omitempty"`
	// Gone are the files it changed that are no longer there.
	Gone []string `json:"gone,omitempty"`
	// Sentences are the indexes in Relations.Sentences of those naming it, or a
	// commit folding it.
	Sentences []int    `json:"sentences,omitempty"`
	Children  []string `json:"children,omitempty"` // short IDs of what it folds
	FoldedIn  string   `json:"foldedIn,omitempty"` // short ID of the commit folding it
	// Undone: a later commit undid it; it is here for the record only.
	Undone bool `json:"undone,omitempty"`
	// Acceptance is where it stands: candidate, accepted or rejected.
	Acceptance string `json:"acceptance,omitempty"`
}

// Baseline reports whether it is the commit for all the project was before tig.
func (c CommitLink) Baseline() bool { return c.Result == ResultBaseline }

// Relations is the whole picture.
type Relations struct {
	Sentences []Sentence   `json:"sentences"`
	Commits   []CommitLink `json:"commits"` // oldest first

	byID   map[string]int
	byFile map[string][]int
	leaves map[string][]string // id → its leaves
	short  *Commits
}

// Relations reads the picture from an analysis.
func (a *Analysis) Relations(ctx context.Context) *Relations {
	r := &Relations{byID: map[string]int{}, byFile: map[string][]int{}, leaves: map[string][]string{}, short: a.commits}
	for _, hash := range a.Chain.Order {
		b := a.Chain.Blocks[hash]
		if !b.IsAcceptance() {
			r.Sentences = append(r.Sentences, b.Sentences...)
		}
	}
	for _, b := range a.Chain.InferredBlocks {
		r.Sentences = append(r.Sentences, b.Sentences...)
	}
	undone := map[string]bool{}
	parent := map[string]string{}
	for _, w := range a.commits.All() {
		for _, id := range w.Extra[ExtraUndoes] {
			undone[id] = true
		}
		for _, k := range w.Children {
			parent[k] = a.commits.Short(w.ID)
		}
	}
	for _, w := range a.commits.All() {
		c := CommitLink{ID: w.ID, Short: a.commits.Short(w.ID), Intent: w.Intent, Result: w.Result, FoldedIn: parent[w.ID], Undone: undone[w.ID], Acceptance: a.Ledger.AcceptanceOf(w)}
		r.leaves[w.ID] = a.commits.Leaves(w.ID)
		for _, k := range w.Children {
			c.Children = append(c.Children, a.commits.Short(k))
		}
		for _, leaf := range r.leaves[w.ID] {
			for _, f := range a.workFiles[leaf] {
				if _, ok := a.Chain.src.hash(ctx, f); ok {
					c.Files = appendNew(c.Files, f)
				} else {
					c.Gone = appendNew(c.Gone, f)
				}
			}
		}
		r.byID[w.ID] = len(r.Commits)
		r.Commits = append(r.Commits, c)
	}
	for i, s := range r.Sentences {
		for _, ref := range s.IDs {
			w, err := a.commits.Resolve(ref)
			if err != nil {
				continue
			}
			for _, id := range append([]string{w.ID}, r.leaves[w.ID]...) {
				c := &r.Commits[r.byID[id]]
				if !slices.Contains(c.Sentences, i) {
					c.Sentences = append(c.Sentences, i)
				}
			}
		}
	}
	for i, c := range r.Commits {
		if r.live(c) {
			for _, f := range c.Files {
				r.byFile[f] = append(r.byFile[f], i)
			}
		}
	}
	return r
}

// counts reports whether a commit stands for code that is there now: not the
// baseline (it stands for everything, so it says nothing), not undone, and not
// a record of undoing.
func (r *Relations) counts(c CommitLink) bool {
	if c.Result == ResultDescribed && len(c.Sentences) == 0 {
		return false // a reading of the code that nobody has agreed with
	}
	return !c.Undone && !c.Baseline() && c.Result != ResultRolledBack && c.Result != ResultAbandoned && c.Acceptance != WorkRejected
}

// live is a commit that counts and holds changes of its own, or has a sentence.
func (r *Relations) live(c CommitLink) bool {
	return r.counts(c) && len(c.Children) == 0 && (len(c.Files) > 0 || len(c.Sentences) > 0)
}

// sentenceLeaves are the commits a sentence stands on that count, and whether
// it names the baseline.
func (r *Relations) sentenceLeaves(s Sentence) (leaves []int, baseline bool) {
	for _, ref := range s.IDs {
		w, err := r.short.Resolve(ref)
		if err != nil {
			continue
		}
		for _, id := range r.leaves[w.ID] {
			i := r.byID[id]
			switch c := r.Commits[i]; {
			case c.Baseline():
				baseline = true
			case r.counts(c) && !slices.Contains(leaves, i):
				leaves = append(leaves, i)
			}
		}
	}
	return leaves, baseline
}

// SetInput is where a set starts from. Any of it may be empty.
type SetInput struct {
	// Intents are references to intents, which still find the line when its
	// words change.
	Intents   []string `json:"intents,omitempty"`
	Sentences []string `json:"sentences,omitempty"` // their words, marks left out
	Commits   []string `json:"commits,omitempty"`   // short IDs, hashes or prefixes
	Files     []string `json:"files,omitempty"`
}

// SetCommit is a commit of a set.
type SetCommit struct {
	Short  string   `json:"short"`
	Intent string   `json:"intent"`
	Files  []string `json:"files,omitempty"`
	Why    string   `json:"why,omitempty"`
}

// How something came to be in a set: asked for, or reached by a commit or a
// file it shares. A shared file says the code depends on it, not that it means
// the same thing.
const (
	WhySeed   = "seed"
	whyCommit = "commit:"
	whyFile   = "file:"
)

// Set is a part of the relations: sentences, the commits they name, the files
// those changed. It is what someone needs in order to work on those sentences,
// and no more. Everything in it says why it is there.
type Set struct {
	Sentences []Sentence  `json:"sentences,omitempty"`
	Commits   []SetCommit `json:"commits,omitempty"`
	Files     []string    `json:"files,omitempty"`
	// FileWhy says why each file is there.
	FileWhy map[string]string `json:"fileWhy,omitempty"`
	// Baseline: a sentence of the set names the baseline, whose files are not
	// listed (they would be the whole project).
	Baseline bool `json:"baseline,omitempty"`
	// Missing is what was asked for and is not there.
	Missing []string `json:"missing,omitempty"`
}

// Set follows the relations one step from where it starts: from a sentence to
// its commits and their files, from a commit to its sentences and files, from
// a file to the commits that changed it and their sentences. It stops there,
// so one file everybody touches does not bring in the whole project.
func (r *Relations) Set(in SetInput) Set {
	out := Set{FileWhy: map[string]string{}}
	sentences := map[int]string{}
	commits := map[int]string{}
	note := func(m map[int]string, i int, why string) {
		if _, ok := m[i]; !ok {
			m[i] = why
		}
	}
	addFile := func(f, why string) {
		if _, ok := out.FileWhy[f]; !ok {
			out.FileWhy[f] = why
		}
	}
	addCommit := func(i int, why string) {
		note(commits, i, why)
		c := r.Commits[i]
		for _, f := range c.Files {
			addFile(f, whyCommit+c.Short)
		}
		for _, s := range c.Sentences {
			if !r.Sentences[s].Inferred {
				note(sentences, s, whyCommit+c.Short)
			}
		}
	}
	fromRefs := func(refs []string, why string) {
		for _, ref := range refs {
			w, err := r.short.Resolve(ref)
			if err != nil {
				continue
			}
			for _, id := range r.leaves[w.ID] {
				i := r.byID[id]
				switch c := r.Commits[i]; {
				case c.Baseline():
					out.Baseline = true
				case r.counts(c):
					addCommit(i, why)
				}
			}
		}
	}
	for _, ref := range in.Intents {
		short := ref
		if p, err := ParseRef(ref); err == nil && p.Kind == RefIntent {
			short = "i" + p.Code
		}
		found := false
		for i, s := range r.Sentences {
			for _, it := range s.Intents {
				if it.Short == short || strings.HasPrefix(it.Short, short) {
					found = true
					sentences[i] = WhySeed
					fromRefs(it.Commits, whyCommit+strings.Join(it.Commits, ","))
				}
			}
		}
		if !found {
			out.Missing = append(out.Missing, ref)
		}
	}
	for _, text := range in.Sentences {
		key, found := sentenceKey(text), false
		for i, s := range r.Sentences {
			match := sentenceKey(s.Text) == key
			for _, it := range s.Intents {
				match = match || sentenceKey(it.Text) == key
			}
			if !match {
				continue
			}
			found = true
			sentences[i] = WhySeed
			leaves, baseline := r.sentenceLeaves(s)
			out.Baseline = out.Baseline || baseline
			for _, c := range leaves {
				addCommit(c, whyCommit+r.Commits[c].Short)
			}
		}
		if !found {
			out.Missing = append(out.Missing, text)
		}
	}
	for _, ref := range in.Commits {
		w, err := r.short.Resolve(ref)
		if err != nil {
			out.Missing = append(out.Missing, ref)
			continue
		}
		c := r.Commits[r.byID[w.ID]]
		out.Baseline = out.Baseline || c.Baseline()
		for _, s := range c.Sentences {
			if !r.Sentences[s].Inferred {
				note(sentences, s, whyCommit+c.Short)
			}
		}
		for _, id := range r.leaves[w.ID] {
			if leaf := r.byID[id]; r.counts(r.Commits[leaf]) {
				addCommit(leaf, WhySeed)
			}
		}
	}
	for _, f := range in.Files {
		out.FileWhy[f] = WhySeed
		for _, c := range r.byFile[f] {
			note(commits, c, whyFile+f)
			for _, s := range r.Commits[c].Sentences {
				if !r.Sentences[s].Inferred {
					note(sentences, s, whyFile+f)
				}
			}
		}
	}
	for i, s := range r.Sentences {
		if why, ok := sentences[i]; ok {
			s.Why = why
			out.Sentences = append(out.Sentences, s)
		}
	}
	for i, c := range r.Commits {
		if why, ok := commits[i]; ok {
			out.Commits = append(out.Commits, SetCommit{Short: c.Short, Intent: c.Intent, Files: c.Files, Why: why})
		}
	}
	for f := range out.FileWhy {
		out.Files = append(out.Files, f)
	}
	slices.Sort(out.Files)
	return out
}

// Kinds of group.
const (
	GroupBound      = "bound"       // sentences on commits
	GroupNoSentence = "no-sentence" // commits no sentence names
	GroupBaseline   = "baseline"    // sentences on the baseline only
	GroupOpen       = "open"        // sentences with no commit yet
)

// Group is sentences, commits and files that belong together.
type Group struct {
	Kind string `json:"kind"`
	// Key stays while the group grows: its oldest commit's short ID, or for
	// sentences without commits, a hash of where they are.
	Key       string     `json:"key"`
	Sentences []Sentence `json:"sentences,omitempty"`
	Commits   []string   `json:"commits,omitempty"` // short IDs, oldest first
	Files     []string   `json:"files,omitempty"`
}

// Groups splits the picture: sentences that share a commit are one group, and
// so are the commits of one fold. Groups sharing most of their files become
// one, not counting the files many groups touch, which say nothing about any
// of them. The baseline joins nothing: sentences on it alone, and sentences
// with no commit, stay together by section.
func (r *Relations) Groups() []Group {
	up := make([]int, len(r.Commits))
	for i := range up {
		up[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if up[i] != i {
			up[i] = find(up[i])
		}
		return up[i]
	}
	// The older commit names the group.
	join := func(a, b int) {
		ra, rb := find(a), find(b)
		up[max(ra, rb)] = min(ra, rb)
	}
	for _, c := range r.Commits {
		first := -1
		for _, id := range r.leaves[c.ID] {
			i := r.byID[id]
			if !r.live(r.Commits[i]) {
				continue
			}
			if first < 0 {
				first = i
			} else {
				join(first, i)
			}
		}
	}
	sections := map[string]*Group{}
	var sectionKeys []string
	bound := make([]int, len(r.Sentences))
	for i, s := range r.Sentences {
		bound[i] = -1
		leaves, baseline := r.sentenceLeaves(s)
		for _, c := range leaves {
			join(leaves[0], c)
		}
		if len(leaves) > 0 {
			bound[i] = leaves[0]
			continue
		}
		kind := GroupOpen
		if baseline {
			kind = GroupBaseline
		}
		key := kind + "\x00" + s.Path + "\x00" + strings.Join(s.Heading, "\x00")
		g := sections[key]
		if g == nil {
			g = &Group{Kind: kind, Key: HashOf([]byte(key))[:8]}
			sections[key] = g
			sectionKeys = append(sectionKeys, key)
		}
		g.Sentences = append(g.Sentences, s)
	}
	r.joinByFiles(find, join)

	byRoot := map[int]*Group{}
	var roots []int
	group := func(root int) *Group {
		g := byRoot[root]
		if g == nil {
			g = &Group{Kind: GroupNoSentence, Key: r.Commits[root].Short}
			byRoot[root] = g
			roots = append(roots, root)
		}
		return g
	}
	for i, c := range r.Commits {
		if !r.live(c) {
			continue
		}
		g := group(find(i))
		short := c.Short
		if c.FoldedIn != "" {
			short = c.FoldedIn
		}
		if !slices.Contains(g.Commits, short) {
			g.Commits = append(g.Commits, short)
		}
		for _, f := range c.Files {
			g.Files = appendNew(g.Files, f)
		}
	}
	for i, s := range r.Sentences {
		if bound[i] >= 0 {
			g := group(find(bound[i]))
			g.Kind = GroupBound
			g.Sentences = append(g.Sentences, s)
		}
	}
	slices.Sort(roots)
	var out []Group
	for _, root := range roots {
		slices.Sort(byRoot[root].Files)
		out = append(out, *byRoot[root])
	}
	for _, key := range sectionKeys {
		out = append(out, *sections[key])
	}
	return out
}

// joinByFiles joins the groups that share at least two files and at least half
// of the smaller one's, until none do.
func (r *Relations) joinByFiles(find func(int) int, join func(a, b int)) {
	for {
		groups := map[int]map[string]bool{}
		for i, c := range r.Commits {
			if !r.live(c) {
				continue
			}
			root := find(i)
			if groups[root] == nil {
				groups[root] = map[string]bool{}
			}
			for _, f := range c.Files {
				groups[root][f] = true
			}
		}
		reach := map[string]int{}
		var roots []int
		for root, files := range groups {
			roots = append(roots, root)
			for f := range files {
				reach[f]++
			}
		}
		slices.Sort(roots)
		common := max(4, len(groups)/4)
		a, b := overlapping(roots, groups, reach, common)
		if a < 0 {
			return
		}
		join(a, b)
	}
}

func overlapping(roots []int, groups map[int]map[string]bool, reach map[string]int, common int) (int, int) {
	for x, a := range roots {
		for _, b := range roots[x+1:] {
			shared := 0
			for f := range groups[a] {
				if groups[b][f] && reach[f] < common {
					shared++
				}
			}
			if shared >= 2 && shared*2 >= min(len(groups[a]), len(groups[b])) {
				return a, b
			}
		}
	}
	return -1, -1
}

// OfFile is the commits that changed a file and count, oldest first.
func (r *Relations) OfFile(file string) []CommitLink {
	var out []CommitLink
	for _, i := range r.byFile[file] {
		out = append(out, r.Commits[i])
	}
	return out
}
