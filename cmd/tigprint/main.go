// Command tigprint keeps a project's tig: snapshots of the project, the
// footprint of how it was built, and the chain of its documents, which it
// checks and serves to agents over MCP.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/h94u041/tig"
	"github.com/h94u041/tig/hub"
	"github.com/h94u041/tig/mcpserver"
)

var version = "0.1.0"

const usage = `tigprint keeps a project's tig: the footprint of how the project was built
and the chain of its documents, checked and served to agents over MCP.

Usage: tigprint [-C dir] [-actor name] [-session name] <command> [arguments]

Project
  init [-root README.md] [-strict] [-outside path,...]  start a tig in this folder
  snap [-reason why]                                     record the project as it is
  log [-n 20]                                            the snapshots, newest first
  diff [from [to]]                                       what changed (default: the last snapshot and now)
  restore [-all] <snapshot> [path...]                    put files back as they were in a snapshot

Chain
  blocks <file.md>                                       a document's text blocks and their hashes
  chain                                                  every block of the chain
  show <block|reference>                                 one block: its text, details and what was built for it; or
                                                         what a reference means: i7f3a, i7f3a#2, lc3d4e5:5-14, a1b2, ev91c0
  search <words>                                         blocks, works, steps and files containing every word
  check [-json]                                          is the chain closed? (exit status 1 when not)

Footprint
  trail [-steps] [-n 20] <block|file>                    the works done for a block or a file
  work start -m intent [block...]                        start a work
  work step -kind kind [-m note] [-command c] [-exit n] [-output file] [block...]
                                                         record a step of it (kind: edit, command,
                                                         install, check, decision, note)
  work finish [-result r] [-approver name,...]           finish it
  work status                                            the work in progress
  commits [-n 20]                                        the commits, newest first, by short ID
  commit <id>                                            one commit: its files, its sentences, what it folds
  set [-json] [-s sentence] [-id a1b2] [-f file]...      what a piece of work needs: sentences, commits, files
  groups [-json]                                         the sentences, commits and files that belong together

Marks
  mark [-id a1b2,c3d4] [-doc f.md] [-under title] [-inferred] [-confirm] <sentence>
                                                         say what a line or a part of it means, and which commits
                                                         follow it. -inferred: a guess, for the inferred document
                                                         unless the main document says it already
  sync                                                   bring the ledger up to edited lines; old marks are converted
  confirm [-under title] [-m why] <intent>               it is what was meant; out of the inferred document if there
  reject [-m why] <intent>                               it is not what was meant
  retire [-by i1a2b] [-m why] <intent>                   it no longer stands, or another took its place
  accept [-reject] [-m why] <id>                         record a decision on a commit; tig judges nothing
  fold -m intent <id> <id>...                            fold commits under a new one; none disappear
  unmark <id>...                                         withdraw undone commits from what they followed; the sentences stay
  baseline -m intent                                     one commit for all the project was before tig

Agents
  mcp                                                    serve this tig over MCP on stdin and stdout

Hub
  push [-token t] [-m note] <url>                        send the documents and the chain to a hub
                                                         (url http://host/owner/name, or $TIG_HUB;
                                                         the token is -token or $TIG_TOKEN)
  version

A block is named by its content hash; any unique prefix of 8 or more characters
works, and a commit by its short ID. The actor is -actor, or $TIG_ACTOR, or
"cli". Agents working at once each use their own -session.
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

type cli struct {
	ctx     context.Context
	out     io.Writer
	errs    io.Writer
	dir     string // -C
	actor   string
	session string
	r       *tig.Repo
}

// errNotClosed ends check with exit status 1 and nothing more to say.
var errNotClosed = errors.New("not closed")

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	g := flag.NewFlagSet("tigprint", flag.ContinueOnError)
	g.SetOutput(stderr)
	g.Usage = func() { fmt.Fprint(stderr, usage) }
	dir := g.String("C", "", "the project folder")
	actor := g.String("actor", "", "who is acting")
	session := g.String("session", "", "keep this work in progress apart from other agents'")
	if err := g.Parse(args); err != nil {
		return 2
	}
	if g.NArg() == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	c := &cli{ctx: ctx, out: stdout, errs: stderr, dir: *dir, actor: *actor, session: *session}
	if c.actor == "" {
		c.actor = os.Getenv("TIG_ACTOR")
	}
	if c.actor == "" {
		c.actor = "cli"
	}
	cmds := map[string]func([]string) error{
		"init": c.init, "snap": c.snap, "log": c.log, "diff": c.diff, "restore": c.restore,
		"blocks": c.blocks, "chain": c.chain, "show": c.show, "search": c.search, "check": c.check,
		"trail": c.trail, "work": c.work, "mcp": c.mcp, "push": c.push,
		"commits": c.commits, "commit": c.commit, "mark": c.mark, "fold": c.fold, "unmark": c.unmark, "baseline": c.baseline,
		"sync": c.sync, "confirm": c.confirm, "reject": c.reject, "retire": c.retire, "accept": c.accept,
		"set": c.set, "groups": c.groups,
		"version": func([]string) error { fmt.Fprintf(c.out, "tigprint %s\n", version); return nil },
	}
	name := g.Arg(0)
	cmd, ok := cmds[name]
	if !ok {
		fmt.Fprintf(stderr, "tigprint: unknown command %q\n\n%s", name, usage)
		return 2
	}
	if err := cmd(g.Args()[1:]); err != nil {
		if errors.Is(err, errNotClosed) {
			return 1
		}
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stderr, "tigprint:", err)
		}
		return 2
	}
	return 0
}

func (c *cli) flags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("tigprint "+name, flag.ContinueOnError)
	fs.SetOutput(c.errs)
	return fs
}

// start is the folder tigprint was pointed at.
func (c *cli) start() (string, error) {
	if c.dir != "" {
		return filepath.Abs(c.dir)
	}
	return os.Getwd()
}

// open finds the project: the nearest folder at or above the start that has
// a .tig.
func (c *cli) open() error {
	start, err := c.start()
	if err != nil {
		return err
	}
	for d := start; ; d = filepath.Dir(d) {
		if fi, err := os.Stat(filepath.Join(d, ".tig")); err == nil && fi.IsDir() {
			c.r = tig.Open(d)
			c.r.Session = c.session
			return nil
		}
		if filepath.Dir(d) == d {
			return fmt.Errorf("no .tig in %s or above it (run tigprint init)", start)
		}
	}
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func when(w interface{ IsZero() bool }, s string) string {
	if w.IsZero() {
		return "-"
	}
	return s
}

func (c *cli) init(args []string) error {
	fs := c.flags("init")
	root := fs.String("root", "README.md", "the main document")
	outside := fs.String("outside", "", "comma-separated path prefixes that are not part of the chain")
	if err := fs.Parse(args); err != nil {
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
	if existed {
		fmt.Fprintf(c.out, "tig already in %s (config kept)\n", r.Dir)
	} else {
		fmt.Fprintf(c.out, "tig started in %s\n", r.Dir)
	}
	fmt.Fprintf(c.out, "main document %s, snapshot %s\n", cfg.Root, short(s.ID))
	return nil
}

func (c *cli) snap(args []string) error {
	fs := c.flags("snap")
	reason := fs.String("reason", tig.ReasonScan, "why the snapshot is taken")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.open(); err != nil {
		return err
	}
	s, changed, err := c.r.Take(c.ctx, *reason, c.actor)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Fprintf(c.out, "nothing changed since snapshot %s\n", short(s.ID))
		return nil
	}
	fmt.Fprintf(c.out, "snapshot %s\n", short(s.ID))
	return nil
}

func (c *cli) log(args []string) error {
	fs := c.flags("log")
	n := fs.Int("n", 20, "how many snapshots")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.open(); err != nil {
		return err
	}
	snaps, err := c.r.Log(c.ctx, *n)
	if err != nil {
		return err
	}
	for _, s := range snaps {
		fmt.Fprintf(c.out, "%s  %s  %-20s %s\n", short(s.ID), s.At.Local().Format("2006-01-02 15:04"), s.Reason, s.Actor)
	}
	return nil
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
	if err := c.open(); err != nil {
		return err
	}
	if len(args) > 2 {
		return errors.New("usage: tigprint diff [from [to]]")
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
	printChanges(c.out, "", changes)
	return nil
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: tigprint restore [-all] <snapshot> [path...]")
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
			return errors.New("name the files to restore, or give -all")
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
	fmt.Fprintf(c.out, "restored %d files from %s (the project before it is snapshot %s)\n", len(paths), short(s.ID), short(before.ID))
	return nil
}

func heading(b *tig.Block) string {
	if b.Level == 0 {
		return b.Path
	}
	return strings.Repeat("#", b.Level) + " " + b.Title
}

func (c *cli) blocks(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: tigprint blocks <file.md>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	// A block's hash includes its path in the project.
	rel := filepath.ToSlash(args[0])
	if c.open() == nil {
		if abs, err := filepath.Abs(args[0]); err == nil {
			if p, err := filepath.Rel(c.r.Root, abs); err == nil && !strings.HasPrefix(p, "..") {
				rel = filepath.ToSlash(p)
			}
		}
	}
	for _, b := range tig.ParseBlocks(rel, data) {
		fmt.Fprintf(c.out, "%s  %s%s  (lines %d-%d)\n", short(b.Hash), strings.Repeat("  ", max(b.Level-1, 0)), heading(&b), b.Start, b.End)
		for _, cmd := range b.Accept {
			fmt.Fprintf(c.out, "              %s$ %s\n", strings.Repeat("  ", b.Level), cmd)
		}
	}
	return nil
}

func (c *cli) analyze() (*tig.Analysis, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	return c.r.Analyze(c.ctx, "")
}

func (c *cli) chain(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: tigprint chain")
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	for _, h := range a.Chain.Order {
		b := a.Chain.Blocks[h]
		fmt.Fprintf(c.out, "%s  %s%s\n", short(h), strings.Repeat("  ", max(b.Level-1, 0)), heading(b))
	}
	return nil
}

func (c *cli) show(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: tigprint show <block|reference>")
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
			fmt.Fprintln(c.out, "  ... (tigprint trail for all)")
			break
		}
		c.printWork(w)
	}
	return nil
}

func (c *cli) search(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: tigprint search <words>")
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	files, err := tig.Files(c.ctx, c.r.Root, c.r.Skip)
	if err != nil {
		return err
	}
	for _, h := range a.Search(strings.Join(args, " "), files) {
		fmt.Fprintf(c.out, "%-5s %s  %s  %s\n", h.Kind, short(h.Ref), h.Path, h.Title)
		if h.Snippet != "" {
			fmt.Fprintf(c.out, "      %s\n", h.Snippet)
		}
	}
	return nil
}

func (c *cli) check(args []string) error {
	fs := c.flags("check")
	asJSON := fs.Bool("json", false, "print the report as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	rep := a.Report
	if *asJSON {
		enc := json.NewEncoder(c.out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			return err
		}
	} else {
		c.printReport(rep)
	}
	if !rep.Closed {
		return errNotClosed
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: tigprint trail [-steps] [-n 20] <block|file>")
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
	if len(args) == 0 {
		return errors.New("usage: tigprint work start|step|finish|status")
	}
	if err := c.open(); err != nil {
		return err
	}
	switch args[0] {
	case "start":
		return c.workStart(args[1:])
	case "step":
		return c.workStep(args[1:])
	case "finish":
		return c.workFinish(args[1:])
	case "status":
		w, err := c.r.Current()
		if err != nil {
			return err
		}
		if w == nil {
			fmt.Fprintln(c.out, "no work in progress")
			return nil
		}
		fmt.Fprintf(c.out, "in progress since %s by %s: %s\n%d steps\n", w.Started.Local().Format("2006-01-02 15:04"), w.Actor, w.Intent, len(w.Steps))
		return nil
	}
	return fmt.Errorf("unknown work command %q", args[0])
}

func (c *cli) commits(args []string) error {
	fs := c.flags("commits")
	n := fs.Int("n", 20, "how many")
	if err := fs.Parse(args); err != nil {
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
	for i, w := range all {
		if i == *n {
			break
		}
		fmt.Fprintf(c.out, "%-6s %s  %-10s %s\n", commits.Short(w.ID), w.At.Local().Format("2006-01-02 15:04"), w.Result, w.Intent)
	}
	return nil
}

// many collects a flag given several times.
type many []string

func (m *many) String() string     { return strings.Join(*m, ",") }
func (m *many) Set(v string) error { *m = append(*m, v); return nil }

func (c *cli) set(args []string) error {
	fs := c.flags("set")
	asJSON := fs.Bool("json", false, "print JSON")
	var in tig.SetInput
	fs.Var((*many)(&in.Sentences), "s", "a sentence, in its words")
	fs.Var((*many)(&in.Commits), "id", "a commit")
	fs.Var((*many)(&in.Files), "f", "a file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	set := a.Relations(c.ctx).Set(in)
	if *asJSON {
		return printJSON(c.out, set)
	}
	for _, s := range set.Sentences {
		fmt.Fprintf(c.out, "sentence %s:%d %s %s\n", s.Path, s.Line, s.Text, strings.Join(s.IDs, " "))
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
	return nil
}

func (c *cli) groups(args []string) error {
	fs := c.flags("groups")
	asJSON := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	groups := a.Relations(c.ctx).Groups()
	if *asJSON {
		return printJSON(c.out, groups)
	}
	for _, g := range groups {
		fmt.Fprintf(c.out, "%-11s %-8s %d sentences, %d commits, %d files\n", g.Kind, g.Key, len(g.Sentences), len(g.Commits), len(g.Files))
	}
	return nil
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (c *cli) commit(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: tigprint commit <id>")
	}
	a, err := c.analyze()
	if err != nil {
		return err
	}
	v, err := a.Commit(args[0])
	if err != nil {
		return err
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

func (c *cli) mark(args []string) error {
	fs := c.flags("mark")
	ids := fs.String("id", "", "the commits, separated by commas")
	doc := fs.String("doc", "", "the document (default: the main document)")
	under := fs.String("under", "", "the section a new sentence goes under")
	inferred := fs.Bool("inferred", false, "it is a guess: for the inferred document unless the main document says it")
	confirm := fs.Bool("confirm", false, "a person says this is what was meant")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return errors.New("usage: tigprint mark [-id a1b2,c3d4] [-doc f.md] [-under title] [-inferred] [-confirm] <sentence>")
	}
	var refs []string
	if *ids != "" {
		refs = strings.Split(*ids, ",")
	}
	in := tig.MarkInput{Path: *doc, Under: *under, IDs: refs, Inferred: *inferred, Actor: c.actor, Via: tig.ViaDirect}
	if *confirm {
		in.State, in.By = tig.IntentConfirmed, tig.ByHuman
	}
	if err := c.open(); err != nil {
		return err
	}
	in.Sentence = strings.Join(fs.Args(), " ")
	m, err := c.r.Mark(c.ctx, in)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "%s:%d  %s  %s\n", m.Path, m.Line, strings.Join(m.IDs, " "), m.Text)
	return nil
}

func (c *cli) fold(args []string) error {
	fs := c.flags("fold")
	intent := fs.String("m", "", "what the commits are, together")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *intent == "" || fs.NArg() < 2 {
		return errors.New("usage: tigprint fold -m intent <id> <id>...")
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
	fmt.Fprintf(c.out, "%s folds %s\n", commits.Short(w.ID), strings.Join(fs.Args(), " "))
	return nil
}

func (c *cli) unmark(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: tigprint unmark <id>...")
	}
	if err := c.open(); err != nil {
		return err
	}
	return c.r.Unmark(c.ctx, args)
}

func (c *cli) sync(_ []string) error {
	if err := c.open(); err != nil {
		return err
	}
	res, err := c.r.Sync(c.ctx, c.actor)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "%d documents rewritten, %d intents touched, %d moved, %d old marks converted\n", len(res.Docs), len(res.Touched), len(res.Moved), res.Migrated)
	for _, i := range res.Touched {
		fmt.Fprintf(c.out, "touched %s: confirm it again\n", i)
	}
	return nil
}

func (c *cli) decide(name string, args []string, do func(tig.DecideInput) (*tig.Intent, error), set func(fs *flag.FlagSet, in *tig.DecideInput)) error {
	fs := c.flags(name)
	var in tig.DecideInput
	fs.StringVar(&in.Reason, "m", "", "why")
	if set != nil {
		set(fs, &in)
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: tigprint %s <intent>", name)
	}
	if err := c.open(); err != nil {
		return err
	}
	in.Intent, in.By, in.Via, in.Actor = fs.Arg(0), tig.ByHuman, tig.ViaDirect, c.actor
	i, err := do(in)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "%s#%d  %s  %s\n", i.Short, i.Revision, i.State, i.Text)
	return nil
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
	var by string
	return c.decide("retire", args, func(in tig.DecideInput) (*tig.Intent, error) {
		if by != "" {
			in.From = []string{by}
		}
		return c.r.RetireIntent(c.ctx, in)
	}, func(fs *flag.FlagSet, _ *tig.DecideInput) {
		fs.StringVar(&by, "by", "", "the intent that took its place")
	})
}

func (c *cli) accept(args []string) error {
	fs := c.flags("accept")
	reject := fs.Bool("reject", false, "record it as rejected")
	reason := fs.String("m", "", "why")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: tigprint accept [-reject] [-m why] <id>")
	}
	if err := c.open(); err != nil {
		return err
	}
	state := tig.WorkAccepted
	if *reject {
		state = tig.WorkRejected
	}
	a, err := c.r.Accept(c.ctx, tig.AcceptInput{Work: fs.Arg(0), State: state, By: tig.ByHuman, Via: tig.ViaDirect, Actor: c.actor, Reason: *reason})
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "%s %s\n", short(a.Work), a.State)
	return nil
}

// showRef prints what a reference means.
func (c *cli) showRef(ref string) error {
	v, err := c.r.Show(c.ctx, ref)
	if err != nil {
		return err
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
			fmt.Fprintf(c.out, "  work   %s %s #%d (%s)\n", w.Short, w.Kind, w.Revision, w.Acceptance)
		}
		if i.Previous != nil {
			fmt.Fprintf(c.out, "  before %s %q", i.Previous.Ref, i.Previous.Text)
			if i.Previous.Reason != "" {
				fmt.Fprintf(c.out, ": %s", i.Previous.Reason)
			}
			fmt.Fprintln(c.out)
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

func (c *cli) baseline(args []string) error {
	fs := c.flags("baseline")
	intent := fs.String("m", "the project as it was before tig", "what it is")
	if err := fs.Parse(args); err != nil {
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
	fmt.Fprintf(c.out, "%s  %s\n", commits.Short(w.ID), w.Intent)
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *intent == "" {
		return errors.New("say what the work is for: -m intent")
	}
	basis, err := c.basis(fs.Args())
	if err != nil {
		return err
	}
	w, err := c.r.StartWork(c.ctx, basis, *intent, c.actor)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.out, "work started (snapshot %s)\n", short(w.Before))
	return nil
}

func (c *cli) workStep(args []string) error {
	fs := c.flags("work step")
	kind := fs.String("kind", "", "edit, command, install, check, decision or note")
	note := fs.String("m", "", "what was done")
	command := fs.String("command", "", "the command that was run")
	exit := fs.Int("exit", 0, "its exit code")
	output := fs.String("output", "", "a file with its output (- for stdin)")
	detail := fs.String("detail", "", "a decision's question and answer, or a longer note")
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch *kind {
	case tig.StepEdit, tig.StepCommand, tig.StepInstall, tig.StepCheck, tig.StepDecision, tig.StepNote:
	default:
		return fmt.Errorf("unknown step kind %q (edit, command, install, check, decision, note)", *kind)
	}
	in := tig.StepInput{Kind: *kind, Note: *note, Command: *command, Detail: *detail, Actor: c.actor}
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "exit" {
			in.Exit = exit
		}
	})
	if *kind == tig.StepCheck && in.Exit == nil {
		return errors.New("a check needs its -exit code")
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch *result {
	case "", tig.ResultPassed, tig.ResultFailed, tig.ResultDone, tig.ResultRolledBack, tig.ResultAbandoned:
	default:
		return fmt.Errorf("unknown result %q", *result)
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
	fmt.Fprintf(c.out, "work %s: %s, %d steps\n", short(w.ID), w.Result, len(w.Steps))
	return nil
}

func (c *cli) mcp(args []string) error {
	fs := c.flags("mcp")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := c.open(); err != nil {
		return err
	}
	s := mcpserver.New(c.r, mcpserver.Options{Version: version})
	return s.Run(c.ctx, &mcp.StdioTransport{})
}

func (c *cli) push(args []string) error {
	fs := c.flags("push")
	token := fs.String("token", os.Getenv("TIG_TOKEN"), "the owner's token on the hub (default $TIG_TOKEN)")
	note := fs.String("m", "", "a note for this publication")
	if err := fs.Parse(args); err != nil {
		return err
	}
	project := os.Getenv("TIG_HUB")
	if fs.NArg() > 0 {
		project = fs.Arg(0)
	}
	if project == "" {
		return errors.New("push needs the project's address on a hub: tigprint push http://host/owner/name (or $TIG_HUB)")
	}
	if err := c.open(); err != nil {
		return err
	}
	res, err := hub.Push(c.ctx, c.r, project, *token, c.actor, *note)
	if len(res.Unsnapped) > 0 {
		fmt.Fprintf(c.errs, "these documents changed since the latest snapshot and are pushed as it has them (run tigprint snap): %s\n",
			strings.Join(res.Unsnapped, ", "))
	}
	if err != nil {
		return err
	}
	if res.UpToDate {
		fmt.Fprintf(c.out, "up to date: %s has these documents and this chain (publication %s)\n", project, short(res.Publication.ID))
		return nil
	}
	fmt.Fprintf(c.out, "pushed publication %s to %s (%d objects sent)\n", short(res.Publication.ID), project, res.Sent)
	return nil
}
