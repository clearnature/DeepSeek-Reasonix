# Team 实时反馈：per-task 自动处理轮设计（2026-09-09，存档待实现）

> 状态：设计定稿，未实现（需新会话完整实施）。目标 = qwen 每任务通信对等：
> leader 可中途介入（修正长程任务/检查偏离/派新任务），通信是必要质量开销。

## 问题（实证）

- 任务完成 → P1 信封**被动**：leader 下次 turn 组装才 drain（input.go:201）→ 无界延迟（leader 空闲时）
- 组完成 Notice 修复后 UI 实时，但**模型侧**仍等轮——无法中途介入（qwen 能）

## 目标机制

```
每任务完成（HandleJobDone）
  → 信封 + 完成上下文 注入 leader 自动处理轮（无需用户回来）
  → leader 消化：检查偏离 → 可 send_message 修正 / 派新任务 / 继续等
  → 回 idle，下一完成再自动醒来
```

## 定案点（实现前必须解决）

| 点 | 挑战 | 方向 |
|---|---|---|
| auto-turn 触发 | controller 无用户输入自动发起模型轮 | 完成事件 → 会话自动轮队列 |
| 上下文/成本 | 每完成一轮 → transcript ×N 膨胀 | 优先：单一"编排等待态"（qwen wait 对等——leader 派发后保持可注入，完成批量/串行消化进同一上下文），非 N 独立轮 |
| 防重入/风暴 | 多任务相继完成 | 串行消化队列 + 等待期合并 |
| 注入时机 | leader 恰在用户轮/工具循环 | 排队 → 下一可用轮 |
| 防循环 | 自动轮不能自触发派发风暴 | 自动轮标记（只消化/修正，后台派发受 gate）|
| 缓存红线 | 注入内容位置 | 只 append turn 尾（位置固定），禁插历史——保持前缀稳定 |

## 关联

- 现修复基线：组完成通知（8d624e957 + 081c47e3c）、只读并行授权（efb44d125/f9131ad43/6e20d8db7）
- qwen 源码对照：TeamManager.waitForTeammateActivity / pollLeaderInbox / leaderMessageCallback（500ms）
- P1 信封 = 结果确定性回 leader（已有）；缺 = 完成→模型处理的自动触发

---

## 六、实现规格（v1，可执行——新会话照此实施）

### 6.1 触发链（函数级伪代码）

```
teammate 任务 job done
  → jobs observer → TeammateStore.HandleJobDone（已有）
    → per-task：信封写 leader 完成 note 队列（已有 P1）
    → 组完成：checkGroupCompleted emit Notice（已实现 081c47e3c）
    → 新增：ts 完成事件 → completion callback → Controller
      装配：NewTeammateStore 后 SetCompletionCallback(func(...))（agent 包新方法）
      boot: ts.SetCompletionCallback(c.onTeamCompletion)  ← 接线点

Controller.onTeamCompletion(group string, done int, total int)
  → c.mu 查 running turn？若 turn 活跃：postpone（完成事件入队，turn 尾注入——input.go compose 处已有 drain 信封）
  → 若无 turn：enqueueAutoDigest(group)（有界队列 chan，单 worker goroutine 串行）
autoDigestWorker:
  → 取队首 → 构造 digest prompt（读完成 note 队列 DrainCompletedNoteForSession 全文）
  → 发起 auto turn：executor 跑（复用 runGoalLoop 无 display 变体 or headless Run）
      prompt = "<team digest>: <group> 有 N 个任务完成，信封如下：\n<notes 截断>\n请汇总报告；如需修正/补充可派新任务（受 gate）"
  → 产出 append 当前 session transcript（位置固定尾，缓存安全）
  → digest 轮运行标记置位（防 digest 轮再触发 digest——除非轮内派新任务，其完成重新排队）
```

### 6.2 队列与并发

- `autoDigestCh chan string`（容量 16）+ 单 worker（autoWorker 同模式）
- **合并**：worker 取队首后把当前排队全部 drain 合并成一次 digest（同批 N 完成 = 1 轮，控转录膨胀）
- digest 轮运行中再有完成 → 入队（下次轮）
- 用户新消息 vs digest 轮：用户 turn 优先（digest 轮开始前查 c.running/用户 pending）

### 6.3 防重入/防循环

- digest 轮内 ctx 带 `digestOnly` 标记：只读/汇总；**默认禁止派发后台任务**（模型要派新任务需显式？v1：digest 轮禁止 add，改派等用户/下轮）
- worker 单飞 + doneCh 关闭守卫（Close 后不触发，仿 autoWorker）

### 6.4 缓存安全（红线）

- digest 输出 = 正常 turn 产出 append transcript 尾（位置固定）——**不插入历史、不改前缀**
- digest prompt 携带信封摘要（截断 ≤N tokens，超限写"见本地信封 task-X.log"——不把全文灌上下文）
- 前缀 = 既有 append-only 语义，缓存逐轮命中保持

### 6.5 装配点

| 文件 | 改动 |
|---|---|
| internal/agent/teammate_store.go | HandleJobDone 完成时调 completion callback（组完成处已有检测）|
| internal/agent/team_completion.go（新）| SetCompletionCallback + 触发 |
| internal/control/auto_digest.go（新）| enqueueAutoDigest + worker + digest prompt 构造 + turn 启动 |
| internal/boot/boot.go | ts.SetCompletionCallback(c.onTeamCompletion)（teammate_assembly 处）|
| internal/control/input.go | 用户 turn compose 时若队列非空 → 顺带消化（信封已在 drain 处）|

### 6.6 测试清单（守卫）

1. agent：完成事件触发 callback（2 成员完成 → callback 收 done/total）
2. control：enqueueAutoDigest → worker 起 digest turn（fake executor 断言 prompt 含信封摘要）
3. 合并：3 完成入队 → 1 次 digest 轮（fake 计数）
4. 防重入：digest 轮中完成 → 排队不并发
5. 缓存：digest 产出 append 尾（messages 尾位置断言——prefix 不变）
6. 装配 boot：SetCompletionCallback 接线（effect 测试——真实 Build 断言回调触发）

### 6.7 风险与边界

- 费用：每 digest 轮 = 模型 token（接受——用户判定必要通信）
- 转录膨胀：合并 + 摘要截断控制
- digest 轮失败：重试 1 次 + 落 notice（失败不吞——留信封下次轮可再 digest）
- 与用户轮竞争：用户输入优先，digest 排队
