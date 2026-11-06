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

func setOf(t *testing.T, r *tig.Repo, in tig.SetInput) tig.Set {
	t.Helper()
	ctx := context.Background()
	a, err := r.Analyze(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	return a.Relations(ctx).Set(in)
}

// reword changes words of a document and leaves its marks.
func reword(t *testing.T, r *tig.Repo, file, from, to string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(r.Root, file))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), from) {
		t.Fatalf("no %q in %s", from, data)
	}
	write(t, r.Root, file, strings.Replace(string(data), from, to, 1))
}

func intentOf(t *testing.T, r *tig.Repo, ref string) *tig.Intent {
	t.Helper()
	l, err := r.Ledger(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	i, err := l.Intent(ref)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func eventKinds(i *tig.Intent) []string {
	var out []string
	for _, e := range i.Log {
		out = append(out, e.Kind)
	}
	return out
}

// Asked for at a revision it has left, a set says what that revision said. It used
// to hand back the latest words as if they were what was asked for.
func TestSetPinsARevision(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nCarts are kept.\n")
	mark(t, r, tig.MarkInput{Sentence: "Carts are kept.", State: tig.IntentConfirmed, By: tig.ByHuman})
	short := sentenceOf(t, r, "Carts").Intents[0].Short

	reword(t, r, "d.md", "Carts are kept.", "Carts are kept for a day.")
	if _, err := r.Sync(ctx, "t"); err != nil {
		t.Fatal(err)
	}

	old := setOf(t, r, tig.SetInput{Intents: []string{short + "#1"}})
	if len(old.Pinned) != 1 {
		t.Fatalf("%+v", old)
	}
	if p := old.Pinned[0]; p.Revision != 1 || p.Current != 2 || !p.Stale || p.Text != "Carts are kept." || p.Short != short {
		t.Fatalf("%+v", p)
	}
	if len(old.Sentences) != 1 || !strings.Contains(old.Sentences[0].Text, "for a day") {
		t.Fatalf("a set holds the sentence as it is now: %+v", old.Sentences)
	}
	if p := setOf(t, r, tig.SetInput{Intents: []string{short + "#2"}}).Pinned[0]; p.Stale {
		t.Fatalf("the latest revision is not stale: %+v", p)
	}
	if plain := setOf(t, r, tig.SetInput{Intents: []string{short}}); len(plain.Pinned) != 0 {
		t.Fatalf("nothing was pinned: %+v", plain.Pinned)
	}
	never := setOf(t, r, tig.SetInput{Intents: []string{short + "#9"}})
	if !slices.Contains(never.Missing, short+"#9") || len(never.Sentences) != 0 {
		t.Fatalf("a revision it never had is not there: %+v", never)
	}

	if _, err := r.RetireIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	gone := setOf(t, r, tig.SetInput{Intents: []string{short + "#2"}})
	if len(gone.Pinned) != 1 || !gone.Pinned[0].Stale || gone.Pinned[0].State != tig.IntentDeprecated {
		t.Fatalf("what no longer stands has moved on: %+v", gone)
	}
}

// The same members are the same set, whichever way someone got there.
func TestSetDigest(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nCarts are kept.\n")
	w := oneCommit(t, r, r.Root, "cart.go", "package cart\n", "the cart")
	mark(t, r, tig.MarkInput{Sentence: "Carts are kept.", IDs: []string{w.ID}, By: tig.ByHuman})
	short := sentenceOf(t, r, "Carts").Intents[0].Short

	a := setOf(t, r, tig.SetInput{Intents: []string{short}})
	b := setOf(t, r, tig.SetInput{Sentences: []string{"Carts are kept."}})
	if a.Digest == "" || a.Digest != b.Digest {
		t.Fatalf("%q %q\n%+v\n%+v", a.Digest, b.Digest, a, b)
	}
	if again := setOf(t, r, tig.SetInput{Intents: []string{short}}); again.Digest != a.Digest {
		t.Fatal("not stable")
	}
	if pinned := setOf(t, r, tig.SetInput{Intents: []string{short + "#1"}}); pinned.Digest == a.Digest {
		t.Fatal("asking for a revision is a different thing to have been given")
	}
	if empty := setOf(t, r, tig.SetInput{Files: nil}); empty.Digest != "" {
		t.Fatalf("an empty set names nothing: %q", empty.Digest)
	}

	reword(t, r, "d.md", "Carts are kept.", "Carts are kept for a day.")
	if _, err := r.Sync(ctx, "t"); err != nil {
		t.Fatal(err)
	}
	if after := setOf(t, r, tig.SetInput{Intents: []string{short}}); after.Digest == a.Digest {
		t.Fatal("the intent moved to a new revision and the set is said to be the same")
	}
}

// A person who edits a line and says why has revised it. An edit nobody explained
// has only touched it, and whoever confirmed it should look again.
func TestRevisedIsNotTouched(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nCarts are kept.\n")
	mark(t, r, tig.MarkInput{Sentence: "Carts are kept.", State: tig.IntentConfirmed, By: tig.ByHuman})
	short := sentenceOf(t, r, "Carts").Intents[0].Short

	reword(t, r, "d.md", "are kept.", "are kept long.")
	if _, err := r.Sync(ctx, "t"); err != nil {
		t.Fatal(err)
	}
	i := intentOf(t, r, short)
	if !slices.Contains(eventKinds(i), tig.IntentTouched) || !i.Review {
		t.Fatalf("%v %+v", eventKinds(i), i)
	}
	if _, err := r.ConfirmIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}

	reword(t, r, "d.md", "are kept long.", "are kept a day.")
	res, err := r.SyncWith(ctx, tig.SyncInput{Actor: "pat", By: tig.ByHuman, Via: "review", Reason: "legal asked for a limit", Source: "ticket 12"})
	if err != nil {
		t.Fatal(err)
	}
	i = intentOf(t, r, short)
	last := i.Log[len(i.Log)-1]
	if last.Kind != tig.IntentRevised || last.Reason != "legal asked for a limit" || last.Source != "ticket 12" || last.By != tig.ByHuman || last.Via != "review" || last.Actor != "pat" {
		t.Fatalf("%+v", last)
	}
	if i.Review {
		t.Fatal("a person said so and said why: nobody needs to look again")
	}
	if len(res.Touched) != 1 || res.Touched[0] != short {
		t.Fatalf("%+v", res)
	}
	v, err := r.Show(ctx, short)
	if err != nil {
		t.Fatal(err)
	}
	if v.Intent.Previous == nil || v.Intent.Previous.Reason != "legal asked for a limit" {
		t.Fatalf("%+v", v.Intent)
	}

	reword(t, r, "d.md", "are kept a day.", "are kept a week.")
	if _, err := r.SyncWith(ctx, tig.SyncInput{Actor: "bot", By: tig.ByAI, Via: "agent", Reason: "seemed better"}); err != nil {
		t.Fatal(err)
	}
	if i = intentOf(t, r, short); !i.Review || i.Log[len(i.Log)-1].Kind != tig.IntentRevised {
		t.Fatalf("an agent's revision still wants a person: %v %+v", eventKinds(i), i)
	}
}

// Where a decision came from stays with it.
func TestSourceIsKept(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nCarts are kept.\n\nPrices are shown.\n")
	mark(t, r, tig.MarkInput{Sentence: "Carts are kept.", By: tig.ByAI, Source: "chat of 3 May"})
	short := sentenceOf(t, r, "Carts").Intents[0].Short
	if got := intentOf(t, r, short).Log[0].Source; got != "chat of 3 May" {
		t.Fatalf("%q", got)
	}
	if _, err := r.ConfirmIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman, Source: "review 7"}); err != nil {
		t.Fatal(err)
	}
	i := intentOf(t, r, short)
	if got := i.Log[len(i.Log)-1].Source; got != "review 7" {
		t.Fatalf("%q", got)
	}
	write(t, r.Root, "price.go", "package price\n")
	d, err := r.Describe(ctx, tig.DescribeInput{Sentence: "Prices are shown.", Files: []string{"price.go"}, By: tig.ByAI, Source: "the code"})
	if err != nil {
		t.Fatal(err)
	}
	if got := intentOf(t, r, d.Mark.Intent).Log[0].Source; got != "the code" {
		t.Fatalf("%q %+v", got, d.Mark)
	}
}

