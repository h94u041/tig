package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/h94u041/tig"
)

func (c *cli) mark(args []string) error {
	fs := c.flags("mark")
	ids := fs.String("id", "", "the commits, separated by commas")
	doc := fs.String("doc", "", "the document (default: the main document)")
	under := fs.String("under", "", "the section a new sentence goes under")
	inferred := fs.Bool("inferred", false, "it is a guess: for the inferred document unless the document says it")
	propose := fs.Bool("propose", false, "the commits only may follow it: nobody has agreed")
	confirm := fs.Bool("confirm", false, "this is what was meant")
	why := fs.String("why", "", "why the commits follow it")
	source := fs.String("source", "", "what it was decided from: a conversation, a ticket, a document")
	var by, via string
	c.who(fs, &by, &via)
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: tigwx mark [-id a1b2,c3d4] [-doc f.md] [-under title] [-inferred] [-propose] [-confirm] [-why why] [-source s] <sentence>")
	}
	var refs []string
	if *ids != "" {
		refs = strings.Split(*ids, ",")
	}
	in := tig.MarkInput{Path: *doc, Under: *under, IDs: refs, Inferred: *inferred, By: by, Via: via, Actor: c.actor, Reason: *why, Source: *source}
	if *confirm {
		in.State = tig.IntentConfirmed
	}
	if *propose || *inferred {
		in.Link = tig.LinkProposed
	}
	if err := c.open(); err != nil {
		return err
	}
	in.Sentence = strings.Join(fs.Args(), " ")
	m, err := c.r.Mark(c.ctx, in)
	if err != nil {
		return err
	}
	return c.emit(m, func() {
		fmt.Fprintf(c.out, "%s  %s:%d  %s  %s\n", m.Intent, m.Path, m.Line, strings.Join(m.IDs, " "), m.Text)
	})
}

// reverse finds which lines seem to be about which files, and proposes them when told to.
func (c *cli) reverse(args []string) error {
	fs := c.flags("reverse")
	least := fs.Int("min", 0, "leave out candidates scoring under this, out of 100")
	write := fs.Bool("write", false, "propose them: someone still has to agree")
	by := fs.String("by", tig.ByTool, "who says so")
	via := fs.String("via", tig.ViaReverse, "how it came about")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if err := c.open(); err != nil {
		return err
	}
	res, err := c.r.Reverse(c.ctx)
	if err != nil {
		return err
	}
	kept := res.Candidates[:0:0]
	for _, cand := range res.Candidates {
		if cand.Score >= *least {
			kept = append(kept, cand)
		}
	}
	res.Candidates = kept
	var written []tig.Described
	if *write {
		if written, err = c.r.ProposeReverse(c.ctx, res.Candidates, *by, *via, c.actor); err != nil {
			return err
		}
	}
	if c.asJSON {
		return printJSON(c.out, map[string]any{"candidates": res.Candidates, "openLines": res.OpenLines, "openFiles": res.OpenFiles, "written": written})
	}
	for _, cand := range res.Candidates {
		fmt.Fprintf(c.out, "%3d  %s:%d:%d-%d  %s\n", cand.Score, cand.Path, cand.Line, cand.From, cand.To, cand.Text)
		fmt.Fprintf(c.out, "     files  %s\n     why    %s\n", strings.Join(cand.Files, ", "), tig.ClueReason(cand.Clues))
	}
	fmt.Fprintf(c.out, "%d candidates; %d lines and %d files with nothing found\n", len(res.Candidates), len(res.OpenLines), len(res.OpenFiles))
	if *write {
		fmt.Fprintf(c.out, "%d proposed; decide on each with tigwx link\n", len(written))
	}
	return nil
}

