# P6「完成事件驱动」× P1/P3/P4 机制交互设计评审

> 角色：**规划 Planner**（独立评审视角，只读，零代码变更）
> 事务：`docs/team/20260810-p6-complete/` · 基线：P1 信封（jobs.go recordCompletion / ResultSnapshotForSession / DrainCompletedNoteForSession）、P3 steer（SendMessageForSession / DrainPendingMessages / pendingMessages 队列）、P4 backgroundize（run_loop.go backgroundizeRequested / task.go backgroundizeForeground）、P5 fork silent（task.go startJob 选择）、P6 team（teammate_store.go Assign / syncStateLocked / Tasks / pendingDependenciesLocked / flushMailbox）
> 对象：「完成事件驱动」= 文档中的 `WithJobDoneObserver` + `TeammateStore.OnJobDone`（**代码当前未实现**，仅存在于 implementation-plan.md / risk-review.md，本评审以其为待实现设计，校验其与既有机制的交互）
> 定位：独立评审，结论先行 + 代码锚点；实现与验证由执行小队承担、纪律团复核（docs/team-org.md 三职能硬性分离）

---

## 0. 结论摘要（五主题一句话）

| # | 主题 | 结论 | 风险等级 |
|---|------|------|---------|
| 1 | P1 信封与完成事件的次序/双通知 | **信封先入队（`recordCompletion` L990-995）→ 完成事件（observer）后触发（L1000-1002）→ closing Notice → `close(j.done)`**；leader 读到信封在**下一轮 turn**（input.go L191-193），与完成事件不同步、不同通道，无直接竞争。teammate job 非 silent → 信封必然送达；完成事件若遵循「零新发送」（不额外 emit 完成类 Notice）则无双通知 | 中（若违反零新发送则高） |
| 2 | silent job 完成时 observer 触发吗 | **触发**（observer 挂两处无锁点，含 suppressEnvelope 分支 L975-977，与 TaskRecorder 语义对齐）。但 teammate job **全部非 silent**（Assign L202 显式 `Silent: false`）→ 走正常分支。OnJobDone 必须按 `ts.tasks[id]` 登记过滤（G8）兜住 silent fork / 任意后台 job | 低（过滤缺失则中） |
| 3 | readOffset 消费 / completion handler 偷读 | **信封与 `ResultSnapshotForSession` 都是非消费快照**（`jobResultTextLocked` L895-910），**偷不走信封**；全仓**唯一消费点**是 `Output/OutputForSession`（L1124-1127 消费 readOffset、L1132-1135 消费 resultRead）。OnJobDone 只用 st 参数 + ts.tasks → 无偷读。**但 teammate_store 既有三处 status-only `jm.Output` 调用（L135/L250/L272）是潜在偷读源**（查 /team-status 会消费 leader 后续 bash_output 增量） | 中 |
| 4 | backgroundize 在 teammate 上下文 | **结构上不可能 handoff**：`backgroundizeRequested` 第一道 guard 即 `!foregroundTaskFromContext(ctx) → false`（run_loop.go L489-491）；teammate job 走 RunInBackground 分支（task.go L940）从不设 `WithForegroundTask`（仅 L1066 foreground 分支设），jobCtx 链（jobs.go L537/L558-559）也不携带 per-turn backgroundize 信号 | 无风险 |
| 5 | 自动 Assign 是否改变 P3 steer 语义 | **机制不变**（新 job 的 pendingMessages 空队列，flushMailbox L430-457 注入 mailbox，run_loop L390 每轮 drain 一条）；**改变的是时序与目标定位**：自动推进提前 steer 窗口、旧 jobID 失效（`SendMessageForSession` 对终态 job 拒绝 L1090-1092）、同步 Assign 阻塞收尾。MVP 不自动 Assign → P3 零影响 | 中（仅未来自动推进场景） |

