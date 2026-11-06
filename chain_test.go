package tig_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/h94u041/tig"
)

const readme = `# Cube game

A cube you can turn in the browser.

## Turning

How layers turn: see [turning](docs/turn.md).

## Scramble

Random moves from [the code](src/scramble.js).

### Acceptance

- ` + "`echo scramble-ok`" + `

## Acceptance

- ` + "`echo game-ok`" + `
`

const turn = `# Turning

Turn one layer.

## 驗收

- ` + "`echo turn-ok`" + `
`

func cubeProject(t *testing.T) (*tig.Repo, string) {
	root := t.TempDir()
	write(t, root, "README.md", readme)
	write(t, root, "docs/turn.md", turn)
	write(t, root, "docs/orphan.md", "# Nobody links here\n")
	write(t, root, "src/scramble.js", "scramble()\n")
	write(t, root, "src/turn.js", "turn()\n")
	return open(t, root), root
}

func find(t *testing.T, c *tig.Chain, path, title string) *tig.Block {
	t.Helper()
	for _, h := range c.Order {
		if b := c.Blocks[h]; b.Path == path && b.Title == title {
			return b
		}
	}
	t.Fatalf("no block %q in %s", title, path)
	return nil
}

func messages(fs []tig.Finding) string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Path+": "+f.Message)
	}
	return strings.Join(out, "\n")
}

func TestChainClosure(t *testing.T) {
	ctx := context.Background()
	r, root := cubeProject(t)

	a, err := r.Analyze(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	c, rep := a.Chain, a.Report
	if strings.Join(c.Docs, ",") != "README.md,docs/turn.md" {
		t.Fatalf("docs: %v", c.Docs)
	}
	if len(rep.Broken) != 1 || rep.Broken[0].Path != "docs/orphan.md" {
		t.Fatalf("broken: %s", messages(rep.Broken))
	}
	if strings.Join(rep.Undesigned, ",") != "src/turn.js" {
		t.Fatalf("undesigned: %v", rep.Undesigned)
	}
	turning := find(t, c, "README.md", "Turning")
	if rep.Closed {
		t.Fatal("not closed yet")
	}

	// An agent builds turn.js for the turning document and reports its checks.
	turnBlock := find(t, c, "docs/turn.md", "Turning")
	r.StartWork(ctx, []string{turnBlock.Hash}, "build turning", "agent")
	write(t, root, "src/turn.js", "turn(layer)\n")
	r.RecordStep(ctx, tig.StepInput{Kind: tig.StepEdit})
	if _, err := r.FinishWork(ctx, "", nil); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(root, "docs/orphan.md"))

	a, _ = r.Analyze(ctx, "")
	rep = a.Report
	if !rep.Closed {
		t.Fatalf("broken %s\nstale %s\nundesigned %v", messages(rep.Broken), messages(rep.Stale), rep.Undesigned)
	}
	if built := a.Built[turnBlock.Hash]; len(built) != 1 || built[0] != "src/turn.js" {
		t.Fatalf("built: %v", a.Built)
	}
	if trail := a.Trail(turning.Hash); len(trail) != 1 || trail[0].Intent != "build turning" {
		t.Fatal("the main document's Turning section sees the work done for its detail document")
	}
	if len(a.FileTrail(ctx, "src/turn.js")) != 1 || len(a.FileTrail(ctx, "src/scramble.js")) != 0 {
		t.Fatal("a file's trail is the works that changed it")
	}

	hits := a.Search("turn LAYER", []string{"src/turn.js"})
	if len(hits) != 2 || hits[1].Ref != turnBlock.Hash {
		t.Fatalf("search: %+v", hits)
	}
	if hits := a.Search("turn.js", []string{"src/turn.js"}); len(hits) != 1 || hits[0].Kind != "file" {
		t.Fatalf("search files: %+v", hits)
	}
}

