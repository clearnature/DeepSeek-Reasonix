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
	Query         string   `json:"query"`
	QueryHash     string   `json:"query_hash"`
	AnswerSummary string   `json:"answer_summary"`
	KeyFacts      []string `json:"key_facts,omitempty"`
	Sources       []Source `json:"sources,omitempty"`
	TotalTokens   int      `json:"total_tokens,omitempty"`

	// TimeSensitive marks time-critical content (news, markets, live
	// events). FreshUntil bounds how long a hit may be served without a
	// refresh; after it, the entry is served as a stale fallback while an
	// incremental web_search refresh is triggered (retrieval tier P1).
	TimeSensitive bool `json:"time_sensitive,omitempty"`
	// FreshUntil is the freshness deadline for TimeSensitive entries. Zero
	// falls back to ExpiresAt. Non-sensitive entries ignore it (facts do
	// not go stale within the TTL).
	FreshUntil time.Time `json:"fresh_until,omitempty"`
	// Tier records the retrieval difficulty that produced this entry:
	// simple / general / complex / deep (retrieval tier P2 routing).
	Tier string `json:"tier,omitempty"`

	// ---- P4: 信息流模型（动态事件追踪，情报模式）----
	// EventChain links this entry to related event queries (初始事件→后续
	// 更新→关联事件)。元素是相关查询的原始文本。
	EventChain []string `json:"event_chain,omitempty"`
	// Confidence 是 0..1 的事件置信度，随增量更新上升、随冲突信号下降。
	Confidence float64 `json:"confidence,omitempty"`
	// UpdateCount 是该事件被增量刷新/更新的次数。
	UpdateCount int `json:"update_count,omitempty"`
	// ConflictDetected 标记多源矛盾（冲突信号），提示答案可能不稳定。
	ConflictDetected bool `json:"conflict_detected,omitempty"`
	// LastUpdatedAt 记录最近一次增量更新的时间。
	LastUpdatedAt time.Time `json:"last_updated_at,omitempty"`

	// SourceRequestID 是产生本缓存记录的上游请求标识（审计第四层）：用于
	// 追溯是哪一次检索/哪个 web_search 结果引入了内容，污染爆发时可
	// 按 request id 定位并回滚。
	SourceRequestID string `json:"source_request_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NeedsRefresh reports whether a cached hit should trigger an incremental
// update rather than being served as final. Time-sensitive entries whose
// FreshUntil (or ExpiresAt fallback) has passed need a refresh; static facts
// never do within their TTL. Callers still receive the stale entry, but
// should kick off a web_search refresh and merge.
func (e *KnowledgeEntry) NeedsRefresh(now time.Time) bool {
	if e == nil {
		return false
	}
	if !e.TimeSensitive {
		return false
	}
	deadline := e.FreshUntil
	if deadline.IsZero() {
		deadline = e.ExpiresAt
	}
	return !deadline.IsZero() && now.After(deadline)
}

// Source is one citation surfaced by the model for a web_search turn.
type Source struct {
	Title   string `json:"title,omitempty"`
	URL     string `json:"url,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	// Domain is the normalized registrable domain (e.g. reuters.com).
	// Populated by quality.go scoring when the entry is saved.
	Domain string `json:"domain,omitempty"`
	// Credibility is the P3 gate-3 quality score (0..1) assigned to this
	// source: whitelist + authority + cross-check + spam penalty.
	Credibility float64 `json:"credibility,omitempty"`
}

// DefaultKnowledgeTTL bounds how long a search result is treated as fresh.
// Web results go stale fast; 7 days matches the knowledge-extraction design.
const DefaultKnowledgeTTL = 7 * 24 * time.Hour

var errCacheDirUnavailable = errors.New("knowledge cache dir unavailable")

// knowledgeDirOverride lets tests redirect the cache root to an isolated
// temp dir so test runs never touch the real user cache (side-effect fix:
// the 100-round hammer test previously wiped ~/.cache/reasonix/websearch).
var knowledgeDirOverride string

