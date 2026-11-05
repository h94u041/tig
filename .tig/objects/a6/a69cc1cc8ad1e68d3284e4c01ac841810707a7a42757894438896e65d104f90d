package main

import "strings"

// A Windows PATH value: folders joined by ";", as the person left it. These two
// change only the one folder, and keep the rest down to its spelling.

const pathSep = ";"

// sameFolder compares two PATH entries the way Windows would.
func sameFolder(a, b string) bool {
	clean := func(s string) string {
		return strings.TrimRight(strings.Trim(strings.TrimSpace(s), `"`), `\/`)
	}
	return strings.EqualFold(clean(a), clean(b))
}

// pathWith is the value with dir on it, at the end. It is already there: the
// value as it was, and false.
func pathWith(path, dir string) (string, bool) {
	for _, e := range strings.Split(path, pathSep) {
		if sameFolder(e, dir) {
			return path, false
		}
	}
	if path == "" {
		return dir, true
	}
	return strings.TrimSuffix(path, pathSep) + pathSep + dir, true
}

// pathWithout is the value with every spelling of dir taken out. It is not
// there: the value as it was, and false.
func pathWithout(path, dir string) (string, bool) {
	entries := strings.Split(path, pathSep)
	kept := make([]string, 0, len(entries))
	for _, e := range entries {
		if !sameFolder(e, dir) {
			kept = append(kept, e)
		}
	}
	if len(kept) == len(entries) {
		return path, false
	}
	return strings.Join(kept, pathSep), true
}
