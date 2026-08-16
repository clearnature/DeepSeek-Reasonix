# 数据模型设计评审：「完成事件驱动」（独立视角）

> 事务：docs/team/20260810-p6-complete/ · 视角：独立评审（只读规划，不写代码）
> 范围：internal/agent/teammate_store.go（Teammate L24-44、TeamTask L53-58、recordTask L234-238、pendingDependenciesLocked L243-262、Tasks L265-284、Complete L302-316、Remove L346-362）+ internal/jobs/jobs.go（Job L138-189、Output L1105-1143、recordCompletion L937-1016）
> 主题：1) 自动推进补存数据；2) 等待队列落点；3) 完成状态跟踪；4) LastJobID 语义演进；5) 两 map 并发安全

---

## 零、结论先行（TL;DR）

| # | 主题 | 裁决 | 一句话理由 |
|---|------|------|-----------|
| 1 | TeamTask 补存字段 | **新增 `JobID`、`Prompt`、`SessionID`、`CreatedAt`；`Owner`/`DependsOn` 已有；`Status` 补初值** | 自动推进需要在无用户输入的完成事件里重放 `Assign(ctx, owner, prompt)`，缺 prompt/session 无法重放；缺 JobID 无法表达「等待任务尚无 job」 |
| 2 | 等待队列落点 | **复用 tasks 列表（TeamTask.JobID 可空 + Status 枚举化），不另建 waiting map** | 单一真源、Tasks() 一张表可见 running/pending/blocked/done；另建 map 引入登记时序与双真源问题。**内存态**，重启丢失可接受（与 teammates 同生命周期，须文档明示） |
| 3 | 完成状态跟踪 | **tasks.Status 补存为主源，弃用 `jm.Output` 查询**；不需要独立完成缓存集合 | `Output` 是**消费性接口**（jobs.go L1105-1143 推进 readOffset/置 resultRead）；jm purge 后 `Output` 返回 ok=false → 依赖检查**悬空永久阻塞**（现存隐藏 bug）；tasks.Status 本身就是依赖树的完成缓存 |
| 4 | LastJobID 演进 | **单值语义在自动推进下成立**（严格匹配 guard 兜底），但**必须修复 tm.Ref 断点** | 现存断点：Assign L218 写回旧 ref → 首轮 fork 后 tm.Ref 仍空 → 续轮走 fork 而非 continue（测试只覆盖首轮，未暴露） |
| 5 | 并发安全 | **teammates/tasks 共用一把 ts.mu 正确**，不拆锁；四处改进（Tasks 纯读化、依赖检查去 jm、OnJobDone 三禁、Remove 清 tasks） | 单锁保证两 map 一致性；OnJobDone 新入口必须禁调 ts.jm.* 防锁序环 |

**与 implementation-plan.md 的显式分歧（本评审独立裁决）**：implementation-plan T2 保持「jm.Output 优先、tasks.Status 兜底」（L87）；本评审建议**反过来**——tasks.Status 为主、jm 不再查询。理由：事件驱动下补存值是事件更新的真值；jm 查询既消费输出、又昂贵、又在 purge 后误导。若执行队坚持 jm 兜底，**至少不得用 `jm.Output`**（须新增 jobs 层非消费状态查询，见 §3.4）。

---

## 一、主题 1：自动推进需要补存的数据

### 1.1 现状事实

```go
// teammate_store.go L53-58
type TeamTask struct {
	ID        string
	Owner     string
	DependsOn []string
	Status    string
}
```

`recordTask`（L234-238）只写 ID/Owner/DependsOn，**Status 恒空串**；`Tasks()`（L265-284）的状态全靠 `jm.Output(id)` 派生，且 L279 在只读方法里**改写** `t.Status`。

### 1.2 自动推进的完整重放路径

完成事件（job A done）→ 找依赖 A 的任务 B → 调 `Assign(ctx, B.Owner, B.Prompt, A)` → B 的 job 启动。逐项核对：

