package responses

import "time"

// 对话中自动检索的策略层（2026-08-03 用户确认的四决策）：
//
//	门控 A 本地缓存：默认允许（零成本，不联网，不打扰用户）
//	门控 B 联网权限：per-session 授权（列表可见，不静默联网）
//	门控 C 频率尺标：五档（关/低/中/高/动态），用户自选
//	门控 D 过期：stale 标注（信息截至 X 时）+ 需授权才联网刷新
//
// 原则（§9 中立护栏）：本地命中永远免费；联网永远需要授权+节流；
// 过期信息标注清楚但绝不静默当新鲜答案。

// FrequencyTier is the user-selectable web-search frequency scale.
type FrequencyTier string

const (
	// FrequencyOff disables web search entirely (local cache only).
	FrequencyOff FrequencyTier = "off"
	// FrequencyLow allows a web refresh at most every 10 minutes.
	FrequencyLow FrequencyTier = "low"
	// FrequencyMedium allows a web refresh at most every 5 minutes.
	FrequencyMedium FrequencyTier = "medium"
	// FrequencyHigh allows a web refresh at most every 1 minute.
	FrequencyHigh FrequencyTier = "high"
	// FrequencyDynamic adapts the cooldown to observed cache behavior
	// (high hit rate -> longer cooldown; low similarity -> shorter).
	FrequencyDynamic FrequencyTier = "dynamic"
)

// FrequencyCooldowns maps each fixed tier to its minimum interval.
var FrequencyCooldowns = map[FrequencyTier]time.Duration{
	FrequencyOff:     24 * time.Hour, // effectively never (gate blocks anyway)
	FrequencyLow:     10 * time.Minute,
	FrequencyMedium:  5 * time.Minute,
	FrequencyHigh:    1 * time.Minute,
	FrequencyDynamic: 5 * time.Minute, // starting point; adapts per hit
}

// RetrievalPolicy is the per-session gate state for conversational
// auto-retrieval. Zero value = local cache only, no web search (safe default:
// nothing ever happens without an explicit opt-in).
type RetrievalPolicy struct {
	// LocalCache allows L1/L2 cache hits to answer without any network.
	// Default true: zero cost, never bothers the user.
	LocalCache bool
	// WebSearch grants per-session web access. Must be explicitly enabled
	// by the user (列表权限，可见不静默).
	WebSearch bool
	// Frequency is the user-selected scale.
	Frequency FrequencyTier
	// Cooldown overrides the tier's default when non-zero (dynamic gate
	// writes its adapted value here).
	Cooldown time.Duration
	// Grant is how long the web-search grant lasts after the user approves.
	Grant GrantDuration
	// GrantUntil is the wall-clock expiry of the current grant (zero =
	// never granted / not set; permanent grants use Grant=GrantPermanent and
	// ignore this field).
	GrantUntil time.Time
	// lastWebAt tracks the most recent web fetch (per-session state).
	lastWebAt time.Time
}

// GrantDuration is the user-selectable web-search authorization window
// (2026-08-03 UI 对话框：每次/本次会话/一周/一月/永久).
type GrantDuration string

const (
	// GrantEach re-prompts for every web fetch.
	GrantEach GrantDuration = "each"
	// GrantSession authorizes for the current session (process lifetime).
	GrantSession GrantDuration = "session"
	// GrantWeek authorizes for one week.
	GrantWeek GrantDuration = "week"
	// GrantMonth authorizes for one month.
	GrantMonth GrantDuration = "month"
	// GrantPermanent authorizes indefinitely.
	GrantPermanent GrantDuration = "permanent"
)

// GrantTTL maps each bounded grant to its duration; session/permanent are
// handled specially (zero = process lifetime, permanent = never expires).
func (g GrantDuration) ttl() time.Duration {
	switch g {
	case GrantWeek:
		return 7 * 24 * time.Hour
	case GrantMonth:
		return 30 * 24 * time.Hour
	default: // each / session
		return 0
	}
}