// knowledgeDir is the per-user cache root for web_search knowledge entries.
func knowledgeDir() (string, error) {
	root := config.CacheDir()
	if knowledgeDirOverride != "" {
		root = knowledgeDirOverride
	}
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
// caching is best-effort and must never break the calling turn. The cache key
// is always re-derived from Query (never trusted from persisted JSON), so a
// malicious stored query_hash cannot escape the cache dir via path traversal.
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
	// Re-derive unconditionally: QueryHash in persisted JSON is advisory
	// metadata, not a path component source.
	e.QueryHash = KnowledgeHash(e.Query)
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
		if de.IsDir() || de.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, de.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var e KnowledgeEntry
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}
		if !e.ExpiresAt.IsZero() && now.After(e.ExpiresAt) {
			_ = os.Remove(path)
			continue
		}
		sim := NgramSimilarity(q, e.Query)
		// 主题一致性：Dice 相似度高但领域词无交集 = 误命中（如"霍尔木兹
		// 化肥"命中"霍尔木兹石油"）。无领域词的查询退化为纯相似度。
		if !topicsOverlap(q, e.Query) {
			sim = 0
		}
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

// ListKnowledge returns all unexpired cache entries (audit layer 4: daily
// sampling to re-check credibility, or operator inspection before rollback).
// Expired entries are pruned during the scan.
func ListKnowledge() []KnowledgeEntry {
	dir, err := knowledgeDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	now := time.Now()
	var out []KnowledgeEntry
	for _, de := range entries {
		if de.IsDir() || de.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, de.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var e KnowledgeEntry
		if err := json.Unmarshal(data, &e); err != nil {
			_ = os.Remove(path)
			continue
		}
		if !e.ExpiresAt.IsZero() && now.After(e.ExpiresAt) {
			_ = os.Remove(path)
			continue
		}
		out = append(out, e)
	}
	return out
}

// DeleteKnowledge removes cache entries matching pred. It returns the number
// of deleted entries and is the rollback primitive for pollution outbreaks
// (audit layer 4): delete by request id, by tier, or sweep everything.
func DeleteKnowledge(pred func(*KnowledgeEntry) bool) int {
	dir, err := knowledgeDir()
	if err != nil {
		return 0
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	deleted := 0
	for _, de := range entries {
		if de.IsDir() || de.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, de.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var e KnowledgeEntry
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}
		if pred(&e) {
			if err := os.Remove(path); err == nil {
				deleted++
			}
		}
	}
	return deleted
}

// domainVocabulary maps domain-significant words; two queries sharing any
// such word are the same topic, sharing none means the semantic hit is a
// false positive (fix: 语义缓存误命中不同主题). Order matters: more
// specific words first.
var domainVocabulary = map[string][]string{
	"农业": {"化肥", "农业", "小麦", "粮食", "收成", "农产品", "氮磷钾", "尿素", "大豆", "玉米"},
	"能源": {"石油", "港口", "吞吐量", "霍尔木兹", "马六甲", "原油", "天然气", "lng", "海峡"},
	"军事": {"战争", "冲突", "军事", "航母", "轰炸", "导弹", "制裁"},
	"经济": {"经济", "gdp", "通胀", "油价", "市场", "衰退", "贸易", "gdp"},
	"科技": {"ai", "芯片", "模型", "算法", "代码", "编程", "开源"},
	"气候": {"天气", "气候", "台风", "暴雨", "干旱", "气温", "降水"},
}

// ExtractTopics returns the domain-significant words present in a query.
func ExtractTopics(query string) []string {
	var out []string
	lower := strings.ToLower(query)
	for _, words := range domainVocabulary {
		for _, w := range words {
			if strings.Contains(lower, w) {
				out = append(out, w)
			}
		}
	}
	return out
}

// topicsOverlap reports whether a and b share at least one domain word.
// 语义：仅当查询本身带明确领域词时才要求交集（跨主题防误命中）；查询
// 无领域词（如英文缩写/代号）退化为纯相似度（保守命中，不拦截）。
func topicsOverlap(a, b string) bool {
	wordsA := ExtractTopics(a)
	if len(wordsA) == 0 {
		return true // 查询无领域信号：不拦截
	}
	wordsB := ExtractTopics(b)
	if len(wordsB) == 0 {
		return true // 缓存无领域标签：无法判断主题，保守命中
	}
	for _, wa := range wordsA {
		for _, wb := range wordsB {
			if wa == wb {
				return true
			}
		}
	}
	return false
}