**一条总纲**：完成事件驱动的安全边界 =「**内存态推进 + 零新发送 + 非消费查询**」。凡越过此边界（emit 完成 Notice / 调 Output / 同步 Assign / 用 jobCtx 调 Assign）即引入与 P1/P3/P4 的交互回归。

---

## 1. 拓扑扫描（代码锚点全表）

```
internal/jobs/jobs.go
 ├─ recordCompletion:937-1016（终态唯一出口；startInvalid:479 / run goroutine:645 两处调用）
 │    ├─ j.mu 非消费快照 + 信封预渲染 :951-961（jobResultTextLocked:895 不消费 readOffset/resultRead）
 │    ├─ suppressEnvelope 分支 :963-978（foregroundClaimPending||silentCompletion :953）→ RecordDone :975-977 → return
 │    ├─ destroying 窗口 :982-985 提前 return（不 append、不 RecordDone）
 │    ├─ m.mu 内 append m.completed :990-995（信封入队点）＋ droppedMsgs 时 m.mu 内 sink.Emit :988（既有 m.mu→syncSink 嵌套点）
 │    └─ 正常分支 RecordDone :1000-1002（完成事件 observer 挂点，m.mu 已释放）→ closing Notice :1013-1015
 ├─ run goroutine 时序 :585-657（recordCompletion :645 → status 发布 :647-655 → close(j.done) :656）
 ├─ StartForSession:485 / StartSilentForSession:503 / StartForegroundForSession:521 → startForSession:525
 ├─ SendMessageForSession:1078-1103（仅 Running task job，16 条/8KB 有界，拒绝不丢）
 ├─ Output/OutputForSession:1107-1143（readOffset :1124-1127 + resultRead :1132-1135 唯一消费点）
 ├─ DrainCompletedNoteForSession:1468-1511（maxResultsPerDrain=8 / 16KB；leader 唯一信封消费点）
 ├─ ResultSnapshotForSession:1550-1559（非消费渲染，与信封同源）
 ├─ DrainPendingMessages:2366-2380（jobCtxKey 定位，FIFO 每轮一条）
 └─ 待新增：WithJobDoneObserver / SetJobDoneObserver / notifyJobDone（implementation-plan T1，挂 :975-977 与 :1000-1002 两处无锁点）

internal/agent/task.go
 ├─ foregroundTaskKey/WithForegroundTask:35-40、foregroundTaskFromContext:43-46（仅 foreground 分支 :1066 设置）
 ├─ buildTaskSpec:703-777（fork 默认 Silent:true :714-718）
 ├─ RunProfileSpec:792（fork→RunInBackground=true :800-802；start 选择 :991-994 `Fork&&Silent → StartSilentForSession`）
 ├─ 背景分支 :940-1048（teammate 走此路；jobCtx=WithSession+jobCtxKey，无 WithManager）
 └─ backgroundizeForeground:1097+（StartForSession :1128 非 silent → 完成走正常分支信封）

internal/agent/run_loop.go
 ├─ errBackgroundizeRequested:292、BackgroundizeSignal:302-331（per-turn 注入，controller.go:917/1092）
 ├─ backgroundizeRequested:485-503（guard1: !foregroundTaskFromContext→false :489-491；guard2: ResumeSession→false :492-494）
 └─ runToolLoop:376+（P3 drain :390-393、P4 checkpoint :400-402）

internal/agent/teammate_store.go
 ├─ Assign:165-231（Fork:true,Silent:false :202 → StartForSession；flushMailbox :225；recordTask :226）
 ├─ syncStateLocked:131-138（懒同步；jm.Output status-only 消费 readOffset）
 ├─ pendingDependenciesLocked:243-262（jm.Output :250 + tasks.Status :246-248）
 ├─ Tasks:265-284（jm.Output 派生 :272）
 ├─ Complete:302-316（全库无调用者——续轮 continue 断点 E1）
 └─ 待新增：OnJobDone（implementation-plan T2：tasks 补存 + LastJobID 严格匹配置 idle + mailbox 唤醒）

internal/control/controller.go
 ├─ applyTeamCommand:6345+（/team-add ctx 装配 :6365-6373：context.Background()+WithForkSource+WithManager+WithSession+WithParentSession）
 └─ input.go:191-193（<background-jobs> 注入 leader 下一轮 input —— P-b 前缀路径）
```