| 数据 | 现状 | 裁决 | 锚点 |
|------|------|------|------|
| `Owner`（即 Assign 的 name） | ✅ 已有 | 保持 | L55 |
| `Prompt`（即 Assign 的 prompt） | ❌ 无 | **必须新增**。完成事件无用户输入，唯一输入源就是登记时快照的 prompt | Assign L165 `prompt` 参数 |
| `SessionID`（leader 会话） | ❌ 无 | **必须新增**。完成事件跑在 job goroutine 上，ctx 不可用；`Assign` 需要 `jobs.SessionFromContext` + `WithParentSession` + `WithForkSource` 才能启动续轮。**存 session id 字符串而非 ctx 对象**（ctx 是接口、不可序列化、可能带 deadline/values） | flushMailbox L225 `jobs.SessionFromContext(ctx)` 是现成来源；risk-review §2.5.2 的「存 leader 模板 ctx」方案不如存 id 干净 |
| `CreatedAt` | ❌ 无 | **建议新增**（低成本）。等待/超时/重试决策（blocked 后 backoff 上限）与 /team-status 展示都需要 | — |
| `Status` 初值 | 恒空串 | **必须补初值**：recordTask 写 `"running"`（implementation-plan T2 已定），终态由 OnJobDone 写、不可变 | L237 |
| `DependsOn`（forward） | ✅ 已有 | 保持 | L56 |
| **反向索引 `dependents`（depID → []taskID）** | ❌ 无 | **自动推进必需**但 MVP 可用全表扫描。OnJobDone 后要知道「谁在等 A」；当前只有 forward 依赖，需扫全表。团队规模小（个位数），O(n) 可接受；标注为推进 worker 内的优化点，不提前建索引 | L56 |

### 1.3 目标结构（评审建议）

```go
type TeamTask struct {
	ID        string    // 分配单 ID（等待任务用占位 ID，见 §2.3）
	JobID     string    // 实际 jobID；等待任务为空串
	Owner     string
	Prompt    string    // 新增：自动推进重放 Assign 的输入
	SessionID string    // 新增：leader 会话（Assign ctx 来源）
	CreatedAt time.Time // 新增：登记时间（超时/展示）
	DependsOn []string
	Status    string    // running/pending/queued/blocked/done/failed/...（终态不可变）
	BlockedReason string // 新增（可后置）：blocked 时携带 reason
}
```

---

## 二、主题 2：等待队列

### 2.1 两个候选

- **A：TeammateStore 独立 waiting map**（`waiting map[taskID]*TeamTask`）
- **B：复用 tasks 列表 + Status 字段**（TeamTask.JobID 空即等待态）

### 2.2 裁决：B（复用 tasks 列表）

理由：
1. **单一真源**：依赖树一张表看全生命周期（pending → running → done/failed），`Tasks()` 零改动展示等待态。
2. **生命周期一致**：等待任务与已派任务同生共死（Remove 清理、Tasks 展示、purge 兜底）；另建 map 引入「等待任务何时移入 tasks」的时序问题。
3. **无双份登记**：A 方案下同一任务要登记两次（waiting + tasks），状态同步是必然 bug 源。

### 2.3 等待态建模

- `JobID == ""` + `Status ∈ {pending, queued, blocked}` 即等待态。
- 登记时序（自动推进 worker，对齐 risk-review G5 状态机）：
  1. `recordTask(占位ID, owner, prompt, sessionID, dependsOn)`，Status=pending；
  2. 尝试 `Assign(ctx, owner, prompt, dep...)`；
  3. 成功 → 回填 `JobID` + `Status=running`；失败（gate 拒绝/teammate 占用）→ `Status=blocked` + `BlockedReason` + backoff 重试（3 次，risk-review G5）。
- **key 冲突注意**：现状 Assign 成功路径在启动后调 `recordTask(jobID, ...)`（L226）；等待预登记后，同一任务会有两个 key（占位 ID 与 jobID）。须统一为**一次登记**：占位 ID 即 TeamTask.ID，Assign 成功后回填 JobID 字段（不改 key），或 recordTask 对已知 ID 幂等更新。**禁止**占位 ID 与 jobID 双条目。

### 2.4 内存态 vs 持久化

现状：`teammates`/`tasks` 全内存（L46-47 注释明言 "in-memory; persistence is P6.1"）；仅 mailbox 有盘（inboxRoot）。

- **重启后**：teammates 空、tasks 空、等待任务全丢。**可接受**，前提三条：
  1. 团队是**会话级概念**（重启 = 新会话 = 重建团队）；
  2. **文档/notice 明示**：重启后未完成的依赖推进丢弃、等待任务不恢复，需重新 `/team-create` + `/team-add`；
  3. **不要单独给 tasks 做持久化**——要与 teammates 持久化**同批做**（P6.1 声明了 teammates 持久化），否则重启后「团队重建、tasks 残留旧 owner」错位。
- 折中（可选、MVP 不做）：TeamTask 复用 inboxRoot 同级目录做 JSON 快照（仿 mailItem L413-417 + PostMail 原子落盘模式 mailbox-design M3）。标注为 P6.1 持久化的一部分，不在本事务范围。

