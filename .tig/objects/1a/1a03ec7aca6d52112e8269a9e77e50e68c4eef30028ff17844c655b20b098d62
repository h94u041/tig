// Command tigwx-setup installs tigwx, the command-line tool of tig, without any
// other tool: it holds the program and puts it where the system finds it. A
// release is built with scripts/release, which places the program in payload/.
package main

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed payload/*
var payload embed.FS

// version is set when a release is built.
var version = "dev"

// Exit statuses, the same as tigwx's.
const (
	exitOK    = 0
	exitNo    = 1
	exitFatal = 128
	exitUsage = 129
)

const usage = `tigwx-setup installs tigwx.

Usage: tigwx-setup [-dir folder] [-yes] [-portable] [-uninstall] [--version]

  -dir folder   where it goes; the program goes in its bin folder
                (default on Windows: %LOCALAPPDATA%\Programs\tigwx; elsewhere: ~/.local)
  -yes          do not ask, and do not wait for Enter at the end
  -portable     only the files: leave PATH and the list of installed programs alone
  -uninstall    take it away again; only what this program put there goes
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// exeName is the program's file name here.
func exeName() string {
	if runtime.GOOS == "windows" {
		return "tigwx.exe"
	}
	return "tigwx"
}

// uninstallerName is the copy of this program kept for taking it away, on the
// systems that list installed programs and cannot remove a program in use.
const uninstallerName = "uninstall.exe"

// program is the tigwx this setup holds.
func program() ([]byte, error) {
	data, err := payload.ReadFile("payload/" + exeName())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errors.New("this setup holds no tigwx to install: it was built without a release. Get one from a release, or build one with: go run ./scripts/release <version>")
	}
	return data, err
}

type setup struct {
	in       *bufio.Reader
	out      io.Writer
	root     string
	portable bool
	yes      bool
	nobody   bool // asked, and the input had ended
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	g := flag.NewFlagSet("tigwx-setup", flag.ContinueOnError)
	g.SetOutput(io.Discard)
	g.Usage = func() {}
	dir := g.String("dir", "", "where it goes")
	yes := g.Bool("yes", false, "do not ask")
	portable := g.Bool("portable", false, "only the files")
	uninstall := g.Bool("uninstall", false, "take it away")
	showVersion := g.Bool("version", false, "the version")
	if err := g.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, usage)
			return exitOK
		}
		fmt.Fprintf(stderr, "tigwx-setup: %v\n\n%s", err, usage)
		return exitUsage
	}
	if g.NArg() != 0 {
		fmt.Fprint(stderr, usage)
		return exitUsage
	}
	if *showVersion {
		fmt.Fprintf(stdout, "tigwx-setup version %s\n", strings.TrimPrefix(version, "v"))
		return exitOK
	}
	self, _ := os.Executable()
	// The copy kept in the install folder is for one thing, and double-clicking it
	// means that.
	if strings.EqualFold(filepath.Base(self), uninstallerName) {
		*uninstall = true
		if *dir == "" {
			*dir = filepath.Dir(self)
		}
	}
	s := &setup{in: bufio.NewReader(stdin), out: stdout, root: *dir, portable: *portable, yes: *yes}
	if s.root == "" {
		root, err := defaultRoot()
		if err != nil {
			fmt.Fprintln(stderr, "fatal:", err)
			return exitFatal
		}
		s.root = root
	}
	root, err := filepath.Abs(s.root)
	if err != nil {
		fmt.Fprintln(stderr, "fatal:", err)
		return exitFatal
	}
	s.root = root
	var code int
	if *uninstall {
		code = s.uninstall(self, stderr)
	} else {
		code = s.install(self, stderr)
	}
	if !s.yes && !s.nobody {
		fmt.Fprint(stdout, "\nPress Enter to close.")
		s.in.ReadString('\n')
	}
	return code
}

func (s *setup) bin() string { return filepath.Join(s.root, "bin") }

// ask is a yes unless the answer says otherwise. Nobody there to answer is a no.
func (s *setup) ask(question string) bool {
	if s.yes {
		return true
	}
	fmt.Fprintf(s.out, "%s [Y/n] ", question)
	line, err := s.in.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		s.nobody = true
		fmt.Fprintln(s.out)
		return false
	}
	a := strings.ToLower(strings.TrimSpace(line))
	return a == "" || a == "y" || a == "yes"
}

func (s *setup) install(self string, stderr io.Writer) int {
	data, err := program()
	if err != nil {
		fmt.Fprintln(stderr, "fatal:", err)
		return exitFatal
	}
	return s.installProgram(data, self, stderr)
}

func (s *setup) installProgram(data []byte, self string, stderr io.Writer) int {
	shown := strings.TrimPrefix(version, "v")
	fmt.Fprintf(s.out, "Install tigwx %s\n  into  %s\n", shown, s.bin())
	if !s.portable {
		for _, line := range registerPlan(s.bin()) {
			fmt.Fprintf(s.out, "  and   %s\n", line)
		}
	}
	if !s.ask("Go ahead?") {
		fmt.Fprintln(s.out, "Nothing changed. Give -yes when there is nobody to ask.")
		return exitNo
	}
	fatal := func(err error) int {
		fmt.Fprintln(stderr, "fatal:", err)
		return exitFatal
	}
	target := filepath.Join(s.bin(), exeName())
	if err := placeFile(target, bytes.NewReader(data), 0o755); err != nil {
		return fatal(err)
	}
	if runtime.GOOS == "windows" {
		if err := s.keepUninstaller(self); err != nil {
			return fatal(err)
		}
	}
	said, err := exec.Command(target, "--version").Output()
	if err != nil {
		return fatal(fmt.Errorf("%s is there but does not run: %w", target, err))
	}
	if !s.portable {
		if err := register(s.root, s.bin(), target, int64(len(data)), shown); err != nil {
			return fatal(err)
		}
	}
	fmt.Fprintf(s.out, "Installed: %s", said)
	switch {
	case s.portable:
		fmt.Fprintf(s.out, "Run it as %s; PATH was left alone.\n", target)
	case onPath(os.Getenv("PATH"), s.bin()):
		fmt.Fprintln(s.out, "Try it: tigwx --version")
	case runtime.GOOS == "windows":
		fmt.Fprintln(s.out, "Open a new terminal, then try it: tigwx --version")
	default:
		fmt.Fprintf(s.out, "%s is not on your PATH. Add it where your shell keeps PATH:\n  export PATH=\"%s:$PATH\"\n", s.bin(), s.bin())
	}
	return exitOK
}

// keepUninstaller copies this program next to what it installed.
func (s *setup) keepUninstaller(self string) error {
	target := filepath.Join(s.root, uninstallerName)
	if same(self, target) {
		return nil
	}
	src, err := os.Open(self)
	if err != nil {
		return err
	}
	defer src.Close()
	return placeFile(target, src, 0o755)
}

func same(a, b string) bool {
	x, err := os.Stat(a)
	if err != nil {
		return false
	}
	y, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(x, y)
}

func onPath(path, dir string) bool {
	for _, e := range filepath.SplitList(path) {
		if e == dir || (runtime.GOOS == "windows" && sameFolder(e, dir)) {
			return true
		}
	}
	return false
}

// placeFile puts a file there whole. One that is in use cannot be overwritten on
// Windows but can be moved aside, and goes when it can.
func placeFile(target string, data io.Reader, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	fresh := target + ".new"
	f, err := os.OpenFile(fresh, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(fresh)
		return err
	}
	old := target + ".old"
	os.Remove(old)
	if err := os.Rename(target, old); err != nil && !errors.Is(err, fs.ErrNotExist) {
		os.Remove(fresh)
		return err
	}
	if err := os.Rename(fresh, target); err != nil {
		os.Rename(old, target)
		return err
	}
	os.Remove(old)
	return nil
}

func (s *setup) uninstall(self string, stderr io.Writer) int {
	target := filepath.Join(s.bin(), exeName())
	_, statErr := os.Stat(target)
	there := statErr == nil
	fmt.Fprintf(s.out, "Remove tigwx\n  from  %s\n", s.bin())
	if !s.portable {
		for _, line := range unregisterPlan(s.bin()) {
			fmt.Fprintf(s.out, "  and   %s\n", line)
		}
	}
	if !s.ask("Go ahead?") {
		fmt.Fprintln(s.out, "Nothing changed. Give -yes when there is nobody to ask.")
		return exitNo
	}
	// The program first: when it is in use nothing has changed yet.
	for _, f := range []string{target, target + ".old", target + ".new"} {
		if err := os.Remove(f); err != nil && !errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(stderr, "fatal: %v\nClose whatever is running tigwx and try again.\n", err)
			return exitFatal
		}
	}
	if !s.portable {
		if err := unregister(s.root, s.bin()); err != nil {
			fmt.Fprintln(stderr, "fatal:", err)
			return exitFatal
		}
	}
	if runtime.GOOS == "windows" {
		uninstaller := filepath.Join(s.root, uninstallerName)
		for _, f := range []string{uninstaller + ".old", uninstaller + ".new"} {
			os.Remove(f)
		}
		os.Remove(s.bin()) // only when nothing else is in it
		if same(self, uninstaller) {
			if err := removeAfterExit(self, s.root); err != nil {
				fmt.Fprintf(s.out, "%s is still there: %v\n", uninstaller, err)
			}
		} else {
			os.Remove(uninstaller)
			os.Remove(s.root)
		}
	}
	if !there {
		fmt.Fprintf(s.out, "There was no tigwx in %s.\n", s.bin())
		return exitNo
	}
	fmt.Fprintln(s.out, "Removed. Your projects and their .tig folders are untouched, and so is your global config.")
	return exitOK
}
