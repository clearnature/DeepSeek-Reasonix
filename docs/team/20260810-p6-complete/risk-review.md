# P6 完成事件驱动「风险审查」

> 角色：**规划 Planner** 对抗自检交付物（fable5 Step 8，devil's advocate 攻击）
> 事务：`docs/team/20260810-p6-complete/` · 状态：只读审查，**零代码变更**
> 审查对象：team 完成事件驱动「风险审查」子机制（独立对抗性视角）
> 代码基线：`teammate_store.go`（syncStateLocked:130 / Complete:302 / Assign:165 / recordTask:234 / pendingDependenciesLocked:243 / PostMail:368 / flushMailbox:430）、`internal/jobs/jobs.go`（recordCompletion:937 / WithJobStartObserver:295 / WithTaskRecorder:310）、`internal/control/input.go:188-195`、`internal/agent/task.go:792-1089`、`internal/event/{sync,coalesce}.go`

---

## 0. 结论摘要

| # | 攻击面 | 风险等级 | 一句话结论 |
|---|--------|---------|-----------|
| 1 | 完成事件在 run goroutine 内触发，handler 重活阻塞 job 收尾 | **高** | P1 信封不延迟（已入队），但 `j.done`/`j.status` 收尾延迟 → Wait/claim/teardown/下一 job 启动全阻塞 |
| 2 | 依赖自动推进无限递归/环 | **高** | `pendingDependenciesLocked` 只查已完成；环 A↔B 死等无报警；失败也被当终态推进 → 雪崩 |
| 3 | 自动 Assign 失败后的状态残留 | **中**（自动推进场景下**高**） | 手动 Assign 回滚完备；自动推进失败既可能"未登记静默丢失"也可能"登记了永不执行" |
| 4 | 完成事件 vs 惰性 syncStateLocked 双写 State | **中** | `LastJobID` guard 挡住多数竞态；但双状态源 + **Ref 更新断点**（见 §3）是设计脆弱点 |
| 5 | 回调里调 Assign 再进 jobs 的重入锁 | **高**（位置敏感） | 挂在 RecordDone 同层（m.mu 外）**安全**；挂在 m.mu 内或 sink 同步路径 → 死锁 |
| 6 | mailbox Notice 风暴 | **中** | P1 信封已有 drain 限流；新推进/完成 Notice 无限流 → fan-out 刷屏 |
| 7 | 缓存红线：新发送内容改变前缀字节 | **高**（架构级） | 注入 leader input 的内容被 model 消费后进 committed history → 后续 fork 前缀碎片化；sink Notice 不进 transcript 则安全 |

**十大必须护栏（G1-G10，代码层见 §4）**：
- G1 完成事件异步分发（有界队列 + 单 worker + 快照）
- G2 回调挂 `m.mu` 之外（RecordDone 同层），禁止 sink 同步回调/锁内调用
- G3 回调装配 leader 模板 ctx（`WithManager`+`WithSession`+`WithParentSession`），禁用 jobCtx
- G4 环检测 + 防重复推进 + 失败策略显式 + 依赖链深度上限
- G5 推进失败可见（pending intent / blocked 状态 / 重试），Tasks() 永不悬空
- G6 Ref 更新原子化（State+Ref 同临界区），修复"Complete 无人调用"断点
- G7 **零新发送原则**（MVP）：不注入 input、不发 sink Notice；进度靠 /team-status + 既有 P1 信封
- G8 只处理已登记 teammate jobID（过滤 silent/foreground job 的 RecordDone 触发）
- G9 幂等 + `LastJobID` guard 保留
- G10 `-race` 并发测试 + 环 e2e + 前缀 byte 断言 + 风暴压力测试

---

## 1. 拓扑扫描（文件依赖 + 级联风险）

