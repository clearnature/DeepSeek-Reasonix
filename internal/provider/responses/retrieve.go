package responses

import (
	"context"
	"math/rand"
	"time"
)

// P5: agent 闭环检索协调器。把 P1-P4 串成完整流程：
//
//	门控0 本地命中（L1 精确 / L2 语义）
//	  ├─ 命中且新鲜（!NeedsRefresh）→ 直接返回，零 API
//	  ├─ 命中但时效过期 → 返回 stale 缓存 + 触发增量刷新（advanceEvent）
//	  └─ 未命中 → ClassifyTier 分级 → 全量检索
//	门控3 质量过滤（ScoreAndTagSources / FilterSources）
//	落盘（SaveKnowledge）→ 下次门控0 命中
//
// API 调用抽象为 FetchFunc，协调器本身零网络依赖（可单测）。

// RetrieveOptions controls one retrieval pass.
type RetrieveOptions struct {
	// TimeSensitive marks the query as time-critical (news/markets/events):
	// cached hits expire after FreshUntil and trigger incremental refresh.
	TimeSensitive bool
	// ForceRefresh skips the cache entirely and re-fetches (e.g. user asked
	// for the very latest, or a previous refresh was explicitly requested).
	ForceRefresh bool
	// MinCredibility is the gate-3 cutoff for kept sources; <=0 uses the
	// 0.5 default.
	MinCredibility float64
	// Tier overrides heuristic classification; empty uses ClassifyTier.
	Tier RetrievalTier
	// BypassProbability (0..1) is the defense-layer-3 anti-echo-chamber knob:
	// on a L2 semantic hit whose similarity is below 0.95, the cache is
	// bypassed with this probability and a live fetch refreshes the entry
	// (incremental update, so the answer stays current instead of being
	// repeatedly served from the same distilled snapshot). 0 disables
	// bypass (default, cost-first); 1 always refreshes near-synonyms.
	BypassProbability float64
}

// bypassThreshold is the L2 similarity above which a hit is treated as
// effectively identical (always served from cache regardless of the bypass
// probability — re-fetching an identical query wastes money for no signal).
const bypassThreshold = 0.95

// FetchFunc performs one real retrieval for query (web_search + json_schema
// extraction) and returns the distilled entry. tier is the classification the
// executor may use to pick round count.
type FetchFunc func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error)

// RetrieveResult reports how a pass resolved.
type RetrieveResult struct {
	Entry       *KnowledgeEntry // final answer (cache or fresh)
	FromCache   bool            // served from L1/L2 without API
	StaleServed bool            // stale cache served while refresh ran
	Bypassed    bool            // defense-layer-3 probabilistic bypass fired
	Refreshed   bool            // incremental refresh applied
	Tier        RetrievalTier   // difficulty used (heuristic or override)
	APIUsed     bool            // any fetch was invoked
}

