# 计划：P4 前台→后台动态降级——运行中 task 后台化并平滑续跑（plan-1.md）

> 生成：2026-08-10 · 分支：team · 规划：Planner（仅计划，交由执行小队实施 + 纪律团审查）
> 关联：CCB 独有亮点参照——sync agent 运行中后台化 + signal-race（`CLAUDE_AUTO_BACKGROUND_TASKS` / `getAutoBackgroundMs`）；`docs/team/20260810-p1-autodeliver/plan.md`（P1 信封/bounded/锁序模式）、`docs/team/20260810-p3-steer/plan.md`（P3 消息通道/jobCtxKey 作用域模式）
> 目标：社区第二大痛点——前台 task 同步阻塞，长任务占用父 turn，用户无法「降级」为后台继续。P4 交付三层能力：**L1 打断后手动后台重启**（Esc 打断 → `/backgroundize <sa_ref>` 同现场续跑）、**L2 运行中无缝切换**（signal 通道 → 前台 task 优雅停止 → 同 prompt 后台 job 跑完，父 turn 继续）、**L3 自动后台化**（默认 120s 阈值，env 控制）。转后台后结果走 P1 信封、可 P3 `/task-message` 指挥；续跑子代理会话前缀逐字节稳定（缓存红线）。

---

## 一、拓扑扫描

### 1.1 现状锚点（grep/read 已验证）

| 位置 | 现状 | 与 P4 的关系 |
|------|------|-------------|
| `internal/agent/task.go:647-673` `TaskTool.Execute` | 解析 `run_in_background` 等参数 → `buildTaskSpec` → `RunProfileSpec`；**后台化只在此刻一次性指定** | P4 的入口前置：运行中不再有切换能力，需在 `RunProfileSpec` 前台分支补机制 |
| `internal/agent/task.go:758-992` `RunProfileSpec` | 前台分支（973-991）：`acquireSlot` 同步阻塞 → `runSession(ctx, trk.wrap(), false)` → 子代理跑完才返回；后台分支（877-971）：`jm.StartForSession` job goroutine 内 `acquireSlot` → `runSession(jobCtx, ...)`，`jobCtx` 带 `jobCtxKey{}`（jobs.go:484） | **P4 主改造点**：前台分支需要「独立 runCtx + 后台化信号 + 优雅停止 + 转 job」；后台分支是转后台后的复用目标 |
| `internal/agent/task.go:863-875` `runSession` 闭包 | 前台/后台共用：`runSubSession`（1610）/`runReadOnlySubSession`（1623）→ `RunSubAgentWithSession`（1792）→ `sub := New(...)` → `sub.Run(ctx, prompt)`（1832） | 子代理 ctx = TaskTool 传入的 ctx → runToolLoop 检查点可读信号 |
| `internal/agent/task.go:1009-1055` `prepareTranscriptRunWithPrompt` | `PrepareFresh`/`PrepareContinue`/`PrepareLegacyForkFrom` 生成 `*SubagentRun`（含 `Ref`/`Session`）；`continue_from` 已是一等公民 | **续跑现场载体**：同 `Ref` + `PrepareContinue` 重载同一 transcript |
| `internal/agent/run_loop.go:275-293` `runToolLoop` | 每轮顶部：`consumeSteer`（282）→ P3 `jobs.DrainPendingMessages`（289，jobCtxKey 作用域）→ schemas/采样 | **P4 检查点位置**：P3 注入块之后、schemas 之前加「后台化信号检查」，作用域用 ctx 标记隔离（父/后台/planner no-op） |
| `internal/agent/subagent_store.go:30-32, 753-781` | `SubagentStatus` 枚举含 `SubagentInterrupted`；`validateMeta` 对 `Running`/`Failed`/`Interrupted` **全部拒绝续跑**（"was interrupted … cannot be continued"） | **P4 状态缺口**：需要新的「可续跑中断」状态 `SubagentBackgroundized`，`validateMeta` 放行 |
| `internal/agent/subagent_store.go:660-712` | `MarkRunning`/`SaveCompleted`/`SaveFailed`；注释「transcripts are written only on completion」；`SaveFailed` 会写盘 session（705） | **现场保存落点**：新增 `SaveBackgroundized(run, snap)`（写盘会话 + 状态 + 后台化快照） |
| `internal/jobs/jobs.go:452-584` `StartForSession` | job goroutine 内 `recordCompletion`（P1 信封）+ `jobCtxKey` 注入（484，P3 消费） | 转后台 = 复用 `StartForSession`，P1/P3 **自动生效**，零改动 |
| `internal/control/controller.go:871-938` `spawnGuardedTurn`/`finishGuardedTurn` | `c.cancel`（turn ctx cancel）为唯一运行中入口；`Cancel()`（1979）取消整 turn；turn 结束后 TurnDone `Cancelled=true` | L1 触发面：Esc 打断 → 前台 task 需保存为 Backgroundized；L2 需要平行于 `Cancel()` 的 `Backgroundize()` 运行中入口 |
| `internal/control/controller.go:1399-1453` | slash 分发 switch `fields[0]`；`/task-message`（1440-1452）模式可仿 | **`/backgroundize <sa_ref>` 注册点**（L1 命令） |
| `internal/control/controller.go:1440-1452, 5508` | P3 已交付 `/task-message` + `SendTaskMessage`（SendMessageForSession） | 转后台后指挥通道已通，P4 零改动 |
| `internal/event/event.go:25-118` | `Kind` 枚举 wire-stable，新 Kind 必须插在 `KindCount` 前 | 后台化需要通知/进度事件时在此扩展（本期可仅用 Notice + 既有 Steer，见 3.6） |
| `internal/control/input.go:191-195` | P1 `<background-job-result>` 信封注入父会话下轮 | 转后台 job 完成 → 信封自动投递，P4 不触碰 |
| `internal/agent/run_loop.go:282-292` P3 注入 | `jobs.DrainPendingMessages(ctx)` 仅 `jobCtxKey` 命中 | P4 信号检查必须与之**正交**：信号在「前台 task 子代理」ctx（有标记、无 jobCtxKey），P3 通道在「后台 job 子代理」ctx（有 jobCtxKey、无标记） |

