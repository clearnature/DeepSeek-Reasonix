# 计划：P1 后台结果自动投递（plan-2）

> 生成：2026-08-10 · 分支：team（基点 main-v2 07fcf585b）· 职能：规划 Planner（fable5 Step 1-3）
> 关联：`docs/team-org.md`（团队章程）、`docs/MULTIAGENT_QWEN_COMPARISON.md` §三.P1（Qwen mailbox 借鉴 + 我方落地要点）
> ⚠️ `docs/team-plan.md` 被 team-org.md 引用但仓库中**不存在**（glob 未命中）——本计划 P1 定义依据：任务描述 + QWEN 对比文档 P1 节 + 代码现状（jobs.go:778-814 / input.go:191-195）。见「遗留问题」。
> 本文件为**规划产出**，实现交由执行小队、审查交纪律团，本文件不包含实现代码。

## 目标（社区 #7962）

后台子代理/后台命令完成后，把**结果正文**（而非仅一行摘要）自动投递给父代理下一条 user 消息，替代「父代理必须 wait/bash_output 轮询取正文」的现状。红线不变：**append turn 尾部、位置固定、不插入历史、前缀字节稳定（缓存）**。

---

## 一、拓扑扫描

### 1.1 文件依赖图（已逐行核实）

| 文件 | 现状锚点 | 与本事务的关系 |
|------|---------|---------------|
| `internal/jobs/jobs.go` | `completion` struct L183-186（仅 `sessionID`+`text`）；`recordCompletion` L778-814（只 emit Notice + 一行摘要 `tag — st`）；`recordStalled` L816-839（append 摘要文本）；`DrainCompletedNoteForSession` L1185-1210（join 文本 + 尾缀 `"Read their output with bash_output or wait..."`）；`DrainCompletedNote` L1178-1180（legacy 空 session）；`results()` L1053-1074（终态正文 fallback 顺序：result → artifact-all → tail） | **核心改造点**：completion 结构化 + recordCompletion 快照正文 + drain 升级 |
| `internal/jobs/artifacts.go` | `defaultTailBytes=64KB` L19、`appendTail` L188 | tail 缓冲常量参照；注入截断常量另立 |
| `internal/control/input.go` | `composeWithGoal` L152-217；`<background-jobs>` 前缀注入 L191-195（prepend 在 memory-update 之后、hook-context 之前） | **注入落点改造**：prepend → append turn 尾部 |
| `internal/agent/preview.go` | `TransientUserBlockTags` L19-29（含 `background-jobs`）；`reTransientUserBlock` 前缀 regex（`^` 锚定）；`stripTrailingDeliveryRuntime` L51-57 / `stripTrailingMemoryRecall` L102-113（byte-exact 尾部剥离先例） | **UI 剥离改造**：尾部 `<background-job-result>` 不在前缀 regex 覆盖内，必须新增尾部剥离，否则泄漏进 preview/title（#3653/#5307 同类事故） |
| `internal/control/input_test.go` | L1386-1389 StripComposePrefixes 前缀块测试；L592 等 compose 断言 | 受影响 + 需新增断言 |
| `internal/jobs/jobs_test.go` / `artifacts_test.go` | L321-327（`Contains(note,"Done")`+`Contains(note,j.ID)`）、L347-353、L578-589、L652/706/748、artifacts L908（均 `Contains(id)` 或非空断言） | 受影响（多数 `Contains` 兼容，个别需核对精确文本） |
| `internal/control/controller_test.go` | L188/303/587 构造 `jobs.NewManager(event.Discard)` | compose 回归面 |
| `internal/jobs/concurrency_stress_test.go` | L36 并发 `DrainCompletedNote()`（忽略返回值） | 并发回归面 |

### 1.2 级联风险清单

