package responses

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"reasonix/internal/config"
)

// KnowledgeEntry is the persisted result of a server-side web_search turn.
// It lets a repeated query short-circuit the API: the search results are
// already distilled into AnswerSummary/KeyFacts/Sources by the model, so a
// cache hit answers instantly with zero token cost.
type KnowledgeEntry struct {
	Query         string    `json:"query"`
	QueryHash     string    `json:"query_hash"`
	AnswerSummary string    `json:"answer_summary"`
	KeyFacts      []string  `json:"key_facts,omitempty"`
	Sources       []Source  `json:"sources,omitempty"`
	TotalTokens   int       `json:"total_tokens,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// Source is one citation surfaced by the model for a web_search turn.
type Source struct {
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// DefaultKnowledgeTTL bounds how long a search result is treated as fresh.
// Web results go stale fast; 7 days matches the knowledge-extraction design.
const DefaultKnowledgeTTL = 7 * 24 * time.Hour

var errCacheDirUnavailable = errors.New("knowledge cache dir unavailable")

// knowledgeDir is the per-user cache root for web_search knowledge entries.
func knowledgeDir() (string, error) {
	root := config.CacheDir()
	if root == "" {
		return "", errCacheDirUnavailable
	}
	dir := filepath.Join(root, "websearch")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// KnowledgeHash derives the cache key for a query (SHA-256, stable across
// runs; the full hash is used so collisions are practically impossible).
func KnowledgeHash(query string) string {
	sum := sha256.Sum256([]byte(query))
	return hex.EncodeToString(sum[:])
}

// LoadKnowledge returns the cached entry for query when present and unexpired.
// The boolean reports a hit; err is nil on any non-fatal path (missing file,
// corrupt JSON, expired entry all count as a miss).
func LoadKnowledge(query string) (*KnowledgeEntry, bool) {
	dir, err := knowledgeDir()
	if err != nil {
		return nil, false
	}
	path := filepath.Join(dir, KnowledgeHash(query)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var e KnowledgeEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, false
	}
	if !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt) {
		_ = os.Remove(path)
		return nil, false
	}
	return &e, true
}

// SaveKnowledge persists a distilled search result. Failures are swallowed:
// caching is best-effort and must never break the calling turn.
func SaveKnowledge(e *KnowledgeEntry) {
	dir, err := knowledgeDir()
	if err != nil {
		return
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	if e.ExpiresAt.IsZero() {
		e.ExpiresAt = time.Now().Add(DefaultKnowledgeTTL)
	}
	if e.QueryHash == "" {
		e.QueryHash = KnowledgeHash(e.Query)
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return
	}
	path := filepath.Join(dir, e.QueryHash+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}
