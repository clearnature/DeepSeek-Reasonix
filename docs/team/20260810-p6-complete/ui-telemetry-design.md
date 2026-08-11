# 设计：完成事件驱动「UI/遥测面」（独立评审视角）

> 事务：docs/team/20260810-p6-complete/ · 视角：独立评审（只读规划，不写代码）· 状态：设计交付物
> 范围：`internal/control/controller.go`（applyTeamCommand L6345-6408、input.go L188-195）、`internal/agent/teammate_store.go`（List L118 / syncStateLocked L131 / notifyMail L402 / Assign L165）、`internal/jobs/jobs.go`（recordCompletion L937-1016、closing Notice L1004-1015）、`internal/event/event.go`（Kind L25-119、Sink L827、Notice L45-47）、`desktop/app.go`（JobPanelView L11954 / JobPanelJobsForTab L12118）、`desktop/frontend/src/components/TaskMonitorPanel.tsx`
> 与事务既有文档的关系：implementation-plan（事件源方案 A + T1-T5）、risk-review（G1-G10）、datamodel-design（tasks.Status 主源）、interaction-design（C1-C8）、mailbox-design（唤醒子机制 + N 文案）。本文件补 **UI/遥测面**，不重推事件源方案。

---

## 零、结论先行（TL;DR）

| # | 主题 | 裁决 | 一句话理由 |
|---|------|------|-----------|
| 1 | 完成事件驱动的用户可见面 | **事件驱动让「可派活状态」更及时——这是它最大的用户可见价值；/team-status 的显示时机不变、显示正确性提升** | 懒同步下 teammate 完成后 `State` 仍 Running，`Assign`（L173-176）只查 State、不查 jm → **第二次 `/team-add` 会被拒**，leader 必须手动 `/team-status` 触发懒同步才能再派活；事件驱动 OnJobDone 主动置 idle → 完成即可复用。另根治 jm purge 后 State 卡 running 与 List 偷读消费（见 §1） |
| 2 | Notice 事件 | **完成/推进类 Notice 不发（零新发送）**；唯一例外 = mailbox 积压聚合 + 推进失败 | jobs 层 closing Notice（jobs.go L1004-1015）**已对正常 done 即时 emit** `"background task finished: <id>"`——再发完成 Notice 是第 4 个重复通道 + fan-out 风暴源；risk-review G7/§2.6 已裁决，mailbox-design 的 N 聚合是用户驱动例外（§2） |
| 3 | 遥测 | **完成事件打 slog 结构化日志**（agent 层 OnJobDone，固定字段 name/job/status/err）；**不新增 event.Kind、不新增 stats 指标** | event.Kind 是 wire 稳定协议（KindCount 前插 + golden）；UI 完成可见性已由 closing Notice + P1 信封承担；slog 与既有 teammate 生命周期日志（execution.md 观测性三件套）同构（§3） |
| 4 | desktop 任务面板 | **直接复用，不新建 TeamPanel** | teammate job 就是普通 `Kind="task"` job（Assign L194-198，label=`"teammate: <name>"`，task.go L891/L962）→ TaskMonitorPanel 的 JobPanelJobsForTab **已经展示 teammate 的 running/done/failed**，含色点/stalled 高亮/failed attention（§4） |
| 5 | 文档 | 本文件 = p6-complete 第 7 份设计；执行后补 **execution.md**、更新 **team-delivery-summary.md**（加 P6.2 完成事件驱动条目）、**p6-team/execution.md** 观测性节 | 交付总结当前只到 P6.1 + 遥测三件套，未含完成事件驱动（§5） |

**一条总纲**：完成事件驱动的 UI/遥测面边界 =「**零新发送（不注入 input、不发完成 Notice）+ 复用既有通道（closing Notice / P1 信封 / TaskMonitor / /team-status）+ 结构化日志兜底观测**」。越过此边界即引入重复通知或前缀漂移回归（risk-review 2.7 高）。

---

## 一、主题 1：完成事件驱动的用户可见面

### 1.1 现状：leader 感知 teammate 完成的四通道（已核实）

