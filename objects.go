package tig

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Objects stores content by its SHA-256 hash (lowercase hex), each content
// once. GetObject returns exactly the bytes that were put.
type Objects interface {
	PutObject(ctx context.Context, data []byte) (string, error)
	GetObject(ctx context.Context, hash string) ([]byte, error)
}

// FilePutter is an Objects that can store a file without reading it all into
// memory.
type FilePutter interface {
	PutFile(ctx context.Context, file string) (hash string, size int64, err error)
}

// ErrNotFound is returned for an object the store does not have.
var ErrNotFound = errors.New("tig: object not found")

// ValidHash reports whether h is a SHA-256 hash in lowercase hex. Every hash
// read from a record or a tree is checked, because a .tig can come from
// someone else's repository.
func ValidHash(h string) bool {
	if len(h) != 64 {
		return false
	}
	for i := 0; i < len(h); i++ {
		if c := h[i]; !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// HashOf is the hash an object with this content has.
func HashOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Dir is an object store in a folder: <dir>/<first two of the hash>/<hash>.
// Objects are kept as they are, not compressed: .tig goes into Git, and Git
// compresses and deltas plain objects far better than already compressed
// ones. An object an older store compressed (<hash>.z, zlib) is still read.
type Dir string

func (d Dir) path(h string) string { return filepath.Join(string(d), h[:2], h) }

// PutObject stores data and returns its hash.
func (d Dir) PutObject(_ context.Context, data []byte) (string, error) {
	h := HashOf(data)
	p := d.path(h)
	if _, err := os.Stat(p); err == nil {
		return h, nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	return h, writeAtomic(p, data, 0o644)
}

// PutFile stores a file's content, hashing it while it is copied.
func (d Dir) PutFile(_ context.Context, file string) (string, int64, error) {
	in, err := os.Open(file)
	if err != nil {
		return "", 0, err
	}
	defer in.Close()
	if err := os.MkdirAll(string(d), 0o755); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp(string(d), ".tmp-*")
	if err != nil {
		return "", 0, err
	}
	defer os.Remove(tmp.Name())
	hasher := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, hasher), in)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", 0, err
	}
	h := hex.EncodeToString(hasher.Sum(nil))
	p := d.path(h)
	if _, err := os.Stat(p); err == nil {
		return h, n, nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", 0, err
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		if _, serr := os.Stat(p); serr == nil {
			return h, n, nil // another process stored it first
		}
		return "", 0, err
	}
	return h, n, nil
}

// GetObject reads an object and checks that its content matches its hash.
func (d Dir) GetObject(_ context.Context, h string) ([]byte, error) {
	if !ValidHash(h) {
		return nil, fmt.Errorf("tig: invalid object hash %q", h)
	}
	data, err := os.ReadFile(d.path(h))
	if errors.Is(err, fs.ErrNotExist) {
		z, zerr := os.ReadFile(d.path(h) + ".z")
		if errors.Is(zerr, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, h)
		}
		if zerr != nil {
			return nil, zerr
		}
		data, err = inflate(z)
	}
	if err != nil {
		return nil, err
	}
	if HashOf(data) != h {
		return nil, fmt.Errorf("tig: object %s is corrupt: its content does not match its hash", h)
	}
	return data, nil
}

// Has reports whether the store has the object.
func (d Dir) Has(h string) bool {
	if !ValidHash(h) {
		return false
	}
	if _, err := os.Stat(d.path(h)); err == nil {
		return true
	}
	_, err := os.Stat(d.path(h) + ".z")
	return err == nil
}

func inflate(z []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(z))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// writeAtomic writes a file through a temporary file and a rename, so a
// reader never sees half of it and a symbolic link in its place is replaced,
// not followed.
func writeAtomic(file string, data []byte, perm fs.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(file), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), file); err != nil {
		// Windows cannot rename over a file another process holds open.
		if _, serr := os.Stat(file); serr == nil && bytes.Equal(readOrNil(file), data) {
			return nil
		}
		return err
	}
	return nil
}

func readOrNil(file string) []byte {
	b, _ := os.ReadFile(file)
	return b
}

func sortStrings(s []string) { sort.Strings(s) }
