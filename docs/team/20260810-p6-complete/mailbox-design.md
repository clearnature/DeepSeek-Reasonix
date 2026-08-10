# 设计：事件驱动「mailbox 唤醒」子机制（独立设计）

> 事务：docs/team/20260810-p6-complete/ · 视角：Planner（只读规划，不写实现）
> 主题：job 完成事件 → teammate 置 idle → inbox 有积压（完成前到达未投递）→ 自动 `notifyMail`，leader 收到 Notice：teammate X 有 N 封未读，可 `/team-add` 派活 flush。
> 与事务既有文档的关系：`implementation-plan.md` 已裁决完成事件源 = jobs 层 `WithJobDoneObserver`（方案 A，同步回调 + recover）；`risk-review.md` 给出 G2（回调 m.mu 外）/G7（MVP 零新发送）/§2.6（Notice 风暴）纪律。**本设计不重推事件源方案**，在 `OnJobDone` 的 mailbox 唤醒步骤（implementation-plan T2 步骤 5）内做细化，并新增两项增强（PostMail 原子落盘、PostMail 按状态分流），其中与 risk-review §2.6 的裁决偏差在 §7 显式标注，交纪律团仲裁。

---

## 一、结论先行（TL;DR）

1. **完成事件源**：复用 implementation-plan 方案 A —— jobs 层 `WithJobDoneObserver`，在 `recordCompletion` 两处无锁点（jobs.go L975-977 / L1000-1002）并列接线；mailbox 唤醒挂在 `TeammateStore.OnJobDone` 内。**不新增第二个全局钩子、不抢 `WithJobStartObserver`（boot.go:517 workspaceLease 已占）与 `TaskRecorder`（controller.go:695 taskmonitor 已占）**。
2. **mailbox 检查点（要点 1）**：`OnJobDone` 顺序固定为「①终态校验 → ②ts.mu 内置 idle（`LastJobID == id` 严格匹配，同 `Complete` L309-311）→ ③ts.mu 外 `countInbox(name)` → ④N>0 → `notifyMail(name, N)`」。**先置 idle 后检查**，理由见 §4.1。
3. **Notice 文案与字段（要点 2）**：复用 `event.Notice + LevelInfo + Text`，**不新增 Kind、不加结构化字段**（event 是 wire 稳定协议，新增 Kind 需动 KindCount 与 golden）。文案固定模板：`teammate X has N unread mail — /team-add X <task> to flush it`（N = inbox 存量快照；命令名为 `/team-add`，controller.go L6359，任务描述中的 `/team-assign` 系笔误，文档内纠正）。
4. **竞态（要点 3）**：三处竞态逐一定性——(R-A) 完成计数 vs Assign flush 并发 → N 为快照、允许轻微过时；(R-B) PostMail 半写文件被 `flushMailbox` 读败仍删除（L451-454）→ **修 PostMail 为 tmp+rename 原子落盘**（新增增强 1）；(R-C) 完成通知 vs 新 Assign → 靠 ts.mu 内状态判定收敛，接受稀有重复。
5. **语义对比（要点 4）**：**「到达即唤醒」与「完成事件唤醒」不互斥，建议 A+B 结合（语义 S2）**——PostMail 按 teammate 状态分流：idle → 即时通知（到达即唤醒）；running → 静默落盘（完成事件聚合兜底）。两者互补且**不产生重复通知**（idle 期信件只有即时通知、无完成事件；running 期信件只有完成聚合、无即时通知）。最小改动 MVP 可只上「完成事件唤醒」并保留 PostMail 现状（无条件即时通知），S2 分流列为紧随其后的小步。
6. **生产接线缺口**：boot.go L1930 `NewTeammateStore(taskTool, jm)` **从未 `SetSink`**（全仓仅测试调用）→ 现有 notifyMail 在生产静默空转；本机制必须一并接 `SetSink(event.Sync(...))`（boot.go L249 的 sink 已是 Sync，并发安全）与 `jm.SetJobDoneObserver(ts.OnJobDone)`。
7. **降级语义**：无 `inboxRoot`（ephemeral）时 inbox 无落盘 → `countInbox` 恒 0 → 完成唤醒不触发，退化为「PostMail 即时通知」（现状行为），文档明示该边界。

---

## 二、拓扑扫描

### 2.1 文件与现状事实（已核实）