### 1.2 级联风险清单

1. **R1 写槽/并发死锁（最高危）**：前台 task 持有 scheduler 写槽（`acquireSlot`，task.go:974），转后台 job 又要 `acquireSlot`（后台分支 944）。若前台 slot 未释放 → 后台排队/失败/自身死锁。**对策**：转后台前**显式释放**前台 slot 并幂等化 `releaseSlot`（once 语义），job 内走既有排队逻辑。
2. **R2 ctx 取消语义回归**：独立 runCtx 若切断「父 cancel → 子代理取消」传导，用户 Esc 将无法停掉前台 task。**对策**：`runCtx = context.WithCancel(ctx)`（继承传播）+ runToolLoop 检查点区分「父取消」vs「后台化信号」；T4/T6 测试断言 Esc 仍能取消。
3. **R3 状态语义污染**：`SubagentInterrupted` 现有语义是「崩溃/关机脏状态」，测试已断言不可续跑（subagent_store_test.go）。若复用会导致续跑失败或语义混乱。**对策**：新增独立 `SubagentBackgroundized`，`validateMeta` 只放行新状态，Failed/Interrupted 拒绝语义**原样保留**。
4. **R4 检查点作用域误伤**：`runToolLoop` 是父/子共用。若检查点对父 agent / 后台 job 子代理 / planner 产生副作用 → 破坏父前缀或错误后台化。**对策**：ctx 标记 `backgroundizeCtxKey{}` 仅由 TaskTool 前台分支注入；无标记恒 no-op（与 P3 `jobCtxKey` 同构）；T4 测试断言父/后台/planner 三处零副作用。
5. **R5 timer 与终态竞态**：自动后台化 timer 到点瞬间子代理刚好完成/失败 → 双重终态或转后台一个已完成的 run。**对策**：信号一次性（`trigger` 幂等关闭 channel）；转后台前检查 `ctx.Err()` 与 run 状态；`SaveBackgroundized` 覆盖已完成 meta 时以 job 终态为准。
6. **R6 续跑前缀漂移**：`PrepareContinue` 装载的 session 若与中断时字节不一致（如打包 prompt 漂移、`subagentStartContext` 重复注入）→ 前缀失效甚至污染。**对策**：snapshot 保存**原始 prompt**（task.go:664 的 `p.Prompt`，未包装），重放经既有确定性包装（`withWorkspaceContext`/`completeSubtaskContract` 是固定拼接）；`isFreshSubagentSession` 保证续跑不重复 prepend `subagentStartContext`；`validateMeta` 强校验 SystemPromptHash/ToolSchemaHash/Model/Effort/ToolScope（subagent_store.go:763-778）。
7. **R7 转后台后父 agent 上下文**：后台化瞬间 task 工具返回「Started background task…」文本（而非子代理答案）→ 父模型可能困惑或重复 wait。**对策**：返回文本复用既有后台启动措辞（task.go:968 同款）+ job id + 指引 `/task-message`；语义与「启动即后台」一致。
8. **R8 jobs manager 缺失**：headless/无 `jobs.FromContext` 上下文下转后台不可用。**对策**：`backgroundizeToJob` 前置 `jm, ok := jobs.FromContext(ctx)` 检查，缺则保持前台并返回明确错误（不静默）。
9. **R9 锁序**：`SaveBackgroundized` 走 subagent_store 的 ref 锁（既有），不与 jobs 的 `m.mu`/`j.mu` 交叉；`Trigger` 只操作信号结构自身锁 → 零死锁面。`-race` 验证。

---

## 二、多路径推演