// Two machines that decide differently: the ledger still gives one answer, but it
// says nobody chose it.
func TestDisputedUntilSomeoneDecides(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nCarts are kept.\n")
	mark(t, r, tig.MarkInput{Sentence: "Carts are kept.", By: tig.ByAI})
	short := sentenceOf(t, r, "Carts").Intents[0].Short
	dir := filepath.Join(r.Root, ".tig", "ledger")

	common := t.TempDir()
	copyDir(t, dir, common)
	if _, err := r.ConfirmIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman, Actor: "a"}); err != nil {
		t.Fatal(err)
	}
	machineA := t.TempDir()
	copyDir(t, dir, machineA)
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	copyDir(t, common, dir)
	if _, err := r.RejectIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman, Actor: "b"}); err != nil {
		t.Fatal(err)
	}
	copyDir(t, machineA, dir)

	i := intentOf(t, r, short)
	if len(i.Forks) != 1 || len(i.Forks[0].Events) != 2 {
		t.Fatalf("%+v", i)
	}
	if got := i.Forks[0].Events[1].Kind; got != i.State {
		t.Fatalf("the last of the fork is the one standing: %s and %s", got, i.State)
	}
	a, err := r.Analyze(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if a.Report.Closed || !slices.Contains(codes(a.Report.Stale), tig.FindDisputed) {
		t.Fatalf("%+v", a.Report)
	}
	v, err := r.Show(ctx, short)
	if err != nil {
		t.Fatal(err)
	}
	if v.Intent.Sync != tig.SyncDisputed || len(v.Intent.Forks) != 1 {
		t.Fatalf("%+v", v.Intent)
	}

	if _, err := r.ConfirmIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman, Actor: "c", Reason: "we talked"}); err != nil {
		t.Fatal(err)
	}
	if i = intentOf(t, r, short); len(i.Forks) != 0 || i.State != tig.IntentConfirmed {
		t.Fatalf("someone decided: %+v", i)
	}
}