| 文件:行 | 现状 | 与本次设计的关系 |
|---------|------|------------------|
| `internal/agent/teammate_store.go:368-397` `PostMail` | 落盘（L391-392 `os.WriteFile` 非原子）+ 无条件 `notifyMail`（L395；ephemeral 分支 L384 同） | ①改原子落盘；②改状态分流通知；③N 计数入口 |
| `:399-410` `notifyMail` | 仅带 name，文案固定 `teammate X received mail — /team-add <name> <task> to flush it`，无未读数；Emit 在 ts.mu 外（先取 sink 指针再释放锁） | 扩展签名带 N + 区分「收到」/「积压」两种文案；锁外 Emit 模式保持 |
| `:428-457` `flushMailbox` | 读 inbox → `jm.SendMessageForSession` 注入 P3 → **无论 Unmarshal 成败都 `os.Remove`（L451-454）** | 半写丢信隐患 → 需 PostMail 原子落盘配合 |
| `:130-138` `syncStateLocked` | 懒同步（List 时查 `jm.Output` 置 idle），注释明言 "no completion callback needed for the MVP" | 事件驱动后降级为兜底（注释更新，behavior 保留兼容） |
| `:300-316` `Complete` | **无调用方**（遗留 API），含 `LastJobID` 严格匹配语义（L309-311） | 完成回调的匹配语义复用；ref 更新链路不在本设计范围（见 §7） |
| `:165-231` `Assign` | 锁内状态判定（L173-186）→ 启动 job → L225 flushMailbox → L226 recordTask | 与完成检查的并发点为 R-A |
| `:60-73` struct / `:68-70` sink / `:151-157` SetSink | sink 字段 + Setter，生产未接 | M5 接线 |
| `internal/jobs/jobs.go:937` `recordCompletion` | 终态唯一出口；L975-977 / L1000-1002 两处无锁点调 `taskRecorder.RecordDone` | `WithJobDoneObserver` 并列接线点（implementation-plan T1，本设计依赖） |
| `:296` `WithJobStartObserver` / `:316` `SetTaskRecorder` | 全局单值钩子，分别被 boot.go:517（workspaceLease）与 controller.go:695（taskmonitor）占用 | 不可抢 → 新 `SetJobDoneObserver`（implementation-plan 裁决） |
| `:1245-1266` `WaitForSession` | 事件驱动（select `j.done`） | 备选事件源参考（方案 B，已否决）；本设计走 jobs 回调 |
| `internal/event/event.go:827-829` Sink / `sync.go:17-33` Sync | Sink 契约 = 串行 Emit；Sync 包装并发安全 | SetSink 必须接 Sync 包装 sink |
| `internal/event/coalesce.go:28` Coalesce | 只合并 Text/Reasoning 流，Notice 不合并（isStreamDelta L58-65） | 每条 Notice 独立送达 → 完成唤醒必须「聚合一条」防风暴 |
| `internal/boot/boot.go:249,1930` | L249 `sink := event.Sync(opts.Sink)`；L1930 `NewTeammateStore(taskTool, jm)` 未接 sink/observer | M5：拆变量 + `SetSink(sink)` + `jm.SetJobDoneObserver(ts.OnJobDone)` |
| `internal/control/controller.go:6359` | `/team-add <name> <task...>`（非 `/team-assign`） | 文案锚点 |
| `docs/team/20260810-p6-complete/implementation-plan.md` | 事件源方案 A 裁决（T1 jobs observer / T2 OnJobDone / T3 boot 接线） | 本设计在其框架内细化 mailbox 部分 |
| `docs/team/20260810-p6-complete/risk-review.md` | G2（回调 m.mu 外）/G7（MVP 零新发送）/§2.6（Notice 风暴）/L167（notifyMail 不进 transcript 安全） | 本设计遵守 G2/G7 精神；§2.6 偏差在 §7 显式标注 |

### 2.2 级联风险

