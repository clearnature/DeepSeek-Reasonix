# 设计：完成事件驱动下 TeammateStore 侧 handler 的完整生命周期（lifecycle-design）

> 角色：**规划 Planner**（独立评审，只读，零实现代码）
> 事务：`docs/team/20260810-p6-complete/` · 基线：`internal/agent/teammate_store.go`（Assign L165-260 / recordTask L234 / pendingDependenciesLocked L243 / Remove L346 / Tasks L265 / Complete L302 / syncStateLocked L130 / SetSink L153）、`internal/jobs/jobs.go`（recordCompletion L937 / startInvalid L479 / run goroutine L645 / KillForSession L1211 / Close L2138）、`docs/team/20260810-p6-complete/jobsevent-design.md`（jobs 层完成事件契约，本设计为其**消费侧**）
> 对象：评审「job 终态 → jobs 层 observer → TeammateStore.OnJobDone」链路上 handler 的**生命周期与幂等**——6 个主题，结论先行 + 代码锚点
> 实现与验证由执行小队承担、纪律团复核（三职能硬性分离；本文件只产出设计）

---

## 零、结论先行（TL;DR）

| # | 主题 | 裁决 | 一句话理由 | 关键锚点 |
|---|------|------|-----------|---------|
| 1 | 同 jobID 重复触发完成事件 / handler 幂等性 | **事件源恰好一次**（recordCompletion 全仓仅两处互斥调用点，每 job 单路径）；**但 handler 必须自幂等**——①tasks 终态只写一次（已有终态跳过）；②teammate 置 idle 仅在 `Running→Idle` 翻转发生时；③mailbox 唤醒**绑定翻转成功**才触发（防重复通知） | 双层防御：源端唯一出口 + 消费端幂等契约；防御测试注入 + 事件真实双达 + 懒同步双路径 | jobs.go:479/:645（互斥）；teammate_store.go:302-316（Complete 幂等范式） |
| 2 | teammate 在 job 完成前被 Remove | **handler 找不到 teammate 是正常路径，no-op**；`tasks[id]` 过滤优先（不存在 → 非 teammate job，直接 return），owner 缺失 → 跳过置 idle，tasks 终态补存照常（若 Remove 同步清 tasks 则全 no-op） | observer 是全 Manager 单例，会收到**所有会话所有 kind** 的事件，必须按 tasks 过滤；Remove 先删 map 再 Kill，Kill 的 Killed 事件必然到达已删的 teammate | teammate_store.go:346-362（Remove）、L1211-1233（Kill 同步置 Killed） |
| 3 | 完成时 job 是 killed（/team-stop 杀） | **killed 是终态**：①tasks.Status 照写 "killed"；②依赖门**放行**（防依赖永久死锁）；③teammate 照常置 idle（防卡死）；④**自动推进不触发**（用户主动终止，自动重启后继 = 覆盖用户意图）——「放行依赖门（手动可继续）」与「自动推进（自动不继续）」是两回事，必须显式区分 | TeamStop 杀掉的 job 也走 recordCompletion（Killed），此时 teammate 已被 TeamStop 置回 Idle → handler 置 idle 步骤因 `Running` 条件不满足而 no-op（幂等自洽） | teammate_store.go:321-341（TeamStop）；:254-255（终态集合含 killed）；jobs.go:594-595（ctx cancelled → Killed） |
| 4 | 完成 → 自动 Assign 下一个 → LastJobID 覆盖语义 | **单值 LastJobID + 严格匹配 guard 在连续任务下成立**（J1 事件迟到时 `LastJobID==J2≠J1` → guard 挡住）；tasks 补存（按 jobID 索引）与 teammate 置 idle（按 LastJobID 匹配）**双层解耦**是并发安全关键；**自动推进必须 worker 化**（datamodel 三禁：OnJobDone 内禁同步 Assign） | handler 只负责「终态落定 + 置 idle」，推进由独立 worker 从 ts.mu 外读状态再 Assign；LastJobID 置 idle 后保留不清空（kill 审计 + tasks 关联），活跃性由 State 表达 | teammate_store.go:309-311（guard）；datamodel-design §4.2 |
| 5 | Assign 的 ctx 生命周期 | **完成回调零 ctx 依赖**（内存操作 + os.ReadDir + sink.Emit 均不需要 ctx；jobs 层回调签名也不携带 ctx）；自动推进所需 ctx **绝不缓存 Assign 的 ctx 对象**（turn ctx，job 完成时早已销毁），改为**存 `TeamTask.SessionID` 字符串**，重放时 `context.Background()` + With* values 重建；session 已关闭的推进降级为日志 + 丢弃（destroying 主路径已被 jobs 层 L982-985 挡掉） | ctx 是接口、不可序列化、带 deadline/values，跨 job 生命周期即失效 | teammate_store.go:225（`jobs.SessionFromContext(ctx)` 是现成来源）；datamodel-design §1.2 |
| 6 | handler 内部错误处理 | **slog 分层记录为主，不新增完成类 Notice**；ReadDir 失败 → slog.Warn + 静默降级（mailbox 唤醒是可选增强，下次 Assign 的 flushMailbox 兜底）；mailbox 唤醒段包 defer recover（防可选通知变管线事故）；Killed/Failed 的 user 可见性**复用 recordCompletion 既有 closing Notice**（L1004-1015），handler 绝不重复 Emit | 完成类通知无限流（risk-review §2.5 fan-out 刷屏风险）；P1 信封已投递结果，leader 下一轮自然读到 | jobs.go:1004-1015（既有 Notice）；teammate_store.go:402-410（notifyMail 模式） |

