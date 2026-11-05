package tig_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/h94u041/tig"
)

// relationRepo is a project with a main document and a way to write commits
// that changed given files.
type relationRepo struct {
	t    *testing.T
	r    *tig.Repo
	root string
}

func newRelationRepo(t *testing.T, doc string) *relationRepo {
	t.Helper()
	root := t.TempDir()
	r := tig.Open(root)
	if err := r.Init(tig.Config{Root: "README.md"}); err != nil {
		t.Fatal(err)
	}
	rr := &relationRepo{t: t, r: r, root: root}
	rr.write("README.md", doc)
	return rr
}

func (rr *relationRepo) write(name, content string) {
	rr.t.Helper()
	p := filepath.Join(rr.root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		rr.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		rr.t.Fatal(err)
	}
}

// commit writes a commit whose one step changes files, and returns its short ID.
func (rr *relationRepo) commit(intent string, extra map[string][]string, files ...string) string {
	rr.t.Helper()
	ctx := context.Background()
	before, _, err := rr.r.Take(ctx, tig.ReasonUpdate, "test")
	if err != nil {
		rr.t.Fatal(err)
	}
	for _, f := range files {
		rr.write(f, intent)
	}
	after, _, err := rr.r.Take(ctx, tig.ReasonUpdate, "test")
	if err != nil {
		rr.t.Fatal(err)
	}
	s, err := rr.r.PutStep(ctx, tig.Step{Kind: tig.StepEdit, Before: before.Tree, After: after.Tree, Actor: "test"})
	if err != nil {
		rr.t.Fatal(err)
	}
	w, err := rr.r.PutWork(ctx, tig.Work{Intent: intent, Before: before.ID, After: after.ID, Steps: []string{s.ID}, Result: tig.ResultDone, Actor: "test", Extra: extra})
	if err != nil {
		rr.t.Fatal(err)
	}
	commits, err := rr.r.Commits(ctx)
	if err != nil {
		rr.t.Fatal(err)
	}
	return commits.Short(w.ID)
}

func (rr *relationRepo) mark(sentence, under string, ids ...string) {
	rr.t.Helper()
	if _, err := rr.r.Mark(context.Background(), tig.MarkInput{Sentence: sentence, Under: under, IDs: ids}); err != nil {
		rr.t.Fatal(err)
	}
}

func (rr *relationRepo) relations() *tig.Relations {
	rr.t.Helper()
	ctx := context.Background()
	a, err := rr.r.Analyze(ctx, "")
	if err != nil {
		rr.t.Fatal(err)
	}
	return a.Relations(ctx)
}

func TestRelationsListEverySentence(t *testing.T) {
	rr := newRelationRepo(t, "# Game\n\nIntro line.\n\n## Board\n\n- The board has nine cells.\n\n| a | b |\n|---|---|\n\n```\nnot a sentence\n```\n")
	id := rr.commit("board", nil, "board.go")
	rr.mark("The board has nine cells.", "", id)
	var got []string
	for _, s := range rr.relations().Sentences {
		got = append(got, s.Text)
	}
	if !slices.Equal(got, []string{"Intro line.", "The board has nine cells."}) {
		t.Fatalf("sentences %q", got)
	}
}

func TestSetFollowsOneStep(t *testing.T) {
	rr := newRelationRepo(t, "# Game\n\nThe board has nine cells.\n\nThe player moves.\n\nThe score is kept.\n")
	board := rr.commit("board", nil, "board.go", "shared.go")
	move := rr.commit("move", nil, "move.go", "shared.go")
	score := rr.commit("score", nil, "score.go")
	rr.mark("The board has nine cells.", "", board)
	rr.mark("The player moves.", "", move)
	rr.mark("The score is kept.", "", score)
	rel := rr.relations()

	set := rel.Set(tig.SetInput{Sentences: []string{"The board has nine cells."}})
	if !slices.Equal(set.Files, []string{"board.go", "shared.go"}) {
		t.Fatalf("from a sentence: files %v", set.Files)
	}
	if len(set.Commits) != 1 || set.Commits[0].Short != board {
		t.Fatalf("from a sentence: commits %v", set.Commits)
	}

	set = rel.Set(tig.SetInput{Files: []string{"shared.go"}})
	if len(set.Commits) != 2 || len(set.Sentences) != 2 {
		t.Fatalf("from a file: %+v", set)
	}
	if !slices.Equal(set.Files, []string{"shared.go"}) {
		t.Fatalf("from a file it stops at the sentences: files %v", set.Files)
	}

	set = rel.Set(tig.SetInput{Commits: []string{score}, Sentences: []string{"nobody wrote this"}})
	if !slices.Equal(set.Files, []string{"score.go"}) || !slices.Equal(set.Missing, []string{"nobody wrote this"}) {
		t.Fatalf("from a commit: %+v", set)
	}
}