// IsGranted reports whether the current grant (if any) still covers now.
// GrantPermanent never expires; session grants never expire within the
// process; bounded grants expire at GrantUntil; GrantEach is only "granted"
// for the single call that approved it, so this returns false (the caller
// must consume a fresh approval).
func (p *RetrievalPolicy) IsGranted(now time.Time) bool {
	if p == nil || !p.WebSearch {
		return false
	}
	switch p.Grant {
	case GrantPermanent:
		return true
	case GrantEach:
		return false // each approval is consumed by one fetch
	case GrantSession:
		return true
	case GrantWeek, GrantMonth:
		return !p.GrantUntil.IsZero() && now.Before(p.GrantUntil)
	default:
		return false
	}
}

// Approve grants web access for the chosen window. For bounded grants it
// sets GrantUntil = now + ttl; permanent ignores the clock; session is
// process-lifetime; each must be re-approved per fetch.
func (p *RetrievalPolicy) Approve(g GrantDuration, now time.Time) {
	if p == nil {
		return
	}
	p.Grant = g
	p.WebSearch = true
	switch g {
	case GrantWeek, GrantMonth:
		p.GrantUntil = now.Add(g.ttl())
	case GrantPermanent, GrantSession:
		p.GrantUntil = time.Time{}
	}
}

// Revoke clears the web grant (user chose "deny" or each expired).
func (p *RetrievalPolicy) Revoke() {
	if p == nil {
		return
	}
	p.WebSearch = false
	p.Grant = ""
	p.GrantUntil = time.Time{}
}

// DefaultPolicy returns the safe default: local cache on, web off.
func DefaultPolicy() RetrievalPolicy {
	return RetrievalPolicy{LocalCache: true}
}

// effectiveCooldown resolves the tier default or the dynamic override.
func (p *RetrievalPolicy) effectiveCooldown() time.Duration {
	if p.Cooldown > 0 {
		return p.Cooldown
	}
	if cd, ok := FrequencyCooldowns[p.Frequency]; ok {
		return cd
	}
	return FrequencyCooldowns[FrequencyMedium]
}

// CanWebSearch reports whether a web fetch is currently permitted: explicit
// per-session grant (still within its window) + tier not off + cooldown
// elapsed.
func (p *RetrievalPolicy) CanWebSearch(now time.Time) bool {
	if p == nil || !p.IsGranted(now) || p.Frequency == FrequencyOff {
		return false
	}
	return now.Sub(p.lastWebAt) >= p.effectiveCooldown()
}

// MarkWebUsed records a web fetch so the cooldown starts counting from now.
func (p *RetrievalPolicy) MarkWebUsed(now time.Time) {
	if p != nil {
		p.lastWebAt = now
	}
}

// RemainingCooldown reports how long until the next web fetch is allowed
// (<=0 means allowed now).
func (p *RetrievalPolicy) RemainingCooldown(now time.Time) time.Duration {
	if p == nil {
		return 0
	}
	elapsed := now.Sub(p.lastWebAt)
	cd := p.effectiveCooldown()
	if elapsed >= cd {
		return 0
	}
	return cd - elapsed
}

// DynamicCooldown adapts the cooldown to observed behavior (frequency tier
// "dynamic"): strong cache signal (high hit rate, high similarity) means the
// cache is trustworthy, so web refreshes can be rarer; weak signal means
// fresh data is worth more, so the cooldown shrinks. result clamps to the
// [1 minute, 30 minutes] band. Callers feed hitRate (0..1) and the L2
// similarity (0..1) of the last hit.
func (p *RetrievalPolicy) DynamicCooldown(hitRate, sim float64) time.Duration {
	// Base from the medium tier, then scale.
	base := FrequencyCooldowns[FrequencyMedium]
	// Strong cache signal: hitRate high + sim high -> multiply up to 6x
	// (30 min); weak signal -> divide by up to 5x (1 min).
	signal := 0.5*hitRate + 0.5*sim // 0..1
	if signal >= 0.8 {
		return 6 * base // 30 min: cache is doing the job
	}
	if signal >= 0.5 {
		return base // 5 min: balanced
	}
	if signal >= 0.2 {
		return base / 2 // ~2.5 min
	}
	return base / 5 // ~1 min: weak signal, refresh more often
}

// ApplyDynamic writes the adapted cooldown into the policy.
func (p *RetrievalPolicy) ApplyDynamic(hitRate, sim float64) {
	if p != nil && p.Frequency == FrequencyDynamic {
		p.Cooldown = p.DynamicCooldown(hitRate, sim)
	}
}