### 方案 A（推荐）：turn-ctx 信号槽 + TaskTool 前台转后台 + 独立 runCtx + runToolLoop 检查点 + `SubagentBackgroundized` 状态

- **L1**（打断后重启）：前台 task 因父 turn 取消而中断时，`TaskTool` 前台分支识别 `ctx.Err()!=nil` → `SaveBackgroundized(run, snap)`（现场落盘 + 快照）→ 用户 `/backgroundize <sa_ref>` → controller 从快照恢复 spec → 后台 job 续跑（同 prompt、`continue_from=run.Ref`）。
- **L2**（运行中无缝）：controller 构造 turn ctx 时预置 `BackgroundizeSlot`（存 `c.bgSlot`，仿 `c.cancel`）→ `Controller.Backgroundize()` 运行中触发 slot → TaskTool 前台分支的 `backgroundizeSignal` 置位 → 子代理 `runToolLoop` 检查点（ctx 标记命中）优雅停止（当前轮已 commit，现场完整）→ 返回 `errBackgroundizeRequested` → TaskTool 前台分支释放前台 slot → 启动后台 job（同 snapshot 路径）→ 返回 job id 文本，**父 turn 继续**。
- **L3**（自动）：TaskTool 前台分支 `time.AfterFunc(autoBackgroundMs, sig.trigger)`（默认 120s，env 控制）→ 与 L2 同一信号汇合。
- **复杂度**：中-高（agent 包 3 处 + control 2 处 + subagent_store 1 处）；**性能**：检查点 O(1)、timer 无轮询、无额外请求；**可维护性**：复用 P3 的 ctx 作用域模式、P1 的既有 job 路径、既有 `continue_from`；**风险**：中（R1-R9 均有明确对策）。
- **关键取舍**：L2/L3 共享同一「信号 → 优雅停止 → 转 job」路径，L1 是它的兜底（turn 已死时由 controller 代执行）。三者共用 `BackgroundizeSnapshot`，避免 spec 重建漂移。

### 方案 B：仅「打断后 `/backgroundize` 重启」（无运行中切换）

- 改动最小（store 状态 + controller 命令），但**缺 L2 运行中无缝切换与 L3 自动后台化**——不满足需求 1/3 的完整语义。且 L3 自动后台化最终仍需要 TaskTool 内部 timer + 信号（即 A 的机制），B 无法避免走向 A。
- **否决**：需求明确要求「运行中切换 + 120s 自动后台化」，B 只交付一半且不收敛。

### 方案 C：`/backgroundize` 翻译为用户消息，让父 agent 自己调 task（`run_in_background=true, continue_from=ref`）

- 零内核改动（纯 UI/命令翻译），但：① 依赖模型正确重建参数（不可靠）；② 每次多一轮模型请求（一次 cache miss）；③ 权限/写槽由模型决定（安全面回归）。与「确定性机制 + 缓存红线」哲学冲突。
- **否决**：机制必须由宿主确定性执行（CCB 同款——宿主重建，不经模型）。

### 方案 D：转后台 = 单纯「继续跑完不打断」（无优雅停止）

- 否决：无优雅停止就没有「动态降级」——父 turn 仍被阻塞，语义是「等它自己跑完」，不是后台化。

### 选型结论

**选 A**。唯一同时满足 ① 运行中无缝切换（L2）② 自动后台化（L3）③ 确定性宿主执行（不依赖模型）④ 续跑前缀稳定（同 Ref + 快照）⑤ 与 P1/P3 零冲突复用 ⑥ 改动面收敛（L1 作为 A 的子集自然包含）。B 缺 L2/L3，C 破坏确定性，D 无语义。

---

## 三、设计规格（A 方案细化）

### 3.1 状态与持久化（subagent_store.go）

```go
// SubagentStatus 新增
SubagentBackgroundized SubagentStatus = "backgroundized"
// 语义：前台 task 因用户打断/后台化信号而优雅停止，现场（transcript + 快照）已持久化，
// 可经 /backgroundize 或宿主后台重启续跑。区别于 SubagentInterrupted（崩溃脏状态，不可续跑）。

// BackgroundizeSnapshot 存于 SubagentMeta（JSON 字段，向后兼容，缺失=旧数据不支持后台化）
type BackgroundizeSnapshot struct {
    Prompt, Description, Profile string
    WritePaths, Tools            []string
    MaxSteps                     int
    Model, Effort                string
    ContinueFrom, Label          string
}

func (s *SubagentStore) SaveBackgroundized(run *SubagentRun, snap *BackgroundizeSnapshot) error
// = ensureBranchCreatedAt + run.Session.Save(sessionPath) + meta.Status=Backgroundized +
//   meta.Backgroundize=snap + saveMeta。与 SaveFailed 同构（session 写盘在置态前）。
```

