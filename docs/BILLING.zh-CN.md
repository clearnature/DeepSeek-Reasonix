# 计费、展示币种与费用报价

Reasonix 将三类事实分开：

1. `original`：按公开/自定义价表计算的原币估算，不是发票或实际扣款。
2. `valuations`：调用发生时记录的 `identity`，以及可用时同模型另一官方区域的
   `official_table` 估算。
3. 钱包余额：供应商接口返回的原币事实。

运行时不再下载、缓存或刷新 FX，也不换算钱包。旧 JSON 中的 `fx`、
`rateSnapshot` 仍可读取为历史估值；新报价永远不会生成它们。

```toml
[billing]
display_currency = "auto"   # auto | CNY | USD

[[providers]]
billing_currency = "USD"    # 价表基准币种，不代表实际结算币种
billing_mode = "payg"       # payg | subscription_equivalent
```

旧 `[desktop].currency` 仍可读并迁移到 `[billing].display_currency`。
`auto` 在配置层保持未解析：单一有效钱包币种可以成为当前 tab/session 的运行时
提示；否则单一原币直接展示，混币则按币种分桶。语言、浏览器 locale、主机区域都不再
改变价表。

## CostQuote

`usage.costQuote` 是所有主机表面的规范 usage 载荷：

| 字段 | 含义 |
| --- | --- |
| `original` | 原币价表估算 |
| `originalTotals[]` | 混币聚合时按 ISO 排序的原币桶 |
| `valuations.*.basis` | 新报价只有 `identity` 或 `official_table` |
| `selected` | 只有形成单一展示总额时才存在 |
| `costComplete` | usage 与价表事实完整 |
| `displayComplete` | 已形成请求的单币种展示总额 |
| `complete` | 兼容别名，始终镜像 `displayComplete` |
| `displayStatus` | `matched`、`fallback_original`、`bucketed`、`unavailable` |
| `aggregateMode` | `single_currency`、`common_valuation`、`currency_buckets` |
| `rateBand` | DeepSeek 发生时刻档位：`peak`、`off_peak`，聚合时可为 `mixed` |
| `ratedAt` | 用于选择时间费率的请求完成时刻（UTC） |

目标币种缺失但所有原币相同时，使用 `fallback_original` 展示原币；混币输出
`originalTotals`，不写伪造的 0。只有缺少 usage/价格时才是 `unavailable` 并显示 `—`。
旧标量别名（`cost`、`costUsd`、`total_cost`）仅在存在 `selected` 时双写。

## DeepSeek 峰谷计价

从北京时间 2026-08-17 00:00 起，DeepSeek 官方 OpenAI、Responses 与
Anthropic 端点上的 V4 Flash、`deepseek-v4-flash-vision-exp`（价卡与 Flash 相同）
和 V4 Pro 按请求发生时刻计价。北京时间 09:00–12:00、14:00–18:00 为高峰，区间左闭右开，
其余为低峰。由于供应商不提供逐 token 计费时刻，Reasonix 使用取得 usage 的请求完成时刻，
并继续把报价标记为估算。发给视觉 SKU 的图片按供应商 usage 计入输入 token。

配置中保存的价格仍是高峰基准价。只有 PAYG 且完整价格精确匹配官方高峰基准价时才启用
动态档位；自定义端点、自定义价格和未知模型仍使用静态费率。已经写入 session、ledger、
stats 的历史报价不会回填或重算。

## 钱包与诊断

钱包余额不换算、不跨币种求和。显式目标有对应钱包时显示该钱包；没有时显示真实币种
并加 ISO 前缀。自动模式只有一个有效钱包币种时才作为运行时费用提示；多钱包、未知币种
或请求失败不会影响费用事实。

```sh
reasonix doctor billing
reasonix doctor billing --json
```

兼容保留的 `fx` 对象固定为 `enabled=false`、无缓存；正文同时展示自动选择策略、价表
基准币种和官方目录匹配情况。

### 各厂商积分/余额接口调研（2026-08）

没有任何厂商在 `usage` 响应块里返回积分/点数消耗——`usage` 只有 token 计数。余额或
积分事实（如果有）来自独立的账户接口；会话费用能否估算仍取决于价表，与这些接口无关：

| 厂商 | 计费模式 | 余额/积分接口 | usage 返回积分？ |
| --- | --- | --- | --- |
| DeepSeek（官方） | 按量付费、按 token | `GET /user/balance` → `balance_infos`（`total_balance`/`granted_balance`/`topped_up_balance`） | 否 |
| GLM | 积分制（100 credits = $1.80，按 $0.018/credit 计费） | 有余额接口；积分耗尽返回 HTTP 402 `insufficient_quota` | 否 |
| OpenRouter | 按量付费（部分积分） | `GET /api/v1/key`、`GET /api/v1/credits`（USD 积分余额） | 否 |
| MiMo Token Plan | 固定订阅费、套餐限量调用 | **无文档化接口（黑箱）** | 否 |

MiMo Token Plan 的积分换算系数（用户调研，2026-08，控制台文档）：

| 模型 | 缓存命中（输入） | 缓存未命中（输入） | 输出 |
| --- | --- | --- | --- |
| mimo-v2.5-pro | 2.5 Credits/Token | 300 Credits/Token | 600 Credits/Token |
| mimo-v2.5 | 2 Credits/Token | 100 Credits/Token | 200 Credits/Token |

未命中的惩罚是命中的 **120 倍**（Pro）；北京时间每日 00:00–08:00 有夜间 8 折。
配置里 MiMo 的价表（`cache_hit` 0.025 / `input` 3 / `output` 6 ¥/M）恰好是积分系数
的 1/100 折算（2.5/300/600）——积分模型与价表的命中/未命中之比一致（0.83%）。
一次 104 万 tokens 全未命中的压缩 fold ≈ 3.12 亿 Credits ≈ Max 套餐额度的 0.38%——
MiMo 上的压缩成本正是由系数编码的 120 倍未命中惩罚决定。

对 Reasonix 的影响：

- `usage` 永不携带积分；会话报价始终是价表估算，不是扣费或积分扣减。
- MiMo Token Plan 完全没有余额接口——其消耗只能参考（如套餐额度），无法查询。这正是
  会话在"有单价模型"和"套餐/额度模型"之间切换时无法给出单一完整费用数字的原因：
  有价段可估算，套餐段不可估算。
- 余额轮询保持可选、按厂商配置：官方 DeepSeek 与 OpenRouter 有文档化接口；MiMo
  Token Plan 刻意不预设 `balance_url`（见 `config.go` 的 `balance_url` 处理）。
