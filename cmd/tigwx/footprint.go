package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/h94u041/tig"
)

func (c *cli) printWork(w tig.Work) {
	id := short(w.ID)
	if id == "" {
		id = "(in progress)"
	}
	at := w.At
	if at.IsZero() {
		at = w.Started
	}
	fmt.Fprintf(c.out, "  %s  %-11s %s  %s  %s\n", id, w.Result, at.Local().Format("2006-01-02 15:04"), w.Actor, w.Intent)
}

func (c *cli) trail(args []string) error {
	fs := c.flags("trail")
	steps := fs.Bool("steps", false, "each work's steps")
	n := fs.Int("n", 20, "how many works")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: tigwx trail [-steps] [-n 20] <block|file>")
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	ref := filepath.ToSlash(fs.Arg(0))
	files, err := tig.Files(c.ctx, c.r.Root, c.r.Skip)
	if err != nil {
		return err
	}
	var works []tig.Work
	if slices.Contains(files, ref) {
		works = a.FileTrail(c.ctx, ref)
	} else {
		b, err := a.Chain.Block(ref)
		if err != nil {
			return err
		}
		works = a.Trail(b.Hash)
	}
	if c.asJSON {
		type entry struct {
			tig.Work
			StepList []tig.Step `json:"stepList,omitempty"`
		}
		out := make([]entry, 0, len(works))
		for i, w := range works {
			if i == *n {
				break
			}
			e := entry{Work: w}
			if *steps {
				e.StepList = a.Steps(w)
			}
			out = append(out, e)
		}
		return printJSON(c.out, out)
	}
	for i, w := range works {
		if i == *n {
			fmt.Fprintf(c.out, "  ... %d more (-n)\n", len(works)-i)
			break
		}
		c.printWork(w)
		if !*steps {
			continue
		}
		for _, s := range a.Steps(w) {
			c.printStep(s)
		}
	}
	return nil
}

func (c *cli) printStep(s tig.Step) {
	line := s.Note
	if s.Command != "" {
		line = strings.TrimSpace("$ " + s.Command + "  " + line)
	}
	if s.Exit != nil {
		line += fmt.Sprintf("  (exit %d", *s.Exit)
		if s.Runner != "" {
			line += ", run by " + s.Runner
		}
		line += ")"
	}
	fmt.Fprintf(c.out, "      %-8s %s\n", s.Kind, line)
	if s.Before != "" && s.After != "" {
		if changes, err := tig.DiffTrees(c.ctx, c.r.Objects, s.Before, s.After); err == nil {
			printChanges(c.out, "               ", changes)
		}
	}
}

func (c *cli) work(args []string) error {
	const use = "usage: tigwx work start|step|finish|status"
	if len(args) == 0 {
		return errors.New(use)
	}
	if args[0] == "-h" || args[0] == "-help" || args[0] == "--help" {
		return c.parse(c.flags("work"), []string{"-h"})
	}
	if !slices.Contains([]string{"start", "step", "finish", "status"}, args[0]) {
		return errors.New(use)
	}
	// Asking how to use one of them does not need a project.
	if !slices.ContainsFunc(args[1:], func(a string) bool { return a == "-h" || a == "-help" || a == "--help" }) {
		if err := c.open(); err != nil {
			return err
		}
	}
	switch args[0] {
	case "start":
		return c.workStart(args[1:])
	case "step":
		return c.workStep(args[1:])
	case "finish":
		return c.workFinish(args[1:])
	case "status":
		fs := c.flags("work status")
		if err := c.parse(fs, args[1:]); err != nil {
			return err
		}
		w, err := c.r.Current()
		if err != nil {
			return err
		}
		if c.asJSON {
			return printJSON(c.out, map[string]any{"inProgress": w != nil, "work": w})
		}
		if w == nil {
			fmt.Fprintln(c.out, "no work in progress")
			return nil
		}
		fmt.Fprintf(c.out, "in progress since %s by %s: %s\n%d steps\n", w.Started.Local().Format("2006-01-02 15:04"), w.Actor, w.Intent, len(w.Steps))
		return nil
	}
	return errors.New(use)
}