| 通道 | 载体 | 时机 | 是否 model 可见 | 锚点 |
|------|------|------|:---:|------|
| A. P1 信封 | `<background-jobs>` 块注入 leader 下一轮 input | 下轮 turn | ✅ | input.go L191-193；teammate 非 silent（L202）→ 必然送达 |
| B. jobs 层 closing Notice | `event.Notice`，UI 即时 | 完成瞬间 | ❌（UI 层） | jobs.go L1004-1015（done 也 emit，`shouldEmit` 时） |
| C. TaskMonitor 面板 | 5s 轮询 JobPanelJobsForTab | ≤5s | ❌ | app.go L12118；teammate job 属 leader 会话 → leader tab 可见 |
| D. /team-status | `c.notice` → Notice | 查询驱动 | ❌ | controller.go L6380-6393（roster + job=<id>） |

**结论**：leader 感知「teammate 完成了」已有**两条自动通道 + 两条查询/面板通道**，事件驱动不增加任何感知通道——它改变的是**状态机推进的时机**（见 1.2）。

### 1.2 事件驱动真正改变的用户可见面：「可派活状态」的及时性（本评审核心论点）

**事实链**：
1. `Assign`（teammate_store.go L173-176）只查 `tm.State` 拒绝 running 中再派活，**不查 jm、不触发懒同步**；
2. 懒同步 `syncStateLocked`（L131-138）**只在 `List()`（L118-128）被调用时执行**（/team-status 的唯一入口）；
3. 因此现状下：teammate 的 job 完成 → 信封已达 leader（model 知道完成了）→ leader 直接再 `/team-add alpha` → **`Assign` 拒绝 "running; steer it or wait"**——除非 leader 先敲一次 `/team-status` 触发 List 翻转 State，才能再派活。

**这是懒同步架构在用户可见面最大的缺陷：完成后不可直接复用，必须先查一次状态。** 完成事件驱动（OnJobDone 主动置 idle，`LastJobID == id` 严格匹配，同 `Complete` L309-311 语义）让「完成 → 可再派活」的间隔从「等一次查询」收敛为「事件到达」——**这就是「State 何时变 idle 让 /team-status 显示更及时」的准确答案：显示时机不变（查询驱动），可派活时机显著提前**。

### 1.3 事件驱动对 /team-status 显示正确性的三项提升

| 项 | 现状缺陷 | 事件驱动后 |
|----|---------|-----------|
| jm purge 后 State 卡 running | `syncStateLocked` 用 `jm.Output`，purge 后 ok=false → 不翻转 → 永久显示 running（datamodel-design §3.2 隐藏 bug） | OnJobDone 主动写终态，`syncStateLocked` 降级兜底，State 不依赖 jm 存活 |
| List 偷读消费 | `syncStateLocked` 的 `jm.Output` 是**消费性**查询（推进 readOffset / 置 resultRead，jobs.go L1124-1135）→ 每次 /team-status 偷读 leader 后续增量（interaction-design 主题 3 中） | 事件驱动后该路径降级为兜底，应迁移到非消费查询（interaction-design C7） |
| 依赖门判定 | `pendingDependenciesLocked` 依赖 `jm.Output(dep)` 实时查 | `tasks[id].Status` 由事件补存为主源（datamodel-design §3.4），`Assign` 的依赖检查纯内存读、purge 后不悬空 |

### 1.4 缺口：依赖链推进无展示入口（建议补 /team-status 依赖段）

**现状**：`Tasks()`（teammate_store.go L265-284）持有依赖树数据，但 **slash 命令无任何展示入口**——/team-status 只渲染 roster（L6386-6392）。leader 无法看到「B 在等 A」（waiting_on）、无法看到 blocked 原因。risk-review §2.2.5 已点名此缺口（"死等护栏：/team-status 显示 waiting_on"）。

**建议（本设计 U2）**：/team-status 在 roster 后追加依赖段，渲染 `Tasks()` 每条目：`<jobID> owner=<name> status=<st>`，blocked/等待态附原因（复用 `pendingDependenciesLocked` 的 reason 文本）。输出仍走 `c.notice`（Notice 事件，UI 层）→ **零发送、前缀零影响**（render.go operator audience 只控 UI 转发，risk-review L167）。

---

## 二、主题 2：Notice 事件——完成 Notice 该不该发

### 2.1 决定性事实：完成 Notice 是第 4 个重复通道

jobs 层 closing Notice（jobs.go L1004-1015）**对正常 done 已 emit**：`"background task finished: <id>"`（LevelInfo，`shouldEmit` = active 会话匹配或父会话为空）。teammate job 非 silent → 走正常分支（L990-995 信封 + L1000-1002 RecordDone + L1013-1015 closing Notice）。