- **R1（生产 sink 缺口）**：不接 SetSink → 唤醒 Notice 生产不可见（现状即如此）。→ M5 修复。
- **R2（完成通知风暴）**：fan-out（1 完成 → 10 后继）若每条完成各发 Notice → 刷屏。→ 每条完成至多 1 条、且**仅当有积压**（积压是用户驱动低频事件）；无积压不发。见 §4.3。
- **R3（重复通知）**：PostMail 无条件即时通知 + 完成聚合 → 完成窗口内理论重复。→ 状态分流（§4.4）消除主路径重复。
- **R4（半写丢信）**：PostMail 非原子写 + flushMailbox 读败仍删 → 丢信。→ tmp+rename（增强 1）。
- **R5（N 过时）**：完成计数与 Assign flush 并发删除 → N 偏差。→ 快照语义 + 文档明示。
- **R6（sink 并发 Emit）**：OnJobDone 在 job goroutine / startInvalid 调用者 goroutine 上运行，与 leader run loop 的 Emit 并发 → 违反 Sink 串行契约。→ SetSink 只接 `event.Sync` 包装 sink（boot.go:249 已是）。
- **R7（锁序）**：回调不得持 ts.mu 做 IO/通知。→ `countInbox` 与 `notifyMail` 都在 ts.mu 释放后执行（G2）。

---

## 三、多路径推演

> 完成事件源的三条路径（jobs 层 observer / done-channel 分发 / 扩展 TaskRecorder）已由 `implementation-plan.md` 裁决为方案 A，此处不重复。本节推演 **mailbox 唤醒自身的实现路径**。

### 方案甲（采纳）：OnJobDone 内同步「置 idle → 锁外 countInbox → notifyMail」
- 逻辑：全部复用现有 `Complete` 匹配语义 + `flushMailbox` 的目录计算 + `notifyMail` 的 Emit 模式；无新 goroutine、无队列。
- 复杂度：低（teammate_store.go +~40 行，其中 countInbox ~12 行）。
- 性能：O(1) 内存 + 一次毫秒级 ReadDir（ts.mu 外）；只在有积压时 Emit。
- 可维护性：与 implementation-plan T2 同构；`countInbox` 与 `flushMailbox` 目录逻辑共享一个私有 helper，避免双真源。
- 风险：ReadDir 与 PostMail/Assign 并发 → N 为快照（接受）；半写文件被计数 → 由增强 1（原子落盘）消除。

### 方案乙（否决）：完成唤醒走「PostMail 时机」的单一触发
- 只保留 PostMail 即时通知、完成事件不做任何检查。则 running 期间到达的信在完成时**无任何提醒**（leader 需手动 /team-status 才发现），不满足任务目标（「job 完成事件 → teammate 置 idle 后，若 inbox 有新信自动 notifyMail」）。否决。

### 方案丙（备选）：惰性唤醒（List/Status 时检查并通知）
- 即把检查点放进 `syncStateLocked`。优点：零 jobs 改动；缺点：**不是事件驱动**（违背本子机制标题），通知时机依赖 leader 主动查询，且 List 是只读路径、塞通知有副作用。否决为主案，列为「若 jobs 层改动被纪律团驳回」的退路。

**裁决**：方案甲。与事务既有实现计划一致，改动面最小、语义最准。

---

## 四、详细设计

### 4.1 mailbox 检查点（要点 1）——「置 idle 前后何时检查」

**结论：先置 idle，后检查 inbox。**

`OnJobDone` 内顺序（实现顺序固定，与 implementation-plan T2 步骤 1-5 对齐）：

```
1. st == jobs.Running → return（只处理终态）
2. ts.mu.Lock()
   - tasks[id].Status = string(st)          // 依赖推进落定（implementation-plan T2 步骤 2）
   - tm := teammates 中 LastJobID == id && State == Running → State = Idle   // 步骤 3，严格匹配
   ts.mu.Unlock()
3. ts.inboxRoot == "" → return              // ephemeral 降级（§一.7）
4. n := countInbox(name)                    // ts.mu 外，ReadDir 快照
5. n > 0 → notifyMail(name, n)              // ts.mu 外 Emit（G2）
```

理由：
- **状态优先**：置 idle 让 `/team-status` 立即反映可派活（leader 可马上 `/team-add`）；inbox 检查与状态无耦合（落盘不看状态）。
- **分流正确性**：置 idle 后再检查，使完成窗口内到达的 PostMail 走「idle 即时通知」分支（§4.4），由 PostMail 自身兜底，而不是依赖完成检查恰好数到——把「漏唤醒」的最坏窗口从「完成检查之后」收窄到「PostMail 与完成检查同瞬间」。
- 若反向（先检查后置 idle）：完成瞬间的 PostMail 走 running 静默 → 只能依赖完成检查数到，窗口更大、更易漏。否决。

代码锚点：`Complete` L300-316（匹配语义）；`syncStateLocked` L130-138（懒同步降级为兜底）。

### 4.2 notifyMail 的 Notice 文案与事件字段（要点 2）

