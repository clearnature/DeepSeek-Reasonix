# 设计：team 完成事件驱动（汇总设计，交付 executor/纪律团）

> 角色：**规划 Planner**（只读设计，零实现代码）
> 事务：`docs/team/20260810-p6-complete/` · 基线：`internal/jobs/jobs.go`（`recordCompletion` L937 / `WithJobStartObserver` L296 / `WithTaskRecorder` L309 / `SetTaskRecorder` L316 / `Output` L1107）、`internal/agent/teammate_store.go`（`syncStateLocked` L131 / `Assign` L165 / `recordTask` L234 / `pendingDependenciesLocked` L243 / `Complete` L302 / `PostMail` L368 / `notifyMail` L399 / `flushMailbox` L430）、`internal/control/controller.go`（L695 taskmonitor 占 TaskRecorder 单槽、L6345-6373 /team-add ctx 装配）、`internal/boot/boot.go`（L1930 生产接线缺口）、`internal/jobs/jobs_recorder_test.go`（TaskRecorder 测试范式）
> 已裁决路径 B（任务红线）：仿 `WithJobStartObserver` 加**多订阅者** `WithJobDoneObserver(func(id string, st jobs.Status, err error))`，在 `recordCompletion` 触发；**禁止抢 TaskRecorder 单槽**（controller.go:695）与 `WithJobStartObserver`（boot.go:517 workspaceLease）。
> 本文件是事务汇总裁决：整合已有 7 份文档，消解签名/语义冲突，给出唯一实现依据。

---

## 0. 结论先行（TL;DR）

| # | 决策点 | 结论 | 锚点 |
|---|--------|------|------|
| C0 | 完成事件签名 | **3 参** `func(id string, st jobs.Status, err error)`（任务裁决），**多订阅者**（切片存储，`m.mu` 保护） | 见 §2；与 jobsevent-design D2 的 struct 式 6 参**偏差**见 §8 |
| C1 | 触发点 | 私有 `notifyTerminal(id, st, err)`：**先 `taskRecorder.RecordDone` 后 observers**；挂 `recordCompletion` 两处无锁点（suppress 分支 L975-977、正常分支 L1000-1002），替换现有两处 RecordDone 调用 | jobs.go:975-977 / 1000-1002；锁序纪律见 §2.3 |
| C2 | 置 idle | `OnJobDone` 内 `ts.mu` 短临界：`tm.LastJobID == id && tm.State == Running` → `State=Idle`（**严格匹配防旧完成**，同 `Complete` L309-311 语义） | teammate_store.go:309-311 |
| C3 | 依赖推进 | 完成事件把**终态落定**到 `tasks[id].Status`（不可变）；再扫描 `DependsOn` 含 id 且 `Status∈{pending,queued}` 的等待任务 → 入队**自动 Assign worker**（有界队列+单 worker，**绝不同步 Assign**） | §3；risk-review G1/G4/G5 |
| C4 | recordTask 补存 | 新增 `JobID / Prompt / SessionID / CreatedAt / Status初值 / BlockedReason`：`Prompt`+`SessionID` 是自动重放 `Assign(ctx, owner, prompt)` 的**必需**补存（完成事件无用户输入） | §3.1；datamodel-design §1.2 |
| C5 | mailbox 唤醒 | `OnJobDone` 置 idle 后、锁外 `countInbox(name)` → N>0 → `notifyMailBacklog(name, N)`（Notice 走 event.Sink，不进 provider 输入） | §4；mailbox-design 方案甲 |
| C6 | 与惰性 syncStateLocked | **事件置 idle 为主、惰性兜底防时序**：`syncStateLocked` 保留，改读 `tasks[LastJobID].Status`（主）+ 新增 jobs 层**非消费** `Status(id)` 兜底；三处 status-only `jm.Output` 全部迁移（修消费语义 + purge 悬空 bug） | §5；datamodel-design §3、interaction-design C7 |
| C7 | 待纪律团仲裁项 | ① 3 参签名 vs jobsevent-design struct 式（本设计已裁决，列偏差表）；② Close 后仍触发（对齐 RecordDone）；③ destroy 窗口 suppress 分支仍触发（对齐 RecordDone 现状）；④ Assign 拒绝即登记等待（**行为变更**，`TestTeammateDependencyGate` 断言需更新） | §8 |

**一条总纲**：完成事件 =「终态参数直通 + 同步轻回调 + recorder 先行 + 消费者幂等」。`OnJobDone` **三禁**：禁调 `ts.jm.*`、禁持 `ts.mu` 做 IO、禁同步 Assign（自动推进必须 worker 化）。

---

## 1. 拓扑扫描

### 1.1 文件依赖图