---

## 三、主题 3：完成状态跟踪

### 3.1 三方案对比

| 方案 | 查询 | 一致性 | 性能 | 风险 |
|------|------|--------|------|------|
| a. `jm.Output(id)` 直查（现状 L250/L272/L135） | 每次 Assign/Tasks/List 查 jm | 与 jm 一致但**消费输出** | 每次持锁+读 | **Output 消费语义 + purge 悬空**（见 3.2） |
| b. 独立完成缓存集合（`completed map[string]Status`） | O(1) | 需与 jm 同步 | 快 | 双真源；缓存生命周期难管（Remove/purge 何时清）；**冗余** |
| c. **tasks.Status 补存为主源（采纳）** | 纯内存读 `ts.tasks[dep].Status` | 终态不可变、事件写入后与 jm 一致 | O(1) 无 jm 锁 | 依赖 OnJobDone 事件不丢（destroy 窗口由 DestroyAll 兜底，同 taskRecorder 语义） |

### 3.2 方案 a 的两个硬伤（现状 bug，非风格问题）

1. **`Output` 是消费性接口**：jobs.go L1105-1143——`readArtifactSinceOffsetLocked` 推进 readOffset（L1121-1127）、对 task job 置 `resultRead`（L1132-1135）。teammate job 全是 `Kind="task"`（Assign L194-198 → task.go L995）。**每次 `/team-status`、每次 Assign 依赖检查、每次 List 懒同步都会消费 teammate job 的 result**。当前 result 已由 P1 信封投递（非静默）故「无害」，但语义错误，且封死未来任何依赖 `Output` 读结果的路径。
2. **jm purge 后依赖悬空**：jobs Manager 的 `jobs` map 在会话清理/加载路径会被清（destroying、loadSessionArtifacts 替换等）。purge 后 `Output(id)` 返回 ok=false → `pendingDependenciesLocked` 默认分支（L254-258）把**已完成的 dep 当作「未完成」** → 依赖永久阻塞。**这是现存隐藏 bug**，事件驱动补存正是根治点。

### 3.3 为什么不需要方案 b（独立缓存集合）

`tasks.Status` 本身就是「job X 完成状态」的缓存，且与依赖树生命周期绑定（Remove 清 tasks 时自然清）——再建独立 completed map 是双真源冗余。**结论：完成状态 = 依赖树自身的 Status 字段，不新增任何缓存结构。**

### 3.4 一致性规则（评审建议）

1. `recordTask` 写初值 `Status="running"`（记录时快照）。
2. `OnJobDone` 写终态，**终态不可变**（done/failed/killed/interrupted/cancelled 写入后不再改；running 是唯一可变值）。
3. `pendingDependenciesLocked`（L243-262）改为**纯内存读 `ts.tasks[dep].Status`**：终态集合放行，其余阻塞。**锁内不再调 jm**（消除 ts.mu → jm.mu 锁序，见 §5）。
4. `Tasks()`（L265-284）以 tasks.Status 为主源，**改为纯读**（不再 L279 改写）；jm 侧仅在 `Status==""`（未登记/未知 job）时降级，且**降级不得用 `Output`**——jobs 层现无纯状态查询 API（只有 Output/Snapshot/Wait），建议补一个非消费 `Status(id)`（Snapshot L1381 已返回 status 字符串、不消费，可直接复用其模式）。
5. `syncStateLocked`（L131-138）的 `jm.Output(LastJobID)` 同样改为读 `tasks[LastJobID].Status` 或非消费接口。

---

## 四、主题 4：LastJobID 语义演进

### 4.1 现状语义

`Teammate.LastJobID`（L31）：单值、「最近一次派活 job」。三处消费：
- `syncStateLocked`（L131-138）查 job 终态翻转 idle；
- `Complete` 严格匹配 guard（L309-311，旧 job 事件不误置 idle）；
- `TeamStop`/`Remove`（L328/L353）kill 当前 job。

### 4.2 自动推进下连续多任务的推演

- **单值语义成立**：同一时刻一个 teammate 至多一个活跃 job（Assign L173-176 拒绝 running 中再派）→ `State==Running` 时 LastJobID 唯一代表活跃 job；OnJobDone 置 idle → 下一轮自动 Assign 设新 LastJobID，自然演进。
- **乱序/迟到安全**：J1 完成事件晚于 J2 启动到达 → `LastJobID==J2 ≠ J1` → guard 挡住，不误置。✅ 现有机制正确，事件驱动沿用同一匹配（同 Complete L309-311）。
- **LastJobID 不清空**：置 idle 后保留（kill 审计 + 与 tasks 条目的关联），语义从「当前 job」演进为「最近 job」，活跃性由 `State` 表达。