### 级联风险（本次交互评审特有）

1. **R-a 时序陷阱**：observer 触发时 `j.status` 仍 Running、`j.done` 未关（recordCompletion 在 :645、发布在 :647-655、close 在 :656）→ OnJobDone 若回查 `jm.Output` 验证终态会读到 **Running**（且顺带消费 readOffset）。
2. **R-b 双真源漂移**：信封快照（m.completed）vs `jm.Output` 增量 vs tasks.Status 补存——三个时间点不一致；必须固定「信封/快照非消费、Output 消费、补存不可变」各自语义。
3. **R-c 单例 observer 覆盖**：`WithJobDoneObserver` 是单例（同 WithJobStartObserver L296）→ 未来第二个消费者 Set 会覆盖 TeammateStore 回调 → 完成事件静默失效。
4. **R-d jobCtx 不可用**：observer 运行在 run goroutine，ctx 是 jobCtx（无 WithManager）→ 回调内未来调 Assign 会报 "background execution is not available"（task.go L941）。
5. **R-e 同步回调阻塞收尾**：observer 同步执行会延迟 close(j.done)（L656）→ Wait / ClaimForegroundResult / onJobStart 的 done 消费者（delivery workspace 租约释放）全阻塞（risk-review 2.1 高）。

---

## 2. 主题 1：P1 信封次序 + 双通知竞争

### 2.1 次序（先信封入队，后完成事件，leader 消费最晚）

```
run goroutine（jobs.go:585-657）
  ① recordCompletion（:645）
       ├─ j.mu 快照 + 预渲染 envelope（:951-961，非消费）
       ├─ m.mu 内 append m.completed（:990-995）   ← P1 信封【已入队】
       ├─ m.mu.Unlock（:998）
       ├─ RecordDone / observer 挂点（:1000-1002） ← 完成事件【触发】
       └─ closing Notice（:1013-1015）             ← job 层 UI 通知【最后】
  ② j.status 发布（:647-655）
  ③ close(j.done)（:656）
leader 下一轮 turn：input.go:191-193 DrainCompletedNoteForSession → <background-jobs> 注入 input ← leader 读到信封【最晚】
```

**结论**：先投递信封（入队）→ 再触发完成事件 → 最后 leader 读到信封。完成事件与信封消费**不同步、不同通道、无竞争**——observer 触发时信封已在 m.completed，但 leader 的 `DrainCompletedNoteForSession` 在下一轮 turn 才执行，与 observer 的执行时机（run goroutine 内）在时间线上不重叠竞争。

### 2.2 teammate job 的两次通知问题

teammate job 非 silent（Assign L202）→ 完成时的既有通知通道：
- **通道 A**：job 层 closing Notice（recordCompletion L1013-1015，立即，UI/operator audience）；
- **通道 B**：P1 信封（m.completed → 下轮 input.go L191-193，model 可见）。

A+B 是 P1 既有双通道设计（UI 即时 + model 下轮），非完成事件引入。

完成事件驱动的**唯一新增通知**是 mailbox 唤醒（`notifyMail`，P6.2 既有方法）：仅当 `inboxRoot != ""` 且完成时 inbox 有积压。与 `PostMail` 即时通知的重复性：
- 持久模式：PostMail 写盘即通知 → `flushMailbox` 在 Assign 后清空 inbox → 完成事件查积压**必为新 mail**（R5 语义不重复）；
- 边缘竞态：PostMail 落在 flush 之后、完成事件查积压之前 → 一次即时 + 一次完成唤醒 = 同一封新 mail 通知两次。低频、语义均为「新 mail 到达」，可合并说明或接受。

