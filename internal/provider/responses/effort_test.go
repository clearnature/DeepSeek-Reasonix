package responses

import (
	"testing"
	"time"

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
	// Vocabulary check: out-of-set override falls back to configured effort.
	if got := requestEffort(provider.Request{EffortOverride: "bogus"}, "high"); got != "high" {
		t.Fatalf("out-of-vocabulary override = %q, want fallback high", got)
	}
	if got := requestEffort(provider.Request{EffortOverride: "xhigh"}, "low"); got != "xhigh" {
		t.Fatalf("alias override = %q, want xhigh", got)
	}
}

func TestSummaryModeSerializedWithReasoning(t *testing.T) {
	deepseek := New(Config{Name: "deepseek", BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-flash", Effort: "high"}).(*client)
	body, _, _ := deepseek.buildRequestBody(provider.Request{Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}}})
	reasoning, _ := body["reasoning"].(map[string]any)
	if got, _ := reasoning["summary"].(string); got != "detailed" {
		t.Fatalf("deepseek summary mode = %q, want detailed", got)
	}
	if got, _ := reasoning["effort"].(string); got != "high" {
		t.Fatalf("deepseek effort = %q, want high", got)
	}
	mimo := New(Config{Name: "mimo", BaseURL: "https://api.xiaomimimo.com", Model: "mimo-v2.5-pro", Effort: "high"}).(*client)
	body, _, _ = mimo.buildRequestBody(provider.Request{Messages: []provider.Message{{Role: provider.RoleUser, Content: "hi"}}})
	reasoning, _ = body["reasoning"].(map[string]any)
	if got, _ := reasoning["summary"].(string); got != "none" {
		t.Fatalf("mimo summary mode = %q, want none", got)
	}
}

func TestCompactionOutputTokensPerVendor(t *testing.T) {
	cases := []struct {
		vendor, baseURL, model string
		want                   int
	}{
		{"deepseek", "https://api.deepseek.com", "deepseek-v4-flash", provider.DefaultOrdinaryOutputTokens},
		{"dashscope", "https://dashscope.aliyuncs.com", "qwen-max", 8192},
		{"mimo", "https://api.xiaomimimo.com", "mimo-v2.5-pro", 4096},
	}
	for _, tc := range cases {
		c := New(Config{Name: tc.vendor, BaseURL: tc.baseURL, Model: tc.model}).(*client)
		if got := c.CompactionOutputTokens(); got != tc.want {
			t.Fatalf("%s CompactionOutputTokens = %d, want %d", tc.vendor, got, tc.want)
		}
	}
}

func TestMimoStreamIdleTimeoutConfigured(t *testing.T) {
	c := New(Config{Name: "mimo", BaseURL: "https://api.xiaomimimo.com", Model: "mimo-v2.5-pro"}).(*client)
	if c.idleTimeout != 8*time.Minute {
		t.Fatalf("mimo idleTimeout = %v, want 8m", c.idleTimeout)
	}
}
