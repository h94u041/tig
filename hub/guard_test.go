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
)

func post(t *testing.T, url, token, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(data)
}

// Every page and answer tells a browser that what people pushed may not run.
func TestPagesAreGuarded(t *testing.T) {
	_, url := hub(t)
	for _, path := range []string{"/", "/api/v1/projects/alice/demo", "/nobody/nothing"} {
		resp, err := http.Get(url + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		csp := resp.Header.Get("Content-Security-Policy")
		for _, want := range []string{"default-src 'none'", "frame-ancestors 'none'", "base-uri 'none'"} {
			if !strings.Contains(csp, want) {
				t.Errorf("%s: no %q in %q", path, want, csp)
			}
		}
		if strings.Contains(csp, "script-src") {
			t.Errorf("%s: scripts are not for anyone: %q", path, csp)
		}
		if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: %q", path, got)
		}
		if got := resp.Header.Get("Referrer-Policy"); got != "no-referrer" {
			t.Errorf("%s: %q", path, got)
		}
	}
}

// Asking what the hub has costs it work, so it is for owners, and small.
func TestMissingNeedsTheOwner(t *testing.T) {
	_, url := hub(t)
	addr := url + "/api/v1/projects/alice/demo/missing"
	body := `{"objects":[]}`
	if code, out := post(t, addr, "", body); code != http.StatusUnauthorized {
		t.Fatal(code, out)
	}
	if code, out := post(t, addr, "bob-token", body); code != http.StatusForbidden {
		t.Fatal(code, out)
	}
	if code, out := post(t, addr, "alice-token", body); code != http.StatusOK {
		t.Fatal(code, out)
	}
	large, _ := json.Marshal(missingRequest{Objects: []string{strings.Repeat("a", maxMissing)}})
	if code, _ := post(t, addr, "alice-token", string(large)); code != http.StatusBadRequest {
		t.Fatal(code)
	}
}

// What goes wrong on the server is its own business: the answer does not name its folders.
func TestErrorsDoNotNameFolders(t *testing.T) {
	s, url := hub(t)
	write(t, s.Dir, "alice/demo/HEAD", "not a hash\n")
	for _, path := range []string{"/api/v1/projects/alice/demo", "/alice/demo", "/alice/demo/log"} {
		code, body := get(t, url+path)
		if code != http.StatusInternalServerError {
			t.Errorf("%s: %d", path, code)
		}
		if strings.Contains(body, s.Dir) || strings.Contains(body, filepath.ToSlash(s.Dir)) || strings.Contains(body, "HEAD") {
			t.Errorf("%s: %s", path, body)
		}
	}
}

func TestTokensFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "tokens")
	if err := os.WriteFile(file, []byte("# owners\nalice short\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTokens(file); err == nil || !strings.Contains(err.Error(), "shorter") {
		t.Fatalf("%v", err)
	}
	if err := os.WriteFile(file, []byte("# owners\nalice 0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tokens, err := LoadTokens(file)
	if err != nil || tokens["0123456789abcdef"] != "alice" {
		t.Fatalf("%v %v", tokens, err)
	}
}

func TestTokenStaysOffPlainHTTP(t *testing.T) {
	for address, want := range map[string]bool{
		"https://hub.example/alice/demo": true,
		"http://localhost:8080/a/b":      true,
		"http://LOCALHOST/a/b":           true,
		"http://127.0.0.1:8080/a/b":      true,
		"http://[::1]:8080/a/b":          true,
		"http://hub.example/a/b":         false,
		"http://10.0.0.5/a/b":            false,
		"http://localhost.evil.test/a/b": false,
		"http://127.0.0.1.evil.test/a/b": false,
	} {
		if got := safeForToken(address); got != want {
			t.Errorf("%s: %v, want %v", address, got, want)
		}
	}

	ctx := context.Background()
	r := project(t)
	_, err := Push(ctx, r, "http://hub.example/alice/demo", "a-token", "t", "")
	if err == nil || !strings.Contains(err.Error(), "plain http") {
		t.Fatalf("%v", err)
	}
	// Without a token there is nothing to read on the way; the hub says no itself.
	if _, err := PushWith(ctx, r, "http://hub.invalid/alice/demo", PushOptions{Actor: "t"}); err == nil || strings.Contains(err.Error(), "plain http") {
		t.Fatalf("%v", err)
	}
	// Said out loud, it goes.
	if _, err := PushWith(ctx, r, "http://hub.invalid/alice/demo", PushOptions{Token: "a-token", Actor: "t", Insecure: true}); err == nil || strings.Contains(err.Error(), "plain http") {
		t.Fatalf("%v", err)
	}
}

// A redirect could carry the token somewhere it was never meant to go.
func TestPushFollowsNoRedirect(t *testing.T) {
	var reached bool
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	defer other.Close()
	moved := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+r.URL.Path, http.StatusFound)
	}))
	defer moved.Close()
	if _, err := Push(context.Background(), project(t), moved.URL+"/alice/demo", "alice-token", "t", ""); err == nil {
		t.Fatal("no error")
	}
	if reached {
		t.Fatal("it followed the redirect")
	}
}
