# 测试策略：完成事件驱动的完整测试矩阵（P6 complete 子机制）

> 事务：docs/team/20260810-p6-complete/ · 视角：独立测试设计（只读规划，不写代码）
> 主题：为「job 完成事件 → jobs 层 observer → TeammateStore handler（置 idle + 依赖推进 + mailbox 唤醒 + tasks 补存）」设计完整测试矩阵，覆盖**真实异步**与**确定性**两条路径。
> 范围：只设计测试。实现细节（observer 注册形态、notifyJobDone 位置）以「测试对实现的硬约束」（见 §9）形式给出，交实现队/纪律团裁决。
> 依据源码：`internal/agent/teammate_store_test.go`（P6.1/P6.2 五测）、`internal/jobs/jobs_recorder_test.go`、`internal/jobs/jobs_silent_test.go`、`internal/jobs/jobs_test.go`（start-observer/destroy 范式）、`internal/jobs/jobs.go`（recordCompletion L937 为终态唯一出口）。

---

## 一、结论（先行）

1. **测试分两层、两条路径**：jobs 层（`jobs_done_observer_test.go` 新增）测 observer 契约；agent 层（建议新建 `teammate_done_test.go`）测 handler。每条又分**确定性**（直接调 `OnJobDone` 注入事件，无 flake）与**真实异步**（job 真跑完触发，验证全链）两种。
2. **确定性测试是矩阵主体**（约 2/3）：handler 是纯内存逻辑，直接调用即可覆盖全部业务分支（防旧完成、2 级依赖链、owner 删除、mailbox 唤醒、tasks 补存、幂等）。真实异步只保留 4 个组合测试 + 竞态骨架——数量少、价值高、flaky 面小。
3. **测试对实现的 5 条硬约束（D1-D5，§9）**：多订阅者注册表、Close/destroy 后不触发、per-subscriber panic 隔离、`notifyJobDone` 必须**同步**于 `recordCompletion`、handler 锁内零 IO。前三条不满足则对应矩阵项无法编写；第四条不满足则组合测试的确定性等待点（`WaitForSession` 返回 ⇒ handler 已执行）崩塌。
4. **现有测试零破坏**：completion 当前无 observer，新增 observer 是纯增量；teammate 五测在事件驱动下断言仍成立（懒同步变兜底、更强而非冲突）；`-race` 全量回归本身就是 observer 锁序错误的护栏。
5. **竞态测试不依赖真实时序**：真实 job 完成做骨架，竞态靶点用手动注入的 `OnJobDone` 并发打击——可控、可复现、不 flake。

---

## 二、现有测试锚点（回归基线，全部必须保持绿色）