```
internal/jobs/jobs.go
 ├─ Manager :192-221（m.mu 保护 jobs/order/completed/…；taskRecorder 单槽 :220）
 ├─ Status :49-56（running/done/failed/killed/interrupted）
 ├─ Option :260-261 · WithJobStartObserver :296（单例，boot.go:517 占用）· WithTaskRecorder :309 / SetTaskRecorder :316（单槽，controller.go:695 taskmonitor 占用）
 ├─ recordCompletion :937-1016（终态唯一出口；两处调用：startInvalid L479、run goroutine L645）
 │    ├─ j.mu 非消费快照 + 信封预渲染 :951-961（不消费 readOffset/resultRead）
 │    ├─ suppressEnvelope 分支 :963-978 → RecordDone :975-977 → return（⚠ 无 destroying 检查）
 │    ├─ destroying 窗口 :982-985 提前 return（不 append、不 RecordDone）
 │    ├─ m.mu 内 append m.completed :990-995（P1 信封入队点）
 │    └─ 正常分支 RecordDone :1000-1002（m.mu 已释放）→ closing Notice :1013-1015
 ├─ run goroutine 时序 :585-657（recordCompletion :645 → status 发布 :647-655 → close(j.done) :656）
 ├─ Output/OutputForSession :1107-1143（readOffset :1124-1127 + resultRead :1132-1135 —— 全仓唯一消费点）
 ├─ Close :2138-2176（cancel → run 以 Killed 收尾 → 仍走 recordCompletion）
 └─ 待新增：JobDoneObserver / WithJobDoneObserver / AddJobDoneObserver / SetJobDoneObserver / notifyTerminal / callDoneObservers / Status（非消费查询）

internal/agent/teammate_store.go
 ├─ Assign :165-231（依赖门 :179-185 拒绝式；RunProfileSpec :207；flushMailbox :225；recordTask :226；Fork:true,Silent:false :202 → 信封必达）
 ├─ recordTask :234-238（只存 ID/Owner/DependsOn，Status 恒空串 —— 需补存）
 ├─ pendingDependenciesLocked :243-262（锁内 jm.Output :250 —— 锁序 + 消费 + purge 悬空三问题）
 ├─ Tasks :265-284（jm.Output 派生 :272 + 只读方法内改写 :279）
 ├─ syncStateLocked :131-138（懒同步置 idle，jm.Output :135）
 ├─ Complete :302-316（无调用者；LastJobID 严格匹配 :309-311 —— 匹配语义复用）
 ├─ PostMail :368-397（落盘 + 无条件 notifyMail :395）· notifyMail :399-410（锁外 Emit）· flushMailbox :430-457
 ├─ Remove :346-362（只删 teammates，不清理 tasks —— P3 缺口）
 └─ DestroyAll :460-469（Remove 循环）

internal/control/controller.go
 ├─ :695 SetTaskRecorder(taskmonitor) —— TaskRecorder 单槽已占，不可抢
 ├─ :6345-6373 applyTeamCommand：/team-add ctx 装配（WithForkSource+WithManager+WithSession+WithParentSession）—— 自动 Assign 模板 ctx 来源
 └─ :6359 /team-add（mailbox 唤醒文案命令锚点）

internal/boot/boot.go :249（sink = event.Sync 包装）/ :1930（NewTeammateStore 内联、未 SetSink —— 生产 mailbox 唤醒缺口）

internal/jobs/jobs_recorder_test.go（recordingRecorder 范式 :13-38；Set 后装配 :101-116；nil no-op :118-128）
internal/jobs/jobs_silent_test.go :71-90（silent 仍触发 recorder —— 完成事件同构）
internal/agent/teammate_store_test.go（testTaskToolForTeam :17；TestTeammateDependencyGate :169-203 —— 待更新断言）
```

### 1.2 级联风险（设计必须覆盖）

- **R-a 时序陷阱**：observer 触发时 `j.status` 仍 Running、`j.done` 未关（recordCompletion L645 早于发布 L647-655 与 close L656）→ `OnJobDone` 只信 `st` 参数，**禁反查 Output**（会读到 Running + 消费 readOffset/resultRead）。
- **R-b 锁序**：`m.mu → j.mu` 从不嵌套（jobs.go 多处注释）。`notifyTerminal` 只能挂两处无锁点；observer 遍历**不得持任何 manager/job 锁**（快照锁内、调用锁外）。
- **R-c 单例覆盖**：若沿用单槽 Set，第二个消费者覆盖 TeammateStore → 完成事件静默失效。多订阅者切片根治。
- **R-d 死锁重入**：observer 同步跑在 run goroutine，回调内对同一 job 调 `m.Wait` → 等 `<-j.done` 而 close 在回调返回后 → 死锁。文档三禁 + J12 固化。
- **R-e 阻塞收尾**：observer 阻塞延迟 `close(j.done)`（L656）→ Wait/claim/workspace 租约全阻塞；`OnJobDone` 内**同步 Assign** 更是把 fork prefill 串进收尾 + 链式递归。→ 自动 Assign 必须 worker 化（C3）。
- **R-f 消费查询 + purge 悬空**：`Output` 是消费性接口；jobs map purge 后 `Output` 返回 ok=false → `pendingDependenciesLocked` 把已完成 dep 当未完成 → 依赖永久阻塞（**现存隐藏 bug**，datamodel-design §3.2）。→ tasks.Status 补存为主源 + 新增非消费 `Status(id)`。
- **R-g 双真源漂移**：信封快照（m.completed）vs `jm.Output` vs tasks.Status 三个时间点不一致 → 固定各自语义：信封非消费、Output 消费、补存终态不可变。
- **R-h 自动推进残留**：推进时 teammate 被占用 → gate 拒绝 → 等待任务必须置 `blocked+reason`（绝不静默悬空）；`jobID=="" && err==nil` 的静默假成功（`teammateJobID` L288-298 解析失败）需在推进路径显式判错。
- **R-i 环死等**：A→B、B→A 相互依赖 → 双方等待永不满足 → 登记时 DFS 环检测拒绝。
- **R-j mailbox 重复/风暴**：`PostMail` 已即时 notifyMail → 完成唤醒仅当 inbox 有**未 flush 新 mail**（flushMailbox 已清空 inbox，L430-457）才发，且每完成至多 1 条聚合 Notice。

---

## 2. 多路径推演

### 2.1 完成事件源：方案 A（采纳，路径 B 裁决落实）vs 备选

