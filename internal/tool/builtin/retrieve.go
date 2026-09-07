package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"reasonix/internal/config"
	"reasonix/internal/provider"
	"reasonix/internal/provider/responses"
	"reasonix/internal/tool"
)

func init() { tool.RegisterBuiltin(retrieveInfo{}) }

// retrieveInfo exposes the knowledge-cache lookup as a model-visible tool so
// conversational auto-retrieval (2026-08-03 design) works: the model can
// consult previously distilled web_search results with zero cost and zero
// network. On a cache miss it reuses the system's deepseek-responses
// provider pipeline (the API key the user already configured for the app) —
// no separate key is required. Web fetch is gated by the session grant +
// cooldown policy; without a configured deepseek-responses provider it
// returns a needs_grant notice instead (§9 no-silent-web guardrail).
type retrieveInfo struct{}

func (retrieveInfo) Name() string { return "retrieve_info" }

func (retrieveInfo) Description() string {
	return "深度研究检索系统（比 web_search / web_fetch 更高级的整合入口）：对一个研究问题执行多级检索并产出综合答案。命中本地知识缓存（此前研究蒸馏落盘）时零成本返回，含关键事实与来源；未命中时自动经 deepseek-responses 深度研究管道联网检索（分级深入、多源综合）并蒸馏记忆供后续复用。返回结构：综合答案 + 关键事实列表 + 来源（标题/URL）+ 时效标注。适合研究型、需要权威多源信息的问题；实时性极高的突发信息用 web_search；抓取具体页面用 web_fetch。未配置检索凭据时返回 needs_grant，请用 ask 工具征询用户是否允许联网检索。"
}

