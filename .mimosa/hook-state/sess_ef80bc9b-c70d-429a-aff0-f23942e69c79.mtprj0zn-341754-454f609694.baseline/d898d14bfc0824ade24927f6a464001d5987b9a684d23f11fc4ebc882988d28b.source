// Package main caches parsed session loads so reopening a large session
// (multi-MB event logs) skips the full JSON replay on every tab rebuild.

package main

import (
	"container/list"
	"os"
	"sync"
	"time"

	"reasonix/internal/agent"
)

// sessionLoadCache bounds how many parsed sessions are retained in memory.
const (
	sessionCacheMaxEntries = 8
	sessionCacheMaxBytes   = 256 << 20 // 256 MB
)

// cachedSession is one parsed session with the file stamp that proves the
// parse is still valid.
type cachedSession struct {
	path    string
	size    int64
	mtime   time.Time
	session *agent.Session
	bytes   int64
}

// sessionLoadCache is an LRU over parsed *agent.Session values. Entries are
// keyed by absolute path and invalidated when the file's size or mtime
// changes — any save rewrites the event log, so a stale parse can never be
// served for live content. Desktop resume paths consume the returned Session
// read-only and clone via sessionWithFreshSystemPrompt before handing it to a
// controller, so reuse across tab rebuilds is safe.
type sessionLoadCache struct {
	mu    sync.Mutex
	order *list.List // *list.Element of *cachedSession, most-recent first
	byKey map[string]*list.Element
	bytes int64
}

func newSessionLoadCache() *sessionLoadCache {
	return &sessionLoadCache{
		order: list.New(),
		byKey: make(map[string]*list.Element),
	}
}

// sessionLoadCacheSingleton is the process-wide parsed-session cache used by
// every tab resume path.
var sessionLoadCacheSingleton = newSessionLoadCache()

// loadResumableSessionCached resolves a session through the parse cache,
// falling back to a full decode (then caching it) on first open.
func loadResumableSessionCached(path string) (*agent.Session, error) {
	if cached, ok := sessionLoadCacheSingleton.get(path); ok {
		return cached, nil
	}
	loaded, err := agent.LoadSession(path)
	if err != nil {
		return nil, err
	}
	sessionLoadCacheSingleton.put(path, loaded)
	return loaded, nil
}

func (c *sessionLoadCache) get(path string) (*agent.Session, bool) {
	if c == nil {
		return nil, false
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.byKey[path]
	if !ok {
		return nil, false
	}
	entry := el.Value.(*cachedSession)
	if entry.size != st.Size() || !entry.mtime.Equal(st.ModTime()) {
		c.removeLocked(el)
		return nil, false
	}
	c.order.MoveToFront(el)
	return entry.session, true
}

func (c *sessionLoadCache) put(path string, session *agent.Session) {
	if c == nil || session == nil {
		return
	}
	st, err := os.Stat(path)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.byKey[path]; ok {
		c.removeLocked(el)
	}
	entry := &cachedSession{
		path:    path,
		size:    st.Size(),
		mtime:   st.ModTime(),
		session: session,
		bytes:   st.Size(),
	}
	c.byKey[path] = c.order.PushFront(entry)
	c.bytes += entry.bytes
	for len(c.byKey) > sessionCacheMaxEntries || c.bytes > sessionCacheMaxBytes {
		c.removeLocked(c.order.Back())
	}
}

func (c *sessionLoadCache) removeLocked(el *list.Element) {
	if el == nil {
		return
	}
	entry := el.Value.(*cachedSession)
	delete(c.byKey, entry.path)
	c.order.Remove(el)
	c.bytes -= entry.bytes
}
