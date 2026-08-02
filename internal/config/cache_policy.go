package config

import (
	"strings"
	"time"
)

// DefaultCacheTTL returns the vendor's known prefix-cache retention based on
// the provider's base_url. Used by cold-resume prune to decide whether the
// provider cache is still warm after a session idle period.
//
// The legacy main-v2 default (24h) is deliberately preserved for DeepSeek and
// unknown vendors: DeepSeek's Context Caching on Disk retains prefixes for
// "several hours to days", so the long-standing 24h threshold is correct for
// it and must not regress. Only vendors with a documented, much shorter
// cache TTL (DashScope 5m, Anthropic 5m) override it.
//
// Values are deliberately conservative: too small burns a live cache (the user
// pays full price for a prefix that was still cached server-side), too large
// only forgoes a prune opportunity. Tighten from measured retention data.
func DefaultCacheTTL(baseURL string) time.Duration {
	switch detectCacheVendor(baseURL) {
	case "dashscope":
		// DashScope Session cache TTL is 5 minutes (documented).
		return 5 * time.Minute
	case "anthropic":
		// Anthropic ephemeral cache TTL is 5 minutes.
		return 5 * time.Minute
	default:
		// DeepSeek and unknown vendors keep the legacy 24h default.
		// DeepSeek Context Caching on Disk retains prefixes for hours to
		// days; shrinking this would prune still-warm caches and burn
		// the user's live cache (measured ~4x miss cost).
		return 24 * time.Hour
	}
}

// EffectiveCacheTTL resolves the provider's cache TTL: an explicit
// cache_ttl_minutes config wins; otherwise the vendor default applies.
func (e *ProviderEntry) EffectiveCacheTTL() time.Duration {
	if e.CacheTTLMinutes > 0 {
		return time.Duration(e.CacheTTLMinutes) * time.Minute
	}
	return DefaultCacheTTL(e.BaseURL)
}

// detectCacheVendor identifies the provider vendor from its base_url for
// cache policy purposes. Mirrors provider/responses.DetectVendor but lives in
// the config layer to avoid an import cycle (control → config, not control →
// provider).
func detectCacheVendor(baseURL string) string {
	u := strings.ToLower(strings.TrimSpace(baseURL))
	switch {
	case strings.Contains(u, "dashscope.aliyuncs.com"), strings.Contains(u, ".maas.aliyuncs.com"):
		return "dashscope"
	case strings.Contains(u, "api.deepseek.com"):
		return "deepseek"
	case strings.Contains(u, "api.xiaomimimo.com"):
		// MiMo: auto cache TTL measured short-prefix ~3.5-7.5min, but the
		// vendor targets long TTLs (Hybrid SWA 1/7 KV + GCache L3, "hours"
		// per its engineering blog). Kept at the 24h unknown-vendor default
		// until a long-prefix (>8K tokens) cross-hour measurement settles it.
		return "mimo"
	case strings.Contains(u, "api.anthropic.com"):
		return "anthropic"
	default:
		return ""
	}
}