**结论：复用现有格式，不新增 Kind/字段；文案改带 N。**

- **事件**：`event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: <文案>}`——与现状 `notifyMail`（L407-408）完全同构。理由：event.Kind 是 wire 稳定协议（新增需插 KindCount 前 + 全套 golden/coalesce 测试，见 event.go L116-118）；Notice 承载「一行可行动提示」的既有职责；UI 端已有 Notice 渲染，零前端改动。
- **文案模板**（固定字节、无时间戳，遵守 risk-review L176 的其余约束）：
  - PostMail 即时（语义 A，保持现状风格）：`teammate X received mail — /team-add X <task> to flush it`
  - 完成事件聚合（语义 B，本机制新增）：`teammate X has N unread mail — /team-add X <task> to flush it`
- **N 的定义**：`countInbox(name)` = `os.ReadDir(inbox 目录)` 中**非目录且以 `.json` 结尾**的文件数（与 `mailItem` 落盘名 `%d.json`（L392）与 `flushMailbox` 消费对象严格一致）。N 是「检查时刻存量快照」，随 flush 消费自然递减（文件被删）。
- **命令名纠正**：文案中的 `/team-add <name> <task>` 与 controller.go L6359 实际命令一致；任务描述中的 `/team-assign` 系笔误，设计内统一为 `/team-add`（避免 leader 按笔误命令空转）。
- **签名建议**：`notifyMail` 由 `notifyMail(name string)` 扩展为内部两个语义函数（私有）：
  - `notifyMailReceived(name string)`（PostMail 即时，文案 A）
  - `notifyMailBacklog(name string, n int)`（完成聚合，文案 B）
  - 公共入口保持 `notifyMail` 私有性不变；Emit 前取 sink 指针、Emit 在锁外（沿用 L402-409 现有模式）。
- **可选 Detail 字段**：不填。N 已在 Text 内；Detail 留空避免过度设计。

### 4.3 与 flushMailbox 的竞态（要点 3）

**结论：三处竞态，两处缓解、一处接受。**

| 竞态 | 场景 | 定性 | 缓解 |
|------|------|------|------|
| **R-A 计数 vs 消费** | 完成 `countInbox` 与并发 Assign 的 `flushMailbox`（删除文件）同时发生 | N 可能过时（报多）或恰好准确；**不会漏**（文件存在即计数；若被 flush 说明信已投递，过时无害） | N 为快照语义；leader 收到后 `/team-add` flush 的是实际存量，偏差仅影响提示精度；文档明示 |
| **R-B 半写 vs 消费** | PostMail `os.WriteFile`（L392）尚未写完，flushMailbox 读到半写 JSON → `json.Unmarshal` 失败（L451）→ **仍 `os.Remove`（L454）→ 信丢失** | **现有 bug 隐患**（竞态窗口极小） | **增强 1：PostMail 改 tmp+rename 原子落盘**——`os.WriteFile(dir/.tmp-<nano>)` 成功后 `os.Rename` 到 `%d.json`；flush 只可能看到完整文件或看不到（不存在半写可见）。`flushMailbox` 的「读败即删」在原子写下变得安全 |
| **R-C 完成通知 vs 新 Assign** | 完成置 idle 与 `/team-add` 的 Assign 并发：Assign 锁内读状态（L173-186）判定 idle 才通过 → 若 Assign 先获锁，完成检查随后数到已被 flush 的信 → 发过时通知（无害）；若完成先置 idle，Assign 正常派活 | 稀有、无害（多余一次信息性 Notice） | 接受；若后续实测噪音，可在 countInbox 与 notifyMail 之间加「inbox 是否已被新 job flush」二次判定（不做，防过度设计） |

补充：`flushMailbox` 本身由 Assign 在 ts.mu 外调用（L225），与完成回调的 ts.mu 短临界不重叠——锁序无新增交叉（G2 遵守）。

### 4.4 语义对比：信件到达即唤醒 vs 完成事件唤醒（要点 4）

**结论：两者互补，建议 A+B 结合（语义 S2）；MVP 可先上 B、A 分流紧随其后。**

