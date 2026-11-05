package tig_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/h94u041/tig"
)

func TestMarksAreNotDesign(t *testing.T) {
	bare := "# A\n\nThe cube turns one layer.\n"
	before := tig.ParseBlocks("d.md", []byte(bare))[1]
	for _, marked := range []string{
		"# A\n\nThe cube turns one layer. <tig l=\"c3d4e5\"/>\n",
		"# A\n\nThe cube turns one layer. <tig l=\"c3d4e5\" n=\"2\" later=\"x\"/>\n",
		"# A\n\n**The cube turns one layer. *a1b2 c3d4***\n",
	} {
		after := tig.ParseBlocks("d.md", []byte(marked))[1]
		if before.Hash != after.Hash || after.Text != before.Text {
			t.Fatalf("a mark changed the block: %q %q", before.Text, after.Text)
		}
	}
}

func TestLineText(t *testing.T) {
	same := []string{
		"Delete the account.",
		"Delete the account.   ",
		"Delete the account. <tig l=\"c3d4e5\"/>",
		"Delete the account. <tig l=\"c3d4e5\" n=\"2\"/>\r",
		"- Delete the account.",
		"  * Delete the account.",
		"1. Delete the account.",
		"> Delete the account.",
		"> - [x] Delete the account.",
		"## Delete the account.",
		"**Delete the account. *a1b2***",
	}
	want := tig.LineHash("Delete the account.")
	for _, line := range same {
		if got := tig.LineHash(tig.LineText(line)); got != want {
			t.Errorf("%q reads as %q", line, tig.LineText(line))
		}
	}
	for _, line := range []string{"**Delete** the account.", "Delete the account!", "`<tig l=\"c3d4e5\"/>` Delete the account."} {
		if got := tig.LineHash(tig.LineText(line)); got == want {
			t.Errorf("%q is a different line", line)
		}
	}
}

