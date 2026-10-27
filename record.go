package tig

import (
	"fmt"
	"strings"
	"time"
)

// Records are tig's text objects: a header line naming the kind and its
// version ("tig-step 1"), then one "key value" line per field. A key may
// repeat and the order is kept. Readers ignore keys they do not know, so a
// later feature can add fields (by convention starting with "x-") without a
// new version, and older readers still read the record.

type field struct{ key, value string }

type record struct {
	header string
	fields []field
}

func newRecord(header string) *record { return &record{header: header} }

// set adds a field; an empty value is left out.
func (r *record) set(key, value string) {
	if value != "" {
		r.fields = append(r.fields, field{key, oneLine(value)})
	}
}

func (r *record) setAll(key string, values []string) {
	for _, v := range values {
		r.set(key, v)
	}
}

func (r *record) setTime(key string, t time.Time) {
	if !t.IsZero() {
		r.set(key, t.UTC().Format(time.RFC3339Nano))
	}
}

func (r *record) setExtra(extra map[string][]string) {
	keys := make([]string, 0, len(extra))
	for k := range extra {
		if strings.HasPrefix(k, "x-") && !strings.ContainsAny(k, " \n\r") {
			keys = append(keys, k)
		}
	}
	sortStrings(keys)
	for _, k := range keys {
		r.setAll(k, extra[k])
	}
}

func (r *record) bytes() []byte {
	var b strings.Builder
	b.WriteString(r.header)
	b.WriteByte('\n')
	for _, f := range r.fields {
		b.WriteString(f.key)
		b.WriteByte(' ')
		b.WriteString(f.value)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

// parseRecord reads a record whose first line is header.
func parseRecord(data []byte, header string) (*record, error) {
	text, ok := strings.CutPrefix(string(data), header+"\n")
	if !ok {
		return nil, fmt.Errorf("tig: not a %s record", header)
	}
	r := &record{header: header}
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		k, v, _ := strings.Cut(line, " ")
		r.fields = append(r.fields, field{k, v})
	}
	return r, nil
}

func (r *record) get(key string) string {
	for _, f := range r.fields {
		if f.key == key {
			return f.value
		}
	}
	return ""
}

func (r *record) all(key string) []string {
	var out []string
	for _, f := range r.fields {
		if f.key == key {
			out = append(out, f.value)
		}
	}
	return out
}

func (r *record) time(key string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, r.get(key))
	return t
}

// extra is the fields a later feature added ("x-…"), kept for it to read.
func (r *record) extra() map[string][]string {
	var out map[string][]string
	for _, f := range r.fields {
		if strings.HasPrefix(f.key, "x-") {
			if out == nil {
				out = map[string][]string{}
			}
			out[f.key] = append(out[f.key], f.value)
		}
	}
	return out
}

func oneLine(s string) string { return strings.NewReplacer("\n", " ", "\r", " ").Replace(s) }