| | 方案 A（采纳）多订阅者同步回调 | 方案 B（否决）done-channel 分发 | 方案 C（否决）扩展 TaskRecorder |
|---|---|---|---|
| API | `WithJobDoneObserver(fn)`（Option 追加）+ `AddJobDoneObserver(fn)` + `SetJobDoneObserver(fn)`（清空+单一） | `WithJobDoneObserver(func(done <-chan struct{}))` 自起 goroutine | 给 RecordDone 加参 / 复用单槽 |
| 终态参数 | 直通 `(id, st, err)` | 拿不到 st/err，需二次查询 | 有，但抢 taskmonitor 单槽 |
| 复杂度 | 低-中（jobs +~35 行） | 中（goroutine 分发 + 错过窗口） | 低但**违反任务红线** |
| 风险 | 回调须快速返回（recover 兜底 + J12 固化） | `startInvalid` 的 done 已提前关闭有错过窗口；状态二次查询竞态 | 职责混淆、强制 taskmonitor 改动 |
| 裁决 | ✅ 与任务路径 B 一致，与 TaskRecorder 先例同构 | 否决（R-a/R-e 放大） | 否决（红线） |

### 2.2 自动推进：方案甲（采纳）worker 化队列 vs 方案乙（否决）同步链式

| | 方案甲（采纳）有界队列 + 单 worker | 方案乙（否决）回调内同步 Assign |
|---|---|---|
| 收尾阻塞 | 无（OnJobDone 只 O(1) 入队） | fork prefill 串进 close(j.done) → 级联 |
| 递归 | 无（worker 独立 goroutine） | 链式回调递归（A→B→C…深度=链长） |
| 锁序 | 三禁成立（OnJobDone 内零 jm 调用） | ts.mu 释放后 m.mu 获取虽不成环，但阻塞/递归不可接受 |
| 失败残留 | blocked+reason+重试可见 | 失败即吞 |
| 裁决 | ✅ risk-review G1/G5 | 否决（R-e/R-h） |

### 2.3 任务状态主源：方案 ①（采纳）tasks.Status 为主 vs 方案 ②（否决）jm.Output 优先

- ① tasks.Status 补存为主源：事件写入、终态不可变、O(1) 纯内存、purge 不悬空；jm 仅经**非消费 `Status(id)`** 兜底（`Status==""` 未登记时）。
- ② jm 优先（implementation-plan T2 原案）：`Output` 消费语义 + purge 悬空 + 锁内 jm 调用，三重硬伤。
- **裁决**：①。与 datamodel-design §3 一致；`pendingDependenciesLocked` 改纯内存读（消除 ts.mu→jm.mu 锁序面）。

---

## 3. 精确 API 与触发点（jobs 层）

### 3.1 签名（任务裁决：3 参 + 多订阅者）

```go
// JobDoneObserver 收到每个后台 job 的终态转移。observer 同步跑在 job 收尾
// goroutine（run goroutine L645，或 startInvalid 的调用者 L479），按注册顺序，
// 在 TaskRecorder.RecordDone 之后。实现必须：
//   - 快速返回（慢 observer 延迟 close(j.done) L656，阻塞 Wait）；
//   - 绝不调 m.Wait 等同一个 job（死锁：done 在回调返回后才关闭）；
//   - 绝不反查 m.Output/Status 推导终态（JobDoneObserver 的 st 是权威，
//     触发时 job 的 j.status 字段仍 Running，见 recordCompletion 调用序）；
//   - 幂等、并发安全。单个 observer panic 被隔离，不打穿 job 管线。
type JobDoneObserver func(id string, st jobs.Status, err error)

// WithJobDoneObserver 追加一个完成观察者（Option，可多次调用追加全部生效）。
// 与 WithTaskRecorder 相互独立，可同时使用。
func WithJobDoneObserver(observer JobDoneObserver) Option

// AddJobDoneObserver 构造后追加（运行时）；注册前已完成的 job 不补发。
func (m *Manager) AddJobDoneObserver(observer JobDoneObserver)

// SetJobDoneObserver 清空后设单一观察者（nil 清空全部）。对称 SetTaskRecorder L316。
func (m *Manager) SetJobDoneObserver(observer JobDoneObserver)

// Status 是非消费状态查询：只读 j.status，不推进 readOffset、不置 resultRead。
// 供 teammate_store 三处 status-only 查询迁移（interaction-design C7）。
func (m *Manager) Status(id string) (Status, bool)
```

**存储与锁**：`Manager` 新增字段 `jobDoneObservers []JobDoneObserver`（`taskRecorder` 旁），读写一律 `m.mu` 下；分发前锁内**快照拷贝**、锁外遍历（回调内可安全重入 `m.Start`/`AddJobDoneObserver`，无迭代器失效）。nil 元素跳过（event.FanOut L18-20 同构）。

### 3.2 触发点：`notifyTerminal` 与 recordCompletion 改造

```go
// notifyTerminal：先 recorder、后 completion observers。只在 recordCompletion
// 两处无锁点调用（均在 m.mu 释放后，锁序纪律 R-b）。
func (m *Manager) notifyTerminal(id string, st Status, err error) {
    if !nilutil.IsNil(m.taskRecorder) {
        m.taskRecorder.RecordDone(id, st, err)          // ① recorder 先（既有行为不动）
    }
    m.mu.Lock()
    obs := append([]JobDoneObserver(nil), m.jobDoneObservers...) // ② 快照
    m.mu.Unlock()
    m.callDoneObservers(obs, id, st, err)               // ③ observer 后，m.mu 外
}
```

