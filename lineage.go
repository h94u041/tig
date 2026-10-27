package tig

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Lineage is how the text blocks of the documents that changed between two
// consecutive snapshots correspond: which new block came from which old one.
// It is worked out by fixed rules, without a model, so anyone recomputes the
// same lineage from the same trees.
type Lineage struct {
	ID      string   `json:"id"`
	Matches []Match  `json:"matches,omitempty"`
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
}

// Match is one new block that came from an old one.
type Match struct {
	Old   string  `json:"old"`
	New   string  `json:"new"`
	Kind  string  `json:"kind"`  // edit, move, rewrite, split, merge
	Score float64 `json:"score"` // how much of the text is the same, 0 to 1
}

// Kinds of match.
const (
	MatchEdit    = "edit"    // same place, the text changed
	MatchMove    = "move"    // same text, another place (a heading above it was renamed, or it moved)
	MatchRewrite = "rewrite" // both changed, the text is still mostly the same
	MatchSplit   = "split"   // one of the new blocks the old one was split into
	MatchMerge   = "merge"   // one of the old blocks merged into the new one
)

const lineageHeader = "tig-lineage 1"

// Markdown larger than this is not split into blocks.
const maxDocument = 4 << 20

// recordLineage works out and stores the lineage between two trees; "" when
// no document changed.
func (r *Repo) recordLineage(ctx context.Context, oldTree, newTree string) (string, error) {
	changes, err := DiffTrees(ctx, r.Objects, oldTree, newTree)
	if err != nil {
		return "", err
	}
	var l Lineage
	for _, c := range changes {
		if !IsMarkdown(c.Path) {
			continue
		}
		before := blocksAt(ctx, r.Objects, oldTree, c.Path)
		after := blocksAt(ctx, r.Objects, newTree, c.Path)
		m, a, d := MatchBlocks(before, after)
		l.Matches = append(l.Matches, m...)
		l.Added = append(l.Added, a...)
		l.Removed = append(l.Removed, d...)
	}
	if len(l.Matches)+len(l.Added)+len(l.Removed) == 0 {
		return "", nil
	}
	rec := newRecord(lineageHeader)
	for _, m := range l.Matches {
		rec.set("m", m.Old+" "+m.New+" "+m.Kind+" "+strconv.FormatFloat(m.Score, 'f', 3, 64))
	}
	rec.setAll("a", l.Added)
	rec.setAll("r", l.Removed)
	return r.Objects.PutObject(ctx, rec.bytes())
}

func blocksAt(ctx context.Context, objects Objects, tree, path string) []Block {
	e, ok, err := Lookup(ctx, objects, tree, path)
	if err != nil || !ok || e.Size > maxDocument {
		return nil
	}
	data, err := objects.GetObject(ctx, e.Hash)
	if err != nil {
		return nil
	}
	return ParseBlocks(path, data)
}

// LoadLineage reads a lineage record.
func LoadLineage(ctx context.Context, objects Objects, id string) (Lineage, error) {
	raw, err := objects.GetObject(ctx, id)
	if err != nil {
		return Lineage{}, err
	}
	rec, err := parseRecord(raw, lineageHeader)
	if err != nil {
		return Lineage{}, err
	}
	l := Lineage{ID: id}
	for _, v := range rec.all("m") {
		f := strings.Fields(v)
		if len(f) != 4 || !ValidHash(f[0]) || !ValidHash(f[1]) {
			return Lineage{}, fmt.Errorf("tig: lineage %s has an invalid match %q", id, v)
		}
		score, _ := strconv.ParseFloat(f[3], 64)
		l.Matches = append(l.Matches, Match{Old: f[0], New: f[1], Kind: f[2], Score: score})
	}
	for _, h := range rec.all("a") {
		if ValidHash(h) {
			l.Added = append(l.Added, h)
		}
	}
	for _, h := range rec.all("r") {
		if ValidHash(h) {
			l.Removed = append(l.Removed, h)
		}
	}
	return l, nil
}

