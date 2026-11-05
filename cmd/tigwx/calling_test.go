package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestMain keeps the tests off the real global config, and lets this test binary
// stand in for a program on PATH: copied to tigwx-<name>, it says what it was given.
func TestMain(m *testing.M) {
	if os.Getenv("TIGWX_TEST_EXTERNAL") != "" {
		wd, _ := os.Getwd()
		fmt.Printf("args=%s\nwd=%s\ntigwx=%s\nactor=%s\nsession=%s\n",
			strings.Join(os.Args[1:], ","), wd, os.Getenv("TIGWX"), os.Getenv("TIG_ACTOR"), os.Getenv("TIG_SESSION"))
		code, _ := strconv.Atoi(os.Getenv("TIGWX_TEST_EXIT"))
		os.Exit(code)
	}
	dir, err := os.MkdirTemp("", "tigwx-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("TIG_CONFIG_GLOBAL", filepath.Join(dir, "global"))
	os.Unsetenv("TIG_ACTOR")
	os.Unsetenv("TIG_SESSION")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// call runs tigwx as a person would, with stdout and stderr kept apart.
func call(args ...string) (code int, stdout, stderr string) {
	var out, errs bytes.Buffer
	code = run(context.Background(), args, &out, &errs)
	return code, out.String(), errs.String()
}

// in runs tigwx in dir.
func in(dir string, args ...string) (int, string, string) {
	return call(append([]string{"-C", dir}, args...)...)
}

func project(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, "README.md", "# Counter\n\nCounts up.\n")
	if code, _, errs := in(dir, "init"); code != 0 {
		t.Fatal(errs)
	}
	return dir
}

func global(t *testing.T) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "tigconfig")
	t.Setenv("TIG_CONFIG_GLOBAL", file)
	return file
}

func TestVersion(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"-version"}, {"-v"}, {"version"}} {
		code, out, _ := call(args...)
		if code != 0 || !strings.HasPrefix(out, "tigwx version ") {
			t.Fatalf("%v: %d %q", args, code, out)
		}
	}
	code, out, _ := call("version", "-json")
	var v map[string]string
	if code != 0 || json.Unmarshal([]byte(out), &v) != nil || v["version"] == "" {
		t.Fatalf("%d %q", code, out)
	}
}

func TestVersionHasNoLeadingV(t *testing.T) {
	old := version
	defer func() { version = old }()
	version = "v1.2.3"
	if got := versionString(); got != "1.2.3" {
		t.Fatal(got)
	}
}

// Help is an answer: it goes to stdout, ends well, and needs no project.
func TestHelp(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"--help"}, {"-h"}, {"help"}} {
		code, out, errs := in(dir, args...)
		if code != 0 || !strings.Contains(out, "Usage: tigwx") || errs != "" {
			t.Fatalf("%v: %d\n%s\n%s", args, code, out, errs)
		}
	}
	for args, parts := range map[string][]string{
		"help log":        {"usage: tigwx log", "-n"},
		"log -h":          {"usage: tigwx log", "-n"},
		"log --help":      {"usage: tigwx log", "-n"},
		"help status":     {"usage: tigwx status", "-json"},
		"help work":       {"tigwx work start", "tigwx work finish"},
		"work -h":         {"tigwx work start", "tigwx work finish"},
		"help work start": {"usage: tigwx work start", "-m"},
		"work step -h":    {"usage: tigwx work step", "-kind"},
		"help init":       {"usage: tigwx init", "-force", "every line of the documents"},
		"help config":     {"tigwx config [-global] <key> [value]", "-unset"},
	} {
		code, out, errs := in(dir, strings.Fields(args)...)
		if code != 0 || errs != "" {
			t.Fatalf("%s: %d\n%s\n%s", args, code, out, errs)
		}
		want(t, out, parts...)
	}
	code, _, errs := in(dir, "help", "nosuch")
	if code != 1 || !strings.Contains(errs, "no help for 'nosuch'") {
		t.Fatalf("%d %s", code, errs)
	}
	code, out, _ := call()
	if code != 1 || !strings.Contains(out, "Usage: tigwx") {
		t.Fatalf("no arguments: %d %s", code, out)
	}
}