- **替换点**：`recordCompletion` 两处现有 `taskRecorder.RecordDone` 调用（L975-977 suppress 分支、L1000-1002 正常分支）替换为 `m.notifyTerminal(id, st, err)` → 「先 recorder 后 observer」**结构化保证**而非靠调用点顺序。`kind/label` 不进 observer（3 参裁决；`recordCompletion` 仍收到但只喂 recorder/Notice）。
- **不新增第三个触发面**：destroying 提前 return（L982-985）在正常分支的 notifyTerminal 之前 → 不触发（与 RecordDone 一致）；suppress 分支无 destroying 检查 → **仍触发**（对齐 RecordDone 现状，J8 silent 变体待仲裁，见 §8）；`recordStalled`（L1018）非终态不触发；loaded tombstone 恢复不调 recordCompletion。
- **触发时序全景**（run goroutine L645-656）：`recordCompletion`（信封入队 L990-995 + notifyTerminal：RecordDone → observers）→ `j.status` 置终态（L649）→ `close(j.done)`（L656）。observer 收 `st=Done/Failed/Killed` 时 `j.status` 字段仍 Running → 权威性契约来源。
- **startInvalid**（L454-481）：`close(j.done)`（L478）→ `recordCompletion`（L479，同步于调用者）→ observer 收 Failed。两条路径下回调都只依赖参数。

### 3.3 抑制面矩阵（与 TaskRecorder.RecordDone 逐格对齐）

| 场景 | RecordDone | 完成事件 | 说明 |
|------|:---:|:---:|------|
| 正常 job（Done/Failed/Killed） | ✓ | ✓ | 唯一出口 recordCompletion |
| silent（P5 fork，L963-978 分支） | ✓ | ✓ | `TestStartSilentStillFiresTaskRecorder` 同构 |
| foreground（claim 分支） | ✓ | ✓ | 同上 |
| startInvalid（L479） | ✓ | ✓ | 消费者对未知 id 幂等丢弃 |
| destroying 窗口正常分支（L982-985） | ✗ | ✗ | 提前 return |
| destroying 窗口 suppress 分支 | ✓（现状） | ✓（对齐现状，待仲裁） | 见 §8 ③ |
| recordStalled（L1018） | ✗ | ✗ | 非终态 |
| Close 后收尾 job | ✓（既有） | ✓（对齐，待仲裁） | 见 §8 ② |
| loaded tombstone 恢复 | ✗ | ✗ | 不经过 recordCompletion |

---

## 4. TeammateStore 完成 handler（置 idle + 依赖推进 + mailbox 唤醒）

### 4.1 OnJobDone 伪代码（实现顺序固定）

```go
// OnJobDone 是 JobDoneObserver：只处理 ts.tasks 已登记或 LastJobID 匹配的 id
// （G8 过滤：P5 silent fork / foreground task / 任意后台 job 未登记 → 丢弃）。
func (ts *TeammateStore) OnJobDone(id string, st jobs.Status, err error) {
    if st == jobs.Running {
        return // 只处理终态（H4 固化）
    }
    ts.mu.Lock()
    // ① 依赖推进 = 终态落定（写后不可变，幂等）
    if t := ts.tasks[id]; t != nil && !isTerminalStatus(t.Status) {
        t.Status = string(st)
    }
    // ② 置 idle：LastJobID 严格匹配防旧完成（同 Complete L309-311）
    var hit *Teammate
    for _, tm := range ts.teammates {
        if tm.LastJobID == id && tm.State == TeammateRunning {
            tm.State = TeammateIdle
            hit = tm
            break
        }
    }
    // ③ 收集等待任务（反向扫描 dependents）
    var pendings []string
    for tid, t := range ts.tasks {
        if t.JobID == "" && (t.Status == "pending" || t.Status == "queued") &&
            contains(t.DependsOn, id) {
            pendings = append(pendings, tid)
        }
    }
    ts.mu.Unlock()
    // ④ 自动推进入队（非阻塞；队列满 → 置 blocked + 记日志，绝不阻塞 run goroutine）
    for _, tid := range pendings {
        ts.enqueueAutoAssign(tid)
    }
    // ⑤ mailbox 唤醒：置 idle 后锁外 countInbox → N>0 → 聚合 Notice（G2）
    if hit != nil && ts.inboxRoot != "" {
        if n := ts.countInbox(hit.Name); n > 0 {
            ts.notifyMailBacklog(hit.Name, n)
        }
    }
}
```

要点：
- **防旧完成**：`LastJobID == id` 严格匹配（R3）。连续两 job 时旧完成事件 `id==J1 ≠ LastJobID==J2` → 不置 idle（H2 固化）。
- **依赖推进的"推进"语义**：完成事件把 dep 的终态写进 `tasks[dep].Status`（①），自动 Assign worker 随后把等待任务拉起（④）；`pendingDependenciesLocked`（纯内存主源 + 非消费兜底，§5）对 `Status∈{done,failed,killed,interrupted,cancelled}` 放行。
- **幂等**：重复完成事件 → ① 终态不可变跳过、② 已 idle 不再置、⑤ 可能重复一条 Notice（H10 固化"状态不变"，通知重复接受并标注）。

### 4.2 recordTask 数据模型补存（回答「没存 prompt 要补什么」）

```go
type TeamTask struct {
    ID            string    // 任务 key；等待任务用占位 ID（如 "pending-N"），派活后 key 不变
    JobID         string    // 新增：实际 jobID；等待任务为空串
    Owner         string    // 已有：= Assign 的 name
    Prompt        string    // 新增：Assign 的 prompt（自动重放 Assign 的必需输入）
    SessionID     string    // 新增：leader 会话（jobs.SessionFromContext(ctx)，Assign L225 现成来源）
    CreatedAt     time.Time // 新增：登记时间（等待/超时/展示）
    DependsOn     []string  // 已有
    Status        string    // 补初值 "running"；pending/queued/blocked 为等待态；终态不可变
    BlockedReason string    // 新增（可后置）：blocked 时携带 reason
}
```

