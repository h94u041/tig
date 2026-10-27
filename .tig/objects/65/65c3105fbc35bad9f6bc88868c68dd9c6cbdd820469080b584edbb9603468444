package tig

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Tree headers. Trees written before tig had its own name are still read.
const (
	TreeHeader    = "tig-tree 1\n"
	oldTreeHeader = "tealprint-tree 1\n"
)

// Entry is one file in a tree.
type Entry struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
	Exec bool   `json:"exec,omitempty"`
}

// Change is one file that differs between two trees.
type Change struct {
	Path string `json:"path"`
	Op   string `json:"op"` // add, modify, delete
}

type treeItem struct {
	name string
	dir  bool
	hash string
	size int64
	exec bool
}

// validName reports whether a name can be one item of a tree: not empty, not
// "." or "..", and without a separator, a line break or NUL. Trees are read
// from other people's repositories, so this is checked on every read.
func validName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\\r\n\x00")
}

// The parsed form of each tree is kept: a tree never changes, and history
// views read the same trees again and again.
var trees = struct {
	sync.Mutex
	m map[string][]treeItem
}{m: map[string][]treeItem{}}

const treeCacheMax = 8192

func readTreeObject(ctx context.Context, objects Objects, hash string) ([]treeItem, error) {
	trees.Lock()
	items, ok := trees.m[hash]
	trees.Unlock()
	if ok {
		return items, nil
	}
	raw, err := objects.GetObject(ctx, hash)
	if err != nil {
		return nil, err
	}
	items, err = parseTree(hash, raw)
	if err != nil {
		return nil, err
	}
	trees.Lock()
	if len(trees.m) >= treeCacheMax {
		trees.m = map[string][]treeItem{}
	}
	trees.m[hash] = items
	trees.Unlock()
	return items, nil
}

func parseTree(hash string, raw []byte) ([]treeItem, error) {
	text, ok := strings.CutPrefix(string(raw), TreeHeader)
	if !ok {
		if text, ok = strings.CutPrefix(string(raw), oldTreeHeader); !ok {
			return nil, fmt.Errorf("tig: %s is not a tree", hash)
		}
	}
	var items []treeItem
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		kind, rest, _ := strings.Cut(sc.Text(), " ")
		h, rest, _ := strings.Cut(rest, " ")
		if !ValidHash(h) {
			return nil, fmt.Errorf("tig: tree %s has an invalid hash %q", hash, h)
		}
		it := treeItem{hash: h}
		switch kind {
		case "d":
			it.dir, it.name = true, rest
		case "f", "x":
			size, name, _ := strings.Cut(rest, " ")
			n, err := strconv.ParseInt(size, 10, 64)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("tig: tree %s has an invalid size %q", hash, size)
			}
			it.size, it.name, it.exec = n, name, kind == "x"
		default:
			return nil, fmt.Errorf("tig: tree %s has an unknown item %q", hash, kind)
		}
		if !validName(it.name) {
			return nil, fmt.Errorf("tig: tree %s has an invalid name %q", hash, it.name)
		}
		items = append(items, it)
	}
	return items, sc.Err()
}

// WriteTree stores the files as tree objects, one per folder (a folder that
// did not change is the same tree), and returns the root tree's hash.
func WriteTree(ctx context.Context, objects Objects, files map[string]Entry) (string, error) {
	type dir struct {
		files map[string]Entry
		dirs  map[string]*dir
	}
	newDir := func() *dir { return &dir{files: map[string]Entry{}, dirs: map[string]*dir{}} }
	root := newDir()
	for rel, e := range files {
		parts := strings.Split(rel, "/")
		for _, p := range parts {
			if !validName(p) {
				return "", fmt.Errorf("tig: %q cannot be in a tree", rel)
			}
		}
		d := root
		for _, name := range parts[:len(parts)-1] {
			next, ok := d.dirs[name]
			if !ok {
				next = newDir()
				d.dirs[name] = next
			}
			d = next
		}
		d.files[parts[len(parts)-1]] = e
	}
	var write func(d *dir) (string, error)
	write = func(d *dir) (string, error) {
		type line struct{ name, text string }
		var lines []line
		for name, sub := range d.dirs {
			h, err := write(sub)
			if err != nil {
				return "", err
			}
			lines = append(lines, line{name, "d " + h + " " + name})
		}
		for name, e := range d.files {
			kind := "f"
			if e.Exec {
				kind = "x"
			}
			lines = append(lines, line{name, kind + " " + e.Hash + " " + strconv.FormatInt(e.Size, 10) + " " + name})
		}
		sort.Slice(lines, func(i, j int) bool { return lines[i].name < lines[j].name })
		var b strings.Builder
		b.WriteString(TreeHeader)
		for _, l := range lines {
			b.WriteString(l.text)
			b.WriteByte('\n')
		}
		return objects.PutObject(ctx, []byte(b.String()))
	}
	return write(root)
}