1. **注入位置 prepend→append 打破前缀剥离机制**：`reTransientUserBlock` 只匹配用户消息**开头**的块。`<background-job-result>` 若 append 尾部，`StripTransientUserBlocks` 不剥离 → preview/session title/rewind picker 泄漏原始 XML 标记（#3653 事故模式）。
2. **尾部剥离顺序陷阱（高）**：设计若让 bg-result 块排在 memory-recall **之后**（最末），则 `stripTrailingMemoryRecall` 的 `HasSuffix("</memory-recall>")` 因后缀是 `</background-job-result>` 而**匹配失败，memory-recall 残留泄漏**。必须「先剥 bg-result、再剥 memory-recall」（从外到内）。
3. **锁序风险**：`recordCompletion` 当前只持 `m.mu`。若在其内再取 `j.mu` 读快照，需确认与既有嵌套方向（`findJobLocked` 持 `m.mu` 但不锁 `j`；`results()` 只锁 `j`）不构成反向嵌套 → 用 `-race` 验证。规定顺序：**先 `j.mu` 快照、后 `m.mu` 排队**。
4. **`recordCompletion` 时 `j.status` 仍是 Running**：L532-535 在 recordCompletion **之后**才写终态 status（防 wait/drain 竞态 flake 的既有设计）。信封 status 必须用**传入参数 `st`**，不能读 `j.status`。
5. **正文读取必须只读**：自动投递若调用 `OutputForSession` 会推进 `readOffset`/置 `resultRead=true`，吞掉后续 `bash_output` 增量输出 → 必须用 `results()` 同款**只读** fallback（result → artifact-all → tail），不动读取游标。
6. **artifact 已 move**：`moveArtifactToDirLocked`（L514-518，更新 `artifactPath` L738）在 recordCompletion 之前完成 → 快照时 `artifactPath` 是最终路径，可作为信封 artifact 引用。
7. **drain 文本格式变更波及 legacy 调用**：`DrainCompletedNote()`（L1178）仅被测试调用（jobs_test L321/347/455、jobs_extra L134、concurrency_stress L36）；L321-327 断言 `Contains(note,"Done")` → 信封 status 属性建议用 `st.String()`（`Done/Failed/Killed`）保持兼容，避免大规模改测试。
8. **旧 transcript 残留** `<background-jobs>` 前缀块 → `TransientUserBlockTags` 中 `background-jobs` **保留**（仅用于剥离旧块），新注入不再产生它。
9. **拦截链**：`interceptInputReceive`（turn_orchestrator L217）在 compose 之后、进 session 之前运行，扩展链可能再改文本 → 位置断言放在 compose 层（尾部后缀），不假设 provider 收到的最终形态。
10. **canonical 持久化**：含信封的 user 消息会被永久存入 canonical transcript（与 memory-update 同模式），恢复后不会重注入（drain 已消费）→ 无重复投递；历史中可见旧信封块为可接受行为。

---

## 二、多路径推演

### 方案 A（推荐）：统一尾部信封，替换前缀摘要

`<background-jobs>` 前缀注入**整体移除**，升级为单一 `<background-job-result>` 信封 **append 到 user turn 尾部**（recall 之后、最末位）。完成（Done/Failed/Killed 带 bounded 正文）与 stalled 通知（无正文）统一进同一信封，`status` 区分。

- **复杂度**：中高。动 jobs.go + input.go + preview.go + 三处测试族。
- **性能**：正文注入 ≤16KB/turn，队列一次消费；无额外 IO。
- **可维护性**：单一注入点、单一剥离点、单一语义（「后台状态+结果」一处看齐）。
- **风险**：改动面最大；尾缀文案 `"Read their output with bash_output..."` 语义反转（正文已投递，无需再提示读全文——但超限降级项仍需要）。

### 方案 B（备选）：保留前缀摘要 + 新增尾部正文块

`<background-jobs>` 前缀原样保留（零 UI 改动、测试零波及），另 append 尾部 `<background-job-result>` 正文块。

- **复杂度**：低。jobs.go 只需新增 drain 路径，input.go/preview.go 增量小。
- **风险**：双机制并存语义冗余——模型同时看到「前缀摘要说 read with bash_output」与「尾部正文已投递」的矛盾指令；两个注入点、两个剥离逻辑；信封命名与摘要职责割裂。
- **可维护性**：差。未来维护者需理解两套 background 注入。

### 方案 C（否决）：保持前缀位置仅升级正文
被任务红线否决——用户明确要求「append turn 尾部」。不展开。

### 选优结论

**选 A**。理由：(1) 与任务「信封结构（`<background-job-result>`）」直接对应；(2) 消除「提示读 bash_output」与「正文已投递」的噪音冲突；(3) 单注入点 = 单缓存关注点 + 单剥离点；(4) 尾部最末位符合模型 recency 关注。**B 作为 A 出问题时的回滚备选**（改动集中在 jobs.go 新增路径，可快速切换）。

---

## 三、设计定稿

### 3.1 信封结构（`<background-job-result>`）

```
<background-job-result>
<job task_id="bash-3" kind="bash" label="build" status="Done" artifact="/abs/path/session-jobs/bash-3.log" usage="">
<output>
...bounded 正文（result → artifact-all → tail fallback，只读快照）...
</output>
</job>
<job task_id="task-1" kind="task" status="Failed" artifact="...">
<output>...</output>
<error>...err.Error()...</error>
</job>
<job task_id="bash-4" kind="bash" status="stalled" artifact="...">
<output>...stalled 提示文案...</output>
</job>
</background-job-result>
```