- `validateMeta`（subagent_store.go:753）：新增 `case meta.Status == SubagentBackgroundized:` **放行**（跳过现有 Failed/Interrupted 的拒绝）；`Running/Failed/Interrupted` 拒绝分支**一字不改**。
- `PrepareContinue`（398）：`validateMeta` 放行后即可装载；`validateContinueOwner`（790）保证只能被原父会话（或 lineage）重启。
- `ReadFinalAnswer`（146）：`!= SubagentCompleted` 拒绝读结果 → Backgroundized 天然不可读，无需改动。

### 3.2 信号通道（agent 包新文件 `backgroundize.go`，与 P3 的 `jobCtxKey` 同构）

```go
// backgroundizeSignal 是单次信号：trigger 幂等（close(ch)），fired 无锁读标志。
type backgroundizeSignal struct {
    mu    sync.Mutex
    fired bool
    ch    chan struct{}
}

// BackgroundizeSlot 挂在 turn ctx 上：TaskTool 前台分支 Register 自己的信号，
// controller.Backgroundize() 经 Trigger 触发。仿 c.cancel 的「运行中唯一入口」语义。
type BackgroundizeSlot struct {
    mu  sync.Mutex
    sig *backgroundizeSignal
}

func WithBackgroundizeSlot(ctx context.Context, slot *BackgroundizeSlot) context.Context
func BackgroundizeSlotFrom(ctx context.Context) (*BackgroundizeSlot, bool)
func (s *BackgroundizeSlot) Register(sig *backgroundizeSignal) // nil 清空
func (s *BackgroundizeSlot) Trigger() bool                     // 有信号则 trigger，返回是否命中

type backgroundizeCtxKey struct{}
func withBackgroundizeSignal(ctx context.Context, sig *backgroundizeSignal) context.Context // 仅 TaskTool 前台分支注入
func backgroundizeSignalFrom(ctx context.Context) (*backgroundizeSignal, bool)

var errBackgroundizeRequested = errors.New("agent: foreground task backgroundized")
```

- **作用域矩阵**（与 P3 注入点正交）：
  | ctx 来源 | `backgroundizeCtxKey` | `jobCtxKey` | 检查点行为 |
  |---|---|---|---|
  | 父 agent（controller turn） | 无 | 无 | no-op ✓ |
  | 前台 task 子代理（TaskTool 前台分支注入） | **有** | 无 | **检查信号，命中则优雅停止** |
  | 后台 job 子代理 | 无 | 有 | no-op ✓ |
  | planner/guardian/review/fleet | 无 | 无 | no-op ✓ |

### 3.3 runToolLoop 检查点（run_loop.go）

位置：`runToolLoop` 每轮顶部，P3 `DrainPendingMessages` 块（289-292）之后、`a.tools.Schemas()`（293）之前：

```go
// P4: foreground-task backgroundize signal. Only contexts stamped by the TaskTool
// foreground branch carry backgroundizeCtxKey (parent/background/planner no-op).
if sig, ok := backgroundizeSignalFrom(ctx); ok && sig.fired() {
    return errBackgroundizeRequested
}
```

- **优雅停止边界**：检查点位于工具轮次顶部 → 当前轮（assistant 消息 + 工具结果）已全部 commit 到 `run.Session` 或尚未开始 → 现场 = 全部已 commit 轮次，逐字节完整。正在采样中的轮次不可打断（采样原子），至多延迟一个采样周期（与 CCB signal-race 语义一致，可接受）。
- 返回 `errBackgroundizeRequested` 后：`streamWithSamplingRecovery` 不进入下一轮 → `sub.Run` 返回 → `RunSubAgentWithSession`（task.go:1832）经 `%w` 包装透传（`errors.Is` 可穿透）→ `runSubSession` → `runSession` 闭包 → TaskTool 前台分支捕获。

### 3.4 TaskTool 前台分支重构（task.go:973-991）

```go
// Foreground: acquire a slot, then run synchronously with backgroundize support.
releaseSlot, err := t.acquireSlot(ctx, acquireReq)
if err != nil { run.Release(); return "", err }
slotReleased := false
releaseSlotOnce := func() { if !slotReleased { slotReleased = true; releaseSlot() } }
defer releaseSlotOnce()

runCtx, cancelRun := context.WithCancel(ctx) // 独立子 ctx：可单独停子代理，父 cancel 仍传导（R2）
defer cancelRun()

bgSig := newBackgroundizeSignal()
if slot, ok := BackgroundizeSlotFrom(ctx); ok { slot.Register(bgSig); defer slot.Register(nil) }
if after := t.autoBackgroundMs(); after > 0 {
    timer := time.AfterFunc(after, bgSig.trigger) // L3：默认 120s
    defer timer.Stop()
}

answer, err := runSession(withBackgroundizeSignal(runCtx, bgSig), trk.wrap(), false)
switch {
case errors.Is(err, errBackgroundizeRequested): // L2/L3：运行中后台化
    cancelRun()
    releaseSlotOnce() // 显式释放前台写槽，避免 R1
    return t.backgroundizeToJob(ctx, spec, run)
case ctx.Err() != nil: // L1 兜底：父 turn 被打断（用户 Esc）→ 现场落盘，不丢
    snap := snapshotFromSpec(spec, t) // 尽力构造快照（spec 在手）
    if serr := t.transcripts.SaveBackgroundized(run, snap); serr == nil {
        return "", fmt.Errorf("foreground task interrupted; rerun it in the background with /backgroundize %s", run.Ref)
    }
    return "", errors.Join(err, t.transcripts.SaveFailed(run))
case err != nil:
    return "", errors.Join(err, t.transcripts.SaveFailed(run))
}
// 正常完成路径不变……
```