func TestSetLeavesTheBaselineFilesOut(t *testing.T) {
	rr := newRelationRepo(t, "# Game\n\nThe old part works.\n")
	rr.write("old.go", "old")
	base, err := rr.r.Baseline(context.Background(), "before tig", "test")
	if err != nil {
		t.Fatal(err)
	}
	commits, _ := rr.r.Commits(context.Background())
	rr.mark("The old part works.", "", commits.Short(base.ID))
	set := rr.relations().Set(tig.SetInput{Sentences: []string{"The old part works."}})
	if !set.Baseline || len(set.Files) != 0 {
		t.Fatalf("%+v", set)
	}
}

func TestGroupsByCommitNotByHeading(t *testing.T) {
	rr := newRelationRepo(t, "# Game\n\n## One\n\nFirst sentence.\n\nUnbuilt here.\n\n## Two\n\nSecond sentence.\n\nThird sentence.\n")
	a := rr.commit("first and second", nil, "a.go")
	b := rr.commit("third", nil, "b.go")
	orphan := rr.commit("nobody wrote about this", nil, "c.go")
	rr.mark("First sentence.", "", a)
	rr.mark("Second sentence.", "", a)
	rr.mark("Third sentence.", "", b)

	groups := rr.relations().Groups()
	kinds := map[string][]tig.Group{}
	for _, g := range groups {
		kinds[g.Kind] = append(kinds[g.Kind], g)
	}
	if len(kinds[tig.GroupBound]) != 2 {
		t.Fatalf("bound %+v", kinds[tig.GroupBound])
	}
	first := kinds[tig.GroupBound][0]
	if first.Key != a || len(first.Sentences) != 2 || !slices.Equal(first.Files, []string{"a.go"}) {
		t.Fatalf("one group across two sections: %+v", first)
	}
	if len(kinds[tig.GroupNoSentence]) != 1 || kinds[tig.GroupNoSentence][0].Key != orphan {
		t.Fatalf("no sentence %+v", kinds[tig.GroupNoSentence])
	}
	if len(kinds[tig.GroupOpen]) != 1 || kinds[tig.GroupOpen][0].Sentences[0].Text != "Unbuilt here." {
		t.Fatalf("open %+v", kinds[tig.GroupOpen])
	}
}

func TestGroupsSharingMostFilesJoin(t *testing.T) {
	rr := newRelationRepo(t, "# Game\n\nOne.\n\nTwo.\n\nThree.\n")
	a := rr.commit("one", nil, "x.go", "y.go", "z.go")
	b := rr.commit("two", nil, "x.go", "y.go")
	c := rr.commit("three", nil, "x.go", "other.go")
	rr.mark("One.", "", a)
	rr.mark("Two.", "", b)
	rr.mark("Three.", "", c)
	groups := rr.relations().Groups()
	if len(groups) != 2 {
		t.Fatalf("groups %+v", groups)
	}
	if groups[0].Key != a || !slices.Equal(groups[0].Commits, []string{a, b}) {
		t.Fatalf("the two sharing most files are one, named by the older: %+v", groups[0])
	}
	if groups[1].Key != c {
		t.Fatalf("one shared file does not join: %+v", groups[1])
	}
}

func TestFilesEverybodyTouchesJoinNothing(t *testing.T) {
	rr := newRelationRepo(t, "# Game\n\nOne.\n\nTwo.\n\nThree.\n\nFour.\n")
	for _, s := range []string{"One.", "Two.", "Three.", "Four."} {
		rr.mark(s, "", rr.commit(s, nil, "app.go", "go.mod", s+".go"))
	}
	if groups := rr.relations().Groups(); len(groups) != 4 {
		t.Fatalf("groups %+v", groups)
	}
}

func TestFoldIsOneGroup(t *testing.T) {
	rr := newRelationRepo(t, "# Game\n\nOne.\n\nTwo.\n")
	a := rr.commit("one", nil, "a.go")
	b := rr.commit("two", nil, "b.go")
	rr.mark("One.", "", a)
	rr.mark("Two.", "", b)
	if _, err := rr.r.Fold(context.Background(), tig.FoldInput{Intent: "both", Commits: []string{a, b}, Actor: "test"}); err != nil {
		t.Fatal(err)
	}
	groups := rr.relations().Groups()
	if len(groups) != 1 || len(groups[0].Files) != 2 || groups[0].Key != a {
		t.Fatalf("groups %+v", groups)
	}
}

func TestUndoneCommitsAreInNothing(t *testing.T) {
	rr := newRelationRepo(t, "# Game\n\nOne.\n")
	a := rr.commit("one", nil, "a.go")
	commits, err := rr.r.Commits(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	w, _ := commits.Resolve(a)
	rr.commit("undo one", map[string][]string{tig.ExtraUndoes: {w.ID}}, "a.go")
	rel := rr.relations()
	for _, c := range rel.Commits {
		if c.Short == a && !c.Undone {
			t.Fatal("not marked undone")
		}
	}
	if set := rel.Set(tig.SetInput{Commits: []string{a}}); len(set.Files) != 0 {
		t.Fatalf("set %+v", set)
	}
	for _, g := range rel.Groups() {
		if g.Key == a {
			t.Fatalf("undone commit has a group: %+v", g)
		}
	}
}