**设计约束**：OnJobDone **禁止** emit 任何「job X 完成/推进」类 Notice——那与通道 A 重复，且违反 G7 零新发送。完成事件在 leader 侧的可见性**只**由通道 A/B 承担（既有）。

### 2.3 若违反零新发送（未来扩展）的后果

完成事件 → leader input 注入（P-b 前缀）→ committed history 随完成频率增长 → 后续 fork 前缀漂移 → **P6 缓存收益归零**（risk-review 2.7 高，G7 是硬护栏）。mailbox 唤醒 notice 走 `event.Sink`（不进 provider 输入）→ 前缀零影响，是「可通知」的唯一合法出口。

---

## 3. 主题 2：silent job 完成时 observer 触发吗；teammate 是否 silent

### 3.1 触发面

observer 挂在 recordCompletion **两处**（implementation-plan T1：suppressEnvelope 分支 L975-977 + 正常分支 L1000-1002）→ **silent（P5 fork）与 foreground job 完成时 observer 同样触发**。这与 `taskRecorder.RecordDone` 既有语义一致（jobs.go L963-978 注释：「The task lifecycle hook still fires so monitoring sees the terminal transition exactly like any other job」）。

特别提醒：**silent/foreground job 走 suppress 分支（L963-978）→ 该路径不产生信封、不 emit Notice，observer（L975-977）是其完成的唯一回调出口**。若未来把完成事件用于非 teammate 消费者，需意识到此路径在名单内。

### 3.2 teammate 全部非 silent

- 首次分配：`ContextRequest{Fork: true, Silent: false}`（teammate_store.go L202）→ `Fork && !Silent` → `StartForSession`（task.go L991-992）→ **P1 信封送达**；
- 后续分配：`ContinueFrom: ref`（L205）非 fork → `StartForSession` → 信封送达；
- 对比 P5 默认：buildTaskSpec L714-718 fork 默认 `Silent: true` → `StartSilentForSession`（L993）→ 信封抑制。

**结论**：teammate job 100% 走正常信封分支；silent 是 P5 fork 专属启动模式，与 teammate 无交集。

### 3.3 过滤护栏（G8）

OnJobDone 必须**只处理 `ts.tasks[id]` 已登记的 jobID**：teammate 的 recordTask（L226）在 Assign 成功后登记；P5 silent fork / 任意非 teammate 后台 job / foreground task 均未登记 → O(1) 查表即弃（仅记 debug 日志）。这同时兜住：
- silent fork 完成（observer 触发但无 teammate 副作用）；
- `startInvalid` job（L478-479 close(done)→recordCompletion，Failed）——未知 id 幂等丢弃；
- 非本会话 job（jobs 层 observer 是全 Manager 单例，teammate 会话与其它会话共享同一 Manager？——是：boot 单 Manager。OnJobDone 收到**所有会话**的后台 job 完成事件，必须按 tasks 过滤 + 可再按 LastJobID 匹配）。

---

## 4. 主题 3：readOffset 消费 / completion handler 偷读信封？

### 4.1 消费面与非消费面（关键事实）

| 路径 | 消费 readOffset | 消费 resultRead | 影响 |
|------|:---:|:---:|------|
| `recordCompletion` 信封（L954）| 否 | 否 | `jobResultTextLocked`（L895-910）非消费，全量 4KB 快照 |
| `ResultSnapshotForSession`（L1556）| 否 | 否 | 注释明言 read-only，与信封同源同渲染 |
| `JobSnapshot.Tail`（L1382）| 否 | 否 | P2 面板非消费 tail |
| `Output/OutputForSession`（L1113-1143）| **是**（L1124-1127 / readArtifactSinceOffsetLocked L1145-1182）| **是**（L1132-1135）| **全仓唯一消费点** |