// ReadTree lists every file under a tree object.
func ReadTree(ctx context.Context, objects Objects, hash string) (map[string]Entry, error) {
	out := map[string]Entry{}
	if hash == "" {
		return out, nil
	}
	var read func(hash, prefix string) error
	read = func(hash, prefix string) error {
		items, err := readTreeObject(ctx, objects, hash)
		if err != nil {
			return err
		}
		for _, it := range items {
			if it.dir {
				if err := read(it.hash, prefix+it.name+"/"); err != nil {
					return err
				}
				continue
			}
			out[prefix+it.name] = Entry{Hash: it.hash, Size: it.size, Exec: it.exec}
		}
		return nil
	}
	return out, read(hash, "")
}

// Lookup finds one file in a tree, reading only the folders on its path.
func Lookup(ctx context.Context, objects Objects, tree, rel string) (Entry, bool, error) {
	if tree == "" {
		return Entry{}, false, nil
	}
	parts := strings.Split(rel, "/")
	h := tree
	for i, name := range parts {
		items, err := readTreeObject(ctx, objects, h)
		if err != nil {
			return Entry{}, false, err
		}
		found := false
		for _, it := range items {
			if it.name != name {
				continue
			}
			last := i == len(parts)-1
			if last && !it.dir {
				return Entry{Hash: it.hash, Size: it.size, Exec: it.exec}, true, nil
			}
			if !last && it.dir {
				h, found = it.hash, true
			}
			break
		}
		if !found {
			return Entry{}, false, nil
		}
	}
	return Entry{}, false, nil
}

// DiffTrees lists the files that differ between two trees. A folder whose
// tree is the same on both sides is skipped without being read. Either tree
// may be empty ("").
func DiffTrees(ctx context.Context, objects Objects, a, b string) ([]Change, error) {
	var out []Change
	var walk func(a, b, prefix string) error
	all := func(h, prefix, op string) error {
		files, err := ReadTree(ctx, objects, h)
		if err != nil {
			return err
		}
		for p := range files {
			out = append(out, Change{prefix + p, op})
		}
		return nil
	}
	walk = func(a, b, prefix string) error {
		if a == b {
			return nil
		}
		var ai, bi []treeItem
		var err error
		if a != "" {
			if ai, err = readTreeObject(ctx, objects, a); err != nil {
				return err
			}
		}
		if b != "" {
			if bi, err = readTreeObject(ctx, objects, b); err != nil {
				return err
			}
		}
		before := map[string]treeItem{}
		for _, it := range ai {
			before[it.name] = it
		}
		after := map[string]treeItem{}
		for _, it := range bi {
			after[it.name] = it
		}
		for name, x := range before {
			y, ok := after[name]
			switch {
			case !ok && x.dir:
				err = all(x.hash, prefix+name+"/", "delete")
			case !ok:
				out = append(out, Change{prefix + name, "delete"})
			case x.dir && y.dir:
				err = walk(x.hash, y.hash, prefix+name+"/")
			case x.dir:
				err = all(x.hash, prefix+name+"/", "delete")
				out = append(out, Change{prefix + name, "add"})
			case y.dir:
				out = append(out, Change{prefix + name, "delete"})
				err = all(y.hash, prefix+name+"/", "add")
			case x.hash != y.hash || x.exec != y.exec:
				out = append(out, Change{prefix + name, "modify"})
			}
			if err != nil {
				return err
			}
		}
		for name, y := range after {
			if _, ok := before[name]; ok {
				continue
			}
			if y.dir {
				err = all(y.hash, prefix+name+"/", "add")
			} else {
				out = append(out, Change{prefix + name, "add"})
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(a, b, ""); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Diff lists the files that differ between two file sets, sorted.
func Diff(before, after map[string]Entry) []Change {
	var out []Change
	for p, e := range after {
		if b, ok := before[p]; !ok {
			out = append(out, Change{p, "add"})
		} else if b.Hash != e.Hash || b.Exec != e.Exec {
			out = append(out, Change{p, "modify"})
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, Change{p, "delete"})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// Paths is the changed paths only.
func Paths(cs []Change) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Path
	}
	return out
}