| 字段 | 现状 | 必要性 | 理由 |
|------|------|--------|------|
| `Prompt` | ❌ | **必须** | 完成事件无用户输入，唯一输入源 = 登记时快照的 prompt（datamodel-design §1.2） |
| `SessionID` | ❌ | **必须** | 自动 Assign 需要 `jobs.WithSession(ctx, sessionID)` 启动续轮；**存 id 字符串而非 ctx**（ctx 不可序列化、可能带 deadline） |
| `JobID` | ❌ | **必须** | 表达「等待任务尚无 job」；`JobID==""` + Status∈{pending,queued,blocked} 即等待态 |
| `CreatedAt` | ❌ | 建议 | 等待/超时/重试决策与 /team-status 展示 |
| `Status` 初值 | 恒空串 | **必须** | 事件驱动下 `tasks.Status` 为主源；`Tasks()`/`pendingDependenciesLocked` 依赖它 |
| `Owner/DependsOn` | ✅ | 保持 | — |

**登记策略（一次登记，禁止双条目）**：手动 `Assign` 成功路径仍 `recordTask(jobID, …)`（L226）；等待任务走 `recordTask(占位ID, …, Status="pending")`，自动 Assign 成功后**回填 `JobID` 字段不改 key**（`recordTask` 改造成 upsert：对已知 ID 幂等更新）。占位 ID 与 jobID 永不双条目（datamodel-design §2.3；测试固化 Tasks() 无重复）。

### 4.3 自动 Assign：等待登记 + worker 化（锁与死锁风险）

**等待任务来源（行为变更，C7④）**：`Assign(ctx, name, prompt, dependsOn…)` 依赖门拒绝时（L179-185），除返回错误外**登记为 pending**（错误文案保留 "blocked by unfinished dependency: …"，附注 "registered as pending, auto-starts when deps finish"）。`TestTeammateDependencyGate`（teammate_store_test.go:169-203）断言需同步更新：拒绝后 alpha 完成 → **自动推进已 Assign beta**（断言 beta 的 `LastJobID != ""` 且 Tasks() 可见），不再二次手动 Assign。注意：**running-teammate 拒绝（L173-176）不登记**（与依赖门拒绝是两条路径，测试须区分）。

**worker（TeammateStore 内部）**：
- 字段：`autoCh chan string`（有界，如 128）+ `autoWG sync.WaitGroup` + 惰性启动单 worker goroutine；`DestroyAll` 时排空。
- 消费伪代码：
```go
func (ts *TeammateStore) autoAssignWorker() {
    for tid := range ts.autoCh {
        ts.tryAutoAssign(tid)
    }
}
func (ts *TeammateStore) tryAutoAssign(tid string) {
    ts.mu.Lock()
    t := ts.tasks[tid]
    if t == nil || t.JobID != "" || t.Status == "blocked" { ts.mu.Unlock(); return } // 幂等/防重复推进
    owner, prompt, sessionID, deps := t.Owner, t.Prompt, t.SessionID, append([]string(nil), t.DependsOn...)
    ts.mu.Unlock()
    ctx := ts.assignTemplateCtx(sessionID) // 模板 ctx：WithManager+WithForkSource+WithSession+WithParentSession
    jobID, err := ts.Assign(ctx, owner, prompt, deps...)
    ts.mu.Lock()
    if err != nil || jobID == "" {
        t.Status, t.BlockedReason = "blocked", reason(err) // 绝不静默悬空（R-h）
        ts.mu.Unlock()
        ts.scheduleRetry(tid) // 可选：3 次 × 1s/5s/30s 退避（risk-review G5）
        return
    }
    t.JobID, t.Status = jobID, "running" // Assign 内 recordTask 已 upsert；此处回填 JobID
    ts.mu.Unlock()
}
```

**锁与死锁分析（结论先行：无环，前提是三禁 + worker 化）**：
1. `OnJobDone` 从 jm **无锁点**进入（m.mu 已释放）→ `ts.mu` 短临界 → 释放。若在 ts.mu 内调 `ts.jm.*`（`Output`/`SendMessageForSession`）→ `ts.mu → jm.mu`，与 `Assign` 的 `ts.mu → jm.mu` 同序但不构成环；**仍禁止**（消费语义 + 重入面 + 性能，datamodel-design §5.2 P1）。→ 三禁①。
2. worker 内 `Assign`：`ts.mu`（状态检查+置 Running）→ 释放 → `RunProfileSpec` → `StartForSession` 的 `m.mu`（L534-563）→ 释放 → `ts.mu`（回填 L216-222）。`m.mu` 获取时 ts.mu 已释放 → **无嵌套无环**（risk-review §2.5 事实核查：RecordDone 同层挂点 m.mu 外，安全）。
3. worker 与手动 `/team-add` 并发：共用 `ts.mu` 串行；手动先占 → worker 的 Assign 收到 "is running" → 置 blocked + 重试（R-h 兜底）。
4. 环检测（R-i）：`recordTask`/等待登记时对新边 `dep → task` 做 DFS 闭包环检测（task 的依赖链中是否含自身），命中则拒绝登记并返回 "dependency cycle"。链深度上限 32（超限拒绝，防 committed 增长失控，risk-review G4-e/G7-4）。
5. `autoCh` 满：`enqueueAutoAssign` 非阻塞；满 → 该任务置 `blocked("queue full")` + `slog.Warn`（绝不阻塞 run goroutine，R-e）。

---

## 5. mailbox 唤醒（job 完成时未投递信件 → Notice 给 leader）