```
teammate_store.go (TeammateStore)
  ├─ Assign:165 → RunProfileSpec(task.go:792) → StartForSession(jobs.go:485) → m.mu
  ├─ pendingDependenciesLocked:243 → jm.Output(dep)（实时状态派生）
  ├─ Tasks:266 → jm.Output(id) 派生 Status（惰性、依赖 jm 存活）
  ├─ syncStateLocked:130 → jm.Output(LastJobID)（惰性 Idle 翻转，不更新 Ref）
  ├─ Complete:302 → 更新 State+Ref（**当前全库无调用点**）
  ├─ PostMail:368 → notifyMail:402 → sink.Emit（P6.2 既有 Notice）
  └─ flushMailbox:430 → jm.SendMessageForSession（写 teammate P3 队列）

jobs.go (Manager)
  ├─ recordCompletion:937（run goroutine 内调用 @ startForSession goroutine:645）
  │    ├─ j.mu 快照 → m.mu append m.completed → RecordDone（无锁）→ sink.Emit（无锁）
  │    ├─ ⚠ droppedMsgs>0 时 sink.Emit 在 m.mu 内（988 行）
  │    └─ ⚠ suppressEnvelope 分支（963-978）也调 RecordDone（silent/foreground job 同样触发）
  ├─ StartForSession:485 → m.mu 短临界 → onJobStart:570（同步，文档要求快速返回）
  ├─ DrainCompletedNoteForSession:1468 → m.mu；限流 maxResultsPerDrain=8 / 16KB
  └─ TaskRecorder.RecordDone（1000-1002，m.mu 外）——**天然回调挂点**

input.go:188-195 → <background-jobs> 注入 leader 下一轮 input（model 消费后进 committed history）
event.Sync:29 → 同步串行 Emit（syncSink.mu）
event.Coalesce:67 → 流式增量聚合（防风暴现成工具）
```

**级联风险清单**：
1. 完成事件 → 回调 → Assign → fork/续轮 → 影响 leader/teammate 前缀字节（缓存红线，§2.7）
2. 完成事件 → 回调 → PostMail → sink Notice → leader UI（风暴，§2.6）
3. 完成事件 → 回调 → jm 方法 → 与 `m.mu→sink.Emit` 既有锁序相交（§2.5）
4. 完成事件 → Ref 更新 → 续轮 fork_first 判定（§3 断点）
5. P5 silent fork / foreground job 完成也触发 RecordDone → 误推进（§3）

---

## 2. 攻击面逐项

### 2.1 完成事件在 run goroutine 内触发 → handler 重活阻塞 job 收尾 —— **高**

**事实链**（jobs.go:645-656）：`m.recordCompletion(...)` 在 run goroutine 内调用，其返回后才有 `j.status=st`（648-650）、artifact 清理（651-654）、`close(j.done)`（656）。`recordCompletion` 内 `m.completed` 的 append（990-995）发生在回调之前 → **P1 信封本身不延迟**（下一轮 drain 已就绪）。

但回调若同步做重活（`Assign` 的 fork 分支会同步执行 `prepareTranscriptForkWithPrompt`（task.go:873）= 捕获 leader 前缀 + prefill teammate 会话 + transcript 磁盘写；`PostMail` = 磁盘 mkdir+write），则：
- `j.done` 关闭延迟 → `Wait(nil)`、`ClaimForegroundResult`（jobs.go:1589，阻塞在 `<-j.done`）、`onJobStart` 注册的 done 消费者（delivery workspace 写租约释放）全部延迟；
- run goroutine 收尾延迟 → `m.wg` 归零延迟 → session teardown 变慢；
- 链式推进时（A 完成→同步 Assign B）→ B 的启动被串行拖进 A 的收尾 → 延迟级联。

**缓解**（G1）：
1. 回调只做**轻量登记**：把 `(jobID, st, ref)` 快照 append 进 teammate_store 的有界 channel/队列（O(1)，无锁或短临界），立即返回。
2. 独立 worker goroutine（TeammateStore 自持，`DestroyAll` 时排空）消费队列，串行执行 Complete/Assign/PostMail。
3. 回调文档契约（照抄 WithJobStartObserver 的 "must return quickly"，jobs.go:295）：禁止同步 Assign、禁止同步 PostMail、禁止同步 sink.Emit 重文本、禁止磁盘 IO。
4. 队列有界（如 128）→ 满则丢弃并记日志（或合并为一条积压 Notice），绝不阻塞 run goroutine。