因此「完成事件驱动的完成 Notice」= 与既有 closing Notice 逐字节重复 + 与 P1 信封（内容）重复 + 与 TaskMonitor（可视化）重复。

### 2.2 裁决：不发，与 risk-review 协调一致

- **G7 零新发送（risk-review L174-178）**：MVP 完成事件完全静默推进，leader 感知靠既有通道；
- **§2.6（risk-review L147-154）**：「推进状态**不新增任何 Notice**——统一由 /team-status + 既有 `<background-jobs>` 承担。零新发送 = 零风暴」；
- **interaction-design C3**：「OnJobDone 禁 emit 完成/推进类 Notice（零新发送 G7）；唯一可选通知 = mailbox 唤醒」；
- **mailbox-design §4.2/§7.1**：mailbox 积压聚合（`teammate X has N unread mail`）是**用户驱动数据**的例外，已显式标注与 risk-review §2.6 的偏差待纪律团仲裁——**若纪律团否决 N，回退无 N 文案，不影响本设计**。

### 2.3 刷屏边界（与 risk-review 协调的完整清单）

| 发送源 | 是否新增 | 限流 | 风暴风险 |
|--------|:---:|------|---------|
| jobs 层 closing Notice | 既有 | 每 job 1 条（无额外限流） | fan-out 场景 1→10 = 10 条**既有**行为，非事件驱动引入 |
| P1 信封 | 既有 | ≤8 条/轮 + 16KB/块（jobs.go L229-233） | 受 drain 限流 |
| 完成/推进 Notice | **禁止新增** | — | 唯一可消除的风暴源 |
| mailbox 积压聚合 | 用户驱动（低频） | 每完成至多 1 条、仅积压>0 才发 | 无（无人 PostMail 则 0 条） |
| 推进失败 Notice | 稀有事件 | 一次一条（失败不聚合防丢失，risk-review L154） | 无 |

**结论**：完成 Notice 不发；mailbox 聚合与推进失败是仅有的两个例外，均非「完成类」流水通知。

---

## 三、主题 3：遥测——完成事件打日志/事件流

### 3.1 现状遥测三件套（execution.md L71-75 / delivery-summary L63-67）

1. `REASONIX_DEBUG=1`：slog debug 级（fork 前缀字节、steer 注入、mailbox flush、teammate 生命周期）；
2. teammate 生命周期日志（**Info 常开**）：created / assigned(job, fork_first, depends_on) / removed / stopped；
3. stats usage 落盘：`cache_hit_tokens` + `prefix_hash`（fork 首请求命中验证唯一通道）。

### 3.2 裁决：完成事件打 slog 结构化日志（agent 层），不新增 event.Kind、不新增 stats 指标

**日志**（在 `TeammateStore.OnJobDone` 内，agent 层）：
```go
slog.Info("team teammate job done", "name", name, "job", id, "status", string(st), "err", err)
```
- 与既有 teammate 生命周期日志（created/assigned/removed）**同构**，补齐「生命周期日志」闭环（assigned → done → removed）；
- 固定字段（name/job/status/err），**无时间戳**（slog 自带）、无随机段 → 输出稳定；
- 被过滤的未知/旧 jobID（G8：未登记 job 完成）走 `slog.Debug`（防每 job 一条 Info 噪音——observer 是全仓单例，**所有**后台 job 都经过它，teammate 是少数）；
- mailbox 积压唤醒并入 done 日志字段（`"mail_backlog", n`）或单独 Debug，不新增 Info 条数。

**在哪层**：jobs 层 observer **不打日志**（保持通用性，防每 job 噪音）；`taskRecorder`（controller.go L695，TaskMonitor 消费）不重复；日志点唯一 = `TeammateStore.OnJobDone`。

**事件流**：**不新增 `event.Kind`**——Kind 是 wire 稳定协议（event.go L25-119，新 Kind 须插 KindCount 前 + 全套 golden/coalesce 测试，mailbox-design §4.2 已裁决）；完成对 UI 的可见性已由 closing Notice（UI）+ P1 信封（model）承担，event 流无需第三个表达。

**stats**：不新增 team 指标——stats usage 是 provider 侧遥测（`cache_hit_tokens`/`prefix_hash`），与 team 生命周期无关；续轮缓存收益的验证通道仍是既有 stats（datamodel-design §4.3 的 Ref 修复后，第二次 `/team-add` 的 cache_hit 即事件驱动的间接证据）。

### 3.3 观测价值