**一条总纲**：handler =「tasks 过滤 → 单临界区原子落定（tasks 终态 + teammate 翻转）→ 锁外副作用（mailbox 唤醒 + 日志）」；**终态落定与置 idle 解耦、置 idle 与推进解耦、handler 与 ctx 解耦**。越过此边界的做法（handler 内同步 Assign、缓存 Assign ctx、锁内 IO、新增完成 Notice、依赖 j.done）均为设计否决项。

---

## 一、拓扑扫描

### 1.1 handler 链路的文件依赖图

```
internal/jobs/jobs.go
 ├─ recordCompletion :937-1016（终态唯一出口）
 │    ├─ 调用点① startInvalid :479（同步于调用者 goroutine；close(j.done) 在前 :478）
 │    ├─ 调用点② run goroutine :645（recordCompletion → status 发布 :647-655 → close(j.done) :656）
 │    ├─ destroy 窗口 :982-985（提前 return，observer 不触发）
 │    ├─ suppressEnvelope 分支 :963-978（silent/foreground 仍触发 recorder）
 │    └─ closing Notice :1004-1015（Failed→warn / Killed→info，用户可见性已由 jobs 层承担）
 ├─ JobCompletion / JobCompletionObserver（jobsevent-design 新增；同步投递、per-observer recover）
 └─ KillForSession :1211-1233（同步置 Killed :1225 → cancel :1231 → run goroutine 仍走 :645）

internal/agent/teammate_store.go（handler 所在）
 ├─ Teammate L24-36（LastJobID L31 / State L32）
 ├─ TeamTask L53-58（ID/Owner/DependsOn/Status；缺 Prompt/SessionID/CreatedAt —— datamodel 已裁决补存）
 ├─ tasks map L72（jobID → TeamTask；recordTask 只在 Assign 成功路径登记 L226 → tasks 过滤 == teammate 过滤）
 ├─ recordTask L234-238（Status 恒空串 —— 需补初值 "running"）
 ├─ pendingDependenciesLocked L243-262（终态集合 L255 含 killed/interrupted/cancelled）
 ├─ Complete L302-316（LastJobID 严格匹配 guard；全库无调用者 —— 遗留 API）
 ├─ Remove L346-362（先删 map :354 → 锁外 Kill :358；不清 tasks —— datamodel P3 建议清）
 ├─ TeamStop L321-341（先置 Idle :333 → 锁外 KillForSession :337）
 ├─ syncStateLocked L130-138（懒同步兜底；jm.Output 消费性 —— datamodel §3.4.5 改非消费）
 ├─ notifyMail L402-410（sink Emit 模式；无 recover）
 └─ flushMailbox L430-457（下次 Assign 时兜底清 inbox —— mailbox 唤醒失败的退路）

internal/boot/boot.go（生产接线点：jm.SetJobDoneObserver(ts.OnJobDone) + ts.SetSink(sink)）
```

### 1.2 级联风险（本设计必须覆盖）

