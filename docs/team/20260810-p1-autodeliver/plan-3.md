# 计划：P1 后台结果自动投递（独立方案 plan-3）

> 生成：2026-08-10 · 分支：team · 规划：Planner（本文件只产出计划，交由执行小队实施、纪律团审查）
> 关联：`docs/MULTIAGENT_QWEN_COMPARISON.md` P1（L48-52）、`docs/team-org.md`、社区 #7962
> ⚠️ 参考缺口：`docs/team-plan.md` 与 CCB 的 `<task-notification>` 命令队列实现**不在本工作区**（`ls docs/` 与全库 grep 无匹配），以下按 `MULTIAGENT_QWEN_COMPARISON.md` 的 P1 定义 + 用户给出的语义（"命令队列注入下一条 user 消息"）补足，见拓扑扫描 §0。

---

## 一、拓扑扫描

### 0. 参考物与缺口
| 参考 | 现状 | 结论 |
|------|------|------|
| `docs/team-plan.md`（P1 定义） | 文件不存在于工作区（仅 `docs/team-org.md` L4 引用其名） | 以 `docs/MULTIAGENT_QWEN_COMPARISON.md` L48-52 的 P1 定义为准：**保持 input.go 注入点位置不变，把"一行摘要"升级为 bounded 完整正文，前缀字节不变、append 尾部、位置固定 → 缓存零影响** |
| CCB `<task-notification>` 命令队列 | 全库 grep `task-notification`/`CCB` 无匹配 | 采用其语义（结果入队、投递到**下一条 user 消息**、不 mid-turn 插入）作为行为基线；我方落点沿用 `internal/control/input.go` 现有注入点，不引入新队列机制 |

### 1. 现状锚点（行号已核实）
- `internal/jobs/jobs.go`
  - L120-149 `Job` 结构：`result`（task 最终答案）、`tail []byte`（bash 流式环形缓冲，`appendTail` limit=`defaultTailBytes`=64KB，见 artifacts.go L19）、`artifactPath/artifactMetaPath/artifactComplete/artifactErr`、`status`、`done chan`、`SessionID/Label/Kind`。**无 usage 字段**。
  - L183-186 `completion` 结构：仅 `{sessionID, text}`——正文缺失的根因。
  - L778-814 `recordCompletion`：`m.completed = append(...)` 一行摘要 `"id (label) — status"` + emit Notice。**无结果正文、无 artifact 引用、无 usage**。
  - L1185-1210 `DrainCompletedNoteForSession`：drain 后拼一行 `"Background job updates since your last message: ...; Read their output with bash_output or wait..."`。**返回纯文本、无结构、无 bounded 控制**（`m.completed` 队列本身 unbounded，L169）。
  - L470-542 `StartForSession` goroutine：L490-501 结果正文已写入 artifact 文件（优先）或 `j.result` + `j.tail`；**L530 `recordCompletion` 在 L536-539 状态翻转/清空之前调用，且结果正文此时已稳定**——快照时机成立。
  - L870-900 `OutputForSession`：bash_output 的读路径，**推进 `readOffset`/置位 `resultRead`**（L882、L891）——自动投递绝不能走它，否则会消费读游标。
- `internal/control/input.go`
  - L152-217 `composeWithGoal`：**注入点在 L191-195**——`<background-jobs>` 块 prepend 到 user 消息开头（memory-update 之后、hook-context/recall 之前），位置固定、不插入历史、不动 system/tool 前缀。这是 P1 必须保留的落点。
  - L293-300 `ComposeSynthetic` 不注入 background-jobs → 合成 turn 不投递，语义正确。
