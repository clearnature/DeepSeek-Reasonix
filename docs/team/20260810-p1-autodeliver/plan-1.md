# 计划：P1 后台结果自动投递（plan-1.md）

> 生成：2026-08-10 · 分支：team · 规划：Planner（仅计划，交由执行小队实施 + 纪律团审查）
> 关联：`docs/MULTIAGENT_QWEN_COMPARISON.md` 三.🟢P1（P0-3 锚点）、`docs/team-org.md`（组织章程）、memory：team-multiagent-qwen-comparison
> 目标：社区 #7962 —— 后台子代理结果不再需要父代理 wait/bash_output 轮询，改为 bounded 信封自动投递进父下一条 user 消息尾部。

---

## 一、拓扑扫描

### 1.1 现状锚点（grep/read 已验证）

| 位置 | 现状 | 与 P1 的关系 |
|------|------|-------------|
| `internal/jobs/jobs.go:778-814` `recordCompletion` | 仅 append `completion{sessionID, text}`（一行 `"id (label) — status"`）+ emit Notice；正文不投递 | **改造主入口**：扩展为结构化完成项并携带 bounded 正文 |
| `internal/jobs/jobs.go:524-530` | 注释与代码：**先入队完成项、后发布 terminal status**（防 Wait 竞态，TestDrainMultiple -race） | P1 必须保持该时序不变量 |
| `internal/jobs/jobs.go:183-186` `completion` | `{sessionID, text}` 纯文本 | 升级为结构化 `completionResult` |
| `internal/jobs/jobs.go:1185-1210` `DrainCompletedNoteForSession` | 按 session **全清**队列并 join 一行摘要 | 新增结构化 drain（**部分 drain + bounded**）；旧方法保留兼容测试 |
| `internal/jobs/jobs.go:870-900` `OutputForSession` | **消费性**读取：推进 `readOffset` / `resultRead` | wait/bash_output 依赖此语义，**不可复用**于信封快照 |
| `internal/jobs/jobs.go:1053-1074` `results`（wait 用） | **非消费**读取：`j.result → readArtifactAllLocked → tail` | 信封正文快照的**模板**（仿此写非消费 snapshot） |
| `internal/jobs/jobs.go:946-957` `readArtifactAllLocked` | `os.ReadFile` 全量读 artifact（有意的 raw 字节语义，见 941-945 注释） | 大 artifact 全量入内存，信封快照需 **LimitReader 限量读** |
| `internal/agent/task.go:924-958, 1741-1753` | 后台 task 的 `result` = `FormatSubagentRunResult`（含 `Subagent reference: sa_…` + `Final answer:`） | 信封正文的**典型样本**；正文可能含 markdown/XML 尖括号 → 必须转义 |
| `internal/control/input.go:191-195` | `<background-jobs>` **前缀**注入 user 消息 | 改造为 `<background-job-result>` **尾部**注入 |
| `internal/control/input.go:203-208` | memory-recall **尾部**注入先例（`TrimRight + "\n\n" + block`） | 新信封沿用此尾部模式 |
| `internal/control/turn_orchestrator.go:202`、`controller.go:1885` | 两条真实入口都走 `Compose`/`composeWithGoal` | 单点注入即全覆盖（orchestrated + headless） |
| `internal/agent/preview.go:19-29` `TransientUserBlockTags` | **前缀**块白名单（含 background-jobs） | 新标签是**尾部**块，不能进该前缀列表；需独立尾部剥离 |
| `internal/agent/preview.go:51-57, 102-113` | `stripTrailingDeliveryRuntime` / `stripTrailingMemoryRecall` 尾部剥离先例 | 新信封的尾部剥离挂进这条链 |
| `internal/history/strip.go:8` `reComposeBlock` | 前缀锚定剥离 | 需补尾部剥离，否则历史渲染泄漏 raw 信封 |
| `internal/jobs/artifacts.go:38-52` `artifactMeta` | 无 usage/token 字段 | **usage 无数据源**（见 1.3 风险） |

### 1.2 级联风险清单

