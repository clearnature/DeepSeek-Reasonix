package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"reasonix/internal/agent"
)

func writeSessionFile(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "test.events.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSessionLoadCacheHitAndMiss(t *testing.T) {
	dir := t.TempDir()
	path := writeSessionFile(t, dir, `{"type":"replace","messages":[]}`+"\n")
	c := newSessionLoadCache()
	if _, ok := c.get(path); ok {
		t.Fatal("empty cache must miss")
	}
	sess := agent.NewSession("sys")
	c.put(path, sess)
	got, ok := c.get(path)
	if !ok {
		t.Fatal("freshly put entry must hit")
	}
	if got != sess {
		t.Fatal("cache must return the stored session")
	}
}

func TestSessionLoadCacheInvalidatesOnFileChange(t *testing.T) {
	dir := t.TempDir()
	path := writeSessionFile(t, dir, "a")
	c := newSessionLoadCache()
	sess := agent.NewSession("sys")
	c.put(path, sess)
	// Rewrite the file: size/mtime change, cache must miss.
	if err := os.WriteFile(path, []byte("bb"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.get(path); ok {
		t.Fatal("stale entry must miss after file change")
	}
}

func TestSessionLoadCacheInvalidatesOnMtimeChange(t *testing.T) {
	dir := t.TempDir()
	path := writeSessionFile(t, dir, "same-size")
	c := newSessionLoadCache()
	c.put(path, agent.NewSession("sys"))
	// Same size, newer mtime: still a miss (file was rewritten).
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.get(path); ok {
		t.Fatal("entry must miss when mtime advances with same size")
	}
}

func TestSessionLoadCacheEvictsLeastRecent(t *testing.T) {
	dir := t.TempDir()
	paths := make([]string, sessionCacheMaxEntries+2)
	for i := range paths {
		paths[i] = filepath.Join(dir, "s"+string(rune('a'+i))+".events.jsonl")
		if err := os.WriteFile(paths[i], []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c := newSessionLoadCache()
	for i, p := range paths {
		c.put(p, agent.NewSession("sys"))
		_ = i
	}
	if _, ok := c.get(paths[0]); ok {
		t.Fatal("least-recent entry must be evicted")
	}
	if _, ok := c.get(paths[len(paths)-1]); !ok {
		t.Fatal("most-recent entry must survive within the cap")
	}
}

func TestSessionLoadCacheConcurrent(t *testing.T) {
	dir := t.TempDir()
	path := writeSessionFile(t, dir, "x")
	c := newSessionLoadCache()
	c.put(path, agent.NewSession("sys"))
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.get(path)
			}
		}()
	}
	wg.Wait()
	if _, ok := c.get(path); !ok {
		t.Fatal("concurrent reads must not lose the entry")
	}
}
