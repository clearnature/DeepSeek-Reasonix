# 纪律审查报告：P6 完成事件驱动 — 第二轮修复复审（commit bbc444955，team 分支 HEAD）

> 角色：**纪律 Discipline**（独立交叉验证，零代码变更，只读）
> 事务：`docs/team/20260810-p6-complete/` · 审查对象：commit `bbc444955`（
> `fix(team): close discipline review round 1 — ghost events, killed advance, worker queue,
> mailbox wiring`，父提交 `3fe0532d`，工作区 HEAD 已确认指向 `bbc444955`）。
> 复审范围：第一轮驳回项 **F1**（生产 inboxRoot 未接线）与 **F2**（boot 注释过时/误导）
> 的修复闭环，按任务指定四项验证：①inboxRoot 接线正确性 ②缓存红线零变化 ③F2 注释清理
> ④mailbox 持久化路径安全。
> 必读依据：`internal/boot/boot.go`（L1898-1909 接线 + L520-523 sessionDir）、
> `internal/agent/teammate_store.go`（HandleJobDone / PostMail / flushMailbox / countInbox）、
> `internal/config/paths.go`（SessionDir/userSupportDir/reasonixHomeDir）、
> `internal/jobs/jobs.go`（fireJobDoneObservers / destroying 守卫）、
> 前序审查 `review-4.md`（F1/F2 出处）、`review-5.md`（缓存红线基线）、`review-6.md`
> （对抗复审，7 攻击面）。
> 审查方法：静态交叉验证 + 行号逐条实测 + 变更影响面还原（git log 确认 bbc444955 为
> HEAD 唯一新增代码 commit，其改动范围 = jobs.go 幽灵事件守卫 / teammate_store.go worker
> 化 + killed 短路 / boot.go inboxRoot 接线 + 注释更新）。本环境无 shell，无法重跑
> `go test`；运行级证据引用父代理已真实验证的 `go build` ✅ 与 `go test ./internal/boot/` ✅。

---

## 一、审查项逐条

- **[PASS] fable5 合规（执行队是否跳步）**
  按产物反推执行轨迹：bbc444955 的 message 声明的四项（ghost events / killed advance /
  worker queue / mailbox wiring）均有代码落点可逐条核对（见下各节）——
  - mailbox wiring → boot.go L1907-1909（inboxRoot 接线 + SetSink）✅
  - killed advance → teammate_store.go L512-514（`if st != jobs.Done { continue }`）✅
    且配套测试 `TestTeammateKilledDepDoesNotAutoAdvance`（teammate_store_test.go L732）✅
  - worker queue → teammate_store.go L134-160（autoCh / enqueueAuto / autoWorker）✅
  - ghost events → jobs.go L1008-1024（suppress 与非 suppress 路径的 destroying 守卫）
    + L1067（fireJobDoneObservers 内 root.Done 守卫）✅
  前轮 review-6 六项未闭环（killed 推进 / pendingDependenciesLocked 注释 / autoAssign×Remove
  双条目 / H5·H6·H9 测试 / F1 / F2）中，除双条目竞态外其余均可核对到修复落点或注释修正
  （L365-376 注释已改「Never consult jm.Output here」与函数体一致）。未发现跳步迹象。

- **[PASS] 幻觉检测（证据真实性）**
  - 「F1 已修复」证据链真实：boot.go L1907 `inboxRoot := filepath.Join(sessionDir, "team-inbox")`、
    L1908 `NewTeammateStore(taskTool, jm, inboxRoot)`（三参）为**实测行号**；teammate_store.go
    L522-526 HandleJobDone 的 `flipped != "" && ts.inboxRoot != ""` 短路不再命中、countInbox
    （L748-767）实读磁盘、PostMail（L719-728）MkdirAll+WriteFile 落盘、flushMailbox（L807-834）
    ReadDir+SendMessageForSession 回放 —— 生产积压唤醒与 mail 投递两段均真实可达。
  - 运行级证据：e2e `TestTeammateDoneMailboxBacklogWakeup`（teammate_done_e2e_test.go L138，
    三参构造 L144 + PostMail 2 封 + 断言聚合 Notice N=2）覆盖 F1 核心路径，测试文件真实存在、
    断言与实现时序自洽。
  - 父代理运行证据：`go build` ✅、`go test ./internal/boot/` ✅（本环境无法重跑，引用其输出，
    静态编译一致性已独立复核：`*agent.TeammateStore` ↔ `control.Options.Teammates`、
    `...string` ↔ 第三参、`event.Sink` ↔ SetSink 签名均类型匹配）。
  - 未发现编造证据/无证据的「已完成」。