| 锚点 | 文件:行 | 与本次的关系 |
|------|---------|-------------|
| `testTaskToolForTeam`（mockProvider 立即 text+done chunk） | teammate_store_test.go:17 | **组合测试复用**：最快终止 job 的既有模式（provider 侧） |
| `TestTeammateAssignStartsBackgroundJobWithEnvelope` 断言 List 后 idle（懒同步） | teammate_store_test.go:110-112 | 事件驱动后 List 前即 idle，断言仍真（更强，非冲突） |
| `TestTeammateDependencyGate`（WaitForSession 后 Assign 放行） | teammate_store_test.go:169-203 | tasks 补存后 `pendingDependenciesLocked` 查 tasks 即放行，仍成立 |
| `TestTeammatePostMailPersistsAndFlushes`（磁盘 inbox + steer flush） | teammate_store_test.go:118 | mailbox 唤醒测试复用其 inbox 落盘/清空语义 |
| `TestTeamMessageToolPostsMail`（builtin.WithMailbox） | teammate_store_test.go:208 | 不变 |
| `recordingRecorder`（mutex + snapshot） | jobs_recorder_test.go:14-38 | done-observer 测试的捕获器同构 |
| `TestTaskRecorderHook_SetAfterConstruction` / `NilRecorderIsNoop` | jobs_recorder_test.go:101-127 | 构造后装配 + nil no-op 范式 |
| `TestStartSilentStillFiresTaskRecorder` | jobs_silent_test.go:71-90 | **silent job 仍触发 hook** 的既有语义 → done-observer 同构 |
| `TestJobStartObserverSeesLifetimeUntilTerminal`（channel 观察范式） | jobs_test.go:84-115 | done-observer 触发测试的写法参考 |
| `TestDestroySessionCancelsOwnedJobsAndSuppressesCompletion`（destroy 窗口） | jobs_test.go:630-655 | destroy 边界测试范式（BeginDestroySession + WaitTeardown） |
| `recordingSink` / `blockingFinishedSink` / `waitFor` | jobs_test.go:17-65 | jobs 包捕获器与轮询工具（**agent 包不可跨包复用，需自建 captureSink**） |
| `recordCompletion` 两处无锁 `RecordDone`（L975-977 suppress 分支、L1000-1002 正常分支） | jobs.go | done-observer 并列接线点；**注意 suppress 分支无 destroying 检查**（见 §9 D2） |
| `startInvalid`（close(done) → recordCompletion） | jobs.go:478-479 | observer 必须覆盖 Failed（startInvalid 也是终态出口） |
| run goroutine（recordCompletion → close(j.done)） | jobs.go:645→656 | **同步性基石**：notifyJobDone 同步于 recordCompletion 时，WaitForSession 返回 ⇒ observer 已执行完 |
| `syncStateLocked`（L131，懒同步，注释明言 MVP 无回调） | teammate_store.go:131-138 | 事件驱动后降级为兜底，**保留不删**（回归锚点） |
| `Complete`（L302，LastJobID 严格匹配语义） | teammate_store.go:309-311 | handler 防旧完成逻辑照抄此语义 |
| boot.go:1930 `agent.NewTeammateStore(taskTool, jm)` 内联，未 SetSink | boot.go | 生产接线缺口（T3），非测试范围但 mailbox 唤醒组合测试需显式 SetSink |

---

## 三、分层总览

| 层 | 文件（建议） | 确定性 | 真实异步 | 竞态 |
|----|-------------|--------|---------|------|
| jobs observer | `internal/jobs/jobs_done_observer_test.go`（新增） | 触发/多订阅者/panic/Close/destroy/silent/Set-时序 | 必须（observer 触发只能真 job 验证） | 并发 start/done 恰一次 |
| agent handler | `internal/agent/teammate_done_test.go`（新增，同包） | 置 idle/防旧/未知/2 级链/owner 删/mailbox/tasks 补存/幂等 | 组合全链 4 例 | Assign vs 完成事件 |
| 回归护栏 | 既有五测 + jobs 全量 | — | — | `-race` 全量 |

---

## 四、jobs 层单测矩阵（`jobs_done_observer_test.go`）

前置（硬约束 D1）：observer 支持**多订阅者注册表**——`WithJobDoneObserver(fn)` 可多次调用追加，全部订阅者收到同一事件；`SetJobDoneObserver(fn)` 清空后设单一（对称 `SetTaskRecorder` 的构造后装配）。

