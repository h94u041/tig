package mcpserver

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/h94u041/tig"
)

func write(t *testing.T, dir, rel, text string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(full), 0o755)
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// connect is an agent named name talking to the project's tig over MCP.
func connect(t *testing.T, r *tig.Repo, name string) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := New(r, Options{Version: "test"}).Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: name, Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close(); ss.Wait() })
	return cs
}

func call[T any](t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) T {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("%s: %s", name, res.Content[0].(*mcp.TextContent).Text)
	}
	var out T
	raw, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: %v\n%s", name, err, raw)
	}
	return out
}

// fails is the error a call ends in.
func fails(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatalf("%s should fail", name)
	}
	return res.Content[0].(*mcp.TextContent).Text
}

func outlined(t *testing.T, o overviewOut, path, title string) string {
	t.Helper()
	for _, it := range o.Outline {
		if it.Path == path && it.Title == title {
			return it.Block
		}
	}
	t.Fatalf("no %q in %s: %+v", title, path, o.Outline)
	return ""
}

const shop = "# Shop\n\nA small shop. Prices are in [pricing](docs/pricing.md).\n\n" +
	"## Cart\n\nThe cart adds items. Code: [cart](src/cart.js).\n\n### Acceptance\n\n- `echo cart-ok`\n"

const pricing = "# Pricing\n\nPrices round to cents.\n\n## Acceptance\n\n- `echo pricing-ok`\n"