**`backgroundizeToJob(ctx, spec, run) (string, error)`**（共享 helper，controller 的 `/backgroundize` 也调）：

1. `jm, ok := jobs.FromContext(ctx)`；缺则返回明确错误（R8）。
2. 构造 `BackgroundizeSnapshot`（用**原始** `spec.Task.Objective` 与调用参数，见 3.1）。
3. `t.transcripts.SaveBackgroundized(run, snap)`。
4. 构造后台 spec：`bgSpec := spec`；`bgSpec.Sched.RunInBackground = true`；`bgSpec.Sched.BackgroundWriter = false`；`bgSpec.Context.ContinueFrom = run.Ref`。
5. `jm.StartForSession(SessionFromContext(ctx), "task", label, func(jobCtx context.Context, _ io.Writer) (string, error){ ... })`——**复用 RunProfileSpec 后台分支既有逻辑**（或抽取 `startBackgroundJob(ctx, spec)` helper 供前台分支与 controller 共用，执行队定；内部 job 闭包经 `prepareTranscriptRunWithPrompt` 的 `ContinueFrom=run.Ref` → `PrepareContinue` 从磁盘重载同一 transcript → `runSession(jobCtx, ...)` 续跑）。
6. 返回文本：`Foreground task backgroundized as %q (%s). It runs across turns; collect its final answer with wait, and you'll be notified when it finishes. Continue steering with /task-message %s <text>.`（job id + label）——语义与 task.go:968 后台启动措辞一致（R7）。

**`autoBackgroundMs` 配置**（L3）：TaskTool 新字段（`WithAutoBackground(after time.Duration)` 注入，boot 装配）或启动时读取：`REASONIX_AUTO_BACKGROUND_TASKS` 非空启用 + `REASONIX_AUTO_BACKGROUND_MS`（默认 `120000`）。默认策略：**启用且 120s**（对齐 CCB `getAutoBackgroundMs`；若产品决策要默认关闭，执行队仅改默认值，机制不变）。

### 3.5 controller 层（L1 命令 + L2 运行中入口）

```go
// controller.go 新增（仿 /task-message 1440-1452 与 Cancel 1979）
func (c *Controller) Backgroundize(ref string) error {
    // 1. 校验：ref 属于当前父会话 lineage、meta.Status==SubagentBackgroundized
    // 2. 从 meta.Backgroundize 快照重建 spec（经 buildTaskSpec 重新解析权限/工具——确定性宿主执行）
    // 3. 调 agent.BackgroundizeRun(ref, spec) → backgroundizeToJob 共享路径 → 返回 job id
}
```

- slash `/backgroundize <sa_ref>`：controller.go:1399 switch 增加 case（turn 结束后可发，仿 `/task-message`）。
- 运行中 `Controller.Backgroundize()`（无参，触发当前前台 task）：读 `c.bgSlot`（turn 启动时构造 `&BackgroundizeSlot{}` 注入 turn ctx + 存字段）→ `Trigger()`。**前端接线为增强项**：chat TUI 的 Esc 双动作 / desktop 按钮 / HTTP 端点，本事务交付 controller API + 测试，UI 接线列遗留（与 P3 的 `send_message` 工具同类取舍，避免缓存红线事务混入 UI 大改）。
- 事件：后台化成功发 `event.Notice`（LevelInfo，含 job id）；`event.Steer` 不误用（那是消息投递语义）；新事件 Kind 本期不新增（wire-stable 成本高，Notice 已够）。

### 3.6 P1/P3 协同（零改动验证）

| 能力 | 机制 | P4 动作 |
|------|------|---------|
| 结果自动投递（P1） | 后台 job 完成 → `recordCompletion` → `<background-job-result>` 信封进父会话下轮 | 转后台走 `StartForSession`，`Kind=="task"` 既有路径，**零改动** |
| 运行中指挥（P3） | `/task-message` → `SendMessageForSession`（status==Running）→ 子代理轮次边界注入 | job 创建即 Running（jobs.go:472），**零改动**；转后台返回文本已提示 `/task-message` |
| 锁序 | `m.mu`/`j.mu` 不嵌套 | P4 只碰 subagent_store ref 锁 + 信号自身锁，与 jobs 锁无交集 |

