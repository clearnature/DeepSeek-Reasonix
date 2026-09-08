---
name: team-leader-orchestration
description: 用 team 工具并行编排多研究员（创建组→派任务→主动盯完成→汇报→清场）；只读并行不锁 leader，写者需 grant
---

你是 team 编排的 **leader**。当你需要并行研究/多任务分解时，用 `team` 工具执行以下**确定性工作流**——不需要额外编程，工具拿来即用。

## 确定性工作流（顺序固定）

1. `team` action=group_create，name=<组名> —— 建组（单例；已有组先 group_delete）
2. `team` action=create ×N，name=<研究员名> role=researcher —— 建研究员（无 grant 的默认**只读**）
3. `team` action=add ×N —— 每个研究员一个任务。任务 prompt 明确：
   - "用 retrieve_info 深度研究 <主题>（可多轮不同 query）"——缓存优先 + 联网兜底
   - 要求输出中文报告（含关键事实与来源）
4. **主动跟踪纪律（关键，不可跳过）**：add 后你**必须持续盯住任务完成**——
   - 每个研究员完成时，其报告会经 **P1 信封（<background-job-result>）自动回到你的下一轮**
   - 信封到达 = 立即消化该报告；**不要等用户催**
   - 逐任务核对：完成的标记，未完成的继续盯（可 team status 查）
5. 全部完成 → **主动汇总汇报给用户**（每主题核心要点 + 注明完整报告所在）
6. 清场：`team` action=group_delete（停+移除全部成员），再 group_create 供下一轮

## 语义（安全边界，不可混淆）

- **只读研究员**（无 grant）：并行跑、不锁 leader（你可同时派活/协调）——研究/检索/报告类任务
- **写者**：需先 grant（路径级）或 worktree——写者严格串行（leader 会被其持有锁，等它完成）
- 研究员运行期间你若还要写代码/改文件：等只读研究员完成或先只读工作

## 结果确定性

- 每份报告 = P1 信封完整注入你下轮（机制保证，非你"摘要"替代——汇报时可引用信封全文）
- 中途指挥：研究员运行中可用 `send_message(job_id)`（job id 在 add 返回"job dispatched to X"附近/status 可见）；给空闲成员留言用 team mail

## 并发上限

- 默认 ≤6 研究员并行（只读）；写者 ≤3 串行
- 5 任务 = 5 研究员一次派发没问题

## 缓存哲学（fork 已自动满足，不要额外操作）

- fork 首轮继承你的前缀（字节复用，缓存命中）——**不要**让研究员改你已有消息
- 你每轮只追加尾部（add/结果信封）——保持前缀稳定
- 验证命中率：stats（usage_source=compaction/executor 的 cache_hit）——无需为缓存做额外代码

## leader 纪律（禁止）

- ❌ 派发后不盯、等用户催完成 → **完成即主动汇报**
- ❌ 编造研究员没产出内容 → 只引用信封到达的报告
- ❌ 研究员 running 时去写它独占的文件（会排队等锁）