- `internal/agent/preview.go` L19-29 `TransientUserBlockTags`：单一真源，含 `background-jobs`；`reTransientUserBlock`（L37-40）由它生成。**任何新顶层块漏登记会泄漏 UI**（注释 L13-14 明确警告）。
- `internal/history/strip.go` L8 `reComposeBlock`：硬编码 `memory-update|background-jobs|active-goal|hook-context`。
- `internal/control/input_test.go` L1386-1389、`internal/agent/agent.go` L2142：剥离路径测试与调用点。
- `internal/tool/builtin/bgjobs.go`：bash_output（L50-84）/wait（L163-199）取正文 + `collectBackgroundEvidence`（L201+，含 `planmode.Active` 守卫与 `TryLeaseEvidenceForSession` ready 守卫）。**证据合并/审查由这两个工具独占，自动投递不得触碰**。
- `internal/agent/task.go` L924-958：后台 task 的 run 函数，result=`FormatSubagentRunResult(answer, run, failed)`（L1741-1753，含 "Final answer:" + 子代理最终答案）；**usage 不进入 result**（子代理 usage 走 event.Usage 流，nested_sink.go `UsageSourceSubagent`）。
- `internal/jobs/artifacts.go`：`ArtifactDir(sessionPath)`、`ListArtifactViews`、`jobLogExt=".log"`、`jobMetaExt=".json"`、`defaultTailBytes=64KB`（artifact 引用可复用的持久化路径）。

### 2. 级联风险清单
| # | 风险 | 影响面 | 等级 |
|---|------|--------|------|
| R1 | 新顶层信封块未登记 `TransientUserBlockTags`/`reComposeBlock` → 泄漏 UI/标题 | preview.go、history/strip.go、input_test.go、agent.go 剥离路径 | 🔴 高 |
| R2 | 自动投递误走 `OutputForSession` → 消费 `readOffset`/`resultRead`，wait/bash_output 之后读到空 | jobs.go L882/891 语义 | 🔴 高 |
| R3 | 锁顺序反转（j.mu→m.mu 与既有 m.mu→j.mu 路径互等）→ 死锁 | jobs.go L2017+ `PendingEvidenceJobIDsForSession` 等先 m.mu 后 j.mu | 🔴 高 |
| R4 | 结果正文 unbounded → 上下文膨胀，父上下文被灌爆 | 注入块字节数 | 🔴 高 |
| R5 | 结果正文含伪造关闭标签（子代理答案里写 `</background-jobs>`）→ 结构逃逸 | 注入块解析 | 🟠 中 |
| R6 | recordCompletion 与状态翻转/证据发布的既有时序被破坏（L524-530 注释的 -race 历史） | jobs_test.go `TestDrainMultiple` | 🟠 中 |
| R7 | 改动 `completion`/drain 返回值破坏既有测试断言（jobs_test.go L578-589、L652-654 等 `strings.Contains(note, id)`） | jobs_test.go | 🟡 低（信封仍含 task_id 则兼容） |
| R8 | 多 session 隔离：非活跃 parentSession 的完成项不得投递（L1183-1203 已按 sessionID 分流） | input.go L192 | 🟡 低（保持现逻辑） |

---

## 二、多路径推演

### 方案 A：`<background-job-result>` 作为 `<background-jobs>` **容器内子条目**（推荐）

信封规格（设计规格，非实现）：

```
<background-jobs>
Background job results since your last message:
<background-job-result task_id="task-3" status="done" label="refactor auth">
Subagent reference: run:20260810-... (task-3)
Final answer:
<已转义的正文，见下>
</background-job-result>
<background-job-result task_id="bash-2" status="failed" artifact=".../session.jobs/bash-2.log">
exit 1: build failed
[truncated: 12,408 more bytes; read with bash_output or wait, or read_file the artifact]
</background-job-result>
</background-jobs>
```

