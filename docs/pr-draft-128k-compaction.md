# feat: raise output budgets to 128K + cache-aware compaction (C1 replay gate + A1 digest merge + B2 fixed small-turn window)

> 中文摘要 / Chinese summary: 本 PR 包含两项正交优化——① 将 Responses/Chat/Anthropic 三协议的默认输出预算从 32K 提升到 128K（防截断、少迭代、降本 40%+）；② 缓存感知智能压缩（冷恢复门控 C1 + 摘要滚动合并 A1 + 位置固定小 turns 窗口 B2），冷恢复首轮输入减少 88-96%。全部数据来自 podman 容器隔离实测（真实历史会话续跑，非模拟）。

## Summary

Two orthogonal optimizations verified together in container-isolated benchmarks against real saved sessions (not synthetic):

### 1. 128K output budget (all three protocols)

| Protocol | File | Old | New |
|----------|------|-----|-----|
| Responses (DeepSeek) | `responses/vendor.go` | 32768 | **131072** |
| Responses (MiMo) | `responses/vendor.go` | 65536 | **128000** |
| Chat Completions (DeepSeek) | `openai/openai.go` | 32768 | **131072** |
| Anthropic Messages | `anthropic/anthropic.go` | 32768 | **131072** |

Rationale: a 32K budget that includes reasoning truncates long tool-call turns, forcing the model into many small write→test→fix iterations. 128K lets it finish in one pass — fewer iterations, far less input tokens (input is 50-120x more expensive than cache-hit input).

**Benchmark (same binary, only `max_output_tokens` changed, container-isolated):**

| Task | 32K | 128K |
|------|-----|------|
| Producer-consumer (Go) | 21 steps, 1.26M prompt, **CANCELLED** (timeout) | 8 steps, 356K prompt, **SUCCESS** |
| 4-task total cost | ¥0.2330 | **¥0.1378 (-40.9%)** |
| Total input | 1,338,745 | **433,760 (-67.6%)** |

### 2. Cache-aware compaction (C1 replay gate + A1 digest merge + B2 fixed small-turn window)

Addresses the "1.2M single-hop replay + 145.7K post-compaction residue" behavior measured in real sessions (see `docs/research/cache-aware-compaction-design.md`).

- **C1 replay gate** (`internal/control/controller.go` `maybeColdResumePrune`): when a saved session is resumed past the provider's cache TTL (DeepSeek/MiMo 24h, DashScope 5m), the prefix is cold — replaying ~1M tokens pays full price. The gate now runs a full compaction (not just stale-tool pruning) so the first send is a small digest+tail prefix. Inside the window, history replays verbatim (warm prefix hits cache; zero-loss multi-topic knowledge restoration).
- **A1 digest rolling-merge** (`internal/agent/compact.go` `pinnedPrefixLen`): only the NEWEST digest is pinned; older digests re-enter the fold region and are re-fed through the summarizer, so a long session cannot accumulate an unbounded digest chain (was 10 digests / 27K tokens).
- **B2 position-fixed small-turn window** (`partitionFold`): keep the first 20 small user turns in the fold region verbatim, fold the rest. The window is position-fixed (never "latest N") so the kept prefix stays byte-stable across compactions — protecting the server-side prefix cache (the core value: prefix byte stability).

**Benchmark (container-isolated, real 1,462-message session, resumed after 6-day idle → cold window):**

| Metric | main-v2 (baseline) | this branch | Delta |
|--------|-------------------|-------------|-------|
| Messages after resume | 1,464 (kept all) | **165** | **-88.7%** |
| Compaction digest | none | **present** (A1 merge) | — |
| Qwen Code converted session (3,942 msgs) | — | **132** | **-96.7%** |
| Resume cost (real session) | ¥0.43325 | **¥0.06820** | **-84.3%** |

### 3. Orthogonality (stacking)

128K works on the output side (no truncation → fewer iterations → less input). C1/A1/B2 work on the input side (history compaction → smaller stable prefix → cache hits). They are orthogonal: both can be merged together and the savings multiply. Container-verified end-to-end: 1,462-message real session resumed cold → 165 messages, and 128K prevents the long-output truncation that caused 21-step iteration loops in the baseline.

## Changes

```
docs/research/cache-aware-compaction-design.md  |  98 ++++++  (design doc)
internal/agent/compact.go                       |  53 ++++   (A1 + B2)
internal/agent/compact_test.go                  |  92 ++++   (tests)
internal/control/controller.go                  |  34 ++-   (C1 gate)
internal/control/resume_prune_test.go           |  71 ++++   (tests)
internal/provider/anthropic/anthropic.go        |   2 +-   (128K)
internal/provider/openai/openai.go              |   2 +-   (128K)
internal/provider/openai/openai_test.go         |   2 +-   (test)
internal/provider/responses/responses.go        |   3 +-   (128K)
internal/provider/responses/responses_test.go   |  18 +-   (tests)
internal/provider/responses/vendor.go           |   6 +-   (128K table)
11 files changed, 348 insertions(+), 33 deletions(-)
```

## Verification

- `go build ./...` — clean
- `go test ./internal/agent/ ./internal/control/ ./internal/provider/...` — all pass
- gofmt / go vet — clean
- Container-isolated benchmarks (podman, source compiled inside image):
  - 128K vs 32K: -40.9% cost, -67.6% input, no timeouts (baseline cancelled at 21 steps)
  - C1/A1/B2 cold resume: 1,462 → 165 messages (-88.7%), ¥0.43325 → ¥0.06820 (-84.3%)
  - Qwen Code converted session: 3,942 → 132 messages (-96.7%)
  - Real resume verified (user turns accumulate; no `--copy` meta reset; explicit `--resume` path)

Cache-impact: high - changes the output budget sent per request (max_output_tokens 32768→131072) and adds cold-resume compaction that rewrites the session prefix once per cold window; both are deliberate one-time prefix changes followed by stable bytes (mental-seal: prefix stability is the first principle — inside the cache window nothing changes; outside it the gate compacts once, then the new small prefix stays stable and cacheable).
Cache-guard: go test ./internal/agent/ ./internal/control/ ./internal/provider/responses/ -count=1 (TestColdResumeCompactsToDigestPrefix, TestWarmResumeSkipsCompaction, TestPartitionFold*, TestCompactRollsOldDigestsIntoNew, TestVendorTableMaxOutputTokens)

Documentation-impact: updated - adds docs/research/cache-aware-compaction-design.md (design rationale, cache-philosophy validation table, benchmark methodology) and documents the 128K budget change rationale in code comments.

System-prompt-review: none - no system-prompt, memory-prefix, output-style, or skill-index changes; the changes are in provider wire format (output budget) and session compaction (agent/control), neither touches the cache-stable system prefix.