1. **R1 消费性读取冲突**：若用 `OutputForSession` 拉正文做信封，会推 `readOffset`/置 `resultRead` → 父代理随后 wait/bash_output 取不到全量，**破坏需求 4 兼容红线**。→ 必须新增非消费 snapshot。
2. **R2 锁顺序**：`recordCompletion` 现在只持 `m.mu` 不碰 `j.mu`；若在持 `m.mu` 时新增读 `j.mu`，需审计是否存在反向 `j.mu→m.mu` 路径（疑似无，但执行时用 `-race` 验证）。计划内规避：**不嵌套持锁**，snapshot 在 m.mu 释放后、再入队前。
3. **R3 时序不变量**（jobs.go:524-530）：信封必须在 status 发布前入队，否则 `Wait` 观察到 terminal 后立即 drain 会漏。P1 把 snapshot+入队放在 recordCompletion 内、先于 `close(j.done)`。
4. **R4 转义放大**：正文含 `&`/`<` 时 XML 转义最多 4× 放大 → 截断必须按**转义后字节**兜底，否则 bounded 失效。
5. **R5 剥离遗漏**：新标签若漏进 preview/title/history 剥离链，raw 信封会泄漏进 UI（`<autoresearch-runtime>` 先例）→ 必须纳入 preview.go 尾部链 + history/strip.go。
6. **R6 usage 无数据源**：`Job`/`artifactMeta`/taskmonitor/evidence 均无 token usage；后台 task 的 agent 句柄在 `runSession` 内部，`recordCompletion` 拿不到 → 信封 usage 字段只能**占位 + 字节代理**，否则就是虚假完成。
7. **R7 队列无界**：现 `completed` 队列只增不减（父代理不发言时堆积）。P1 需队列硬上限 + 丢弃计数。
8. **R8 参考文档缺失**：`docs/team-plan.md` 当前工作树未检出（`glob docs/**/*.md` 无匹配）；`<task-notification>`/CCB 无符号匹配。P1 定义以 `MULTIAGENT_QWEN_COMPARISON.md` 三.🟢P1 为准，CCB 仅作外部参考模式。

---

## 二、多路径推演

### 方案 A（推荐）：jobs 结构化队列 + bounded snapshot + control 尾部注入

- **jobs**：`completion` 升级为结构化 `completionResult{sessionID, id, kind, label, status, body(已 bounded+转义), artifactRef, truncated, bytes}`；`recordCompletion` 用**非消费 snapshot**（仿 `results()`，`io.LimitReader` 限量读）取正文、rune 安全截断、XML 转义后入队；新增 `DrainCompletedEnvelopesForSession` 渲染 `<background-job-result>` 信封（部分 drain：每轮 ≤8 条，剩余留队；队列硬上限 64，超限丢最旧并记 dropped）。
- **control**：`input.go` 用新 drain，`text = TrimRight(text,"\n") + "\n\n" + envelope`（尾部，仿 memory-recall 先例）；移除 `<background-jobs>` 前缀注入（剥离列表保留以兼容历史会话）。
- **剥离**：`preview.go` 尾部剥离链加入新标签（泛化 `stripTrailingBlock`）；`history/strip.go` 补尾部剥离。
- **复杂度**：中（跨 4 个包联动）；**性能**：snapshot 限量读，大 artifact 不爆内存；**可维护性**：bounded/转义/截断集中在 jobs 数据层，control 薄；**风险**：低（-race 覆盖锁序）。
- **关键取舍**：正文快照**非消费**（不推 offset/resultRead）→ wait/bash_output 全量兼容；旧 `DrainCompletedNoteForSession` 保留 → 既有测试零改动。

### 方案 B：仅扩展现有 completion.text（把正文塞进一行文本）

- 改动最小（只动 recordCompletion + 摘要拼接）。
- **否决**：无法结构化（task_id 靠文本解析）、无法转义（正文含 `</…>` 破坏信封）、仍是前缀注入（不符尾部要求）、部分 drain/溢出策略无法表达。长期不可维护。

### 方案 C：control 侧回读（drain 时调 OutputForSession 拉正文）

- jobs 改动最小。
- **否决**：`OutputForSession` 是**消费性**读取 → 推进 `readOffset`/置 `resultRead` → 父代理后续 wait/bash_output 取不到全量，**直接违反需求 4**；且 bounded 逻辑散落 control，drain 与 Output 竞态。

### 选型结论