| 维度 | A 到达即唤醒（PostMail 时机） | B 完成事件唤醒（job 完成时机） |
|------|------------------------------|-------------------------------|
| 触发 | 每封 PostMail | 每次 teammate job 完成（有积压才发） |
| 聚合 | 无（每封一条 Notice） | 有（N 封一条） |
| 覆盖盲区 | idle 期信件即时可见 | **running 期积压信在完成时可见**（A 单独存在时 leader 无感知） |
| 缺陷 | running 期也发 → 噪音；无 N | idle 期信件要等下一次 job 完成才提醒；若长期不派活 → 提醒无限延迟 |
| 现有实现 | ✅ 已存在（PostMail 无条件 notifyMail） | ❌ 缺失（本设计补齐） |

**建议（语义 S2，推荐落地形态）**：PostMail 按 teammate 状态分流——
- `tm.State == TeammateIdle` → 落盘 + `notifyMailReceived(name)`（A：到达即唤醒，即时、无聚合压力）；
- `tm.State == TeammateRunning` → 落盘 + **静默**（B 兜底：job 完成时 `notifyMailBacklog(name, N)` 一次聚合）。

不重复论证：idle 期信件只走 A（teammate 无进行中 job → 无完成事件）；running 期信件只走 B（A 静默）→ 每条信恰好触发一次通知路径，主路径零重复。

**MVP 分阶**：
- 阶 1（最小，满足任务硬需求）：仅加 B（完成事件唤醒），PostMail 保持现状无条件即时通知。重复窗口：running 期 PostMail 现在会即时通知 + 完成聚合再通知 → 存在重复（现状已如此，不回归）。代码量最小，先合入。
- 阶 2（推荐跟进）：PostMail 状态分流（A 仅 idle、running 静默）→ 消除重复、降噪。改动仅 PostMail 内加一次 ts.mu 读状态（注意先落盘后判状态，保证落盘先于分流判断的时序）。

**建议直接按 S2 落地**（本设计任务分解 M4 即分流；若纪律团要求 MVP 最小化，删 M4、保 M2/M3 即可，互不阻塞）。

### 4.5 生产接线（依赖 implementation-plan T3）

- `internal/boot/boot.go` L1930：拆出变量
  ```go
  teammates := agent.NewTeammateStore(taskTool, jm)
  teammates.SetSink(sink)                 // sink = event.Sync(...)（L249 已有，并发安全，满足 R6）
  jm.SetJobDoneObserver(teammates.OnJobDone)  // implementation-plan T1 的 API
  ```
  `Options{... Teammates: teammates ...}`。
- 顺序约束：boot 组装早期（L249）→ Options 组装（L1930），sink 变量已就绪；`SetJobDoneObserver` 在 `SetTaskRecorder`（controller.go:695）之后也无冲突（各自独立字段）。
- 防御性兜底（可选，同 implementation-plan T3b）：`control.New` 中 `if opts.Teammates != nil { opts.Teammates.SetSink(opts.Sink) }`，覆盖非 boot 组装路径；SetSink 幂等（重复调用覆盖）。

### 4.6 私有 helper：countInbox

```go
// countInbox 返回 teammate 磁盘 inbox 中待投递的 mail 文件数（快照）。
// 目录不存在返回 0；与 flushMailbox（L434-455）共享 sanitizeMailName 目录计算。
func (ts *TeammateStore) countInbox(name string) int {
    dir := filepath.Join(ts.inboxRoot, sanitizeMailName(name), "inbox")
    entries, err := os.ReadDir(dir)
    if err != nil { return 0 }
    n := 0
    for _, e := range entries {
        if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") { n++ }
    }
    return n
}
```
（目录计算与 `flushMailbox` L434 保持同一 helper 抽取，避免双真源。）

---

## 五、任务分解（mailbox 子机制，交付 executor）

> 前置依赖：implementation-plan **T1**（jobs 层 `WithJobDoneObserver`）合入后才可运行 M2 集成测试。M1/M3/M4 的单元测试不依赖 T1。

- [ ] **M1**：notifyMail 扩展——拆 `notifyMailReceived` / `notifyMailBacklog(name, n)`，文案模板固定（§4.2）；新增 `countInbox` helper（§4.6）
  → 验证: `go test ./internal/agent/ -run 'Teammate'`；单测断言文案含 name + N、无时间戳、emitter 在锁外（-race）
- [ ] **M2**：`OnJobDone` mailbox 唤醒——置 idle 后锁外 `countInbox` → N>0 → `notifyMailBacklog`（顺序 §4.1；`LastJobID` 严格匹配；ephemeral 降级）
  → 验证: 完成事件有积压 → sink 收 1 条含 N；无积压 → 0 条；旧 jobID 完成 → 不误唤醒