| # | 测试 | 方式 | 检查点（验证点） |
|---|------|------|-----------------|
| J1 | `TestJobDoneObserverFiresOnDone` | 真实：`StartForSession` 闭包立即 `return "answer", nil` → `Wait` | 回调收到 (parentSession, id, kind, label, Done) 恰一次；参数与 `*Job` 字段一致 |
| J2 | `TestJobDoneObserverFiresOnFailed` | 真实：run 返回 error | Failed、恰一次 |
| J3 | `TestJobDoneObserverFiresOnKilled` | 真实：run 阻塞 `<-release` → `Kill` | Killed、恰一次；Kill 后终态仍走 recordCompletion（终态出口唯一） |
| J4 | `TestJobDoneObserverFiresOnInvalidStart` | 真实：parentSession 含 `/` 触发 `startInvalid` | Failed 也通知（覆盖 close(done)→recordCompletion 路径） |
| J5 | `TestJobDoneObserverMultipleSubscribers` | 真实 | **两个订阅者都收到同一事件**（fanout 顺序不定、次数各一）；这是 D1 的验收 |
| J6 | `TestJobDoneObserverPanicIsolated` | 真实 | 订阅者 A panic、订阅者 B 正常 → B 收到、job 仍 Done、`Wait` 正常返回、manager 可继续用（D3 验收） |
| J7 | `TestJobDoneObserverNotCalledAfterClose` | 真实：阻塞 job → `Close()`（cancel）→ job 以 Killed 完成 | 订阅者**未收到**（D2：notifyJobDone 检查 `m.root.Done()`） |
| J8 | `TestJobDoneObserverNotCalledWhileDestroying` | 真实：阻塞 job → `BeginDestroySession` → `WaitTeardown` | 订阅者未收到；**silent 变体**（`StartSilentForSession`）同样未收到（D2：两种路径一致） |
| J9 | `TestJobDoneObserverSilentAndForegroundStillFire` | 真实 | silent/foreground job 完成 → 通知（同构 `TestStartSilentStillFiresTaskRecorder`，仅 destroy 窗口例外） |
| J10 | `TestJobDoneObserverNilNoop` | 构造 | 无订阅者：零开销、不 panic、行为与现状逐字节一致 |
| J11 | `TestSetJobDoneObserverAfterConstruction` | 构造后 `SetJobDoneObserver` | Set 后启动的 job 触发；Set 前已完成的 job 不补发（时序语义固化） |
| J12 | `TestJobDoneObserverBlockingStallsWait`（文档化权衡） | 真实：订阅者阻塞 `<-release` | 证明回调同步于 `recordCompletion`（Wait 被延迟）——**固化 D4 语义**，防实现擅自改 goroutine 分发 |

回归：`go test ./internal/jobs/`（重点 `TestDrainMultiple`、`TestDestroySession*`、`TestStartSilent*`）零失败。

---

## 五、agent 层 handler 单测矩阵（`teammate_done_test.go`，全部确定性）

前置（D4/D5）：`OnJobDone` 同步调用、锁内零 IO。方式：`ts.OnJobDone("sess", jobID, kind, label, st, err)` 直接注入——**不需要真实 job、不需要 jm**（除 mailbox 用例需 inboxRoot + captureSink）。

| # | 测试 | 场景注入 | 检查点 |
|---|------|---------|--------|
| H1 | `TestTeammateOnJobDoneIdleTransition` | Create → 手动置 Running+LastJobID → 注入 Done | `Status(name).State == Idle`（**不调 List/Tasks**，证明事件驱动而非懒同步） |
| H2 | `TestTeammateOnJobDoneStaleJobNoop` | Running+LastJobID=J2 → 注入旧 J1 完成 | 仍 Running（**LastJobID 严格匹配**，R3，同 Complete L309-311） |
| H3 | `TestTeammateOnJobDoneUnknownJobNoop` | 注入未 track 的 id | teammate 状态不变、tasks 无新增、无 panic |
| H4 | `TestTeammateOnJobDoneRunningStatusIgnored` | 注入 st=Running | 直接返回（只处理终态），teammate 仍 Running |
| H5 | `TestTeammateDependencyChainTwoLevels` | 2 级链 a→b→c，**纯内存**：Create 三人 → 手动 Assign 骨架（或直接 recordTask）→ 依次注入 a 完成、b 完成 | 注入 a 完成后 `Assign(b, dep a)` 放行；注入 b 完成后 `Assign(c, dep b)` 放行——**依赖推进 = 终态落定到 `tasks[].Status`** |
| H6 | `TestTeammateDependencyFailureUnblocks` | 注入 dep 以 Failed/Killed 落定 | 同样放行（`pendingDependenciesLocked` 终态集合覆盖全部 5 态） |
| H7 | `TestTeammateOnJobDoneOwnerRemoved` | Create → Remove → 注入完成 | 不 panic；teammate 不存在；`tasks[id].Status` 仍补存（任务记录与成员生命周期解耦） |
| H8 | `TestTeammateMailboxWakeupOnDone` | inboxRoot + captureSink；inbox 有积压 → 注入 Done | 收到 `"teammate X received mail"` notice；**无积压 → 无 notice**（R5 防噪音） |
| H9 | `TestTeammateTaskStatusPersistedOnDone` | 注入 Done → 断言 `Tasks()` 条目 Status==Done；再**断开 jm**（nil 或未 Start 的 id）断言 Tasks() 兜底返回补存值 | R4 双真源：jm 优先、补存兜底，purge 后不丢状态 |
| H10 | `TestTeammateOnJobDoneIdempotent` | 同一事件注入两次 | 状态不变、notice 不重复（handler 幂等，R4/R5 兜底） |

