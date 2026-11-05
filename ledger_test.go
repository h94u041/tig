package tig_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/h94u041/tig"
)

func ledgerRepo(t *testing.T, doc string) *tig.Repo {
	t.Helper()
	r := tig.Open(t.TempDir())
	if err := r.Init(tig.Config{Root: "d.md"}); err != nil {
		t.Fatal(err)
	}
	write(t, r.Root, "d.md", doc)
	return r
}

func sentenceOf(t *testing.T, r *tig.Repo, words string) tig.Sentence {
	t.Helper()
	ctx := context.Background()
	a, err := r.Analyze(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range a.Relations(ctx).Sentences {
		if strings.Contains(s.Text, words) {
			return s
		}
	}
	t.Fatalf("no sentence with %q", words)
	return tig.Sentence{}
}

func mark(t *testing.T, r *tig.Repo, in tig.MarkInput) tig.Mark {
	t.Helper()
	m, err := r.Mark(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// One piece of text may belong to any number of intents, and they may overlap.
func TestOverlappingIntents(t *testing.T) {
	r := ledgerRepo(t, "# A\n\nUsers can delete their account for good.\n")
	mark(t, r, tig.MarkInput{Sentence: "Users can delete their account"})
	mark(t, r, tig.MarkInput{Sentence: "delete their account for good"})
	s := sentenceOf(t, r, "Users")
	if s.Text != "Users can delete their account for good." || len(s.Intents) != 2 {
		t.Fatalf("%+v", s)
	}
	if a, b := s.Intents[0], s.Intents[1]; a.From != 0 || a.To != 30 || b.From != 10 || b.To != 39 || a.Text != "Users can delete their account" {
		t.Fatalf("%+v %+v", a, b)
	}
	// Marking the same range again is the same intent.
	mark(t, r, tig.MarkInput{Sentence: "delete their account for good"})
	if s := sentenceOf(t, r, "Users"); len(s.Intents) != 2 {
		t.Fatalf("%+v", s.Intents)
	}
	data, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	if strings.Count(string(data), "<tig") != 1 {
		t.Fatalf("one mark a line: %s", data)
	}
}

// An edit asks again about the intents it reaches, and only those.
func TestEditTouchesOnlyWhatItReaches(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nDelete the account, and keep the invoices.\n")
	mark(t, r, tig.MarkInput{Sentence: "Delete the account", State: tig.IntentConfirmed, By: tig.ByHuman})
	mark(t, r, tig.MarkInput{Sentence: "keep the invoices", State: tig.IntentConfirmed, By: tig.ByHuman})
	data, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	write(t, r.Root, "d.md", strings.Replace(string(data), "keep the invoices", "keep the old invoices", 1))
	if s := sentenceOf(t, r, "Delete"); !s.Unsynced || len(s.Intents) != 2 {
		t.Fatalf("before the sync it is known by the old hash: %+v", s)
	}
	res, err := r.Sync(ctx, "test")
	if err != nil || len(res.Touched) != 1 || len(res.Moved) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	s := sentenceOf(t, r, "Delete")
	if s.Unsynced || len(s.Intents) != 2 {
		t.Fatalf("%+v", s)
	}
	first, second := s.Intents[0], s.Intents[1]
	if first.Revision != 1 || first.Review || first.Text != "Delete the account" {
		t.Fatalf("not reached: %+v", first)
	}
	if second.Revision != 2 || !second.Review || second.Text != "keep the old invoices" {
		t.Fatalf("reached: %+v", second)
	}
	if got := sliceOf(s.Text, second.From, second.To); got != "keep the old invoices" {
		t.Fatalf("the range follows: %q", got)
	}
	// Confirming answers it. The old revision still reads.
	if _, err := r.ConfirmIntent(ctx, tig.DecideInput{Intent: second.Short, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceOf(t, r, "Delete"); s.Intents[1].Review {
		t.Fatalf("%+v", s.Intents[1])
	}
	v, err := r.Show(ctx, second.Short+"#1")
	if err != nil || v.Intent.Text != "keep the invoices" || v.Intent.Current != 2 {
		t.Fatalf("%+v %v", v.Intent, err)
	}
	v, err = r.Show(ctx, second.Short)
	if err != nil || v.Intent.Previous == nil || v.Intent.Previous.Text != "keep the invoices" || v.Intent.Sync != tig.SyncOK || len(v.Intent.Places) != 1 {
		t.Fatalf("%+v %v", v.Intent, err)
	}
}

func sliceOf(s string, from, to int) string { return string([]rune(s)[from:to]) }

// A range of the whole line is touched by whatever is added to the line.
func TestAddingToAWholeLineTouchesIt(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nDelete the account.\n")
	mark(t, r, tig.MarkInput{Sentence: "Delete the account.", State: tig.IntentConfirmed})
	data, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	write(t, r.Root, "d.md", strings.Replace(string(data), "account.", "account. Except for admins.", 1))
	res, err := r.Sync(ctx, "test")
	if err != nil || len(res.Touched) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	if s := sentenceOf(t, r, "Delete"); s.Intents[0].Text != s.Text {
		t.Fatalf("%+v", s.Intents[0])
	}
}

// What only places a line is not an edit of it.
func TestMovingALineIsNotAnEdit(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nDelete the account.\n")
	mark(t, r, tig.MarkInput{Sentence: "Delete the account.", State: tig.IntentConfirmed})
	data, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	write(t, r.Root, "d.md", "# B\n\nNew first line.\n\n## C\n\n"+strings.Replace(strings.Split(string(data), "\n")[2], "Delete", "- Delete", 1)+"\n")
	res, err := r.Sync(ctx, "test")
	if err != nil || len(res.Touched)+len(res.Moved)+len(res.Docs) != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	if s := sentenceOf(t, r, "Delete"); len(s.Intents) != 1 || s.Line != 7 {
		t.Fatalf("%+v", s)
	}
}

func TestSameTextTwiceIsTwoLines(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nThe same words.\n\n# B\n\nThe same words.\n")
	mark(t, r, tig.MarkInput{Sentence: "The same words."})
	data, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	lines := strings.Split(string(data), "\n")
	// The second copy gets a mark of its own when someone copies the first's.
	lines[6] = lines[2]
	write(t, r.Root, "d.md", strings.Join(lines, "\n"))
	if _, err := r.Sync(ctx, "test"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(filepath.Join(r.Root, "d.md"))
	if !strings.Contains(string(data), `n="2"`) {
		t.Fatalf("%s", data)
	}
	a, _ := r.Analyze(ctx, "")
	var covered int
	for _, s := range a.Relations(ctx).Sentences {
		covered += len(s.Intents)
	}
	if covered != 1 {
		t.Fatalf("the copy has none of the intents: %d", covered)
	}
}

// A guess waits in the inferred document, and joins the design when agreed with.
func TestInferredDocument(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nWhat a person said.\n")
	w, err := r.PutWork(ctx, tig.Work{Intent: "x", Result: tig.ResultDone})
	if err != nil {
		t.Fatal(err)
	}
	guess := mark(t, r, tig.MarkInput{Sentence: "A guess.", Under: "Guesses", IDs: []string{w.ID}, Inferred: true, By: tig.ByAI, Via: "analysis"})
	if guess.Path != "inferred.md" {
		t.Fatalf("%+v", guess)
	}
	s := sentenceOf(t, r, "A guess.")
	if !s.Inferred || s.Intents[0].State != tig.IntentProposed || s.Intents[0].Via != "analysis" {
		t.Fatalf("%+v", s)
	}
	a, _ := r.Analyze(ctx, "")
	if len(a.Report.Broken) != 0 {
		t.Fatalf("it is not an orphan: %+v", a.Report.Broken)
	}
	if set := a.Relations(ctx).Set(tig.SetInput{Commits: []string{w.ID[:4]}}); len(set.Sentences) != 0 {
		t.Fatalf("a set does not bring in guesses by itself: %+v", set.Sentences)
	}
	// The words of the main document stay where they are; nobody confirmed them.
	said := mark(t, r, tig.MarkInput{Sentence: "What a person said.", IDs: []string{w.ID}, Inferred: true, By: tig.ByAI, Via: "builder"})
	if said.Path != "d.md" {
		t.Fatalf("%+v", said)
	}
	if s := sentenceOf(t, r, "What a person"); s.Inferred || s.Intents[0].State != tig.IntentProposed || s.Intents[0].By != tig.ByAI {
		t.Fatalf("%+v", s)
	}

	short := s.Intents[0].Short
	if _, err := r.ConfirmIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman, Via: tig.ViaDirect}); err != nil {
		t.Fatal(err)
	}
	s = sentenceOf(t, r, "A guess.")
	if s.Inferred || s.Path != "d.md" || s.Intents[0].State != tig.IntentConfirmed || s.Intents[0].By != tig.ByHuman || s.Intents[0].Short != short || !slices.Contains(s.Heading, "Guesses") {
		t.Fatalf("%+v", s)
	}
	if !slices.Equal(s.IDs, []string{w.ID[:4]}) {
		t.Fatalf("its work comes with it: %+v", s.IDs)
	}

	second := mark(t, r, tig.MarkInput{Sentence: "Wrong guess.", Inferred: true, By: tig.ByAI, Via: "analysis"})
	wrong := sentenceOf(t, r, "Wrong guess.")
	if _, err := r.RejectIntent(ctx, tig.DecideInput{Intent: wrong.Intents[0].Short, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(r.Root, second.Path))
	if strings.Contains(string(data), "Wrong guess.") {
		t.Fatalf("%s", data)
	}
	if _, err := r.Show(ctx, wrong.Intents[0].Short); err != nil {
		t.Fatal("the ledger keeps what was rejected:", err)
	}
}

func TestOldMarksConvert(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n")
	w, err := r.PutWork(ctx, tig.Work{Intent: "x", Result: tig.ResultDone})
	if err != nil {
		t.Fatal(err)
	}
	commits, _ := r.Commits(ctx)
	short := commits.Short(w.ID)
	before := "# A\n\nHas **one part *" + short + "*** and **another *" + short + " dead***.\n"
	write(t, r.Root, "d.md", before)
	hash := tig.ParseBlocks("d.md", []byte(before))[1].Hash
	a, _ := r.Analyze(ctx, "")
	if got := codes(a.Report.Stale); !slices.Equal(got, []string{tig.FindUnsynced}) {
		t.Fatalf("%v", got)
	}
	res, err := r.Sync(ctx, "test")
	if err != nil || res.Migrated != 2 {
		t.Fatalf("%+v %v", res, err)
	}
	data, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	if strings.Contains(string(data), "***") || !strings.Contains(string(data), "Has one part and another. <tig l=") {
		t.Fatalf("%s", data)
	}
	if got := tig.ParseBlocks("d.md", data)[1].Hash; got != hash {
		t.Fatal("the block is the same block")
	}
	s := sentenceOf(t, r, "Has one part")
	if len(s.Intents) != 2 || s.Intents[0].Text != "one part" || s.Intents[0].From != 4 || s.Intents[1].Text != "another" || !slices.Equal(s.IDs, []string{short}) {
		t.Fatalf("%+v", s)
	}
	if s.Intents[0].State != tig.IntentProposed || s.Intents[0].Via != tig.ViaMigrated {
		t.Fatalf("where it came from is not known: %+v", s.Intents[0])
	}
	a, _ = r.Analyze(ctx, "")
	if got := codes(a.Report.Broken); !slices.Equal(got, []string{tig.FindBrokenMark}) {
		t.Fatalf("the commit that was never there is reported: %v", got)
	}
}

// Two machines merge by putting their records together.
func TestLedgersMerge(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nOne line here.\n")
	mark(t, r, tig.MarkInput{Sentence: "One line here."})
	dir := filepath.Join(r.Dir, "ledger")
	other := t.TempDir()
	copyDir(t, dir, other)
	s := sentenceOf(t, r, "One line")
	if _, err := r.ConfirmIntent(ctx, tig.DecideInput{Intent: s.Intents[0].Short, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	// The other machine rejects the same intent from where it was.
	theirs := tig.Open(r.Root)
	theirs.LedgerDir = other
	if _, err := theirs.RetireIntent(ctx, tig.DecideInput{Intent: s.Intents[0].Short, Reason: "not needed", By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	copyDir(t, other, dir)
	l1, err := r.Ledger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	l2, err := r.Ledger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	i1, i2 := l1.Intents()[0], l2.Intents()[0]
	if i1.State != i2.State || len(i1.Log) != 3 {
		t.Fatalf("%+v %+v", i1, i2)
	}
	// A record that is not what its name says is refused.
	bad := filepath.Join(dir, strings.Repeat("0", 64))
	if err := os.WriteFile(bad, []byte("tig-line-event 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Ledger(ctx); err == nil {
		t.Fatal("a forged record")
	}
}

func copyDir(t *testing.T, from, to string) {
	t.Helper()
	entries, err := os.ReadDir(from)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(from, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(to, e.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAcceptance(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	write(t, root, "d.md", "# A\n\nDo it.\n")
	r := tig.Open(root)
	if err := os.MkdirAll(filepath.Join(root, ".tig"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := r.SetConfig(tig.Config{Root: "d.md"}); err != nil {
		t.Fatal(err)
	}
	// A project from before the ledger: its works count as accepted.
	old, err := r.PutWork(ctx, tig.Work{Intent: "old", Result: tig.ResultDone})
	if err != nil {
		t.Fatal(err)
	}
	mark(t, r, tig.MarkInput{Sentence: "Do it.", IDs: []string{old.ID}})
	l, _ := r.Ledger(ctx)
	if got := l.AcceptanceOf(old); got != tig.WorkAccepted {
		t.Fatal(got)
	}
	fresh, err := r.PutWork(ctx, tig.Work{Intent: "new", Result: tig.ResultDone})
	if err != nil {
		t.Fatal(err)
	}
	l, _ = r.Ledger(ctx)
	if got := l.AcceptanceOf(fresh); got != tig.WorkCandidate {
		t.Fatal(got)
	}
	mark(t, r, tig.MarkInput{Sentence: "Do it.", IDs: []string{fresh.ID}})
	if _, err := r.Accept(ctx, tig.AcceptInput{Work: fresh.ID, State: "maybe"}); err == nil {
		t.Fatal("two decisions only")
	}
	if _, err := r.Accept(ctx, tig.AcceptInput{Work: fresh.ID, State: tig.WorkRejected, By: tig.ByHuman, Reason: "wrong"}); err != nil {
		t.Fatal(err)
	}
	s := sentenceOf(t, r, "Do it.")
	commits, _ := r.Commits(ctx)
	if !slices.Equal(s.IDs, []string{commits.Short(old.ID)}) {
		t.Fatalf("a rejected work follows nothing: %v", s.IDs)
	}
	if _, err := r.Accept(ctx, tig.AcceptInput{Work: fresh.ID, State: tig.WorkAccepted, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceOf(t, r, "Do it."); len(s.IDs) != 2 {
		t.Fatalf("the last decision counts: %v", s.IDs)
	}
}

func TestSetFromAnIntentSurvivesRewording(t *testing.T) {
	ctx := context.Background()
	rr := newRelationRepo(t, "# Game\n\nThe account can be deleted.\n\nOrders are kept.\n")
	acct := rr.commit("account", nil, "auth/check.go", "account.go")
	order := rr.commit("order", nil, "auth/check.go", "order.go")
	rr.mark("The account can be deleted.", "", acct)
	rr.mark("Orders are kept.", "", order)
	rel := rr.relations()
	var short string
	for _, s := range rel.Sentences {
		if s.Text == "The account can be deleted." {
			short = s.Intents[0].Short
		}
	}
	data, _ := os.ReadFile(filepath.Join(rr.root, "README.md"))
	rr.write("README.md", strings.Replace(string(data), "can be deleted", "can be removed for good", 1))
	if _, err := rr.r.Sync(ctx, "test"); err != nil {
		t.Fatal(err)
	}
	set := rr.relations().Set(tig.SetInput{Intents: []string{short}})
	if len(set.Sentences) != 1 || set.Sentences[0].Why != tig.WhySeed || !strings.Contains(set.Sentences[0].Text, "removed for good") {
		t.Fatalf("%+v", set.Sentences)
	}
	if len(set.Commits) != 1 || set.Commits[0].Short != acct || !strings.HasPrefix(set.FileWhy["account.go"], "commit:") {
		t.Fatalf("%+v %+v", set.Commits, set.FileWhy)
	}
	// From a file: both sentences, and each says it is there by the file.
	set = rr.relations().Set(tig.SetInput{Files: []string{"auth/check.go"}})
	if len(set.Sentences) != 2 || set.Sentences[0].Why != "file:auth/check.go" {
		t.Fatalf("%+v", set.Sentences)
	}
}