// MatchBlocks pairs the blocks of one document before and after a change.
// A block whose hash did not change is left out. The rest is matched in
// order: the same headings (edit), the same text (move), mostly the same
// text (rewrite), then a block split into several or several merged into one.
// What is left over was added or removed.
func MatchBlocks(before, after []Block) (matches []Match, added, removed []string) {
	inAfter := map[string]bool{}
	for _, b := range after {
		inAfter[b.Hash] = true
	}
	inBefore := map[string]bool{}
	for _, b := range before {
		inBefore[b.Hash] = true
	}
	var olds, news []Block
	for _, b := range before {
		if !inAfter[b.Hash] {
			olds = append(olds, b)
		}
	}
	for _, b := range after {
		if !inBefore[b.Hash] {
			news = append(news, b)
		}
	}
	oldDone := make([]bool, len(olds))
	newDone := make([]bool, len(news))
	pair := func(i, j int, kind string, score float64) {
		matches = append(matches, Match{Old: olds[i].Hash, New: news[j].Hash, Kind: kind, Score: round3(score)})
		oldDone[i], newDone[j] = true, true
	}
	byKey := func(key func(Block) string, kind string) {
		for i, o := range olds {
			if oldDone[i] {
				continue
			}
			for j, n := range news {
				if !newDone[j] && key(o) != "" && key(o) == key(n) {
					pair(i, j, kind, similarity(o.Text, n.Text))
					break
				}
			}
		}
	}
	byKey(func(b Block) string { return strings.Join(b.Heading, "\x1f") + "\x00" + strconv.Itoa(b.Level) }, MatchEdit)
	byKey(func(b Block) string { return b.Text }, MatchMove)

	// An old block that two or more new, smaller blocks are mostly made of
	// was split into them; two or more old, smaller blocks mostly found in a
	// new one were merged into it. This comes before rewrites, or one part of
	// a split would pass for a rewrite of the whole.
	part := func(small, big string) (float64, bool) {
		s := containment(small, big)
		return s, small != "" && s >= 0.6 && total(bigrams(small)) < total(bigrams(big))*4/5
	}
	for i, o := range olds {
		if oldDone[i] {
			continue
		}
		var parts []int
		for j, n := range news {
			if _, ok := part(n.Text, o.Text); ok && !newDone[j] {
				parts = append(parts, j)
			}
		}
		if len(parts) < 2 {
			continue
		}
		for _, j := range parts {
			s, _ := part(news[j].Text, o.Text)
			matches = append(matches, Match{Old: o.Hash, New: news[j].Hash, Kind: MatchSplit, Score: round3(s)})
			newDone[j] = true
		}
		oldDone[i] = true
	}
	for j, n := range news {
		if newDone[j] {
			continue
		}
		var parts []int
		for i, o := range olds {
			if _, ok := part(o.Text, n.Text); ok && !oldDone[i] {
				parts = append(parts, i)
			}
		}
		if len(parts) < 2 {
			continue
		}
		for _, i := range parts {
			s, _ := part(olds[i].Text, n.Text)
			matches = append(matches, Match{Old: olds[i].Hash, New: n.Hash, Kind: MatchMerge, Score: round3(s)})
			oldDone[i] = true
		}
		newDone[j] = true
	}

	type cand struct {
		i, j  int
		score float64
	}
	var cands []cand
	for i, o := range olds {
		if oldDone[i] {
			continue
		}
		for j, n := range news {
			if newDone[j] {
				continue
			}
			if s := similarity(o.Text, n.Text); s >= 0.5 {
				cands = append(cands, cand{i, j, s})
			}
		}
	}
	sort.SliceStable(cands, func(a, b int) bool { return cands[a].score > cands[b].score })
	for _, c := range cands {
		if !oldDone[c.i] && !newDone[c.j] {
			pair(c.i, c.j, MatchRewrite, c.score)
		}
	}

	// A new block mostly made of an old block's text was split from it; an
	// old block mostly found in a new block was merged into it.
	for j, n := range news {
		if newDone[j] || n.Text == "" {
			continue
		}
		best, bestScore := -1, 0.6
		for i, o := range olds {
			if s := containment(n.Text, o.Text); s >= bestScore {
				best, bestScore = i, s
			}
		}
		if best >= 0 {
			matches = append(matches, Match{Old: olds[best].Hash, New: n.Hash, Kind: MatchSplit, Score: round3(bestScore)})
			newDone[j] = true
		}
	}
	for i, o := range olds {
		if oldDone[i] || o.Text == "" {
			continue
		}
		best, bestScore := -1, 0.6
		for j, n := range news {
			if s := containment(o.Text, n.Text); s >= bestScore {
				best, bestScore = j, s
			}
		}
		if best >= 0 {
			matches = append(matches, Match{Old: o.Hash, New: news[best].Hash, Kind: MatchMerge, Score: round3(bestScore)})
			oldDone[i] = true
		}
	}
	for j, n := range news {
		if !newDone[j] && !splitTarget(matches, n.Hash) {
			added = append(added, n.Hash)
		}
	}
	for i, o := range olds {
		if !oldDone[i] {
			removed = append(removed, o.Hash)
		}
	}
	return matches, added, removed
}

func splitTarget(ms []Match, h string) bool {
	for _, m := range ms {
		if m.New == h {
			return true
		}
	}
	return false
}

func round3(f float64) float64 { return float64(int(f*1000+0.5)) / 1000 }

// bigrams counts each pair of neighbouring characters. Counting characters,
// not words, works the same for text with and without spaces.
func bigrams(s string) map[string]int {
	rs := []rune(strings.Join(strings.Fields(s), " "))
	out := map[string]int{}
	for i := 0; i+1 < len(rs); i++ {
		out[string(rs[i:i+2])]++
	}
	if len(rs) == 1 {
		out[string(rs)]++
	}
	return out
}

func common(a, b map[string]int) int {
	n := 0
	for k, x := range a {
		n += min(x, b[k])
	}
	return n
}

func total(m map[string]int) int {
	n := 0
	for _, x := range m {
		n += x
	}
	return n
}

// similarity is how much of two texts is the same (Dice over bigrams).
func similarity(a, b string) float64 {
	if a == b {
		return 1
	}
	x, y := bigrams(a), bigrams(b)
	if total(x)+total(y) == 0 {
		return 0
	}
	return 2 * float64(common(x, y)) / float64(total(x)+total(y))
}

// containment is how much of a is found in b.
func containment(a, b string) float64 {
	x := bigrams(a)
	if total(x) == 0 {
		return 0
	}
	return float64(common(x, bigrams(b))) / float64(total(x))
}