### 2.2 依赖自动推进无限递归/环 —— **高**

**事实链**：`pendingDependenciesLocked`（243-262）只查 `ts.tasks[dep].Status` + `jm.Output(dep)` 实时状态，只认 `done/failed/killed/interrupted/cancelled` 为终态。无环检测、无超时。

- **环死等**：A 依赖 B、B 依赖 A，均未完成 → 两者 gate 都拒绝 → 永不启动 → 永久死等，无任何报警（`Tasks()` 只能看到 running/空状态）。
- **失败也推进**：`failed/killed/interrupted/cancelled` 全部 `continue`（255 行）→ A 失败后自动推进 B → 错误雪崩。MVP 若延续此语义必须**显式**（配置化 fail-fast vs continue），不能默认。
- **链式同步递归**：若自动推进在回调 goroutine 内用 while 同步推进整条链 → 阻塞深度 = 链长（回到 2.1）。
- **登记即悬空**：环中的任务即使被登记，因依赖永不满足，`Tasks()` 长期显示 waiting/running。

**缓解**（G4）：
1. **环检测**：`Assign`/`recordTask` 登记 dependsOn 时，构建有向依赖图（jobID → dependsOn[]），对"新边"做 DFS 闭包环检测，命中则拒绝登记并返回明确错误（"dependency cycle: A → B → A"）。
2. **反向索引 + 防重复推进**：完成事件触发时查"依赖该 job 的后继集合"（新增 `dependents map[string][]string`，recordTask 时双向登记）；每个后继打 `advanced` 标记（幂等，防重复 Assign 竞态与双 worker 重复触发）。
3. **失败策略显式**：默认 fail-fast（依赖 failed → 后继标记 `blocked(failed dep)`，不启动）；continue 模式必须 Notice 说明。两者都在推进处显式判定。
4. **深度上限**：依赖链最大深度（如 32）超限拒绝登记；推进层不做同步链式 while（每个后继独立入队）。
5. **死等护栏**：`/team-status` 显示 `waiting_on` 原因（复用 `pendingDependenciesLocked` 的 reason 文本）；可选 `/team-retry <jobID>` 手动重推。

### 2.3 自动 Assign 失败后的状态残留 —— **中**（自动推进场景**高**）

**事实链**：手动 Assign 的回滚完备——依赖 gate 拒绝（180-184）不动状态；`RunProfileSpec` err（208-212）回滚 Idle；jobID==""（219-221）置 Idle 且不登记。**但自动推进**没有这条回滚链的可见出口：
- 场景 A（未登记丢失）：推进 worker 调 `Assign(B dependsOn A)` → B 的 teammate 恰好被并发 `/team-add` 占用（running）→ gate 拒绝 → B 未登记 → 依赖 B 的 C 永久等待，C 在 Tasks() 显示 waiting → **悬空**。
- 场景 B（登记无 job）：若 worker"先登记后 Assign"（先 recordTask 再 Assign）→ Assign 失败 → B 已登记但无 job → Tasks() 派生不出状态 → 显示过时值 → **悬空**。
- 场景 C（静默假成功）：`teammateJobID`（288）解析不出 jobID 时返回 `("", nil)`——自动推进会当成功处理，任务既未执行也无错误。

**缓解**（G5）：
1. 推进 = 两步显式：**登记 pending intent**（`recordTask` 时 Status="pending"）→ 尝试 `Assign` → 成功则 intent 升级为 job（Status 由回调/派生接管）；失败则 intent 置 `blocked + reason`（gate 拒绝原因 / Assign err）。
2. `Tasks()` 派生规则扩展：`pending` → 显示 `queued`；`blocked` → 显示 reason；`jm.Output` 不可用（manager 已关）时回退到主动写入的终态而非旧派生值。
3. **jobID=="" 且 err==nil 必须改为返回错误**（`Assign` 229 行签名层），消灭静默假成功。
4. 失败入队重试（有界 backoff，如 3 次 × 1s/5s/30s）+ 最终失败写 blocked + 一条聚合 Notice（低频，见 2.6）。
5. 单 worker 串行消费 → 推进之间无并发残留竞态（用户手动 Assign 与 worker 推进共用 ts.mu，互斥安全）。