### 3.7 缓存红线（需求 5）

1. **续跑前缀稳定**：`PrepareContinue(run.Ref, spec)` 从磁盘加载**同一 transcript** → 子代理请求前缀 = 原历史逐字节一致；`validateMeta` 强校验 `SystemPromptHash`/`ToolSchemaHash`/`Model`/`Effort`/`ToolScope`（subagent_store.go:763-778）→ persona/工具/模型漂移在装载时即拒绝。
2. **重启 prompt 逐字节一致**：snapshot 存**原始 prompt**；续跑经确定性包装（`withWorkspaceContext`+`completeSubtaskContract`，固定字符串拼接）→ 重启 user 消息与初次一致。`subagentStartContext` 由 `isFreshSubagentSession`（task.go:2039，仅 system 一条）守卫 → 续跑**不重复 prepend** → 前缀无重复块。
3. **一次 cache miss 不可避免**：续跑首轮新增重启 prompt user 消息（模型需看到「继续」指令）——与 P3 steer/既有注入注释（run_loop.go:280-281）同款，此后前缀稳定。
4. **父会话**：后台化瞬间 task 工具结果尾部增量（job id 文本）；后台 job 完成 → P1 信封进下轮注入位置（input.go:191-195 既有）→ 历史前缀零变化。
5. **发送侧审计**：转后台不修改父系统提示/工具 schema/历史；父请求字节变化仅限「task 工具结果内容」与「下轮信封块」两处尾部增量。

---

## 四、任务分解（子任务 + 验证点）

- [ ] **T1 subagent_store 状态扩展**
      `SubagentBackgroundized` 状态 + `BackgroundizeSnapshot` 结构 + `SaveBackgroundized`；`validateMeta` 放行新状态、**原样保留** Failed/Interrupted 拒绝。
      → 验证：`go test ./internal/agent/ -run 'SubagentStore|ContinueFrom|Backgroundiz' -race`；新单测 `TestBackgroundizedContinueAllowed`（Backgroundized 可 PrepareContinue 装载）、`TestBackgroundizedSnapshotRoundTrip`（save→load 快照逐字段一致）、`TestFailedStillRejectedAfterBackgroundize`（Failed/Interrupted 拒绝语义回归）。
- [ ] **T2 信号通道 + runToolLoop 检查点**
      新文件 `backgroundize.go`（signal/slot/ctx key/err）；`run_loop.go` runToolLoop P3 注入块后加检查点；作用域矩阵 no-op 全覆盖。
      → 验证：`go test ./internal/agent/ -run 'Backgroundize|Steer' -race`；`TestParentAgentIgnoresBackgroundizeSignal`（父 ctx 无标记 → session 零变化）、`TestForegroundSubagentStopsOnSignal`（带标记 + fired → runToolLoop 返回 errBackgroundizeRequested，已 commit 轮次完整）、`TestBackgroundJobIgnoresSignal`（jobCtxKey ctx → no-op）。
- [ ] **T3 TaskTool 前台分支重构 + backgroundizeToJob**
      独立 runCtx + 幂等 releaseSlotOnce + 信号注册 + timer + `errBackgroundizeRequested`/`ctx.Err()` 双分支 + `backgroundizeToJob`（快照保存 + `ContinueFrom=run.Ref` 后台 job）；`autoBackgroundMs` 注入/env 读取。
      → 验证：`go test ./internal/agent/ -run 'Task|Backgroundiz' -race`；`TestForegroundTaskBackgroundizesMidRun`（timer 或直接 trigger → task 工具返回 job id 文本、父 turn 可继续、job 完成走 P1 信封）、`TestForegroundTaskCancelledSavesBackgroundized`（ctx cancel → meta=Backgroundized、session 落盘、ref 可续跑）、`TestForegroundSlotReleasedBeforeJobStart`（scheduler mock 断言前台释放先于后台 acquire，-race）。
- [ ] **T4 controller 入口 + /backgroundize 命令**
      `Controller.Backgroundize(ref)`（校验 → 快照恢复 → 后台 job）+ 运行中 `Controller.Backgroundize()`（bgSlot → Trigger）+ slash case + turn ctx 预置 slot（`c.bgSlot`）。
      → 验证：`go test ./internal/control/ -run 'Backgroundize' -race`；`TestBackgroundizeUnknownRef`/`TestBackgroundizeWrongState`（Running/Failed 拒绝）、`TestBackgroundizeResumesSameRef`（Backgroundized → 后台 job 用同 Ref 续跑）、`TestBackgroundizeRunningTurnTrigger`（running 中 Trigger 命中前台 task）。
