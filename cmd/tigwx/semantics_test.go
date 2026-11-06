package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Under go test stdin may or may not be a terminal; the tests decide.
func init() { atTerminal = func() bool { return false } }

func TestNobodySaidWho(t *testing.T) {
	global(t)
	dir := project(t)
	code, out, errs := in(dir, "status")
	if code != 0 {
		t.Fatal(errs)
	}
	want(t, out, "by unknown (no terminal)")

	// Run by a program, it does not pass for a person: no decision is made for one.
	_, out, _ = in(dir, "show", "-json", marked(t, dir, "Counts up."))
	want(t, out, `"by": "unknown"`, `"state": "proposed"`)

	atTerminal = func() bool { return true }
	defer func() { atTerminal = func() bool { return false } }()
	if code, out, errs = in(dir, "status"); code != 0 {
		t.Fatal(errs)
	}
	want(t, out, "by human (a terminal)")

	// A config says who, whatever is or is not there.
	in(dir, "config", "user.by", "tool")
	atTerminal = func() bool { return false }
	_, out, _ = in(dir, "status")
	want(t, out, "by tool (project config)")
}

// marked marks a sentence and gives its intent's short ID.
func marked(t *testing.T, dir string, args ...string) string {
	t.Helper()
	code, out, errs := in(dir, append([]string{"mark", "-json"}, args...)...)
	if code != 0 {
		t.Fatal(errs)
	}
	var m struct {
		Intent string `json:"intent"`
	}
	if err := json.Unmarshal([]byte(out), &m); err != nil || m.Intent == "" {
		t.Fatalf("%v %s", err, out)
	}
	return m.Intent
}

// The whole of what an intent went through, and what it has to do with others,
// from the command line.
func TestIntentsFromTheCommandLine(t *testing.T) {
	global(t)
	dir := t.TempDir()
	write(t, dir, "README.md", "# Shop\n\nCarts are kept.\n\nPrices come from the table.\n")
	if code, _, errs := in(dir, "init"); code != 0 {
		t.Fatal(errs)
	}
	carts := marked(t, dir, "-by", "human", "-source", "chat of 3 May", "-confirm", "Carts are kept.")
	prices := marked(t, dir, "-by", "human", "Prices come from the table.")
	var (
		code      int
		out, errs string
	)

	// A relation.
	if code, out, errs = in(dir, "relate", "-by", "human", "-kind", "parallel", "-m", "nothing in common", carts, prices); code != 0 {
		t.Fatal(errs)
	}
	want(t, out, "parallel", "confirmed")
	_, out, _ = in(dir, "show", carts)
	want(t, out, "relate parallel "+prices+" (confirmed): nothing in common")
	if code, out, errs = in(dir, "relate", "-by", "ai", "-via", "my-tool", carts, prices); code != 0 {
		t.Fatal(errs)
	}
	want(t, out, "depends_on", "proposed")
	_, out, _ = in(dir, "status")
	want(t, out, "proposed relations: 1")
	if code, _, _ = in(dir, "relate", "-propose", "-reject", carts, prices); code != 129 {
		t.Fatalf("both at once: %d", code)
	}

	// A set, pinned, with what it holds named.
	data, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	write(t, dir, "README.md", strings.Replace(string(data), "Carts are kept.", "Carts are kept a day.", 1))
	if code, out, errs = in(dir, "sync", "-by", "human", "-via", "review", "-m", "legal asked", "-source", "ticket 12"); code != 0 {
		t.Fatal(errs)
	}
	want(t, out, "revised "+carts+": legal asked")
	if code, out, errs = in(dir, "sync", "-m", "nothing changed since"); code != 0 {
		t.Fatal(errs)
	}
	want(t, out, "the reason was not recorded")

	_, out, _ = in(dir, "set", "-i", carts+"#1")
	want(t, out, "pinned   "+carts+"#1 Carts are kept. (it has moved on: #2, confirmed)", "digest   ")
	_, one, _ := in(dir, "set", "-i", carts)
	_, two, _ := in(dir, "set", "-i", carts)
	if one != two || !strings.Contains(one, "digest") {
		t.Fatalf("%s\n%s", one, two)
	}

	// Everything that happened to it.
	code, out, errs = in(dir, "log", carts)
	if code != 0 {
		t.Fatal(errs)
	}
	want(t, out, carts+"#2  confirmed", "#1 proposed", "source  chat of 3 May", "#2 revised", "why     legal asked", "source  ticket 12", "human review")
	_, out, _ = in(dir, "log", "-json", carts)
	var history struct {
		Short string `json:"short"`
		Log   []struct {
			Kind string `json:"kind"`
		} `json:"log"`
	}
	if err := json.Unmarshal([]byte(out), &history); err != nil || history.Short != carts || len(history.Log) < 3 {
		t.Fatalf("%v %s", err, out)
	}
	if code, _, _ = in(dir, "log", "i0000"); code != 128 {
		t.Fatalf("no such intent: %d", code)
	}
	// log without an intent is still the snapshots.
	code, out, _ = in(dir, "log")
	if code != 0 || !strings.Contains(out, "open") {
		t.Fatalf("%d %s", code, out)
	}
}

