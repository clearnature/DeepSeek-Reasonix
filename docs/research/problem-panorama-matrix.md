# 问题全景矩阵（问题 × 架构层）

| # | 问题 | 归属层 | 根因 | 修复 |
|---|---|---|---|---|
| #7357 | custom provider 继承 DeepSeek 默认(balance/price/cw) | 配置层 | Default()+decode 按 index 合并 slice（7de6a2474 初始，417366028 balance_url 加剧） | 2db41eb95（本地 dev）：用户声明 providers 则清空默认 |
| #7273 | supported_efforts 被 DeepSeek thinking 硬编码校验覆盖 | 配置层/协议层 | 官方 DeepSeek 判定硬编码 thinking 校验，覆盖第三方代理自定义 effort | 上游已合（2026-08-03） |
| #7234 | MiMo Responses wire 对齐（summary/reasoning/JSON） | 协议层 | vendor 差异散落：summary 无字段折回膨胀/温度无效/max_output_tokens 截断/reasoning id 未贯通 | vendor 能力表 + 评审 3 点修复（本地） |
| #7168 | DashScope 协议兼容 + completion 语义 | 协议层 | summary 必须（缺 400）/reasoning id 重放/zero-usage/FinishReason/TTL | 统一实现（本地） |
| #4711 | mimo 模型 400（name is not set） | 协议层/会话层 | chatMessage.Name omitempty 省略空 name，MiMo 严格校验 400 | c7cbed768（name 改 *string） |
| #6259 | 工具轮缺失思考参数（DeepSeek） | 会话层 | 服务端偶发不发 reasoning（responses 40-75% 缺）；客户端降级 | 客户端降级+24h 限流（#6259 已关） |
| #7200 | Responses API 事实标准接口（feature） | 架构层 | 各厂商 Responses 支持差异不统一——生态级问题 | vendor 能力表（本地） |
| #7191 | corrupt subagent meta.json 中止启动 | 会话层 | 0 字节 meta.json 残留（强杀），stale cleanup 未容错 | a7f7cd4aa（跳过 corrupt meta） |
| #7099 | 上游已合 Responses 支持（地基） | 架构层 | Responses provider kind + stateless + deepseek-responses preset（7-31 合） | 上游 MERGED——#7234/#7168 建立其上 |
| #7184 | 中断轮次计费丢失（~100k tokens） | 会话层/计费 | usage chunk 在 SSE 流末尾——用户 stop 中断时未记录 | （与 #7168 zero-usage/完成语义交叉；agent 失败路径 usage 对齐） |
| #7111 | 微信 bot 工具轮静默无进度 | 会话层 | 工具轮 N 轮无实时进度，最后一次性 dump | open——UI/流式进度 |
| #7304 | ChatGPT/Codex OAuth 原生 provider | 配置层 | 新协议入口（OAuth）——provider 抽象扩展 | open——feature |
| #4099 | MiniMax thinking effort 校验错误 | 协议层 | effort 必须 adaptive/disabled 硬校验 | closed——与 #7273 同族 |
| #3561 | deepseek thinking effort 校验错误 | 协议层 | effort 必须 high/max 硬校验 | closed——与 #7273 同族 |
| #7451 | subagent effort 被拦截：TokenHub 实际支持却被判不可配置 | 协议层 | effort 判定过于自信——实际支持 reasoning_effort 的 vendor 被客户端拦截 | open——effort 家族第 4 成员 |
| #7410 | retrieve_info 工具上游 feature 请求（检索系统） | 工具层/架构层 | DeepSeek flash 免费 web-search 服务端检索 + 检索系统能力（grant/缓存/变体/护栏/编排） | open——本地 dev 完整实现，待上游化 |
| #7358 | 自定义 provider 默认 context_window=1M，压缩永不触发 | 配置层 | backfillOfficialContextWindow 用 deepseek 默认 1M 回填——小后端（131072）永不压缩→请求被拒 | #7357 同源（issue 内 'filed separately'）——清空默认修复同样覆盖 |

## 时间线（问题演变）

