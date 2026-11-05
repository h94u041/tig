package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMain lets this test binary stand in for the installed program.
func TestMain(m *testing.M) {
	if os.Getenv("TIGWX_SETUP_TEST_PROGRAM") != "" {
		fmt.Println("tigwx version stand-in")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func standIn(t *testing.T) []byte {
	t.Helper()
	t.Setenv("TIGWX_SETUP_TEST_PROGRAM", "1")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPathWith(t *testing.T) {
	for _, c := range []struct {
		path, dir, want string
		changed         bool
	}{
		{"", `C:\t\bin`, `C:\t\bin`, true},
		{`C:\a`, `C:\t\bin`, `C:\a;C:\t\bin`, true},
		{`C:\a;`, `C:\t\bin`, `C:\a;C:\t\bin`, true},
		{`%USERPROFILE%\go\bin;C:\a`, `C:\t\bin`, `%USERPROFILE%\go\bin;C:\a;C:\t\bin`, true},
		{`C:\a;c:\T\BIN\;C:\b`, `C:\t\bin`, `C:\a;c:\T\BIN\;C:\b`, false},
		{`C:\a;"C:\t\bin";C:\b`, `C:\t\bin`, `C:\a;"C:\t\bin";C:\b`, false},
	} {
		got, changed := pathWith(c.path, c.dir)
		if got != c.want || changed != c.changed {
			t.Errorf("%q: %q %v", c.path, got, changed)
		}
	}
}

func TestPathWithout(t *testing.T) {
	for _, c := range []struct {
		path, dir, want string
		changed         bool
	}{
		{"", `C:\t\bin`, "", false},
		{`C:\a`, `C:\t\bin`, `C:\a`, false},
		{`C:\t\bin`, `C:\t\bin`, "", true},
		{`C:\a;c:\T\BIN\;%X%\b`, `C:\t\bin`, `C:\a;%X%\b`, true},
		{`C:\t\bin;C:\a;C:\t\bin`, `C:\t\bin`, `C:\a`, true},
		{`C:\t\bin2;C:\t`, `C:\t\bin`, `C:\t\bin2;C:\t`, false},
	} {
		got, changed := pathWithout(c.path, c.dir)
		if got != c.want || changed != c.changed {
			t.Errorf("%q: %q %v", c.path, got, changed)
		}
	}
}

// One that goes in and comes out leaves the value as it was.
func TestPathRoundTrip(t *testing.T) {
	for _, path := range []string{`C:\a`, `C:\a;%X%\b`, `C:\a;;C:\b`} {
		in, _ := pathWith(path, `C:\t\bin`)
		out, _ := pathWithout(in, `C:\t\bin`)
		if out != path {
			t.Errorf("%q → %q → %q", path, in, out)
		}
	}
}

func TestPlaceFileOverAnother(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sub", "file")
	for _, text := range []string{"one", "two"} {
		if err := placeFile(target, strings.NewReader(text), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(target)
		if err != nil || string(got) != text {
			t.Fatalf("%q %v", got, err)
		}
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Fatalf("left behind: %v", entries)
	}
}

func TestSourceTreeHoldsNoProgram(t *testing.T) {
	var out, errs bytes.Buffer
	code := run([]string{"-dir", t.TempDir(), "-yes", "-portable"}, strings.NewReader(""), &out, &errs)
	if code != exitFatal || !strings.Contains(errs.String(), "holds no tigwx") {
		t.Fatalf("%d %s", code, errs.String())
	}
}

func TestFlags(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run([]string{"--version"}, strings.NewReader(""), &out, &errs); code != exitOK || !strings.HasPrefix(out.String(), "tigwx-setup version ") {
		t.Fatalf("%d %s", code, out.String())
	}
	out.Reset()
	if code := run([]string{"--help"}, strings.NewReader(""), &out, &errs); code != exitOK || !strings.Contains(out.String(), "-uninstall") {
		t.Fatalf("%d %s", code, out.String())
	}
	if code := run([]string{"-nosuch"}, strings.NewReader(""), &out, &errs); code != exitUsage {
		t.Fatal(code)
	}
	if code := run([]string{"extra"}, strings.NewReader(""), &out, &errs); code != exitUsage {
		t.Fatal(code)
	}
}

func portable(root string, answer string) (*setup, *bytes.Buffer) {
	var out bytes.Buffer
	yes := answer == ""
	return &setup{in: bufio.NewReader(strings.NewReader(answer)), out: &out, root: root, portable: true, yes: yes}, &out
}

func TestInstallAndUninstallFiles(t *testing.T) {
	data := standIn(t)
	self, _ := os.Executable()
	root := filepath.Join(t.TempDir(), "with space", "tigwx")
	s, out := portable(root, "")
	if code := s.installProgram(data, self, io.Discard); code != exitOK {
		t.Fatalf("%d %s", code, out.String())
	}
	target := filepath.Join(root, "bin", exeName())
	if _, err := os.Stat(target); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Installed: tigwx version stand-in") {
		t.Fatal(out.String())
	}
	// Again, over what is there.
	if code := s.installProgram(data, self, io.Discard); code != exitOK {
		t.Fatal(out.String())
	}

	// It takes only what it put there.
	mine := filepath.Join(root, "bin", "mine.txt")
	if err := os.WriteFile(mine, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, out = portable(root, "")
	if code := s.uninstall(self, io.Discard); code != exitOK {
		t.Fatalf("%d %s", code, out.String())
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("still there")
	}
	if _, err := os.Stat(mine); err != nil {
		t.Fatal("took what was not its own")
	}
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(filepath.Join(root, uninstallerName)); !os.IsNotExist(err) {
			t.Fatal("the uninstaller is still there")
		}
	}

	// Nothing there: it says so.
	s, out = portable(root, "")
	if code := s.uninstall(self, io.Discard); code != exitNo || !strings.Contains(out.String(), "no tigwx") {
		t.Fatalf("%d %s", code, out.String())
	}
}

func TestAskingFirst(t *testing.T) {
	data := standIn(t)
	self, _ := os.Executable()
	for answer, wantCode := range map[string]int{"n\n": exitNo, "no\n": exitNo, "maybe\n": exitNo, "\n": exitOK, "y\n": exitOK, "YES\n": exitOK} {
		root := t.TempDir()
		s, out := portable(root, answer)
		if code := s.installProgram(data, self, io.Discard); code != wantCode {
			t.Fatalf("%q: %d\n%s", answer, code, out.String())
		}
		_, err := os.Stat(filepath.Join(root, "bin", exeName()))
		if (err == nil) != (wantCode == exitOK) {
			t.Fatalf("%q: %v", answer, err)
		}
	}
	// Nobody there to answer is a no.
	root := t.TempDir()
	s := &setup{in: bufio.NewReader(strings.NewReader("")), out: io.Discard, root: root, portable: true}
	if code := s.installProgram(data, self, io.Discard); code != exitNo {
		t.Fatal(code)
	}
	if _, err := os.Stat(filepath.Join(root, "bin")); !os.IsNotExist(err) {
		t.Fatal("it went ahead")
	}
}