### 2.4 完成事件 vs 惰性 syncStateLocked 双写 State 竞态 —— **中**

**事实链**：`syncStateLocked`（131-138，List 惰性路径）只翻转 `State`；`Complete`（302-316）翻转 `State`+`Ref` 且带 `LastJobID != jobID` guard。两者都在 ts.mu 下，**锁互斥安全**。逐交错推演：
- 完成 →（回调延迟）→ List() 惰性置 Idle → 用户 Assign（新 job 占槽，LastJobID=新）→ 回调 Complete(旧 jobID) → guard 挡住 → **安全**。
- 回调先置 Idle+Ref → 用户 Assign 用新 Ref → continue → **安全**。
- 回调与新 Assign 并发 → ts.mu 串行化 → **安全**。

**残余风险**：双状态源并存（惰性派生 vs 主动写入）。若完成事件不更新 Ref（§3 断点），惰性路径也补不了 Ref → 续轮永远 fork_first。**State 谁写无所谓（幂等），Ref 必须由完成事件原子写**。

**缓解**（G6）：
1. `Complete` 保持幂等，但**必须被调用**：完成事件 worker 对登记过的 teammate jobID 调 `Complete(name, jobID, ref)`（ts.mu 内原子更新 State+Ref）。
2. `syncStateLocked` 保持为"无事件机制时的兜底"（不写 Ref），避免双源写 Ref。
3. 回调携带的 `ref` 必须是 job 完成时的 transcript ref 快照（来源见 §3），出队时已固定，防 worker 延迟期间 ref 漂移。

### 2.5 completion 回调里调 Assign 再进 jobs —— jobs 层重入锁 —— **高**（位置敏感）

**事实核查**（关键）：
- `recordCompletion` 的 `m.mu` 临界区是 981-998；`taskRecorder.RecordDone` 在 **1000-1002，m.mu 之外**。→ 回调挂在 RecordDone 同层时，回调内 `Assign → RunProfileSpec → StartForSession → m.mu.Lock()` **不重入，安全**。
- 三个陷阱：
  1. **回调挂在 m.mu 内**（如 981-998 之间）→ `StartForSession` 的 `m.mu.Lock` 重入 → Go `sync.Mutex` 直接死锁。**执行小队最可能的省事写法，必须禁止**。
  2. **完成事件走 sink 同步回调**：`recordCompletion` 988 行（droppedMsgs>0）在 m.mu 内 `sink.Emit` → sink 消费者若同步回调进 jm/teammate_store → `m.mu → syncSink.mu → (consumer) → ts.mu/m.mu`；而 Assign 路径 `ts.mu →(释放)→ m.mu →(释放)→ syncSink.mu` → 两序相交，潜在死锁环。**完成事件绝不能实现为"sink 收到 Notice 再回调"**。
  3. **ctx 缺失**：run goroutine 的 jobCtx 只有 `WithSession`+jobCtxKey（jobs.go:558-559），**没有 `jobs.WithManager`** → 回调里直接调 Assign → RunProfileSpec 941 行 `jobs.FromContext(ctx)` 失败 → "background execution is not available"。回调必须用装配好的 ctx。

**缓解**（G2+G3）：
1. jobs 层新增 `WithJobDoneObserver(observer func(id, st Status, err error))`（仿 `WithJobStartObserver`，调用点在 recordCompletion 的 RecordDone 同层、m.mu 外），文档三禁：禁锁内调用、禁同步 Assign、禁重活。
2. TeammateStore 构造时保存 leader 模板 ctx（`jobs.WithManager + jobs.WithSession(leader) + WithParentSession(leader)`，Controller 在 `/team-add` 处注入，controller.go:6370 先例）；回调 worker 用它调 Assign。manager 与当前 jm 同一实例 → 锁无嵌套。
3. **锁序总表强制写入执行手册**：`ts.mu` 短临界独立 → `m.mu` 短临界独立 → `syncSink.mu` 独立；任何新增路径不得在持 A 锁时获取 B 锁（各自释放后再取）。
4. 完成事件快照化：回调入队时携带 `(jobID, st, ref)`，worker 处理时**不再读活 job 对象**（job 可能已被 kill/清理/manager 关闭）。

