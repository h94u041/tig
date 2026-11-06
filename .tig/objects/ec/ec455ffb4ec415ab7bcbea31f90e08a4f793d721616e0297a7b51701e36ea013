package main

import (
	"bytes"
	"io"
)

// cleanWriter leaves out control characters. What tigwx prints can come from
// someone else's documents and records, and a terminal obeys what it is sent:
// an escape sequence in a sentence could rewrite what is on the screen. JSON
// already escapes them, and a program reading it wants the data as it is, so it
// passes through.
type cleanWriter struct {
	w    io.Writer
	json *bool
}

func (c cleanWriter) Write(p []byte) (int, error) {
	if (c.json != nil && *c.json) || !hasControl(p) {
		return c.w.Write(p)
	}
	if _, err := c.w.Write(stripControls(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

func hasControl(p []byte) bool {
	for i, b := range p {
		switch {
		case b == '\n' || b == '\t':
		case b == '\r':
			if i+1 < len(p) && p[i+1] != '\n' || i+1 == len(p) {
				return true
			}
		case b < 0x20 || b == 0x7f:
			return true
		case b == 0xc2 && i+1 < len(p) && p[i+1] >= 0x80 && p[i+1] <= 0x9f:
			return true
		}
	}
	return false
}

// stripControls is p without C0 and C1 control characters, keeping tabs and line
// ends. A carriage return with no line feed moves the cursor back over what is
// already on the line, so it goes too.
func stripControls(p []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(p))
	for i := 0; i < len(p); i++ {
		b := p[i]
		switch {
		case b == '\n' || b == '\t':
			out.WriteByte(b)
		case b == '\r':
			if i+1 < len(p) && p[i+1] == '\n' {
				out.WriteByte(b)
			}
		case b < 0x20 || b == 0x7f:
		case b == 0xc2 && i+1 < len(p) && p[i+1] >= 0x80 && p[i+1] <= 0x9f:
			i++
		default:
			out.WriteByte(b)
		}
	}
	return out.Bytes()
}
