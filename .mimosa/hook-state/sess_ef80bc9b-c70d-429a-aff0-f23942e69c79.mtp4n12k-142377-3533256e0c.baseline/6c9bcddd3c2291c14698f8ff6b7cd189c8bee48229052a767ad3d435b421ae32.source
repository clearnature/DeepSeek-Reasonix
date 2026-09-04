package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCustomProviderDoesNotInheritDeepSeekDefaults（issue #7357 回归）：
// 用户 config 声明自定义 [[providers]] 数组时，不得与内置默认 provider
// （deepseek-flash/pro 带 balance_url/price/context_window）按 index 合并
// 字段——自定义 openai provider 必须保持零值（用户只写了自己的字段）。
func TestCustomProviderDoesNotInheritDeepSeekDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	raw := `config_version = 1
default_model = "gateway/my-model"

[[providers]]
name        = "gateway"
kind        = "openai"
base_url    = "http://localhost:8021/v1"
models      = ["my-model"]
api_key_env = "GATEWAY_API_KEY"
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := LoadForEdit(path)
	if len(cfg.Providers) != 1 {
		t.Fatalf("providers = %d, want 1 (user array replaces defaults, not merges by index)", len(cfg.Providers))
	}
	p := cfg.Providers[0]
	if p.Name != "gateway" {
		t.Fatalf("provider name = %q", p.Name)
	}
	if p.BalanceURL != "" {
		t.Fatalf("custom provider inherited DeepSeek balance_url: %q", p.BalanceURL)
	}
	if p.ContextWindow != 0 {
		t.Fatalf("custom provider inherited DeepSeek context_window: %d", p.ContextWindow)
	}
	if p.Price != nil {
		t.Fatalf("custom provider inherited DeepSeek price table: %+v", p.Price)
	}
	// 未声明 providers 的 config：默认 providers 保留（deepseek 官方可用）。
	noProv := filepath.Join(dir, "empty.toml")
	if err := os.WriteFile(noProv, []byte("config_version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg2 := LoadForEdit(noProv)
	if len(cfg2.Providers) == 0 {
		t.Fatal("default providers must remain when user config declares none")
	}
}
