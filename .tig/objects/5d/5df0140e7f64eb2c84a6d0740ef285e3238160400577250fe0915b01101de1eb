package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, rel, text string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(full), 0o755)
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// tp runs tigprint in dir and gives its exit status and output.
func tp(t *testing.T, dir string, args ...string) (int, string) {
	t.Helper()
	var out, errs bytes.Buffer
	code := run(context.Background(), append([]string{"-C", dir, "-actor", "tester"}, args...), &out, &errs)
	return code, out.String() + errs.String()
}

func want(t *testing.T, got string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(got, p) {
			t.Fatalf("no %q in:\n%s", p, got)
		}
	}
}

func TestCLI(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "README.md", "# Counter\n\nCounts up. Code: [count](src/count.js).\n\n## Acceptance\n\n- `echo counted`\n")
	write(t, dir, "src/count.js", "n++\n")

	if code, out := tp(t, dir, "chain"); code != 2 || !strings.Contains(out, "tigprint init") {
		t.Fatalf("no tig yet: %d %s", code, out)
	}
	code, out := tp(t, dir, "init")
	if code != 0 {
		t.Fatal(out)
	}
	want(t, out, "tig started", "main document README.md, verify loose")
	if attrs, _ := os.ReadFile(filepath.Join(dir, ".tig", ".gitattributes")); !strings.Contains(string(attrs), "* -text") {
		t.Fatalf("Git must not change the objects' line endings: %q", attrs)
	}

	_, out = tp(t, dir, "chain")
	want(t, out, "# Counter  [direction]", "## Acceptance")
	_, out = tp(t, dir, "blocks", filepath.Join(dir, "README.md"))
	want(t, out, "# Counter", "$ echo counted")

	if code, out = tp(t, dir, "check"); code != 0 {
		t.Fatalf("closed, nothing passed yet: %d\n%s", code, out)
	}
	want(t, out, "closed: yes", "never   echo counted")

	write(t, dir, "src/stray.js", "x\n")
	if code, out = tp(t, dir, "check"); code != 1 {
		t.Fatalf("a file no design reaches: %d\n%s", code, out)
	}
	want(t, out, "closed: no", "no design reaches:\n  src/stray.js")
	os.Remove(filepath.Join(dir, "src/stray.js"))

	if code, out = tp(t, dir, "check", "-run-all"); code != 0 {
		t.Fatalf("run all: %d\n%s", code, out)
	}
	want(t, out, "passed  echo counted", "guidelines 1", "by tig")

	// A work by hand.
	_, out = tp(t, dir, "chain")
	counter := strings.Fields(out)[2] // the first line is README.md itself
	if code, out = tp(t, dir, "work", "start", "-m", "count by two", counter); code != 0 {
		t.Fatal(out)
	}
	write(t, dir, "src/count.js", "n += 2\n")
	_, out = tp(t, dir, "work", "step", "-kind", "edit", "-m", "by two")
	want(t, out, "(edit)", "M src/count.js")
	if code, out = tp(t, dir, "work", "step", "-kind", "check", "-command", "echo counted"); code != 2 {
		t.Fatalf("a check needs its exit code: %s", out)
	}
	tp(t, dir, "work", "step", "-kind", "check", "-command", "echo counted", "-exit", "0")
	_, out = tp(t, dir, "work", "finish")
	want(t, out, "passed, 2 steps")

	_, out = tp(t, dir, "trail", "-steps", "src/count.js")
	want(t, out, "count by two", "edit     by two", "M src/count.js")
	_, out = tp(t, dir, "show", counter)
	want(t, out, "status: guideline", "links to: src/count.js", "count by two")
	_, out = tp(t, dir, "search", "counts")
	want(t, out, "block", "README.md")

	_, out = tp(t, dir, "log")
	// The opening snapshot and the one after the work; snapshots that would
	// change nothing are not taken.
	if n := strings.Count(out, "\n"); n != 2 {
		t.Fatalf("log: %s", out)
	}
	first := strings.Fields(out[strings.LastIndex(strings.TrimSpace(out), "\n")+1:])[0]
	_, out = tp(t, dir, "diff", first)
	want(t, out, "M src/count.js")
	if code, _ := tp(t, dir, "restore", first); code != 2 {
		t.Fatal("restore wants files or -all")
	}
	_, out = tp(t, dir, "restore", first, "src/count.js")
	want(t, out, "restored 1 files")
	if data, _ := os.ReadFile(filepath.Join(dir, "src/count.js")); string(data) != "n++\n" {
		t.Fatalf("restored: %q", data)
	}
	if code, out = tp(t, dir, "restore", first, "../outside"); code != 2 {
		t.Fatalf("outside the project: %s", out)
	}
}