- **R-a 事件面比 teammate 宽**：observer 是全 Manager 单例 → OnJobDone 收到**所有会话、所有 kind**（bash/fleet/parallel_tasks）的完成事件。不按 `tasks[id]` 过滤 → 每个非 teammate 后台任务都会空跑 handler（无害但噪音）甚至误伤（若错误地按 id 遍历 teammates）。**过滤是第一行**。
- **R-b 翻转条件不严 → 重复通知**：handler 若不把 mailbox 唤醒绑定「Running→Idle 翻转发生」，重复事件（测试注入 + 真实双达）或 List 懒同步先置 idle 后事件到达 → backlog 存在即重复 notifyMail。
- **R-c killed 语义混用**：若把「依赖门放行」误读为「自动推进触发」→ 用户 /team-stop 杀 job 后系统自动重启后继 → 无限循环 + 用户意图被覆盖。
- **R-d ctx 失效**：若未来自动推进缓存 Assign 的 ctx 对象（turn ctx）→ job 完成时 ctx 已 cancel，`Assign(ctx, ...)` 启动的 job 立即 Killed 或 panic。
- **R-e 锁序/重入**：handler 从 jm 无锁点进入，持 ts.mu 写 tasks/teammates；**临界区内禁调 ts.jm.\***（否则 jm→ts.mu→jm.mu 重入环）、禁 IO（ReadDir 放锁外）、禁同步 Assign（datamodel 三禁）。
- **R-f handler panic 打穿管线**：jobs 层 per-observer recover（jobsevent D7）兜底，但 handler 内可选段（mailbox）应自包 recover，避免把增强功能变成状态写一半的 panic。
- **R-g 终态回退**：若 tasks.Status 写入不设「终态不可变」，乱序事件（J2 done 先到、J1 killed 后到——不同 task 条目，无冲突；同条目不可能乱序，但防御性）或未来事件重放可能把终态回退为另一终态。

---

## 二、多路径推演（handler 实现形态）

### 方案 A（采纳）：单临界区同步 handler，幂等翻转 + 锁外副作用
- **结构**：jobs 同步回调（jobsevent D5）→ `OnJobDone(id, st, ...)`：
  1. `st == jobs.Running` → return（只处理终态，防御）；
  2. `ts.mu.Lock()`：`tasks[id]` 不存在 → unlock + return（**过滤**，非 teammate job）；
  3. 同临界区：`tasks[id].Status` 若为空/非终态 → 写 `string(st)`（终态不可变）；遍历 `ts.teammates` 找 `tm.LastJobID == id && tm.State == TeammateRunning` → 置 `TeammateIdle`，记 `flipped=true`；
  4. `ts.mu.Unlock()`；
  5. `flipped && ts.inboxRoot != ""` → 锁外 `countInbox(name)`（ReadDir 容错）→ 有积压 → `notifyMail(name)`（可选段 defer recover）。
- 复杂度：低-中（teammate +~45 行；无新结构）。
- 性能：O(1) 内存 + 每翻转一次一次 ReadDir（毫秒级）；零 goroutine、零队列。
- 可维护性：单锁保证 tasks/teammates 原子一致；幂等点显式（`Running` 条件 + 终态不可变 + `flipped` 门控）；与 Complete 匹配语义同构。
- 风险：同步回调阻塞 job 收尾（jobs 层 recover + 文档约束「快速返回」，本 handler 全部内存级 + 可选 ReadDir）；自动推进需另行 worker 化（本方案 handler 本身不含推进）。

### 方案 B（否决为主选，记录代价）：handler worker 化（队列 + 单 goroutine）
- **结构**：handler 只做非阻塞 enqueue（`chan JobCompletion`），独立 worker 串行消费：置 idle + 推进 + mailbox。
- 复杂度：中-高。三件难事：①**队列生命周期**——DestroyAll/Close 时排空还是丢弃？排空拖慢关闭、丢弃破坏状态；②**背压**——容量满时阻塞 enqueue 违背初衷、丢弃破坏一致性；③**测试确定性**——异步消费下 `WaitForSession` 返回 ≠ handler 已执行，组合测试失去确定等待点（test-strategy D4 明确依赖同步）。
- 性能：每事件一次 channel 往返 + 调度延迟。
- 风险：R-g 放大（关闭时队列残留）；MVP 无自动推进，队列只为「置 idle + 补存」而建 = 过度设计。
- **裁决**：否决为主选。**当自动推进（主题 4 的推进 worker）真正落地时，推进 worker 自然承担队列角色**——届时 handler 保持同步「终态落定 + 置 idle」，推进由 worker 入队执行；本设计不提前建队列。

