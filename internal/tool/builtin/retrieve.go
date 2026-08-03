package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/internal/provider/responses"
	"reasonix/internal/tool"
)

func init() { tool.RegisterBuiltin(retrieveInfo{}) }

// retrieveInfo exposes the knowledge-cache lookup as a model-visible tool so
// conversational auto-retrieval (2026-08-03 design) works: the model can
// consult previously distilled web_search results with zero cost and zero
// network. Web refresh is NOT granted through this tool — it returns a
// blocked notice instead, keeping the "no silent web" guardrail (§9).
type retrieveInfo struct{}

func (retrieveInfo) Name() string { return "retrieve_info" }

func (retrieveInfo) Description() string {
	return "查询本地知识缓存（此前 web_search 蒸馏并落盘的检索结果）。零成本、不联网。命中返回缓存摘要与来源；未命中提示需联网授权。适合追问已检索过的事实/新闻/知识，避免重复联网。"
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

	// Local-only lookup: fetch stub is never called because the nil policy
	// blocks web (WebBlocked path returns before fetch).
	res, err := responses.Retrieve(ctx, p.Query, responses.RetrieveOptions{}, fetchStub)
	if err != nil {
		return "", err
	}

	if res.Entry == nil {
		if res.WebBlocked {
			// 未授权/授权过期：返回结构化标志，前端据此弹授权对话框。
			// 简化（2026-08-03）：只提供两档时长——本次会话 / 永久。
			return `{"needs_grant":true,"reason":"local cache miss; web fetch requires user grant","options":["session","permanent"],"message":"本地知识缓存未命中。允许联网检索吗？（本次会话 / 永久）"}`, nil
		}
		return "本地知识缓存未命中。", nil
	}

	if res.StaleServed {
		return "⚠️ " + p.Query + "\n\n（缓存信息可能过期，标注见下文）\n" + res.Entry.AnswerSummary, nil
	}

	var b strings.Builder
	b.WriteString("【本地知识缓存命中】\n\n")
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

// fetchStub is a fetch that must never run: the zero RetrieveOptions policy
// blocks web access, so the WebBlocked path returns before it is called. It
// exists to satisfy the non-nil FetchFunc contract defensively.
var fetchStub responses.FetchFunc = func(ctx context.Context, query string, tier responses.RetrievalTier) (*responses.KnowledgeEntry, error) {
	return nil, fmt.Errorf("retrieve_info: web fetch is not granted")
}
