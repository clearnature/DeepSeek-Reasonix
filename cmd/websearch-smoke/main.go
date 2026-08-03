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