---

## 六、组合测试（真实异步全链，4 例）

复用 `testTaskToolForTeam`（mockProvider 立即 done chunk = 最快终止 job 的 agent 侧模式；jobs 侧最快模式是 `StartForSession` 闭包立即返回）。**确定性等待点**：`recordCompletion`（L645）→ `notifyJobDone` → `close(j.done)`（L656）同 goroutine 串行 ⇒ `jm.WaitForSession` 返回时 handler 必已执行完，**无需轮询**（依赖 D4，见 §9）。

| # | 测试 | 流程 | 检查点 |
|---|------|------|--------|
| C1 | `TestTeammateCompletionDrivesIdleEndToEnd` | Create → `jm.SetJobDoneObserver(ts.OnJobDone)`（显式接线）→ Assign → `WaitForSession` → 直接断言 `ts.Status` | **不调 List/Tasks** 即 Idle——证明事件驱动全链（Assign→job→recordCompletion→observer→handler） |
| C2 | `TestTeammateCompletionDependencyChainRealJobs` | alpha 先 Assign → WaitForSession(jobA) → 不调 Tasks → `Assign(beta, ..., dep jobA)` | 真实事件链下依赖放行（强化现有 `TestTeammateDependencyGate`：不触发懒同步也放行） |
| C3 | `TestTeammateMailboxWakeupRealJob` | inboxRoot+captureSink → Assign → **job 运行中 PostMail**（落盘积压，flush 不跑）→ WaitForSession → 断言 notice | 「完成事件查到的积压必为未 flush 新 mail」语义（R5）：运行中 PostMail + 完成唤醒正是价值场景，非噪音 |
| C4 | `TestTeammateCompletionFailedKilledChain`（终态全覆盖） | mockProvider 注入错误 / `Kill(jobID)` | Failed/Killed 同样置 idle + tasks 补存终态（H6 的真实异步版） |

---

## 七、时序竞态测试（-race）

原则：**真实异步只做骨架，竞态靶点用手动注入并发打击**——真实完成事件不可控（无法保证在 Assign 写 LastJobID 的精确窗口到达），手动注入才可复现。

| # | 测试 | 设计 | 检查点 |
|---|------|------|--------|
| R1 | `TestTeammateAssignCompletionRace` | 单 teammate 循环：Assign → WaitForSession → 断言 idle → 再 Assign；**同时**后台 goroutine 反复注入完成事件（旧/新 id 混合）与 Assign 并发 | `-race` 无报告；终态必为 Idle（新 job 完成）或 Running（新 job 未完成），绝不因旧事件误置（R3 竞态版） |
| R2 | `TestJobDoneObserverConcurrentStartDone` | 10 个 job 并发启动/完成 | 每 job 每订阅者恰一次（注册表并发安全，D1） |
| R3 | `TestTeammateRemoveDuringCompletionRace` | goroutine A 循环 Create/Remove；goroutine B 注入完成事件 | 不 panic（H7 竞态版；Remove 后 handler 路径零副作用） |

命令：`go test -race ./internal/jobs/ ./internal/agent/ -run 'DoneObserver|Teammate'`；CI 门：`go test -race ./...`（现有全量即回归护栏）。

---

## 八、确定性 vs 真实异步划分（为什么这样分）