逐项回应任务要求：

| 要求 | 设计 |
|------|------|
| **task_id** | 属性 `task_id`，必填，取自 job.ID |
| **status** | 属性 `status`，取 `st.String()`（Done/Failed/Killed，与现有文本语义/测试断言兼容）；stalled 通知用 `status="stalled"`（非终态，无正文快照语义，仅提示文案） |
| **结果正文 bounded** | 见 3.2 |
| **artifact 引用** | 属性 `artifact`（可选，omitempty）：最终 `.log` 路径（recordCompletion 时已 move 完成），供父代理自行读全文 |
| **usage** | 属性 `usage`（**预留、当前恒空 omitempty**）——jobs 包无 usage 跟踪（grep 证实），真实 token 数依赖子代理会话汇总，属后续（P2/P4 范畴），本事务**不虚报**该字段值 |

- **转义**：属性值转义 `& < > " '`；正文转义 `& < >`（防信封被正文中的 XML/代码标签破坏，参考 Qwen escapeEnvelopeTags 结构性转义哲学）。
- 多作业按完成顺序排列多个 `<job>` 块；一条消息内同一 job 只出现一次（drain 消费）。

### 3.2 bounded 策略（防上下文膨胀）

常量（新建 `internal/jobs/bgdeliver.go`）：

```go
maxBackgroundResultBodyBytes  = 4 * 1024   // 单作业正文上限
maxBackgroundResultJobs       = 8          // 单 turn 携带正文的作业数上限
maxBackgroundResultTotalBytes = 16 * 1024  // 单 turn 信封总字节上限
```

截断规则（双层保护）：
1. **单作业**：正文 > 4KB → 保留**尾部** 4KB（shell/task 最终结论在末尾，`defaultTailBytes=64KB` 的既有取舍一致），前缀追加 `[... truncated <n> bytes ...]\n` 标记。
2. **总量**：排队作业 > 8 条 或 累计信封 > 16KB → 后续作业降级为**摘要块**（仅属性 + `full output: use bash_output or wait`），但**队列全部消费**（不滞留到下一轮，防重复/滞留）。
3. **空正文**（killed/无输出/纯 stalled）：省略 `<output>` 元素。

### 3.3 注入落点（input.go）

- **删除** L191-195 的 `<background-jobs>` 前缀 prepend 段。
- **追加**：`composeWithGoal` return 之前（memory-recall 注入之后，**最末位**）：
  ```
  if c.jobs != nil {
      if block := c.jobs.DrainCompletedNoteForSession(c.parentSessionID()); block != "" {
          text = strings.TrimRight(text, "\n") + "\n\n<background-job-result>\n" + block + "\n</background-job-result>"
      }
  }
  ```
- 位置固定性：无论 goal/plan/memory/recall/hook 各条件如何组合，`<background-job-result>` 恒为 user 消息**尾部后缀**。synthetic/continuation turn 同样注入（`c.jobs != nil` 统一判断）。
- 不插入历史：只影响**本条新 user 消息**的内容，不改写 canonical 已存在消息。

### 3.4 与 wait/bash_output 兼容

- 自动投递正文 = **只读快照**：recordCompletion 时在 `j.mu` 下读 `result → artifact-all → tail`（复用 `results()` 的 fallback 逻辑，提取共享 helper），**不推进 `readOffset`、不置 `resultRead`**。
- `bash_output` 增量语义、`wait` 阻塞语义、`kill_shell` 不变；模型后续仍可调 `bash_output` 拿全文。
- status 用参数 `st`（见级联风险 4），与既有「先排队后置终态」的防竞态顺序不冲突。

---

## 四、任务分解（子任务 + 验证点）

### 子任务 1：`internal/jobs/bgdeliver.go` 新建（构建/转义/截断）

**Files**：Create `internal/jobs/bgdeliver.go`、`internal/jobs/bgdeliver_test.go`

