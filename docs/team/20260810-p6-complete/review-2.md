# 纪律审查报告：P6 完成事件驱动（teammate 侧，commit 8f76af02f）

> 审查对象：`internal/agent/teammate_store.go`、`internal/agent/teammate_store_test.go`、
> `internal/agent/teammate_done_e2e_test.go`、`internal/jobs/jobs.go`（交叉验证）、
> `internal/boot/boot.go`（生产接线）。
> 裁决依据：`docs/team/20260810-p6-complete/arbitration.md`（以本记录为准）。
> 审查方式：静态代码验证 + 时序推演 + 测试逻辑逐条复核（本环境无命令执行工具，
> 无法重跑 `go test`/`go test -race`，回归命令已列于文末由执行队提交前执行）。

## 一、审查项逐条

### [PASS] fable5 合规（执行队是否跳步）
- 任务分解/拓扑扫描：仲裁 1-7 全部有实现落点；仲裁 7（tm.Ref 断点）明确标注不并入，与仲裁记录一致，未混入。
- 多路径推演：完成事件的四条路径（正常 done / startInvalid Failed / Kill Killed / destroy 吞事件）在
  `jobsevent-design.md` R-a~R-e 与 `lifecycle-design.md` 表项 3 均有覆盖，实现与事件源时序吻合
  （jobs.go L675 `recordCompletion` → L1008/L1034 observer → L677-680 status flip → L686 close(j.done)）。
- 自我验证/沙盒验证：`teammate_store_test.go`（T2 系列 10 个）+ `teammate_done_e2e_test.go`（T4 系列 4 个）
  存在且逻辑自洽（逐条复核见 §三）；`jobs_done_observer_test.go`（J 系列）覆盖 observer 机制本体。
- **未跳步但留痕不足**：killed 语义矛盾（见发现 1）与测试缺口（见发现 5）执行队未在交付物中报告。

### [PASS] 幻觉检测（证据真实性——静态交叉验证）
- **时序陷阱验证成立**：jobs.go L675 `recordCompletion`（内含 `fireJobDoneObservers`）在 L677-680
  `j.status = st` 之前执行 ⇒ observer 触发时 `j.status` 仍 Running、`j.done` 未关。
  父代理修复（`pendingDependenciesLocked` 去掉 `jm.Output`）**方向正确**：若回调内反查
  `jm.Output` 必读到 Running（误判"未完成"→ 门永堵），且 `OutputForSession`（L1181/L1189）
  推进 `readOffset`、置 `resultRead`，是一次性消费读取。
- **消费性表述微瑕**（INFO）：P1 envelope 由 `recordCompletion` 用 `boundedResult` 预渲染入队
  （L978-992），`Output` 不再触碰 `m.completed`，故"steal the P1 envelope tail"在信封层面不精确；
  但消费 job 级 result/tail 会使后续 wait/bash_output 读空——修复动机仍然成立，仅注释措辞可精化。
- **测试证据真实性**：10+4 个测试均走真实 job 全链或直接注入 `HandleJobDone`（确定性路径），
  断言与实现逐条对应（如 mailbox 唤醒断言绑定 `Running→Idle` 翻转），无 `t.Skip`、无空断言。
  编译符号全部可解析（mockProvider/sequencingProvider/releaseBlockingProvider/teamAssignCtx 均有定义）。

### [PASS] 缓存红线（前缀字节稳定）
- 发送侧零字节变化：本实现是进程内回调 + `event.Notice`（UI 层），不进 provider 输入；`m.completed`
  入队与 closing Notice 输出未改（jobs.go recordCompletion 入队/Emitting 段逐字节不变）。
- 自动 Assign 的 prompt = `recordPendingTaskLocked` 登记时原文（`cur.Prompt` 快照，autoAssign 原样回放）；
  teammate fork/continue 前缀逻辑未动（Ref 由 `teammateRef` 回填，arbitration 7 独立跟踪）。
- profile 名/动态值未进 schema：SessionID 以字符串入 `TeamTask`，ctx 每次重建，无 schema 污染。

