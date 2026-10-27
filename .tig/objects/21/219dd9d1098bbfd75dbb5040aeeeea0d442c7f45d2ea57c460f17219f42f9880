package tig

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// A head (HEAD, WORKS, LINKS) moves by reading it and writing a new one on
// top. Two processes doing that at once would lose a record, so a lock file
// in .tig lets one at a time through. A lock left by a process that died is
// taken over after staleLock.
const (
	staleLock = 2 * time.Minute
	lockWait  = 30 * time.Second
)

var inProcess sync.Map // lock file -> *sync.Mutex

func lockRepo(ctx context.Context, dir string) (func(), error) { return lockFile(ctx, dir, "lock") }

// lockWork lets one change to the work in progress (.tig/WORK) through at a
// time. It is taken outside the head lock, never inside it.
func lockWork(ctx context.Context, dir string) (func(), error) {
	return lockFile(ctx, dir, "work.lock")
}

func lockFile(ctx context.Context, dir, name string) (func(), error) {
	p := filepath.Join(dir, name)
	m, _ := inProcess.LoadOrStore(p, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		mu.Unlock()
		return nil, err
	}
	deadline := time.Now().Add(lockWait)
	for {
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(p); mu.Unlock() }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			mu.Unlock()
			return nil, err
		}
		if st, err := os.Stat(p); err == nil && time.Since(st.ModTime()) > staleLock {
			os.Remove(p)
			continue
		}
		if time.Now().After(deadline) {
			mu.Unlock()
			return nil, fmt.Errorf("tig: %s is locked by another process (remove %s if no tig is running)", dir, p)
		}
		select {
		case <-ctx.Done():
			mu.Unlock()
			return nil, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}
