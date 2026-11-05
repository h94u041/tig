package tig

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// A line of a document is the unit tig marks, the way Git's unit is a line of
// code. The mark is written at the end of the line and says which text the
// ledger knows the line by:
//
//	The user can delete their account for good. <tig l="c3d4e5"/>
//
// What the line means is not in the document: intents live in the ledger (see
// ledger.go), each with ranges of a line, so one piece of text can belong to
// any number of intents. Change the line and it is a new line: tig works out
// which of its intents the change touched, and only those are asked about
// again.
//
// The tag is valid HTML, MDX and CommonMark inline HTML, so renderers show the
// text and drop the tag. Attributes tig does not know are ignored.

const (
	minLineShort   = 6
	minIntentShort = 4
	minEventShort  = 4
)

var (
	lineMarkPattern = regexp.MustCompile(`[ \t]*<tig(?:[ \t]+[A-Za-z][A-Za-z0-9_-]*="[^"\n]*")*[ \t]*/>`)
	lineAttr        = regexp.MustCompile(`([A-Za-z][A-Za-z0-9_-]*)="([^"\n]*)"`)
	linePrefix      = regexp.MustCompile(`^[ \t]*(?:>[ \t]*|#{1,6}[ \t]+|(?:[-*+]|[0-9]+[.)])[ \t]+(?:\[[ xX]\][ \t]+)?)`)
	tableDivider    = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(?:\|\s*:?-+:?\s*)*\|?\s*$`)
	lineTail        = regexp.MustCompile(`(?:\\)?[ \t\r]*$`)
)

// LineMark is a mark as a document holds it.
type LineMark struct {
	Hash string // the line's hash, shortened, as written
	N    int    // which of the lines with this text; 1 when not written
}

// readLineMark finds a line's mark. Marks quoted as inline code do not count.
func readLineMark(line string) (LineMark, bool) {
	masked := maskCode(line)
	locs := lineMarkPattern.FindAllStringIndex(masked, -1)
	for i := len(locs) - 1; i >= 0; i-- {
		m := LineMark{N: 1}
		for _, a := range lineAttr.FindAllStringSubmatch(line[locs[i][0]:locs[i][1]], -1) {
			switch a[1] {
			case "l":
				m.Hash = a[2]
			case "n":
				if n, err := strconv.Atoi(a[2]); err == nil && n > 0 {
					m.N = n
				}
			}
		}
		if m.Hash != "" {
			return m, true
		}
	}
	return LineMark{}, false
}

// stripLineMarks takes the marks out of a line, with the space in front of them.
func stripLineMarks(line string) string {
	locs := lineMarkPattern.FindAllStringIndex(maskCode(line), -1)
	if locs == nil {
		return line
	}
	var b strings.Builder
	end := 0
	for _, l := range locs {
		b.WriteString(line[end:l[0]])
		end = l[1]
	}
	b.WriteString(line[end:])
	return b.String()
}

// stripMarks is a line without any mark, old or new.
func stripMarks(line string) string { return stripLineMarks(stripLegacyMarks(line)) }

// setLineMark writes a line's mark, replacing the one it has. In a table it
// goes inside the last cell; before a hard line break it goes in front of the
// break, which has to stay last.
func setLineMark(line string, m LineMark) string {
	tag := `<tig l="` + m.Hash + `"/>`
	if m.N > 1 {
		tag = `<tig l="` + m.Hash + `" n="` + strconv.Itoa(m.N) + `"/>`
	}
	line = stripLineMarks(line)
	cr := ""
	if strings.HasSuffix(line, "\r") {
		line, cr = line[:len(line)-1], "\r"
	}
	if tableLine.MatchString(line) {
		if t := strings.TrimRight(line, " \t"); strings.HasSuffix(t, "|") {
			body := strings.TrimRight(t[:len(t)-1], " \t")
			return body + " " + tag + " |" + cr
		}
	}
	at := lineTail.FindStringIndex(line)[0]
	body, tail := strings.TrimRight(line[:at], " \t"), line[at:]
	if body == "" {
		return tag + tail + cr
	}
	return body + " " + tag + tail + cr
}

// LineText is what a line says: the text its hash is taken from. The mark and
// the space at the end do not count, nor does what only places the line (its
// indent, the # of a heading, a quote's >, a list's marker or checkbox), so
// changing those is not changing the intent. The rest is taken as written.
func LineText(line string) string {
	text := strings.TrimRightFunc(stripMarks(strings.TrimRight(line, "\r\n")), unicode.IsSpace)
	for {
		loc := linePrefix.FindStringIndex(text)
		if loc == nil || loc[1] == 0 {
			return text
		}
		text = text[loc[1]:]
	}
}

// LineHash is the hash of what a line says: an object's hash, so the text is
// kept as an object and comes back by this hash.
func LineHash(text string) string { return HashOf([]byte(text)) }

// isTrackedLine reports whether a line can hold intents: not blank, and not the
// divider of a table.
func isTrackedLine(line string) bool {
	text := LineText(line)
	return text != "" && !tableDivider.MatchString(text)
}

// A reference names anything tig keeps. IDs say who something is and never
// change; whatever changes is in events, so a reference stays good.
//
//	reference = intent | line | work | event
//	intent    = "i" code [ "#" revision ] [ "@" line ]
//	line      = "l" code [ "." occurrence ] [ ":" from "-" to ]
//	work      = code
//	event     = "ev" code
//
// Ranges count code points from 0 and leave out their end, and they mean
// something only with the line they are of.

// What a reference is to.
const (
	RefIntent = "intent"
	RefLine   = "line"
	RefWork   = "work"
	RefEvent  = "event"
)

// Ref is a parsed reference.
type Ref struct {
	Kind     string `json:"kind"`
	Code     string `json:"code"`               // the hash, or the start of it, of what it names
	Revision int    `json:"revision,omitempty"` // an intent's revision; 0 for the current one
	Line     string `json:"line,omitempty"`     // the line's hash, or the start of it
	N        int    `json:"n,omitempty"`        // which of the lines with this text; 0 when not said
	HasRange bool   `json:"hasRange,omitempty"`
	From     int    `json:"from,omitempty"`
	To       int    `json:"to,omitempty"`
}

var (
	intentRef = regexp.MustCompile(`^i([0-9a-f]{4,64})(?:#([0-9]+))?(?:@(l.+))?$`)
	lineRef   = regexp.MustCompile(`^l([0-9a-f]{6,64})(?:\.([0-9]+))?(?::([0-9]+)-([0-9]+))?$`)
	workRef   = regexp.MustCompile(`^[0-9a-f]{4,64}$`)
	eventRef  = regexp.MustCompile(`^ev([0-9a-f]{4,64})$`)
)

// ParseRef reads a reference.
func ParseRef(s string) (Ref, error) {
	bad := func() (Ref, error) { return Ref{}, fmt.Errorf("tig: %q is not a reference", s) }
	if m := intentRef.FindStringSubmatch(s); m != nil {
		r := Ref{Kind: RefIntent, Code: m[1]}
		if m[2] != "" {
			n, err := strconv.Atoi(m[2])
			if err != nil || n == 0 {
				return bad()
			}
			r.Revision = n
		}
		if m[3] != "" {
			l, err := ParseRef(m[3])
			if err != nil {
				return bad()
			}
			r.Line, r.N, r.HasRange, r.From, r.To = l.Code, l.N, l.HasRange, l.From, l.To
		}
		return r, nil
	}
	if m := lineRef.FindStringSubmatch(s); m != nil {
		r := Ref{Kind: RefLine, Code: m[1]}
		var err error
		if m[2] != "" {
			if r.N, err = strconv.Atoi(m[2]); err != nil || r.N == 0 {
				return bad()
			}
		}
		if m[3] != "" {
			r.HasRange = true
			if r.From, err = strconv.Atoi(m[3]); err != nil {
				return bad()
			}
			if r.To, err = strconv.Atoi(m[4]); err != nil || r.To < r.From {
				return bad()
			}
		}
		return r, nil
	}
	if m := eventRef.FindStringSubmatch(s); m != nil {
		return Ref{Kind: RefEvent, Code: m[1]}, nil
	}
	if workRef.MatchString(s) {
		return Ref{Kind: RefWork, Code: s}, nil
	}
	return bad()
}

// String writes a reference back the way ParseRef reads it.
func (r Ref) String() string {
	lineOf := func(code string) string {
		s := "l" + code
		if r.N > 0 {
			s += "." + strconv.Itoa(r.N)
		}
		if r.HasRange {
			s += ":" + strconv.Itoa(r.From) + "-" + strconv.Itoa(r.To)
		}
		return s
	}
	switch r.Kind {
	case RefIntent:
		s := "i" + r.Code
		if r.Revision > 0 {
			s += "#" + strconv.Itoa(r.Revision)
		}
		if r.Line != "" {
			s += "@" + lineOf(r.Line)
		}
		return s
	case RefLine:
		return lineOf(r.Code)
	case RefEvent:
		return "ev" + r.Code
	}
	return r.Code
}

// carry is where a range of the old text of a line is in its new text. The
// part of the line that changed is what is left between what the two have in
// common at the start and at the end: a range wholly before it stays, one
// wholly after it moves, and one that meets it was touched and grows to take
// in the change. A range of the whole line is touched by any change: what was
// added at either end is in the same line, and whoever meant the line has to
// say if they still do. Several changes in a line count as one big change, so
// this asks too often rather than too little.
func carry(oldText, newText string, from, to int) (newFrom, newTo int, touched bool) {
	o, n := []rune(oldText), []rune(newText)
	p := 0
	for p < len(o) && p < len(n) && o[p] == n[p] {
		p++
	}
	s := 0
	for s < len(o)-p && s < len(n)-p && o[len(o)-1-s] == n[len(n)-1-s] {
		s++
	}
	oldEnd, newEnd, delta := len(o)-s, len(n)-s, len(n)-len(o)
	whole := from == 0 && to >= len(o)
	switch {
	case string(o) == string(n):
		return from, to, false
	case !whole && to <= p:
		return from, to, false
	case !whole && from >= oldEnd:
		return from + delta, to + delta, false
	}
	return min(from, p), min(max(to+delta, newEnd), len(n)), true
}

// sliceRunes is a range of text, counted in code points.
func sliceRunes(text string, from, to int) string {
	r := []rune(text)
	from, to = min(max(from, 0), len(r)), min(max(to, 0), len(r))
	if from >= to {
		return ""
	}
	return string(r[from:to])
}