func TestHelpLists(t *testing.T) {
	code, out, _ := call("help", "-a")
	if code != 0 {
		t.Fatal(out)
	}
	want(t, out, "  status\n", "  config\n", "  check\n")
	code, out, _ = call("help", "-json")
	var v struct {
		Commands []string `json:"commands"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &v) != nil || len(v.Commands) < 20 {
		t.Fatalf("%d %s", code, out)
	}
}

func TestUnknownCommandIsOneLine(t *testing.T) {
	code, out, errs := call("nosuch")
	if code != 1 || out != "" || errs != "tigwx: 'nosuch' is not a tigwx command. See 'tigwx --help'.\n" {
		t.Fatalf("%d %q %q", code, out, errs)
	}
	code, _, errs = call("chekc")
	if code != 1 {
		t.Fatal(code)
	}
	want(t, errs, "'chekc' is not a tigwx command", "The most similar command is\n\tcheck\n")
}

func TestExitStatus(t *testing.T) {
	none := t.TempDir()
	code, _, errs := in(none, "chain")
	if code != 128 || !strings.HasPrefix(errs, "fatal: not a tig repository") || !strings.Contains(errs, "tigwx init") {
		t.Fatalf("no project: %d %s", code, errs)
	}
	dir := project(t)
	code, _, errs = in(dir, "commit")
	if code != 129 || !strings.HasPrefix(errs, "usage: tigwx commit") {
		t.Fatalf("usage: %d %s", code, errs)
	}
	code, _, errs = in(dir, "log", "-nosuch")
	if code != 129 || !strings.Contains(errs, "flag provided but not defined: -nosuch") || !strings.Contains(errs, "See 'tigwx help log'.") {
		t.Fatalf("flag: %d %s", code, errs)
	}
	code, _, errs = in(dir, "work", "start", "-m")
	if code != 129 || !strings.Contains(errs, "flag needs an argument: -m") {
		t.Fatalf("flag with no value: %d %s", code, errs)
	}
	code, _, errs = in(dir, "show", "nosuchthing")
	if code != 128 || !strings.HasPrefix(errs, "fatal: ") {
		t.Fatalf("cannot be done: %d %s", code, errs)
	}
	code, _, errs = call("-C", filepath.Join(dir, "nosuch"), "status")
	if code != 128 || !strings.Contains(errs, "cannot change to") {
		t.Fatalf("no such folder, inside a project: %d %s", code, errs)
	}
	code, _, errs = call("-nosuch")
	if code != 129 || !strings.Contains(errs, "See 'tigwx --help'.") {
		t.Fatalf("global flag: %d %s", code, errs)
	}
	write(t, dir, "README.md", "# Counter\n\nCounts up. See [gone](gone.md).\n")
	if code, _, errs = in(dir, "check"); code != 1 || errs != "" {
		t.Fatalf("not closed: %d %s", code, errs)
	}
}

func TestFlagsGoAnywhere(t *testing.T) {
	dir := project(t)
	write(t, dir, "README.md", "# Counter\n\nCounts up, -json in words.\n")
	code, out, errs := in(dir, "search", "Counts", "-json")
	if code != 0 || !json.Valid([]byte(out)) {
		t.Fatalf("after: %d %s %s", code, out, errs)
	}
	code, out2, _ := in(dir, "search", "-json", "Counts")
	if code != 0 || out != out2 {
		t.Fatalf("before: %d %s", code, out2)
	}
	// -- ends the flags: this is a search for the word.
	code, out, _ = in(dir, "search", "--", "-json")
	if code != 0 || json.Valid([]byte(out)) || !strings.Contains(out, "README.md") {
		t.Fatalf("-- : %d %q", code, out)
	}
	// A flag's value is its own, even when it looks like a flag.
	if code, _, errs = in(dir, "work", "start", "-m", "-json is a word here"); code != 0 {
		t.Fatal(errs)
	}
	_, out, _ = in(dir, "work", "status")
	want(t, out, "-json is a word here")
	// Both forms, and = too.
	if code, _, errs = in(dir, "log", "--n=1", "--json"); code != 0 {
		t.Fatal(errs)
	}
}

func TestStatus(t *testing.T) {
	dir := project(t)
	code, out, errs := in(dir, "status")
	if code != 0 {
		t.Fatal(errs)
	}
	want(t, out, "main document: README.md", "acting as: cli (default), by human (default)", "last snapshot: ", "chain closed: yes", "proposed intents: 0")

	// It only reads.
	_, before, _ := in(dir, "log", "-json")
	heads, _ := os.ReadFile(filepath.Join(dir, ".tig", "HEAD"))
	write(t, dir, "README.md", "# Counter\n\nCounts up and down.\n")
	if code, out, errs = in(dir, "status"); code != 0 {
		t.Fatal(errs)
	}
	want(t, out, "changed since the last snapshot:", "M README.md")
	_, after, _ := in(dir, "log", "-json")
	again, _ := os.ReadFile(filepath.Join(dir, ".tig", "HEAD"))
	if before != after || !bytes.Equal(heads, again) {
		t.Fatal("status changed what the project holds")
	}

	if code, _, errs = in(dir, "work", "start", "-m", "count down"); code != 0 {
		t.Fatal(errs)
	}
	if code, _, errs = in(dir, "mark", "-propose", "-inferred", "a guess about counting"); code != 0 {
		t.Fatal(errs)
	}
	code, out, errs = in(dir, "status", "-json")
	if code != 0 {
		t.Fatal(errs)
	}
	var s statusOut
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if s.Main != "README.md" || s.Snapshot == nil || s.Work == nil || s.Work.Intent != "count down" || s.ActorFrom != "default" || len(s.Changes) == 0 {
		t.Fatalf("%+v", s)
	}
	if s.ProposedIntents != 1 {
		t.Fatalf("proposed: %+v", s)
	}
	if code, _, errs = in(t.TempDir(), "status"); code != 128 {
		t.Fatalf("no project: %d %s", code, errs)
	}
	if code, _, errs = in(dir, "status", "extra"); code != 129 {
		t.Fatalf("extra: %d %s", code, errs)
	}
}

func TestConfig(t *testing.T) {
	file := global(t)
	dir := project(t)

	// Nothing set: exit 1, nothing said.
	if code, out, errs := in(dir, "config", "user.name"); code != 1 || out != "" || errs != "" {
		t.Fatalf("%d %q %q", code, out, errs)
	}
	// Global works anywhere, and is a record like any other.
	if code, _, errs := in(t.TempDir(), "config", "-global", "user.name", "Pat Lee"); code != 0 {
		t.Fatal(errs)
	}
	raw, err := os.ReadFile(file)
	if err != nil || string(raw) != "tig-config 1\nuser.name Pat Lee\n" {
		t.Fatalf("%v %q", err, raw)
	}
	if _, out, _ := in(dir, "config", "user.name"); out != "Pat Lee\n" {
		t.Fatal(out)
	}
	// The project's comes first, and leaves the global one alone.
	if code, _, errs := in(dir, "config", "user.name", "Robin"); code != 0 {
		t.Fatal(errs)
	}
	if _, out, _ := in(dir, "config", "user.name"); out != "Robin\n" {
		t.Fatal(out)
	}
	if _, out, _ := in(dir, "config", "-global", "user.name"); out != "Pat Lee\n" {
		t.Fatal(out)
	}
	if code, _, errs := in(dir, "config", "-unset", "user.name"); code != 0 {
		t.Fatal(errs)
	}
	if _, out, _ := in(dir, "config", "user.name"); out != "Pat Lee\n" {
		t.Fatal(out)
	}
	// Project keys.
	if _, out, _ := in(dir, "config", "root"); out != "README.md\n" {
		t.Fatal(out)
	}
	if code, _, errs := in(dir, "config", "outside", "i18n, docs/old"); code != 0 {
		t.Fatal(errs)
	}
	if _, out, _ := in(dir, "config", "outside"); out != "i18n,docs/old\n" {
		t.Fatal(out)
	}
	if code, _, errs := in(dir, "config", "-global", "root", "x.md"); code != 129 || !strings.Contains(errs, "belongs to a project") {
		t.Fatalf("%d %s", code, errs)
	}
	if code, _, errs := in(t.TempDir(), "config", "root", "x.md"); code != 128 || !strings.Contains(errs, "not a tig repository") {
		t.Fatalf("%d %s", code, errs)
	}
	// What cannot be read back the same is refused.
	for _, args := range [][]string{{"config", "nosuch"}, {"config", "user.name", "two\nlines"}, {"config", "user.name", ""}, {"config"}, {"config", "-unset"}, {"config", "-list", "root"}} {
		if code, _, _ := in(dir, args...); code != 129 {
			t.Fatalf("%v: %d", args, code)
		}
	}
	code, out, errs := in(dir, "config", "-list")
	if code != 0 || out != "user.name=Pat Lee\nroot=README.md\noutside=i18n,docs/old\n" {
		t.Fatalf("%d %q %s", code, out, errs)
	}
	var values map[string]string
	if code, out, _ = in(dir, "config", "-list", "-json"); code != 0 || json.Unmarshal([]byte(out), &values) != nil || values["user.name"] != "Pat Lee" {
		t.Fatalf("%d %q", code, out)
	}
	// It holds through the commands that write it back.
	if _, out, _ = in(dir, "config", "root"); out != "README.md\n" {
		t.Fatal(out)
	}
	// A broken global config is named, not skipped; config itself still runs.
	if err := os.WriteFile(file, []byte("nonsense\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs = in(dir, "status"); code != 128 || !strings.Contains(errs, file) {
		t.Fatalf("%d %s", code, errs)
	}
	if code, _, _ = in(dir, "version"); code != 0 {
		t.Fatal(code)
	}
}

// Who is acting: what was said here, then the project, then the global file.
func TestIdentity(t *testing.T) {
	global(t)
	dir := project(t)
	actor := func(args ...string) (string, string, string, string) {
		t.Helper()
		code, out, errs := call(append([]string{"-C", dir}, append(args, "status", "-json")...)...)
		if code != 0 {
			t.Fatal(errs)
		}
		var s statusOut
		if err := json.Unmarshal([]byte(out), &s); err != nil {
			t.Fatal(err)
		}
		return s.Actor, s.ActorFrom, s.By, s.ByFrom
	}
	if a, f, b, bf := actor(); a != "cli" || f != "default" || b != "human" || bf != "default" {
		t.Fatal(a, f, b, bf)
	}
	in(dir, "config", "-global", "user.name", "Pat")
	in(dir, "config", "-global", "user.by", "ai")
	if a, f, b, bf := actor(); a != "Pat" || f != "global config" || b != "ai" || bf != "global config" {
		t.Fatal(a, f, b, bf)
	}
	in(dir, "config", "user.name", "Robin")
	in(dir, "config", "user.by", "tool")
	if a, f, b, bf := actor(); a != "Robin" || f != "project config" || b != "tool" || bf != "project config" {
		t.Fatal(a, f, b, bf)
	}
	t.Setenv("TIG_ACTOR", "from-env")
	if a, f, _, _ := actor(); a != "from-env" || f != "$TIG_ACTOR" {
		t.Fatal(a, f)
	}
	if a, f, _, _ := actor("-actor", "said"); a != "said" || f != "-actor" {
		t.Fatal(a, f)
	}

	// It reaches what gets recorded: the actor on a snapshot and -by on a decision.
	os.Unsetenv("TIG_ACTOR")
	write(t, dir, "README.md", "# Counter\n\nCounts up and down.\n")
	if code, _, errs := in(dir, "snap"); code != 0 {
		t.Fatal(errs)
	}
	_, out, _ := in(dir, "log", "-json")
	if !strings.Contains(out, `"actor": "Robin"`) {
		t.Fatal(out)
	}
	if code, _, errs := in(dir, "mark", "Counts up and down."); code != 0 {
		t.Fatal(errs)
	}
	_, out, _ = in(dir, "set", "-s", "Counts up and down.", "-json")
	if !strings.Contains(out, `"by": "tool"`) {
		t.Fatal(out)
	}
	if code, _, errs := in(dir, "mark", "-by", "human", "Counter"); code != 0 {
		t.Fatal(errs)
	}
	_, out, _ = in(dir, "set", "-s", "Counter", "-json")
	if !strings.Contains(out, `"by": "human"`) {
		t.Fatal(out)
	}
}

func TestSessionFromTheEnvironment(t *testing.T) {
	dir := project(t)
	t.Setenv("TIG_SESSION", "a")
	if code, _, errs := in(dir, "work", "start", "-m", "the first"); code != 0 {
		t.Fatal(errs)
	}
	t.Setenv("TIG_SESSION", "b")
	if _, out, _ := in(dir, "work", "status"); !strings.Contains(out, "no work in progress") {
		t.Fatal(out)
	}
	if _, out, _ := call("-C", dir, "-session", "a", "work", "status"); !strings.Contains(out, "the first") {
		t.Fatal(out)
	}
}

// hello puts this test binary on PATH as tigwx-<name>.
func hello(t *testing.T, name string) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	file := "tigwx-" + name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	src, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.OpenFile(filepath.Join(bin, file), os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatal(err)
	}
	dst.Close()
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TIGWX_TEST_EXTERNAL", "1")
	return bin
}

func TestExternalCommand(t *testing.T) {
	hello(t, "hello")
	dir := t.TempDir()
	code, out, errs := call("-C", dir, "-actor", "someone", "-session", "s1", "hello", "one", "-two", "--", "three")
	if code != 0 {
		t.Fatalf("%d %s %s", code, out, errs)
	}
	want(t, out, "args=one,-two,--,three", "actor=someone", "session=s1", "tigwx=")
	realDir, _ := filepath.EvalSymlinks(dir)
	wd := ""
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(line, "wd="); ok {
			wd = strings.TrimSpace(v)
		}
	}
	if got, _ := filepath.EvalSymlinks(wd); !strings.EqualFold(got, realDir) {
		t.Fatalf("wd %q, wanted %q", got, realDir)
	}

	// Its exit status is the answer.
	t.Setenv("TIGWX_TEST_EXIT", "7")
	if code, _, _ = call("hello"); code != 7 {
		t.Fatal(code)
	}
	t.Setenv("TIGWX_TEST_EXIT", "0")

	// It is on the lists, and help asks it.
	_, out, _ = call("help", "-a")
	want(t, out, "commands on PATH:\n  hello\n")
	_, out, _ = call("help", "-json")
	want(t, out, `"external"`, `"hello"`)
	_, out, _ = call("help", "hello")
	want(t, out, "args=--help")

	// A typo finds it too.
	if code, _, errs = call("helo"); code != 1 || !strings.Contains(errs, "\thello\n") {
		t.Fatalf("%d %s", code, errs)
	}
}

func TestBuiltInsComeFirst(t *testing.T) {
	hello(t, "version")
	code, out, _ := call("version")
	if code != 0 || !strings.HasPrefix(out, "tigwx version ") {
		t.Fatal(out)
	}
	if _, out, _ = call("help", "-a"); strings.Contains(out, "commands on PATH") {
		t.Fatal(out)
	}
}

func TestExternalNames(t *testing.T) {
	bin := hello(t, "good")
	for _, name := range []string{"..", "../x", "a/b", `a\b`, "UPPER", "-x", "a.b", "a b", ""} {
		if _, ok := external(name); ok {
			t.Fatalf("%q should not run", name)
		}
	}
	if _, ok := external("good"); !ok {
		t.Fatal("not found")
	}

	// Being in the current folder is not being on PATH.
	t.Setenv("PATH", "."+string(os.PathListSeparator)+t.TempDir())
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(bin); err != nil {
		t.Fatal(err)
	}
	if _, ok := external("good"); ok {
		t.Fatal("ran what PATH only reaches through the current folder")
	}
}

func TestEditDistance(t *testing.T) {
	for _, c := range []struct {
		a, b string
		n    int
	}{{"check", "check", 0}, {"chekc", "check", 2}, {"sta", "status", 3}, {"", "abc", 3}} {
		if got := editDistance(c.a, c.b); got != c.n {
			t.Fatalf("%q %q: %d", c.a, c.b, got)
		}
	}
}