func TestStaleLinks(t *testing.T) {
	ctx := context.Background()
	r, root := cubeProject(t)
	os.Remove(filepath.Join(root, "docs/orphan.md"))
	c, _ := r.Chain(ctx, "")
	turnBlock := find(t, c, "docs/turn.md", "Turning")
	r.StartWork(ctx, []string{turnBlock.Hash}, "build turning", "agent")
	write(t, root, "src/turn.js", "turn(layer)\n")
	r.RecordStep(ctx, tig.StepInput{Kind: tig.StepEdit})
	r.FinishWork(ctx, "", nil)

	// The design changes; the code does not follow.
	write(t, root, "docs/turn.md", strings.Replace(turn, "Turn one layer.", "Turn one layer, or the whole cube.", 1))
	a, _ := r.Analyze(ctx, "")
	if len(a.Report.Stale) != 1 || !strings.Contains(a.Report.Stale[0].Message, "design changed") || a.Report.Stale[0].Path != "src/turn.js" {
		t.Fatalf("stale after the design changed: %s", messages(a.Report.Stale))
	}
	// Building it again for the new design follows it.
	newBlock := find(t, a.Chain, "docs/turn.md", "Turning")
	r.StartWork(ctx, []string{newBlock.Hash}, "turn the whole cube", "agent")
	write(t, root, "src/turn.js", "turn(layer)\nturnCube()\n")
	r.RecordStep(ctx, tig.StepInput{Kind: tig.StepEdit})
	r.FinishWork(ctx, "", nil)
	if a, _ = r.Analyze(ctx, ""); len(a.Report.Stale) != 0 {
		t.Fatalf("followed: %s", messages(a.Report.Stale))
	}
	if len(a.Before(newBlock.Hash)) != 1 {
		t.Fatal("the new block knows what it was")
	}

	// A work that changes its own design, then the code: the code follows
	// the design the work leaves, while it is in progress and after.
	r.StartWork(ctx, []string{newBlock.Hash}, "turn faces too", "agent")
	write(t, root, "docs/turn.md", strings.Replace(turn, "Turn one layer.", "Turn one layer, the whole cube, or a face.", 1))
	r.RecordStep(ctx, tig.StepInput{Kind: tig.StepEdit})
	write(t, root, "src/turn.js", "turn(layer)\nturnCube()\nturnFace()\n")
	r.RecordStep(ctx, tig.StepInput{Kind: tig.StepEdit})
	if a, _ = r.Analyze(ctx, ""); len(a.Report.Stale) != 0 {
		t.Fatalf("in progress: %s", messages(a.Report.Stale))
	}
	r.FinishWork(ctx, "", nil)
	if a, _ = r.Analyze(ctx, ""); len(a.Report.Stale) != 0 {
		t.Fatalf("finished: %s", messages(a.Report.Stale))
	}

	// The code changes outside the footprint.
	write(t, root, "src/turn.js", "hand edit\n")
	a, _ = r.Analyze(ctx, "")
	if len(a.Report.Stale) != 1 || !strings.Contains(a.Report.Stale[0].Message, "outside the footprint") {
		t.Fatalf("stale after a hand edit: %s", messages(a.Report.Stale))
	}
}

func TestBroken(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	write(t, root, "README.md", "# Game\n\nNo acceptance here. See [missing](docs/missing.md) and [detail](docs/detail.md).\n\n## Rules\n\nSome rules.\n")
	write(t, root, "docs/detail.md", "# Detail\n\nDesign without acceptance.\n")
	write(t, root, "docs/empty.md", "")
	write(t, root, "notes/draft.md", "# Draft\n")
	r := open(t, root)
	r.SetConfig(tig.Config{Root: "README.md", Outside: []string{"notes"}})
	a, _ := r.Analyze(ctx, "")
	rep := a.Report
	if len(rep.Broken) != 2 || !strings.Contains(messages(rep.Broken), "docs/missing.md, which does not exist") || !strings.Contains(messages(rep.Broken), "docs/empty.md") {
		t.Fatalf("broken: %s", messages(rep.Broken))
	}
	if rep.Closed {
		t.Fatal("not closed")
	}
}

// A file a work took away is not a file that failed to follow its design.
func TestDeletedFileDoesNotGoStale(t *testing.T) {
	ctx := context.Background()
	r, root := cubeProject(t)
	os.Remove(filepath.Join(root, "docs/orphan.md"))
	c, _ := r.Chain(ctx, "")
	turnBlock := find(t, c, "docs/turn.md", "Turning")
	r.StartWork(ctx, []string{turnBlock.Hash}, "build turning", "agent")
	write(t, root, "src/helper.js", "help()\n")
	r.RecordStep(ctx, tig.StepInput{Kind: tig.StepEdit})
	os.Remove(filepath.Join(root, "src/helper.js"))
	r.RecordStep(ctx, tig.StepInput{Kind: tig.StepEdit})
	r.FinishWork(ctx, "", nil)

	write(t, root, "docs/turn.md", strings.Replace(turn, "Turn one layer.", "Turn one layer, or the whole cube.", 1))
	a, _ := r.Analyze(ctx, "")
	for _, f := range a.Report.Stale {
		if f.Path == "src/helper.js" {
			t.Fatalf("a deleted file is reported: %s", messages(a.Report.Stale))
		}
	}
}
