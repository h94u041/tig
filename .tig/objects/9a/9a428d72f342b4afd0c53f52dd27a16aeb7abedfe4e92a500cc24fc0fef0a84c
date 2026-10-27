package tig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// The stat cache: each file's size, modification time, executable bit and
// content hash at the last scan, so a scan reads only files that changed.
// It is local (not in Git); losing it only costs a full read.

type statEntry struct {
	Size int64  `json:"s"`
	Mod  int64  `json:"m"`
	Exec bool   `json:"x,omitempty"`
	Hash string `json:"h"`
}

type statCache struct {
	file    string
	entries map[string]statEntry
	dirty   bool
}

var statCacheMu sync.Mutex

func openStatCache(dir string) *statCache {
	statCacheMu.Lock()
	c := &statCache{entries: map[string]statEntry{}}
	if dir == "" {
		return c
	}
	c.file = filepath.Join(dir, "stat.json")
	if raw, err := os.ReadFile(c.file); err == nil {
		json.Unmarshal(raw, &c.entries)
	}
	return c
}

func (c *statCache) get(rel string) (statEntry, bool) {
	e, ok := c.entries[rel]
	return e, ok
}

func (c *statCache) put(rel string, e statEntry) {
	if c.entries[rel] != e {
		c.entries[rel] = e
		c.dirty = true
	}
}

func (c *statCache) drop(rel string) {
	if _, ok := c.entries[rel]; ok {
		delete(c.entries, rel)
		c.dirty = true
	}
}

func (c *statCache) keepOnly(files map[string]Entry) {
	for rel := range c.entries {
		if _, ok := files[rel]; !ok {
			delete(c.entries, rel)
			c.dirty = true
		}
	}
}

func (c *statCache) close() {
	defer statCacheMu.Unlock()
	if c.file == "" || !c.dirty {
		return
	}
	raw, err := json.Marshal(c.entries)
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(c.file), 0o755) == nil {
		writeAtomic(c.file, raw, 0o600)
	}
}