### [PASS] 回归（静态 + 待执行）
- 静态层面：未破坏既有导出符号；`SetJobDoneObserver` 为构造后单槽接线，全仓仅 `NewTeammateStore`
  一处调用（grep 核实），不覆盖任何既有 observer；`Complete`/`TeamStop`/`List` 等旧 API 保持兼容。
- 待执行队提交前跑：
  - `go test ./internal/jobs/ ./internal/agent/ -count=1`（重点 `TestDrainMultiple`、`TestDestroySession*`、
    `TestStartSilent*`、本次 teammate 全部测试）
  - `go test -race ./internal/jobs/ ./internal/agent/`（锁序/并发护栏，覆盖 R1 骨架测试）
  - `go test ./internal/boot/`（生产接线：NewTeammateStore + SetSink）

### [FAIL] 对抗自检（devil's advocate 攻击发现的缺陷）
见 §二 发现 1-5。核心：killed 完成事件触发后继自动推进，与生命周期设计裁决（arbitration 3
引用的权威）"自动推进不触发"直接矛盾；另有 Remove 竞态双条目、注释-实现不符、测试缺口。

## 二、发现清单

### 发现 1（HIGH，须父代理裁决）killed 完成事件触发后继自动推进 ≠ 生命周期设计裁决
- 实现：`HandleJobDone`（teammate_store.go L430-474）的 ready 扫描**不区分 `st`**；
  `isTerminalStatus` 含 `Killed`，`depsTerminalLocked` 亦含 `Killed`。⇒ 被 kill 的 job
  （如 /team-stop）若被其他 teammate 的 pending 任务依赖，后继**会被自动 Assign**。
- 权威文本：`lifecycle-design.md` 表项 3（arbitration.md L29 明确引用为 killed 语义裁决）：
  "④**自动推进不触发**（用户主动终止，自动重启后继 = 覆盖用户意图）——「放行依赖门（手动可继续）」
  与「自动推进（自动不继续）」是两回事，必须显式区分"。
- 矛盾判定：实现只落实了"依赖门放行"（isTerminalStatus 含 Killed），未落实"自动推进不触发"。
- 仲裁 3 正文（"完成事件到达 → 找出 dependsOn 全部终态 → 自动 Assign"）字面支持全终态推进，
  与 lifecycle 表项 3 冲突——**父代理裁决文本自身有内部矛盾**，实现取其一且未上报。
- 必须动作（二选一）：
  a) 若裁决"killed 不自动推进"：`HandleJobDone` 在 ready 扫描前对 `st == jobs.Killed` 跳过推进
     （依赖门照常放行），并补 H6/C4 测试钉住"Killed → 后继不自动 Assign、Tasks() 可见 pending"；
  b) 若裁决"全终态推进"：修订 lifecycle 表项 3 与 arbitration L29 措辞，并补测试钉住 killed 推进语义。
- 现状无任何测试覆盖该分支 ⇒ 语义漂移无护栏。

### 发现 2（LOW）`pendingDependenciesLocked` 注释与实现不符
- 注释（L323）称 "the manager is only consulted as a fallback for deps that were never registered"，
  但函数体（L324-344）**从不 consult `ts.jm`**——未注册 dep 一律 `st == ""` 走 default → 永不放行。
- 行为本身安全（仲裁 3 缓解 2：环/洞死等，无死循环），属保守选择；但注释误导后续维护者
  （会以为有 fallback 查 jm）。修：删掉 fallback 措辞或按注释补查询（后者需注意 Output 消费性，
  应使用只读的 status 查询路径，勿用 `jm.Output`）。

### 发现 3（LOW）autoAssign 与 Remove 竞态产生双条目（违反"exactly one entry per task"）
- 触发链：HandleJobDone 扫描出 ready 后、autoAssign 的 `Assign` 执行前，依赖 owner 被 `Remove`
  （Remove 删除其**含已终态**的全部 tasks 条目）→ `Assign` 内 `pendingDependenciesLocked` 查到
  依赖消失 → `recordPendingTaskLocked` 注册**新** placeholder → 返回 error → autoAssign 将**旧**
  placeholder 标 blocked。结果：同一逻辑任务出现 pending + blocked 两个条目。