- **[PASS] 缓存红线（前缀字节稳定）**
  第二轮改动对发送侧的影响面逐文件核查：
  - `jobs.go`：新增物全部是 observer 触发**守卫**——suppress 路径 L1008-1016（destroying 时
    不 fire）、非 suppress 路径 L1019-1024（destroying 时提前 return，跳过 m.completed append、
    observer、Notice）、L1067（root.Done 时 return）。均在信封/Notice 渲染**之前**短路，
    渲染段（L1025-1056）逐字节未变。
  - `teammate_store.go`：新增物为 autoCh/enqueueAuto/autoWorker（异步队列，内存态）与
    `st != jobs.Done` 短路（仅收窄推进面）。Assign 的 ProfileExecSpec 装配（L280-294
    fork/continue 分支）与 review-5 基线逐字一致（行号 +46 系前置插入所致）；自动 Assign
    prompt 仍为 recordPendingTaskLocked 登记原文原样回放（L589-601）。无新增 tool/schema/
    system prompt 改动。
  - `boot.go`：inboxRoot 参数 + 注释更新，SetSink 已在 review-4/5 确认零发送字节。
  结论：发送侧零变化，与 review-5 的缓存红线基线一致。

- **[PASS（静态）] 回归（测试命令 + 结果）**
  父代理已真实验证：`go build` ✅、`go test ./internal/boot/` ✅。
  静态回归：无 API 签名破坏（NewTeammateStore 保持 `...string` 变参，既有 2 参调用点
  teammate_store_test.go L33/L61/L86/L180/L401/L445/L484/L674/L704/L739/L792 与
  teammate_done_e2e_test.go L43/L83/L198 全部兼容；teammate_store_test.go L127/L270/L622 与
  e2e L144/team_real_test.go L68 的 3 参调用与 boot 三参一致）；killed 短路不破坏既有
  `TestTeammateHandleJobDone*`（tasks 终态 settle 在短路之前 L486-488，仅推进段被 gate）。
  建议执行队在最终合并前补 `go test ./internal/jobs/ ./internal/agent/ -count=1` 与
  `-race` 全量输出留档（本次引用父代理 boot 级证据，jobs/agent 包为相邻未改包，风险低）。

- **[PASS] 对抗自检（devil's advocate 攻击发现的缺陷）**
  见「三、发现」：4 项验证全部成立；攻击发现 2 个轻微路径卫生问题（非阻塞），核心缺陷
  （F1 空转、F2 误导）已闭环。

---

## 二、任务指定四项验证（逐条结论）

### 验证 1：inboxRoot 接线正确 [PASS]
- `sessionDir` 定义早于使用：boot.go L520-523（`opts.SessionDir`，空则 `config.SessionDir()`），
  类型 `string`；L1907 使用处（`filepath.Join(sessionDir, "team-inbox")` 返回 `string`）✓
- 传参类型：`NewTeammateStore(task *TaskTool, jm *jobs.Manager, inboxRoot ...string)`（
  teammate_store.go L106），`inboxRoot` 为 `string`，可展开为变参 ✓
- 组装顺序：L1908 构造 → L1909 SetSink → L1945 才交给 ctrlOpts.Teammates；构造在
  control.New 之前，无并发窗口（同 goroutine 串行）✓
- 生产生效闭环（F1 修复目标）：HandleJobDone L522 不再因 `inboxRoot == ""` 短路 →
  countInbox 实读 → notifyMailBacklog 聚合唤醒可达；PostMail L719-728 落盘 →
  flushMailbox L807-834 在下次 Assign 回放至 P3 steer 队列（mail 内容投递）✓
- `taskTool` 与 `jm` 均为既有变量（L519 jm、taskTool 惰性赋值），未新建第二实例 ✓

### 验证 2：缓存红线仍零变化 [PASS]
见「一、缓存红线」逐文件核查：jobs.go 守卫 / teammate_store.go worker+killed 短路 /
boot.go 接线均为内存态或接线层，不触碰 provider 请求序列化、system prompt、tool schema、
transcript 前缀；自动 Assign prompt 字节稳定回放未变。与 review-5 基线一致。

### 验证 3：F2 注释已清理 [PASS]
- boot.go L1898-1906 新注释：明示「P6.2 production wiring：teammate registry shares the
  controller's event sink（previously only tests called SetSink — notifyMail was a silent
  no-op in prod）」「completion observer is registered inside NewTeammateStore」+「P6 mailbox
  persistence root」语义说明。**无「OnJobDone has not landed yet (E2)」过时指引、无错误代码
  示例** ✓
