package hub

import (
	"bufio"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/h94u041/tig"
)

// Limits on what a push may carry.
const (
	maxPush   = 32 << 20 // the whole request
	maxObject = 4 << 20  // one object; tig reads no larger document
)

// Server is a hub: projects' publications, kept in a folder, one folder per
// owner and project, each with its objects and its HEAD.
type Server struct {
	Dir    string
	Tokens map[string]string // token → owner

	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewServer is a hub keeping its projects in dir.
func NewServer(dir string, tokens map[string]string) *Server {
	return &Server{Dir: dir, Tokens: tokens, locks: map[string]*sync.Mutex{}}
}

// LoadTokens reads a tokens file: one "owner token" per line; blank lines and
// lines starting with # are skipped.
func LoadTokens(file string) (map[string]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		owner, token, ok := strings.Cut(line, " ")
		token = strings.TrimSpace(token)
		if !ok || !validName(owner) || token == "" {
			return nil, fmt.Errorf("hub: %s:%d: want \"owner token\"", file, n)
		}
		out[token] = owner
	}
	return out, sc.Err()
}

var nameRule = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// validName reports whether a hub takes s as an owner or project name; "api"
// is the hub's own.
func validName(s string) bool {
	return nameRule.MatchString(s) && !strings.Contains(s, "..") && !strings.EqualFold(s, "api")
}

// Project is one project on the hub.
type Project struct {
	Owner, Name string
	srv         *Server
}

func (p Project) dir() string      { return filepath.Join(p.srv.Dir, p.Owner, p.Name) }
func (p Project) Objects() tig.Dir { return tig.Dir(filepath.Join(p.dir(), "objects")) }
func (p Project) String() string   { return p.Owner + "/" + p.Name }

// Head is the project's latest publication ("" when it has none).
func (p Project) Head() (string, error) {
	data, err := os.ReadFile(filepath.Join(p.dir(), "HEAD"))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	h := strings.TrimSpace(string(data))
	if err == nil && !tig.ValidHash(h) {
		err = fmt.Errorf("hub: %s has an invalid HEAD", p)
	}
	return h, err
}

// setHead moves the head, through a temporary file and a rename, so a reader
// never sees half of it.
func (p Project) setHead(id string) error {
	if err := os.MkdirAll(p.dir(), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(p.dir(), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.WriteString(id + "\n")
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(p.dir(), "HEAD"))
}

// Latest is the project's head publication and its chain.
func (p Project) Latest(ctx context.Context) (Publication, *ChainView, error) {
	head, err := p.Head()
	if err != nil || head == "" {
		return Publication{}, nil, err
	}
	pub, err := LoadPublication(ctx, p.Objects(), head)
	if err != nil {
		return Publication{}, nil, err
	}
	v, err := loadChain(ctx, p.Objects(), pub.Chain)
	return pub, v, err
}

func loadChain(ctx context.Context, objects tig.Objects, id string) (*ChainView, error) {
	raw, err := objects.GetObject(ctx, id)
	if err != nil {
		return nil, err
	}
	var v ChainView
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("hub: the chain %s is not readable: %v", id, err)
	}
	return &v, nil
}

// Project is a project by its owner and name; false for names a hub does not
// take.
func (s *Server) Project(owner, name string) (Project, bool) {
	if !validName(owner) || !validName(name) {
		return Project{}, false
	}
	return Project{Owner: owner, Name: name, srv: s}, true
}