- 排障：完成事件日志让「信封已达 / State 未翻转 / 依赖门放行」三者的时序可核对（execution.md 快速排障流程第 2 步 grep 模式加 `team` 已覆盖）；
- 审计：`LastJobID == id` 严格匹配的迟到/乱序完成事件（datamodel-design §4.2）在日志中可见（err 为旧 job 时 Debug 标记）。

---

## 四、主题 4：desktop 任务面板（TaskMonitor）可复用性

### 4.1 决定性事实：teammate job 已在任务面板中

- teammate job = 普通 `Kind="task"` job（Assign L194-198 `TaskSpec{Description: "teammate: " + name}`）→ job label = `firstNonEmpty(spec.Task.Description, ...)` = `"teammate: <name>"`（task.go L891/L962）;
- `TaskMonitorPanel` 经 `JobPanelJobsForTab(tabID)`（app.go L12118，内部 `Controller.JobSnapshots` 非消费）→ `JobPanelView{ID, Kind, Label, Status, Stalled, Tail}`（L11954-11966）已渲染 **teammate 的 running/done/failed**；
- tab-scoped（按 tab 的 controller 过滤，L12085-12100）→ teammate job 属 leader 会话 → **leader 的 tab 面板可见**。

**结论**：不需要新面板。teammate 完成在桌面的可视化 = 既有 TaskMonitor 面板行（色点 + kind badge + tail + stalled 高亮 + failed attention）。

### 4.2 可复用资产清单（供未来 team 展示扩展）

| 资产 | 位置 | 复用方式 |
|------|------|---------|
| 非消费投影模式（JobPanelView ← JobSnapshot） | app.go L11954-12142 | 任何新的 teammate 状态走桌面必须沿用（P2 HIGH 红线：不消费 readOffset/resultRead） |
| STATE_CONFIG 色点 + `isTerminalStatus` 终态优先 | TaskMonitorPanel.tsx L25-103 | 依赖树 queued/blocked/waiting 状态展示可直接复用该渲染映射 |
| `onJobAttention`（stalled/failed → transcript notice，去重 Set） | TaskMonitorPanel.tsx L218-232 | teammate job failed 已自动通知 leader transcript——「teammate 失败感知」零成本已覆盖 |
| 5s 轮询 + expandedRef 竞态修复范式 | TaskMonitorPanel.tsx L147/L297-300 | 未来任何轮询展示沿用（P2 修复的前端竞态范式） |

### 4.3 缺口与边界

- **依赖树桌面无展示**：TeamTask（pending/queued/blocked/done/failed）是 team 中心概念，TaskMonitor 是 task/job 中心视图。MVP 用 /team-status 文本段（§1.4 U2）承担；若产品要求桌面化，扩展 TaskMonitorPanel 加「team 段」复用行渲染（或独立 TeamPanel）——**标注为 P6.2/后续，不在本事务**；
- **roster（谁在队里 / 谁 idle）无桌面展示**：同属后续；/team-status 已是文本真源。

---

## 五、主题 5：文档——docs/team/ 下需要更新哪些

### 5.1 现状（p6-complete 目录，glob 核实）

已有 6 份设计：`datamodel-design.md` / `implementation-plan.md` / `interaction-design.md` / `mailbox-design.md` / `risk-review.md` / `test-strategy.md`。**注意**：implementation-plan L29 写作时断言「仓内无 test-strategy 文档」，现已存在（后补）——执行队引用时以现状为准。

### 5.2 本事务交付

- **`docs/team/20260810-p6-complete/ui-telemetry-design.md`**（本文件）＝第 7 份设计。

### 5.3 执行后需新增/更新（U6）

| 文件 | 动作 | 内容 |
|------|------|------|
| `docs/team/20260810-p6-complete/execution.md` | **新增**（当前不存在） | 执行证据链：测试输出 / 文件 diff / 数字（章程要求 executor 产出） |
| `docs/team/20260810-team-delivery-summary.md` | **更新** | 交付总览表加「完成事件驱动（P6.2）」行（jobs observer + OnJobDone + boot 接线 + mailbox 唤醒）；§三观测性补「完成事件日志」；§四缓存红线补「零新发送（完成 Notice 不发）已落实」；§六下一步建议删除已完成的 P6.2 mailbox 唤醒项 |
| `docs/team/20260810-p6-team/execution.md` | 小改 | §观测性工具补完成事件日志字段（`grep -E 'team'` 已覆盖，仅补说明） |
| `test-strategy.md` | 核对 | 若已含 T1-T5 测试清单，补 UI/遥测用例（无完成 Notice 断言、slog 字段稳定、/team-status 依赖段） |
| `REASONIX.md` / 用户文档 | **不改** | /team-status 语义未变（仅增依赖段），无用户文档契约变化 |

