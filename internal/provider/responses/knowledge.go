package responses

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// DefaultSemanticThreshold is the character-set similarity cutoff for L2
// semantic hits. Near-synonym Chinese queries (今天北京天气 vs 北京今天天气)
// score ~0.5 under unigram Jaccard; unrelated queries score near 0. Tune per
// corpus: higher = fewer false positives but lower recall.
const DefaultSemanticThreshold = 0.35

// NgramSimilarity returns the Dice coefficient of character sets between a
// and b (0..1): 2·|A∩B| / (|A|+|B|). It is a cheap, local, dependency-free
// proxy for "semantic" matching on short Chinese/English queries. Unlike
// Jaccard it normalizes by the average length rather than the union, so a
// verbose query ("2026年8月3日北京天气怎么样") still scores well against a
// terse near-synonym ("北京今天天气如何") instead of being diluted by the
// extra characters. Word-order changes keep the same set, so near-synonym
// phrasings score high; punctuation and whitespace are ignored.
func NgramSimilarity(a, b string) float64 {
	ga := charSet(a)
	gb := charSet(b)
	if len(ga) == 0 || len(gb) == 0 {
		return 0
	}
	inter := 0
	for g := range ga {
		if gb[g] {
			inter++
		}
	}
	denom := len(ga) + len(gb)
	if denom == 0 {
		return 0
	}
	return 2 * float64(inter) / float64(denom)
}

func charSet(s string) map[rune]bool {
	out := make(map[rune]bool)
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r >= 0x4E00 && r <= 0x9FFF: // CJK unified ideographs
			out[r] = true
		}
	}
	return out
}

// LoadKnowledgeSemantic scans the cache for the unexpired entry whose query is
// most similar to q (L2 fallback after LoadKnowledge's exact hash miss). It
// returns the best match, its similarity, and true when that similarity meets
// or exceeds threshold. Zero-dependency local matching — no vector DB, no
// embedding API.
func LoadKnowledgeSemantic(q string, threshold float64) (*KnowledgeEntry, float64, bool) {
	dir, err := knowledgeDir()
	if err != nil {
		return nil, 0, false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, false
	}
	var (
		best    *KnowledgeEntry
		bestSim float64
	)
	now := time.Now()
	for _, de := range entries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, de.Name()))
		if err != nil {
			continue
		}
		var e KnowledgeEntry
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}
		if !e.ExpiresAt.IsZero() && now.After(e.ExpiresAt) {
			_ = os.Remove(filepath.Join(dir, de.Name()))
			continue
		}
		sim := NgramSimilarity(q, e.Query)
		if sim > bestSim {
			bestSim = sim
			best = &e
		}
	}
	if best != nil && bestSim >= threshold {
		return best, bestSim, true
	}
	return nil, 0, false
}