**选 A**。理由：唯一同时满足 ① 结构化信封 ② 尾部落点 ③ bounded 硬顶 ④ wait/bash_output 全量兼容 ⑤ 时序不变量 ⑥ 剥离链完整的方案。B/C 分别死于可维护性和兼容红线。

---

## 三、设计规格（A 方案细化）

### 3.1 信封结构 `<background-job-result>`

```xml
{父 user 文本}

<background-job-result>
<result task_id="task-7" kind="task" status="done" label="write plan"
        artifact=".reasonix/jobs/tasks/task-7.log" bytes="12840"
        truncated="true" usage="n/a">
&lt;正文，XML 转义后，&lt;= 4096 字节（转义后）&gt;
</result>
<result task_id="bash-3" kind="bash" status="failed" artifact="…" bytes="2048" truncated="false" usage="n/a">
command exited 1…
</result>
<result-overflow count="2"/>
</background-job-result>
```

- 信封标签固定 `<background-job-result>`（顶层一个块、内嵌多个 `<result>` + 可选 `<result-overflow>`）。
- 字段（用户要求全覆盖）：
  - `task_id`：job.ID（如 `task-7`）
  - `status`：`done | failed | killed`（Running 不投递）
  - 结果正文：`<result>` 元素文本，**XML 转义**（`&` `<` `>` → 实体），bounded（见 3.3）
  - `artifact`：引用 job artifact 相对路径（`filepath.Base(j.artifactPath)` 前缀会话相对目录），保证全量可达
  - `usage`：**P1 占位 `"n/a"`**（R6：jobs 层无 usage 数据源，如实标注，禁止编造）；`bytes` 属性给出正文原始字节数作为量级代理
  - `truncated`：`true` 表示正文被截断，提示 wait/bash_output 取全量
- 转义而非 CDATA 的原因：子代理正文常含 `]]>`/畸形 XML（`FormatSubagentRunResult` 样本含 `</review-report-contract>` 类内容），转义方案自洽、可安全嵌套、无歧义。

### 3.2 注入落点（需求 2）

- **落点**：父**下一条** user 消息（orchestrated 与 headless 均经 `Compose`，单点覆盖）；synthetic continuation（`turn_orchestrator.go:202`）同样受益。
- **位置**：user 消息**尾部**，`strings.TrimRight(text, "\n") + "\n\n" + envelope`，固定追加在末尾；无完成项时 user 消息与现状逐字节一致（唯一差异：移除 `<background-jobs>` 前缀块）。
- **不插入历史**：注入只发生在 Compose 时该条 user 消息内；不修改 system prompt / tools / 既有历史消息字节（缓存红线，见第六节）。
- **剥离链（防泄漏）**：
  - `preview.go`：`stripTrailingMemoryRecall` 泛化为 `stripTrailingBlock`（open/close 参数化），链式处理 `<memory-recall>` 与 `<background-job-result>`；`StripTransientUserBlocks`/`UserPreviewText`/标题派生自动覆盖。
  - `history/strip.go`：补尾部块剥离（现 reComposeBlock 仅前缀锚定）。
  - `TransientUserBlockTags`：**不加入**（前缀白名单，不适用尾部块；加注释防误加）。
  - `StripComposePrefixes`（input.go:65-71）走 `agent.StripTransientUserBlocks`，自动继承。

### 3.3 bounded 策略（需求 3）

| 常量 | 值 | 说明 |
|------|----|----|
| `maxResultBodyBytes` | 4096 | 每条正文上限，**转义后**字节计（防 R4 转义放大） |
| `maxResultEnvelopeEntries` | 8 | 每轮注入最多条数；其余**留队**下一轮（部分 drain） |
| `maxQueuedCompletions` | 64 | `completed` 队列硬上限；超限丢最旧并累计 `dropped` 计数 |
| 信封总预算 | ~8×4KB + 固定开销 | 条目×条目上限隐含总量硬顶，无额外总量校验 |