### 4.3 前置断点：tm.Ref 必须修复（本评审新增发现）

**事实链**：
1. `Assign` L187 快照 `ref := tm.Ref`；L202 `if ref == ""` 决定走 fork 还是 continue。
2. `RunProfileSpec` 后台路径返回 out 含 `FormatSubagentReference(run)`（task.go L1040/1042），即文本 `Subagent reference: <ref>`。
3. `teammateJobID(out)`（L288-298）只解析 jobID；L218 `tm.Ref = ref` 写回的是 **L187 的旧值**。
4. 首轮 fork（`prepareTranscriptForkWithPrompt` → `PrepareParentFork` → `newRef`，subagent_store.go L908）产生**新 ref**，但 `tm.Ref` 写回 `""` → **首轮后 tm.Ref 仍空**。
5. 第二次 Assign：`ref==""` → 又走 fork 分支 → **续轮 continue 语义从未生效**（每次重新 fork，缓存全失）。
6. 现有测试 `TestTeammateAssignStartsBackgroundJobWithEnvelope` 只派一次，未覆盖续轮 → 断点未暴露。

**修复（数据模型层面）**：`Assign` 从 out 解析 `Subagent reference:` 后的 ref token，**首轮 fork 后 `tm.Ref = 新ref`**；续轮 continue 时 ref 不变，写旧值无害（同 transcript 续写）。可加 `teammateRef(out)` 解析函数（仿 `teammateJobID`）。

**与完成事件的关系**：`Complete(name, jobID, ref)`（L302）带 ref 参数、但**无调用者**，说明原设计意图是「完成时回传 ref 更新」；但 jobs 层 observer 回调只有 `(parentSession, id, kind, label, st, err)`（recordCompletion L937），**没有 ref**，teammate 侧也拿不到 job goroutine 内的 `run.Ref` → **ref 更新必须走 Assign 解析路径，不走完成事件**。文档化，防执行队在 observer 里找 ref 而不得。

---

## 五、主题 5：并发安全

### 5.1 现状

`teammates` 与 `tasks` 共享一把 `ts.mu`（L61）。所有入口（Create/List/Status/SetSink/Assign/recordTask/pendingDependenciesLocked/Tasks/Complete/TeamStop/Remove/PostMail/notifyMail/DestroyAll）均持锁。**单锁 = 两 map 一致性由构造保证，这是正确且简单的选择；不建议拆成两把锁**（拆锁引入跨 map 事务与锁序问题）。

### 5.2 评审发现的问题与改进（4 处）

**P1（高）锁内调 jm——`pendingDependenciesLocked` L250 `ts.jm.Output(dep)`**：
- 现状锁序：Assign 持 `ts.mu` → `jm.mu`。
- 新增 OnJobDone：从 jm 无锁点进入 → 持 `ts.mu` 写 tasks/teammates。
- 两方向不构成环的**前提**：OnJobDone 内**绝不调 ts.jm.\***（否则 jm 无锁点 → ts.mu → jm.mu 重入）。依赖检查改为纯内存读（§3.4.3）后，ts.mu 临界区不再触碰 jm，锁序彻底解耦。
- **三禁文档化**（对齐 risk-review G1/G2）：OnJobDone 内①禁调 `ts.jm.*`、②禁持 `ts.mu` 做 IO（ReadDir/notify 放锁外，沿用 notifyMail L402-410 模式）、③禁同步 Assign（自动推进必须 worker 化）。

**P2（中）`Tasks()` 读改写**：L279 `t.Status = st` 在「只读」方法里写 map 元素——有锁安全但语义脏。事件驱动后 Status 由事件维护，`Tasks()` 改纯读。

**P3（中）`Remove` 清理缺口**：L346-362 只 `delete(ts.teammates, name)`，不清理 `tasks` 中该 owner 的条目 → orphan 任务残留（/team-status 显示已删成员的任务）。`Remove`/`DestroyAll` 应清理 tasks（按 Owner 匹配删除；running 的 orphan job 照常跑完，完成事件写不存在条目即 no-op）。

**P4（低）`syncStateLocked` 消费 jm.Output**：L135 同 §3.4.5，事件驱动后降级为兜底，改读 tasks.Status 或非消费接口。

### 5.3 锁序总表（评审定稿）

