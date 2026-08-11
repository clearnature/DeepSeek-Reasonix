# Team 多智能体功能交付汇总（P1-P6.1）

> 日期：2026-08-10 · 分支：team（基点 main-v2 07fcf585b）· 已推送到个人 fork
> 范围：11 个功能 commit（226365254 → cc20c9aca），198 文件 / +16,916 行
> 流程：团队章程（3 职能分离 + 规划/执行/纪律）+ fable5 九原则 + 独立审查

## 一、交付总览

| 阶段 | 功能 | 核心 commit | 关键文件 | 验证 |
|------|------|-----------|---------|------|
| **P1** | 后台结果自动投递（`<background-job-result>` 信封 + bounded） | a9b85c6e7 | jobs.go | 信封/部分 drain/16KB 块单测 + e2e |
| **P2** | 任务管理面板（jobs 快照 + desktop 面板 + /status 明细 + session-scoped stop） | cde444833 | JobSnapshot/desktop/chat_tui | 快照非消费锁定 + 前端 23 测试 |
| **P3** | steer 通道（send_message 工具 + /task-message + run-loop 注入） | 180444dab | jobs 队列/run_loop/bgjobs | 队列/注入/FIFO/no-op 测试 |
| **P4** | 前台→后台降级（/background + 120s 自动后台化 + 现场续跑） | 0f1bd5e08 | BackgroundizeSignal/交接 | 检查点/交接/续跑测试 |
| **P5** | fork 机制（子代理继承父前缀，首请求缓存命中，省 token） | aa143c1f4 | subagent_fork.go/fork_gate.go | byte-identical + cache_hit e2e |
| **P6** | team 组织（teammate 注册 + fork-assign + /team-* slash） | 28b668551 | teammate_store.go | 注册/派活/信封 e2e |
| **P6.1** | 增强（writable gate + 磁盘 mailbox + 依赖树完成门） | ef0cca13a | teammate_store.go | mailbox flush + 依赖门测试 |
| **遥测** | REASONIX_DEBUG + teammate 生命周期日志 | fdd22b702 | main.go/teammate_store | build + repolint |

## 二、逐项功能要点

### P1 后台结果自动投递
- job 完成 → `recordCompletion` j.mu 临界区预渲染 `<background-job-result>` 信封（单真源）
- `DrainCompletedNoteForSession`：部分 drain（≤8 条/轮余留下轮）+ 16KB 块上限（丢最旧 + `<result-overflow>`），input.go 零改动
- bounded：4096 字节/条 rune 安全 + `[truncated…]` 计入预算
- 缓存红线：只改新 user turn 的 `<background-jobs>` 块内容，稳定前缀零变化

### P2 任务管理面板
- `JobSnapshot`（9 字段）+ `JobSnapshotsForSession`（非消费 tail，锁序 m.mu→j.mu 不嵌套）
- Controller.JobSnapshots + CancelJob 收紧 KillForSession（session 边界）
- desktop bridge JobPanelJobsForTab/JobOutputForTab + TaskMonitorPanel 增强（kind badge/tail/stalled 高亮）
- /status 任务明细段（向下兼容）
- **修复的前端竞态**：useEffect 依赖 expanded → fetchJobs 覆盖 fetchJobOutput 的 tail（expandedRef 修复）

### P3 steer 通道
- `Job.pendingMessages`（16 条/8KB，j.mu）+ SendMessageForSession（Running + Kind=='task' 才收，超限 ErrPendingQueueFull）
- run_loop consumeSteer 后经 jobCtxKey 消费（父/前台/planner no-op），每轮 1 条
- send_message 工具（父可见/子代理隐藏）+ /task-message slash
- **契约同步**：acceptsDefaultSnip + boot 工具列表 + TOOL_CONTRACT + golden

### P4 前台→后台降级
- `BackgroundizeSignal`（幂等 one-shot）+ runToolLoop 检查点 sentinel + 交接串行点（同 goroutine：MarkRunning → StartForSession → resume 续跑同一 Session）
- 自动阈值：`REASONIX_AUTO_BACKGROUND_MS`（默认 120s/0 禁用）+ ablation.AutoBackground
- **独立审查修复 2 个 blocking 缺陷**（见 §五）

