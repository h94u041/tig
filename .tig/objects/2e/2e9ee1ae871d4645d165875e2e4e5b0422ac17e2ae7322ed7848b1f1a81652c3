package tig

import (
	"strings"
)

// Hit is one thing a search found. Found says how: tig searches by keyword
// only, so every hit is an observed match of the words, not a guess at
// meaning.
type Hit struct {
	Kind    string `json:"kind"` // block, work, step, file
	Ref     string `json:"ref"`  // the block, work or step hash, or the file path
	Path    string `json:"path,omitempty"`
	Title   string `json:"title,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	Found   string `json:"found"`
}

// Search finds the blocks, works, steps and files that contain every word of
// the query (case does not matter).
func (a *Analysis) Search(query string, files []string) []Hit {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return nil
	}
	match := func(texts ...string) (bool, string) {
		all := strings.ToLower(strings.Join(texts, "\n"))
		for _, w := range words {
			if !strings.Contains(all, w) {
				return false, ""
			}
		}
		for _, t := range texts {
			for _, line := range strings.Split(t, "\n") {
				if strings.Contains(strings.ToLower(line), words[0]) {
					return true, clip(strings.TrimSpace(line), 160)
				}
			}
		}
		return true, ""
	}
	var out []Hit
	for _, h := range a.Chain.Order {
		b := a.Chain.Blocks[h]
		if ok, snip := match(strings.Join(b.Heading, " / "), b.Text); ok {
			out = append(out, Hit{Kind: "block", Ref: h, Path: b.Path, Title: b.Title, Snippet: snip, Found: "keyword"})
		}
	}
	for _, w := range a.Works() {
		if w.ID == "" {
			continue
		}
		if ok, snip := match(w.Intent); ok {
			out = append(out, Hit{Kind: "work", Ref: w.ID, Title: w.Intent, Snippet: snip, Found: "keyword"})
		}
		for _, s := range a.Steps(w) {
			if ok, snip := match(s.Note, s.Command); ok {
				out = append(out, Hit{Kind: "step", Ref: s.ID, Title: s.Kind, Snippet: snip, Found: "keyword"})
			}
		}
	}
	for _, f := range files {
		if ok, _ := match(f); ok {
			out = append(out, Hit{Kind: "file", Ref: f, Path: f, Found: "keyword"})
		}
	}
	return out
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
