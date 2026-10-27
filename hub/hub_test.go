package hub

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

const secret = "do-not-send-this-code"

// project is a small project with a tig: a main document, a detail
// document, and code the hub must never get.
func project(t *testing.T) *tig.Repo {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, "README.md", "# Demo\n\nA demo. Details in [the design](docs/design.md).\n")
	write(t, dir, "docs/design.md", "# Design\n\nHow it works. Code: [main.go](../main.go).\n\n## Acceptance\n\n- `go version`\n")
	write(t, dir, "main.go", "package main // "+secret+"\n")
	write(t, dir, "notes/orphan.md", "# Not in the chain\n")
	r := tig.Open(dir)
	if err := r.Init(tig.Config{Root: "README.md", Outside: []string{"notes"}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Take(context.Background(), tig.ReasonOpen, "test"); err != nil {
		t.Fatal(err)
	}
	return r
}

// hub is a hub in the test, where "alice" can push.
func hub(t *testing.T) (*Server, string) {
	t.Helper()
	s := NewServer(t.TempDir(), map[string]string{"alice-token": "alice", "bob-token": "bob"})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return s, ts.URL
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestPush(t *testing.T) {
	ctx := context.Background()
	r := project(t)
	s, url := hub(t)
	addr := url + "/alice/demo"

	res, err := Push(ctx, r, addr, "alice-token", "test", "first")
	if err != nil {
		t.Fatal(err)
	}
	if res.UpToDate || res.Sent == 0 {
		t.Fatalf("first push: %+v", res)
	}

	// Only the documents of the chain and README.md, and the chain.
	p, _ := s.Project("alice", "demo")
	pub, v, err := p.Latest(ctx)
	if err != nil || v == nil {
		t.Fatalf("latest: %v %v", v, err)
	}
	docs, err := tig.ReadTree(ctx, p.Objects(), pub.Docs)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs["README.md"].Hash == "" || docs["docs/design.md"].Hash == "" {
		t.Fatalf("documents on the hub: %v", docs)
	}
	filepath.WalkDir(string(p.Objects()), func(file string, d os.DirEntry, err error) error {
		if data, _ := os.ReadFile(file); strings.Contains(string(data), secret) {
			t.Errorf("the hub has the project's code: %s", file)
		}
		return nil
	})
	design := v.Block(tig.ParseBlocks("docs/design.md", []byte("# Design\n\nHow it works. Code: [main.go](../main.go).\n\n## Acceptance\n\n- `go version`\n"))[1].Hash)
	if design == nil || design.Status != tig.StatusDirection || len(design.Refs) != 1 || design.Refs[0] != "main.go" {
		t.Fatalf("the design block on the hub: %+v", design)
	}

	// Nothing changed: nothing to push.
	res, err = Push(ctx, r, addr, "alice-token", "test", "")
	if err != nil || !res.UpToDate {
		t.Fatalf("second push: %+v %v", res, err)
	}

	// A changed document, snapped, is a new publication on top of the first.
	write(t, r.Root, "README.md", "# Demo\n\nA demo, changed. Details in [the design](docs/design.md).\n")
	res, err = Push(ctx, r, addr, "alice-token", "test", "")
	if err != nil || !res.UpToDate || len(res.Unsnapped) != 1 {
		t.Fatalf("unsnapped push: %+v %v", res, err)
	}
	if _, _, err := r.Take(ctx, tig.ReasonScan, "test"); err != nil {
		t.Fatal(err)
	}
	res, err = Push(ctx, r, addr, "alice-token", "test", "second")
	if err != nil || res.UpToDate || res.Publication.Parent != pub.ID {
		t.Fatalf("third push: %+v %v", res, err)
	}
	if res.Sent != 4 { // the publication, the chain, the tree and README.md
		t.Errorf("sent %d objects, want only the 4 new ones", res.Sent)
	}
}

func TestRefuse(t *testing.T) {
	ctx := context.Background()
	r := project(t)
	_, url := hub(t)
	addr := url + "/alice/demo"

	for token, want := range map[string]string{"": "needs the owner's token", "bob-token": "not alice's", "wrong": "needs the owner's token"} {
		if _, err := Push(ctx, r, addr, token, "test", ""); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("push with token %q: %v, want %q", token, err, want)
		}
	}

	push := func(req pushRequest) error {
		c := &client{api: url + "/api/v1/projects/alice/demo", token: "alice-token"}
		return c.call(ctx, "POST", "/push", req, nil)
	}
	pub, objects, err := Build(ctx, r, "", "test", "")
	if err != nil {
		t.Fatal(err)
	}
	all := func() map[string][]byte {
		out := map[string][]byte{}
		for h, d := range objects {
			out[h] = d
		}
		return out
	}

	// An object that does not match its name.
	bad := all()
	bad[pub.Chain] = []byte("{}")
	if err := push(pushRequest{New: pub.ID, Objects: bad}); err == nil || !strings.Contains(err.Error(), "does not match its hash") {
		t.Errorf("corrupt object: %v", err)
	}

	// A tree with code in it.
	m := memory{}
	for h, d := range objects {
		m[h] = d
	}
	files, _ := tig.ReadTree(ctx, m, pub.Docs)
	code, _ := m.PutObject(ctx, []byte(secret))
	files["main.go"] = tig.Entry{Hash: code, Size: int64(len(secret))}
	withCode := pub
	withCode.Docs, _ = tig.WriteTree(ctx, m, files)
	withCode.ID, _ = m.PutObject(ctx, withCode.bytes())
	if err := push(pushRequest{New: withCode.ID, Objects: m}); err == nil || !strings.Contains(err.Error(), "documents only") {
		t.Errorf("code in the push: %v", err)
	}

	// A chain that is not the documents' chain.
	m = memory{}
	for h, d := range objects {
		m[h] = d
	}
	v, _ := loadChain(ctx, m, pub.Chain)
	v.Blocks[1].Hash = strings.Repeat("a", 64)
	raw, _ := json.Marshal(v)
	lying := pub
	lying.Chain, _ = m.PutObject(ctx, raw)
	lying.ID, _ = m.PutObject(ctx, lying.bytes())
	if err := push(pushRequest{New: lying.ID, Objects: m}); err == nil || !strings.Contains(err.Error(), "documents do not have") {
		t.Errorf("a chain of other documents: %v", err)
	}

	// Two pushes built on the same head: the second is refused.
	if err := push(pushRequest{New: pub.ID, Objects: all()}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	again, objects2, _ := Build(ctx, r, "", "test", "")
	if err := push(pushRequest{New: again.ID, Objects: objects2}); err == nil || !strings.Contains(err.Error(), "no longer the one") {
		t.Errorf("a push on an old head: %v", err)
	}

	// Names a hub does not take.
	if _, err := Push(ctx, r, url+"/alice/..", "alice-token", "test", ""); err == nil {
		t.Error("a push to /alice/.. went through")
	}
}

func TestPages(t *testing.T) {
	ctx := context.Background()
	r := project(t)
	_, url := hub(t)
	if _, err := Push(ctx, r, url+"/alice/demo", "alice-token", "test", "first"); err != nil {
		t.Fatal(err)
	}

	for path, wants := range map[string][]string{
		"/":                              {"alice/demo", `badge closed">closed`},
		"/alice/demo":                    {"docs/design.md", "direction", "go version", "first"},
		"/alice/demo/log":                {"first", "test"},
		"/alice/demo/doc/README.md":      {`href="/alice/demo/doc/docs/design.md"`, "A demo."},
		"/alice/demo/doc/docs/design.md": {"How it works", "links <code>main.go</code>"},
		"/api/v1/projects/alice/demo":    {`"root":"README.md"`},
	} {
		code, body := get(t, url+path)
		if code != http.StatusOK {
			t.Errorf("%s: %d", path, code)
			continue
		}
		for _, want := range wants {
			if !strings.Contains(body, want) {
				t.Errorf("%s does not show %q", path, want)
			}
		}
	}
	// A link to code the hub does not have is only its name.
	if _, body := get(t, url+"/alice/demo/doc/docs/design.md"); strings.Contains(body, `href="../main.go"`) {
		t.Error("the design page links to code the hub does not have")
	}
	for _, path := range []string{"/alice/nothing", "/alice/demo/doc/main.go", "/alice/demo/doc/notes/orphan.md"} {
		if code, _ := get(t, url+path); code != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", path, code)
		}
	}
}