- **截断规则**：快照读取用 `io.LimitReader`（只读 `maxResultBodyBytes` 余量，大 artifact 不爆内存；注意不要复用 `readArtifactAllLocked` 的 `os.ReadFile` 全量语义）→ **rune 安全**截断（不得切断多字节字符）→ XML 转义 → 若转义后仍超上限则按转义后字节二次截断 → 附 `…[truncated; use wait/bash_output for full output]`。
- **溢出语义**：`<result-overflow count="N"/>` 告知父代理有 N 条因队列上限被丢弃；被丢结果仍可通过 wait（wait 读 job 表而非 completion 队列）恢复 → 自动投递丢、手动取不丢。

### 3.4 与 wait/bash_output 的兼容（需求 4）

- 信封正文快照**非消费**：不推进 `readOffset`、不置 `resultRead`（新增 `snapshotResultForEnvelope`，仿 `results()` 模板）。
- drain 之后 `OutputForSession`/`Wait` 仍返回全量正文与全量 artifact。
- 时序保持 jobs.go:524-530 不变量：**入队信封 → 发布 terminal status → close(j.done)**；`startInvalid`（jobs.go:406）失败路径同样入队 `status=failed` 信封。
- `recordStalled` 不投递正文（stalled 非终局，维持摘要 + 提示 wait），P1 不动。
- 锁序：`recordCompletion` 重构为「① `m.mu` 取 job 句柄+destroying 检查 → 释放；② `j.mu` snapshot bounded 正文 → 释放；③ `m.mu` 入队 completionResult → 释放；④ emit Notice」。**全程不嵌套 m.mu/j.mu**，无死锁面；步骤②③窗口内 job 被 destroy 属会话销毁中，丢弃可接受（与现有 destroying 检查语义一致）。

---

## 四、任务分解（子任务 + 验证点）

- [ ] **T1 jobs 结构化完成项 + bounded snapshot**
      - `completion` → `completionResult`；常量（3.3 四值）；`escapeEnvelopeText`/`boundedBody`（rune 安全 + 转义后兜底）；`snapshotResultForEnvelope`（LimitReader、非消费）；`recordCompletion` 重构（锁序 3.4）+ `startInvalid` 失败路径。
      - 验证：新单测 `TestEnvelopeRuneBoundary`（切断多字节字符的输入）、`TestEnvelopeSnapshotNonConsuming`（drain 后 OutputForSession 仍全量）、`TestEnvelopeEscape`（含 `</…>`/`&`/`]]>` 正文）；`TestDrainMultiple -race` 仍绿。
- [ ] **T2 DrainCompletedEnvelopesForSession（部分 drain + 渲染）**
      - 返回 `DrawnEnvelope{XML, Remaining, Dropped}`；每轮 ≤8 条、留队、64 上限丢弃计数；渲染 `<result>`/`<result-overflow>`；`DrainCompletedNoteForSession` **保留**（deprecated 注释：与新方法互斥使用，供测试/兼容）。
      - 验证：`TestEnvelopeShape`（task_id/status/label/artifact/usage/truncated 属性齐全）、`TestEnvelopePartialDrain`（9 条完成 → 首轮 8 + Remaining=1 → 次轮 1）、`TestEnvelopeQueueCap`（65 条 → dropped≥1）；既有 `DrainCompletedNoteForSession` 测试全绿。
- [ ] **T3 control 尾部注入**
      - `input.go` 改用新 drain、尾部 append（TrimRight+"\n\n"+envelope）；移除 `<background-jobs>` 前缀注入；确认 `turn_orchestrator.go:202` 与 `controller.go:1885` 两条路径都覆盖。
      - 验证：`input_test` 断言信封在 user 文本**之后**、位置固定、无完成项时不出现；`StripComposePrefixes(composed) == 用户原文`。
- [ ] **T4 显示/历史剥离（缓存红线配套）**
      - `preview.go`：泛化 `stripTrailingBlock` 并链入 `<background-job-result>`；`history/strip.go` 补尾部剥离；`TransientUserBlockTags` 加注释防误加。
      - 验证：`TestUserPreviewTextStripsEnvelope`、标题/预览派生不泄漏 raw 信封（`<autoresearch-runtime>` 教训回归）。
- [ ] **T5 兼容与端到端回归**
      - 非消费验证：drain 后 wait 仍取全量；后台 task 端到端（起任务→完成→父下一轮尾部信封→wait 全量）。
      - 验证：`go test ./internal/jobs ./internal/control ./internal/agent -race` 相关包全绿；e2e 手动脚本记录在 execution.md。