func TestLineMarkPlaces(t *testing.T) {
	ctx := context.Background()
	cases := []struct{ name, line, sentence, wantHas string }{
		{"table", "| a | b |", "| a | b |", ` |`},
		{"hard break", "first \\", "first \\", `/>\`},
		{"two spaces", "first  ", "first", `/>  `},
		{"list", "- item", "item", "- item <tig"},
	}
	for _, c := range cases {
		r := tig.Open(t.TempDir())
		if err := r.Init(tig.Config{Root: "d.md"}); err != nil {
			t.Fatal(err)
		}
		write(t, r.Root, "d.md", "# A\n\n"+c.line+"\nmore\n")
		m, err := r.Mark(ctx, tig.MarkInput{Sentence: c.sentence})
		if err != nil {
			t.Fatal(c.name, err)
		}
		data, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
		line := strings.Split(string(data), "\n")[2]
		if !strings.Contains(line, c.wantHas) || !strings.Contains(line, `<tig l="`+m.Hash[:6]+`"/>`) {
			t.Errorf("%s: %q", c.name, line)
		}
		if c.name == "table" && !strings.HasSuffix(line, "/> |") {
			t.Errorf("a table's mark stays in its last cell: %q", line)
		}
	}
}

func TestParseRef(t *testing.T) {
	for _, s := range []string{"i7f3a", "i7f3a#2", "lc3d4e5", "lc3d4e5:5-14", "lc3d4e5.2", "lc3d4e5.2:0-3", "i7f3a#2@lc3d4e5:5-14", "i7f3a@lc3d4e5.2:0-1", "a1b2", "ev91c0", "e91c"} {
		r, err := tig.ParseRef(s)
		if err != nil || r.String() != s {
			t.Errorf("%q: %+v %v", s, r, err)
		}
	}
	if r, _ := tig.ParseRef("i7f3a#2@lc3d4e5:5-14"); r.Kind != tig.RefIntent || r.Revision != 2 || r.Line != "c3d4e5" || r.From != 5 || r.To != 14 {
		t.Fatalf("%+v", r)
	}
	if r, _ := tig.ParseRef("e91c"); r.Kind != tig.RefWork {
		t.Fatal("a work's hash can start with e; an event has ev")
	}
	for _, s := range []string{"", "i7f", "lc3d4", "lc3d4e5:9-1", "i7f3a#0", "i7f3a@a1b2", "zzzz", "abc", "a1B2"} {
		if r, err := tig.ParseRef(s); err == nil {
			t.Errorf("%q is no reference: %+v", s, r)
		}
	}
}

// oneCommit writes a commit that changes a file.
func oneCommit(t *testing.T, r *tig.Repo, root, file, content, intent string) tig.Work {
	t.Helper()
	ctx := context.Background()
	if _, err := r.StartWork(ctx, nil, intent, "agent"); err != nil {
		t.Fatal(err)
	}
	write(t, root, file, content)
	if _, err := r.RecordStep(ctx, tig.StepInput{Kind: tig.StepEdit}); err != nil {
		t.Fatal(err)
	}
	w, err := r.FinishWork(ctx, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func codes(fs []tig.Finding) []string {
	var out []string
	for _, f := range fs {
		if f.Code != "" {
			out = append(out, f.Code)
		}
	}
	return out
}

func TestShortIDs(t *testing.T) {
	a := strings.Repeat("a", 64)
	b := "aaaa" + strings.Repeat("b", 60)
	c := "aaaab" + strings.Repeat("c", 59)
	commits := tig.NewCommits([]tig.Work{{ID: a}, {ID: b}, {ID: c}})
	if commits.Short(a) != "aaaa" || commits.Short(b) != "aaaab" || commits.Short(c) != "aaaabc" {
		t.Fatal(commits.Short(a), commits.Short(b), commits.Short(c))
	}
	for short, id := range map[string]string{"aaaa": a, "aaaab": b, "aaaabc": c} {
		if w, err := commits.Resolve(short); err != nil || w.ID != id {
			t.Fatal(short, w.ID, err)
		}
	}
	if _, err := commits.Resolve("zzzz"); err == nil {
		t.Fatal("no such commit")
	}
	if _, err := commits.Resolve("aaa"); err == nil {
		t.Fatal("at least four")
	}
	if w, err := commits.Resolve("aaaabb"); err != nil || w.ID != b {
		t.Fatal("a longer prefix still resolves", err)
	}
}

func TestMarkFoldAndCommitView(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	write(t, root, "README.md", "# A\n\nIt turns.\n")
	r := open(t, root)
	one := oneCommit(t, r, root, "a.go", "package a\n", "turn")
	two := oneCommit(t, r, root, "b.go", "package b\n", "turn faster")
	commits, _ := r.Commits(ctx)
	s1, s2 := commits.Short(one.ID), commits.Short(two.ID)
	if len(s1) != 4 {
		t.Fatal(s1)
	}
	m, err := r.Mark(ctx, tig.MarkInput{Sentence: "It turns.", IDs: []string{s1, two.ID}})
	if err != nil || !slices.Equal(m.IDs, []string{s1, s2}) {
		t.Fatalf("%+v %v", m, err)
	}
	if _, err := r.Mark(ctx, tig.MarkInput{Sentence: "It turns.", IDs: []string{"ffff"}}); err == nil {
		t.Fatal("no such commit")
	}
	a, err := r.Analyze(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if a.Report.Marks != 1 || len(a.Report.Unbound) != 0 {
		t.Fatalf("%+v", a.Report)
	}
	b := a.Marks()[0].Block
	if built := a.Built[b]; !slices.Contains(built, "a.go") || !slices.Contains(built, "b.go") {
		t.Fatal("what the commits built counts for the block:", built)
	}
	if len(a.Report.Undesigned) != 0 {
		t.Fatal("a mark reaches its files:", a.Report.Undesigned)
	}
	if got := a.CommitsOf(b); len(got) != 2 || got[0].ID != one.ID {
		t.Fatalf("%+v", got)
	}
	if got := a.Trail(b); len(got) != 2 || got[0].ID != two.ID {
		t.Fatalf("trail: %+v", got)
	}

	parent, err := r.Fold(ctx, tig.FoldInput{Commits: []string{s1, s2}, Intent: "turning", Actor: "agent"})
	if err != nil {
		t.Fatal(parent, err)
	}
	commits, _ = r.Commits(ctx)
	short := commits.Short(parent.ID)
	a, _ = r.Analyze(ctx, "")
	if got := a.Marks()[0].IDs; !slices.Equal(got, []string{short}) {
		t.Fatalf("a line shows the fold: %v", got)
	}
	v, err := a.Commit(short)
	if err != nil || !slices.Equal(v.ChildIDs, []string{s1, s2}) || !slices.Equal(v.Files, []string{"a.go", "b.go"}) || len(v.Marks) != 1 || v.Result != tig.ResultFolded {
		t.Fatalf("%+v %v", v, err)
	}
	if w, err := a.Commit(s1); err != nil || w.ID != one.ID || len(w.Marks) != 1 {
		t.Fatalf("a folded commit is still there: %+v %v", w, err)
	}
	if len(a.Report.Unbound) != 0 || !slices.Contains(a.Built[a.Marks()[0].Block], "b.go") {
		t.Fatalf("%+v", a.Report)
	}
	if _, err := r.Fold(ctx, tig.FoldInput{Commits: []string{s1, s1}}); err == nil {
		t.Fatal("two commits at least")
	}
}

func TestMarkFindings(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	write(t, root, "README.md", "# A\n\nIt turns.\n\nBroken <tig l=\"dead00\"/>\n")
	r := open(t, root)
	w := oneCommit(t, r, root, "a.go", "package a\n", "turn")
	free := oneCommit(t, r, root, "b.go", "package b\n", "not in the design")
	if _, err := r.Mark(ctx, tig.MarkInput{Sentence: "It turns.", IDs: []string{w.ID}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Take(ctx, tig.ReasonScan, ""); err != nil {
		t.Fatal(err)
	}
	a, _ := r.Analyze(ctx, "")
	if got := codes(a.Report.Broken); !slices.Equal(got, []string{tig.FindBrokenMark}) {
		t.Fatal(got, a.Report.Broken)
	}
	if len(codes(a.Report.Stale)) != 0 {
		t.Fatalf("nothing moved yet: %+v", a.Report.Stale)
	}
	commits, _ := r.Commits(ctx)
	if !slices.Equal(a.Report.Unbound, []string{commits.Short(free.ID)}) {
		t.Fatal(a.Report.Unbound)
	}

	// The code moves outside any commit.
	write(t, root, "a.go", "package a // changed by hand\n")
	a, _ = r.Analyze(ctx, "")
	if got := codes(a.Report.Stale); !slices.Equal(got, []string{tig.FindCodeAway}) {
		t.Fatal(got)
	}
	// A commit brings it back into the footprint.
	follow := oneCommit(t, r, root, "a.go", "package a // and committed\n", "follow")
	a, _ = r.Analyze(ctx, "")
	if got := codes(a.Report.Stale); len(got) != 0 {
		t.Fatal("the file is in the footprint again:", got)
	}

	// The sentence is reworded: seen before and after the ledger catches up.
	data, _ := os.ReadFile(filepath.Join(root, "README.md"))
	write(t, root, "README.md", strings.ReplaceAll(strings.ReplaceAll(string(data), "It turns.", "It turns twice."), "Broken <tig l=\"dead00\"/>\n", ""))
	a, _ = r.Analyze(ctx, "")
	if got := codes(a.Report.Stale); !slices.Equal(got, []string{tig.FindIntentChanged}) {
		t.Fatal(got)
	}
	if !a.Chain.Blocks[a.Marks()[0].Block].Sentences[0].Unsynced {
		t.Fatal("it is not synced yet")
	}
	if _, err := r.Sync(ctx, "test"); err != nil {
		t.Fatal(err)
	}
	a, _ = r.Analyze(ctx, "")
	if got := codes(a.Report.Stale); !slices.Equal(got, []string{tig.FindIntentChanged}) {
		t.Fatal(got)
	}
	// A new commit joining the sentence answers it.
	if _, err := r.Mark(ctx, tig.MarkInput{Sentence: "It turns twice.", IDs: []string{follow.ID}}); err != nil {
		t.Fatal(err)
	}
	a, _ = r.Analyze(ctx, "")
	if got := codes(a.Report.Stale); len(got) != 0 {
		t.Fatal("a commit followed the sentence:", got)
	}
}

func TestBaseline(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	write(t, root, "README.md", "# A\n\nThe old part.\n")
	write(t, root, "old.go", "package old\n")
	r := open(t, root)
	base, err := r.Baseline(ctx, "before tig", "user")
	if err != nil || base.Result != tig.ResultBaseline {
		t.Fatal(base, err)
	}
	if _, err := r.Baseline(ctx, "again", "user"); err == nil {
		t.Fatal("one baseline")
	}
	a, _ := r.Analyze(ctx, "")
	if len(a.Report.Unbound) != 0 || !slices.Contains(a.Report.Undesigned, "old.go") {
		t.Fatalf("no design reaches it yet: %+v", a.Report)
	}
	if _, err := r.Mark(ctx, tig.MarkInput{Sentence: "The old part.", IDs: []string{base.ID}}); err != nil {
		t.Fatal(err)
	}
	a, _ = r.Analyze(ctx, "")
	if len(a.Report.Undesigned) != 0 || !slices.Contains(a.Built[a.Marks()[0].Block], "old.go") {
		t.Fatalf("the sentence reaches the old code: %+v", a.Report)
	}
}

func TestSessionsWorkAtOnce(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	write(t, root, "README.md", "# A\n")
	r := open(t, root)
	alice, bob := *r, *r
	alice.Session, bob.Session = "alice", "bob"
	if _, err := alice.StartWork(ctx, nil, "one", "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := bob.StartWork(ctx, nil, "two", "bob"); err != nil {
		t.Fatal("the second session is not held up:", err)
	}
	if _, err := alice.StartWork(ctx, nil, "again", "alice"); !errors.Is(err, tig.ErrWorkOpen) {
		t.Fatal("one work a session:", err)
	}
	bad := *r
	bad.Session = "../up"
	if _, err := bad.StartWork(ctx, nil, "x", ""); err == nil {
		t.Fatal("a session name stays in the cache")
	}
	if open, err := r.OpenWorks(); err != nil || len(open) != 2 {
		t.Fatal(open, err)
	}
	w1, err := alice.FinishWork(ctx, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	w2, err := bob.FinishWork(ctx, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	works, _ := r.Works(ctx, 0)
	if len(works) != 2 || works[0].ID != w2.ID || works[1].ID != w1.ID {
		t.Fatalf("no work lost: %+v", works)
	}
	if open, _ := r.OpenWorks(); len(open) != 0 {
		t.Fatal(open)
	}
}

func TestMarkMakesTheFolderAndUnmarkKeepsTheText(t *testing.T) {
	ctx := context.Background()
	r := tig.Open(t.TempDir())
	if err := r.Init(tig.Config{Root: ".tealprint/tealprint.md"}); err != nil {
		t.Fatal(err)
	}
	w, err := r.PutWork(ctx, tig.Work{Intent: "x", Result: tig.ResultDone})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Mark(ctx, tig.MarkInput{Sentence: "第一句", IDs: []string{w.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := r.Unmark(ctx, []string{w.ID}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(r.Root, ".tealprint", "tealprint.md"))
	if !strings.Contains(string(raw), "第一句") {
		t.Fatalf("doc: %q", raw)
	}
	a, _ := r.Analyze(ctx, "")
	if a.Report.Marks != 0 {
		t.Fatalf("the link is withdrawn: %+v", a.Marks())
	}
}

// A sentence with no commit yet goes into the design: under its section, or a
// new one, and only once.
func TestAddSentence(t *testing.T) {
	doc := []byte("# A\n\n## Parts\n\nOne.\n\n## Other\n\nTwo.\n")
	got, err := tig.AddSentence("d.md", doc, "Three.", "Parts")
	if err != nil {
		t.Fatal(err)
	}
	if s := string(got); !strings.Contains(s, "One.") || strings.Index(s, "Three.") > strings.Index(s, "## Other") || strings.Index(s, "Three.") < strings.Index(s, "## Parts") {
		t.Fatalf("under Parts:\n%s", s)
	}
	again, err := tig.AddSentence("d.md", got, "Three.", "Parts")
	if err != nil || string(again) != string(got) {
		t.Fatalf("written twice:\n%s (%v)", again, err)
	}
	made, err := tig.AddSentence("d.md", doc, "Four.", "New")
	if err != nil {
		t.Fatal(err)
	}
	if s := string(made); !strings.Contains(s, "## New") || strings.Index(s, "Four.") < strings.Index(s, "## New") {
		t.Fatalf("a new section:\n%s", s)
	}
	fresh, err := tig.AddSentence("d.md", nil, "Five.", "")
	if err != nil || !strings.Contains(string(fresh), "Five.") {
		t.Fatalf("an empty document: %q %v", fresh, err)
	}
}