- [ ] **M3**（增强 1）：PostMail 原子落盘——`WriteFile(.tmp)` + `Rename`；`os.Rename` 失败返回错误、不残留可见半写
  → 验证: 现有 `TestTeammatePostMailPersistsAndFlushes` 零回归；并发 PostMail+flush 压力测试（-race）无丢信
- [ ] **M4**（增强 2，S2 分流）：PostMail 状态分流——先落盘，后读 `ts.mu` 判 idle/running：idle → `notifyMailReceived`；running → 静默（ephemeral 分支保持现状即时通知）
  → 验证: idle PostMail → sink 收；running PostMail → sink 不收且落盘成功；完成后聚合 1 条含 N
- [ ] **M5**：生产接线——boot.go 拆变量 + `SetSink` + `SetJobDoneObserver`（§4.5）；可选 control.New 兜底
  → 验证: `go build ./...`；`go test ./internal/control/` 零失败；`team_real_test.go`（`//go:build realapi`）编译通过
- [ ] **M6**：文档——本设计合入事务；execution.md 记录测试证据
  → 验证: 文件存在 + repolint + 四验证（gofmt/vet/repolint/test）

---

## 六、测试设计（要点 5）

复用 `teammate_store_test.go` 现有基建（`testTaskToolForTeam`、`jobs.NewManager(event.Discard)`）；sink 用带 channel 的 record sink（参考 `coalesce_test.go:21` 的 record 模式），断言 Notice 条数、Text 内容。

| 用例 | 场景 | 断言 |
|------|------|------|
| `TestOnJobDoneNotifiesBacklog` | Assign 后台 job；job 完成前 PostMail×2（落盘）；等终态（`jm.WaitForSession`） | sink 恰收 1 条 Notice，Text 含 `alpha`、`2`、`/team-add`；`ts.Status("alpha").State == Idle`（**不调 List**，证明事件驱动） |
| `TestOnJobDoneEmptyInboxSilent` | 同上去掉 PostMail | sink 0 条 Notice |
| `TestOnJobDoneStaleJobNoop` | 完成事件带旧 jobID（当前 LastJobID 不同） | 不置 idle、不通知 |
| `TestOnJobDoneEphemeralNoWakeup` | 无 inboxRoot（NewTeammateStore 不传 root）| 完成不通知（降级语义固化） |
| `TestPostMailIdleNotifiesImmediate`（M4） | idle 时 PostMail | sink 收 1 条 `received mail` 文案 |
| `TestPostMailRunningSilentThenAggregates`（M4） | running 中 PostMail×2 → 完成 | running 期 0 条；完成后 1 条 `has 2 unread mail` |
| `TestPostMailAtomicWriteSurvivesConcurrentFlush`（M3） | 并发 goroutine：PostMail 循环 ×N vs flushMailbox 循环；`-race` | 无丢信（inbox 终态 + 消息注入计数一致） |
| `TestNotifyMailTextNoTimestampStable` | 固定 name/N 两次调用 | Text 字节一致（无时间戳/随机段）→ 前缀/输出稳定 |
| 回归 | 现有 5 个 teammate 测试 + `TestDrainMultiple -race`（jobs） | 零失败 |

**竞态专项**：`go test -race ./internal/agent/ ./internal/jobs/`（R-A/R-B/R-C 覆盖）；fan-out 风暴测试（1 完成 + 有积压 → 恰 1 条，无放大）。

---

## 七、对抗自检（devil's advocate）