func (retrieveInfo) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "query":{"type":"string","description":"要查询的问题（与之前的检索问题相同或语义相近）"}
},
"required":["query"]
}`)
}

func (retrieveInfo) ReadOnly() bool { return true }

func (retrieveInfo) SnipHint() tool.SnipHint {
	return tool.SnipHint{Head: 60, Tail: 10, HeadChars: 6000, TailChars: 1500}
}

// Execute runs a policy-gated lookup. The zero policy (local cache only, no
// web) is the safe default: this tool never triggers a network fetch. A
// blocked notice tells the model the answer needs a web refresh instead of
// silently going online.
func (retrieveInfo) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("retrieve_info: parse args: %w", err)
	}
	p.Query = strings.TrimSpace(p.Query)
	if p.Query == "" {
		return "", fmt.Errorf("retrieve_info: query is required")
	}

	// 会话策略：配置了 deepseek-responses 即视为会话级联网授权（管道复用，
	// 不要求用户单独提供 API）。冷却/频率由 DynamicCooldown 控制。
	pol := responses.DefaultPolicy()
	pol.Approve(responses.GrantSession, time.Now())

	res, err := responses.Retrieve(ctx, p.Query, responses.RetrieveOptions{
		Policy:    &pol,
		PanicMode: true, // #13 破壁引导（中立护栏，仅当前问题文本，无用户画像）
	}, retrieveFetch)
	if err != nil {
		// 未配置 deepseek-responses / 凭据不可用 → 授权提示而非报错；
		// 其余管道错误原样返回。
		if errors.Is(err, errNoResponsesProvider) {
			return blockedNotice(), nil
		}
		return "", err
	}

	if res.Entry == nil {
		if res.WebBlocked {
			return blockedNotice(), nil
		}
		return "本地知识缓存未命中。", nil
	}

	if res.StaleServed {
		return "⚠️ " + p.Query + "\n\n（缓存信息可能过期，标注见下文）\n" + res.Entry.AnswerSummary, nil
	}

	var b strings.Builder
	if res.FromCache {
		b.WriteString("【本地知识缓存命中】\n\n")
	} else {
		b.WriteString("【联网检索完成（deepseek-responses 管道）】\n\n")
	}
	b.WriteString(res.Entry.AnswerSummary)
	if len(res.Entry.KeyFacts) > 0 {
		b.WriteString("\n\n关键事实：\n")
		for _, f := range res.Entry.KeyFacts {
			b.WriteString("- " + f + "\n")
		}
	}
	if len(res.Entry.Sources) > 0 {
		b.WriteString("\n来源：\n")
		for _, s := range res.Entry.Sources {
			line := "- " + s.Title
			if s.URL != "" {
				line += " (" + s.URL + ")"
			}
			b.WriteString(line + "\n")
		}
	}
	if res.Entry.TimeSensitive && res.Entry.FreshUntil.After(res.Entry.CreatedAt) {
		fmt.Fprintf(&b, "\n（时效信息，截至 %s，如需最新请联网刷新）", res.Entry.FreshUntil.Format("2006-01-02 15:04"))
	}
	return b.String(), nil
}

func blockedNotice() string {
	return `{"needs_grant":true,"reason":"local cache miss and no deepseek-responses provider configured; web fetch requires user grant","options":["session","permanent"],"message":"本地知识缓存未命中，且系统未配置 deepseek-responses 供应商。请使用 ask 工具询问用户：允许联网检索吗？（选项：本次会话 / 永久 / 拒绝）"}`
}

// errNoResponsesProvider marks a missing/unresolvable deepseek-responses
// provider — the tool reports needs_grant instead of a hard error.
var errNoResponsesProvider = errors.New("deepseek-responses provider not configured")

// systemFetchTestHook lets tests replace the real network pipeline. The zero
// value uses systemFetch (real deepseek-responses pipeline).
var systemFetchTestHook responses.FetchFunc

// retrieveFetch is the fetch indirection used by Execute: tests inject a
// fake via systemFetchTestHook, production runs the system pipeline.
func retrieveFetch(ctx context.Context, query string, tier responses.RetrievalTier) (*responses.KnowledgeEntry, error) {
	if systemFetchTestHook != nil {
		return systemFetchTestHook(ctx, query, tier)
	}
	return systemFetch(ctx, query, tier)
}

// systemFetch performs one real web_search retrieval through the system's
// deepseek-responses provider pipeline — the same API key / endpoint the app
// already uses for chat, so the tool needs no separate credentials. Returns
// a distilled KnowledgeEntry (JSON extraction with markdown fallback).
func systemFetch(ctx context.Context, query string, tier responses.RetrievalTier) (*responses.KnowledgeEntry, error) {
	entry := responsesEntry()
	if entry == nil {
		return nil, errNoResponsesProvider
	}
	key := entry.APIKey()
	if key == "" {
		return nil, errNoResponsesProvider
	}
	p := responses.New(responses.Config{
		Name: entry.Name, APIKey: key,
		BaseURL: entry.BaseURL, Model: entry.Model,
		Effort: "low",
		// Server-side search is a client-level policy since #9826: the
		// web_search entry no longer rides the per-request tool list.
		WebSearch: true,
	})
	req := provider.Request{
		Messages:       []provider.Message{{Role: provider.RoleUser, Content: query}},
		ResponseFormat: provider.JSONSchemaFormat("knowledge_extract", knowledgeSchema),
	}
	ch, err := p.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("retrieve_info: web_search stream: %w", err)
	}
	var sb strings.Builder
	tokens := 0
	for c := range ch {
		switch c.Type {
		case provider.ChunkText:
			sb.WriteString(c.Text)
		case provider.ChunkUsage:
			if c.Usage != nil {
				tokens = c.Usage.TotalTokens
			}
		}
	}
	text := sb.String()
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("retrieve_info: web_search returned no text for %q", query)
	}
	return responses.DistillEntry(query, text, tokens, string(tier)), nil
}

// responsesEntry finds the system's deepseek-responses provider entry
// (kind="responses" on api.deepseek.com). LoadForRootReadOnly never writes
// config files.
func responsesEntry() *config.ProviderEntry {
	cfg, err := config.LoadForRootReadOnly("")
	if err != nil {
		return nil
	}
	for i := range cfg.Providers {
		e := &cfg.Providers[i]
		if e.Kind == "responses" && strings.Contains(e.BaseURL, "api.deepseek.com") {
			// 手动添加的 entry 常只有 models 列表、无顶层 model（如
			// deepseek-responses preset #7103）：取第一个模型兜底。
			if e.Model == "" && len(e.Models) > 0 {
				e.Model = e.Models[0]
			}
			if e.Model == "" {
				e.Model = "deepseek-v4-flash"
			}
			return e
		}
	}
	return nil
}

// knowledgeSchema instructs the model to return structured knowledge; the
// extraction is advisory (DeepSeek often replies markdown — DistillEntry
// falls back to markdown source/fact extraction).
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

// SaveCompactionDigest distills an A1 rolling-merge summary into the shared
// knowledge cache (P2 dream bridge). Called by the boot sink listener when a
// compaction pass completes with a non-empty summary. The entry is stored
// under a synthetic query so future sessions can L2-semantic-recall what the
// session learned even after its canonical transcript is compacted away.
// Best-effort: failures are swallowed by SaveKnowledge.
func SaveCompactionDigest(summary string) {
	if strings.TrimSpace(summary) == "" {
		return
	}
	// Noise gate: skip digests that are semantically near-duplicates of an
	// existing compaction-digest entry (sim >= 0.6). Repeated compactions of
	// the same session produce similar rolling summaries; storing every one
	// would flood the knowledge cache with near-identical frames.
	if hit, sim, ok := responses.LoadKnowledgeSemantic(summary, 0.6); ok && hit.Tier == "compaction-digest" && sim >= 0.6 {
		return
	}
	// Query is the L2 semantic key; answer summary carries the digest body.
	// Tier "compaction-digest" lets retrieve tiers distinguish distilled
	// session knowledge from web_search snapshots.
	responses.SaveKnowledge(&responses.KnowledgeEntry{
		Query:         "compaction-digest:" + firstLine(summary),
		AnswerSummary: summary,
		Tier:          "compaction-digest",
	})
}

// firstLine returns the first non-empty line of s, trimmed, for use as the
// semantic key prefix.
func firstLine(s string) string {
	for ln := range strings.SplitSeq(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			return t
		}
	}
	return ""
}

// RetrieveInfoQuery runs the retrieve_info tool on query and returns its
// rendered answer — the same path the agent tool executes. Exported so slash
// commands (/retrieve_info) and hosts can invoke retrieval without a model
// round-trip; the retrieval pipeline is the system's deepseek-responses
// channel (config provider, shared credentials) — no separate API key.
func RetrieveInfoQuery(ctx context.Context, query string) (string, error) {
	args, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return "", err
	}
	return (retrieveInfo{}).Execute(ctx, args)
}

// SetSystemFetchTestHook replaces the retrieval network pipeline with hook
// (nil restores the real one). Test-only: lets cross-package tests exercise
// retrieval without a live provider.
func SetSystemFetchTestHook(hook responses.FetchFunc) responses.FetchFunc {
	prev := systemFetchTestHook
	systemFetchTestHook = hook
	return prev
}

// RetrieveSystem runs the system retrieval pipeline for host commands such as
// /retrieve_info. Tests replace the network leg via SetSystemFetchTestHook.
// It returns the rendered answer and the structured outcome (cache hit vs
// paid call) so telemetry can distinguish zero-cost hits from API usage.
func RetrieveSystem(ctx context.Context, query string) (string, *responses.RetrieveResult, error) {
	pol := responses.DefaultPolicy()
	pol.Approve(responses.GrantSession, time.Now())
	res, err := responses.Retrieve(ctx, query, responses.RetrieveOptions{
		Policy:    &pol,
		PanicMode: true,
	}, retrieveFetch)
	if err != nil {
		return "", res, err
	}
	if res.Entry == nil {
		return "", res, fmt.Errorf("retrieval returned no entry")
	}
	text := res.Entry.AnswerSummary
	if res.StaleServed {
		text = "⚠️ " + query + "\n\n（缓存信息可能过期，标注见下文）\n" + text
	}
	return text, res, nil
}
