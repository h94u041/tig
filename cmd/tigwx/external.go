package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
)

// externalName is what can follow tigwx- in the name of a program on PATH.
var externalName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// external finds tigwx-<name> on PATH. A program in the current folder that PATH
// only reaches by being relative does not count: being in a folder is not
// asking for it to run.
func external(name string) (string, bool) {
	if !externalName.MatchString(name) {
		return "", false
	}
	path, err := exec.LookPath("tigwx-" + name)
	if err != nil {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	return abs, true
}

// externals lists the commands on PATH.
func externals() []string {
	seen := map[string]bool{}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name, ok := strings.CutPrefix(e.Name(), "tigwx-")
			if !ok {
				continue
			}
			if runtime.GOOS == "windows" {
				name = strings.TrimSuffix(name, filepath.Ext(name))
			}
			if _, builtin := (&cli{}).commands()[name]; builtin {
				continue
			}
			if _, ok := external(name); ok {
				seen[name] = true
			}
		}
	}
	found := make([]string, 0, len(seen))
	for name := range seen {
		found = append(found, name)
	}
	slices.Sort(found)
	return found
}

// runExternal runs a program from PATH as a command of tigwx and ends the way it
// ended.
func (c *cli) runExternal(path string, args []string) int {
	cmd := exec.CommandContext(c.ctx, path, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, c.rawOut, c.rawErr
	cmd.Dir = c.dir
	cmd.Env = os.Environ()
	if self, err := os.Executable(); err == nil {
		cmd.Env = append(cmd.Env, "TIGWX="+self)
	}
	if c.actor != "" {
		cmd.Env = append(cmd.Env, "TIG_ACTOR="+c.actor)
	}
	if c.session != "" {
		cmd.Env = append(cmd.Env, "TIG_SESSION="+c.session)
	}
	err := cmd.Run()
	var failed *exec.ExitError
	if errors.As(err, &failed) {
		if code := failed.ExitCode(); code >= 0 {
			return code
		}
		return exitFatal
	}
	if err != nil {
		return c.exit(err)
	}
	return exitOK
}

// notACommand answers a name nothing has, and offers the nearest there is.
func (c *cli) notACommand(name string) int {
	fmt.Fprintf(c.errs, "tigwx: '%s' is not a tigwx command. See 'tigwx --help'.\n", name)
	names := externals()
	for n := range c.commands() {
		names = append(names, n)
	}
	slices.Sort(names)
	best, bestDist := "", 3 // further than this is not a slip
	for _, n := range names {
		if d := editDistance(name, n); d < bestDist {
			best, bestDist = n, d
		}
	}
	if best != "" {
		fmt.Fprintf(c.errs, "\nThe most similar command is\n\t%s\n", best)
	}
	return exitNo
}

// editDistance counts the single-character edits between two words.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