- [ ] **T5 端到端 + 缓存红线回归**
      起前台 task → 触发后台化（L2 信号 + L3 timer 两路）→ 断言：job 启动、父 turn 继续、`/task-message` 可指挥、完成后 P1 信封注入、`/backgroundize`（L1）打断后重启同现场；**前缀稳定断言**：续跑首轮请求前缀与中断前历史逐字节一致（含 `subagentStartContext` 不重复、原始 prompt 一致）。
      → 验证：定向 e2e（agent+control+jobs 联动）；`go test ./internal/agent/ ./internal/control/ ./internal/jobs/ -race` 相关包全绿；`go build ./...`；`input.go` 零 diff 确认（P1 注入位置不动）。
- [ ] **T6 静态与纪律门**
      `gofmt -w .`、`go vet ./...`、`make lint`（repolint 基线不拓宽）；REASONIX.md 无需改（P4 不改系统提示/前缀）。
      → 验证：三命令零新增告警。
- [ ] **T7 文档交付**
      本 plan-1.md；执行后由执行小队产 execution.md（证据链）、纪律团产 review.md。
      → 验证：文件存在、五节格式完整。

---

## 五、对抗自检（devil's advocate）

1. **攻击：独立 runCtx 会不会让用户 Esc 再也停不掉前台 task，导致「后台化」变成「失控」？**
   `runCtx = context.WithCancel(ctx)`——父 turn cancel 沿 ctx 传导，子代理照常停（R2）。检查点只在「信号 fired 且非 ctx 取消」时返回 errBackgroundizeRequested；两者可区分。T3 `TestForegroundTaskCancelledSavesBackgroundized` 显式断言 Esc 仍取消。**这是本计划最关键薄弱环节**——执行时必须验证「父 cancel → runCtx cancel → 子代理停」链路不被信号分支短路。
2. **攻击：用户 Esc 的本意是「停掉」，现在却把任务悄悄存成 Backgroundized，是否越权？**
   「保存现场 ≠ 启动后台」。Esc 取消只落盘不续跑，后台化必须显式 `/backgroundize` 或运行中 `Backgroundize()`。语义与 CCB「Esc 后显示 Rerun in background 选项」等价——本计划把「选项」落成命令，UI 按钮列遗留。无静默后台化（自动后台化是 L3 独立特性，默认启用但阈值可关）。
3. **攻击：转后台后 job 闭包 `PrepareContinue` 从磁盘重载，若 `SaveBackgroundized` 尚未落盘完成（写盘竞态）就启动 job，会不会读到旧 transcript？**
   `backgroundizeToJob` 顺序保证：`SaveBackgroundized` 返回 nil（session + meta 已 fsync 语义）**之后**才 `StartForSession`。job 闭包在 goroutine 内启动，天然晚于顺序点；且 ref 锁（subagent_store.lock）串行化 Save/PrepareContinue。`-race` + T5 e2e 覆盖。
4. **攻击：快照重建 spec 时 `WritePaths` 已解析为 claim，重建会重新 resolve——权限会不会比初次更宽？**
   snapshot 只存**原始调用参数**（prompt/profile/write_paths/tools/max_steps/model/effort/continue_from），重建走 `buildTaskSpec`（task.go:677）重新解析/归一化/校验 → 权限集与初次同源（同 profile 上限 + 同参数下限），不会更宽。写路径 claim 在 job 内重新 acquire。
5. **攻击：`runToolLoop` 检查点与 P3 注入点的先后顺序，会不会让「后台化信号」和「steer 消息」互相吞掉？**
   P3 块（DrainPendingMessages → 注入）在前，P4 检查在后：同一轮若既有 steer 又有信号，steer 先注入（提交）、随后返回 errBackgroundizeRequested（本轮结束，注入消息已在 session）→ 续跑时该 steer 在历史中，无丢失。两通道正交（不同 ctx 键），无互斥。
6. **攻击：自动后台化 timer 触发时子代理正卡在长 bash——优雅停止要等 bash 返回，会不会「自动后台化」实际不生效？**
   timer 触发 → 信号置位 → 子代理当前工具（bash）跑完后回到检查点才停。长 bash 场景延迟 = bash 剩余时长（CCB 同款：工具执行中不可硬切）。T5 用短工具 e2e 验证机制，长工具延迟语义在文档标注为已知边界（不阻塞）。
7. **攻击：L1 的 `/backgroundize` 需要快照，而被打断瞬间 `spec` 一定在 TaskTool 前台分支手上吗？**
   是——L1 保存发生在 `RunProfileSpec` 前台分支的 err 处理里，`spec` 是该函数局部变量（task.go:758 参数），`snapshotFromSpec` 直接可构造。唯一例外：崩溃/断电（进程死了没机会保存）→ 状态停驻 Running 或脏恢复路径，不属于 P4 承诺（遗留标注）。
8. **攻击：转后台的 job 与 P1/P3 的 bounded/背压会不会被「同一 ref 复用」弄坏（如 pendingMessages 残留）？**
   转后台是新 `StartForSession`（新 job 句柄、空 pendingMessages），与原前台 run 无共享字段；`recordCompletion` 只认 job 终态。同一 Ref 只影响 transcript 存储，与 jobs 队列解耦。
