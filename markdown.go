package tig

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Block is one text block of a Markdown document: a heading and its own text
// up to the next heading, or the document itself (level 0) with the text
// before its first heading.
//
// A block is identified by its content hash. The content is where it sits
// (the file and the headings above it) and what it says (its own text), so
// changing a block's text or moving it makes a new block; tig records which
// block it came from (see Lineage). A child's text is not part of its
// parent's hash, so changing a child leaves its parent as it was.
type Block struct {
	Hash    string   `json:"hash"`
	Path    string   `json:"path"`
	Parent  string   `json:"parent,omitempty"`
	Level   int      `json:"level"`
	Title   string   `json:"title"`
	Heading []string `json:"heading"` // the titles from the top of the document down to this block
	Anchor  string   `json:"anchor,omitempty"`
	Start   int      `json:"start"`  // first line, 1-based
	OwnEnd  int      `json:"ownEnd"` // last line of its own text
	End     int      `json:"end"`    // last line, its children included
	Text    string   `json:"text"`   // its own text, without the heading line
	Accept  []string `json:"accept,omitempty"`
	Links   []string `json:"links,omitempty"` // link targets in its own text, as written
}

// IsAcceptance reports whether the block is an acceptance section: a heading
// "驗收" or "Acceptance". Its commands are what accepts the part of the
// document it sits in.
func (b Block) IsAcceptance() bool { return isAcceptanceTitle(b.Title) }

func isAcceptanceTitle(t string) bool {
	t = strings.TrimRight(strings.TrimSpace(t), ":：")
	return t == "驗收" || strings.EqualFold(t, "acceptance")
}

// IsMarkdown reports whether a path is a Markdown document.
func IsMarkdown(path string) bool {
	low := strings.ToLower(path)
	return strings.HasSuffix(low, ".md") || strings.HasSuffix(low, ".markdown")
}