// Two machines agreeing is not a dispute.
func TestAgreeingIsNotDisputed(t *testing.T) {
	ctx := context.Background()
	r := ledgerRepo(t, "# A\n\nCarts are kept.\n")
	mark(t, r, tig.MarkInput{Sentence: "Carts are kept.", By: tig.ByAI})
	short := sentenceOf(t, r, "Carts").Intents[0].Short
	dir := filepath.Join(r.Root, ".tig", "ledger")
	common := t.TempDir()
	copyDir(t, dir, common)
	if _, err := r.ConfirmIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman, Actor: "a"}); err != nil {
		t.Fatal(err)
	}
	machineA := t.TempDir()
	copyDir(t, dir, machineA)
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0o755)
	copyDir(t, common, dir)
	if _, err := r.ConfirmIntent(ctx, tig.DecideInput{Intent: short, By: tig.ByHuman, Actor: "b"}); err != nil {
		t.Fatal(err)
	}
	copyDir(t, machineA, dir)
	if i := intentOf(t, r, short); len(i.Forks) != 0 || len(i.Log) != 3 {
		t.Fatalf("%+v", i)
	}
}

func relateRepo(t *testing.T) (r *tig.Repo, carts, prices, receipts string) {
	t.Helper()
	r = ledgerRepo(t, "# A\n\nCarts are kept.\n\nPrices come from the table.\n\nReceipts print.\n")
	shared := oneCommit(t, r, r.Root, "shared.go", "package shared\n", "the shared part")
	own := oneCommit(t, r, r.Root, "price.go", "package price\n", "the prices")
	mark(t, r, tig.MarkInput{Sentence: "Carts are kept.", IDs: []string{shared.ID}, By: tig.ByHuman})
	mark(t, r, tig.MarkInput{Sentence: "Prices come from the table.", IDs: []string{own.ID}, By: tig.ByHuman})
	mark(t, r, tig.MarkInput{Sentence: "Receipts print.", IDs: []string{shared.ID}, By: tig.ByHuman})
	return r, sentenceOf(t, r, "Carts").Intents[0].Short, sentenceOf(t, r, "Prices").Intents[0].Short, sentenceOf(t, r, "Receipts").Intents[0].Short
}