9. **攻击：新增 `SubagentBackgroundized` 会不会破坏 `CleanupStaleRunning` / `subagent_registry` 等既有状态机？**
   `CleanupStaleRunning`（subagent_store.go:248）只处理 Running 脏态；Backgroundized 是合法终态（不参与清理）。既有 `SubagentInterrupted` 语义与测试不动。T1 回归 `TestSubagentStoreCleanupStaleRunningMarksInterrupted` 等全绿确认。
10. **攻击：自动后台化默认开启（120s）会不会让用户意外——长任务突然变成后台，父 turn 冒出 job id？**
    这是 L3 的产品语义（CCB `CLAUDE_AUTO_BACKGROUND_TASKS` 默认即启用）。本计划保留默认启用但提供 `REASONIX_AUTO_BACKGROUND_TASKS=0` 关闭；若产品决策默认关闭，仅改默认值（T3 内一行）。后台化返回文本明确告知 job id + wait 指引，父 agent 有信息继续。

---

## 六、缓存/纪律检查点（Reasonix 领域）

- **发送侧字节变化**：
  - 父会话：仅两处尾部增量——① 后台化瞬间 task 工具结果文本（job id 启动措辞，替代子代理答案）；② 后台 job 完成后的 P1 `<background-job-result>` 信封（既有 input.go:191-195 注入位置，零改动）。**历史前缀零变化** ✓
  - 子代理会话：续跑 = `PrepareContinue` 装载**同一 transcript**（逐字节一致）+ 新增一条重启 prompt user 消息 → 仅此一处 cache miss，此后前缀稳定；`subagentStartContext` 不重复 prepend ✓
- **前缀稳定机制**：`validateMeta` 强校验 SystemPromptHash/ToolSchemaHash/Model/Effort/ToolScope（subagent_store.go:763-778）→ persona/工具/模型漂移在装载时拒绝，续跑前缀不可能漂移 ✓
- **不插入历史**：后台化不修改子代理已 commit 历史（检查点在轮次顶部，现场 = 完整已 commit 轮次）；父历史仅尾部增量 ✓
- **防虚假完成**：`backgroundizeToJob` 只在 `SaveBackgroundized` 成功后才启动 job；转后台返回 job id 文本（可 wait/可 /task-message）；L1 保存失败时如实走 `SaveFailed` + 错误返回，绝不静默宣称已后台化 ✓
- **锁序**：信号/slot 各自独立锁，subagent_store ref 锁既有，jobs `m.mu`/`j.mu` 不触碰 → 零新增嵌套，`-race` 验证 ✓
- **作用域隔离（R4 防线）**：`backgroundizeCtxKey` 仅 TaskTool 前台分支注入；父 agent / 后台 job 子代理 / planner / guardian 检查点恒 no-op，T2 显式断言 ✓
- **⚠️ 唯一缓存注意项**：L2 运行中 `Controller.Backgroundize()` 的**前端接线**（TUI/desktop 按键/按钮）会触碰前端代码，本期只交付 controller API + 测试，UI 接线列遗留——避免在缓存红线事务混入前端大改。

---

## 七、遗留/假设

- **UI 接线**：chat TUI 的 Esc 双动作 / desktop「Rerun in background」按钮 / HTTP 端点调 `Controller.Backgroundize()`——本期交付 controller API + `/backgroundize` slash（turn 后），运行中触发入口的 UI 绑定列后续事务（缓存代价 🟡 已在第六节标注）。
- **长工具延迟**：信号在工具轮次边界生效；子代理卡长 bash 时优雅停止延迟 = bash 剩余时长（CCB 同款语义），文档标注为已知边界。
- **崩溃/断电现场**：进程死亡时无机会 `SaveBackgroundized`（状态停驻 Running/脏恢复），不属 P4 承诺；恢复语义沿用既有 `CleanupStaleRunning`。
- **事件扩展**：后台化成功用 `event.Notice`；不新增 wire-stable 事件 Kind（成本/收益比不划算）。若 UI 需要「后台化卡片」，另立事务加 Kind。
- **自动后台化默认值**：默认 120s 启用（对齐 CCB `getAutoBackgroundMs`）；若产品决策默认关闭，执行队仅改 `autoBackgroundMs` 默认值，机制不变。
- **`backgroundizeToJob` 与 RunProfileSpec 后台分支的代码复用粒度**（抽取共享 helper vs 内联）由执行队在 T3 内定，接口不变。
- **仅 `task` 工具（Kind=="task"）进入 P4 范围**：`parallel_tasks`/`fleet`/`run_skill` 不承诺（与 P3 范围限定一致）；`read_only_task` 前台也走 `RunProfileSpec`，机制天然覆盖（无写槽，仅 slot 释放语义略简化），执行队按同一实现路径覆盖。