### 4.2 completion handler 偷读分析

- **偷不走信封**：信封在 recordCompletion 时已用非消费快照预渲染并存入 m.completed；任何后续 `Output` 调用都不影响已渲染信封的字节。
- **但会偷走后续增量**：completion handler 若调 `jm.Output(id)` 只为取 status → 副作用消费 readOffset + resultRead → leader 之后用 `bash_output` 读该 job 时得到「消费点之后的空增量」→ 结果缺失。task job 的 result 路径（L1132-1135）尤其敏感：一次 status-only Output 会把 `resultRead` 置位，leader 后续 bash_output 读不到 result（虽 P1 信封已含 result 快照，可兜底）。
- **时序陷阱叠加**（R-a）：observer 触发时 `j.status` 尚未发布（发布在 recordCompletion 之后 :647-655）→ OnJobDone 内调 `jm.Output` 会读到 **Running** 且消费 readOffset——双重错误。

**设计约束**：
1. OnJobDone **只用回调参数 st** 与 `ts.tasks` 内存表，**禁止调 Output/OutputForSession**；
2. completion handler 若确需读终态输出（未来场景），用 `ResultSnapshotForSession`（非消费）而非 Output；
3. **推荐新增非消费 status-only 查询**（如 `Manager.StatusForSession(parentSession, id)`，内部仅读 j.status 不碰 readOffset/resultRead），并把 teammate_store 三处 status-only `jm.Output`（syncStateLocked L135、pendingDependenciesLocked L250、Tasks L272）迁移过去——这同时修复**既有偷读隐患**（/team-status 每次查询都消费 leader 的 bash_output 增量）。

---

## 5. 主题 4：backgroundize 信号在 teammate 上下文

### 5.1 双重隔离（结构保证，非运行时保证）

1. **foregroundTaskFromContext guard**：`backgroundizeRequested`（run_loop.go L485-503）第一行 `if !foregroundTaskFromContext(ctx) { return false }`（L489-491）。`WithForegroundTask` 只在 foreground 同步分支设置（task.go L1066 `runSessionMode(WithForegroundTask(ctx), ...)`）；teammate 走 RunInBackground 分支（task.go L940-1048），从不设置。
2. **信号不可达**：backgroundize 信号是 per-turn 注入（controller.go L917/L1092 `WithBackgroundizeSignal` 在 leader turn ctx 上）；job goroutine 的 ctx 链 = `m.root`（jobs.go L537）→ jobCtx（WithSession L558 + jobCtxKey L559）→ runCtx（task.go L999/L1138），不含任何信号。即使信号存在，guard 1 已拦截。

### 5.2 teammate Assign 的调用 ctx

/team-add → applyTeamCommand（controller.go L6365-6373）装配 `context.Background()` + WithForkSource + WithManager + WithSession + WithParentSession → **无 foregroundTask 标记、无 backgroundize 信号** → fork 的 run_loop 永不返回 `errBackgroundizeRequested`。

### 5.3 完成事件回调的 ctx 注意（未来自动 Assign）

observer 运行在 run goroutine，ctx 是 jobCtx：**含 jobCtxKey，但无 `jobs.WithManager`** → 回调内直接调 `Assign` → RunProfileSpec L941 `jobs.FromContext(ctx)` 失败 → "background execution is not available"。未来自动推进必须用装配好的 leader 模板 ctx（G3，risk-review 2.5.3）；MVP 不涉及（OnJobDone 不调 Assign）。

### 5.4 交叉点（背景化 job 的完成事件）

被 backgroundize 的 foreground task 最终以 `StartForSession` 注册（backgroundizeForeground task.go L1128，**非 silent**）→ 完成走正常分支 → **信封 + closing Notice + observer 全部触发**。此类 job 不在 ts.tasks → OnJobDone 过滤丢弃 → 无副作用。提醒：observer 是全仓单例，**所有**后台 job（含非 teammate）完成都经过它，过滤必须严格。

