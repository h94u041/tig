package tig_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/h94u041/tig"
)

// How long things take as a project grows. It writes a lot and says what it found,
// so it runs only when asked:
//
//	TIG_SCALE=2000,10000 go test -run TestScale -v .
//	TIG_SCALE_PROJECT=/path/to/a/big/folder TIG_SCALE=1 go test -run TestScale -v .
func TestScale(t *testing.T) {
	sizes := os.Getenv("TIG_SCALE")
	if sizes == "" {
		t.Skip("set TIG_SCALE to the numbers of intents to try")
	}
	if dir := os.Getenv("TIG_SCALE_PROJECT"); dir != "" {
		scaleProject(t, dir)
	}
	for _, s := range strings.Split(sizes, ",") {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatal(err)
		}
		if n > 1 {
			scaleIntents(t, n)
		}
	}
}

func timed(t *testing.T, what string, do func() error) {
	t.Helper()
	start := time.Now()
	if err := do(); err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	t.Logf("%-34s %8.0f ms", what, float64(time.Since(start).Microseconds())/1000)
}

// scaleProject takes snapshots of a folder of the size given.
func scaleProject(t *testing.T, dir string) {
	ctx := context.Background()
	r := tig.Open(dir)
	if err := r.Init(tig.Config{Root: "README.md"}); err != nil {
		t.Fatal(err)
	}
	files, err := tig.Files(ctx, dir, r.Skip)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("project: %d files", len(files))
	timed(t, "first snapshot", func() error { _, _, err := r.Take(ctx, tig.ReasonOpen, "t"); return err })
	timed(t, "second snapshot, nothing changed", func() error { _, _, err := r.Take(ctx, tig.ReasonUpdate, "t"); return err })
	timed(t, "analyze", func() error { _, err := r.Analyze(ctx, ""); return err })
	head, err := r.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	timed(t, "what changed", func() error { _, err := r.Changed(ctx, head.Tree); return err })
}

// scaleIntents makes a ledger of n intents directly, n lines of one document, then
// does what a person would do next.
func scaleIntents(t *testing.T, n int) {
	ctx := context.Background()
	root := t.TempDir()
	r := tig.Open(root)
	if err := r.Init(tig.Config{Root: "d.md"}); err != nil {
		t.Fatal(err)
	}
	ledger := filepath.Join(root, ".tig", "ledger")
	if err := os.MkdirAll(ledger, 0o755); err != nil {
		t.Fatal(err)
	}
	var doc strings.Builder
	doc.WriteString("# A\n\n")
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var first string
	for i := 0; i < n; i++ {
		text := fmt.Sprintf("Sentence number %d says one thing.", i)
		hash := tig.LineHash(text)
		if _, err := r.Objects.PutObject(ctx, []byte(text)); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&doc, "%s <tig l=\"%s\"/>\n\n", text, hash[:16])
		rec := fmt.Sprintf("tig-intent-event 1\nkind confirmed\nrevision 1\ntext %s\nspan %s 1 0 %d\nby human\nvia direct\nat %s\n",
			text, hash, len(text), at.Add(time.Duration(i)*time.Second).Format(time.RFC3339))
		id := tig.HashOf([]byte(rec))
		if i == 0 {
			first = id
		}
		if err := os.WriteFile(filepath.Join(ledger, id), []byte(rec), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	doc.WriteString("A line with no intent yet.\n")
	if err := os.WriteFile(filepath.Join(root, "d.md"), []byte(doc.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("--- %d intents, a document of %d KB", n, doc.Len()/1024)
	timed(t, "read the ledger", func() error { _, err := r.Ledger(ctx); return err })
	var a *tig.Analysis
	timed(t, "analyze (check, status)", func() (err error) { a, err = r.Analyze(ctx, ""); return err })
	var rel *tig.Relations
	timed(t, "relations", func() error { rel = a.Relations(ctx); return nil })
	timed(t, "a set of one intent", func() error {
		if set := rel.Set(tig.SetInput{Intents: []string{"i" + first[:8]}}); len(set.Sentences) != 1 {
			return fmt.Errorf("%d sentences", len(set.Sentences))
		}
		return nil
	})
	timed(t, "mark one more line", func() error {
		_, err := r.Mark(ctx, tig.MarkInput{Sentence: "A line with no intent yet.", By: tig.ByHuman})
		return err
	})
	timed(t, "sync, nothing changed", func() error { _, err := r.Sync(ctx, "t"); return err })
	data, err := os.ReadFile(filepath.Join(root, "d.md"))
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "Sentence number 7 says", "Sentence number 7 now says", 1)
	if err := os.WriteFile(filepath.Join(root, "d.md"), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	timed(t, "sync, one line edited", func() error {
		res, err := r.Sync(ctx, "t")
		if err == nil && len(res.Touched) != 1 {
			return fmt.Errorf("%+v", res)
		}
		return err
	})
}