// Projects are the projects that have a publication, by owner and name.
func (s *Server) Projects() []Project {
	var out []Project
	owners, _ := os.ReadDir(s.Dir)
	for _, o := range owners {
		names, _ := os.ReadDir(filepath.Join(s.Dir, o.Name()))
		for _, n := range names {
			if p, ok := s.Project(o.Name(), n.Name()); ok {
				if h, _ := p.Head(); h != "" {
					out = append(out, p)
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

func (s *Server) lock(p Project) func() {
	s.mu.Lock()
	m := s.locks[p.String()]
	if m == nil {
		m = &sync.Mutex{}
		s.locks[p.String()] = m
	}
	s.mu.Unlock()
	m.Lock()
	return m.Unlock
}

// owner is who the request's token belongs to ("" for none).
func (s *Server) owner(r *http.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return ""
	}
	found := ""
	for t, owner := range s.Tokens {
		if subtle.ConstantTimeCompare([]byte(t), []byte(token)) == 1 {
			found = owner
		}
	}
	return found
}

// The API.

// ProjectInfo is what GET /api/v1/projects/owner/name answers.
type ProjectInfo struct {
	Owner string       `json:"owner"`
	Name  string       `json:"name"`
	Head  *Publication `json:"head,omitempty"`
	Chain *ChainView   `json:"chain,omitempty"`
}

type missingRequest struct {
	Objects []string `json:"objects"`
}

type missingResponse struct {
	Missing []string `json:"missing"`
}

type pushRequest struct {
	Old     string            `json:"old"` // the head the push was built on
	New     string            `json:"new"`
	Objects map[string][]byte `json:"objects"`
}

type pushResponse struct {
	Head string `json:"head"`
}

type apiError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, format string, args ...any) {
	writeJSON(w, code, apiError{Error: fmt.Sprintf(format, args...)})
}

// Handler serves the hub: the API and the pages.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/projects/{owner}/{name}", s.apiProject)
	mux.HandleFunc("POST /api/v1/projects/{owner}/{name}/missing", s.apiMissing)
	mux.HandleFunc("POST /api/v1/projects/{owner}/{name}/push", s.apiPush)
	mux.HandleFunc("GET /{$}", s.pageIndex)
	mux.HandleFunc("GET /{owner}/{name}", s.pageProject)
	mux.HandleFunc("GET /{owner}/{name}/log", s.pageLog)
	mux.HandleFunc("GET /{owner}/{name}/doc/{path...}", s.pageDoc)
	return mux
}

func (s *Server) project(w http.ResponseWriter, r *http.Request) (Project, bool) {
	p, ok := s.Project(r.PathValue("owner"), r.PathValue("name"))
	if !ok {
		fail(w, http.StatusNotFound, "no such project")
	}
	return p, ok
}

func (s *Server) apiProject(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	pub, v, err := p.Latest(r.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	info := ProjectInfo{Owner: p.Owner, Name: p.Name, Chain: v}
	if pub.ID != "" {
		info.Head = &pub
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) apiMissing(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	var req missingRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxPush)).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	resp := missingResponse{Missing: []string{}}
	for _, h := range req.Objects {
		if tig.ValidHash(h) && !p.Objects().Has(h) {
			resp.Missing = append(resp.Missing, h)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) apiPush(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	switch owner := s.owner(r); {
	case owner == "":
		fail(w, http.StatusUnauthorized, "a push needs the owner's token")
		return
	case owner != p.Owner:
		fail(w, http.StatusForbidden, "the token is not %s's", p.Owner)
		return
	}
	var req pushRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxPush)).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, "the push is not readable or too large: %v", err)
		return
	}
	defer s.lock(p)()
	head, err := p.Head()
	if err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	if req.Old != head {
		fail(w, http.StatusConflict, "the project's head is no longer the one the push was built on; push again")
		return
	}
	keep, err := s.accept(r.Context(), p, head, req)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	store := p.Objects()
	for _, data := range keep {
		if _, err := store.PutObject(r.Context(), data); err != nil {
			fail(w, http.StatusInternalServerError, "%v", err)
			return
		}
	}
	if err := p.setHead(req.New); err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, pushResponse{Head: req.New})
}

// incoming reads a push's objects, then the project's, and remembers every
// object read.
type incoming struct {
	push  map[string][]byte
	store tig.Dir
	read  map[string]bool
}

func (in *incoming) PutObject(context.Context, []byte) (string, error) {
	return "", errors.New("hub: read only")
}

func (in *incoming) GetObject(ctx context.Context, h string) ([]byte, error) {
	in.read[h] = true
	if data, ok := in.push[h]; ok {
		return data, nil
	}
	return in.store.GetObject(ctx, h)
}