| 场景 | 方式 | 理由 |
|------|------|------|
| observer 触发（J1-J9、J12） | **必须真实** | 触发点在 recordCompletion 的 goroutine/锁/Close/destroy 边界，直接调用无法模拟 |
| observer 注册形态（J5/J6/J11/J10） | 真实（构造+启动 job） | 注册/替换/panic 与管线交互只能真 job 验证 |
| handler 全部业务分支（H1-H10） | **直接调用** | handler 是纯内存逻辑；真实 job 只增加时序噪音，不增加覆盖 |
| 组合全链（C1-C4） | **必须真实** | 验证 Assign→job→observer→handler 全链路与接线（`SetJobDoneObserver` 是否真被调） |
| 竞态（R1-R3） | 真实骨架 + 手动注入靶点 | 手动注入可精确打击 LastJobID/State 临界区，真实时序不可控 |
| 现有回归（teammate 五测 + jobs 全量） | 不变 | 零改动即护栏 |

**划分边界**：一切「状态落定结果」断言放确定性层（快、准、无 flake）；一切「触发/接线」断言放真实异步层（少而精）；重叠处（如 H1 vs C1）两层都写，H1 保护逻辑、C1 保护接线。

---

## 九、实现决策点（测试对实现的硬约束）

> 若实现不满足 D1/D2/D3，对应矩阵项无法编写；若实现不满足 D4，组合测试需改轮询（flake 面扩大）。测试策略按以下约束设计，实现偏离需纪律团显式裁决并删/改对应矩阵项。

| # | 约束 | 依据测试 | 缺省后果 |
|---|------|---------|---------|
| D1 | **多订阅者注册表**：`WithJobDoneObserver` 可多次调用追加，`SetJobDoneObserver` 清空后设单。不采用实现计划方案 A 的单例覆盖 | J5/J6/R2 | 多订阅者矩阵不可写；panic 隔离退化为单订阅者场景 |
| D2 | **Close/destroy 后不触发**：notifyJobDone 检查 `m.root.Done()` 与 destroying（含 **suppress 分支**——注意 recordCompletion L975-977 现无 destroying 检查，须在 notifyJobDone 统一补齐，与 taskRecorder 现有行为略有出入，但更符合销毁期不通知直觉且不影响 taskRecorder 语义） | J7/J8 | "Close 后不触发"矩阵项不可写；destroy 窗口 silent job 会漏通知 handler（幂等可兜底但语义脏） |
| D3 | **per-subscriber panic 隔离**：notifyJobDone 逐个 recover，一个订阅者 panic 不影响其他订阅者与 recordCompletion/close(j.done) | J6 | 订阅者 panic 打穿 run goroutine → job 管线损坏 |
| D4 | **同步调用**：notifyJobDone 同步于 recordCompletion（不另起 goroutine），回调在 close(j.done) 之前完成 | J12/C1-C4 | `WaitForSession` 返回 ≠ handler 已执行，组合测试失去确定性等待点，必须轮询 |
| D5 | **handler 锁内零 IO**：ts.mu 内只做 O(1) 内存（置 idle + tasks 补存 + 找命中者），ReadDir/notifyMail 在锁外 | H8/R1 + 现有 `-race` 全量 | `ts.mu ↔ jm 锁`逆序死锁（R1）；mailbox 用例在锁内 ReadDir 拖慢 |

---

## 十、现有测试破坏性分析（逐项）

1. **jobs 层现有测试**：completion 当前无 observer，新增字段默认 nil → 行为逐字节不变（J10 固化）。`TestDestroySession*`（L630/663）不涉及 observer → 无影响。
2. **`TestTeammateAssignStartsBackgroundJobWithEnvelope`**（teammate_store_test.go:110）：事件驱动后 List 前即 idle，`List()` 断言仍真（更强）。**不破坏**。
3. **`TestTeammateDependencyGate`**（L169）：WaitForSession 后 `Assign(beta, dep first)`——补存后 `pendingDependenciesLocked` 查 `tasks[dep].Status` 即放行，路径不变。**不破坏**。
4. **mailbox 三测**（PostMail/TeamMessageTool）：均未接线 done-observer，断言不涉及完成事件。**不破坏**。
5. **`syncStateLocked` 保留**：惰性兜底是**删除**才破坏（现有测试依赖 List 懒同步），本次保留 → 无行为变化。
6. **隐藏风险（正是护栏价值）**：若实现违反 D5（锁序）或 D2（suppress 分支 destroy 通知），`go test -race ./...` 现有全量立即红——现有测试是 observer 正确性的第一道闸。