---

## 6. 主题 5：完成事件驱动的自动 Assign × P3 steer 语义

### 6.1 P3 机制本身不变

- 新 job 的 `pendingMessages` 为空队列（jobs.go L168-172 注释 + startForSession 初始化）；
- `flushMailbox`（L430-457）在 Assign 后把 mailbox 积压 `SendMessageForSession` 进新 job 的 steer 队列（有界 16/8KB，L237-239，拒绝不丢）；
- run_loop 每 tool round `DrainPendingMessages` 一条（L390-393）→ `session.Add(midTurnSteerMessage)` → 改变 teammate transcript（P-c 前缀，steer 一次 cache miss 不可避免，run_loop.go L381-382 注释）。

自动 Assign（未来）与手动 Assign 对新 job 的 steer 语义**逐字节一致**——队列结构、上限、FIFO、拒绝策略全同。

### 6.2 改变的三个面（时序/目标/阻塞）

1. **steer 窗口提前**：完成事件 → 立即新 job → mailbox 积压更快被 flush 进队列、更快消费。leader 侧感知滞后（信封下轮才到）→ 用户可能在「不知道新 job 已启动」时 steer 旧 jobID。
2. **jobID 错位**：自动推进更新 `LastJobID` 后，对旧 job 的 `send_message` 被拒（终态 job `SendMessageForSession` L1090-1092 返回 "job X is done, not running"）→ 报错文案必须引导查新 jobID（`/team-status` 已显示 `job=<id>`）；steer 队列对已终态 job 不可用（设计如此，拒绝不丢）。
3. **同步 Assign 阻塞收尾**：observer 内同步 Assign → 新 fork 的 prefill（task.go L875 prepareTranscriptRunWithPrompt = 捕获前缀 + transcript 磁盘写）串行进旧 job 的 run goroutine 收尾 → 延迟 close(j.done)（L656）→ 级联阻塞 Wait/claim/teardown（risk-review 2.1 高）。**自动推进必须 worker 化（G1 有界队列 + 单 worker），绝不同步 Assign**。

### 6.3 MVP 裁决

implementation-plan 明确「完成事件只落定状态、不派活」（依赖推进=状态落定，Assign 仍手动 /team-add）→ **P3 零影响**。§6.2 是未来自动推进的护栏清单，执行队不得在本次擅自引入自动 Assign。

---

## 7. 交互设计约束清单（给实现/审查的可执行护栏）

| # | 约束 | 锚点 | 违反后果 |
|---|------|------|---------|
| C1 | observer 挂 recordCompletion 两处无锁点（:975-977 与 :1000-1002），**禁入 m.mu 临界区** | jobs.go L963-998 | 死锁（m.mu 重入 StartForSession）|
| C2 | OnJobDone 只用 st 参数 + ts.tasks，**禁调 jm.Output**；终态输出读取用 ResultSnapshotForSession | jobs.go L1550 | 偷读 readOffset/resultRead + 读到 Running（R-a）|
| C3 | OnJobDone **禁 emit 完成/推进类 Notice**（零新发送 G7）；唯一可选通知 = mailbox 唤醒（event.Sink，不进 input）| input.go L191-193 | 双通知 + P-b 前缀漂移 |
| C4 | OnJobDone 只处理 `ts.tasks[id]` 已登记 job；未知/旧 jobID 幂等丢弃 | teammate_store.go L226/L243 | silent fork / 旧 job 完成误置 idle |
| C5 | observer 回调禁同步 Assign/PostMail/重活；自动推进必须 worker 化（本 MVP 不做）| jobs.go L645-656 | close(j.done) 延迟级联 |
| C6 | 回调 ctx 禁用 jobCtx（无 WithManager）；未来 Assign 用 leader 模板 ctx | task.go L941 | "background execution is not available" |
| C7 | 新增非消费 status-only 查询并迁移 syncStateLocked/pendingDependenciesLocked/Tasks 的 status-only Output | teammate_store.go L135/L250/L272 | 既有偷读持续存在 |
| C8 | `WithJobDoneObserver` 单例语义文档化；未来多消费者走注册表/fanout | jobs.go L296 | 覆盖后完成事件静默失效 |

