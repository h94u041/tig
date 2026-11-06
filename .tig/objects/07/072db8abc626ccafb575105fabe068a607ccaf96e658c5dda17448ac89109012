package tig_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/h94u041/tig"
)

// fromV010 is a copy of testdata/v0.1.0: a project written by tigwx 0.1.0 as it was
// published, with a work, marks by a person and by an agent, a guess in the inferred
// document, and an edit followed by a sync. It is never regenerated: if this test
// fails, what people already have stopped reading.
func fromV010(t *testing.T) *tig.Repo {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join("testdata", "v0.1.0")
	err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		if head, rest, ok := strings.Cut(filepath.ToSlash(rel), "/"); ok && head == "dot-tig" {
			rel = filepath.Join(".tig", filepath.FromSlash(rest))
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		return os.WriteFile(full, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return tig.Open(root)
}

func TestReadsWhatV010Wrote(t *testing.T) {
	ctx := context.Background()
	r := fromV010(t)
	a, err := r.Analyze(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if !a.Report.Closed {
		t.Fatalf("%+v", a.Report)
	}
	if got := len(a.Ledger.Intents()); got != 3 {
		t.Fatalf("%d intents", got)
	}
	byText := func(words string) *tig.Intent {
		t.Helper()
		for _, i := range a.Ledger.Intents() {
			if strings.Contains(i.Text, words) {
				return i
			}
		}
		t.Fatalf("no intent says %q", words)
		return nil
	}
	if i := byText("Counts up"); i.State != tig.IntentConfirmed || i.By != tig.ByHuman {
		t.Fatalf("%+v", i)
	}
	touched := byText("twenty")
	if touched.State != tig.IntentProposed || touched.By != tig.ByAI || touched.Via != "old-agent" || touched.Revision != 2 {
		t.Fatalf("%+v", touched)
	}
	if len(touched.Forks) != 0 {
		t.Fatalf("%+v", touched.Forks)
	}
	if i := byText("in ones"); i.State != tig.IntentProposed {
		t.Fatalf("%+v", i)
	}
	set := a.Relations(ctx).Set(tig.SetInput{Intents: []string{touched.Short + "#1"}})
	if len(set.Pinned) != 1 || set.Pinned[0].Text != "Stops at ten." || !set.Pinned[0].Stale {
		t.Fatalf("%+v", set)
	}

	// And it can go on from there, with everything that came after.
	reword(t, r, "README.md", "Stops at twenty.", "Stops at thirty.")
	if _, err := r.SyncWith(ctx, tig.SyncInput{Actor: "new", By: tig.ByHuman, Reason: "more room", Source: "ticket 1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Relate(ctx, tig.RelateInput{From: touched.Short, To: byText("Counts up").Short, Kind: tig.RelationDependsOn, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	i := intentOf(t, r, touched.Short)
	if i.Revision != 3 || i.Log[len(i.Log)-1].Kind != tig.IntentRevised {
		t.Fatalf("%+v", i)
	}
}

func putRecord(t *testing.T, r *tig.Repo, text string) string {
	t.Helper()
	id := tig.HashOf([]byte(text))
	if err := os.MkdirAll(filepath.Join(r.Root, ".tig", "ledger"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.Root, ".tig", "ledger", id), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return id
}

// A later version will write things this one does not know. They stay in the
// ledger, unread, and everything else goes on. docs/format.md promises this.
func TestRecordsFromALaterVersion(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nCarts are kept.\n")
	mark(t, r, tig.MarkInput{Sentence: "Carts are kept.", State: tig.IntentConfirmed, By: tig.ByHuman})
	before := sentenceOf(t, r, "Carts")
	short := before.Intents[0].Short
	i := intentOf(t, r, short)
	last := i.Log[len(i.Log)-1].ID

	// A kind of record nobody here has heard of.
	unknown := putRecord(t, r, "tig-something-new 1\nfield value\nat 2030-01-01T00:00:00Z\n")
	// An event with a field nobody here has heard of.
	withField := putRecord(t, r, "tig-intent-event 1\nintent "+i.ID+"\nparent "+last+"\nkind confirmed\nrevision 1\nby human\nfuture-field something\nat 2030-01-02T00:00:00Z\n")
	// An event of a kind nobody here has heard of, in the middle of the chain.
	putRecord(t, r, "tig-intent-event 1\nintent "+i.ID+"\nparent "+withField+"\nkind weighed\nrevision 1\nby human\nat 2030-01-03T00:00:00Z\n")

	got := intentOf(t, r, short)
	if got.State != tig.IntentConfirmed || got.Text != "Carts are kept." || len(got.Log) != 4 {
		t.Fatalf("%+v", got)
	}
	if got.Log[3].Kind != "weighed" {
		t.Fatalf("it is kept and read as nothing: %+v", got.Log[3])
	}
	if _, err := os.Stat(filepath.Join(r.Root, ".tig", "ledger", unknown)); err != nil {
		t.Fatal(err)
	}
	a, err := r.Analyze(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if !a.Report.Closed {
		t.Fatalf("%+v", a.Report)
	}
	// And it goes on after them.
	if _, err := r.RetireIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	if got = intentOf(t, r, short); got.State != tig.IntentDeprecated {
		t.Fatalf("%+v", got)
	}
}

// An intent whose first event is of a kind nobody here knows stands for nothing yet:
// reading it must not fail, and it must not pass for a decision.
func TestAnIntentStartedByALaterVersion(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nCarts are kept.\n")
	id := putRecord(t, r, "tig-intent-event 1\nkind drafted\nrevision 1\ntext A later idea\nat 2030-01-01T00:00:00Z\n")
	l, err := r.Ledger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	i, err := l.Intent(id)
	if err != nil {
		t.Fatal(err)
	}
	if i.Active() || i.State != "" {
		t.Fatalf("%+v", i)
	}
	if _, err := r.Analyze(ctx, ""); err != nil {
		t.Fatal(err)
	}
}
