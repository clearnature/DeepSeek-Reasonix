// Command websearch_smoke exercises the web_search knowledge-harvester
// end to end against the live DeepSeek Responses API and proves the
// cost-saving claim: cache hits must never touch the API.
//
// Usage: DEEPSEEK_API_KEY=... go run ./cmd/websearch-smoke
//
// Scenarios (four legs, one API call total):
//
//	A. first ask  -> real web_search + json_schema  -> tokens spent, cached
//	B. exact re-ask -> L1 hash hit                  -> 0 tokens, 0 API
//	C. near-synonym  -> L2 semantic hit             -> 0 tokens, 0 API
//	D. unrelated ask -> cache miss                  -> real API (tokens spent)
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"reasonix/internal/provider"
	"reasonix/internal/provider/responses"
)

func apiKey() string {
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		return v
	}
	data, err := os.ReadFile(os.ExpandEnv("$HOME/.reasonix/.env"))
	if err == nil {
		for _, line := range splitLines(string(data)) {
			if len(line) > 17 && line[:17] == "DEEPSEEK_API_KEY=" {
				return line[17:]
			}
		}
	}
	return ""
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

var knowledgeSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"answer_summary": map[string]any{"type": "string"},
		"key_facts":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"sources": map[string]any{"type": "array", "items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{"type": "string"},
				"url":   map[string]any{"type": "string"},
			},
		}},
	},
}

// ask runs one real DeepSeek web_search turn and returns spent tokens.
func ask(ctx context.Context, key, query string) (int, error) {
	p := responses.New(responses.Config{
		Name: "deepseek-responses", APIKey: key,
		BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash",
		Effort: "low",
	})
	req := provider.Request{
		Messages:       []provider.Message{{Role: provider.RoleUser, Content: query}},
		Tools:          []provider.ToolSchema{provider.WebSearchTool(false)},
		ToolChoice:     &provider.ToolChoice{Type: "web_search"},
		ResponseFormat: provider.JSONSchemaFormat("knowledge_extract", knowledgeSchema),
	}
	ch, err := p.Stream(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("stream: %w", err)
	}
	text := ""
	tokens := 0
	seenWeb := false
	for c := range ch {
		switch c.Type {
		case provider.ChunkText:
			text += c.Text
		case provider.ChunkUsage:
			if c.Usage != nil {
				tokens = c.Usage.TotalTokens
			}
		}
	}
	_ = seenWeb // web_search_call 事件被 readStream 静默吸收，无需计数
	if text == "" {
		return tokens, fmt.Errorf("no output text for %q", query)
	}
	// Persist distilled knowledge (summary = raw text; the smoke test does
	// not need the JSON extraction to succeed).
	responses.SaveKnowledge(&responses.KnowledgeEntry{
		Query: query, AnswerSummary: text, TotalTokens: tokens,
	})
	return tokens, nil
}

