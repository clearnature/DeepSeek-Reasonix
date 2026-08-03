package responses

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRetrieveCacheHitZeroAPI(t *testing.T) {
	cleanKnowledgeCache(t)
	q := "2026年8月3日北京天气"
	SaveKnowledge(&KnowledgeEntry{Query: q, AnswerSummary: "晴朗", TimeSensitive: false})
	defer cleanupEntry(t, q)

	fetches := 0
	res, err := Retrieve(context.Background(), q, RetrieveOptions{},
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			fetches++
			return nil, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !res.FromCache || res.APIUsed {
		t.Fatalf("want cache hit zero API, got FromCache=%v APIUsed=%v", res.FromCache, res.APIUsed)
	}
	if fetches != 0 {
		t.Fatalf("fetch called %d times on cache hit", fetches)
	}
}

func TestRetrieveStaleServedThenRefresh(t *testing.T) {
	cleanKnowledgeCache(t)
	q := "美军航母抵达波斯湾最新进展"
	SaveKnowledge(&KnowledgeEntry{
		Query: q, AnswerSummary: "旧闻", TimeSensitive: true,
		FreshUntil: time.Now().Add(-time.Hour), // 已过期
	})
	defer cleanupEntry(t, q)

	res, err := Retrieve(context.Background(), q, RetrieveOptions{},
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			return &KnowledgeEntry{Query: query, AnswerSummary: "最新消息", KeyFacts: []string{"已抵达"}}, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !res.StaleServed || !res.Refreshed || !res.APIUsed {
		t.Fatalf("want stale-served + refresh, got %+v", res)
	}
	if res.Entry.AnswerSummary != "最新消息" {
		t.Fatalf("stale entry should be updated in place, got %q", res.Entry.AnswerSummary)
	}
	if res.Entry.UpdateCount != 1 {
		t.Fatalf("update count=%d want 1", res.Entry.UpdateCount)
	}
	// 刷新后 FreshUntil 前移，不再需要刷新
	if res.Entry.NeedsRefresh(time.Now()) {
		t.Fatal("refreshed entry must be fresh again")
	}
}

func TestRetrieveMissFetchAndQualityGate(t *testing.T) {
	cleanKnowledgeCache(t)
	q := "对比ChatGPT和DeepSeek"

	res, err := Retrieve(context.Background(), q, RetrieveOptions{},
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			if tier != TierComplex {
				t.Fatalf("heuristic should classify 对比 as complex, got %s", tier)
			}
			return &KnowledgeEntry{
				AnswerSummary: "对比结果",
				Sources: []Source{
					{URL: "https://reuters.com/a"},                 // whitelist
					{URL: "https://spam-site.com/b?utm_source=ad"}, // junk
				},
			}, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !res.APIUsed || res.FromCache {
		t.Fatalf("miss must fetch, got %+v", res)
	}
	if res.Tier != TierComplex {
		t.Fatalf("tier=%s want complex", res.Tier)
	}
	// 质量过滤：spam 剔除
	if len(res.Entry.Sources) != 1 || res.Entry.Sources[0].Domain != "reuters.com" {
		t.Fatalf("quality gate failed: %#v", res.Entry.Sources)
	}
	// 落盘后 L1 命中
	if _, hit := LoadKnowledge(q); !hit {
		t.Fatal("result should be persisted")
	}
	defer cleanupEntry(t, q)
}

func TestRetrieveForceRefresh(t *testing.T) {
	cleanKnowledgeCache(t)
	q := "世界杯冠军"
	SaveKnowledge(&KnowledgeEntry{Query: q, AnswerSummary: "旧结果", TimeSensitive: false})
	defer cleanupEntry(t, q)

	fetches := 0
	res, err := Retrieve(context.Background(), q, RetrieveOptions{ForceRefresh: true},
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			fetches++
			return &KnowledgeEntry{Query: query, AnswerSummary: "新结果"}, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if fetches != 1 || res.APIUsed != true || res.FromCache {
		t.Fatalf("force refresh must bypass cache, got %+v fetches=%d", res, fetches)
	}
	if res.Entry.AnswerSummary != "新结果" {
		t.Fatalf("want fresh result, got %q", res.Entry.AnswerSummary)
	}
}

func TestRetrieveNilFetch(t *testing.T) {
	if _, err := Retrieve(context.Background(), "q", RetrieveOptions{}, nil); err == nil {
		t.Fatal("nil fetch must error")
	}
}

func cleanupEntry(t *testing.T, q string) {
	t.Helper()
	dir := mustKnowledgeDir(t)
	_ = removeEntryFile(dir, KnowledgeHash(q))
}

func removeEntryFile(dir, hash string) error {
	return os.Remove(filepath.Join(dir, hash+".json"))
}

func TestRetrieveBlocksManipulatedContentFromCache(t *testing.T) {
	cleanKnowledgeCache(t)
	q := "这个事件是否危险"
	// Fetch 返回含恐慌/营销操纵的内容 → 必须透传但不落盘
	res, err := Retrieve(context.Background(), q, RetrieveOptions{},
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			return &KnowledgeEntry{
				Query:         query,
				AnswerSummary: "紧急预警！这场灾难即将失控，必须转发给所有人", // emotion + marketing
				KeyFacts:      []string{"最有效的应对方案，零风险"},
			}, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if res.Entry == nil {
		t.Fatal("manipulated content must still be served live")
	}
	if _, hit := LoadKnowledge(q); hit {
		t.Fatal("manipulated content must NOT be persisted to cache")
	}
}

func TestSaveKnowledgeIgnoresMaliciousQueryHash(t *testing.T) {
	cleanKnowledgeCache(t)
	dir := mustKnowledgeDir(t)
	defer func() { _ = os.RemoveAll(dir) }()

	// 恶意 query_hash（路径穿越尝试）必须被忽略，键始终从 Query 重新派生
	e := &KnowledgeEntry{Query: "安全查询", AnswerSummary: "x", QueryHash: "../../evil"}
	SaveKnowledge(e)
	// 正确键存在
	if _, hit := LoadKnowledge("安全查询"); !hit {
		t.Fatal("entry must be saved under re-derived hash")
	}
	// 恶意路径未被创建
	if _, err := os.Stat(filepath.Join(dir, "..", "..", "evil.json")); err == nil {
		t.Fatal("malicious query_hash must not create files outside cache dir")
	}
}

func TestRetrieveBypassProbability(t *testing.T) {
	cleanKnowledgeCache(t)
	q1 := "2026年8月3日北京天气怎么样？"
	SaveKnowledge(&KnowledgeEntry{Query: q1, AnswerSummary: "缓存快照", TimeSensitive: false})
	defer cleanupEntry(t, q1)

	// 近义查询，BypassProbability=1 → 必绕过（走 API 刷新）
	fetches := 0
	res, err := Retrieve(context.Background(), "北京今天天气如何", RetrieveOptions{BypassProbability: 1},
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			fetches++
			return &KnowledgeEntry{Query: q1, AnswerSummary: "实时刷新", KeyFacts: []string{"新数据"}}, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !res.Bypassed || !res.APIUsed || fetches != 1 {
		t.Fatalf("bypass=1 must refresh: Bypassed=%v APIUsed=%v fetches=%d", res.Bypassed, res.APIUsed, fetches)
	}
	if res.Entry.AnswerSummary != "实时刷新" {
		t.Fatalf("bypass must update entry, got %q", res.Entry.AnswerSummary)
	}

	// BypassProbability=0 → 永不绕过（默认省钱）
	fetches = 0
	res2, err := Retrieve(context.Background(), "北京今天天气如何", RetrieveOptions{},
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			fetches++
			return nil, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if res2.Bypassed || res2.APIUsed || fetches != 0 {
		t.Fatalf("bypass=0 must serve cache: Bypassed=%v APIUsed=%v fetches=%d", res2.Bypassed, res2.APIUsed, fetches)
	}
}

func TestRetrieveExactHitNeverBypasses(t *testing.T) {
	cleanKnowledgeCache(t)
	q := "完全相同的查询"
	SaveKnowledge(&KnowledgeEntry{Query: q, AnswerSummary: "快照", TimeSensitive: false})
	defer cleanupEntry(t, q)

	fetches := 0
	res, err := Retrieve(context.Background(), q, RetrieveOptions{BypassProbability: 1},
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			fetches++
			return nil, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	// L1 精确命中（sim=1.0）即使 BypassProbability=1 也不绕过
	if res.APIUsed || fetches != 0 {
		t.Fatalf("exact hit must never bypass: APIUsed=%v fetches=%d", res.APIUsed, fetches)
	}
}

func TestRetrievePanicModeAppendsReassurance(t *testing.T) {
	cleanKnowledgeCache(t)
	q := "今晚北京会地震吗？"
	defer cleanupEntry(t, q)

	// PanicMode 开启：答案追加安抚行（不拦截检索本身）
	res, err := Retrieve(context.Background(), q, RetrieveOptions{PanicMode: true},
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			return &KnowledgeEntry{Query: query, AnswerSummary: "未监测到异常地震活动"}, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !contains(res.Entry.AnswerSummary, "温馨提示") || !contains(res.Entry.AnswerSummary, "官方应急渠道") {
		t.Fatalf("panic mode must append reassurance, got %q", res.Entry.AnswerSummary)
	}
	// 检索答案本身未被拦截
	if !contains(res.Entry.AnswerSummary, "未监测到异常地震活动") {
		t.Fatal("original answer must be preserved")
	}
	// 落盘时也带安抚行（缓存命中同样生效）
	if e, hit := LoadKnowledge(q); !hit || !contains(e.AnswerSummary, "温馨提示") {
		t.Fatalf("cached panic answer must carry reassurance, got %q", e.AnswerSummary)
	}
}

func TestRetrievePanicModeOffByDefault(t *testing.T) {
	cleanKnowledgeCache(t)
	q := "明天会有海啸吗？"
	defer cleanupEntry(t, q)

	res, err := Retrieve(context.Background(), q, RetrieveOptions{}, // PanicMode 默认 false
		func(ctx context.Context, query string, tier RetrievalTier) (*KnowledgeEntry, error) {
			return &KnowledgeEntry{Query: query, AnswerSummary: "请关注官方预警"}, nil
		})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if contains(res.Entry.AnswerSummary, "温馨提示") {
		t.Fatalf("panic mode off must not append reassurance, got %q", res.Entry.AnswerSummary)
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
