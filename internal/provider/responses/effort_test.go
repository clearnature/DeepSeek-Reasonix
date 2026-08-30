package responses

import (
	"testing"

	"reasonix/internal/provider"
)

func TestEffortOverrideWinsOverConfiguredEffort(t *testing.T) {
	client := New(Config{Name: "deepseek", BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash", Effort: "high"}).(*client)
	// 摘要/压缩请求：EffortOverride=none 必须覆盖配置的高思维等级。
	body, _, _ := client.buildRequestBody(provider.Request{
		Messages:       []provider.Message{{Role: provider.RoleUser, Content: "hi"}},
		EffortOverride: "none",
	})
	reasoning, _ := body["reasoning"].(map[string]any)
	if got, _ := reasoning["effort"].(string); got != "none" {
		t.Fatalf("EffortOverride=none serialized as %q, want none", got)
	}
	// 普通请求：EffortOverride 为空 → 回退配置 effort。
	body, _, _ = client.buildRequestBody(provider.Request{Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}}})
	reasoning, _ = body["reasoning"].(map[string]any)
	if got, _ := reasoning["effort"].(string); got != "high" {
		t.Fatalf("no override serialized as %q, want configured high", got)
	}
}

func TestRequestEffortFallbackAndNormalization(t *testing.T) {
	if got := requestEffort(provider.Request{}, "High"); got != "high" {
		t.Fatalf("fallback = %q, want normalized high", got)
	}
	if got := requestEffort(provider.Request{EffortOverride: "none"}, "high"); got != "none" {
		t.Fatalf("override = %q, want none", got)
	}
	if got := requestEffort(provider.Request{}, ""); got != "" {
		t.Fatalf("empty = %q, want empty", got)
	}
}