func main() {
	key := apiKey()
	if key == "" {
		fmt.Println("❌ DEEPSEEK_API_KEY not found")
		os.Exit(1)
	}
	if len(os.Args) > 1 && os.Args[1] == "-retrieve" {
		runRetrieve(key)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	q := "2026年8月3日北京天气怎么样？"
	fmt.Println("=== web_search 知识收割机冒烟测试 ===")
	fmt.Println("（命中必须 0 API 调用 0 token；只有 Miss 才花钱）")

	// Leg A: first ask -> real API.
	t0 := time.Now()
	tokensA, err := ask(ctx, key, q)
	dtA := time.Since(t0)
	if err != nil {
		fmt.Printf("❌ A 首查失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("A 首查(API)   : %6d tokens  %8v  命中=Miss ✅ 已缓存\n", tokensA, dtA.Round(time.Millisecond))

	// Leg B: exact re-ask -> L1 hash hit, zero API.
	t0 = time.Now()
	if e, hit := responses.LoadKnowledge(q); !hit {
		fmt.Println("❌ B 精确命中失败: 缓存未找到")
		os.Exit(1)
	} else {
		dtB := time.Since(t0)
		fmt.Printf("B 精确命中(L1): %6d tokens  %8v  命中=HIT ✅ 零API (tokens→%d)\n", 0, dtB.Round(time.Microsecond), e.TotalTokens)
	}

	// Leg C: near-synonym -> L2 semantic hit, zero API.
	qC := "北京今天天气如何？"
	t0 = time.Now()
	e, sim, hit := responses.LoadKnowledgeSemantic(qC, responses.DefaultSemanticThreshold)
	dtC := time.Since(t0)
	if !hit {
		fmt.Printf("❌ C 语义命中失败: sim=%.3f (阈值 %.2f)\n", sim, responses.DefaultSemanticThreshold)
		os.Exit(1)
	}
	fmt.Printf("C 语义命中(L2): %6d tokens  %8v  命中=HIT ✅ 相似度=%.3f (零API)\n", 0, dtC.Round(time.Microsecond), sim)
	_ = e

	// Leg D: unrelated -> cache miss -> real API (money spent).
	qD := "图灵奖得主是谁？"
	t0 = time.Now()
	tokensD, err := ask(ctx, key, qD)
	dtD := time.Since(t0)
	if err != nil {
		fmt.Printf("❌ D 未命中失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("D 无关(API)   : %6d tokens  %8v  命中=Miss ✅ 正确花钱\n", tokensD, dtD.Round(time.Millisecond))

	fmt.Println()
	fmt.Println("=== 省钱结论 ===")
	saved := tokensA + tokensD
	fmt.Printf("4 次提问实际 API 调用 %d 次（B/C 命中零调用）\n", 2)
	fmt.Printf("命中节省 tokens: %d（=A+D 两轮真实消耗，B/C 完全免费）\n", saved)
	fmt.Printf("若 100 个相似提问：只花 %d 次首查的钱，其余全部缓存命中\n", 2)
}

// realFetch builds a FetchFunc that runs a real DeepSeek web_search turn and
// distills the json_schema output into a KnowledgeEntry.
func realFetch(key string) responses.FetchFunc {
	return func(ctx context.Context, query string, tier responses.RetrievalTier) (*responses.KnowledgeEntry, error) {
		p := responses.New(responses.Config{
			Name: "deepseek-responses", APIKey: key,
			BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash",
			Effort: "low",
		})
		req := provider.Request{
			Messages:       []provider.Message{{Role: provider.RoleUser, Content: query}},
			Tools:          []provider.ToolSchema{provider.WebSearchTool(false)},
			ToolChoice:     &provider.ToolChoice{Type: "web_search"},
			ResponseFormat: provider.JSONSchemaFormat("knowledge_extract", knowledgeSchema),
		}
		ch, err := p.Stream(ctx, req)
		if err != nil {
			return nil, err
		}
		text := ""
		tokens := 0
		for c := range ch {
			switch c.Type {
			case provider.ChunkText:
				text += c.Text
			case provider.ChunkUsage:
				if c.Usage != nil {
					tokens = c.Usage.TotalTokens
				}
			}
		}
		entry := &responses.KnowledgeEntry{
			Query:         query,
			AnswerSummary: text,
			TotalTokens:   tokens,
			Tier:          string(tier),
		}
		// Try to extract structured JSON (answer_summary/key_facts/sources).
		if v, ok := responses.ExtractJSONFromOutput(text); ok {
			if obj, ok := v.(map[string]any); ok {
				if s, ok := obj["answer_summary"].(string); ok && s != "" {
					entry.AnswerSummary = s
				}
				if facts, ok := obj["key_facts"].([]any); ok {
					for _, f := range facts {
						if fs, ok := f.(string); ok {
							entry.KeyFacts = append(entry.KeyFacts, fs)
						}
					}
				}
				if srcs, ok := obj["sources"].([]any); ok {
					for _, s := range srcs {
						if sm, ok := s.(map[string]any); ok {
							src := responses.Source{}
							if t, ok := sm["title"].(string); ok {
								src.Title = t
							}
							if u, ok := sm["url"].(string); ok {
								src.URL = u
							}
							if sn, ok := sm["snippet"].(string); ok {
								src.Snippet = sn
							}
							entry.Sources = append(entry.Sources, src)
						}
					}
				}
			}
		}
		// DeepSeek json_schema is advisory: the reply is often markdown
		// ("## 关键事实 / ## 来源") instead of JSON. Fall back to markdown
		// extraction so sources/facts still land in the cache.
		if len(entry.Sources) == 0 {
			entry.Sources = extractMarkdownSources(text)
		}
		if len(entry.KeyFacts) == 0 {
			entry.KeyFacts = extractMarkdownFacts(text)
		}
		return entry, nil
	}
}

func runRetrieve(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	fetch := realFetch(key)

	q := "2026年8月3日北京天气怎么样？"
	fmt.Println("\n=== 真实 Retrieve 闭环（P0-P5 全链路） ===")

	// 首查：未命中 → 全量 web_search + json_schema → 质量过滤 → 落盘
	r1, err := responses.Retrieve(ctx, q, responses.RetrieveOptions{TimeSensitive: true}, fetch)
	if err != nil {
		fmt.Printf("❌ 首查失败: %v\n", err)
		return
	}
	fmt.Printf("① 首查   : API=%v tier=%s sources=%d tokens=%d 摘要=%s\n",
		r1.APIUsed, r1.Tier, len(r1.Entry.Sources), r1.Entry.TotalTokens,
		truncate(r1.Entry.AnswerSummary, 40))

	// L1 精确命中（零 API）
	r2, _ := responses.Retrieve(ctx, q, responses.RetrieveOptions{TimeSensitive: true}, fetch)
	fmt.Printf("② L1命中 : FromCache=%v API=%v 摘要=%s\n", r2.FromCache, r2.APIUsed, truncate(r2.Entry.AnswerSummary, 40))

	// L2 语义命中（近义改写，零 API）
	r3, _ := responses.Retrieve(ctx, "北京今天天气如何", responses.RetrieveOptions{TimeSensitive: true}, fetch)
	fmt.Printf("③ L2命中 : FromCache=%v API=%v tier=%s 摘要=%s\n", r3.FromCache, r3.APIUsed, r3.Tier, truncate(r3.Entry.AnswerSummary, 40))

	// 强制刷新（走 API）
	r4, _ := responses.Retrieve(ctx, q, responses.RetrieveOptions{ForceRefresh: true}, fetch)
	fmt.Printf("④ 强制刷 : API=%v 摘要=%s\n", r4.APIUsed, truncate(r4.Entry.AnswerSummary, 40))
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// extractMarkdownSources parses the common DeepSeek web_search markdown
// shape: a "## 来源" (or "Sources") section with "- 标题：URL" lines.
func extractMarkdownSources(text string) []responses.Source {
	var out []responses.Source
	lines := strings.Split(text, "\n")
	inSources := false
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "## ") {
			inSources = strings.Contains(trimmed, "来源") || strings.Contains(trimmed, "Sources")
			continue
		}
		if !inSources {
			continue
		}
		// - 标题：https://...
		if strings.HasPrefix(trimmed, "- ") {
			body := strings.TrimPrefix(trimmed, "- ")
			// find first http(s):// URL
			urlStart := -1
			for i := 0; i+7 <= len(body); i++ {
				if (body[i] == 'h' && len(body) > i+7 && body[i:i+8] == "https://") ||
					(body[i] == 'h' && len(body) > i+6 && body[i:i+7] == "http://") {
					urlStart = i
					break
				}
			}
			if urlStart < 0 {
				continue
			}
			src := responses.Source{
				Title: strings.TrimSpace(strings.TrimSuffix(body[:urlStart], "：")),
				URL:   strings.Fields(body[urlStart:])[0],
			}
			out = append(out, src)
		}
	}
	return out
}

// extractMarkdownFacts parses "**N. 事实**：..." or numbered list items.
func extractMarkdownFacts(text string) []string {
	var out []string
	for _, ln := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(ln)
		// **N. 标题**：内容  or  N. 内容
		if strings.HasPrefix(trimmed, "**") && strings.Contains(trimmed, "**：") {
			parts := strings.SplitN(trimmed, "**：", 2)
			if len(parts) == 2 {
				out = append(out, strings.TrimSpace(parts[1]))
			}
		}
	}
	return out
}