func (c *cli) commits(args []string) error {
	fs := c.flags("commits")
	n := fs.Int("n", 20, "how many")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if err := c.open(); err != nil {
		return err
	}
	commits, err := c.r.Commits(c.ctx)
	if err != nil {
		return err
	}
	all := commits.All()
	slices.Reverse(all)
	if len(all) > *n {
		all = all[:*n]
	}
	l, err := c.r.Ledger(c.ctx)
	if err != nil {
		return err
	}
	type line struct {
		Short      string `json:"short"`
		ID         string `json:"id"`
		At         string `json:"at"`
		Result     string `json:"result"`
		Acceptance string `json:"acceptance"`
		Intent     string `json:"intent"`
		Actor      string `json:"actor,omitempty"`
	}
	lines := make([]line, 0, len(all))
	for _, w := range all {
		lines = append(lines, line{commits.Short(w.ID), w.ID, w.At.UTC().Format("2006-01-02T15:04:05Z"), w.Result, l.AcceptanceOf(w), w.Intent, w.Actor})
	}
	return c.emit(lines, func() {
		for i, w := range all {
			fmt.Fprintf(c.out, "%-6s %s  %-10s %-9s %s\n", lines[i].Short, w.At.Local().Format("2006-01-02 15:04"), w.Result, lines[i].Acceptance, w.Intent)
		}
	})
}

// many collects a flag given several times.
type many []string

func (m *many) String() string     { return strings.Join(*m, ",") }
func (m *many) Set(v string) error { *m = append(*m, v); return nil }

func (c *cli) set(args []string) error {
	fs := c.flags("set")
	var in tig.SetInput
	fs.Var((*many)(&in.Intents), "i", "an intent, by reference")
	fs.Var((*many)(&in.Sentences), "s", "a sentence, in its words")
	fs.Var((*many)(&in.Commits), "id", "a commit")
	fs.Var((*many)(&in.Files), "f", "a file")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	set := a.Relations(c.ctx).Set(in)
	if c.asJSON {
		return printJSON(c.out, set)
	}
	for _, p := range set.Pinned {
		fmt.Fprintf(c.out, "pinned   %s %s", p.Ref, p.Text)
		if p.Stale {
			fmt.Fprintf(c.out, " (it has moved on: #%d, %s)", p.Current, p.State)
		}
		fmt.Fprintln(c.out)
	}
	for _, s := range set.Sentences {
		fmt.Fprintf(c.out, "sentence %s:%d %s %s\n", s.Path, s.Line, s.Text, strings.Join(s.IDs, " "))
		if len(s.Parallel) > 0 {
			fmt.Fprintf(c.out, "         said to stand apart from %s; it is here for the code they share\n", strings.Join(s.Parallel, " "))
		}
	}
	for _, w := range set.Commits {
		fmt.Fprintf(c.out, "commit   %s %s\n", w.Short, w.Intent)
	}
	for _, f := range set.Files {
		fmt.Fprintf(c.out, "file     %s\n", f)
	}
	if set.Baseline {
		fmt.Fprintln(c.out, "a sentence names the baseline; its files are not listed")
	}
	for _, m := range set.Missing {
		fmt.Fprintf(c.out, "missing  %s\n", m)
	}
	if set.Digest != "" {
		fmt.Fprintf(c.out, "digest   %s\n", set.Digest)
	}
	return nil
}

func (c *cli) groups(args []string) error {
	fs := c.flags("groups")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	groups := a.Relations(c.ctx).Groups()
	if c.asJSON {
		return printJSON(c.out, groups)
	}
	for _, g := range groups {
		fmt.Fprintf(c.out, "%-11s %-8s %d sentences, %d commits, %d files\n", g.Kind, g.Key, len(g.Sentences), len(g.Commits), len(g.Files))
	}
	return nil
}

