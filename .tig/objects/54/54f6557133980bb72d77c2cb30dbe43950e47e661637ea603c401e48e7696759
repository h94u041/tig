package hub

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/h94u041/tig"
)

//go:embed pages.html
var pagesFS embed.FS

var pages = template.Must(template.New("").Funcs(template.FuncMap{
	"short": func(h string) string {
		if len(h) > 12 {
			return h[:12]
		}
		return h
	},
	"when": func(t time.Time) string {
		if t.IsZero() {
			return "-"
		}
		return t.UTC().Format("2006-01-02 15:04 UTC")
	},
	"docURL": func(base, doc string) string { return base + "/doc/" + escapePath(doc) },
	"indent": func(level int) int { return level * 18 },
}).ParseFS(pagesFS, "pages.html"))

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := pages.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func notFound(w http.ResponseWriter, what string) {
	http.Error(w, what, http.StatusNotFound)
}

// projectPage is what every page of a project knows.
type projectPage struct {
	Owner, Name, Base string
	Head              Publication
	Chain             *ChainView
}

func (s *Server) loadProject(w http.ResponseWriter, r *http.Request) (*projectPage, Project, bool) {
	p, ok := s.Project(r.PathValue("owner"), r.PathValue("name"))
	if !ok {
		notFound(w, "no such project")
		return nil, p, false
	}
	pub, v, err := p.Latest(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, p, false
	}
	if v == nil {
		notFound(w, "no such project")
		return nil, p, false
	}
	return &projectPage{Owner: p.Owner, Name: p.Name, Base: "/" + p.Owner + "/" + p.Name, Head: pub, Chain: v}, p, true
}

type indexRow struct {
	Project
	Head  Publication
	Chain *ChainView
}

func (s *Server) pageIndex(w http.ResponseWriter, r *http.Request) {
	var rows []indexRow
	for _, p := range s.Projects() {
		pub, v, err := p.Latest(r.Context())
		if err != nil || v == nil {
			continue
		}
		rows = append(rows, indexRow{Project: p, Head: pub, Chain: v})
	}
	s.render(w, "index", rows)
}

// docRow is one document of the chain with how much of it is accepted.
type docRow struct {
	Path                         string
	Guidelines, Directions, Open int
}

func (s *Server) pageProject(w http.ResponseWriter, r *http.Request) {
	pp, _, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	var docs []docRow
	for _, d := range pp.Chain.Docs {
		row := docRow{Path: d}
		for _, b := range pp.Chain.Blocks {
			if b.Path != d {
				continue
			}
			switch b.Status {
			case tig.StatusGuideline:
				row.Guidelines++
			case tig.StatusDirection:
				row.Directions++
			case tig.StatusOpen:
				row.Open++
			}
		}
		docs = append(docs, row)
	}
	s.render(w, "project", struct {
		*projectPage
		Docs []docRow
	}{pp, docs})
}

type logRow struct {
	Publication
	Chain *ChainView
}

func (s *Server) pageLog(w http.ResponseWriter, r *http.Request) {
	pp, p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	var rows []logRow
	seen := map[string]bool{}
	for id := pp.Head.ID; id != "" && !seen[id] && len(rows) < 200; {
		seen[id] = true
		pub, err := LoadPublication(r.Context(), p.Objects(), id)
		if err != nil {
			break
		}
		v, _ := loadChain(r.Context(), p.Objects(), pub.Chain)
		rows = append(rows, logRow{Publication: pub, Chain: v})
		id = pub.Parent
	}
	s.render(w, "log", struct {
		*projectPage
		Rows []logRow
	}{pp, rows})
}

// blockRow is one block of a document as its page shows it.
type blockRow struct {
	tig.Block
	ID   string // the element's id
	View *BlockView
	HTML template.HTML
}

func (s *Server) pageDoc(w http.ResponseWriter, r *http.Request) {
	pp, p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	doc := r.PathValue("path")
	files, err := tig.ReadTree(r.Context(), p.Objects(), pp.Head.Docs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	e, ok := files[doc]
	if !ok {
		notFound(w, "no such document")
		return
	}
	data, err := p.Objects().GetObject(r.Context(), e.Hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	md := markdown(pp.Base, doc, files)
	var rows []blockRow
	for _, b := range tig.ParseBlocks(doc, data) {
		row := blockRow{Block: b, ID: "b-" + b.Hash[:12], View: pp.Chain.Block(b.Hash)}
		row.HTML = renderBlock(md, b)
		rows = append(rows, row)
	}
	s.render(w, "doc", struct {
		*projectPage
		Doc    string
		Blocks []blockRow
	}{pp, doc, rows})
}

// renderBlock is a block's heading and own text as HTML.
func renderBlock(md goldmark.Markdown, b tig.Block) template.HTML {
	src := b.Text
	if b.Level > 0 {
		src = strings.Repeat("#", b.Level) + " " + b.Title + "\n\n" + b.Text
	}
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(src))
	}
	// goldmark leaves raw HTML out (it is not built WithUnsafe), so the
	// output is safe to show as it is.
	return template.HTML(buf.String())
}

// markdown renders the blocks of one document: links to the chain's
// documents go to their pages on the hub; a link to a file the hub does not
// have becomes its text.
func markdown(base, doc string, files map[string]tig.Entry) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithASTTransformers(
			util.Prioritized(&links{base: base, doc: doc, files: files}, 100))),
	)
}

type links struct {
	base, doc string
	files     map[string]tig.Entry
}

func (l *links) Transform(node *ast.Document, _ text.Reader, _ parser.Context) {
	var unwrap []ast.Node
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var dest *[]byte
		switch n := n.(type) {
		case *ast.Link:
			dest = &n.Destination
		case *ast.Image:
			dest = &n.Destination
		default:
			return ast.WalkContinue, nil
		}
		target, keep := l.resolve(string(*dest))
		switch {
		case keep:
		case target != "":
			*dest = []byte(target)
		default:
			unwrap = append(unwrap, n)
		}
		return ast.WalkContinue, nil
	})
	for _, n := range unwrap {
		parent := n.Parent()
		for c := n.FirstChild(); c != nil; {
			next := c.NextSibling()
			parent.InsertBefore(parent, n, c)
			c = next
		}
		parent.RemoveChild(parent, n)
	}
}

// resolve is where a link of the document goes on the hub: keep for a link
// that leaves the project or stays in the page, the document's page for a
// document the hub has, and "" for a project file it does not have.
func (l *links) resolve(target string) (string, bool) {
	if strings.HasPrefix(target, "#") || strings.HasPrefix(target, "//") || strings.Contains(strings.SplitN(target, "/", 2)[0], ":") {
		return "", true
	}
	p, frag, _ := strings.Cut(target, "#")
	p, _, _ = strings.Cut(p, "?")
	if u, err := url.PathUnescape(p); err == nil {
		p = u
	}
	if strings.HasPrefix(p, "/") {
		p = path.Clean(strings.TrimPrefix(p, "/"))
	} else {
		p = path.Clean(path.Join(path.Dir(l.doc), p))
	}
	if _, ok := l.files[p]; !ok {
		return "", false
	}
	out := l.base + "/doc/" + escapePath(p)
	if frag != "" {
		out += "#" + frag
	}
	return out, false
}