---

## 六、任务分解（交付 executor 队；前置依赖 implementation-plan T1-T3）

> 前置：**implementation-plan T1**（jobs 层 `WithJobDoneObserver`）与 **T3**（boot.go `SetSink` + `SetJobDoneObserver` 生产接线）合入后，本设计 U2-U5 才可完整验证。U1 与 T1-T3 无依赖。

- [ ] **U1**（前置确认，无代码）：核实 T3 生产接线已落地——boot.go L1930 拆变量 + `teammates.SetSink(sink)` + `jm.SetJobDoneObserver(ts.OnJobDone)`；notifyMail 生产路径不再空转（mailbox-design M5）
  → 验证: `grep -n 'SetSink\|SetJobDoneObserver' internal/boot/boot.go`；`go test ./internal/control/` 零失败
- [ ] **U2**：/team-status 追加依赖段——roster 后渲染 `ts.Tasks()`（jobID / owner / status），等待/阻塞任务附原因（复用 `pendingDependenciesLocked` reason 文本）；输出走 `c.notice`（零发送）
  → 验证: 造 A→B 依赖（`Assign(ctx, b, prompt, aID)` 后 A 未完成）→ /team-status 显示 B blocked 原因；A 完成 → 再次 /team-status 显示 B 可派活；`go test ./internal/control/` 零回归
- [ ] **U3**：`OnJobDone` 完成事件日志——`slog.Info("team teammate job done", name, job, status, err)`；未登记/旧 jobID 走 `slog.Debug`；mailbox 积压并入字段或 Debug
  → 验证: 单测断言 Info 日志字段固定（name/job/status/err，无时间戳/随机段）；未知 jobID 完成仅 Debug 级
- [ ] **U4**：TaskMonitor 复用确认（无代码）——真实/测试环境跑 teammate 后台 job，断言 TaskMonitorPanel 显示 `teammate: <name>` 行、完成转 done 色点；failed job 触发 onJobAttention
  → 验证: `desktop/frontend/src/components/TaskMonitorPanel.test.tsx` 零回归；手动真实测试（execution.md 测试表 P6 行扩展）
- [ ] **U5**：UI/遥测测试清单（映射 test-strategy 分层）：
  - 单元：完成事件日志字段稳定（无时间戳）；/team-status 依赖段文本
  - 集成：完成事件后**无新增完成类 Notice**（sink 记录断言：完成事件仅既有 closing Notice，teammate 侧 0 条完成 Notice）；mailbox 聚合例外（mailbox-design M2 已有）
  - 竞态：`go test -race ./internal/jobs/ ./internal/agent/ ./internal/control/`
  - 全量：`go test ./...` + 四验证（gofmt/vet/repolint/test）
  → 验证: 全部绿；无新发送断言（sink 记录比对）
- [ ] **U6**：文档更新（§5.3 表）——execution.md 新增、delivery-summary.md 补 P6.2 条目、p6-team/execution.md 观测性节、test-strategy.md 核对
  → 验证: 文件存在 + repolint + 内容与实现一致（纪律团审查）

### 依赖关系

```
T1/T2/T3（implementation-plan，事件源 + 接线）
 └─► U1（接线确认）→ U3（日志）→ U5（测试门）
U2（/team-status 依赖段）不依赖 T1，可与 U1 并行
U4（TaskMonitor 复用）零代码，仅验证
U6（文档）收尾
```

---

## 七、对抗自检（devil's advocate）