### 2.6 mailbox Notice 风暴 —— **中**

**事实链**：P1 信封已有限流（`maxResultsPerDrain=8`、`maxResultBlockBytes=16KB`，jobs.go:229-233 + DrainCompletedNoteForSession:1468）。但完成事件驱动的**新** Notice（推进成功/失败/完成通知）没有对应限流；`PostMail` 每次已 emit 一条（notifyMail:402-409）。fan-out 场景（1 完成 → 10 后继）→ 10 完成信封（限流，可接受）+ 10 推进 Notice（**无限流**）→ 刷屏。

**缓解**：
1. **首选（G7 零新发送）**：推进状态**不新增任何 Notice**——统一由 `/team-status`（惰性派生）+ 既有 `<background-jobs>`（P1 信封已含完成信息）承担。零新发送 = 零风暴。
2. 若产品要求可见性：聚合 Notice（"team: 3 完成 / 2 推进 / 1 失败"），复用 `event.Coalesce`（coalesce.go:28）或手动计数窗口；文本固定字节、无时间戳。
3. 推进**失败**必须可见但低频：一次一条 Notice（失败是稀有事件，不聚合防丢失）；`PostMail` 通知维持 P6.2 既有行为（用户驱动，预期内）。

### 2.7 缓存红线：完成事件驱动引入的任何新发送内容是否改变前缀字节 —— **高**（架构级）

**三条前缀路径**（P6 定稿，docs/team/20260810-p6-team/plan.md）：
- P-a：leader **committed history** = teammate 首次 fork 前缀来源（`captureForkPrefix → PrepareParentFork`，task.go:869-873）；
- P-b：leader **turn input 注入**（`<background-jobs>`，input.go:193）→ model 消费后进 committed history；
- P-c：teammate **自身 transcript** = 续轮前缀来源（`PrepareContinue`，tm.Ref）。

**逐发送字节审查**：
| 新发送内容 | 落点 | 前缀影响 |
|---|---|---|
| 新增 `<team-*>` 块注入 leader input | P-b | **危险**：每次完成都注入 → committed history 随完成事件增长 → 后续 fork 前缀每次不同 → 缓存碎片化、命中率崩塌 |
| 新增 sink Notice（notifyMail 模式） | 不进 transcript（UI 事件，render.go:196 operator audience 只控 UI 转发） | **安全**（前缀零影响） |
| 自动推进自动给 teammate 发消息（PostMail/steer） | P-c（teammate transcript） | **危险**：改变续轮前缀 → teammate 缓存 miss；且 PostMail 通知用户驱动（预期内），自动消息是意外字节 |
| 完成事件文本含 jobID/时间戳/动态计数 | 任何落点 | **危险**：jobID 每次不同 → 即使注入一次也令前缀含不稳定字节 |
| 既有 P1 信封（含 jobID） | P-b（已存在） | 既有行为，随用户轮次提交；完成事件驱动使完成更频繁 → committed 增长更快 → **链长必须设上限** |

**核心结论**：完成事件驱动若"每完成一通知"，等于让 leader 的 committed history 被后台事件驱动改版 → fork 前缀漂移 → **P6 缓存收益归零**。唯一安全设计是**零新发送**。

**缓解**（G7）：
1. **MVP 零新发送**：完成事件完全静默推进——不注入 input、不发 sink Notice、不自动写 teammate transcript。leader 感知靠既有 `<background-jobs>`（P1 信封）与 `/team-status`。缓存红线零接触。
2. 若后续要通知：只走 sink Notice（`NoticeAudienceOperator` 或默认），**永不注入 input**；文本无时间戳、无动态计数；聚合限流（2.6）。
3. 自动推进**禁止**自动 PostMail/steer 写入 teammate transcript；teammate 收到的自动输入仅 = 首次 fork 前缀（既有）。
4. 依赖链深度上限（G4-e）同时约束 committed 增长速率。

---

## 3. 额外发现（执行前必读）