- [ ] 常量 `maxBackgroundResultBodyBytes/maxBackgroundResultJobs/maxBackgroundResultTotalBytes`
- [ ] `escapeXML`（属性全转义 + 正文 `<>&` 转义）
- [ ] `boundedBody(text)`（尾部 4KB + 省略标记）
- [ ] `buildJobResultBlock(...)`（组装单个 `<job>` 块；status/正文/error/artifact/usage）
- [ ] 写失败测试 → 实现 → 跑测试
- 验证：
  - `go test ./internal/jobs -run 'TestBuildJobResultBlock|TestBoundedBody|TestEscapeXML' -count=1`
  - 断言：超限保尾部+省略标记、属性转义、空正文省略 `<output>`、usage 空时属性省略、error 仅 Failed 时出现

### 子任务 2：completion 结构化 + recordCompletion/recordStalled 快照化

**Files**：Modify `internal/jobs/jobs.go`、`internal/jobs/jobs_test.go`、`internal/jobs/artifacts_test.go`

- [ ] `completion` struct 扩展（sessionID + 结构化 job 快照字段，或直接存渲染块文本）
- [ ] 提取只读终态快照 helper（result → artifact-all → tail，不碰 readOffset/resultRead）
- [ ] `recordCompletion` 改为：`j.mu` 快照（status 用参数 `st`）→ `m.mu` 排队（锁序：先 j 后 m）
- [ ] `recordStalled` 改为构建 `status="stalled"` 块
- 验证：
  - `go test ./internal/jobs -count=1`（重点：TestStartWaitDoneAndDrain L347、TestSessionScoped* L578-589、L321 断言 `Contains("Done")` 须兼容、artifacts_test L908）
  - `go test -race ./internal/jobs -run 'TestDrainMultiple|TestSessionScoped|TestConcurrentDrain' -count=1`（锁序无死锁）

### 子任务 3：`DrainCompletedNoteForSession` 升级（块序列 + 总量截断）

**Files**：Modify `internal/jobs/jobs.go`、`internal/jobs/jobs_test.go`

- [ ] drain 返回 `<job>` 块序列（保持 session 隔离语义：非本 session 的完成项滞留队列）
- [ ] 超 8 条 / 16KB → 后续降级摘要块，**队列全消费**；空时返回 `""`
- [ ] 尾缀文案改为超限时 `use bash_output or wait for full output`（正文已投递时不再提示）
- 验证：
  - `go test ./internal/jobs -run 'TestDrain|TestStartWaitDoneAndDrain|TestSessionScoped' -count=1`
  - 新增：9 个作业 → 第 9 个为摘要块但 task_id 仍在；二次 drain 返回空

### 子任务 4：input.go 注入落点（prepend → append 尾部）

**Files**：Modify `internal/control/input.go`、`internal/control/input_test.go`、必要时 `controller_test.go`

- [ ] 删除 L191-195 前缀段
- [ ] return 前追加尾部信封（见 3.3）；与 memory-recall 相对顺序固定（bg-result 恒最末）
- [ ] 既有 compose 测试回归（input_test L592/597/702-709/821/833、controller_test L364/803/2549/4807/4856）
- 验证：
  - `go test ./internal/control -run 'TestCompose|TestStrip' -count=1`
  - 新增：compose 返回文本**以 `</background-job-result>` 结尾**（位置固定断言）；无 jobs 时不注入；synthetic turn 注入；与 memory-update/hook 前缀共存顺序

### 子任务 5：UI 剥离（preview.go，含剥离顺序陷阱）

**Files**：Modify `internal/agent/preview.go`、`internal/agent/preview_test.go`（或对应测试文件）

- [ ] 新增 `stripTrailingBackgroundJobResult`（byte-exact，仿 stripTrailingMemoryRecall）
- [ ] `StripTransientUserBlocks` 调用链：**先 strip bg-result、再 strip memory-recall**（bg-result 在 recall 之后，从外到内，否则 memory-recall 残留泄漏——级联风险 2）
- [ ] `TransientUserBlockTags` 保留 `background-jobs`（兼容旧 transcript 剥离），注释说明新块走尾部函数
- 验证：
  - `go test ./internal/agent -run 'TestStripTransientUserBlocks|TestStripTrailing' -count=1`
  - 新增：尾部 bg-result 剥离后 `StripComposePrefixes` 还原纯用户文本；用户文本含不成对标签不误伤；截断/悬空标签不剥离

### 子任务 6：全量回归 + 文档标注

**Files**：Modify `docs/MULTIAGENT_QWEN_COMPARISON.md`（P1 状态标注「已设计/落地中」）

- [ ] `go test ./internal/jobs ./internal/control ./internal/agent ./internal/tool/builtin -count=1`
- [ ] `go test -race ./internal/jobs ./internal/control -count=1`
- [ ] 更新 QWEN 对比文档 P1 行状态 + 本 plan 作为证据链入口
- 验证：全绿；无 `-race` 报错