1. **「closing Notice 只在 shouldEmit 时发——leader 非 active 会话时完成无 UI 即时通知」**——攻击成立：`shouldEmit = active=="" || parentSession=="" || active==parentSession`（jobs.go L997），leader 切到其它 tab 时 teammate 完成不发 closing Notice。缓解：感知退化为 **TaskMonitor（按 tab 的 controller 过滤，跨 tab 仍显示，app.go L12085-12100）+ P1 信封（下轮必达）**，仍覆盖两条通道；且「不 active 会话无即时通知」是 P1 既有语义，非事件驱动引入。**标注为薄弱环节 ①**（本设计依赖「closing Notice 覆盖即时完成感知」的论证在跨会话场景弱化）。
2. **「/team-status 追加依赖段改变命令输出文本 → 前端 Notice 渲染测试断言可能挂」**——Notice 文本无 golden 绑定（event golden 覆盖 Kind 枚举，非 Notice 自由文本）；前端 Notice 测试按文本子串断言、/team-status 输出无既有断言锚点（grep 未发现）。风险低；U5 加回归兜底。
3. **「完成事件日志含动态 jobID → 日志不稳定」**——slog 是进程内输出、不进 provider input、不进 event 流（§3.2），无前缀影响；jobID 动态值是日志审计的必要信息（排障第 2 步 grep 依赖它）。非缺陷。
4. **「TaskMonitor 已展示 teammate 的论证依赖 label=Description 的契约」**——已核实三处锚点（Assign L194-198 Description + task.go L891/L962 firstNonEmpty → label），且 JobPanelView 展示即使 label 变化，status/kind 仍正确。低风险。
5. **「依赖段 waiting_on 需要反向索引，O(n) 全表扫描」**——teammate 个位数规模，O(n) 可接受；datamodel-design §1.2 已将反向索引标注为推进 worker 内优化点，U2 不做索引。
6. **「U2 是否越界（MVP 无自动 Assign，依赖段展示有无意义）」**——依赖段是**展示**现状（Tasks() 数据已存在、只是无出口），不授权自动调度；datamodel-design §2 已裁决「等待队列复用 tasks 列表」→ 展示 blocked/pending 是完成事件驱动让 Status 补存后的**直接收益**，非调度行为。
7. **「本评审自身未覆盖：完成事件日志与 taskRecorder 重复记录」**——taskRecorder（TaskMonitor 消费，controller.go L695）已记录 teammate job 完成到 `.reasonix/tasks`；slog 日志与其**职责不同**（运行时观测 vs 持久化任务视图），不重复、不同层；日志点唯一性声明（§3.2）指 jobs 层 observer 不打，非禁止 taskRecorder。

**薄弱环节排名**：① 跨会话时 closing Notice 不触发（感知依赖 TaskMonitor+信封）→ ② /team-status 文本变化的前端回归 → ③ mailbox N 聚合与 risk-review §2.6 的偏差（mailbox-design 已标注，纪律团仲裁）。

---

## 八、缓存/纪律检查点（Reasonix 领域）

- ✅ **发送侧字节零变化**：本设计全部改动在 `controller.go`（/team-status 文本，走 Notice）+ `teammate_store.go`（slog 日志）+ 文档；不触碰 schema/system prompt/transcript/task.go/subagent_store；/team-status 输出进 `event.Notice`（UI 层，不进 provider input，risk-review L167）。
- ✅ **前缀稳定**：完成事件零新发送（不注入 input、不发完成 Notice、不自动 PostMail/steer）；mailbox 唤醒 Notice 只进 `event.Sink`；无新发送前缀结构、无历史插入。
- ✅ **运行值不进 schema/system prompt**：name/job/status 均为 Notice 文本与 slog 日志，非发送前缀一部分。
- ✅ **锁序纪律**：本设计不新增锁路径；日志在 OnJobDone 内、ts.mu 释放后或 O(1) 内存段内（G2）；/team-status 依赖段读 `Tasks()`（纯读化后，datamodel-design §5.2-P2）。
- ✅ **防虚假完成**：本文件为零代码变更的设计交付物；U1-U6 的「完成」= 测试输出 + 四验证 + execution.md 证据链，纪律团独立审查。

---

## 附：执行队禁止事项（按章程补充，UI/遥测面）

- 不得在完成事件驱动中新增任何「完成/推进类」Notice（§2.2；唯一例外 = mailbox 聚合与推进失败，且须经纪律团批准）。
- 不得新增 `event.Kind` 或 stats 指标承载完成事件（§3.2；wire 稳定协议 + 职责边界）。
- 不得在 jobs 层 observer 内打 Info 日志（§3.2；日志点唯一 = TeammateStore.OnJobDone）。
- 不得新建 TeamPanel 桌面组件（§4；MVP 复用 TaskMonitor + /team-status 文本，桌面依赖树展示须另行立项）。
- 不得把 U2 依赖段展示解读为自动调度授权（§7.6；边界同 implementation-plan）。
- 不得声称完成——须附证据（测试输出/文件 diff）。