| 日期 | 事件 |
|---|---|
| 2026-06-17 | #4711 上报：mimo 400（name is not set）——协议严格校验首现 |
| 2026-07-09 | #6259 上报：DeepSeek 工具轮缺 thinking——会话层降级问题 |
| 2026-08-01 | #7168 提交：DashScope Responses 兼容（summary 必须/reasoning 重放） |
| 2026-08-02 | #7234 提交：MiMo wire 对齐 + #7200 提交（Responses 标准生态主张） |
| 2026-08-03 | #7273 关闭（effort 硬编码覆盖修复）；#6259 关闭（降级方案）；#7357 上报（自定义 provider 继承默认——隐私外呼） |
| 2026-08-04 | #7191 关闭（corrupt meta 容错）；#4711 关闭；本地 #7357 修复（2db41eb95） |

## 层间关联（初步）

- 配置层缺陷（#7357/#7273）→ 影响所有协议（provider 是协议入口）
- 协议层缺陷（#7234/#7168/#4711）→ vendor 判定散落是共同根因
- 会话层缺陷（#6259/#7191）→ 多轮工具循环的鲁棒性
- 架构层（#7200）→ 生态标准化诉求，vendor 能力表是代码内对应物

## effort 校验家族（#4099 / #3561 / #7273 同源）

| # | 现象 | 校验方向 |
|---|---|---|
| #4099 | MiniMax effort 必须 adaptive/disabled | vendor 特定硬校验打回用户配置 |
| #3561 | deepseek effort 必须 high/max | 同上 |
| #7273 | supported_efforts 覆盖第三方代理 | 官方判定后强加校验（已修） |
| #7451 | TokenHub 实际支持 reasoning_effort 却被判不可配置 | subagent profile effort 判定拦截（open） |

**同源结论**：三问题都是"vendor 判定后硬校验 effort"——用户配置在
"代码假定我知道 vendor"时被覆盖。修复模式统一：能力表/条件序列化。

## 计费链路交叉（#7184 ↔ #7168 ↔ MiMo 截断）

```
#7184：usage 在 SSE 流尾 → 中断丢失（~100k tokens 没记录）
#7168：zero-usage 抑制 + FinishReason 合成——完成语义与计费分离
MiMo 截断（da3aadd34）：长 reasoning 顶到 max_output_tokens → incomplete
      → 若用户中断则计费同样丢失（#7184 同路径）

交叉结论：计费记录依赖"流正常走完"——中断/截断/异常终止三路径
都必须显式处理 usage 上报（agent 失败路径对齐已部分覆盖）。

## 检索系统上游化（#7410 ↔ tools-server-web-search 分支）

#7410 是 retrieve_info 的上游 feature 请求（body 描述即本地 dev 的检索系统：
grant 简化/知识缓存/变体/护栏/编排层）——与 tools-server-web-search 分支
（检索系统完整版 + 场景覆盖优化）对应。上游化路径：feature 请求已提交，
代码在本地分支待 PR。

## 货币计价家族（#4814 等 8 条——#7357 的"货币版"）

| # | 状态 | 现象 | 与 #7357 同源点 |
|---|---|---|---|
| #4814 | open | DeepSeekOfficialPricingLanguage() 硬编码 zh → RMB 覆盖用户 USD 配置 | **默认注入覆盖用户配置**（官方 provider 路径） |
| #3883 | open | 余额多币种显示（RMB+USD 切换） | 币种体系诉求 |
| #4498/#4546/#4565/#3527 | closed | 币种不一致（CNY/USD 混乱、会话前后币种跳变） | 计价体系缺陷 |
| #4617 | closed | mimo 等模型估算费用功能 | 计价覆盖诉求 |
| #724 | closed | wallet 只显示 CNY（/user/balance 双币种） | 币种显示缺陷 |

**同源结论**：货币计价家族与 #7357 共享根因模式——"**默认/硬编码注入
覆盖用户配置**"（#4814 的 RMB 覆盖 USD = #7357 的 DeepSeek 价格继承
在官方 provider 路径的变体）。#7357 修复（用户声明 providers 清空默认）
覆盖自定义路径；#4814（官方路径 RMB 覆盖）仍是 open——修复方向：
DeepSeekOfficialPricingLanguage 尊重用户 currency 配置（去掉硬编码 zh）。