- **触发点**：`OnJobDone` 步骤⑤（§4.1）。顺序固定「先置 idle 后检查」：置 idle 让 `/team-status` 即时可派活；检查在锁外（`countInbox` 毫秒级 ReadDir，G2）。
- **判定**：`countInbox(name)` = inbox 目录下非目录且 `.json` 结尾的文件数（与 `flushMailbox` L434-455 消费对象一致；共享目录 helper 防双真源）。`ts.inboxRoot == ""`（ephemeral）→ 恒 0 → 降级为「PostMail 即时通知」现状（mailbox-design §4.1）。
- **Notice 文案**（固定字节、无时间戳、经 event.Sink 只进 UI 不进 provider 输入）：
  - 完成唤醒（新）：`teammate X has N unread mail — /team-add X <task> to flush it`
  - 复用 `notifyMail` 既有 Emit 模式（L402-410：锁内取 sink 指针、锁外 Emit）。建议拆 `notifyMailReceived`（PostMail 即时，现状）与 `notifyMailBacklog(name, n)`（完成聚合）。
- **与 PostMail 即时通知的关系**：`flushMailbox`（Assign 后）已清空 inbox → 完成事件查到的积压**必为未 flush 的新 mail**，语义不重复（R5）；边缘竞态（PostMail 落在 flush 与完成检查之间）→ 一次即时 + 一次完成唤醒，低频接受。可选跟进 S2 分流（PostMail 按 idle/running 状态分流）列为阶 2，不进 MVP 主路径。
- **风暴控制**：每完成至多 1 条、且仅当有积压；N 是存量快照。⚠ N 动态计数与 risk-review L176「文本无动态计数」冲突——**已由 mailbox-design §7 标注交纪律团仲裁**；若否决，回退文案 `has unread mail`（不带 N）。

---

## 6. 与惰性 syncStateLocked 的关系（事件为主、惰性兜底）

| 维度 | 事件驱动（主） | 惰性 syncStateLocked（兜底） |
|------|---------------|------------------------------|
| 触发 | `OnJobDone`（recordCompletion 无锁点，同步） | `List()`（L118-128）时 |
| 置 idle | 立即（`State=Idle` + tasks.Status 终态落定） | 延迟到下次 List（不写 Ref） |
| 防时序 | `LastJobID == id` 严格匹配（R3） | 同上匹配（L132-138） |
| 职责 | **主路径**：事件到达即置 idle，/team-status 即时准确 | **兜底**：observer 未接线窗口（测试/旧路径）、destroy 窗口吞事件、历史 teammate（LastJobID 未登记） |

**改造**：
1. `syncStateLocked` 从 `jm.Output(LastJobID)`（L135）改为：先读 `ts.tasks[LastJobID].Status`（内存主源）→ 缺失时经**非消费 `jm.Status(id)`** 兜底（绝不再用 `Output`）。
2. `pendingDependenciesLocked`（L243-262）改**纯内存读为主** `ts.tasks[dep].Status`：终态集合（done/failed/killed/interrupted/cancelled）放行、其余阻塞；**能力兜底**：`Status==""`（dep 未登记，如任意外部后台 job 作依赖）时经非消费 `jm.Status(dep)` 判定终态（保持现状对任意 job 依赖的兼容，但不再消费 Output、不再有 purge 悬空）。锁内零 jm 调用（ts.mu→jm.mu 锁序面消除，P1；兜底查询也在锁外完成再判）。
3. `Tasks()`（L265-284）改纯读：以 `tasks[id].Status` 为主源；`Status==""`（未登记）降级非消费 `jm.Status(id)`；不再在只读方法内改写（L279 移除）。
4. `syncStateLocked` 注释更新：「懒同步降级为兜底，事件驱动为主」（现状 L116-117 注释明言 "no completion callback needed for the MVP" 已过时）。

**防时序例**：完成事件 →（回调延迟）→ `List()` 惰性置 idle → 用户 Assign（新 job 占槽）→ 旧完成事件晚到 → `LastJobID` guard 挡住 → 安全（risk-review §2.4 逐交错已验证，本设计沿用同一 guard）。

---

## 7. 任务分解（子任务 + 验证点）

依赖序：T1 → T2/T3 → T4 → T5；T6 贯穿。可并行：T2 与 T3 的测试起草。

- [ ] **T1（jobs 层多订阅者完成事件）**：`internal/jobs/jobs.go` 新增 `JobDoneObserver` 类型 + `WithJobDoneObserver`（追加 Option）/`AddJobDoneObserver`/`SetJobDoneObserver`（清空+单一）+ 私有 `notifyTerminal`/`callDoneObservers`（per-observer recover）；替换 `recordCompletion` 两处 `taskRecorder.RecordDone` 调用（L975-977 / L1000-1002）为 `notifyTerminal`；新增非消费 `Status(id)`。
  → 验证：`internal/jobs/jobs_done_observer_test.go`（J1-J12，范式对齐 jobs_recorder_test.go）：Done/Failed/Killed/InvalidStart 各恰一次；**多订阅者 J5（3 个全收、注册序一致）**；panic 隔离 J6；silent/foreground 仍触发 J9；Set 时序 J11（Set 前已完成不补发）；阻塞同步固化 J12；`go test ./internal/jobs/` 零失败（重点 TestDrainMultiple -race 不 flake）。
- [ ] **T2（TeammateStore.OnJobDone + recordTask 补存）**：`internal/agent/teammate_store.go` 新增 `OnJobDone`（§4.1 顺序）；`TeamTask` 增字段 + `recordTask` 改 upsert（Status 初值）；`Remove`（L346-362）清理该 owner 的 tasks（含等待态；running orphan job 照跑，完成事件写不存在条目 no-op）；`isTerminalStatus`/`contains` helper。
  → 验证：`internal/agent/teammate_done_test.go` H1-H10（确定性直接注入）：置 idle（不调 List）/旧 job noop/未知 noop/Running 忽略/tasks 补存幂等/owner 已删不 panic/mailbox 唤醒（有积压 1 条、无积压 0 条）。