---

## 8. 对抗自检（devil's advocate 攻击本评审）

1. **「信封先入队 → 完成事件后触发」是否可能在下一轮 drain 时反而晚于信封？** 攻击：observer 若 worker 化（G1 异步），完成事件处理可能晚于 leader 下轮读信封——但两者通道不同（信封=model input，完成事件=内存状态推进），无同一资源的读写竞争，晚到只影响 UI 侧感知，语义安全。本评审结论不依赖同步/异步。
2. **「teammate 非 silent」是否会被 future 代码打破？** 攻击：未来若有人直接构造 `ProfileExecSpec{Fork:true, Silent:true}` 绕过 Assign 启动 teammate job → silent。缓解：C4 按 tasks 过滤兜底，不依赖启动模式假设；Assign L202 显式 Silent:false 是契约而非巧合。
3. **「Output 偷读」是否言过其实？** 攻击：task job 输出走 result 而非 tail，readOffset 影响小。反制：resultRead（L1132-1135）才是敏感项——status-only Output 一次即置位 resultRead，leader 后续 bash_output 读不到 result；且 /team-status 高频调用（P6.2 现状）会系统性消费。C7 修复是必要而非可选。
4. **「backgroundize 与 teammate 零交集」是否遗漏 autoBackgroundize 路径？** 攻击：run_loop.go L499-503 的自动阈值在 foreground 分支才可能触发（L501 在 guard1 L489 之后）→ teammate 恒 false。已核对 L485-503 全文，无遗漏。
5. **本评审未覆盖**：完成事件与 `TeamStop`/`Remove`/`DestroyAll` 交错（worker 处理时 teammate 已移除）——补充：OnJobDone 处理前查 `ts.tasks`/`teammates` 存在性，缺失即丢弃（幂等，G9）；`TeamStop`/`Remove` 的 Kill → recordCompletion 正常分支 → observer 触发 → OnJobDone 对已 idle 的 teammate 幂等 no-op。
6. **「零新发送」是否牺牲产品可见性？** 反制：leader 感知 = P1 信封（下轮）+ closing Notice（即时）+ /team-status（查询），已覆盖三类需求；「实时推送完成」与 P6 前缀稳定架构冲突，若强制只允许 operator-audience Notice（G7 子集）。

---

## 9. 缓存/纪律检查点（Reasonix 领域）

- ✅ **发送侧字节零变化**：完成事件 = 进程内回调（jobs→agent）内存态推进；无新 input 注入、无新 transcript 写入、无 schema/system prompt 改动；teammate fork/continue 三要素与前缀完全不受影响。
- ✅ **前缀稳定**：mailbox 唤醒 notice 走 `event.Sink`（前端显示，不进 provider 输入）；steer 为既有 P3 路径（本就有 cache miss 语义，run_loop.go L381-382）。
- ✅ **锁序纪律**：observer 两处挂点均无 m.mu/j.mu 持有（:975-977、:1000-1002）；OnJobDone 先释 ts.mu 再做 ReadDir/通知；唯一 m.mu 内 Emit 是既有 droppedMsgs 路径（:988），完成事件不新增。
- ✅ **防虚假完成**：本评审为零代码变更的规划交付物；实现是否遵守 C1-C8 由纪律团代码审查（重点：observer 挂点位置、OnJobDone 无 Output、无新 Notice、tasks 过滤）+ `-race` 全量测试裁决。

---

*本文档为规划交付物；实现与验证由执行小队承担，纪律检查团复核。*