// describe says a line is about files already there.
func (c *cli) describe(args []string) error {
	fs := c.flags("describe")
	var in tig.DescribeInput
	fs.Var((*many)(&in.Files), "f", "a file")
	fs.StringVar(&in.Path, "doc", "", "the document (default: the main document)")
	fs.StringVar(&in.Under, "under", "", "the section a new sentence goes under")
	fs.IntVar(&in.Line, "line", 0, "the 1-based line, to tell apart lines that say the same")
	fs.StringVar(&in.Intent, "m", "", "the commit's one line (default: the sentence)")
	fs.StringVar(&in.Reason, "why", "", "why the line goes with the files")
	fs.StringVar(&in.Source, "source", "", "what it was decided from: a conversation, a ticket, a document")
	propose := fs.Bool("propose", false, "only propose it: nobody has agreed")
	c.who(fs, &in.By, &in.Via)
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 || len(in.Files) == 0 {
		return errors.New("usage: tigwx describe -f file... [-doc f.md] [-under title] [-line n] [-m intent] [-why why] [-propose] [-source s] <sentence>")
	}
	if *propose {
		in.Link = tig.LinkProposed
	}
	in.Sentence, in.Actor = strings.Join(fs.Args(), " "), c.actor
	if err := c.open(); err != nil {
		return err
	}
	d, err := c.r.Describe(c.ctx, in)
	if err != nil {
		return err
	}
	return c.emit(d, func() {
		fmt.Fprintf(c.out, "%s  %s:%d  %s  %s\n", d.Mark.Intent, d.Mark.Path, d.Mark.Line, short(d.Work.ID), d.Mark.Text)
	})
}

// link records whether this code goes with this intent.
func (c *cli) link(args []string) error {
	fs := c.flags("link")
	reject := fs.Bool("reject", false, "it does not")
	var in tig.LinkDecision
	fs.StringVar(&in.Reason, "m", "", "why")
	c.who(fs, &in.By, &in.Via)
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("usage: tigwx link [-reject] [-m why] <intent> <id>")
	}
	if err := c.open(); err != nil {
		return err
	}
	in.Intent, in.Work, in.Actor, in.State = fs.Arg(0), fs.Arg(1), c.actor, tig.LinkConfirmed
	if *reject {
		in.State = tig.LinkRejected
	}
	e, err := c.r.DecideLink(c.ctx, in)
	if err != nil {
		return err
	}
	return c.emit(e, func() { fmt.Fprintf(c.out, "%s %s %s\n", fs.Arg(0), short(e.Work), e.State) })
}

// relate says what two intents have to do with each other.
func (c *cli) relate(args []string) error {
	fs := c.flags("relate")
	var in tig.RelateInput
	fs.StringVar(&in.Kind, "kind", tig.RelationDependsOn, "depends_on, part_of, parallel, or your own word")
	fs.StringVar(&in.Reason, "m", "", "why")
	propose := fs.Bool("propose", false, "only propose it: nobody has agreed")
	reject := fs.Bool("reject", false, "it does not hold")
	c.who(fs, &in.By, &in.Via)
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 2 || (*propose && *reject) {
		return errors.New("usage: tigwx relate [-kind depends_on] [-propose] [-reject] [-m why] <intent> <intent>")
	}
	switch {
	case *propose:
		in.State = tig.LinkProposed
	case *reject:
		in.State = tig.LinkRejected
	}
	if err := c.open(); err != nil {
		return err
	}
	in.From, in.To, in.Actor = fs.Arg(0), fs.Arg(1), c.actor
	e, err := c.r.Relate(c.ctx, in)
	if err != nil {
		return err
	}
	return c.emit(e, func() { fmt.Fprintf(c.out, "%s %s %s: %s\n", fs.Arg(0), e.Kind, fs.Arg(1), e.State) })
}

func (c *cli) fold(args []string) error {
	fs := c.flags("fold")
	intent := fs.String("m", "", "what the commits are, together")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if *intent == "" || fs.NArg() < 2 {
		return errors.New("usage: tigwx fold -m intent <id> <id>...")
	}
	if err := c.open(); err != nil {
		return err
	}
	w, err := c.r.Fold(c.ctx, tig.FoldInput{Commits: fs.Args(), Intent: *intent, Actor: c.actor})
	if err != nil {
		return err
	}
	commits, err := c.r.Commits(c.ctx)
	if err != nil {
		return err
	}
	return c.emit(map[string]any{"short": commits.Short(w.ID), "id": w.ID, "folds": fs.Args()}, func() {
		fmt.Fprintf(c.out, "%s folds %s\n", commits.Short(w.ID), strings.Join(fs.Args(), " "))
	})
}

func (c *cli) unmark(args []string) error {
	fs := c.flags("unmark")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	args = fs.Args()
	if len(args) == 0 {
		return errors.New("usage: tigwx unmark <id>...")
	}
	if err := c.open(); err != nil {
		return err
	}
	if err := c.r.Unmark(c.ctx, args); err != nil {
		return err
	}
	return c.emit(map[string]any{"unmarked": args}, func() {})
}