- [ ] **T6 文档交付**
      - 本 plan-1.md；执行后由执行小队产 execution.md（证据链）、纪律团产 review.md。

---

## 五、对抗自检（devil's advocate）

1. **攻击：usage 字段无法填 → 是不是虚假功能？**
   是已知缺口：jobs 层拿不到子代理 token usage（agent 句柄在 runSession 内部，taskmonitor/evidence 无 usage）。**对策**：信封如实标 `usage="n/a"` + `bytes` 代理，绝不编造数字；真实 usage 归入 P2（job 消息通道）或后续事务，本计划不承诺。
2. **攻击：部分 drain 改变"全清"语义 → 父代理长期不发言，队列到 64 上限丢最旧结果。**
   **缓解**：丢弃只影响"自动投递"，wait 读 job 表全量仍在 → 兼容红线兜底；信封带 `<result-overflow count>` 显式告知；丢最旧（而非最新）保最新结果优先。
3. **攻击：锁序改动引入死锁（m.mu/j.mu）。**
   **对策**：计划 3.4 强制"不嵌套持锁"（三步分段锁）；`TestDrainMultiple -race` 已有历史覆盖该竞态（jobs.go:524-530 注释），T1 必须保持其绿。
4. **攻击：XML 转义 4× 放大击穿 bounded。**
   **对策**：截断口径定为"转义后字节"（3.3），`boundedBody` 双段校验（转义前 rune 截断 + 转义后兜底二次截断）；T1 单测含大量 `&`/`<` 输入。
5. **攻击：剥离遗漏 → raw 信封泄漏进标题/预览（`<autoresearch-runtime>` 教训）。**
   **对策**：T4 专列；新标签走 `stripTrailingBlock` 链而非前缀白名单（防"误加进 TransientUserBlockTags 却不匹配尾部"的假修复）；测试断言标题/预览零泄漏。
6. **攻击：尾部注入与 memory-recall 尾部并存时顺序漂移。**
   **对策**：位置固定（信封恒在 user 文本之后、相对位置固定），T3 用字符串断言固定顺序，避免后续改动悄悄移位。
7. **攻击：`<background-jobs>` 移除后，历史会话回放/Strip 还匹配它吗？**
   **匹配**（剥离列表保留该标签）→ 历史安全；但 input.go 不再注入新 `<background-jobs>`。T3/T4 覆盖。
8. **攻击：DrainCompletedNoteForSession（全清）与新方法（部分 drain）混用会互吃队列。**
   **对策**：文档 + deprecated 注释标注互斥；input.go 只用新方法；测试确认两方法各自独立 green。

---

## 六、缓存/纪律检查点（Reasonix 领域）

- **发送侧字节变化**：信封 append 到 **user 消息尾部**，属请求可变尾部；system prompt / tools / 既有历史消息字节零变化 → 稳定前缀缓存命中不受影响（与 memory-recall/steer 尾部先例同一保证）。
- **前缀稳定**：注入位置固定（尾部、仅在有完成项时出现）；无完成项时 user 消息与现状一致；移除 `<background-jobs>` 前缀块只改变 user 消息自身头部（仍在稳定前缀之后），不影响缓存前缀。
- **不插入历史**：注入仅发生在 Compose 当时该条 user 消息；不改写历史消息；信封随 user 消息持久化但由 T4 剥离链在显示/标题/回放层剔除。
- **防虚假完成**：usage 占位如实标注；所有"完成"声明须附测试证据（纪律团按 team-org.md 第六节交叉验证）。
- **bounded 审计**：4 常量集中定义在 jobs，单测覆盖截断/溢出/转义边界。

---

## 七、遗留/假设

- `docs/team-plan.md`（用户引用的 P1 定义）当前工作树未检出；CCB `<task-notification>` 无符号匹配 → P1 定义以 `MULTIAGENT_QWEN_COMPARISON.md` 三.🟢P1 为准，CCB 仅作外部注入模式参考。
- 真实 token usage 入信封需跨层改造（agent→jobs 传递），本 P1 不做，归 P2；已通过 `usage="n/a"` + `bytes` 留出字段位。
- `recordStalled` 不投递正文（非终局），若社区期望 stalled 也投递，另立事务。