### E1. `Complete` 全库无调用点 → 续轮 continue 语义当前未生效（**既有断点，完成事件驱动必须接管**）
- grep 全库：`TeammateStore.Complete` 仅定义（teammate_store.go:302），**无任何调用**。
- 后果：`tm.Ref` 永远保持 ""（Assign:187 取旧 ref、218 写回旧值）→ 第二次 `/team-add` 走 `fork_first` 分支（Assign:199-202）而非 `PrepareContinue` → teammate 重新 fork leader 前缀 → **续轮前缀语义错误 + 缓存 miss**。
- 完成事件驱动的**第一职责**就是补上这个断点：完成时原子更新 `State=Idle + Ref=<job 完成后的 transcript ref>`。ref 快照来源：task.go job 闭包内 `run.Ref`（SaveCompleted 后）；需在完成事件出队时从 job 快照/结果提取（T0 探查项）。

### E2. `suppressEnvelope` 分支也调 RecordDone → 完成回调会对 P5 silent fork / foreground job 触发
- jobs.go:963-978：silent/foreground job 走同一 `RecordDone`（975-977）→ 完成事件若不按 jobID 过滤，**P5 fork（silent）完成也会触发依赖推进**。
- 缓解（G8）：teammate_store 侧只处理 `ts.tasks` 已登记的 jobID（P5 fork / foreground 不登记 → 天然过滤）；未登记 jobID 的完成事件直接丢弃（仅记 debug 日志）。

### E3. jobCtx 无 `jobs.Manager` → 回调不能直接用 run goroutine 的 ctx
- jobs.go:558-559：jobCtx 仅 `WithSession`+jobCtxKey；`jobs.WithManager` 只在 controller/agent 入口注入（controller.go:6370 先例）。
- 缓解（G3）：回调 worker 用装配好的 leader 模板 ctx；禁止从 jobCtx 派生。

### E4. `recordStalled`（1018）的 sink.Emit 在 m.mu 外、`recordCompletion` 的 droppedMsgs Emit 在 m.mu 内——既有不对称
- 完成事件若新增任何在 m.mu 内的 Emit 路径即踩 2.5 陷阱 2。改动后必须 `go test -race ./internal/jobs/ ./internal/agent/` 全绿。

### E5. `Tasks()` 的状态权威源
- Tasks():266-284 完全依赖 `jm.Output(id)` 派生；manager 关闭/会话销毁后 `ok=false` → 回退 `t.Status`（最后一次派生值，可能过时）。
- 完成事件驱动建议：`ts.tasks[id].Status` 由回调主动写入终态（权威），`jm.Output` 仅作兜底派生 → 状态不依赖 jm 存活。

---

## 4. 必须的护栏清单（代码层，供执行小队落实）

| 护栏 | 代码层动作 | 验证 |
|---|---|---|
| G1 | jobs 层新增 `WithJobDoneObserver`（RecordDone 同层调用，快照 `(id,st,err)`）；teammate_store 有界队列（128）+ 单 worker + `DestroyAll` 排空 | 队列满不阻塞 run goroutine；`-race` |
| G2 | 回调位置 = m.mu 外；禁止 sink 同步回调路径承载完成事件 | `-race` 并发测试；死锁超时用例 |
| G3 | TeammateStore 持 leader 模板 ctx（WithManager+WithSession+WithParentSession）；回调用模板 ctx 调 Assign | 回调 Assign 成功 e2e |
| G4 | 环检测（dependsOn 闭包 DFS）+ 反向索引 dependents + 防重复 advanced 标记 + 失败策略显式（默认 fail-fast）+ 深度上限 32 | 环登记被拒；失败不推进；链长上限生效 |
| G5 | 推进两步：recordTask(Status=pending) → Assign → 成功升级/失败置 blocked+reason+backoff 重试（3 次）；jobID=="" 改为返回 error | Tasks() 无悬空（queued/blocked 有语义） |
| G6 | 完成事件调 `Complete(name,jobID,ref)`（ts.mu 内 State+Ref 原子）；syncStateLocked 不写 Ref；ref 快照出队时固定 | 续轮 PrepareContinue 生效；e2e 第二次 /team-add 非 fork_first |
| G7 | MVP 零新发送：不注入 input、不发 sink Notice、不自动 PostMail/steer | 前缀 byte-identical 断言（复用 cachehit_e2e 基建） |
| G8 | 只处理 `ts.tasks` 已登记 jobID；未登记丢弃 | P5 silent fork 完成不触发推进 |
| G9 | Complete/推进幂等 + `LastJobID` guard 保留 | 重复完成事件无副作用 |
| G10 | `-race` 全量 + 环 e2e + 前缀断言 + fan-out 风暴压力（100 完成） | 测试全绿 + 无 Notice 风暴 |