- 无死循环（新 pending 的依赖永不终态）；两条目均 Tasks() 可见；触发窗口极窄（并发 Remove）。
- 建议：autoAssign 失败路径若发现"因依赖消失而 gate 拒绝"，不依赖 `Assign` 内部重复注册
  （或在 autoAssign 内删除新注册的孤儿 placeholder）。

### 发现 4（LOW）boot.go 接线注释过时
- `internal/boot/boot.go` L1901-1904 注释称 "jm.SetJobDoneObserver(ts.OnJobDone) is intentionally
  deferred… E2 not landed"，但 E2 已合入且 `NewTeammateStore` 构造器内已注册（teammate_store.go L111）。
- 功能无影响（接线已生效），但注释会误导维护者以为 observer 未接。修：更新注释为"构造器已注册"。

### 发现 5（MEDIUM）测试矩阵缺口（test-strategy 声明 vs 交付）
- 缺失：H3 未知 job noop、H5 两级依赖链 a→b→c、H6 **Failed/Killed 依赖语义**、H9 关闭 jm 后
  Tasks 快照、R1/R3 并发竞态测试。
- 其中 **H6 与发现 1 直接相关**：killed 的 gate 放行与自动推进语义无测试固化，是本次最大测试盲区。

## 三、六大审查点结论

1. **仲裁 3/4/5 落实**：依赖自动推进（ctx 重建 + 原 prompt + 原 owner + 环死等 + owner 被删保留）
   已落实；mailbox 聚合唤醒（置 idle 后 countInbox>0 单次 Notice 计数 N、PostMail 即达即通知）已落实；
   严格 LastJobID 匹配 + 仅 Running→Idle 翻转已落实（含 stale-event noop 测试）。唯 killed 推进语义待裁决（发现 1）。
2. **父代理修复**：✓ 正确。`pendingDependenciesLocked` 以 tasks 快照为主源、不再查 `jm.Output`，
   时序陷阱（recordCompletion 先于 status flip）与 Output 消费性均已核实，修复消除了"门永堵 + 偷读结果"双重风险。
3. **autoAssign 幂等/owner 删除/锁序**：✓ 幂等推演通过（并发双推进两种交错均收敛到单条目）；
   owner 删除 → blocked + reason 保留；锁序——HandleJobDone 持 ts.mu 时零 jm 调用，唯一
   ts.mu→j.mu 嵌套在 syncStateLocked（方向唯一、无反向、安全）。
4. **assignContext 重建**：✓ SessionID 字符串、每次重建、不缓存 ctx、leader 存 *Agent 而非 ctx。
5. **Remove 清任务**：✓ 删除 teammates + 该 owner 全部 tasks（仲裁 6），完成事件对孤儿 job no-op。
6. **惰性 syncStateLocked 兜底**：✓ 保留于 List()，observer 未注册/时序窗口时仍能自愈。

## 四、结论

**驳回**（附必须修复项）。

主体质量高、父代理修复正确、仲裁 4/5/6 落实无瑕疵，但**存在一个实质语义缺口和一处竞态缺陷**，
且执行队未在交付物中上报即宣称完成（防虚假完成红线）：

1. **必须修复**：killed 完成事件的自动推进语义与 lifecycle 表项 3（arbitration 引用的权威）矛盾。
   父代理裁决二选一并落实代码/文档 + 补 H6/C4 测试钉住。
2. **必须修复**：autoAssign 与 Remove 竞态的双条目（发现 3）——至少加护栏，最好补并发测试。
3. **建议修复**：发现 2 注释与实现不符；发现 4 boot 注释过时；发现 5 补齐 H5/H6/H9。
4. **回归命令**（执行队提交前必须执行并附输出）：
   `go test ./internal/jobs/ ./internal/agent/ ./internal/boot/ -count=1` 与 `go test -race ./internal/jobs/ ./internal/agent/`。

待上述 1、2 落实且回归通过后复审放行。