### 方案 C（否决）：handler 复用 Complete + 独立写 tasks（两个临界区）
- **结构**：`OnJobDone` 先锁内写 `tasks[id].Status`，再调 `Complete(name, jobID, ref)`（Complete 自己加锁置 idle + 写 ref）。
- 问题：①两个临界区拆散原子性（tasks 补存成功、置 idle 被 guard 挡——结果虽正确因为两操作本就解耦，但语义分散难验证）；②Complete 的 `ref` 参数是**死参数**（datamodel §4.3 已裁决 ref 更新必须走 Assign 解析路径，完成事件拿不到 `run.Ref`）→ 误导执行队去 observer 里找 ref；③Complete 置 idle 不区分「已 Idle 重复置」→ mailbox 唤醒无法绑定翻转。
- **裁决**：否决。OnJobDone **内联**单临界区（tasks + teammates 同锁原子）；Complete 保留为遗留 API（兼容测试/未来调用），**不被 OnJobDone 调用**。

**裁决：方案 A。** 与 datamodel-design §5.3 锁序总表、test-strategy D4/D5 完全一致；最小侵入、幂等点显式、为未来推进 worker 留好边界（handler 同步、推进异步，二者职责分离）。

---

## 三、主题详述（结论 → 事实链 → 设计裁决）

### 主题 1：同 jobID 重复触发完成事件？handler 幂等性

**结论：事件源恰好一次；handler 自幂等是硬契约（三层防御）。**

**事实链**：
1. `recordCompletion`（jobs.go:937）全仓**仅两处调用**：`startInvalid`（:479）与 run goroutine（:645），两者互斥（一个 Job 要么校验失败走 startInvalid、要么正常启动走 run goroutine）→ **每 jobID 至多一次终态事件**。
2. suppressEnvelope 分支（:963-978）与正常分支（:980-1016）是 if/else 互斥 → silent/foreground job 也**只触发一次** recorder/observer。
3. `loaded tombstone` 恢复路径（:1963-1981）**不调 recordCompletion** → 不产生事件（对 TeammateStore 无影响：teammates/tasks 全内存，重启即空）。
4. `Complete`（teammate_store.go:302-316）的既有幂等范式：找不到 teammate → return；`LastJobID != jobID` → return；匹配 → 置 idle。**但重复事件第二次仍会再次置 Idle + 写 ref**（LastJobID 不清空 → 条件仍成立）——"无害但非 no-op"，handler 需比它更严。

**设计裁决**（handler 三层幂等）：
- **① tasks 终态不可变**：`tasks[id].Status` 为空或 `"running"` 才写终态；已有终态 → 跳过写入（防 R-g 终态回退）。
- **② 置 idle 仅当翻转发生**：`tm.LastJobID == id && tm.State == TeammateRunning` 才置 Idle；重复事件第二次时 State 已 Idle → 条件不满足 → no-op。
- **③ mailbox 唤醒绑定翻转**：仅当 ② 的翻转实际发生时（`flipped`）才查 backlog + notifyMail——否则重复事件导致重复通知（R-b）。
- 防御对象：测试路径手动注入 + 真实事件双达（同实例重复）；未来 jobs 层多订阅者/重放演进；懒同步 `syncStateLocked`（List 先置 idle、事件后到 → ② 的 Running 条件挡住）。

### 主题 2：teammate 在 job 完成前被 Remove → handler 找不到 teammate

**结论：找不到 teammate 是正常路径，no-op；tasks 过滤优先，owner 缺失仅跳过置 idle。**

**事实链**：
1. `Remove`（:346-362）：锁内 `delete(ts.teammates, name)`（:354）→ 锁外 `jm.Kill(jobID)`（:358）。
2. `Kill`（jobs.go:1211-1233）：同步置 `j.status = Killed`（:1225）→ `j.cancel()`（:1231）→ run goroutine 的 `ctx.Err() != nil`（:594-595）→ `st = Killed` → `recordCompletion`（:645）→ observer 触发。
3. 所以 **Remove 必然在 job 完成前制造一个 Killed 完成事件**，而该事件到达时 teammate 已从 map 删除。
4. 反向竞争：完成事件先到（置 idle）→ Remove 后到 → Remove 看到 Idle 不 Kill（:329-331）或 `jm.Kill` 对已终态 job 返回 false（:1216-1230）→ 无害。
5. observer 是全 Manager 单例 → 事件覆盖所有非 teammate job（bash/fleet/parallel_tasks）→ **必须按 `tasks[id]` 过滤**（recordTask 只在 Assign 成功路径登记 :226，故 tasks 过滤 == teammate job 过滤）。

