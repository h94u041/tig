package tig

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var defaultSkip = []string{".git", ".tig", "node_modules"}

// Files lists the project's files: in a Git repository the files Git sees
// (tracked ones that exist, and new ones .gitignore does not exclude);
// otherwise every file, without the skipped folders. tig's own .tig is never
// listed, and neither is a name a tree cannot hold (a backslash or a line
// break in it).
func Files(ctx context.Context, root string, skip []string) ([]string, error) {
	if out, err := git(ctx, root, "rev-parse", "--is-inside-work-tree"); err == nil && strings.TrimSpace(string(out)) == "true" {
		if out, err := git(ctx, root, "ls-files", "-co", "--exclude-standard", "-z"); err == nil {
			var files []string
			seen := map[string]bool{}
			for _, rel := range strings.Split(string(out), "\x00") {
				if rel == "" || seen[rel] || !listable(rel) {
					continue
				}
				seen[rel] = true
				if fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel))); err == nil && fi.Mode().IsRegular() {
					files = append(files, rel)
				}
			}
			sort.Strings(files)
			return files, nil
		}
	}
	skipped := map[string]bool{}
	for _, d := range append(append([]string{}, defaultSkip...), skip...) {
		skipped[d] = true
	}
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root {
				return err
			}
			return nil // an unreadable folder is not part of the snapshot
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && skipped[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && listable(rel) {
			files = append(files, rel)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func listable(rel string) bool {
	if rel == ".tig" || strings.HasPrefix(rel, ".tig/") {
		return false
	}
	if strings.ContainsAny(rel, "\\\r\n\x00") {
		return false
	}
	for _, part := range strings.Split(rel, "/") {
		if !validName(part) {
			return false
		}
	}
	return true
}

func git(ctx context.Context, root string, args ...string) ([]byte, error) {
	// A repository's own core.fsmonitor setting would run a program on every
	// listing; tig only needs Git's file list.
	full := append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