// Retrieve runs the closed loop. fetch must be non-nil; a nil ctx uses
// context.Background.
func Retrieve(ctx context.Context, query string, opts RetrieveOptions, fetch FetchFunc) (*RetrieveResult, error) {
	if fetch == nil {
		return nil, errFetchFuncRequired
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	res := &RetrieveResult{}

	// 门控 0：本地命中（L1 精确，L2 语义兜底）
	if !opts.ForceRefresh {
		if e, hit := LoadKnowledge(query); hit {
			res.Entry = e
			res.FromCache = true
			res.Tier = tierOf(e)
			if e.NeedsRefresh(now) {
				// 时效过期：serve stale + 增量刷新（不阻塞主路径，结果合入缓存）
				res.StaleServed = true
				fresh, err := fetch(ctx, query, tierOf(e))
				if err != nil {
					return res, err // stale entry still returned via res.Entry
				}
				advanceEvent(e, now, fresh.KeyFacts, nil)
				mergeEntry(e, fresh)
				SaveKnowledge(e)
				res.Refreshed = true
				res.APIUsed = true
			}
			return res, nil
		}
		if e, sim, hit := LoadKnowledgeSemantic(query, DefaultSemanticThreshold); hit {
			res.Entry = e
			res.FromCache = true
			res.Tier = tierOf(e)
			stale := e.NeedsRefresh(now)
			// Defense layer 3 (anti-echo-chamber): a near-synonym hit below
			// the identity threshold is bypassed with the configured
			// probability so the cache is refreshed by live data instead of
			// repeatedly serving the same distilled snapshot.
			bypass := !stale && !opts.ForceRefresh &&
				opts.BypassProbability > 0 &&
				sim < bypassThreshold &&
				rand.Float64() < opts.BypassProbability
			if stale || bypass {
				res.StaleServed = stale
				res.Bypassed = bypass
				fresh, err := fetch(ctx, query, tierOf(e))
				if err != nil {
					return res, err
				}
				advanceEvent(e, now, fresh.KeyFacts, nil)
				mergeEntry(e, fresh)
				SaveKnowledge(e)
				res.Refreshed = true
				res.APIUsed = true
			}
			return res, nil
		}
	}

	// 未命中 / 强制刷新：分级 → 全量检索 → 质量过滤 → 落盘
	tier := opts.Tier
	if tier == "" {
		tier, _ = ClassifyTier(query)
	}
	res.Tier = tier

	fresh, err := fetch(ctx, query, tier)
	if err != nil {
		return res, err
	}
	if fresh == nil {
		return res, errEmptyFetchResult
	}
	if fresh.Query == "" {
		fresh.Query = query
	}
	if !fresh.TimeSensitive && opts.TimeSensitive {
		fresh.TimeSensitive = true
	}
	if fresh.Tier == "" {
		fresh.Tier = string(tier)
	}
	ScoreAndTagSources(fresh)
	fresh.Sources = FilterSources(fresh.Sources, opts.MinCredibility)
	if fresh.CreatedAt.IsZero() {
		fresh.CreatedAt = now
	}
	// Defense layer 2, check 1: engineered panic/marketing content must not
	// be persisted (cache would "fix" pollution and amplify it on later
	// L1/L2 hits). Score the distilled answer, not just source snippets.
	manip := emotionHits(fresh.AnswerSummary) + marketingHits(fresh.AnswerSummary)
	for _, f := range fresh.KeyFacts {
		manip += emotionHits(f) + marketingHits(f)
	}
	if manip > 0 {
		// Do not cache manipulated content; the caller still receives the
		// live answer, it just never poisons the cache.
		res.Entry = fresh
		res.APIUsed = true
		return res, nil
	}
	SaveKnowledge(fresh)
	res.Entry = fresh
	res.APIUsed = true
	return res, nil
}

// tierOf reads an entry's stored tier, defaulting to heuristic.
func tierOf(e *KnowledgeEntry) RetrievalTier {
	if e == nil || e.Tier == "" {
		return TierSimple
	}
	return RetrievalTier(e.Tier)
}

// mergeEntry folds fresh data into an existing stale entry (incremental
// update semantics): replace answer/sources, keep event-chain history.
func mergeEntry(dst, src *KnowledgeEntry) {
	if src == nil {
		return
	}
	if src.AnswerSummary != "" {
		dst.AnswerSummary = src.AnswerSummary
	}
	if len(src.KeyFacts) > 0 {
		dst.KeyFacts = src.KeyFacts
	}
	if len(src.Sources) > 0 {
		dst.Sources = src.Sources
	}
	if src.TotalTokens > 0 {
		dst.TotalTokens = src.TotalTokens
	}
	// Audit: a refresh supersedes the origin, so the provenance ID follows
	// the freshest request (rollback by request id must catch replaced
	// content, not just the original).
	if src.SourceRequestID != "" {
		dst.SourceRequestID = src.SourceRequestID
	}
}

var (
	errFetchFuncRequired = errCacheAPI("fetch function required")
	errEmptyFetchResult  = errCacheAPI("fetch returned nil entry")
)

type errCacheAPI string

func (e errCacheAPI) Error() string { return string(e) }