**设计裁决**：
- **第一行过滤**：`tasks[id]` 不存在 → return（非 teammate job，含所有其它 kind）。
- **owner 缺失**（tasks 存在但 teammates 无该 name）：跳过置 idle（无可置），tasks 终态补存照常写入——/team-status 仍展示已删成员的 job 终态？取决于 datamodel P3「Remove 清 tasks」是否落地：**若 Remove 同步清 tasks → handler 全 no-op（tasks[id] 也没了）；若不清 → 终态补存无害**。两者都安全，执行队按 datamodel P3 取舍。
- **handler 不得在找不到 teammate 时 panic / 报错**——这是 Remove 的正常生命周期收尾，不是异常。

### 主题 3：完成时 job 是 killed（/team-stop 杀）→ 依赖自动推进继续还是停？被杀的 job 算「完成」吗

**结论：killed 是终态——依赖门放行、teammate 置 idle、自动推进不触发；「被杀」在完成事件意义上算「终态」而非「成功完成」。**

**事实链**：
1. `TeamStop`（:321-341）：锁内先置 Idle（:333）→ 锁外 `jm.KillForSession("", jobID)`（:337）→ Killed 事件随后到达。
2. `pendingDependenciesLocked` 终态集合（:255）：`"done","failed","killed","interrupted","cancelled"` —— **killed 已含**。
3. 若依赖门对 killed **不放行**：依赖该 job 的后继任务将**永久 blocked**（该 job 不会再产出任何进展）——死锁。
4. `syncStateLocked`（:130-138）与 `Complete` 都不区分 killed/failed/done——任何非 Running 即 Idle。

**设计裁决**（按 Status 分派，语义分层）：
- **tasks.Status 照写 "killed"**（终态不可变）→ 后续手动 Assign 的 `pendingDependenciesLocked` 放行 ✅（防依赖死锁）。
- **teammate 置 idle**：TeamStop 已置 → handler 的 `Running` 条件不满足 → no-op（幂等）；若事件先于 TeamStop 到达（用户先杀后查）→ handler 置 Idle → TeamStop 后到看到 Idle 不重复 Kill → 自洽。
- **自动推进不触发**（未来功能契约）：killed/failed/interrupted 只「落定终态 + 解锁依赖门」，**绝不自动启动后继**；只有 `done` 才触发推进。理由：①kill 是用户主动终止，自动重启 = 覆盖用户意图；②推进→再被杀→再推进 = 无限循环；③failed 自动重试需要 backoff/上限策略（risk-review G5），不在终态语义内。
- **「放行依赖门（手动可继续）」≠「自动推进（自动不继续）」**——必须显式写进 OnJobDone 注释与测试，防执行队把门禁放行误读为推进授权。

### 主题 4：完成 → 自动 Assign 下一个 → LastJobID 覆盖语义

**结论：单值 LastJobID + 严格匹配 guard 在连续任务下成立；tasks 补存与 teammate 置 idle 双层解耦是并发安全关键；推进必须 worker 化。**

**事实链**：
1. `Assign` 拒绝 running 中再派（:173-176）→ 同一时刻一个 teammate 至多一个活跃 job → `State==Running` 时 LastJobID 唯一代表活跃 job。
2. 覆盖时序：J1 完成（LastJobID=J1 保留）→ 推进/手动 Assign J2（LastJobID=J2）→ **J1 事件迟到** → handler 查 `LastJobID==J2≠J1` → guard 挡住 ✅（与 Complete :309-311 同一语义）。
3. `recordTask` 以 **jobID 为 key**（:237）→ tasks 条目天然按 job 唯一，与 teammate 的 LastJobID 覆盖**互不干扰**：J1 的终态照写 tasks[J1]，J2 是独立条目。
4. `LastJobID` 置 idle 后**保留不清空**（datamodel §4.2）：TeamStop/Remove 对 Idle teammate 的 Kill 或对已终态 job 的 Kill 均返回 false 无害（:329-331 / :1216-1230）。