---

## 十一、对抗自检（devil's advocate）

1. **「多订阅者注册表是过度设计——实现计划方案 A 是单例」**：用户要点 1 明确要求「多订阅者」矩阵项；fanout O(n) 同步、n≤3，开销可忽略；TeammateStore 之外未来 taskmonitor/workspaceLease 都是候选订阅者。**薄弱环节**：若纪律团裁单例，删 J5/J6(多订阅者版)/R2 三项，J6 降级为「单订阅者 panic 不破坏管线」。已标注，裁决成本低。
2. **「『WaitForSession 返回即 handler 完成』依赖 D4 同步调用——万一实现改 goroutine 分发」**：J12 把同步语义固化成测试（阻塞订阅者 ⇒ Wait 延迟），实现偏离时 J12 与 C1-C4 同时暴露。**薄弱环节**：J12 用「阻塞」断言同步是负向验证，可能被误读为「回调必须慢」——文档已标注为权衡固化而非性能目标。
3. **「手动注入 OnJobDone 绕过 jm 状态，可能固化错误行为（补存值与真实 jm 冲突）」**：H9 的「断开 jm 兜底」专门覆盖；C1/C2 真实链覆盖 jm 一致侧。双面夹击后仍留漂移窗口（R4）——已按实现计划「jm 优先、补存兜底」顺序固化，属接受边界。
4. **「mailbox 唤醒在运行中 PostMail 场景是重复通知（PostMail 已即时 notifyMail）」**：运行中 PostMail 的即时通知已发过，但 mail 未被 flush（teammate running 时 flushMailbox 不跑）→ 完成事件后的唤醒通知**语义不同**（告知 leader 可重新 Assign 以 flush），非噪音。C3 固化该价值场景，H8 固化无积压不通知。
5. **「竞态测试用手动注入可能测不出真实窗口」**：手动注入是**靶点**，真实 job 完成是**骨架**（R1 的 WaitForSession 真实触发 + 手动并发打击）；真实窗口不可复现是测试基本原则（非确定性即弃），接受此边界。
6. **「现有测试不破坏的结论可能因 boot 接线（T3）而失效」**：T3 只在 boot 层接线，不改 agent/jobs 行为；teammate 五测不跑 boot 路径。**薄弱环节**：`internal/control` 的 team 命令测试若断言「List 懒同步」时序（先 Wait 后 List）——已核实 teammate_store_test 无此时序依赖；control 层测试在 T3 后全量回归兜底。

---

## 十二、缓存/纪律检查点（Reasonix 领域）

- ✅ **发送侧字节零变化**：observer 与 handler 均为进程内回调（jobs→agent），不触碰 schema/system prompt/transcript/task.go/subagent_store；全部测试只读生产字节、不改发送内容。
- ✅ **前缀稳定**：无新发送前缀结构、无历史插入、无 canonical 重写；mailbox 唤醒 notice 走 event.Sink（前端显示），不进 provider 输入。
- ✅ **运行值不进 schema/system prompt**：tasks.Status、LastJobID、mail 积压均为内存态/磁盘态，非发送前缀组成部分。
- ✅ **锁序纪律**：测试矩阵把「handler 锁内零 IO」（D5）与「observer 无锁点调用」（J12）固化为断言，`-race` 全量作为持续闸。
- ✅ **防虚假完成**：每个矩阵项带明确检查点；「完成」= 测试输出 + `go test -race ./...` 全绿 + 事务四验证（gofmt/vet/repolint/test），execution.md 留证据链，discipline 独立审查。

---

## 附：执行队禁止事项（按章程）

- 不得把本测试策略的 D1（多订阅者）擅自解释为可选——除非纪律团显式裁决，否则实现必须满足以保持矩阵完整。
- 不得删除 `syncStateLocked` 懒同步兜底（现有五测的回归锚点）。
- 不得把 J12 的同步语义改写成「goroutine 分发 + 轮询」而不更新 C1-C4 的等待点设计（flake 责任在实现方）。
- 不得声称完成——须附测试输出与 -race 证据。
