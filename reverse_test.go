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

func reverseRepo(t *testing.T, doc string, files map[string]string) *tig.Repo {
	t.Helper()
	r := ledgerRepo(t, doc)
	for f, text := range files {
		write(t, r.Root, f, text)
	}
	return r
}

func candidateOf(t *testing.T, res tig.ReverseResult, line int) tig.ReverseCandidate {
	t.Helper()
	for _, c := range res.Candidates {
		if c.Line == line {
			return c
		}
	}
	t.Fatalf("no candidate on line %d: %+v", line, res.Candidates)
	return tig.ReverseCandidate{}
}

func kinds(c tig.ReverseCandidate) []string {
	var out []string
	for _, clue := range c.Clues {
		out = append(out, clue.Kind)
	}
	return out
}

// What a line points at is found with no model, and nothing is written.
func TestReverseFindsWhatLinesPointAt(t *testing.T) {
	ctx := context.Background()
	r := reverseRepo(t, strings.Join([]string{
		"# App",
		"",
		"Saving goes through [the store](internal/store/save.go).", // 3
		"The parser is in `internal/parse/`.",                      // 4
		"Tokens are checked by `CheckToken` before anything else.", // 5
		"See internal/report/report.go for the output.",            // 6
		"Nothing here points anywhere.",                            // 7
		"",
		"```",
		"`internal/store/save.go` in a fence is not a line.",
		"```",
		"",
		"## Billing",
		"",
		"Invoices go out monthly.", // 15
		"They are never late.",     // 16
		"",
	}, "\n"), map[string]string{
		"internal/store/save.go":    "package store\n",
		"internal/parse/a.go":       "package parse\n",
		"internal/parse/b.go":       "package parse\n",
		"internal/auth/token.go":    "package auth\nfunc CheckToken() {}\n",
		"internal/report/report.go": "package report\n",
		"billing/invoice.go":        "package billing\n",
		"loose/unused.go":           "package loose\n",
	})
	before, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	res, err := r.Reverse(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if c := candidateOf(t, res, 3); !slices.Equal(c.Files, []string{"internal/store/save.go"}) || c.Score != 95 || !slices.Equal(kinds(c), []string{tig.ClueLink}) {
		t.Fatalf("link: %+v", c)
	}
	if c := candidateOf(t, res, 4); !slices.Equal(c.Files, []string{"internal/parse/a.go", "internal/parse/b.go"}) || !slices.Equal(kinds(c), []string{tig.CluePath}) {
		t.Fatalf("folder: %+v", c)
	}
	if c := candidateOf(t, res, 5); !slices.Equal(c.Files, []string{"internal/auth/token.go"}) || !slices.Equal(kinds(c), []string{tig.ClueSymbol}) {
		t.Fatalf("symbol: %+v", c)
	}
	if c := candidateOf(t, res, 6); !slices.Equal(c.Files, []string{"internal/report/report.go"}) || !slices.Equal(kinds(c), []string{tig.ClueWord}) {
		t.Fatalf("written out: %+v", c)
	}
	// A section titled like a folder gives one weak hint, on its first line.
	if c := candidateOf(t, res, 15); !slices.Equal(c.Files, []string{"billing/invoice.go"}) || c.Score != 30 || !slices.Equal(kinds(c), []string{tig.ClueHeading}) {
		t.Fatalf("heading: %+v", c)
	}
	for _, c := range res.Candidates {
		if c.Line == 16 || c.Line == 7 || c.Line == 10 {
			t.Fatalf("no clue here: %+v", c)
		}
	}
	if !slices.ContainsFunc(res.OpenLines, func(l tig.OpenLine) bool { return l.Line == 7 }) {
		t.Fatalf("open lines: %+v", res.OpenLines)
	}
	if !slices.Equal(res.OpenFiles, []string{"loose/unused.go"}) {
		t.Fatalf("open files: %+v", res.OpenFiles)
	}
	after, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	if string(before) != string(after) {
		t.Fatal("reverse wrote to the document")
	}
	if ls, _ := r.Ledger(ctx); len(ls.Intents()) != 0 {
		t.Fatal("reverse wrote to the ledger")
	}
}

// A clue covers the sentence it is in, not the whole line.
func TestReverseRangeIsTheSentence(t *testing.T) {
	r := reverseRepo(t, "# A\n\n儲存在 `store/save.go`。匯出在 `report/out.go`。其他沒有說。\n", map[string]string{
		"store/save.go": "x\n",
		"report/out.go": "y\n",
	})
	res, err := r.Reverse(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("%+v", res.Candidates)
	}
	if a, b := res.Candidates[0], res.Candidates[1]; a.Text != "儲存在 `store/save.go`。" || a.From != 0 || b.Text != "匯出在 `report/out.go`。" || b.From != a.To {
		t.Fatalf("%+v %+v", a, b)
	}
}

// A symbol everybody uses says nothing.
func TestReverseDropsCommonSymbols(t *testing.T) {
	files := map[string]string{}
	for _, f := range []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go"} {
		files[f] = "func Helper() {}\n"
	}
	r := reverseRepo(t, "# A\n\nEverything uses `Helper`.\n", files)
	res, err := r.Reverse(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("%+v", res.Candidates)
	}
}