func sentenceIn(set tig.Set, words string) *tig.Sentence {
	for i := range set.Sentences {
		if strings.Contains(set.Sentences[i].Text, words) {
			return &set.Sentences[i]
		}
	}
	return nil
}

// A shared commit brings another intent into a set. Saying they stand apart keeps
// it in sight, and says so: tig does not decide what goes in a piece of work.
func TestParallelIsNotedAndNothingIsDropped(t *testing.T) {
	ctx := context.Background()
	r, carts, _, receipts := relateRepo(t)

	before := setOf(t, r, tig.SetInput{Intents: []string{carts}})
	if s := sentenceIn(before, "Receipts"); s == nil || len(s.Parallel) != 0 {
		t.Fatalf("the shared commit brings it: %+v", before)
	}

	// Proposed by an agent: nothing changes in a set.
	if _, err := r.Relate(ctx, tig.RelateInput{From: carts, To: receipts, Kind: tig.RelationParallel, By: tig.ByAI, Via: tig.ViaAgent}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceIn(setOf(t, r, tig.SetInput{Intents: []string{carts}}), "Receipts"); s == nil || len(s.Parallel) != 0 {
		t.Fatalf("nobody agreed yet: %+v", s)
	}

	e, err := r.Relate(ctx, tig.RelateInput{From: carts, To: receipts, Kind: tig.RelationParallel, By: tig.ByHuman, Reason: "they share a helper, nothing more"})
	if err != nil {
		t.Fatal(err)
	}
	if e.State != tig.LinkConfirmed {
		t.Fatalf("a person's word: %+v", e)
	}
	after := setOf(t, r, tig.SetInput{Intents: []string{carts}})
	s := sentenceIn(after, "Receipts")
	if s == nil || !slices.Equal(s.Parallel, []string{carts}) {
		t.Fatalf("%+v", after)
	}
	if after.Digest != before.Digest {
		t.Fatal("nothing went in or out, so it is the same set")
	}
	if s := sentenceIn(after, "Carts"); s == nil || len(s.Parallel) != 0 {
		t.Fatalf("what was asked for is not apart from itself: %+v", s)
	}
	// It reads the same from the other side.
	if s := sentenceIn(setOf(t, r, tig.SetInput{Intents: []string{receipts}}), "Carts"); s == nil || !slices.Equal(s.Parallel, []string{receipts}) {
		t.Fatalf("%+v", s)
	}
	// Both asked for: they were asked for together.
	both := setOf(t, r, tig.SetInput{Intents: []string{carts, receipts}})
	if s := sentenceIn(both, "Receipts"); s == nil || len(s.Parallel) != 0 {
		t.Fatalf("%+v", s)
	}

	// Rejected later, by the same word: the last thing said stands, from either side.
	if _, err := r.Relate(ctx, tig.RelateInput{From: receipts, To: carts, Kind: tig.RelationParallel, State: tig.LinkRejected, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceIn(setOf(t, r, tig.SetInput{Intents: []string{carts}}), "Receipts"); s == nil || len(s.Parallel) != 0 {
		t.Fatalf("%+v", s)
	}
	l, err := r.Ledger(ctx)
	if err != nil {
		t.Fatal(err)
	}
	id := intentOf(t, r, carts).ID
	if len(l.RelationsOf(id)) != 0 || len(l.RejectedRelationsOf(id)) != 1 {
		t.Fatalf("%+v %+v", l.RelationsOf(id), l.RejectedRelationsOf(id))
	}
}

// What an intent depends on comes with it, one step, and only one way.
func TestDependsOnBringsOneStep(t *testing.T) {
	ctx := context.Background()
	r, carts, prices, _ := relateRepo(t)

	if s := sentenceIn(setOf(t, r, tig.SetInput{Intents: []string{carts}}), "Prices"); s != nil {
		t.Fatalf("nothing relates them yet: %+v", s)
	}
	if _, err := r.Relate(ctx, tig.RelateInput{From: carts, To: prices, Kind: tig.RelationDependsOn, By: tig.ByAI}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceIn(setOf(t, r, tig.SetInput{Intents: []string{carts}}), "Prices"); s != nil {
		t.Fatalf("a proposal brings nothing: %+v", s)
	}
	if _, err := r.Relate(ctx, tig.RelateInput{From: carts, To: prices, Kind: tig.RelationDependsOn, By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	set := setOf(t, r, tig.SetInput{Intents: []string{carts}})
	s := sentenceIn(set, "Prices")
	if s == nil || s.Why != "relation:"+carts {
		t.Fatalf("%+v", set)
	}
	if !slices.Contains(set.Files, "price.go") || set.FileWhy["price.go"] != "relation:"+carts {
		t.Fatalf("%+v", set)
	}
	var commit bool
	for _, c := range set.Commits {
		commit = commit || c.Why == "relation:"+carts
	}
	if !commit {
		t.Fatalf("%+v", set.Commits)
	}
	// The other way: prices do not depend on carts.
	if s := sentenceIn(setOf(t, r, tig.SetInput{Intents: []string{prices}}), "Carts"); s != nil {
		t.Fatalf("%+v", s)
	}
	v, err := r.Show(ctx, prices)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Intent.Relations) != 1 || !v.Intent.Relations[0].Incoming || v.Intent.Relations[0].With != carts {
		t.Fatalf("%+v", v.Intent.Relations)
	}
	if st := sentenceOf(t, r, "Carts").Intents[0]; len(st.Relations) != 1 || st.Relations[0].Incoming || st.Relations[0].With != prices {
		t.Fatalf("%+v", st.Relations)
	}
}

func TestRelateRefuses(t *testing.T) {
	ctx := context.Background()
	r, carts, prices, _ := relateRepo(t)
	for name, in := range map[string]tig.RelateInput{
		"itself":      {From: carts, To: carts, Kind: tig.RelationParallel},
		"unknown":     {From: carts, To: "i0000", Kind: tig.RelationParallel},
		"no kind":     {From: carts, To: prices},
		"two words":   {From: carts, To: prices, Kind: "depends on"},
		"bad state":   {From: carts, To: prices, Kind: tig.RelationParallel, State: "maybe"},
		"bad state 2": {From: carts, To: prices, Kind: tig.RelationParallel, State: tig.WorkAccepted},
	} {
		if _, err := r.Relate(ctx, in); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	// A word of someone's own is kept, and means nothing to tig.
	if _, err := r.Relate(ctx, tig.RelateInput{From: carts, To: prices, Kind: "reviewed_with", By: tig.ByHuman}); err != nil {
		t.Fatal(err)
	}
	if s := sentenceIn(setOf(t, r, tig.SetInput{Intents: []string{carts}}), "Prices"); s != nil {
		t.Fatalf("%+v", s)
	}
	if st := sentenceOf(t, r, "Carts").Intents[0]; len(st.Relations) != 1 || st.Relations[0].Kind != "reviewed_with" {
		t.Fatalf("%+v", st.Relations)
	}
}

// A picture someone put together by hand still answers.
func TestSetWithoutALedger(t *testing.T) {
	set := (&tig.Relations{}).Set(tig.SetInput{Intents: []string{"i1234#2"}})
	if !slices.Contains(set.Missing, "i1234#2") {
		t.Fatalf("%+v", set)
	}
}
