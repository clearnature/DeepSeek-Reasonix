# Billing, display currency, and cost quotes

Reasonix keeps three facts separate:

1. `original`: an estimate from the selected public/custom rate card in its
   pricing-table currency. It is not an invoice or a provider debit.
2. `valuations`: occurrence-time `identity` and, when available, an
   `official_table` estimate for the same model in the other official region.
3. Wallet balances: the exact original-currency values returned by a provider.

Reasonix has no runtime FX download, cache, refresh goroutine, or wallet
conversion. Old `fx`/`rateSnapshot` fields remain readable for history only;
new quotes never generate them.

```toml
[billing]
display_currency = "auto"   # auto | CNY | USD

[[providers]]
billing_currency = "USD"    # pricing-table basis, not settlement currency
billing_mode = "payg"       # payg | subscription_equivalent
```

Legacy `[desktop].currency` remains readable and migrates to
`[billing].display_currency`. `auto` is intentionally unresolved in config:
one valid wallet currency may become a tab/session hint; otherwise a single
original currency is selected or mixed currencies are shown as buckets. A
language, browser locale, or host region never changes a rate card.

## CostQuote

`usage.costQuote` is the canonical host-side usage payload:

| Field | Meaning |
| --- | --- |
| `original` | Original-currency rate-card estimate |
| `originalTotals[]` | ISO-sorted original buckets for mixed aggregates |
| `valuations.*.basis` | `identity` or `official_table` for new quotes |
| `selected` | A single amount only when a display total exists |
| `costComplete` | Usage and pricing facts are complete |
| `displayComplete` | A requested single-currency total exists |
| `complete` | Compatibility alias mirroring `displayComplete` |
| `displayStatus` | `matched`, `fallback_original`, `bucketed`, or `unavailable` |
| `aggregateMode` | `single_currency`, `common_valuation`, or `currency_buckets` |
| `rateBand` | DeepSeek occurrence-time band: `peak`, `off_peak`, or aggregate `mixed` |
| `ratedAt` | UTC request-completion time used to select a scheduled rate |

If a requested currency is unavailable but every original is the same, the
original amount is shown with `fallback_original`. Mixed originals produce
`originalTotals` and no scalar zero. `—` is reserved for missing usage or
pricing (`unavailable`). Legacy scalar aliases (`cost`, `costUsd`,
`total_cost`) are written only when `selected` exists.

## DeepSeek scheduled rates

For official DeepSeek OpenAI, Responses, and Anthropic endpoints, V4 Flash,
`deepseek-v4-flash-vision-exp` (same list price as Flash), and V4 Pro use
occurrence-time pricing from 2026-08-17 00:00 Beijing time. Peak windows are
09:00–12:00 and 14:00–18:00 Beijing time; boundaries are left-closed/right-open
and all other times are off-peak. The request-completion timestamp is used
because the provider does not report per-token billing time. Images sent to the
vision SKU are billed as input tokens from provider usage.

The stored provider price remains the peak anchor. Dynamic resolution is
enabled only for PAYG configurations whose complete rate card exactly matches
that official anchor. Custom endpoints, edited prices, and unrecognized models
remain static. Persisted session, ledger, and stats quotes are never repriced.

## Wallets and diagnostics

Wallets are never converted or cross-added. An explicit target uses the exact
matching wallet; if it is absent, the real currency is shown with an ISO
prefix. Automatic mode uses a single valid wallet currency only as a runtime
hint. Multiple/unknown/error responses do not affect the cost facts.

```sh
reasonix doctor billing
reasonix doctor billing --json
```

The compatible `fx` report is always `enabled=false` and has no cache. The
report also lists the automatic selection policy, pricing-table currencies,
and official-catalog matches.

### Provider credit / balance interfaces (survey, 2026-08)

No provider returns point/credit consumption inside the `usage` block of a
completion or responses payload — `usage` carries only token counts. Balance
or credit facts, when available at all, come from separate account endpoints,
and the ability to quote session cost still depends on a rate card, not on
those endpoints:

| Provider | Billing model | Balance/credit endpoint | Usage-level points? |
| --- | --- | --- | --- |
| DeepSeek (official) | pay-as-you-go, per token | `GET /user/balance` → `balance_infos` (`total_balance`, `granted_balance`, `topped_up_balance`) | no |
| GLM | credits (100 credits = $1.80; charged at $0.018/credit) | balance endpoint exists; exhaustion returns HTTP 402 `insufficient_quota` | no |
| OpenRouter | pay-as-you-go (some credits) | `GET /api/v1/key`, `GET /api/v1/credits` (USD credit balance) | no |
| MiMo Token Plan | fixed subscription, quota-limited calls | **none documented** (black box) | no |

MiMo Token Plan publishes credit conversion coefficients in its console
documentation (user-surveyed, 2026-08):

| Model | cache-hit input | cache-miss input | output |
| --- | --- | --- | --- |
| mimo-v2.5-pro | 2.5 credits/token | 300 credits/token | 600 credits/token |
| mio-v2.5 | 2 credits/token | 100 credits/token | 200 credits/token |

The miss penalty is 120× the hit rate (Pro), and a nightly 20% discount
applies 00:00–08:00 Beijing time. The list prices configured for MiMo
(`cache_hit` 0.025 / `input` 3 / `output` 6 ¥ per M) are exactly the credit
coefficients scaled by 1/100 (2.5/300/600), so the credit model and the rate
card agree on the hit-to-miss ratio (0.83%). A 104万-token all-miss fold
therefore burns ≈3.12亿 credits ≈ 0.38% of a Max plan — compression on MiMo
is priced by the same 120× miss penalty the coefficients encode.

Consequences for Reasonix:

- `usage` never carries credits; a session quote is always a rate-card
  estimate, never a debit or credit deduction.
- MiMo Token Plan has no balance endpoint at all — its consumption can only be
  referenced (e.g. plan quota), never queried. This is why sessions that
  switch between a priced model and a quota/plan model cannot produce one
  single complete cost figure: the priced segment is estimable, the plan
  segment is not.
- Balance polling stays optional and per-provider: official DeepSeek and
  OpenRouter expose documented endpoints; MiMo Token Plan deliberately has no
  `balance_url` preset (see `config.go` `balance_url` handling).

## Compaction economics vs cache pricing

Compaction's payback is set by the ratio between the cache-hit price and the
full input price, not by the absolute prices:

```
payback = (cache_hit / input) × rounds_saved
compaction cost  = fold × input price (a fold is a miss-heavy summarize call)
compaction gain  = fold × cache_hit price × remaining rounds
```

Measured ratios (list price):

| Provider | cache_hit / input | payback @160 rounds | verdict |
| --- | --- | --- | --- |
| DeepSeek v4-flash | 0.1 / 3 = 3.3% | 5.3× | compact — saves ~34% |
| GLM 5.3 | 0.26 / 1.4 = 18.6% | 29.8× | compact aggressively — saves ~79% |
| MiMo v2.5-pro | 0.025 / 3 = 0.83% | 1.3× | defer — retention at hit price nearly free |

The higher the cache price, the more expensive keeping history verbatim is and
the bigger the compaction win. Reasonix reflects this in `priceAwareCompactRatio`
(agent/compact.go): when `cache_hit / input < 1.5%` the automatic compaction
trigger moves from 0.80 to 0.90 of the window (compaction runs less often);
providers with expensive caches (GLM-style) keep the default. A user-configured
`compact_ratio` always wins.

Cache pricing is also a proxy for the provider's KV-cache engineering: cheap
cache (DeepSeek, ~3%) means efficient server-side caching, so a session can
afford to keep history verbatim; expensive cache (GLM, ~19%) forces more
aggressive folding. This is the economic anchor behind the byte-stable prefix
philosophy.

Sensitivity check (local model, no API): `go run ./cmd/cost-model -days 7
-hit-price 2.5` overrides the cache-hit price. At 2.5 (GLM-style ratio 83%)
no-compaction costs ¥28.1 and compacting every 20 rounds saves 78.6%; at the
default 0.1 the same policy saves 34%. The verdict flips only at very low
hit-to-input ratios, which is exactly what the price-aware trigger encodes.

Replay is not a cache reorder: resuming a session re-sends the exact stored
prefix bytes (C1 warm-window gate, `cacheColdAfter`), so the server charges
the cache-hit price — measured 99.4% hit on a 1.55M-token replay (¥0.043).
Only cold-window resumes prune stale tool results and re-send a small prefix.
Summaries themselves are paid for (the fold request), which is why reducing
misses — not summaries — is the lever on MiMo (120× miss penalty).