- **信封字段**：`task_id`（job.ID）、`status`（done/failed/killed）、`label`（可选）、结果正文（bounded，≤X KB/条）、`artifact`（可选，持久化日志路径引用，仅当正文被截断或输出走 artifact 时）、`usage`（可选/预留，v1 语义见方案 C）。
- **结构转义**：正文内凡出现 `<` 开头的信封标签字样（`</background-job-result>`、`</background-jobs>`、`<background-job-result`）统一替换为 `<\\/...`（沿用 `escapeHookContext` input.go L285-287 先例；Qwen `escapeEnvelopeTags` 同思路），防伪造关闭标签（R5）。
- **bounded（R4）**：三条上限叠加——`maxResultBodyBytes`（默认 4096 字节/条，rune 截断，参考 `clipHookContext` L274-283）、`maxResultsPerDrain`（默认 8 条/次，超出留队下轮）、`maxBlockBytes`（默认 16KB/块，超限丢最旧并附 `[N more result(s) queued]` 标记）。
- **注入落点**：`input.go` L191-195 **原位置不动**，仅把 `DrainCompletedNoteForSession` 的返回值从一行摘要换成上面信封块。位置固定、turn 首部、不插入历史、不动已发送消息 → 缓存零影响。
- **剥离兼容**：容器标签仍是 `background-jobs` → `TransientUserBlockTags`、`reComposeBlock`、`StripComposePrefixes` **全部零改动**（R1 直接消解）。
- **兼容 wait/bash_output（R2）**：新增只读快照函数（如 `ResultSnapshotForSession(parentSession, id)`），**不调用 `OutputForSession`**，不推进 `readOffset`、不置位 `resultRead`、不触发 `collectBackgroundEvidence`。快照内容与 wait/bash_output 同源（j.result / j.tail / artifact 文件），一致性由同一 Job 状态机保证；父代理后续再调 wait/bash_output 幂等无害。**mutation 证据仍只由 wait/bash_output 或 turn 起始 re-lease 合并，自动投递不越权**（审查红线）。
- **快照时机与锁（R3/R6）**：在 `StartForSession` goroutine L503-523 的 `j.mu` 临界区内，把 result/tail 的 bounded 前缀拷贝到局部变量（此时正文已写完且稳定，L490-501），随后 L530 `recordCompletion(parentSession, id, kind, label, st, err, snapshot)` 只持 `m.mu`。**不在 `m.mu` 内拿 `j.mu`、也不在 `j.mu` 内拿 `m.mu`**——彻底回避锁序死锁；`startInvalid`（L381-408）传空快照。
- **usage（方案 C 见下）**：v1 信封字段预留、缺省省略；独立子任务采集，成本超限则降级。

### 方案 B：`<background-job-result>` 作为**独立顶层块**（备选，不推荐）

- 优点：信封语义独立、不嵌套。
- 代价：必须同步改 3 处剥离路径（`preview.go` L19-29 登记标签、`history/strip.go` L8 正则、`input_test.go` 用例 + `agent.go` 透传），R1 风险高；且 `composeWithGoal` 要新增第二个注入分支（L191-195 之外），位置稳定性下降。收益仅"语义更纯"，对模型与 UI 无实质差异。
- 判定：**否决**，作为对抗自检的对照保留。

### 方案 C：usage 字段两档实现
- **C1（完整，成本中高）**：让 task run 把子代理 usage 汇总进 job（新增 `Job.usage` 字段 + `runSession` 暴露 usage 或经 ctx 记录，task.go L950），信封渲染 `usage="prompt 12k/comp 3k"`。涉及 task.go 签名面，回归风险偏大。
- **C2（v1 默认，推荐）**：信封 schema 预留 `usage`（omitempty），v1 不采集，正文尾部不含 usage；文档标注 P1.5 待办。**理由**：#7962 的核心痛点是"正文不投递"，usage 是锦上添花；把 usage 采集拖入 P1 会放大 R6/R3 风险面。
- **判定**：默认 C2，执行小队若在完成任务分解 T1-T6 后仍有预算可升 C1（须附纪律团评审）。

### 方案选择总结
| 维度 | 方案 A | 方案 B |
|------|--------|--------|
| 复杂度 | 低-中（信封渲染 + 快照函数） | 中-高（+3 处剥离路径同步） |
| 性能 | 无热路径影响（drain 每轮一次，bounded I/O） | 同左 |
| 可维护性 | 单真源标签不变，剥离零改动 | 两套标签漂移风险 |
| 风险 | 低（R2/R3/R5/R6 有明确对策） | 高（R1） |
| **选优** | ✅ **采用** | ❌ 否决 |

---

## 三、任务分解（子任务 + 验证点）

> 交付物文件：`docs/team/20260810-p1-autodeliver/execution.md`（执行小队）+ `review.md`（纪律团）。执行小队 ≥3 人，可并行 T1/T2 与 T4/T5。