func TestTools(t *testing.T) {
	root := t.TempDir()
	write(t, root, "README.md", shop)
	write(t, root, "docs/pricing.md", pricing)
	write(t, root, "src/cart.js", "add()\n")
	r := tig.Open(root)
	if err := r.Init(tig.Config{Root: "README.md"}); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, r, "agent-a")

	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 17 {
		t.Fatalf("tools: %v %v", tools, err)
	}
	o := call[overviewOut](t, cs, "overview", nil)
	if !o.Closed || strings.Join(o.Docs, ",") != "README.md,docs/pricing.md" || o.InProgress != nil {
		t.Fatalf("overview: %+v", o)
	}
	cart := outlined(t, o, "README.md", "Cart")
	shopRoot, pricingRoot := o.Outline[0].Block, outlined(t, o, "docs/pricing.md", "")

	b := call[blockOut](t, cs, "read_block", map[string]any{"block": cart[:8]})
	if b.Title != "Cart" || !slices.Contains(b.Files, "src/cart.js") || len(b.Children) != 1 {
		t.Fatalf("read_block: %+v", b)
	}
	down := call[followOut](t, cs, "follow", map[string]any{"block": shopRoot, "depth": 3})
	if !slices.ContainsFunc(down.Nodes, func(n node) bool { return n.Path == "docs/pricing.md" && n.Via == "detail" }) {
		t.Fatalf("follow down: %+v", down)
	}
	up := call[followOut](t, cs, "follow", map[string]any{"block": pricingRoot, "direction": "up", "depth": 1})
	if len(up.Nodes) != 2 || up.Nodes[1].Block != outlined(t, o, "README.md", "Shop") || up.Nodes[1].Via != "handed-on-from" {
		t.Fatalf("follow up: %+v", up)
	}
	if s := call[searchOut](t, cs, "search", map[string]any{"query": "round CENTS"}); len(s.Hits) != 1 || s.Hits[0].Path != "docs/pricing.md" {
		t.Fatalf("search: %+v", s)
	}

	w := call[tig.OpenWork](t, cs, "start_work", map[string]any{"basis": []string{cart[:10]}, "intent": "cart totals"})
	if w.Actor != "agent-a" || w.Basis[0] != cart {
		t.Fatalf("start_work: %+v", w)
	}
	write(t, root, "src/cart.js", "add()\ntotal()\n")
	edit := call[tig.Step](t, cs, "record_step", map[string]any{"kind": "edit", "note": "totals"})
	for _, command := range []string{"echo cart-ok", "echo pricing-ok"} {
		call[tig.Step](t, cs, "record_step", map[string]any{"kind": "check", "command": command, "exit": 0, "output": "ok"})
	}
	fails(t, cs, "record_step", map[string]any{"kind": "guess"})
	done := call[tig.Work](t, cs, "finish_work", nil)
	if done.Result != tig.ResultPassed || done.Actor != "agent-a" || len(done.Steps) != 3 {
		t.Fatalf("finish_work: %+v", done)
	}

	tr := call[trailOut](t, cs, "trail", map[string]any{"block": cart, "steps": true})
	if len(tr.Works) != 1 || len(tr.Works[0].StepList) != 3 {
		t.Fatalf("trail: %+v", tr)
	}
	first := tr.Works[0].StepList[0]
	if first.ID != edit.ID || len(first.Changes) != 1 || first.Changes[0].Path != "src/cart.js" {
		t.Fatalf("the edit's changes: %+v", first)
	}
	if out := tr.Works[0].StepList[1].OutputText; out != "ok" {
		t.Fatalf("the check's output: %q", out)
	}
	if ft := call[trailOut](t, cs, "trail", map[string]any{"file": "src/cart.js"}); len(ft.Works) != 1 {
		t.Fatalf("file trail: %+v", ft)
	}

	if f := call[readFileOut](t, cs, "read_file", map[string]any{"path": "src/cart.js", "tree": edit.Before}); f.Content != "add()\n" {
		t.Fatalf("read_file in a tree: %+v", f)
	}
	if f := call[readFileOut](t, cs, "read_file", map[string]any{"path": "src/cart.js"}); f.Content != "add()\ntotal()\n" {
		t.Fatalf("read_file now: %+v", f)
	}
	if f := call[readFileOut](t, cs, "read_file", map[string]any{"path": "src/none.js"}); f.Exists {
		t.Fatal("no such file")
	}
	for _, p := range []string{".tig/config", "../outside.txt", "src/../../outside.txt"} {
		fails(t, cs, "read_file", map[string]any{"path": p})
	}

	if c := call[checkOut](t, cs, "check", nil); !c.Closed {
		t.Fatalf("closed: %+v", c.Report)
	}

	l := call[tig.Link](t, cs, "propose_link", map[string]any{"from": cart, "toFile": "src/cart.js", "kind": "implements"})
	if l.ID == "" || l.Actor != "agent-a" {
		t.Fatalf("propose_link: %+v", l)
	}

	ctx := context.Background()
	res, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tig://block/" + cart})
	if err != nil || !strings.Contains(res.Contents[0].Text, "The cart adds items.") {
		t.Fatalf("block resource: %+v %v", res, err)
	}
	res, err = cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tig://work/" + done.ID})
	if err != nil || !strings.Contains(res.Contents[0].Text, "cart totals") {
		t.Fatalf("work resource: %+v %v", res, err)
	}
	if _, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "tig://object/../../config"}); err == nil {
		t.Fatal("not an object")
	}

	// An agent's sentence the main document does not say is only a guess.
	m := call[tig.Mark](t, cs, "mark_sentence", map[string]any{"sentence": "Totals include tax.", "commits": []string{done.ID[:4]}})
	if m.Path != "inferred.md" {
		t.Fatalf("a guess: %+v", m)
	}
	said := call[tig.Mark](t, cs, "mark_sentence", map[string]any{"sentence": "The cart adds items.", "commits": []string{done.ID[:4]}})
	if said.Path != "README.md" {
		t.Fatalf("what the document says: %+v", said)
	}
	if v := call[tig.RefView](t, cs, "resolve", map[string]any{"ref": "l" + said.Hash[:6]}); v.Line == nil || !strings.HasPrefix(v.Line.Text, "The cart adds items.") || len(v.Line.Intents) != 1 {
		t.Fatalf("resolve a line: %+v", v)
	}
	if v := call[tig.RefView](t, cs, "resolve", map[string]any{"ref": done.ID[:4]}); v.Work == nil || len(v.Work.Intents) != 2 || v.Work.Acceptance != tig.WorkCandidate {
		t.Fatalf("resolve a work: %+v", v)
	}
	data, _ := os.ReadFile(filepath.Join(root, "README.md"))
	write(t, root, "README.md", strings.Replace(string(data), "The cart adds items.", "The cart adds and removes items.", 1))
	if res := call[tig.SyncResult](t, cs, "sync_lines", nil); len(res.Touched) != 1 {
		t.Fatalf("sync_lines: %+v", res)
	}
	fails(t, cs, "resolve", map[string]any{"ref": "nonsense"})
}

