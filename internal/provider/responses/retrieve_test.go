package responses

import (
	"context"
	"os"
	"path/filepath"
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