- [ ] **T1 结构化完成记录**：扩展 `internal/jobs/jobs.go` 的 `completion` 结构（L183-186）为结构化字段（id/kind/label/status/sessionID/正文快照 bounded 前缀/artifact 引用/usage 预留），并把 `recordCompletion`（L778-814）改收快照参数；`startInvalid`（L406）与 goroutine 完成路径（L530）传快照；`DrainCompletedNoteForSession`（L1185-1210）改为渲染 `<background-jobs>` 信封（含结构转义 + 截断标记）。**锁纪律**：快照在 `j.mu` 临界区内拷贝，`recordCompletion`/drain 只持 `m.mu`，禁止 j.mu→m.mu 嵌套。
  → 验证：`go test ./internal/jobs/ -run 'TestDrain|TestSessionScoped' -count=1` 全绿；新增 `TestDrainRendersResultEnvelope`（含 task 正文、bash 截断、artifact 引用、伪造关闭标签转义、条数/字节上限、跨 session 分流）；既有 `jobs_test.go` L578-589/L652-654 断言（`strings.Contains(note, id)`）不回归。

- [ ] **T2 只读快照函数（兼容 wait/bash_output 的关键）**：在 `internal/jobs` 新增 `ResultSnapshotForSession(parentSession, id)`（或等价命名），从 `j.result`/`j.tail`/artifact 文件取 bounded 前缀，**不得调用 `OutputForSession`、不得动 `readOffset`/`resultRead`、不得触碰 evidence lease**。
  → 验证：新增 `TestResultSnapshotDoesNotConsumeOutputCursor`——先 snapshot 再 `OutputForSession`，断言 `Output` 仍返回全量新输出（证明游标未被消费）；`go test ./internal/jobs/ -race -count=1` 通过。

- [ ] **T3 注入落点接线**：`internal/control/input.go` L191-195 **保持注入位置与容器标签 `background-jobs` 不变**，仅消费 T1 的新渲染结果（`composeWithGoal` 内逻辑调整）；确认 `ComposeSynthetic` 仍不投递。
  → 验证：`go test ./internal/control/ -run 'TestStripComposePrefixes|TestCompose' -count=1`；新增 `TestComposeInjectsBoundedResultEnvelope`——Compose 后 user 消息含 `<background-job-result task_id=...>` 且 `StripComposePrefixes` 后回退到用户原文；`controller_test.go` 既有 Compose 断言不回归。

- [ ] **T4 剥离路径回归（R1 防线）**：由于容器标签不变，`internal/agent/preview.go` L19-29、`internal/history/strip.go` L8、`internal/control/input.go` `StripComposePrefixes` **预期零改动**；执行小队须 grep 验证无新增顶层标签泄漏，并跑 `internal/agent` + `internal/history` 相关测试。
  → 验证：`grep -rn 'background-job-result' internal/ --include='*.go'` 只出现在信封渲染/解析测试位置（不得出现在 TransientUserBlockTags/reComposeBlock）；`go test ./internal/agent/ -run 'StripTransient|TransientTags' ./internal/history/ -count=1` 全绿。

- [ ] **T5 bounded 策略参数与单测**：实现 `maxResultBodyBytes`(4096)/`maxResultsPerDrain`(8)/`maxBlockBytes`(16KB) 三档上限（常量放 jobs 包，可被测试覆盖；数值经纪律团复核防上下文膨胀）。
  → 验证：新增 `TestEnvelopeBounds`——10KB 正文截到 ≤4096 且带 `[truncated...]` 标记；20 条完成只投 8 条且余留队列、块总字节 ≤16KB、丢最旧带 `[N more result(s) queued]`。

- [ ] **T6 端到端 + 缓存红线验证**：后台 task（含真实子代理 run 或测试替身）完成后，下一条 user 消息自动携带正文，**无需 wait/bash_output**；同轮次 wait 仍可取全量输出。
  → 验证：仿 `internal/agent/task_test.go` L944+ 的 `StartForSession` 驱动场景，断言第二条 user 消息含 `Final answer:` 正文；`go test ./internal/agent/ -run 'Background|Task' ./internal/control/ -count=1`；缓存侧跑 `internal/agent/cachehit_e2e_test.go` 既有套件确认前缀稳定性无回归（无新增 system/tool 前缀字节）。

- [ ] **T7 usage 决策记录**：默认 C2（信封预留 `usage` 字段、v1 省略）；若执行小队升 C1 须在 execution.md 记录 `runSession` 改动面与证据并交纪律团评审。本子任务=产出决策记录，无代码。
  → 验证：`docs/team/20260810-p1-autodeliver/execution.md` 含 usage 档位选择与理由；信封示例含 `usage` 字段声明（omitempty 语义）。