### P5 fork 机制
- `captureForkPrefix`（父 system + 历史截断去未完成轮次 + cloneForkMessages 深拷贝）——byte-identical 硬前提
- task 加 `fork` 参数（互斥 continue_from/fork_from）+ forkReadOnlyGate（只读，schema 全量保 ToolsHash）
- fire-and-forget：StartSilentForSession（P1 信封静默）+ 大小 guard（≤80% 窗口）+ 递归 guard
- **收益**：fork 首请求前缀 = 父前缀 → 命中父已建缓存（T0-C 确认 system/tools/messages 三者一致）

### P6 team 组织
- TeammateStore：注册（name/role）+ 三要素快照（model/effort/toolset）+ 派活管线
- 首轮 = 非静默 fork（继承 leader 前缀 + P1 信封回 leader）；续轮 = continue_from（前缀稳定）
- /team-create /team-add /team-status /team-remove + destroy 钩子
- 惰性状态同步（List 时查 job 终态置 idle）

### P6.1 增强
- **writable gate**：forkReadOnlyGate{writable} 参数化（可写 teammate 选项，默认只读保 P5）
- **磁盘 mailbox**：PostMail 落盘（inboxRoot）+ Assign 后 flush 进 P3 队列（重启不丢）
- **依赖树完成门**：Assign 支持 dependsOn，未完成拒绝（gate 检查在锁内防死锁——修复了 pendingDependencies 自死锁）

## 三、观测性（遥测三件套）

- `REASONIX_DEBUG=1`：slog debug 级（fork 前缀字节、steer 注入、mailbox flush、teammate 生命周期）
- teammate 生命周期日志（Info 常开）：created/assigned(job, fork_first, depends_on)/removed；依赖门拒绝 Debug
- stats usage 落盘：`cache_hit_tokens` + `prefix_hash`（fork 首请求命中验证通道）

## 四、缓存红线合规（全部）

1. 所有投递/steer 只 append turn 尾部（位置固定），零插入历史、零重写 canonical
2. fork 捕获零发送、父 Session 零改动；子代理首请求 byte-identical
3. teammate 轮消息只 append 自身 transcript；leader 前缀零变化
4. 运行值（name/role/任务文本）不进 schema/system prompt；fork 参数静态 bool
5. send_message 加入 provider 工具面 = deliberate 一次性打穿后稳定（golden 更新）

## 五、独立审查发现的缺陷与修复

| 缺陷 | 严重度 | 修复 |
|------|--------|------|
| P4 后台任务被父 turn 生命周期绑架（turn 结束杀 job） | blocking | job runCtx 只基于 jobCtx（detach 父 turn），测试改 TestBackgroundizeSurvivesParentTurnCancel |
| P4 sentinel 主 agent 无捕获者（长对话 turn 报错） | blocking | WithForegroundTask 标记只注入前台 task；主 agent/后台/planner 恒 false；子代理继承阈值 |
| 前端 locale 回归（zh 系统 label 变中文） | 严重 | 测试固定 navigator.language=en-US |
| 前端竞态（fetchJobs 覆盖 tail） | 严重 | expandedRef 修复 |
| P6 依赖门自死锁（ts.mu 重入） | 严重 | pendingDependenciesLocked（调用方持锁） |

## 六、下一步建议

1. **真实发布验证**：按 execution.md 的测试表跑一遍真实场景（task/steer/background/fork/team），重点确认 fork 首请求 cache_hit > 0
2. **P6.2 增强**：mailbox 磁盘唤醒（未运行消息转投）、task_stop/paused（Qwen）、teammate↔teammate 直连
3. **合并决策**：team 分支 11 commit 是否并入 dev/clearnature（cherry-pick 主题 commit，按 PR 更新纪律）
4. **上游 PR**：按「上游 PR 不可预期」纪律，不主动跟进；本地 dev/team 为真正成果