var (
	headingLine = regexp.MustCompile(`^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*$`)
	closingHash = regexp.MustCompile(`[ \t]+#+$`)
	fenceLine   = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})")
	inlineLink  = regexp.MustCompile(`\]\(\s*(<[^>]*>|[^)\s]+)(?:\s+["'(][^)]*)?\)`)
	refLink     = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:\s*(<[^>]*>|\S+)`)
	codeSpan    = regexp.MustCompile("`+[^`]*`+")
	listItem    = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+` + "`([^`]+)`")
	scheme      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
)

// ParseBlocks splits a Markdown document into its blocks, in document order;
// the first is the document itself.
func ParseBlocks(path string, content []byte) []Block {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	type section struct {
		level, start, ownEnd, end int
		title                     string
		parent                    int
	}
	secs := []section{{level: 0, start: 1, parent: -1}}
	fenced := make([]bool, len(lines)) // a line inside a code fence (or the fence itself)
	var fenceMark string
	for i, line := range lines {
		if fenceMark != "" {
			fenced[i] = true
			if m := fenceLine.FindStringSubmatch(line); m != nil && m[1][0] == fenceMark[0] && len(m[1]) >= len(fenceMark) && strings.TrimSpace(line[strings.Index(line, m[1])+len(m[1]):]) == "" {
				fenceMark = ""
			}
			continue
		}
		if m := fenceLine.FindStringSubmatch(line); m != nil {
			fenced[i], fenceMark = true, m[1]
			continue
		}
		m := headingLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		level := len(m[1])
		title := strings.TrimSpace(closingHash.ReplaceAllString(m[2], ""))
		if strings.Trim(m[2], "# \t") == "" {
			title = ""
		}
		parent := len(secs) - 1
		for parent > 0 && secs[parent].level >= level {
			parent = secs[parent].parent
		}
		secs = append(secs, section{level: level, start: i + 1, title: title, parent: parent})
	}
	// Own text runs to the next heading; the whole section to the next
	// heading at its level or above.
	for i := range secs {
		next := len(lines)
		if i+1 < len(secs) {
			next = secs[i+1].start - 1
		}
		secs[i].ownEnd = next
		secs[i].end = len(lines)
		for j := i + 1; j < len(secs); j++ {
			if secs[j].level <= secs[i].level {
				secs[i].end = secs[j].start - 1
				break
			}
		}
	}

	blocks := make([]Block, len(secs))
	seen := map[string]int{}
	anchors := map[string]int{}
	for i, s := range secs {
		b := Block{Path: path, Level: s.level, Title: s.title, Start: s.start, OwnEnd: s.ownEnd, End: s.end}
		if i == 0 {
			b.Start = 1
		}
		if s.parent >= 0 {
			b.Heading = append(append([]string{}, blocks[s.parent].Heading...), s.title)
		} else {
			b.Heading = []string{}
		}
		first := s.start // the heading line, not part of its own text
		if i == 0 {
			first = 0
		}
		var own []string
		for ln := first; ln < s.ownEnd && ln < len(lines); ln++ {
			own = append(own, strings.TrimRightFunc(lines[ln], unicode.IsSpace))
		}
		b.Text = trimBlankLines(own)
		ownFenced := fenced[min(first, len(fenced)):min(s.ownEnd, len(fenced))]
		b.Links = linksIn(own, ownFenced)
		if b.IsAcceptance() {
			b.Accept = commandsIn(own, ownFenced)
		}
		if i > 0 {
			slug := slugify(s.title)
			if n := anchors[slug]; n > 0 {
				b.Anchor = slug + "-" + strconv.Itoa(n)
			} else {
				b.Anchor = slug
			}
			anchors[slug]++
		}
		h := blockHash(path, b.Heading, b.Text, 0)
		if n := seen[h]; n > 0 {
			// Two blocks alike in the same place still need two names.
			h = blockHash(path, b.Heading, b.Text, n)
		}
		seen[blockHash(path, b.Heading, b.Text, 0)]++
		b.Hash = h
		if s.parent >= 0 {
			b.Parent = blocks[s.parent].Hash
		}
		blocks[i] = b
	}
	return blocks
}

func blockHash(path string, heading []string, text string, n int) string {
	h := sha256.New()
	h.Write([]byte("tig-block 1\x00" + path + "\x00" + strings.Join(heading, "\x1f") + "\x00" + text))
	if n > 0 {
		h.Write([]byte("\x00" + strconv.Itoa(n)))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func trimBlankLines(lines []string) string {
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// linksIn lists the link targets outside code.
func linksIn(lines []string, fenced []bool) []string {
	var out []string
	for i, line := range lines {
		if i < len(fenced) && fenced[i] {
			continue
		}
		if m := refLink.FindStringSubmatch(line); m != nil {
			out = append(out, strings.Trim(m[1], "<>"))
			continue
		}
		plain := codeSpan.ReplaceAllStringFunc(line, func(s string) string { return strings.Repeat(" ", len(s)) })
		for _, m := range inlineLink.FindAllStringSubmatch(plain, -1) {
			out = append(out, strings.Trim(m[1], "<>"))
		}
	}
	return out
}

// commandsIn lists an acceptance section's commands: each line of its code
// blocks, and each list item that starts with inline code.
func commandsIn(lines []string, fenced []bool) []string {
	var out []string
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if i < len(fenced) && fenced[i] {
			if fenceLine.MatchString(line) || t == "" || strings.HasPrefix(t, "#") {
				continue
			}
			out = append(out, strings.TrimPrefix(t, "$ "))
			continue
		}
		if m := listItem.FindStringSubmatch(line); m != nil {
			out = append(out, strings.TrimPrefix(strings.TrimSpace(m[1]), "$ "))
		}
	}
	return out
}

// slugify is the anchor GitHub gives a heading.
func slugify(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

// external reports whether a link target leaves the project (a URL, mail).
func external(target string) bool { return scheme.MatchString(target) && !isWindowsDrive(target) }

func isWindowsDrive(t string) bool { return len(t) > 1 && t[1] == ':' && unicode.IsLetter(rune(t[0])) }