---

## 五、对抗自检（devil's advocate）

1. **锁序死锁**：recordCompletion 内先 `j.mu` 后 `m.mu`，与既有任何 `m.mu→j.mu` 反向嵌套是否冲突？——grep 证实现有 `m.mu` 内从不锁 `j.mu`（findJobLocked 只返回指针），唯一 `j.mu` 内调 `writeJobMetaLocked`（内部若再取 `m.mu` 需核查）→ **薄弱环节 A**：实现时 `go test -race` 验证，若 `writeJobMetaLocked` 反向嵌套则把快照挪出 `j.mu` 临界区。
2. **正文膨胀**：每 turn 最多 16KB 注入；但连续作业持续完成时每轮都有新注入 → 每轮注入的都是「本轮新完成」项，总量受 16KB 封顶 → 可接受；降级项靠 artifact 引用 + bash_output 兜底。
3. **XML 转义损害可读性**：正文中 `<` 变 `&lt;`，模型看到的 shell 输出略失真 → 权衡接受（shell 输出含 XML 标签罕见；信封完整性优先）；若社区反馈差，后续可改 CDATA（需处理 `]]>` 边界，另立任务）。
4. **stalled 混入 result 信封**：status="stalled" 非终态、无正文快照 → 模型可能误当完成 → 块内文案沿用现有「may be stalled — inspect with wait/bash_output」措辞，语义可辨。
5. **drain 文本变更破坏既有断言**：L321-327 断言 `Contains("Done")` → status 属性用 `st.String()` 保证兼容；其余断言均为 `Contains(id)`/非空 → 兼容；执行时逐个核对（**薄弱环节 B**：若存在未 grep 到的精确文本断言，须更新测试而非改产品语义）。
6. **尾部剥离顺序**：bg-result 在 recall 之后，若 `stripTrailingBackgroundJobResult` 未先于 `stripTrailingMemoryRecall` 执行，memory-recall 泄漏进 preview（#3653 复犯）→ 子任务 5 明确顺序 + 专项测试。
7. **重复注入**：同一 user 消息 compose 被调两次（重试/恢复路径）→ 第一次 drain 后队列空，第二次返回 ""，不重复 → 安全；`interceptInputReceive` 拦截/重试在 compose 之后，不影响。
8. **usage 虚报风险**：P1 明确 usage 为**预留空字段**，绝不填充估算值 → 不构成虚假完成；真实 usage 留待子代理会话 token 汇总（P2/P4）。

---

## 六、缓存/纪律检查点（Reasonix 领域）

### 发送侧字节变化
- **前缀（system/tools/历史消息）**：零变化。`<background-job-result>` 只 append 到**新 user 消息**尾部，属于该消息本体；已缓存前缀不包含它、不被它改写。
- **append-only 模式**：与 `<memory-update>`/`<delivery-runtime>`/`<memory-recall>` 同模式（input.go L174-176 注释明确的「ride the turn, never the cached system prefix」哲学）。
- **前缀稳定性**：该 user turn 一旦发送即成为历史，其后恒定；不随条件（goal/plan/recall 开关组合）漂移——位置固定性由子任务 4 测试锁定。
- **不插入历史**：不对 canonical transcript 已存在消息做 insert/rewrite；cache-aware projection（docs/research/cache-aware-compaction-design.md）的 `CoveredPrefixHash` 基于 canonical ModelMessages，append 新 turn 与普通用户输入无异，不触发失效。

### 纪律检查点
- 三职能分离：本文件为规划产出；实现→执行小队，审查→纪律团（team-org.md §二）。
- fable5 全流程：任务分解（§四）/拓扑扫描（§一）/多路径推演（§二）/对抗自检（§五）已覆盖 Step 1-3；Step 4-9 由执行与纪律阶段完成。
- 证据链：子任务 1-6 每个验证点命令即证据；禁止口头「完成」。

---

## 遗留问题（unresolved）

1. `docs/team-plan.md` 缺失：team-org.md 引用但仓库无此文件；P1 定义以任务描述 + QWEN 对比文档 §三.P1 + 代码现状为准。若存在未同步到工作区的权威 P1 定义，需补充对齐。
2. `DrainCompletedNote()` 的空 session legacy 行为保留（仅测试调用），不改语义。
3. usage 字段真实值来源未定（依赖子代理会话 token 汇总，另立任务）。
4. 降级摘要块的确切文案措辞由执行/纪律阶段定稿（保「可 wait/bash_output 取全文」语义即可）。