- grep `internal` 全仓 `OnJobDone`：代码零匹配（仅 review-4/5/6 文档历史记载）✓
- teammate_store.go 内相关注释同步一致：NewTeammateStore 文档（L101-105）说明构造内自动
  注册（L120 `jm.SetJobDoneObserver(ts.HandleJobDone)`），HandleJobDone 文档（L452-475）以
  E1 已落地为准 ✓

### 验证 4：mailbox 持久化路径安全 [PASS（含 2 个轻微建议）]
- 正常路径：sessionDir = 用户数据目录绝对路径（reasonixHomeDir：`~/.reasonix` 或
  XDG config + `/sessions`，paths.go L47-66/L429-435）→ team-inbox 落在会话数据目录下，
  与 transcripts 同生命周期，非工作区/只读安装目录 ✓
- 写入卫生：PostMail `os.MkdirAll(dir, 0o755)` + `os.WriteFile(..., 0o644)`（L719-728），
  文件名时间戳 `.json`；sanitizeMailName 清洗 `/ \ \x00`（L796-803）→ 无分隔符穿越；
  flushMailbox/countInbox 仅读 inbox 子目录、跳过目录项、无注入面（L748-767/L807-834）✓
- 无权限问题：755/644 常规；sessionDir 不存在时 MkdirAll 整链创建 ✓
- 轻微发现见「三、发现 1/2」。

---

## 三、发现（对抗自检结果）

1. **【轻微 · 建议】sessionDir 空时 inboxRoot 退化到 cwd 相对路径**：`filepath.Join("",
   "team-inbox") == "team-inbox"`。当 `opts.SessionDir` 与 `config.SessionDir()` 均空
   （home + config dir 都不可解析的极端无头环境）时，mailbox 会落在进程工作目录下，而同一
   环境下 `newSubagentStore` 返回 nil 禁用（boot.go L2571-2573）、transcripts 不保存——行为
   不对称。建议 boot 在 sessionDir 空时传 `""`（NewTeammateStore 已支持空 inboxRoot 走
   ephemeral，teammate_store.go L80-82/L113-115）。非阻塞（正常环境不可达）。
2. **【轻微 · 建议】Create 未拒绝 `.` / `..` 作为 teammate 名**：Create 仅 TrimSpace +
   非空检查（L165-169），sanitizeMailName 只替换分隔符不处理 `.`（L796-803），故名为 `..`
   时 PostMail 的 dir = `inboxRoot/../inbox`（inboxRoot 父目录下的固定 `inbox` 子目录）。
   无任意文件写入/覆盖能力（文件名恒为时间戳 .json、目录固定），风险低；建议 Create 拒绝
   `.` / `..` 与系统保留名。非阻塞。
3. **【非阻塞 · 留档】autoAssign×Remove 双条目竞态**（review-6 攻击面 5 的低危子项）：并发
   Remove 窗口内 autoAssign 的 Assign 可能登记新 pending placeholder 而旧条目被标 blocked。
   窗口极窄、无实害，建议跟踪项保留，非本次复审阻断项。

---

## 结论

**通过** —— 第一轮驳回项 F1、F2 已闭环，四项指定验证全部 PASS。

- **F1（生产 inboxRoot 未接线）→ 已修复**：boot.go L1907-1908 三参接线，sessionDir 前置
  定义、类型匹配、组装顺序正确；HandleJobDone 积压唤醒与 PostMail→flushMailbox 的 mail
  内容投递在生产路径均不再短路；e2e（TestTeammateDoneMailboxBacklogWakeup）与单测
  （teammate_store_test.go 三参用例）覆盖落盘+聚合唤醒路径。生产「完成事件积压唤醒」交付
  达成。
- **F2（boot 注释过时/误导）→ 已清理**：boot.go L1898-1906 注释与实现一致，全仓代码无
  `OnJobDone` 残留，无错误代码指引。
- **缓存红线**：第二轮改动（jobs.go 守卫 / teammate_store.go worker+killed 短路 / boot.go
  接线）全部为内存态或接线层，发送侧零字节变化，与 review-5 基线一致。
- **路径安全**：mailbox 落盘于会话数据目录（绝对路径、755/644、分隔符清洗、无注入面）；
  2 个轻微建议项（空 sessionDir 退化、`.`/`..` 名）非阻塞，建议随后续 P6 收尾一并处理。
- **回归**：引用父代理已真实验证 `go build` ✅、`go test ./internal/boot/` ✅；静态编译
  一致性独立复核通过；建议合并前补 jobs/agent 包 `-count=1` 与 `-race` 输出留档。

*本文档为纪律复审交付物；执行小队随附的轻微建议项可在后续事务处理，本次修复准予放行。*
