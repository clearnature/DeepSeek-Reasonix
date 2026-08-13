# 本地压缩修复链 vs main-v2：核心技术差别与风险识别报告（2026-08-14）

> 场景：用户实测 main-v2 出现 2.4M/1M 超窗（240%）。本报告逐机制对比
> 本地（dev/clearnature-copy，8/13 修复链后）与上游 main-v2 的压缩核心，
> 识别本地已识别、上游未识别的风险与场景。

## 1. 核心机制对照表（本地 vs 上游）

| 机制 | 本地（copy） | main-v2 | 差别性质 |
|---|---|---|---|
| resume/启动守卫 | `MaybeCompactOnResume`（budget.go——**本地独有文件**）：恢复超窗会话先压缩再发 | **无**——恢复首轮裸发 canonical 全量 | 上游缺失——**启动超窗的直接防线** |
| 输出预算 | `effectiveOutputBudget`：窗口-估算-保留，**实测优先**（lastUsage） | 固定 131K/16K 梯子，无窗口联动 | 上游缺共享窗口约束（input+output≤window） |
| 估算口径 | `estimatedVisibleRequestTokens`：**校准优先**（per-model tokPerChar）+ 语言感知兜底 + lastUsage 实测 | fallback 0.25（英文）+ 0.6/0.3（CJK） | 上游无校准体系——长会话 drift |
| 校准持久化 | `calibration.go`（**本地独有**）：~/.reasonix/calibration/<model>.json 跨重启 | **无**——每次重启冷 fallback | 上游重启即丢失模型比率 |
| 投影校验 | `projectionContentValid`（**本地独有**）：coveredPrefixHash + **语义哈希**（prune 容忍）+ 版本漂移容忍 + rebind | 上游最新版无投影校验模块（无投影失效防护） | 上游投影一失效即退回 canonical 全量 |
| 投影持久化 | sidecar `.context.json` + 加载校验 | 有 sidecar 但无内容校验（加载即信） | 上游可能加载坏投影/不加载 |
| summarize 预算 | `summaryInputBudget`：窗口-头部前缀-输出预算；`keepFoldWithinSummaryBudget` 裁剪 fold 移回原文 | 无——summarize 请求可能超窗失败 | 上游 summarize 超窗 → degraded 机械折叠 |
| degraded 保数据 | `keepDegradedUserTurnsVerbatim`：机械折叠时保留全部 user 原文 | 无——degraded 丢 retention 外 user 消息 | 上游机械折叠丢语义 |
| est 触发 | `estimatedVisibleRequestTokens` 实测/校准/语言感知三层 | 0.25/0.6/0.3 fallback | 上游搜索密集/代码密集低估 |
| fork 子代理 | `captureForkInheritance`（继承父校准/lastUsage）+ 投影视图前缀 | 无（fork 原始全量 + 冷 fallback） | 上游子代理首请求必低估 |
| web_search 估算 | 计入 Raw（官方口径）+ `officialMessagesTokens` 已补 ServerSearch | **#8718 排除 Raw**（与官方口径相反） | 上游搜索密集会话低估 → 超窗 |

## 2. 2.4M/1M 场景拆解（上游超窗 = 本地已防）

2.4M/1M = 模型可见（canonical 或投影失效后的全量）超过窗口 1M 的 2.4 倍。
上游发生此场景的路径（每条本地都有对应防线）：

| # | 上游场景 | 本地防线 |
|---|---|---|
| ① | **恢复超窗会话**：无 resume 守卫 → 首轮裸发 canonical（含 1.5M 历史） | `MaybeCompactOnResume`：先压缩再发 |
| ② | **投影失效**（prune/模型切换/版本漂移）→ 退回 canonical 全量 | 语义哈希容忍 prune + 版本漂移不失效 + 模型切换 gate 修正 |
| ③ | **搜索/代码密集会话低估**：0.25 fallback + #8718 排除 Raw → est < fold 不触发 | 计入 Raw + 校准 + 语言感知兜底 |
| ④ | **重启后校准丢失**：fallback 低估 → 触发延迟 | calibration.json 跨重启恢复 |
| ⑤ | **summarize 超窗**：无预算裁剪 → 摘要失败 → degraded → 继续涨 | summaryInputBudget + keepFoldWithinSummaryBudget |
| ⑥ | **子代理 fork**：继承原始全量 + 冷 fallback | captureForkInheritance + 投影视图前缀 |

**2.4M/1M 的完整链条（上游）**：① 或 ② 触发（裸发/投影失效）→ 无守卫拦截 →
发送 1.5-2.4M → 400（或显示 240%）→ ③④⑤ 使后续压缩无法及时触发 → 会话卡死。

## 3. 本地识别到、上游未识别的风险/场景

1. **启动/恢复超窗**（#8739 域同源）：上游 v1.25.x 的会话恢复重建本身丢历史，
   且压缩层无 resume 守卫——两个问题叠加。本地两者都已处理。
2. **#8718 方向错误**：上游"排除 Raw"与 Anthropic 官方口径相反（重放搜索仍计
   input tokens）——本地计入 Raw，且补了 `officialMessagesTokens` 缺口。
3. **校准跨会话丢失**：上游每次重启冷 fallback——长会话 drift 无收敛。
4. **投影失效无防护**：上游投影失效即退回 canonical——本地语义哈希容忍
   prune/rewrite（工具结果瘦身场景）。
5. **degraded 数据丢失**：上游机械折叠丢 retention 外 user 消息——本地
   keepDegradedUserTurnsVerbatim。
6. **summarize 超窗失败**：上游无预算裁剪——本地 keepFoldWithinSummaryBudget。
7. **fork 子代理冷启动**：上游子代理首请求必 fallback 低估——本地继承校准。

## 4. 结论

- **本地方向胜出**：8/13 修复链（投影保持/校准/语义哈希/resume gate/est 口径/
  summarize 预算/degraded 保数据）系统性覆盖了 2.4M/1M 的全部上游触发路径。
- **2.4M/1M 不是单一 bug，是上游压缩体系缺 7 类防线**——上游每修一个
  表象（#8718 修"高估"）反而引入新缺口（低估超窗）——因为缺校准/实测闭环。
- **对上游**：建议以"校准 + 实测优先 + resume 守卫 + 投影校验"为体系方向，
  而非继续打补丁。