**设计裁决**：
- **双层解耦**：handler 的「tasks 终态补存」（按 jobID，天然幂等）与「teammate 置 idle」（按 LastJobID 严格匹配）在同一临界区内原子完成，但**互不依赖**——旧 job 事件永远只能写自己的 tasks 条目、永远无法误置新 job 占槽的 teammate。
- **推进 worker 化（硬约束）**：OnJobDone **禁同步 Assign**（datamodel 三禁）。推进 worker 从 ts.mu 外读 `tasks` 状态 → 对 `Status==done` 的后继任务重放 Assign → 若 teammate 已被用户手动占用（`is running` 拒绝）→ 任务置 blocked + backoff（datamodel §2.3），**绝不抢占**。handler 与推进 worker 之间只通过 `tasks.Status` 通信，无共享可变状态。
- **LastJobID 语义演进文档化**：「当前 job」→「最近 job」，活跃性由 `State` 表达；guard 是唯一正确性闸门，任何绕过 guard 直接按 name 置 idle 的实现都是 bug。

### 主题 5：Assign 的 ctx 生命周期——完成回调 ctx 从哪拿

**结论：handler 零 ctx 依赖；自动推进所需 ctx 存 session id 字符串重建，绝不缓存 Assign 的 ctx 对象。**

**事实链**：
1. jobs 层回调签名（jobsevent D2 struct `JobCompletion` / implementation-plan T1 六参）**均不携带 ctx**。
2. Assign 的 ctx（:165/:192/:207）是 leader 的 **turn 上下文**（controller 调 applyTeamCommand 传入），生命周期 = 该 turn；job 完成在跨 turn 后台（jobs.go 注释「a job started in one turn keeps running across turns」）→ **完成时该 ctx 早已 cancel/销毁**。
3. 完成回调跑在 run goroutine（jobs.go:585-657），其 jobCtx 由 `context.WithCancel(m.root)` 创建（:537）——但 observer 不接收它。
4. `flushMailbox` 在 Assign 内用 `jobs.SessionFromContext(ctx)`（:225）——这是 Assign 时取 session，与完成回调无关；**但它是 `TeamTask.SessionID` 补存的现成来源**。

**设计裁决**：
- **handler 本体零 ctx**：内存操作 + `os.ReadDir` + `sink.Emit` 均不需要 ctx（sink 已在 SetSink 时持有）。
- **自动推进的 ctx 重建**：`TeamTask` 补存 `SessionID string`（datamodel §1.2）→ 推进 worker 用 `context.Background()` + `jobs.WithManager` + `jobs.WithSession(sessionID)` + `WithParentSession` + `WithForkSource` 重建 Assign 所需上下文。**存 id 不存 ctx 对象**：ctx 是接口、不可序列化、可能带 deadline/values，跨 job 生命周期即失效（R-d）。
- **session 已关闭的处理**：destroying 主路径已被 jobs 层挡掉（:982-985 提前 return，事件根本不触发）；残留竞态窗口（destroy 始于 recordCompletion 之后）→ 推进 Assign 到销毁中的 session → **降级为 slog.Warn + 丢弃**，不阻塞 destroy、不重试（MVP）。MVP 无推进 worker，此窗口仅文档化。
- **防御注释**：OnJobDone 签名处明示「调用方（jobs 层）不提供 ctx；本函数不得依赖任何调用链 ctx」。

### 主题 6：handler 内部错误处理

**结论：slog 分层记录为主；不新增完成类 Notice；可选段 defer recover；Killed/Failed 的可见性复用 jobs 层既有 Notice。**

**事实链**：
1. handler 潜在错误点：内存操作（无 error 返回）；`os.ReadDir(inbox)`（IO 可失败）；`sink.Emit`（接口无 error 返回，实现可能阻塞/panic，event.go:827-829）。
2. jobs 层已对 observer 做 per-observer recover（jobsevent D7）→ handler panic 不打断管线，但 panic 时状态可能写了一半。
3. `recordCompletion` 已发 closing Notice（:1004-1015）：Failed→LevelWarn、Killed→info → **用户已能看到 teammate job 的失败/被杀**，handler 再 Emit 是双重通知。
4. risk-review §2.5：完成驱动的 fan-out Notice 无**限流**（maxResultsPerDrain=8 / maxResultBlockBytes=16KB 只限信封，jobs.go:229-233）→ 大量完成类 Notice 刷屏风险。

