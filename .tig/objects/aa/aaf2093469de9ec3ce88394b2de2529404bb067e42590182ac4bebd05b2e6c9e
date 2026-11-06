package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/h94u041/tig"
)

// status says where the project stands. It only reads, and it never says whether
// that is good: check does, and its exit status says it.
func (c *cli) status(args []string) error {
	fs := c.flags("status")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("usage: tigwx status")
	}
	if err := c.open(); err != nil {
		return err
	}
	cfg, err := c.r.Config()
	if err != nil {
		return err
	}
	head, err := c.r.Head(c.ctx)
	if err != nil {
		return err
	}
	open, err := c.r.Current()
	if err != nil {
		return err
	}
	var changes []tig.Change
	if head.ID != "" {
		if changes, err = c.r.Changed(c.ctx, head.Tree); err != nil {
			return err
		}
	}
	a, err := c.r.Analyze(c.ctx, "")
	if err != nil {
		return err
	}
	commits, err := c.r.Commits(c.ctx)
	if err != nil {
		return err
	}
	leftOut, err := tig.LeftOut(c.ctx, c.r.Root, c.r.Skip)
	if err != nil {
		return err
	}
	out := statusOut{
		LeftOut: leftOut,
		Root:    c.r.Root, Main: cfg.Root,
		Actor: c.actor, ActorFrom: c.actorFrom, By: c.by, ByFrom: c.byFrom,
		Changes: changes, Closed: a.Report.Closed,
	}
	if head.ID != "" {
		out.Snapshot = &head
	}
	if open != nil {
		out.Work = &statusWork{Intent: open.Intent, Actor: open.Actor, Steps: len(open.Steps), Started: open.Started}
	}
	blocks := make([]*tig.Block, 0, len(a.Chain.Order)+len(a.Chain.InferredBlocks))
	for _, hash := range a.Chain.Order {
		blocks = append(blocks, a.Chain.Blocks[hash])
	}
	blocks = append(blocks, a.Chain.InferredBlocks...)
	for _, b := range blocks {
		for _, s := range b.Sentences {
			if s.Unsynced {
				out.Unsynced++
			}
		}
	}
	out.ProposedRelations = a.Ledger.ProposedRelations()
	for _, i := range a.Ledger.Intents() {
		if i.State == tig.IntentProposed {
			out.ProposedIntents++
		}
		if len(i.Forks) > 0 {
			out.Disputed++
		}
		for _, e := range a.Ledger.LinksOf(i.ID) {
			if a.Ledger.LinkState(e) == tig.LinkProposed {
				out.ProposedLinks++
			}
		}
	}
	for _, w := range commits.All() {
		if a.Ledger.AcceptanceOf(w) == tig.WorkCandidate {
			out.Candidates++
		}
	}
	return c.emit(out, func() { c.printStatus(out) })
}

type statusWork struct {
	Intent  string    `json:"intent"`
	Actor   string    `json:"actor,omitempty"`
	Steps   int       `json:"steps"`
	Started time.Time `json:"started"`
}

type statusOut struct {
	Root            string        `json:"root"`
	Main            string        `json:"main"`
	Actor           string        `json:"actor"`
	ActorFrom       string        `json:"actorFrom"`
	By              string        `json:"by"`
	ByFrom          string        `json:"byFrom"`
	Snapshot        *tig.Snapshot `json:"snapshot,omitempty"`
	Work            *statusWork   `json:"work,omitempty"`
	Changes         []tig.Change  `json:"changes"`
	Unsynced        int           `json:"unsynced"`
	ProposedIntents int           `json:"proposedIntents"`
	ProposedLinks   int           `json:"proposedLinks"`
	Candidates      int           `json:"candidates"`
	Closed          bool          `json:"closed"`
	// ProposedRelations and Disputed are what wants a decision besides those above.
	ProposedRelations int `json:"proposedRelations"`
	Disputed          int `json:"disputed"`
	// LeftOut is what looks like a key or a credential, and so is in no snapshot.
	LeftOut []string `json:"leftOut,omitempty"`
}

