package tig_test

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"testing"

	"github.com/h94u041/tig"
)

func TestLooksSecret(t *testing.T) {
	for path, want := range map[string]bool{
		".env":                   true,
		".env.local":             true,
		".env.production":        true,
		"app/.env":               true,
		".ENV":                   true,
		".env.example":           false,
		".env.sample":            false,
		".env.template":          false,
		".env.dist":              false,
		"id_rsa":                 true,
		"home/id_ed25519":        true,
		"id_rsa.pub":             false,
		".ssh/config":            true,
		"a/.aws/credentials":     true,
		".gnupg/pubring.kbx":     true,
		"server.pem":             true,
		"certs/server.KEY":       true,
		"store.p12":              true,
		"credentials.json":       true,
		".npmrc":                 true,
		".netrc":                 true,
		"src/env.go":             false,
		"README.md":              false,
		"docs/keys.md":           false,
		"src/key.go":             false,
		"secrets.go":             false,
		"environment.md":         false,
		"docs/.ssh-explained.md": false,
	} {
		if got := tig.LooksSecret(path); got != want {
			t.Errorf("%s: %v, want %v", path, got, want)
		}
	}
}

// Without Git nothing says what should be kept, so anything that looks like a key
// stays out of the snapshots, and everything built on the file list with it.
func TestSecretsStayOutWithoutGit(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	for _, f := range []string{"README.md", "src/app.go", ".env", ".env.example", ".ssh/id_rsa", "certs/server.pem"} {
		write(t, root, f, "the content of "+f+"\n")
	}
	r := tig.Open(root)
	if err := r.Init(tig.Config{Root: "README.md"}); err != nil {
		t.Fatal(err)
	}
	files, err := tig.Files(ctx, root, r.Skip)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(files, []string{".env.example", "README.md", "src/app.go"}) {
		t.Fatalf("%v", files)
	}
	left, err := tig.LeftOut(ctx, root, r.Skip)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(left, []string{".env", ".ssh/id_rsa", "certs/server.pem"}) {
		t.Fatalf("%v", left)
	}
	snap, _, err := r.Take(ctx, tig.ReasonOpen, "t")
	if err != nil {
		t.Fatal(err)
	}
	tree, err := tig.ReadTree(ctx, r.Objects, snap.Tree)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tree[".env"]; ok {
		t.Fatalf("%v", tree)
	}
	if _, ok := tree["README.md"]; !ok {
		t.Fatalf("%v", tree)
	}
	if _, err := r.ReadFile(ctx, ".env"); err == nil {
		t.Fatal("read_file reads through this: it must not reach a secret")
	}
}

// Git's word counts: what it tracks is already in its history and stays listed, and
// what it ignores is not listed and not worth a mention.
func TestSecretsWithGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip(err)
	}
	ctx := context.Background()
	root := t.TempDir()
	for _, f := range []string{"README.md", "tracked.pem", ".env", "ignored.key", ".gitignore"} {
		write(t, root, f, "x\n")
	}
	write(t, root, ".gitignore", "ignored.key\n")
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatal(errors.New(string(out)))
		}
	}
	git("init", "-q")
	git("add", "README.md", "tracked.pem", ".gitignore")

	files, err := tig.Files(ctx, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(files, []string{".gitignore", "README.md", "tracked.pem"}) {
		t.Fatalf("%v", files)
	}
	left, err := tig.LeftOut(ctx, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(left, []string{".env"}) {
		t.Fatalf("%v", left)
	}

	git("add", ".env")
	if files, err = tig.Files(ctx, root, nil); err != nil || !slices.Contains(files, ".env") {
		t.Fatalf("tracking it is how to keep it: %v %v", files, err)
	}
	if left, err = tig.LeftOut(ctx, root, nil); err != nil || len(left) != 0 {
		t.Fatalf("%v %v", left, err)
	}
}