func TestStatusSaysWhatIsLeftOut(t *testing.T) {
	global(t)
	dir := project(t)
	if _, out, _ := in(dir, "status"); strings.Contains(out, "left out") {
		t.Fatal(out)
	}
	write(t, dir, ".env", "API_KEY=do-not-keep-this\n")
	_, out, _ := in(dir, "status")
	want(t, out, "left out of snapshots because they look like secrets", "  .env\n")
	_, out, _ = in(dir, "status", "-json")
	want(t, out, `"leftOut"`, `".env"`)
	if _, out, _ = in(dir, "status"); strings.Contains(out, "do-not-keep-this") {
		t.Fatal(out)
	}
	in(dir, "snap")
	_, out, _ = in(dir, "search", "do-not-keep-this")
	if strings.Contains(out, ".env") {
		t.Fatal(out)
	}
}

func TestCleanOutput(t *testing.T) {
	for in, want := range map[string]string{
		"plain line\n":            "plain line\n",
		"tab\tand\r\nline ends":   "tab\tand\r\nline ends",
		"esc \x1b[2Jhere":         "esc [2Jhere",
		"bell\x07 and del\x7f.":   "bell and del.",
		"back\rover":              "backover",
		"line end\r":              "line end",
		"c1 \xc2\x9b31m csi":      "c1 31m csi",
		"中文 ÛÝ é keeps its bytes": "中文 ÛÝ é keeps its bytes",
	} {
		var out bytes.Buffer
		n, err := cleanWriter{w: &out}.Write([]byte(in))
		if err != nil || n != len(in) {
			t.Errorf("%q: %d %v", in, n, err)
		}
		if out.String() != want {
			t.Errorf("%q: %q, want %q", in, out.String(), want)
		}
	}
	asJSON := true
	var out bytes.Buffer
	cleanWriter{w: &out, json: &asJSON}.Write([]byte("raw \x1b stays\n"))
	if !strings.Contains(out.String(), "\x1b") {
		t.Fatalf("%q", out.String())
	}
}

// Someone else's document cannot write to the terminal of whoever reads it, and a
// program reading JSON still gets the document as it is.
func TestDocumentsCannotDriveTheTerminal(t *testing.T) {
	global(t)
	dir := t.TempDir()
	write(t, dir, "README.md", "# Counter\n\nCounts \x1b[2Jup.\n")
	if code, _, errs := in(dir, "init"); code != 0 {
		t.Fatal(errs)
	}
	for _, args := range [][]string{{"chain"}, {"search", "Counts"}, {"mark", "-by", "human", "Counts"}, {"status"}} {
		code, out, errs := in(dir, args...)
		if code != 0 {
			t.Fatalf("%v: %d %s", args, code, errs)
		}
		if strings.ContainsRune(out+errs, 0x1b) {
			t.Errorf("%v wrote an escape: %q", args, out)
		}
	}
	code, out, errs := in(dir, "blocks", "README.md")
	if code != 0 {
		t.Fatal(errs)
	}
	if strings.ContainsRune(out, 0x1b) {
		t.Errorf("%q", out)
	}
	_, out, _ = in(dir, "blocks", "-json", "README.md")
	if !strings.Contains(out, `u001b`) {
		t.Errorf("JSON keeps what the document says: %s", out)
	}
}