func (c *cli) printStatus(s statusOut) {
	fmt.Fprintf(c.out, "project: %s\n", s.Root)
	fmt.Fprintf(c.out, "main document: %s\n", s.Main)
	fmt.Fprintf(c.out, "acting as: %s (%s), by %s (%s)\n", s.Actor, s.ActorFrom, s.By, s.ByFrom)
	if s.Snapshot != nil {
		fmt.Fprintf(c.out, "last snapshot: %s  %s  %s\n", short(s.Snapshot.ID), s.Snapshot.At.Local().Format("2006-01-02 15:04"), s.Snapshot.Reason)
	} else {
		fmt.Fprintln(c.out, "last snapshot: none")
	}
	if s.Work != nil {
		fmt.Fprintf(c.out, "work in progress: %s (%d steps, %s)\n", s.Work.Intent, s.Work.Steps, s.Work.Actor)
	}
	if len(s.Changes) > 0 {
		fmt.Fprintln(c.out, "changed since the last snapshot:")
		printChanges(c.out, "  ", s.Changes)
	}
	fmt.Fprintf(c.out, "lines waiting for a sync: %d\n", s.Unsynced)
	fmt.Fprintf(c.out, "proposed intents: %d\n", s.ProposedIntents)
	fmt.Fprintf(c.out, "proposed links: %d\n", s.ProposedLinks)
	if s.ProposedRelations > 0 {
		fmt.Fprintf(c.out, "proposed relations: %d\n", s.ProposedRelations)
	}
	if s.Disputed > 0 {
		fmt.Fprintf(c.out, "disputed intents: %d (decisions taken apart; see tigwx log <intent>)\n", s.Disputed)
	}
	fmt.Fprintf(c.out, "candidate commits: %d\n", s.Candidates)
	closed := "no"
	if s.Closed {
		closed = "yes"
	}
	fmt.Fprintf(c.out, "chain closed: %s\n", closed)
	if len(s.LeftOut) > 0 {
		fmt.Fprintln(c.out, "left out of snapshots because they look like secrets (git add one to keep it):")
		for _, f := range s.LeftOut {
			fmt.Fprintf(c.out, "  %s\n", f)
		}
	}
}

// start is the folder tigwx was pointed at.
func (c *cli) start() (string, error) {
	if c.dir != "" {
		return filepath.Abs(c.dir)
	}
	return os.Getwd()
}

func (c *cli) init(args []string) error {
	fs := c.flags("init")
	root := fs.String("root", "README.md", "the main document")
	outside := fs.String("outside", "", "comma-separated path prefixes that are not part of the chain")
	force := fs.Bool("force", false, "the code there is one commit, and every line of the documents goes with it")
	by := fs.String("by", c.by, "who says so: human, ai, tool, or your own word")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	dir, err := c.start()
	if err != nil {
		return err
	}
	r := tig.Open(dir)
	existed := r.Exists()
	cfg := tig.Config{Root: *root}
	for _, p := range strings.Split(*outside, ",") {
		if p = strings.TrimSpace(p); p != "" {
			cfg.Outside = append(cfg.Outside, p)
		}
	}
	if err := r.Init(cfg); err != nil {
		return err
	}
	s, _, err := r.Take(c.ctx, tig.ReasonOpen, c.actor)
	if err != nil {
		return err
	}
	cfg, err = r.Config()
	if err != nil {
		return err
	}
	out := struct {
		Dir      string               `json:"dir"`
		Existed  bool                 `json:"existed"`
		Root     string               `json:"root"`
		Snapshot string               `json:"snapshot"`
		Force    *tig.InitForceResult `json:"force,omitempty"`
	}{Dir: r.Dir, Existed: existed, Root: cfg.Root, Snapshot: s.ID}
	if *force {
		res, err := r.InitForce(c.ctx, tig.InitForceInput{Config: cfg, By: *by, Actor: c.actor})
		if err != nil {
			return err
		}
		out.Force = &res
	}
	return c.emit(out, func() {
		if existed {
			fmt.Fprintf(c.out, "tig already in %s (config kept)\n", r.Dir)
		} else {
			fmt.Fprintf(c.out, "tig started in %s\n", r.Dir)
		}
		fmt.Fprintf(c.out, "main document %s, snapshot %s\n", cfg.Root, short(s.ID))
		if out.Force != nil {
			fmt.Fprintf(c.out, "baseline %s; %d lines marked in %d documents\n", short(out.Force.Baseline.ID), out.Force.Lines, len(out.Force.Docs))
		}
	})
}

func (c *cli) snap(args []string) error {
	fs := c.flags("snap")
	reason := fs.String("reason", tig.ReasonScan, "why the snapshot is taken")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if err := c.open(); err != nil {
		return err
	}
	s, changed, err := c.r.Take(c.ctx, *reason, c.actor)
	if err != nil {
		return err
	}
	return c.emit(map[string]any{"snapshot": s, "changed": changed}, func() {
		if !changed {
			fmt.Fprintf(c.out, "nothing changed since snapshot %s\n", short(s.ID))
			return
		}
		fmt.Fprintf(c.out, "snapshot %s\n", short(s.ID))
	})
}

func (c *cli) log(args []string) error {
	fs := c.flags("log")
	n := fs.Int("n", 20, "how many snapshots")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return errors.New("usage: tigwx log [-n 20] [intent]")
	}
	if err := c.open(); err != nil {
		return err
	}
	if fs.NArg() == 1 {
		return c.intentLog(fs.Arg(0))
	}
	snaps, err := c.r.Log(c.ctx, *n)
	if err != nil {
		return err
	}
	return c.emit(snaps, func() {
		for _, s := range snaps {
			fmt.Fprintf(c.out, "%s  %s  %-20s %s\n", short(s.ID), s.At.Local().Format("2006-01-02 15:04"), s.Reason, s.Actor)
		}
	})
}