**设计裁决**：
- **slog 分层**：正常翻转 → `slog.Debug`；ReadDir 失败 → `slog.Warn("team mailbox check failed", "name", ..., "err", ...)`；tasks 过滤未命中 → `slog.Debug`（预期高频，非异常）。
- **不新增完成类 Notice**：handler 不 Emit 「job done / failed / killed」类通知——P1 信封（结果）+ jobs 层 closing Notice（状态）已覆盖；mailbox 唤醒 notice（`notifyMail`，P6.2）是**唯一例外**（有积压才发，且绑定翻转）。
- **mailbox 段 defer recover**：`countInbox` + `notifyMail` 包独立 defer recover（slog.Error），防可选增强变管线事故；失败静默降级——下次 Assign 的 `flushMailbox`（:430-457）兜底清 inbox。
- **绝不返回 error / 绝不 panic 出 handler**：OnJobDone 签名无 error 返回（对齐 TaskRecorder「best-effort」契约，jobsevent D 系列）。

---

## 四、任务分解（子任务 + 验证点）

- [ ] **L1 handler 契约骨架**：`OnJobDone` 定义——`st==Running` 早退；`tasks[id]` 过滤先行；终态分派按 `st` 写 tasks + 翻转 teammate；三禁注释（禁 jm.* / 禁锁内 IO / 禁同步 Assign）。→ 验证: `TestOnJobDoneNonTaskJobNoop`（未登记 id 注入 → 零副作用）；`TestOnJobDoneRunningNoop`（st=Running → 零副作用）
- [ ] **L2 单临界区原子落定**：同锁内 tasks 终态补存（终态不可变）+ 严格匹配置 idle（`LastJobID==id && State==Running`）+ 记 `flipped`。→ 验证: `TestOnJobDoneFiresIdle`（注入 Done → Status(name).State==Idle，不调 List）；`TestOnJobDoneStaleJobNoop`（LastJobID=J2、事件 J1 → 不误置）；`TestOnJobDoneIdempotent`（同事件注入两次 → 状态不变、无重复副作用）
- [ ] **L3 Remove/TeamStop 竞争路径**：Remove 后事件 no-op；TeamStop 杀后 Idle 保持；事件先于 TeamStop 的乱序。→ 验证: `TestOnJobDoneAfterRemoveNoop`（Remove 后注入 → 无 panic、tasks 终态或全无）；`TestOnJobDoneKilledKeepsStoppedIdle`（TeamStop 后注入 Killed → State 仍 Idle）
- [ ] **L4 killed/failed 语义固化**：tasks 写 killed 终态 + 依赖门放行 + 不触发推进（推进范围外，但断言「killed 后手动 Assign 依赖放行」）。→ 验证: `TestOnJobDoneKilledUnblocksDependency`（A 被杀 → B 带 A 依赖的 Assign 通过）；`TestOnJobDoneKilledWritesTerminalStatus`（tasks[A].Status=="killed"）
- [ ] **L5 ctx 零依赖 + SessionID 预留**：OnJobDone 无 ctx 参数使用（编译期保证）；`recordTask` 补存 `SessionID`（`jobs.SessionFromContext(ctx)`，:225 现成来源）+ `Status:"running"` 初值。→ 验证: `TestRecordTaskPersistsSessionAndRunning`（Assign 后 tasks 条目含 SessionID/Status=="running"）；`TestOnJobDoneDoesNotNeedCtx`（裸 `context.Background()` 之外的调用方式注入，无依赖）
- [ ] **L6 mailbox 唤醒绑定翻转 + 错误处理**：`flipped && inboxRoot!=""` → 锁外 countInbox（容错）→ 有积压 → notifyMail（defer recover）；slog 分层；不新增完成类 Notice。→ 验证: `TestOnJobDoneMailboxWakeupOnce`（重复事件 → notice 恰一次）；`TestOnJobDoneMailboxReadDirFailSilent`（inbox 不可读 → slog.Warn、无 panic、job 收尾正常）
- [ ] **L7 接线与文档闭环**：boot.go `jm.SetJobDoneObserver(ts.OnJobDone)` + `ts.SetSink(sink)`；本设计结论落入 implementation-plan T2 的 OnJobDone 步骤（若有冲突以本设计裁决为准，冲突清单交纪律团）。→ 验证: `go test ./internal/jobs/ ./internal/agent/ -race` 零失败；`go build ./...`

**依赖关系**：L1 → L2 → L3/L4（并行）→ L5（可先于 L2 起草）→ L6（依赖 L2 的 flipped）→ L7。阻塞项：L2 的集成断言依赖 jobs 层 observer（jobsevent T1）合入。

