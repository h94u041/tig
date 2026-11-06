package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/h94u041/tig"
)

func heading(b *tig.Block) string {
	if b.Level == 0 {
		return b.Path
	}
	return strings.Repeat("#", b.Level) + " " + b.Title
}

func (c *cli) blocks(args []string) error {
	fs := c.flags("blocks")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	args = fs.Args()
	if len(args) != 1 {
		return errors.New("usage: tigwx blocks <file.md>")
	}
	file := args[0]
	if c.dir != "" && !filepath.IsAbs(file) {
		file = filepath.Join(c.dir, file) // -C is as if started there
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	// A block's hash includes its path in the project.
	rel := filepath.ToSlash(args[0])
	if c.open() == nil {
		if abs, err := filepath.Abs(file); err == nil {
			if p, err := filepath.Rel(c.r.Root, abs); err == nil && !strings.HasPrefix(p, "..") {
				rel = filepath.ToSlash(p)
			}
		}
	}
	blocks := tig.ParseBlocks(rel, data)
	return c.emit(blocks, func() {
		for _, b := range blocks {
			fmt.Fprintf(c.out, "%s  %s%s  (lines %d-%d)\n", short(b.Hash), strings.Repeat("  ", max(b.Level-1, 0)), heading(&b), b.Start, b.End)
			for _, cmd := range b.Accept {
				fmt.Fprintf(c.out, "              %s$ %s\n", strings.Repeat("  ", b.Level), cmd)
			}
		}
	})
}

func (c *cli) chain(args []string) error {
	fs := c.flags("chain")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: tigwx chain")
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	blocks := make([]*tig.Block, 0, len(a.Chain.Order))
	for _, h := range a.Chain.Order {
		blocks = append(blocks, a.Chain.Blocks[h])
	}
	return c.emit(blocks, func() {
		for _, b := range blocks {
			fmt.Fprintf(c.out, "%s  %s%s\n", short(b.Hash), strings.Repeat("  ", max(b.Level-1, 0)), heading(b))
		}
	})
}

func (c *cli) show(args []string) error {
	fs := c.flags("show")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	args = fs.Args()
	if len(args) != 1 {
		return errors.New("usage: tigwx show <block|reference>")
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	ch := a.Chain
	b, err := ch.Block(args[0])
	if err != nil {
		if _, perr := tig.ParseRef(args[0]); perr != nil {
			return err
		}
		return c.showRef(args[0])
	}
	if c.asJSON {
		var stale []tig.Finding
		for _, f := range a.Report.Stale {
			if f.Block == b.Hash {
				stale = append(stale, f)
			}
		}
		return printJSON(c.out, map[string]any{"kind": "block", "block": b, "handsOn": ch.Details[b.Hash], "links": ch.Refs[b.Hash], "built": a.Built[b.Hash],
			"was": a.Before(b.Hash), "stale": stale, "works": a.Trail(b.Hash)})
	}
	fmt.Fprintf(c.out, "%s\n%s:%d-%d  %s\n", b.Hash, b.Path, b.Start, b.End, strings.Join(b.Heading, " > "))
	if b.Text != "" {
		fmt.Fprintf(c.out, "\n  %s\n\n", strings.ReplaceAll(b.Text, "\n", "\n  "))
	}
	list := func(label string, items []string) {
		if len(items) > 0 {
			fmt.Fprintf(c.out, "%s: %s\n", label, strings.Join(items, ", "))
		}
	}
	list("hands on to", ch.Details[b.Hash])
	list("links to", ch.Refs[b.Hash])
	list("built", a.Built[b.Hash])
	var was []string
	for _, m := range a.Before(b.Hash) {
		was = append(was, short(m.Old))
	}
	list("was", was)
	for _, f := range a.Report.Stale {
		if f.Block == b.Hash {
			fmt.Fprintf(c.out, "stale: %s: %s\n", f.Path, f.Message)
		}
	}
	for i, w := range a.Trail(b.Hash) {
		if i == 0 {
			fmt.Fprintln(c.out, "works:")
		}
		if i == 5 {
			fmt.Fprintln(c.out, "  ... (tigwx trail for all)")
			break
		}
		c.printWork(w)
	}
	return nil
}

func (c *cli) search(args []string) error {
	fs := c.flags("search")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	args = fs.Args()
	if len(args) == 0 {
		return errors.New("usage: tigwx search <words>")
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	files, err := tig.Files(c.ctx, c.r.Root, c.r.Skip)
	if err != nil {
		return err
	}
	hits := a.Search(strings.Join(args, " "), files)
	return c.emit(hits, func() {
		for _, h := range hits {
			fmt.Fprintf(c.out, "%-5s %s  %s  %s\n", h.Kind, short(h.Ref), h.Path, h.Title)
			if h.Snippet != "" {
				fmt.Fprintf(c.out, "      %s\n", h.Snippet)
			}
		}
	})
}

func (c *cli) check(args []string) error {
	fs := c.flags("check")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	rep := a.Report
	if err := c.emit(rep, func() { c.printReport(rep) }); err != nil {
		return err
	}
	if !rep.Closed {
		return errNo
	}
	return nil
}

func (c *cli) printReport(rep tig.Report) {
	closed := "no"
	if rep.Closed {
		closed = "yes"
	}
	fmt.Fprintf(c.out, "chain from %s: %d documents, %d blocks, %d lines with work\n", rep.Root, rep.Docs, rep.Blocks, rep.Marks)
	fmt.Fprintf(c.out, "closed: %s\n", closed)
	findings := func(label string, fs []tig.Finding) {
		if len(fs) == 0 {
			return
		}
		fmt.Fprintf(c.out, "\n%s:\n", label)
		for _, f := range fs {
			where := f.Path
			if f.Line > 0 {
				where = fmt.Sprintf("%s:%d", f.Path, f.Line)
			}
			if f.Title != "" {
				where += " (" + f.Title + ")"
			}
			fmt.Fprintf(c.out, "  %s: %s\n", where, f.Message)
		}
	}
	findings("broken", rep.Broken)
	findings("stale", rep.Stale)
	if len(rep.Undesigned) > 0 {
		fmt.Fprintf(c.out, "\nno design reaches:\n")
		for _, f := range rep.Undesigned {
			fmt.Fprintf(c.out, "  %s\n", f)
		}
	}
}

// showRef prints what a reference means.
func (c *cli) showRef(ref string) error {
	v, err := c.r.Show(c.ctx, ref)
	if err != nil {
		return err
	}
	if c.asJSON {
		return printJSON(c.out, v)
	}
	switch {
	case v.Intent != nil:
		i := v.Intent
		who := i.By
		if i.Via != "" {
			who = strings.TrimSpace(who + " via " + i.Via)
		}
		fmt.Fprintf(c.out, "%s#%d  %s", i.Short, i.Revision, i.State)
		if who != "" {
			fmt.Fprintf(c.out, " (%s)", who)
		}
		fmt.Fprintf(c.out, "\n  text   %s\n", i.Text)
		for _, p := range i.Places {
			fmt.Fprintf(c.out, "  place  %s:%d  %s\n", p.Path, p.Line, p.Ref)
		}
		for _, w := range i.Works {
			fmt.Fprintf(c.out, "  work   %s %s #%d (%s, link %s)\n", w.Short, w.Kind, w.Revision, w.Acceptance, w.State)
		}
		if i.Previous != nil {
			fmt.Fprintf(c.out, "  before %s %q", i.Previous.Ref, i.Previous.Text)
			if i.Previous.Reason != "" {
				fmt.Fprintf(c.out, ": %s", i.Previous.Reason)
			}
			fmt.Fprintln(c.out)
		}
		for _, rel := range i.Relations {
			arrow := rel.Kind + " " + rel.With
			if rel.Incoming {
				arrow = rel.With + " " + rel.Kind + " this"
			}
			fmt.Fprintf(c.out, "  relate %s (%s)", arrow, rel.State)
			if rel.Reason != "" {
				fmt.Fprintf(c.out, ": %s", rel.Reason)
			}
			fmt.Fprintln(c.out)
		}
		for _, f := range i.Forks {
			fmt.Fprintf(c.out, "  fork   after ev%s:", f.After[:6])
			for _, e := range f.Events {
				fmt.Fprintf(c.out, " %s by %s", e.Kind, strings.TrimSpace(e.By+" "+e.Actor))
			}
			fmt.Fprintln(c.out, "; see tigwx log, and decide again")
		}
		fmt.Fprintf(c.out, "  sync   %s\n", i.Sync)
	case v.Line != nil:
		fmt.Fprintf(c.out, "%s\n  text   %s\n", v.Line.Hash, v.Line.Text)
		if v.Line.Range != "" {
			fmt.Fprintf(c.out, "  range  %s\n", v.Line.Range)
		}
		for _, p := range v.Line.Places {
			fmt.Fprintf(c.out, "  place  %s:%d\n", p.Path, p.Line)
		}
		for _, i := range v.Line.Intents {
			fmt.Fprintf(c.out, "  intent %s\n", i)
		}
	case v.Work != nil:
		fmt.Fprintf(c.out, "%s  %s  %s\n", v.Work.Short, v.Work.Acceptance, v.Work.Intent)
		for _, f := range v.Work.Files {
			fmt.Fprintf(c.out, "  file   %s\n", f)
		}
		for _, i := range v.Work.Intents {
			fmt.Fprintf(c.out, "  intent %s\n", i)
		}
	case v.Event != nil:
		fmt.Fprintf(c.out, "ev%s  %s %s #%d  %s via %s  %s\n", v.Event.ID[:8], v.Event.Short, v.Event.Kind, v.Event.Revision, v.Event.By, v.Event.Via, v.Event.Reason)
	}
	return nil
}