---

## 5. 给执行小队的验证点（检查清单）

1. **锁序**：`go test -race ./internal/jobs/ ./internal/agent/ ./internal/control/` 全绿；新增用例：完成回调中调 Assign（断言不 deadlock，2s 超时）。
2. **收尾不阻塞**：用例：回调 handler 故意 sleep 2s → 断言 `j.done` 关闭 <500ms（worker 化生效）；`Wait` 不延迟。
3. **环**：登记 A→B、B→A 被拒（错误含 cycle）；链 32 上限生效。
4. **失败残留**：推进 B 时 teammate 被并发占用 → B 显示 `blocked(<reason>)`，重试后可恢复；Tasks() 永不显示无语义状态。
5. **Ref 断点**：首次 /team-add → 完成 → 再次 /team-add → 断言走 ContinueFrom（非 Fork）；续轮前缀 byte 稳定（cache_hit>0）。
6. **前缀红线**：完成事件驱动全程（自动推进 5 级链）→ leader committed transcript 字节 == 无完成事件基线（零新发送生效）；teammate 续轮前缀 == 基线。
7. **风暴**：100 个依赖并行完成 → sink 收到的完成相关 Notice 数量有界（聚合/零发送）；`<background-jobs>` 不超 16KB（既有 drain 限流）。
8. **过滤**：P5 silent fork 完成 → 无推进动作、无 teammate 状态变化。

---

## 6. 对抗自检（devil's advocate 攻击本审查自身）

1. **"零新发送"是否过度保守？** 产品可能要求 leader 实时感知推进。反制：感知走既有 P1 信封（已完成即通知）+ /team-status（查询即见），"实时"需求本身与 P6 前缀稳定架构冲突——若强制，只允许 operator-audience Notice（不进 transcript），仍是 G7 子集。
2. **单 worker 是否引入队头阻塞？** 一个慢 Assign（fork prefill）会阻塞后续全部推进。反制：worker 内 Assign 已是"后台 job 立即返回"（RunProfileSpec RunInBackground 路径同步段仅 fork+prefill），耗时有限；可加每项超时（如 10s）与队列水位告警；MVP 可接受串行（正确性优先于吞吐）。
3. **环检测的 DFS 是否会误杀合法 DAG？** DAG 无环不误杀；只有真环被拒。若执行小队实现成"仅查直接父"会漏多跳环（A→B→C→A）——检查点明确要求**闭包 DFS**。
4. **`ref` 快照来源（E1）是最大实现不确定性**：若 recordCompletion 无法低成本携带 ref，退路 = worker 在 job 完成后从 transcript store 查 ref（`SubagentStore` 按 job/session 查）。需 T0 探查记录。
5. **G2 的"m.mu 外回调"依赖执行小队不把回调挪进临界区**：纪律团需代码审查确认 `WithJobDoneObserver` 调用点位于 recordCompletion 的 m.mu.Unlock 之后；加注释 + 测试桩（在回调内断言 m.mu 未被持有——不可行，改为在回调内调 StartForSession 的 race 用例）。
6. **本审查自身未覆盖**：完成事件与 `TeamStop`/`Remove`/`DestroyAll` 的交错（worker 处理时 teammate 已被移除）——补充：worker 处理前查 `ts.tasks`/`teammates` 存在性，缺失即丢弃（幂等，G9）。

---

*本文档为规划交付物；实现与验证由执行小队承担，纪律检查团复核（docs/team-org.md 三职能硬性分离）。*