// A description binds nothing until someone agrees. Confirming closes the file;
// rejecting is remembered, so it is not proposed again.
func TestDescribeIsAProposalUntilDecided(t *testing.T) {
	ctx := context.Background()
	r := reverseRepo(t, "# A\n\nSaving is in `a.go`.\nLoading is in `b.go`.\n", map[string]string{"a.go": "a\n", "b.go": "b\n"})
	res, err := r.Reverse(ctx)
	if err != nil {
		t.Fatal(err)
	}
	written, err := r.ProposeReverse(ctx, res.Candidates, tig.ByTool, tig.ViaReverse, "t")
	if err != nil || len(written) != 2 {
		t.Fatalf("%v %+v", err, written)
	}
	if written[0].Work.Result != tig.ResultDescribed {
		t.Fatalf("%+v", written[0].Work)
	}

	saving := sentenceOf(t, r, "Saving")
	it := saving.Intents[0]
	if it.State != tig.IntentProposed || it.By != tig.ByTool || it.Via != tig.ViaReverse {
		t.Fatalf("%+v", it)
	}
	if len(saving.IDs) != 0 || len(it.Commits) != 0 {
		t.Fatalf("a proposal binds nothing: %+v", saving)
	}
	if len(it.Links) != 1 || it.Links[0].State != tig.LinkProposed || it.Links[0].Kind != tig.LinkDescribes || !slices.Equal(it.Links[0].Files, []string{"a.go"}) || !strings.Contains(it.Links[0].Reason, "path: a.go") {
		t.Fatalf("%+v", it.Links)
	}
	a, _ := r.Analyze(ctx, "")
	if !slices.Contains(a.Report.Undesigned, "a.go") || len(a.Report.Unbound) != 0 {
		t.Fatalf("it does not close the chain, and is not a stray commit: %+v", a.Report)
	}
	rel := a.Relations(ctx)
	for _, g := range rel.Groups() {
		if slices.Contains(g.Files, "a.go") {
			t.Fatalf("nothing groups around a proposal: %+v", g)
		}
	}
	if again, _ := r.Reverse(ctx); len(again.Candidates) != 0 {
		t.Fatalf("what was said is not proposed again: %+v", again.Candidates)
	}

	if _, err := r.DecideLink(ctx, tig.LinkDecision{Intent: it.Short, Work: written[0].Work.ID, State: tig.LinkConfirmed, By: tig.ByHuman, Via: tig.ViaDirect}); err != nil {
		t.Fatal(err)
	}
	saving = sentenceOf(t, r, "Saving")
	if len(saving.IDs) != 1 || saving.Intents[0].Links[0].State != tig.LinkConfirmed {
		t.Fatalf("%+v", saving)
	}
	if saving.Intents[0].State != tig.IntentProposed {
		t.Fatalf("whether the code goes with it says nothing about the intent: %+v", saving.Intents[0])
	}
	a, _ = r.Analyze(ctx, "")
	if slices.Contains(a.Report.Undesigned, "a.go") {
		t.Fatalf("%+v", a.Report.Undesigned)
	}

	loading := sentenceOf(t, r, "Loading")
	if _, err := r.DecideLink(ctx, tig.LinkDecision{Intent: loading.Intents[0].Short, Work: written[1].Work.ID, State: tig.LinkRejected, By: tig.ByHuman, Reason: "no"}); err != nil {
		t.Fatal(err)
	}
	loading = sentenceOf(t, r, "Loading")
	if len(loading.Intents[0].Links) != 0 || len(loading.IDs) != 0 {
		t.Fatalf("%+v", loading)
	}
	if again, _ := r.Reverse(ctx); len(again.Candidates) != 0 {
		t.Fatalf("what was turned down is not proposed again: %+v", again.Candidates)
	}
	// Someone else guessing the same thing does not undo that.
	if _, err := r.Mark(ctx, tig.MarkInput{Sentence: "Loading is in `b.go`.", IDs: []string{written[1].Work.ID}, LinkKind: tig.LinkDescribes, By: tig.ByAI, Via: "x"}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceOf(t, r, "Loading"); len(s.Intents[0].Links) != 0 {
		t.Fatalf("%+v", s.Intents[0].Links)
	}
	// A person can.
	if _, err := r.Mark(ctx, tig.MarkInput{Sentence: "Loading is in `b.go`.", IDs: []string{written[1].Work.ID}, LinkKind: tig.LinkDescribes, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceOf(t, r, "Loading"); len(s.Intents[0].Links) != 1 || s.Intents[0].Links[0].State != tig.LinkConfirmed {
		t.Fatalf("%+v", s.Intents[0].Links)
	}
}

// Lines saying the same thing are told apart by number.
func TestDescribeTellsApartLines(t *testing.T) {
	ctx := context.Background()
	r := reverseRepo(t, "# A\n\nSame words.\n\n## B\n\nSame words.\n", map[string]string{"a.go": "a\n"})
	d, err := r.Describe(ctx, tig.DescribeInput{Sentence: "Same words.", Line: 7, Files: []string{"a.go"}, By: tig.ByAI, Via: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Mark.Line != 7 {
		t.Fatalf("%+v", d.Mark)
	}
	data, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	lines := strings.Split(string(data), "\n")
	if strings.Contains(lines[2], "<tig") || !strings.Contains(lines[6], "<tig") {
		t.Fatalf("%s", data)
	}
	if _, err := r.Describe(ctx, tig.DescribeInput{Sentence: "x", Files: []string{"missing.go"}}); err == nil {
		t.Fatal("a file that is not there")
	}
	if _, err := r.Describe(ctx, tig.DescribeInput{Sentence: "x"}); err == nil {
		t.Fatal("no file")
	}
}

// A person saying the code goes with the line is not a proposal.
func TestDescribeByAPerson(t *testing.T) {
	ctx := context.Background()
	r := reverseRepo(t, "# A\n\nSaving.\n", map[string]string{"a.go": "a\n"})
	if _, err := r.Describe(ctx, tig.DescribeInput{Sentence: "Saving.", Files: []string{"a.go"}, By: tig.ByHuman, Via: tig.ViaDirect}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceOf(t, r, "Saving"); len(s.IDs) != 1 || s.Intents[0].Links[0].State != tig.LinkConfirmed {
		t.Fatalf("%+v", s)
	}
}

// A sentence an AI names is not taken as anyone's confirmation; accepting the work
// carries what it said about the code, and only that.
func TestLinksAreNotConfirmations(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nWhat the document says.\n")
	w, err := r.PutWork(ctx, tig.Work{Intent: "x", Result: tig.ResultDone})
	if err != nil {
		t.Fatal(err)
	}
	mark(t, r, tig.MarkInput{Sentence: "What the document says.", IDs: []string{w.ID}, Inferred: true, By: tig.ByAI, Via: "builder"})
	s := sentenceOf(t, r, "What the document")
	it := s.Intents[0]
	if s.Inferred || it.State != tig.IntentProposed || it.By != tig.ByAI || it.Via != "builder" {
		t.Fatalf("nobody confirmed it: %+v", it)
	}
	if len(it.Links) != 1 || it.Links[0].State != tig.LinkProposed || it.Links[0].Acceptance != tig.WorkCandidate {
		t.Fatalf("%+v", it.Links)
	}
	// A work that really changed files stands on the line while it waits.
	if len(s.IDs) != 1 {
		t.Fatalf("%+v", s.IDs)
	}
	if _, err := r.Accept(ctx, tig.AcceptInput{Work: w.ID, State: tig.WorkAccepted, By: tig.ByHuman, Via: tig.ViaDirect}); err != nil {
		t.Fatal(err)
	}
	it = sentenceOf(t, r, "What the document").Intents[0]
	if it.Links[0].State != tig.LinkConfirmed || it.State != tig.IntentProposed {
		t.Fatalf("%+v", it)
	}
	v, err := r.Show(ctx, it.Short)
	if err != nil || v.Intent.Works[0].State != tig.LinkConfirmed {
		t.Fatalf("%v %+v", err, v.Intent)
	}
	// Rejecting the link stays, whatever the work's acceptance says.
	if _, err := r.DecideLink(ctx, tig.LinkDecision{Intent: it.Short, Work: w.ID, State: tig.LinkRejected, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceOf(t, r, "What the document"); len(s.IDs) != 0 || len(s.Intents[0].Links) != 0 {
		t.Fatalf("%+v", s)
	}
}

// A record written before links had a state stands as confirmed, and a reader
// ignores keys it does not know.
func TestOldLinkRecordsStand(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nA line.\n")
	w, err := r.PutWork(ctx, tig.Work{Intent: "x", Result: tig.ResultDone})
	if err != nil {
		t.Fatal(err)
	}
	mark(t, r, tig.MarkInput{Sentence: "A line."})
	it := sentenceOf(t, r, "A line").Intents[0]
	l, err := r.Ledger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	i, err := l.Intent(it.Short)
	if err != nil {
		t.Fatal(err)
	}
	rec := "tig-link-event 1\nwork " + w.ID + "\nintent " + i.ID + "\nrevision 1\nkind implements\nfuture something\nat 2030-01-01T00:00:00Z\n"
	sum := tig.HashOf([]byte(rec))
	if err := os.WriteFile(filepath.Join(r.Dir, "ledger", sum), []byte(rec), 0o644); err != nil {
		t.Fatal(err)
	}
	got := sentenceOf(t, r, "A line").Intents[0]
	if len(got.Links) != 1 || got.Links[0].State != tig.LinkConfirmed || len(got.Commits) != 1 {
		t.Fatalf("%+v", got)
	}
}

// init -force says it all at once, and can run again.
func TestInitForce(t *testing.T) {
	ctx := context.Background()
	r := tig.Open(t.TempDir())
	write(t, r.Root, "d.md", "# A\n\nOne.\n\n```\nnot this\n```\n\nSee [more](m.md).\n\nOne.\n")
	write(t, r.Root, "m.md", "# M\n\nMore.\n")
	write(t, r.Root, "a.go", "a\n")
	res, err := r.InitForce(ctx, tig.InitForceInput{Config: tig.Config{Root: "d.md"}, By: tig.ByHuman, Actor: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Created || res.Baseline.Result != tig.ResultBaseline || res.Lines != 4 || len(res.Docs) != 2 {
		t.Fatalf("%+v", res)
	}
	data, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	if strings.Contains(string(data), "not this <tig") || strings.Count(string(data), "<tig") != 3 || !strings.Contains(string(data), `n="2"`) {
		t.Fatalf("%s", data)
	}
	s := sentenceOf(t, r, "More.")
	it := s.Intents[0]
	if it.State != tig.IntentProposed || it.Via != tig.ViaInit || it.By != tig.ByHuman || len(it.Commits) != 1 || it.Links[0].State != tig.LinkConfirmed {
		t.Fatalf("%+v", it)
	}
	a, _ := r.Analyze(ctx, "")
	if len(a.Report.Broken) != 0 || len(a.Report.Unbound) != 0 {
		t.Fatalf("%+v", a.Report)
	}
	again, err := r.InitForce(ctx, tig.InitForceInput{Config: tig.Config{Root: "d.md"}, By: tig.ByHuman, Actor: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if again.Created || again.Lines != 0 || again.Baseline.ID != res.Baseline.ID {
		t.Fatalf("%+v", again)
	}
	after, _ := os.ReadFile(filepath.Join(r.Root, "d.md"))
	if string(after) != string(data) {
		t.Fatalf("%s", after)
	}
	// A line added later joins on the next run; what is there is left alone.
	write(t, r.Root, "d.md", string(after)+"\nNew line.\n")
	third, err := r.InitForce(ctx, tig.InitForceInput{Config: tig.Config{Root: "d.md"}, By: tig.ByAI, Actor: "t"})
	if err != nil || third.Lines != 1 {
		t.Fatalf("%v %+v", err, third)
	}
	if it := sentenceOf(t, r, "New line").Intents[0]; it.Links[0].State != tig.LinkProposed || it.By != tig.ByAI {
		t.Fatalf("only a person's word is taken as agreed: %+v", it)
	}
}

// After saying it all at once, the rules can still say which lines are about what:
// the baseline reaches nothing in particular.
func TestReverseAfterInitForce(t *testing.T) {
	ctx := context.Background()
	r := tig.Open(t.TempDir())
	write(t, r.Root, "d.md", "# A\n\nSaving is in `a.go`.\n\nNothing here.\n")
	write(t, r.Root, "a.go", "a\n")
	write(t, r.Root, "b.go", "b\n")
	if _, err := r.InitForce(ctx, tig.InitForceInput{Config: tig.Config{Root: "d.md"}, By: tig.ByHuman, Actor: "t"}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceOf(t, r, "Saving"); s.Intents[0].Links[0].Files != nil {
		t.Fatalf("the baseline is not listed file by file: %+v", s.Intents[0].Links[0])
	}
	res, err := r.Reverse(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].Files[0] != "a.go" {
		t.Fatalf("%+v", res.Candidates)
	}
	if !slices.Equal(res.OpenFiles, []string{"b.go"}) || len(res.OpenLines) != 1 || res.OpenLines[0].Text != "Nothing here." {
		t.Fatalf("%+v", res)
	}
	// What is proposed is not open any more, and running again finds nothing new.
	if _, err := r.ProposeReverse(ctx, res.Candidates, tig.ByTool, tig.ViaReverse, "t"); err != nil {
		t.Fatal(err)
	}
	again, err := r.Reverse(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Candidates) != 0 || !slices.Equal(again.OpenFiles, []string{"b.go"}) || len(again.OpenLines) != 1 {
		t.Fatalf("%+v", again)
	}
}