- [ ] **T3（自动推进：等待登记 + worker + 环检测）**：`Assign` 门拒绝时登记 pending（行为变更，C7④）；`autoCh` 有界队列 + 单 worker + `tryAutoAssign`（blocked/重试）+ `assignTemplateCtx`；DFS 环检测 + 深度上限 32。
  → 验证：H5/H6 确定性（2 级链：注入 a 完成 → 等待 b 自动 Assign；Failed/Killed 终态同样放行）；环登记被拒（A→B→A 错误含 cycle）；worker 与手动 Assign 并发不重复推进（`-race`）；更新 `TestTeammateDependencyGate`（拒绝→自动推进→断言 beta LastJobID 非空）。
- [ ] **T4（惰性兜底改造 + 非消费查询迁移）**：`syncStateLocked`/`pendingDependenciesLocked`/`Tasks` 迁移到 tasks.Status 主源 + `jm.Status(id)` 兜底；删 `jm.Output` 三处 status-only 调用。
  → 验证：H9（断开 jm 兜底返回补存值）；`TestTeammateDependencyGate` 纯内存路径（dep 完成不再依赖 jm 存活）；`-race` 无锁序报告。
- [ ] **T5（生产接线）**：`internal/boot/boot.go` L1930 拆变量 `teammates := agent.NewTeammateStore(taskTool, jm)` → `teammates.SetSink(sink)`（sink 为 L249 的 event.Sync 包装）+ `jm.SetJobDoneObserver(teammates.OnJobDone)` → `Options{Teammates: teammates, …}`；controller.New 内注入自动 Assign 模板 ctx（复用 applyTeamCommand L6365-6373 装配，`SetAssignTemplateContext`）。可选 T3b：`control.New` 兜底 `SetSink`（幂等）。
  → 验证：`go build ./...`；`go test ./internal/control/` 零失败；team_real_test（`//go:build realapi`）编译通过。
- [ ] **T6（测试矩阵 + 四验证 + 提交）**：组合测试 C1-C4（真实异步全链：`jm.SetJobDoneObserver` 接线 → Assign → WaitForSession → 不调 List 断言 Idle；依赖链真实放行；运行中 PostMail + 完成唤醒；Failed/Killed 终态）；竞态 R1-R3（手动注入并发打击）；`go test -race ./...` 全绿；四验证 gofmt/vet/repolint/test；单主题 commit（message 建议 `team: drive teammate lifecycle from job completion events`）。
  → 验证：T6 自身即门；execution.md 留证据链，discipline 独立审查。

---

## 8. 对抗自检（devil's advocate）

1. **「3 参签名丢掉 parentSession/kind/label，够用吗？」**——攻击：跨会话同 id 冲突、kind 过滤。反制：`jobID`（`kind-seq`，L536）在 Manager 内全局唯一；`OnJobDone` 以 `ts.tasks[id]` 登记过滤（G8）不依赖 kind；parentSession 对 handler 无用（tasks 以 id 为键）。**薄弱环节①**：jobsevent-design D2 的 struct 式 6 参是既有文档裁决，本设计 3 参是任务更高裁决——已列偏差表（§9），纪律团须确认后执行，避免执行队按旧文档实现 struct 式。
2. **「Close 后仍触发 vs test-strategy J7 不触发」**——攻击：Close 收尾 Killed 事件会置 teammate idle + 补存 Killed，语义是否多余？反制：对齐 `TaskRecorder.RecordDone` 既有行为（taskmonitor 在 Close 路径写 Killed，jobs.go L2138-2143），零新抑制面、幂等无害。**薄弱环节②**：J7 断言互斥，执行前必须仲裁定稿；若纪律团否决，回退 = `notifyTerminal` 入口检查 `m.root.Done()`，改动收敛一行（J1-J6/J8-J12 不受影响）。
3. **「自动推进在 MVP 引入同步 Assign 风险」**——攻击：worker 虽独立 goroutine，但 `tryAutoAssign` 内 `Assign` 的 fork prefill（捕获 leader 前缀 + transcript 磁盘写）仍可能慢。反制：worker 串行 + 有界队列（不阻塞 run goroutine）；Assign 的 RunInBackground 同步段仅 fork+prefill（task.go L873-875），可加每项超时（如 10s）。MVP 接受串行（正确性优先）。**薄弱环节③**：worker 队头阻塞 → 后续推进延迟，队列水位告警列为可选。
4. **「Assign 拒绝即登记 = 行为变更，可能误登记用户本要放弃的派活」**——攻击：用户 `dependsOn` 写错 → 任务被挂起等待永不执行。反制：Tasks() 展示 queued/blocked + reason（可见）；错误文案明示已登记；环检测 + 深度上限挡住畸形链；`/team-status` 是既有查询出口。**薄弱环节④**：`TestTeammateDependencyGate` 与 `TestTeammateAssignRejectsUnknownAndRunning` 中 running 拒绝**不**登记（仅依赖门拒绝登记），测试须区分两条拒绝路径。
5. **「tasks.Status 为主源，observer 未接线时 Tasks() 空」**——攻击：测试/旧路径展示空态。反制：`Status==""` 降级非消费 `jm.Status(id)`；boot 接线是 T5 必做，生产无窗口；H9 固化。
6. **「mailbox N 动态计数 vs risk-review L176」**——已由 mailbox-design §7 仲裁路径覆盖（N 是用户驱动存量快照，非事件流累计计数）；若纪律团从严，回退文案一行改动（不带 N）。
7. **「Ref 断点（Complete 无调用者）是否要本设计接管？」**——**不接管**：datamodel-design §4.3 已裁决 ref 更新走 `Assign` 解析路径（`teammateRef(out)` 解析 `Subagent reference:`，task.go L1973-1978），**不走完成事件**（3 参签名拿不到 ref）。列为独立跟踪项（T3 可顺带修复 `tm.Ref` 写回旧值 bug，L218——首轮 fork 后 Ref 仍空、续轮误走 fork）。标注防越界。
8. **「destroy 窗口 suppress 分支仍触发（§3.3）vs test-strategy J8 断言 silent 不通知」**——攻击：destroy 期 silent fork 完成 → 通知 handler。反制：对齐 RecordDone 现状（非回归）；handler 对未登记 id 幂等丢弃。**薄弱环节⑤**：J8 的 silent 变体断言需随 §8 ②仲裁一并定稿（统一「destroy 不通知」需在 notifyTerminal 加 destroying 检查，但 3 参签名无 parentSession → 需 `m.get("", id)` 反查或改签名，成本上升——默认走「对齐现状」）。