func (c *cli) commit(args []string) error {
	fs := c.flags("commit")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	args = fs.Args()
	if len(args) != 1 {
		return errors.New("usage: tigwx commit <id>")
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	v, err := a.Commit(args[0])
	if err != nil {
		return err
	}
	if c.asJSON {
		return printJSON(c.out, v)
	}
	fmt.Fprintf(c.out, "%s  %s\n%s  %s by %s\n", v.Short, v.ID, v.At.Local().Format("2006-01-02 15:04"), v.Result, v.Actor)
	fmt.Fprintf(c.out, "\n  %s\n\n", v.Intent)
	if len(v.ChildIDs) > 0 {
		fmt.Fprintf(c.out, "folds: %s\n", strings.Join(v.ChildIDs, ", "))
	}
	if len(v.Files) > 0 {
		fmt.Fprintf(c.out, "files: %s\n", strings.Join(v.Files, ", "))
	}
	for _, m := range v.Marks {
		fmt.Fprintf(c.out, "sentence: %s:%d  %s\n", m.Path, m.Line, m.Text)
	}
	return nil
}

// basis turns block references into hashes.
func (c *cli) basis(refs []string) ([]string, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	ch, err := c.r.Chain(c.ctx, "")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, ref := range refs {
		b, err := ch.Block(ref)
		if err != nil {
			return nil, err
		}
		out = append(out, b.Hash)
	}
	return out, nil
}

func (c *cli) workStart(args []string) error {
	fs := c.flags("work start")
	intent := fs.String("m", "", "what the work is meant to do")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if *intent == "" {
		return &flagError{fs.Name(), errors.New("say what the work is for: -m intent")}
	}
	basis, err := c.basis(fs.Args())
	if err != nil {
		return err
	}
	w, err := c.r.StartWork(c.ctx, basis, *intent, c.actor)
	if err != nil {
		return err
	}
	return c.emit(w, func() { fmt.Fprintf(c.out, "work started (snapshot %s)\n", short(w.Before)) })
}

func (c *cli) workStep(args []string) error {
	fs := c.flags("work step")
	kind := fs.String("kind", "", "edit, command, install, check, decision or note")
	note := fs.String("m", "", "what was done")
	command := fs.String("command", "", "the command that was run")
	exit := fs.Int("exit", 0, "its exit code")
	output := fs.String("output", "", "a file with its output (- for stdin)")
	detail := fs.String("detail", "", "a decision's question and answer, or a longer note")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	switch *kind {
	case tig.StepEdit, tig.StepCommand, tig.StepInstall, tig.StepCheck, tig.StepDecision, tig.StepNote:
	default:
		return &flagError{fs.Name(), fmt.Errorf("unknown step kind %q (edit, command, install, check, decision, note)", *kind)}
	}
	in := tig.StepInput{Kind: *kind, Note: *note, Command: *command, Detail: *detail, Actor: c.actor}
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "exit" {
			in.Exit = exit
		}
	})
	if *kind == tig.StepCheck && in.Exit == nil {
		return &flagError{fs.Name(), errors.New("a check needs its -exit code")}
	}
	if *output != "" {
		var data []byte
		var err error
		if *output == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(*output)
		}
		if err != nil {
			return err
		}
		in.Output = string(data)
	}
	var err error
	if in.Basis, err = c.basis(fs.Args()); err != nil {
		return err
	}
	s, err := c.r.RecordStep(c.ctx, in)
	if err != nil {
		return err
	}
	if c.asJSON {
		return printJSON(c.out, s)
	}
	fmt.Fprintf(c.out, "step %s (%s)\n", short(s.ID), s.Kind)
	if s.Before != "" && s.After != "" {
		changes, err := tig.DiffTrees(c.ctx, c.r.Objects, s.Before, s.After)
		if err != nil {
			return err
		}
		printChanges(c.out, "  ", changes)
	}
	return nil
}

func (c *cli) workFinish(args []string) error {
	fs := c.flags("work finish")
	result := fs.String("result", "", "passed, failed, done, rolled-back or abandoned (default: from the checks)")
	approvers := fs.String("approver", "", "comma-separated: who approved the work")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	switch *result {
	case "", tig.ResultPassed, tig.ResultFailed, tig.ResultDone, tig.ResultRolledBack, tig.ResultAbandoned:
	default:
		return &flagError{fs.Name(), fmt.Errorf("unknown result %q", *result)}
	}
	var who []string
	for _, a := range strings.Split(*approvers, ",") {
		if a = strings.TrimSpace(a); a != "" {
			who = append(who, a)
		}
	}
	w, err := c.r.FinishWork(c.ctx, *result, who)
	if err != nil {
		return err
	}
	return c.emit(w, func() { fmt.Fprintf(c.out, "work %s: %s, %d steps\n", short(w.ID), w.Result, len(w.Steps)) })
}