// intentLog is everything that happened to an intent, oldest first.
func (c *cli) intentLog(ref string) error {
	l, err := c.r.Ledger(c.ctx)
	if err != nil {
		return err
	}
	i, err := l.Intent(ref)
	if err != nil {
		return err
	}
	return c.emit(i, func() {
		fmt.Fprintf(c.out, "%s#%d  %s\n", i.Short, i.Revision, i.State)
		for _, e := range i.Log {
			fmt.Fprintf(c.out, "%s  ev%s  #%d %s", e.At.Local().Format("2006-01-02 15:04"), e.ID[:6], e.Revision, e.Kind)
			if who := strings.TrimSpace(e.By + " " + e.Via + " " + e.Actor); who != "" {
				fmt.Fprintf(c.out, "  (%s)", who)
			}
			fmt.Fprintln(c.out)
			if e.Text != "" {
				fmt.Fprintf(c.out, "  text    %s\n", e.Text)
			}
			if e.Reason != "" {
				fmt.Fprintf(c.out, "  why     %s\n", e.Reason)
			}
			if e.Source != "" {
				fmt.Fprintf(c.out, "  source  %s\n", e.Source)
			}
			if len(e.From) > 0 {
				fmt.Fprintf(c.out, "  from    %s\n", strings.Join(e.From, " "))
			}
		}
		for _, f := range i.Forks {
			fmt.Fprintf(c.out, "disputed: decisions taken apart after ev%s; the last stands only by the order of the clocks:\n", f.After[:6])
			for _, e := range f.Events {
				fmt.Fprintf(c.out, "  ev%s  %s  (%s)\n", e.ID[:6], e.Kind, strings.TrimSpace(e.By+" "+e.Actor))
			}
		}
	})
}

// snapshot is a snapshot by a unique prefix of its hash, or HEAD.
func (c *cli) snapshot(ref string) (tig.Snapshot, error) {
	if ref == "" || ref == "HEAD" {
		s, err := c.r.Head(c.ctx)
		if err == nil && s.ID == "" {
			err = errors.New("no snapshot yet")
		}
		return s, err
	}
	if len(ref) < 6 {
		return tig.Snapshot{}, fmt.Errorf("%q: give 6 or more characters of the snapshot", ref)
	}
	snaps, err := c.r.Log(c.ctx, 0)
	if err != nil {
		return tig.Snapshot{}, err
	}
	var found []tig.Snapshot
	for _, s := range snaps {
		if strings.HasPrefix(s.ID, ref) {
			found = append(found, s)
		}
	}
	switch len(found) {
	case 0:
		return tig.Snapshot{}, fmt.Errorf("no snapshot %q", ref)
	case 1:
		return found[0], nil
	}
	return tig.Snapshot{}, fmt.Errorf("%q is more than one snapshot", ref)
}

func (c *cli) diff(args []string) error {
	fs := c.flags("diff")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	args = fs.Args()
	if err := c.open(); err != nil {
		return err
	}
	if len(args) > 2 {
		return errors.New("usage: tigwx diff [from [to]]")
	}
	from, err := c.snapshot(at(args, 0))
	if err != nil {
		return err
	}
	var changes []tig.Change
	if len(args) == 2 {
		to, err := c.snapshot(args[1])
		if err != nil {
			return err
		}
		changes, err = tig.DiffTrees(c.ctx, c.r.Objects, from.Tree, to.Tree)
		if err != nil {
			return err
		}
	} else if changes, err = c.r.Changed(c.ctx, from.Tree); err != nil {
		return err
	}
	return c.emit(changes, func() { printChanges(c.out, "", changes) })
}

func at(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

func printChanges(w io.Writer, indent string, changes []tig.Change) {
	ops := map[string]string{"add": "A", "modify": "M", "delete": "D"}
	for _, ch := range changes {
		fmt.Fprintf(w, "%s%s %s\n", indent, ops[ch.Op], ch.Path)
	}
}

func (c *cli) restore(args []string) error {
	fs := c.flags("restore")
	all := fs.Bool("all", false, "every file that differs from the snapshot")
	if err := c.parse(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: tigwx restore [-all] <snapshot> [path...]")
	}
	if err := c.open(); err != nil {
		return err
	}
	s, err := c.snapshot(fs.Arg(0))
	if err != nil {
		return err
	}
	paths := fs.Args()[1:]
	if len(paths) == 0 {
		if !*all {
			return &flagError{fs.Name(), errors.New("name the files to restore, or give -all")}
		}
		changes, err := c.r.Changed(c.ctx, s.Tree)
		if err != nil {
			return err
		}
		for _, ch := range changes {
			paths = append(paths, ch.Path)
		}
	}
	// The project as it is now is kept first, so the restore can be undone.
	before, _, err := c.r.Take(c.ctx, tig.ReasonScan, c.actor)
	if err != nil {
		return err
	}
	if err := c.r.Restore(c.ctx, s.Tree, paths); err != nil {
		return err
	}
	return c.emit(map[string]any{"restored": paths, "from": s.ID, "before": before.ID}, func() {
		fmt.Fprintf(c.out, "restored %d files from %s (the project before it is snapshot %s)\n", len(paths), short(s.ID), short(before.ID))
	})
}