---

## 五、对抗自检（devil's advocate）

1. **攻击：「事件源已恰好一次，幂等是多余防御」**——回应：三层防御合计约 10 行，成本极低；防御的是**真实存在**的三条路径（测试注入 + 懒同步双路径 + jobs 层未来多订阅者/重放演进）。且幂等不是为「重复」而写，是为「乱序 + 双状态源」而写（R3/R4/R-g）。
2. **攻击：「Remove 后 tasks 终态补存=给已删成员写墓碑，语义脏」**——回应：取决于 datamodel P3 是否落地「Remove 清 tasks」；不清则墓碑无害（/team-status 显示已删成员的任务终态反而是审计价值），清了则全 no-op。**两分支都安全**是本设计的正确性边界；执行队按 datamodel 取舍即可。
3. **攻击：「killed 放行依赖门，leader 误以为后继任务可以正常跑」**——回应：放行 ≠ 成功；后继任务启动时读到的是「前置任务被杀了」——这是**显式信息**而非静默丢失（Tasks() 展示 killed；P1 信封与 closing Notice 均告知）。不放行的代价是**依赖永久死锁**，更糟。
4. **攻击：「翻转绑定 mailbox 唤醒，但翻转后 backlog 又新增的 mail 不通知」**——回应：PostMail（:368-397）每次落盘后已即时 notifyMail——**翻转后新 mail 由 PostMail 自己的通知覆盖**，handler 只需管「完成瞬间的积压」。语义无洞。
5. **攻击：「自动推进 worker 与手动 Assign 并发抢占同一 teammate」**——回应：推进 worker 的 Assign 会被 `is running` 拒绝（:173-176）→ 任务置 blocked + backoff（datamodel §2.3）→ 手动操作永远优先，推进永不抢占；**「绝不抢占」写进 worker 契约**。
6. **攻击：「handler 锁内遍历 teammates 是 O(n)，n 大时拖慢 run goroutine」**——回应：团队规模个位数（/team-create 手建），O(n) 微秒级；jobs 层同步回调本就要求「快速返回」（jobsevent D5）；若未来团队规模化，优化点为 LastJobID 反向索引（datamodel §1.2 已标注，不在本次范围）。

**薄弱环节标注**：① L6 的 mailbox 段「defer recover」若实现时漏包 → panic 虽被 jobs 层 recover 接住，但状态已写一半（tasks 终态已写、flipped 已记）——翻转与 notify 之间无事务性，属可接受的不一致（mailbox 通知丢失可由下次 Assign flush 兜底）；② L4「自动推进不触发」是**未来契约**，当前代码无推进逻辑，测试只能断言「无推进行为」而无法断言「推进被抑制」——需以文档 + 注释固化，防后续实现者误解；③ L5 的 session 已关闭竞态窗口无测试覆盖（destroy 时序难构造）——标注为已知未测边界，靠文档约束。

---

## 六、缓存/纪律检查点（Reasonix 领域）

- ✅ **发送侧字节零变化**：handler 全部改动在 `teammate_store.go` 内存态结构 + `recordTask` 补存 + jobs 层既有 observer 机制；不触碰 schema/system prompt/transcript/task.go/subagent_store 发送路径；不新增任何完成类 Notice（mailbox 唤醒是既有 P6.2 通知模式的复用，非新增发送侧注入）。
- ✅ **前缀稳定**：ref 更新**不走完成事件**（datamodel §4.3：observer 无 `run.Ref`，teammate 侧拿不到）→ OnJobDone 不触碰 fork/continue 三要素；续轮 continue 语义由 Assign 解析路径（datamodel §4.3 修复项）单独承担，与 handler 解耦。
- ✅ **运行值不进 schema/system prompt**：tasks.Status / SessionID / flipped 均为内存态登记数据，非发送前缀一部分。
- ✅ **锁序纪律**：OnJobDone 从 jm 无锁点进入（jobsevent D4）→ ts.mu 单临界区（禁 jm.* / 禁 IO / 禁同步 Assign）→ 锁外 mailbox/日志；与 datamodel §5.3 锁序总表逐格一致。
- ✅ **防虚假完成**：本设计只产出评审结论与任务分解，不宣称任何实现完成；是否采纳与落地由执行小队按 implementation-plan 决定，纪律团独立审查；每个子任务的「完成」= 对应测试断言通过 + 四验证 + execution.md 证据链。