---

## 四、对抗自检（devil's advocate）

1. **攻击"正文投递后父代理不再 wait，证据审查被绕过"**：反驳——自动投递**只投文本快照，不合并 evidence lease**；mutation 证据仍由 wait/bash_output（`collectBackgroundEvidence`）或 turn 起始 re-lease（`PendingEvidenceJobIDsForSession`）独占合并，审查门不变。⚠️ 薄弱环节：若执行者图省事在 snapshot 里顺手调 `LeaseEvidenceForSession`，将引入审查绕过——T2 验证点必须显式断言"snapshot 不产生 lease"。
2. **攻击"锁序仍可能死锁"**：快照在 `j.mu` 内拷贝、`recordCompletion` 只持 `m.mu`，理论上无嵌套；但 `recordCompletion` 若在 `m.mu` 内调 `m.get`（get 也持 `m.mu`）会二次加锁。⚠️ 薄弱环节：实现时严禁在持锁路径内调任何 `m.*` 查询方法；验证点 `go test -race` + 纪律团人工审 T1 diff 的锁块。
3. **攻击"截断正文丢关键尾部"**：bash 输出截取 `tail` 是环形缓冲（64KB 内最新），若关键结论在输出头部会被截掉。对策：信封附带 `artifact` 路径引用 + 明确 `[truncated...]` 指示，父代理可 read_file 补全；正文截断取**尾部优先**（与 tail 语义一致）还是**头部优先**需纪律团拍板——默认取尾部（离结论最近）。
4. **攻击"结构转义不彻底"**：只转义关闭标签不够——正文里 `<background-job-result` 开标签也会被 regex 误配。对策：转义所有以 `<` 开头的信封标签字样（开/闭/自闭合），验证点覆盖伪造开标签用例。
5. **攻击"每轮 8 条、队里 50 条时父代理永远看不到第 9 条"**：不会——未投递的留在 `m.completed`，下轮继续 drain（现逻辑 L1183-1203 已保留下轮），前提是 T1 实现不把剩余条目误清空。验证点：T5 的"余留队列"断言。
6. **攻击"缓存红线被破坏"**：注入块在 user turn 首部、动态生成，未改动 system/tool 前缀与已发送消息。但若 T3 误把渲染函数调进 `ComposeSynthetic` 或误改 `reComposeBlock` 会影响 display。验证点：T3/T4 断言 + cachehit e2e 回归。
7. **攻击"这个方案只是把摘要变长，没有解决'等待到完成'的主动唤醒"**：承认——P1 定义为"结果在**下一条 user 消息**投递"，不包含"主动 push/打断 idle 会话"（那是 Qwen 500ms 轮询注入，我方红线禁止 mid-turn 注入）。#7962 的"父代理轮询"痛点在父代理**继续发言**的场景下被消除；父代理完全静默的场景不在 P1 范围，标注为 P2/P3 候选。

---

## 五、缓存/纪律检查点（Reasonix 领域）

- **发送侧字节变化**：P1 只改 **user turn 首部动态块**的内容（`<background-jobs>` 内从一行摘要变信封）；**system prompt、tool schema、已发送消息字节零变化** → provider 前缀缓存零失效。验证：`cachehit_e2e_test.go` 回归 + 手工核对 T3 不触碰 `ComposeSynthetic`/系统前缀。
- **前缀稳定性**：注入位置固定（input.go L191-195 原位置）、容器标签固定（`background-jobs`）、不插入历史、不 mid-turn 注入（红线）。
- **bounded 纪律**：三档上限（4KB/条、8 条/次、16KB/块）由纪律团复核数值；`m.completed` 队列本身也应加上限防长期不发言场景内存膨胀（T5 覆盖）。
- **防虚假完成**：本计划未改任何源码；交付前纪律团须核实 execution.md 引用的每个命令输出与 git diff 真实存在，`completion` 结构变更不得以"只改了注释"蒙混。
- **越权审查红线**：自动投递不得触发 evidence lease/commit；纪律团专项核查 T2 diff 无 `LeaseEvidence`/`CommitEvidence` 调用。