1. **「N 是动态计数，违反 risk-review L176『文本无动态计数』」**——攻击成立，但语境不同：L176 针对**自动推进**的状态流水计数（"3 完成 / 2 推进 / 1 失败"）；本设计的 N 是**用户驱动数据（mail）的存量快照**，一条 Notice 只报一次快照，不随事件流累计。裁决：任务硬需求（「N 封未读」）优先；风暴风险由「每完成至多 1 条 + 仅积压才发 + 固定模板」控制。**此为本设计与 risk-review §2.6 的显式偏差，标注交纪律团仲裁**；若纪律团否决 N，回退文案为 `has unread mail`（不带数），M2 改动一行。
2. **「G7 零新发送：完成事件加 Notice 属新增发送」**——G7 的语义是「不注入 input、不自动写 teammate transcript、推进状态不新增 Notice」。mailbox 唤醒 = P6.2 既有 `notifyMail`（risk-review L45/L153 认可的用户驱动通知）的**触发时机扩展**，Emit 只进 UI 层（risk-review L167 确认不进 transcript）→ 前缀零影响，属 G7 精神（不进 provider 输入）内。若纪律团从严解读，M2 可整体挂到阶 2（M4 分流后才启用完成聚合），现状行为不变。
3. **「fan-out 风暴：1 完成 → 10 后继，各带积压 → 10 条」**——每条完成至多 1 条且必须**有积压**才发；积压是用户驱动低频事件（无人 PostMail 则 0 条）；10 后继各自完成是时间分散的。可加日志级 `slog.Debug` 计数观测，无熔断需求。
4. **「R-A：countInbox 与 flush 并发导致 N 虚高 → leader 白派一次活」**——白派活无害（job 正常跑、flush 空转）；虚高仅在「完成检查与 Assign 同瞬间」发生，概率低。若实测噪音，加「flush 后 N=0 抑制」标记——不做（过度设计）。
5. **「R-B 半写修复引入 tmp 残留」**——Rename 失败时 `.tmp` 残留；`flushMailbox` 只认 `.json` 后缀（L451 后仍会 Remove 非 json？——现状 flush 对所有非目录文件都 Read+Remove，`.tmp` 会被当垃圾删掉，无残留风险；但为严谨，`flushMailbox` 建议加 `.json` 后缀过滤，避免吞掉未来合法文件）。标注为 M3 附带小改。
6. **「sink 未 Sync 就 SetSink（R6）」**——SetSink 文档契约改为「必须传并发安全的 sink（event.Sync 包装）」，boot 接线传 L249 的 Sync sink；测试 record sink 单线程无碍。加 `slog.Warn` 防御？不——接口约定即可。
7. **「ref 更新链路缺失」**——`Complete` 的 ref 参数（L313-315）在本机制中传 `""`（waiter/回调不提供新 ref），续轮 ref 更新维持现状（现状 `Complete` 本就无调用方，ref 停在旧值）。**这是既有缺口，非本机制引入**；标注为独立跟踪项，不在本子机制内解决（防越界）。
8. **「M4 分流后 ephemeral 分支不一致」**——ephemeral（无 root）时 PostMail 立即 notifyMail（L384），running 也发（无落盘可聚合）。裁决：ephemeral 是简化部署，保持现状（即时通知）合理；文档明示降级语义。

**薄弱环节排名**：① N 动态计数与 risk-review L176 的冲突（需纪律团裁决）→ ② R-A 快照过时（接受）→ ③ M4 分流时序（落盘先于状态判断，窗口内 running 判定延迟）。

---

## 八、缓存/纪律检查点（Reasonix 领域）

- ✅ **发送侧字节零变化**：本设计全部是进程内回调（jobs → TeammateStore）+ event 层 Notice（UI 显示）；不触碰 schema/system prompt/transcript/task.go/subagent_store；teammate fork/continue 三要素与前缀完全不受影响。
- ✅ **前缀稳定**：mailbox 唤醒 Notice 只进 `event.Sink`（前端/UI），**永不注入 leader input**（G7）；teammate transcript 不被自动写入（无自动 PostMail/steer）。
- ✅ **运行值不进 schema/system prompt**：name/N/积压状态均为内存与磁盘 inbox 数据，非发送前缀的一部分。
- ✅ **锁序**：完成回调在 `recordCompletion` 无锁点（implementation-plan T1 保证）；`countInbox`/`notifyMail` 均在 ts.mu 释放后执行（G2）；无新增锁序交叉。
- ✅ **防虚假完成**：每个 M 的完成 = 测试输出 + 四验证通过 + execution.md 证据链，discipline 独立审查；本设计与 risk-review §2.6 的偏差显式标注，交纪律团仲裁后才可执行 M2/M4 全量。

---

## 附：执行队禁止事项

- 不得新增 `event.Kind` 或结构化字段承载未读数（§4.2 裁决）。
- 不得在 `recordCompletion` 的 m.mu 临界区内调用 observer（implementation-plan 禁止项）。
- 不得自动 PostMail/steer 写入 teammate transcript（G7/risk-review L168 红线）。
- 不得擅自定义完成事件源替代方案（implementation-plan 已裁决方案 A；若 jobs 层改动被驳回，退路方案丙须先经纪律团批准）。
- 不得声称完成——须附证据（测试输出/文件 diff）。