**薄弱环节排名**：① 3 参 vs struct 式（纪律团确认）→ ② Close/destroy 语义仲裁 → ④ 拒绝即登记的行为变更（现有测试更新）→ ③ worker 队头阻塞 → ⑦ Ref 断点（顺带修复，独立跟踪）。

---

## 9. 与既有文档偏差表（交纪律团仲裁）

| 文档 | 原裁决 | 本设计 | 偏差性质 |
|------|--------|--------|---------|
| jobsevent-design D2 | struct 式 `JobCompletionObserver func(JobCompletion)`（6 字段） | **3 参** `func(id string, st jobs.Status, err error)`（任务路径 B 裁决） | **实质**：任务红线优先；参数集合裁剪（kind/label/parentSession 对消费方无用） |
| jobsevent-design D3 / test-strategy D1 | 多订阅者（With 追加/Add/Set） | 采纳（3 参版） | 一致 |
| test-strategy J7 | Close 后**不通知**（检查 m.root.Done()） | Close 后**仍通知**（对齐 RecordDone） | **实质冲突**，§8②，需仲裁 |
| test-strategy J8 silent 变体 | destroy 窗口 silent **不通知** | destroy 窗口 suppress 分支**仍通知**（对齐 RecordDone 现状） | **实质冲突**，§8⑧，随 ② 一并仲裁 |
| implementation-plan T2 | jm.Output 优先、tasks.Status 兜底；MVP 不自动 Assign | **tasks.Status 为主源**；**自动 Assign 等待任务**（任务硬需求②） | 演进：datamodel-design §3 + 用户任务覆盖 |
| implementation-plan T1 | 单例 WithJobDoneObserver + Set | 多订阅者切片 + With(追加)/Add/Set | 演进：用户「多订阅者」裁决 |
| risk-review L176 | 文本无动态计数 | mailbox N 存量快照（mailbox-design §7 已标注） | 已标注，维持 |
| mailbox-design | 完成唤醒为「检查积压并通知」 | 采纳；未上 S2 分流（阶 2） | 一致（MVP 范围裁剪） |

---

## 10. 缓存/纪律检查点（Reasonix 领域）

- ✅ **发送侧字节零变化**：完成事件 = 进程内回调（jobs→agent）+ event 层 Notice（UI）；不触碰 schema/system prompt/transcript/task.go 发送路径/subagent_store；teammate fork/continue 三要素与前缀完全不受影响。
- ✅ **前缀稳定**：`notifyTerminal` 不新增任何 Notice/envelope/事件注入（recordCompletion 的 m.completed 入队与 closing Notice 输出**逐字节不变**）；mailbox 唤醒 Notice 只进 event.Sink（前端显示，不进 provider 输入，risk-review L167）；自动推进不自动 PostMail/steer 写 teammate transcript（G7）。
- ✅ **运行值不进 schema/system prompt**：name/prompt/sessionID/tasks.Status/LastJobID/mail 积压均为内存/磁盘态登记数据，非发送前缀组成部分。
- ✅ **锁序纪律**：`notifyTerminal` 两处调用点均无 m.mu/j.mu 持有；`OnJobDone` 先释 ts.mu 再做 ReadDir/通知；worker 内 Assign 的 m.mu 在 ts.mu 释放后获取（无环）；`pendingDependenciesLocked` 改纯内存读后 ts.mu 临界区零 jm 交互。
- ✅ **防虚假完成**：每个 T 的「完成」= 测试输出 + `go test -race ./...` 全绿 + 事务四验证（gofmt/vet/repolint/test）+ execution.md 证据链；§8 薄弱环节 ①②④⑤ 须纪律团先裁决后执行。

---

## 附：执行队禁止事项（按章程）

- 不得实现 struct 式 6 参 `JobCompletion`（本设计已裁决 3 参，jobsevent-design 相应段落作废）。
- 不得在 `recordCompletion` 的 m.mu 临界区内调用 observer；不得在 `OnJobDone` 内调 `ts.jm.*`、持 ts.mu 做 IO、同步 Assign（三禁）。
- 不得改动 `WithJobStartObserver` 既有签名与行为（boot.go:517 workspaceLease 依赖）；不得改 TaskRecorder 接口（controller.go:695 taskmonitor）。
- 不得把「字段补存」解读为「已获自动调度授权之外的扩展」——自动推进仅限依赖门拒绝登记 + 完成事件驱动的等待任务，链深度上限 32 必须生效。
- 不得删除 `syncStateLocked` 惰性兜底（现有 teammate 五测回归锚点）。
- 不得声称完成——须附测试输出与 -race 证据，纪律团独立审查。
