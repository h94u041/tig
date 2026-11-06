// Command tigwx keeps a project's tig: snapshots of the project, the
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
	"runtime/debug"
	"strings"

	"golang.org/x/term"

	"github.com/h94u041/tig"
)

const fallbackVersion = "0.2.0-dev"

// versionString is the version with no leading v.
func versionString() string {
	v := version
	if v == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
	}
	if v == "" {
		v = fallbackVersion
	}
	return strings.TrimPrefix(v, "v")
}

const usage = `tigwx keeps a project's tig: the footprint of how the project was built
and the chain of its documents, checked and served to agents over MCP.

Usage: tigwx [-C dir] [-actor name] [-session name] [--version] [--help] <command> [arguments]

Every command but mcp takes -json and answers in JSON instead. Flags may go before
or after the other arguments; -- ends them. Decisions and marks take -by (human,
ai, tool, or your own word; default user.by, else human at a terminal and unknown
anywhere else) and -via (your own word for how it came about; default direct), and
tig keeps what it is told: -by is a claim, not a proof.

Exit status: 0 done; 1 the answer is no (check not closed, config key not set,
unknown command); 128 it could not be done; 129 it was asked for wrongly.

Project
  init [-root README.md] [-outside path,...] [-force]    start a tig in this folder; it asks nothing and records
                                                         no commit. -force: the code there is one commit, and
                                                         every line of the documents goes with it
  status                                                 where this project stands; it only reads
  config [-global] <key> [value]                         read or set user.name, user.by, root, inferred, outside
  config [-global] -list | -unset <key>                  every value set, or take one away. user.* are also global
  snap [-reason why]                                     record the project as it is
  log [-n 20] [intent]                                   the snapshots, newest first; or everything that
                                                         happened to an intent, oldest first
  diff [from [to]]                                       what changed (default: the last snapshot and now)
  restore [-all] <snapshot> [path...]                    put files back as they were in a snapshot

Chain
  blocks <file.md>                                       a document's text blocks and their hashes
  chain                                                  every block of the chain
  show <block|reference>                                 one block: its text, details and what was built for it; or
                                                         what a reference means: i7f3a, i7f3a#2, lc3d4e5:5-14, a1b2, ev91c0
  search <words>                                         blocks, works, steps and files containing every word
  check                                                  is the chain closed? (exit status 1 when not)

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
  set [-s sentence] [-i i7f3a[#2]] [-id a1b2] [-f file]...
                                                         what a piece of work needs: sentences, commits, files;
                                                         -i i7f3a#2 also says what #2 said and whether it moved on
  groups                                                 the sentences, commits and files that belong together

Marks
  mark [-id a1b2,c3d4] [-doc f.md] [-under title] [-inferred] [-propose] [-confirm] [-source s] <sentence>
                                                         say what a line or a part of it means, and which commits
                                                         follow it. -inferred: a guess, for the inferred document
                                                         unless the document says it already. -propose: the
                                                         commits only may follow it. -confirm: it is what was meant
  reverse [-min n] [-write]                              which lines seem to be about which files, by what they
                                                         point at, with no model. -write: propose them
  describe -f file... [-doc f.md] [-under title] [-line n] [-m intent] [-why why] [-propose] [-source s] <sentence>
                                                         a line is about files already there; nothing changes
  link [-reject] [-m why] <intent> <id>                  whether this code goes with this intent
  relate [-kind depends_on] [-propose] [-reject] [-m why] <intent> <intent>
                                                         what two intents have to do with each other: depends_on,
                                                         part_of, parallel, or your own word; tig acts on none of it
  sync [-m why] [-source s]                              bring the ledger up to edited lines; with -m the change is a revision, said on purpose, and not a touch; old marks are converted
  confirm [-under title] [-m why] <intent>               it is what was meant; out of the inferred document if there
  reject [-m why] <intent>                               it is not what was meant
  retire [-for i1a2b] [-m why] <intent>                  it no longer stands, or another took its place
  accept [-reject] [-m why] <id>                         record a decision on a commit; tig judges nothing
  fold -m intent <id> <id>...                            fold commits under a new one; none disappear
  unmark <id>...                                         withdraw undone commits from what they followed; the sentences stay
  baseline -m intent                                     one commit for all the project was before tig

Agents
  mcp                                                    serve this tig over MCP on stdin and stdout

Hub
  push [-token t] [-m note] [-insecure] <url>            send the documents and the chain to a hub
                                                         (url https://host/owner/name, or $TIG_HUB; the token
                                                         is $TIG_TOKEN or -token; a token goes over plain http
                                                         only to this machine, unless -insecure)

Help
  help [-a] [command]                                    this, one command, or (-a) every command and those on PATH
  version                                                the version

A block is named by its content hash; any unique prefix of 8 or more characters
works, and a commit by its short ID. The actor is -actor, or $TIG_ACTOR, or
user.name, or "cli". Agents working at once each use their own -session, or
$TIG_SESSION.

Any program on PATH named tigwx-<name> runs as tigwx <name>. Built-in commands
come first. It gets the arguments, the working folder from -C, and $TIGWX, the
path of this program.
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
	rawOut  io.Writer // for a program of someone else's: its output is its own
	rawErr  io.Writer
	dir     string // -C
	actor   string
	session string
	by      string // what -by says when nobody says otherwise
	r       *tig.Repo
	asJSON  bool // -json

	// Where actor and by came from, for status.
	actorFrom, byFrom string
}

// emit answers in JSON when asked to, and otherwise lets text do it.
func (c *cli) emit(v any, text func()) error {
	if c.asJSON {
		return printJSON(c.out, v)
	}
	text()
	return nil
}

// who adds -by and -via. tig keeps whatever it is told: a tool on top of it says
// its own ways of working here.
func (c *cli) who(fs *flag.FlagSet, by, via *string) {
	fs.StringVar(by, "by", c.by, "who says so: human, ai, tool, or your own word")
	fs.StringVar(via, "via", tig.ViaDirect, "how it came about, in your own word")
}

// Exit statuses, the way git uses them.
const (
	exitOK    = 0
	exitNo    = 1   // the answer is no
	exitFatal = 128 // it could not be done
	exitUsage = 129 // it was asked for wrongly
)

// errNo ends a command with exit status 1 and nothing more to say.
var errNo = errors.New("no")

// flagError is a command given a flag it cannot use.
type flagError struct {
	command string
	err     error
}

func (e *flagError) Error() string { return e.err.Error() }

// exit prints what err calls for and says how to end.
func (c *cli) exit(err error) int {
	var fe *flagError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return exitOK
	case errors.Is(err, errNo):
		return exitNo
	case errors.As(err, &fe):
		fmt.Fprintf(c.errs, "tigwx: %v\nSee 'tigwx help %s'.\n", fe.err, fe.command)
		return exitUsage
	case strings.HasPrefix(err.Error(), "usage: "):
		fmt.Fprintln(c.errs, err)
		return exitUsage
	}
	fmt.Fprintf(c.errs, "fatal: %s\n", strings.TrimPrefix(err.Error(), "tig: "))
	return exitFatal
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	g := flag.NewFlagSet("tigwx", flag.ContinueOnError)
	g.SetOutput(io.Discard)
	g.Usage = func() {}
	dir := g.String("C", "", "the project folder")
	actor := g.String("actor", "", "who is acting")
	session := g.String("session", "", "keep this work in progress apart from other agents'")
	var showVersion bool
	g.BoolVar(&showVersion, "version", false, "the version")
	g.BoolVar(&showVersion, "v", false, "the version")
	if err := g.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return exitOK
		}
		fmt.Fprintf(stderr, "tigwx: %v\nSee 'tigwx --help'.\n", err)
		return exitUsage
	}
	if showVersion {
		fmt.Fprintf(stdout, "tigwx version %s\n", versionString())
		return exitOK
	}
	if g.NArg() == 0 {
		fmt.Fprint(stdout, usage)
		return exitNo
	}
	c := &cli{ctx: ctx, rawOut: stdout, rawErr: stderr, dir: *dir, actor: *actor, session: *session}
	c.out, c.errs = cleanWriter{w: stdout, json: &c.asJSON}, cleanWriter{w: stderr}
	if c.session == "" {
		c.session = os.Getenv("TIG_SESSION")
	}
	if c.dir != "" {
		if fi, err := os.Stat(c.dir); err != nil || !fi.IsDir() {
			return c.exit(fmt.Errorf("cannot change to '%s': no such folder", c.dir))
		}
	}
	name, rest := g.Arg(0), g.Args()[1:]
	if cmd, ok := c.commands()[name]; ok {
		if !saysNothingOfWho[name] {
			if err := c.identify(); err != nil {
				return c.exit(err)
			}
		}
		return c.exit(cmd(rest))
	}
	if path, ok := external(name); ok {
		return c.runExternal(path, rest)
	}
	return c.notACommand(name)
}

// saysNothingOfWho is the commands that run whatever the config holds: they are
// how you find out what is wrong with it, or fix it.
var saysNothingOfWho = map[string]bool{"help": true, "version": true, "config": true}

func (c *cli) commands() map[string]func([]string) error {
	return map[string]func([]string) error{
		"init": c.init, "status": c.status, "config": c.config, "snap": c.snap, "log": c.log, "diff": c.diff, "restore": c.restore,
		"blocks": c.blocks, "chain": c.chain, "show": c.show, "search": c.search, "check": c.check,
		"trail": c.trail, "work": c.work, "mcp": c.mcp, "push": c.push,
		"commits": c.commits, "commit": c.commit, "mark": c.mark, "fold": c.fold, "unmark": c.unmark, "baseline": c.baseline,
		"sync": c.sync, "confirm": c.confirm, "reject": c.reject, "retire": c.retire, "accept": c.accept,
		"reverse": c.reverse, "describe": c.describe, "link": c.link, "relate": c.relate,
		"set": c.set, "groups": c.groups, "help": c.help, "version": c.version,
	}
}

// identify works out who is acting: what was said here, then the project's
// config, then the global one. Outside a project there is only the global one.
func (c *cli) identify() error {
	global, err := tig.ReadGlobalConfig()
	if err != nil {
		return err
	}
	var project tig.Config
	root, err := c.find()
	if err != nil {
		return err
	}
	if root != "" {
		if project, err = tig.Open(root).Config(); err != nil {
			return err
		}
	}
	c.actor, c.actorFrom = first(
		[2]string{c.actor, "-actor"},
		[2]string{os.Getenv("TIG_ACTOR"), "$TIG_ACTOR"},
		[2]string{project.UserName, "project config"},
		[2]string{global.UserName, "global config"},
		[2]string{"cli", "default"},
	)
	c.by, c.byFrom = first(
		[2]string{project.UserBy, "project config"},
		[2]string{global.UserBy, "global config"},
		anyoneThere(),
	)
	return nil
}

// anyoneThere is who to record when nobody said. A person at a terminal typed it.
// A program running tigwx may be anything, and saying "human" for it would make its
// decisions look like a person's.
func anyoneThere() [2]string {
	if atTerminal() {
		return [2]string{tig.ByHuman, "a terminal"}
	}
	return [2]string{tig.ByUnknown, "no terminal"}
}

var atTerminal = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

// first is the first value that holds something, and where it came from.
func first(values ...[2]string) (value, from string) {
	for _, v := range values {
		if v[0] != "" {
			return v[0], v[1]
		}
	}
	return "", ""
}

func (c *cli) flags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	if name != "mcp" {
		fs.BoolVar(&c.asJSON, "json", false, "answer in JSON")
	}
	return fs
}

// parse reads a command's flags from anywhere among its arguments, the way git
// does; -- ends them.
func (c *cli) parse(fs *flag.FlagSet, args []string) error {
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			rest = append(rest, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			continue
		}
		f := fs.Lookup(name)
		if f == nil {
			continue // fs.Parse says what is wrong with it
		}
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			continue
		}
		if i+1 == len(args) {
			return &flagError{fs.Name(), fmt.Errorf("flag needs an argument: %s", a)}
		}
		i++
		flags = append(flags, args[i])
	}
	err := fs.Parse(append(append(flags, "--"), rest...))
	if errors.Is(err, flag.ErrHelp) {
		c.commandHelp(fs)
		return err
	}
	if err != nil {
		return &flagError{fs.Name(), err}
	}
	return nil
}

// find is the project: the nearest folder at or above the start that has a .tig.
// None: "".
func (c *cli) find() (string, error) {
	start, err := c.start()
	if err != nil {
		return "", err
	}
	for d := start; ; d = filepath.Dir(d) {
		if fi, err := os.Stat(filepath.Join(d, ".tig")); err == nil && fi.IsDir() {
			return d, nil
		}
		if filepath.Dir(d) == d {
			return "", nil
		}
	}
}

// open finds the project, or says there is none.
func (c *cli) open() error {
	root, err := c.find()
	if err != nil {
		return err
	}
	if root == "" {
		return errors.New("not a tig repository (or any of the parent directories): .tig\nhint: start one here with 'tigwx init'")
	}
	c.r = tig.Open(root)
	c.r.Session = c.session
	return nil
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

func (c *cli) analyze() (*tig.Analysis, error) {
	if err := c.open(); err != nil {
		return nil, err
	}
	return c.r.Analyze(c.ctx, "")
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