// accept checks a push before any of it is kept, and returns the objects to
// keep: those of the push the publication is made of.
func (s *Server) accept(ctx context.Context, p Project, head string, req pushRequest) (map[string][]byte, error) {
	for h, data := range req.Objects {
		if len(data) > maxObject {
			return nil, fmt.Errorf("object %s is too large", h)
		}
		if tig.HashOf(data) != h {
			return nil, fmt.Errorf("object %s does not match its hash", h)
		}
	}
	in := &incoming{push: req.Objects, store: p.Objects(), read: map[string]bool{}}
	if !tig.ValidHash(req.New) {
		return nil, fmt.Errorf("the new head %q is not a hash", req.New)
	}
	pub, err := LoadPublication(ctx, in, req.New)
	if err != nil {
		return nil, err
	}
	if pub.Parent != head {
		return nil, fmt.Errorf("the publication's parent is not the project's head")
	}
	// tig.ReadTree keeps the trees it has read, across projects, so every
	// tree object of the push is read here, from the push or this project.
	if err := readTrees(ctx, in, pub.Docs); err != nil {
		return nil, fmt.Errorf("the documents: %v", err)
	}
	docs, err := tig.ReadTree(ctx, in, pub.Docs)
	if err != nil {
		return nil, fmt.Errorf("the documents: %v", err)
	}
	texts := map[string][]byte{}
	for rel, e := range docs {
		if !tig.IsMarkdown(rel) {
			return nil, fmt.Errorf("%s is not a Markdown document: a hub keeps documents only", rel)
		}
		if texts[rel], err = in.GetObject(ctx, e.Hash); err != nil {
			return nil, fmt.Errorf("%s: %v", rel, err)
		}
	}
	v, err := loadChain(ctx, in, pub.Chain)
	if err != nil {
		return nil, err
	}
	if err := v.check(texts); err != nil {
		return nil, err
	}
	keep := map[string][]byte{}
	for h := range in.read {
		if data, ok := req.Objects[h]; ok {
			keep[h] = data
		}
	}
	return keep, nil
}

// check reports whether the chain is the chain of these documents: every
// document it names is there, and its blocks are what the documents split
// into, no more and no less.
func (v *ChainView) check(texts map[string][]byte) error {
	inChain := map[string]bool{}
	for _, doc := range v.Docs {
		if _, ok := texts[doc]; !ok {
			return fmt.Errorf("the chain names %s, which the push does not have", doc)
		}
		inChain[doc] = true
	}
	for rel := range texts {
		if !inChain[rel] && rel != "README.md" {
			return fmt.Errorf("%s is not a document of the chain", rel)
		}
	}
	want := map[string]string{} // block → its document
	for _, doc := range v.Docs {
		for _, b := range tig.ParseBlocks(doc, texts[doc]) {
			want[b.Hash] = doc
		}
	}
	for _, b := range v.Blocks {
		doc, ok := want[b.Hash]
		if !ok || doc != b.Path {
			return fmt.Errorf("the chain has a block %s the documents do not have", b.Hash)
		}
		delete(want, b.Hash)
	}
	for h, doc := range want {
		return fmt.Errorf("the chain lacks block %s of %s", h, doc)
	}
	return nil
}

// readTrees reads a tree object and every tree under it.
func readTrees(ctx context.Context, objects tig.Objects, hash string) error {
	raw, err := objects.GetObject(ctx, hash)
	if err != nil {
		return err
	}
	text, ok := strings.CutPrefix(string(raw), strings.TrimSuffix(tig.TreeHeader, "\n")+"\n")
	if !ok {
		return fmt.Errorf("%s is not a tree", hash)
	}
	for _, line := range strings.Split(text, "\n") {
		if sub, ok := strings.CutPrefix(line, "d "); ok {
			sub, _, _ = strings.Cut(sub, " ")
			if !tig.ValidHash(sub) {
				return fmt.Errorf("tree %s has an invalid hash %q", hash, sub)
			}
			if err := readTrees(ctx, objects, sub); err != nil {
				return err
			}
		}
	}
	return nil
}