const greeter = "# Greeter\n\nA program that greets. Its module is [go.mod](go.mod) and its check is [check](check/main.go).\n\n" +
	"## Greeting\n\n`hello.txt` says hello to the world.\n\n### Acceptance\n\n- `go run ./check`\n"

const checkProgram = `package main

import (
	"fmt"
	"os"
)

func main() {
	b, err := os.ReadFile("hello.txt")
	if err != nil || string(b) != "hello, world\n" {
		fmt.Println("hello.txt does not say hello, world")
		os.Exit(1)
	}
	fmt.Println("ok")
}
`

func goRun(dir string) ([]byte, error) {
	cmd := exec.Command("go", "run", "./check")
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// TestRebuild is the point of tig: one agent builds what the documents
// design, and another agent, given only the documents and the footprint,
// builds it again by following the footprint, and knows when it is done.
func TestRebuild(t *testing.T) {
	if testing.Short() {
		t.Skip("runs go")
	}
	a := t.TempDir()
	write(t, a, "README.md", greeter)
	write(t, a, "go.mod", "module greeter\n\ngo 1.22\n")
	write(t, a, "check/main.go", checkProgram)
	ra := tig.Open(a)
	if err := ra.Init(tig.Config{Root: "README.md"}); err != nil {
		t.Fatal(err)
	}

	// Agent A builds it.
	ca := connect(t, ra, "agent-a")
	o := call[overviewOut](t, ca, "overview", nil)
	greeting := outlined(t, o, "README.md", "Greeting")
	call[tig.OpenWork](t, ca, "start_work", map[string]any{"basis": []string{greeting}, "intent": "write hello.txt"})
	write(t, a, "hello.txt", "hello, world\n")
	call[tig.Step](t, ca, "record_step", map[string]any{"kind": "edit", "note": "hello.txt says hello"})
	if out, err := goRun(a); err != nil {
		t.Fatalf("agent A's check: %v\n%s", err, out)
	}
	call[tig.Step](t, ca, "record_step", map[string]any{"kind": "check", "command": "go run ./check", "exit": 0})
	call[tig.Work](t, ca, "finish_work", nil)

	// Agent B gets the documents, the check and the footprint; not the file.
	b := t.TempDir()
	filepath.WalkDir(a, func(p string, d fs.DirEntry, err error) error {
		rel, _ := filepath.Rel(a, p)
		rel = filepath.ToSlash(rel)
		if err != nil || d.IsDir() || rel == "hello.txt" || slices.Contains([]string{".tig/lock", ".tig/work.lock", ".tig/WORK"}, rel) || strings.HasPrefix(rel, ".tig/cache/") {
			return err
		}
		data, _ := os.ReadFile(p)
		write(t, b, rel, string(data))
		return nil
	})
	rb := tig.Open(b)
	cb := connect(t, rb, "agent-b")
	c := call[checkOut](t, cb, "check", nil)
	if c.Closed || len(c.Stale) != 1 || c.Stale[0].Path != "hello.txt" || !strings.Contains(c.Stale[0].Message, "gone") {
		t.Fatalf("agent B sees what is missing: %+v", c.Report)
	}
	tr := call[trailOut](t, cb, "trail", map[string]any{"block": greeting, "steps": true})
	if len(tr.Works) != 1 {
		t.Fatalf("trail: %+v", tr)
	}
	call[tig.OpenWork](t, cb, "start_work", map[string]any{"basis": []string{greeting}, "intent": "rebuild by the footprint"})
	for _, s := range tr.Works[0].StepList {
		for _, ch := range s.Changes {
			f := call[readFileOut](t, cb, "read_file", map[string]any{"path": ch.Path, "tree": s.After})
			write(t, b, ch.Path, f.Content)
		}
	}
	call[tig.Step](t, cb, "record_step", map[string]any{"kind": "edit", "note": "followed agent A's footprint"})
	if out, err := goRun(b); err != nil {
		t.Fatalf("agent B's check: %v\n%s", err, out)
	}
	call[tig.Step](t, cb, "record_step", map[string]any{"kind": "check", "command": "go run ./check", "exit": 0})
	w := call[tig.Work](t, cb, "finish_work", nil)
	if w.Result != tig.ResultPassed || w.Actor != "agent-b" {
		t.Fatalf("agent B's work: %+v", w)
	}
	if c := call[checkOut](t, cb, "check", nil); !c.Closed {
		t.Fatalf("closed again: %+v", c.Report)
	}
}