func (c *cli) sync(args []string) error {
	fs := c.flags("sync")
	var in tig.SyncInput
	fs.StringVar(&in.Reason, "m", "", "why the words changed: it makes the change a revision")
	fs.StringVar(&in.Source, "source", "", "what the change came from")
	c.who(fs, &in.By, &in.Via)
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if err := c.open(); err != nil {
		return err
	}
	in.Actor = c.actor
	res, err := c.r.SyncWith(c.ctx, in)
	if err != nil {
		return err
	}
	return c.emit(res, func() {
		fmt.Fprintf(c.out, "%d documents rewritten, %d intents touched, %d moved, %d old marks converted\n", len(res.Docs), len(res.Touched), len(res.Moved), res.Migrated)
		for _, i := range res.Touched {
			if in.Reason != "" {
				fmt.Fprintf(c.out, "revised %s: %s\n", i, in.Reason)
			} else {
				fmt.Fprintf(c.out, "touched %s: confirm it again\n", i)
			}
		}
		if in.Reason != "" && len(res.Touched) == 0 {
			fmt.Fprintln(c.out, "no edit reached an intent, so the reason was not recorded")
		}
	})
}

func (c *cli) decide(name string, args []string, do func(tig.DecideInput) (*tig.Intent, error), set func(fs *flag.FlagSet, in *tig.DecideInput)) error {
	fs := c.flags(name)
	var in tig.DecideInput
	fs.StringVar(&in.Reason, "m", "", "why")
	fs.StringVar(&in.Source, "source", "", "what it was decided from: a conversation, a ticket, a document")
	c.who(fs, &in.By, &in.Via)
	if set != nil {
		set(fs, &in)
	}
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: tigwx %s <intent>", name)
	}
	if err := c.open(); err != nil {
		return err
	}
	in.Intent, in.Actor = fs.Arg(0), c.actor
	i, err := do(in)
	if err != nil {
		return err
	}
	return c.emit(i, func() { fmt.Fprintf(c.out, "%s#%d  %s  %s\n", i.Short, i.Revision, i.State, i.Text) })
}

func (c *cli) confirm(args []string) error {
	return c.decide("confirm", args, func(in tig.DecideInput) (*tig.Intent, error) { return c.r.ConfirmIntent(c.ctx, in) },
		func(fs *flag.FlagSet, in *tig.DecideInput) {
			fs.StringVar(&in.Under, "under", "", "the section of the main document it goes under")
		})
}

func (c *cli) reject(args []string) error {
	return c.decide("reject", args, func(in tig.DecideInput) (*tig.Intent, error) { return c.r.RejectIntent(c.ctx, in) }, nil)
}

func (c *cli) retire(args []string) error {
	var with string
	return c.decide("retire", args, func(in tig.DecideInput) (*tig.Intent, error) {
		if with != "" {
			in.From = []string{with}
		}
		return c.r.RetireIntent(c.ctx, in)
	}, func(fs *flag.FlagSet, _ *tig.DecideInput) {
		fs.StringVar(&with, "for", "", "the intent that took its place")
	})
}

func (c *cli) accept(args []string) error {
	fs := c.flags("accept")
	reject := fs.Bool("reject", false, "record it as rejected")
	reason := fs.String("m", "", "why")
	var by, via string
	c.who(fs, &by, &via)
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: tigwx accept [-reject] [-m why] <id>")
	}
	if err := c.open(); err != nil {
		return err
	}
	state := tig.WorkAccepted
	if *reject {
		state = tig.WorkRejected
	}
	a, err := c.r.Accept(c.ctx, tig.AcceptInput{Work: fs.Arg(0), State: state, By: by, Via: via, Actor: c.actor, Reason: *reason})
	if err != nil {
		return err
	}
	return c.emit(a, func() { fmt.Fprintf(c.out, "%s %s\n", short(a.Work), a.State) })
}

func (c *cli) baseline(args []string) error {
	fs := c.flags("baseline")
	intent := fs.String("m", "the project as it was before tig", "what it is")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if err := c.open(); err != nil {
		return err
	}
	w, err := c.r.Baseline(c.ctx, *intent, c.actor)
	if err != nil {
		return err
	}
	commits, err := c.r.Commits(c.ctx)
	if err != nil {
		return err
	}
	return c.emit(map[string]any{"short": commits.Short(w.ID), "work": w}, func() { fmt.Fprintf(c.out, "%s  %s\n", commits.Short(w.ID), w.Intent) })
}
