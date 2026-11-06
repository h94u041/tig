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
// break in it). A skip entry with a slash in it is a path from the root, and it
// holds in a Git repository too: a tool can keep its own records in the project
// without them counting as the project's files. An entry without one is a folder
// name, for a project without Git. A file that looks like a key or a credential
// is left out unless Git already tracks it: see LooksSecret and LeftOut.
func Files(ctx context.Context, root string, skip []string) ([]string, error) {
	files, _, err := listFiles(ctx, root, skip)
	return files, err
}

// LeftOut is what Files leaves out because it looks like a secret, so whoever
// wants one of them kept can say so to Git by tracking it.
func LeftOut(ctx context.Context, root string, skip []string) ([]string, error) {
	_, out, err := listFiles(ctx, root, skip)
	return out, err
}

func listFiles(ctx context.Context, root string, skip []string) (files, secret []string, err error) {
	var under []string
	for _, d := range skip {
		if strings.Contains(d, "/") {
			under = append(under, strings.TrimSuffix(d, "/")+"/")
		}
	}
	listed := func(rel string) bool {
		for _, u := range under {
			if strings.HasPrefix(rel, u) {
				return false
			}
		}
		return listable(rel)
	}
	if out, err := git(ctx, root, "rev-parse", "--is-inside-work-tree"); err == nil && strings.TrimSpace(string(out)) == "true" {
		tracked, err1 := git(ctx, root, "ls-files", "-c", "-z")
		others, err2 := git(ctx, root, "ls-files", "-o", "--exclude-standard", "-z")
		if err1 == nil && err2 == nil {
			seen := map[string]bool{}
			add := func(list []byte, guard bool) {
				for _, rel := range strings.Split(string(list), "\x00") {
					if rel == "" || seen[rel] || !listed(rel) {
						continue
					}
					seen[rel] = true
					fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
					if err != nil || !fi.Mode().IsRegular() {
						continue
					}
					if guard && LooksSecret(rel) {
						secret = append(secret, rel)
						continue
					}
					files = append(files, rel)
				}
			}
			add(tracked, false)
			add(others, true)
			sort.Strings(files)
			sort.Strings(secret)
			return files, secret, nil
		}
	}
	skipped := map[string]bool{}
	for _, d := range append(append([]string{}, defaultSkip...), skip...) {
		skipped[d] = true
	}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
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
		if d.Type().IsRegular() && listed(rel) {
			if LooksSecret(rel) {
				secret = append(secret, rel)
			} else {
				files = append(files, rel)
			}
		}
		return nil
	})
	sort.Strings(files)
	sort.Strings(secret)
	return files, secret, err
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