| 路径 | 锁序 | 说明 |
|------|------|------|
| Assign | ts.mu（短临界）→ 释 → jm.StartForSession → 释 → ts.mu（回填 LastJobID） | L167-222，启动在锁外 ✅ |
| OnJobDone（新增） | jm 无锁点 → ts.mu（写 tasks/teammates）→ 释 → 锁外 ReadDir/notify | 禁调 jm，禁同步 Assign |
| pendingDependenciesLocked | ts.mu 内**纯内存读** | 改后不触 jm（P1） |
| notifyMail | ts.mu 取 sink 指针 → 释 → 锁外 Emit | L402-410 ✅ |
| 依赖检查 | 无 jm 交互 | 悬空 bug 根治（§3.2） |

---

## 六、对抗自检（devil's advocate）

1. **「tasks.Status 为主源，observer 未接线时全空」**——攻击：boot 接线前（测试/旧路径）Tasks() 展示空状态。缓解：未登记/Status 空时降级为**非消费** jm 快照（§3.4.4）；测试路径显式 SetJobDoneObserver；boot 接线是 T3 必做项，生产路径无窗口。
2. **「等待任务预登记后 Assign 失败，任务悬空」**——攻击：B 预登记 pending，teammate 被并发占用 → gate 拒绝 → B 永远等待。缓解：`Status=blocked` + `BlockedReason` + backoff 重试 3 次（risk-review G5）；3 次后 Tasks() 显式展示 blocked，**绝不静默**；重试由推进 worker 串行消费，无并发残留。
3. **「占位 ID 与 jobID 双条目」**——攻击：预登记占位 ID，Assign 成功后又 recordTask(jobID)，同一任务两个条目。缓解：§2.3 明确一次登记、Assign 成功后回填 JobID 字段；recordTask 对已知 ID 幂等。测试固化（Tasks() 无重复）。
4. **「ref 文本解析脆弱」**——攻击：`Subagent reference:` 文本变动 → 解析失败 → 续轮回退 fork。缓解：fork 回退 fail-safe（正确性在、缓存失）；解析函数单测固化；更稳的方案是 RunProfileSpec 返回结构化结果（超出本事务，标注）。
5. **「终态不可变 vs 手工改状态」**——攻击：外部把已 done 任务改回 running。缓解：OnJobDone 只写终态、且**先校验当前非终态才写**（幂等）；Tasks() 纯读后无其它写路径。
6. **「本评审越界到自动调度」**——攻击：数据模型加 Prompt/SessionID 是在为自动调度铺路，超出 implementation-plan「MVP 不自动 Assign」边界。缓解：**字段补存 ≠ 自动调度**；本评审只定「存什么」，调度决策（何时 Assign、worker 化）仍按 implementation-plan/risk-review 边界执行。防执行队把字段扩展误读为授权自动 Assign。

---

## 七、缓存/纪律检查点（Reasonix 领域）

- ✅ **发送侧字节零变化**：全部改动在 `teammate_store.go` 内存态结构（TeamTask/Teammate 字段、读取路径），不触碰 schema/system prompt/transcript/task.go 发送路径。
- ✅ **前缀稳定**：tm.Ref 修复（首轮存新 ref）**恢复**而非破坏「续轮 continue 前缀稳定」的既定语义（P6 team plan 三要素纪律）；不引入任何新发送前缀结构。
- ✅ **运行值不进 schema/system prompt**：prompt/sessionID/createdAt 均为内存态登记数据，非发送前缀一部分。
- ✅ **锁序纪律**：OnJobDone 从 jm 无锁点进入、ts.mu 临界内不碰 jm/不做 IO（P1-P4 已列）；防死锁、防重入。
- ✅ **防虚假完成**：本评审只产出结论与锚点，不宣称任何实现完成；改动与否由执行队按 implementation-plan 决定，纪律团独立审查。

---

## 附：执行队禁止事项（按章程补充）

- 不得在 `Tasks()`/`pendingDependenciesLocked`/`syncStateLocked` 继续用 `jm.Output` 做状态查询（消费语义 + purge 悬空）；若需 jm 兜底，必须新增非消费 `Status(id)`。
- 不得在 OnJobDone 内调用 `ts.jm.*`、不得持 ts.mu 做 IO、不得同步 Assign（三禁）。
- 不得新建独立 waiting map 或独立 completed 缓存集合（§2.2/§3.3 已否决）。
- 不得把本评审的「自动推进数据字段」解读为「自动调度授权」（边界同 implementation-plan）。
- 不得声称完成——须附证据（测试输出/文件 diff）。
