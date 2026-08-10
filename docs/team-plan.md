# Team 多智能体协作：落地规划

> 生成：2026-08-10 · 分支：dev/clearnature · 基线：main-v2（07fcf585b）· 关联 memory：team-multiagent-qwen-comparison
> 支撑材料：`docs/MULTIAGENT_QWEN_COMPARISON.md`（Qwen 源码级验证）、CCB（claude-code-best）团队功能研究报告

## 一、背景

社区 issue [#7962](https://github.com/esengine/DeepSeek-Reasonix/issues/7962)（open）抱怨后台子代理交互模型：
- 后台子代理运行期间父代理 write 类工具被 write-claim 互锁阻塞（`scheduler.go:187`），静默半冻结
- 子代理间被整工作区 claim 串行化（两个后台任务实际串行，快任务反而后完成）
- 用户中途转向输入排队/丢弃，无法指挥运行中的子代理（`admission_guard.go`、`run_loop.go:282-285`）
- 后台结果不自动投递，需 `wait`/`bash_output` 轮询（`jobs.go:778-814` 仅 Notice + 一行摘要）

2026-08-10 用 fleet 双智能体研究 Qwen Code Agent Team + CCB（claude-code-best）实现，得出对比与阶段化规划。

## 二、三方对比增量结论

| 机制 | Qwen | CCB | 我们的差距 |
|------|------|-----|-----------|
| 结果投递 | mailbox 500ms 轮询注入 `<teammate_message>` | `<task-notification>` 命令队列 → 下一条 user 消息注入（`wrapCommandText` 加前缀） | 两者都优于我们（仅一行摘要） |
| 中途指挥 | `send_message(task_id)` 轮次边界投递 | `queuePendingMessage` → 下轮 `queued_command` 注入 | 我们无输入通道 |
| 前台→后台动态降级 | 无 | sync agent 2s 提示 / 120s 自动转后台（signal-race 平滑续跑） | 我们无（CCB 独有） |
| fork 前缀优化 | 继承父上下文 | 占位 tool_result 保证 byte-identical 前缀 + 复用父 rendered system prompt | 我们无（P4 已验证 + 实现技巧） |
| 写冲突 | 无互锁（文件锁 + 原子写） | 文件锁 + high-watermark + 原子 claim + `agent_busy` 检查 | 我们整工作区互锁（最保守） |
| 停止 | task_stop / 协作关闭 | kill → 未完成任务退回 pending + leader 收 `teammate_terminated` | 我们 kill_shell，缺友好 stop 与任务回收 |

## 三、六阶段规划

| 阶段 | 内容 | 借鉴 | 落点（我们代码） | 缓存影响 | 验证 |
|------|------|------|-----------------|---------|------|
| **P1 自动投递** | job 完成 → `<background-job-result>` 信封注入父下一条 user 消息（bounded 正文+引用） | CCB 命令队列 + Qwen mailbox | `jobs.go:778-814` 构造信封 → `input.go:191-195` 升级注入 | 🟢 append turn 尾部 | effect test + 单测 |
| **P2 任务管理** | 后台任务面板（列表/stop/进度）+ kill 后任务回收 | CCB BackgroundTasksDialog | desktop + stopTask 等价物 | 🟢 | desktop UI |
| **P3 steer 通道** | job 加输入队列，`send_message(task_id)` 子代理轮次边界消费 | CCB queuePendingMessage + Qwen waitForMessages | `jobs.go` 队列 + 子代理 run-loop 消费 | 🟢 不影响父前缀 | 单测 |
| **P4 前台→后台降级** | sync agent 运行中可转后台平滑续跑（120s 自动） | CCB signal-race（独有） | LocalAgentTask 等价物 | 🟢 | 单测 |
| **P5 fork** | fire-and-forget 子代理继承父前缀，占位 tool_result 保字节一致 | CCB 占位 + Qwen fork | 新 fork 子代理类型 | 🟢🟢 省 80%+ token，与第一原则同向 | 前缀稳定测试 |
| **P6 team 组织** | leader+teammate、共享任务列表、mailbox、TeamCreate | Qwen + CCB 文件锁体系 | 最大工程，最后做 | 🟢（teammate 独立会话） | 集成测试 |
| **P7（决策项）** | 运行时按实际写路径收窄 write-claim | Qwen 无互锁 | `scheduler.go` | 🟡 需拍板 | 设计评审 |

## 四、依赖关系与执行顺序

```
P1 ──→ P2 ──→ P3 ──→ P4 ──→ P5 ──→ P6
（投递）  （管理）  （指挥）  （降级）  （fork）  （team）
```

- **P1/P2/P3** 是同一交互模型的三个面（投递/管理/指挥），社区 #7962 核心诉求，全部 🟢 缓存安全——**先做**
- **P5（fork）** 独立且与缓存哲学同向（前缀稳定 = 省钱），**可并行启动**
- **P6（team 组织）** 依赖 P1-P3 的消息/任务基座，最后做
- **P7** 是哲学决策（安全 vs 协作），单独拎出，动 `scheduler.go` 互锁前必须设计评审

建议：本周先在 team 分支做 P1（最小改动：`jobs.go` + `input.go`），并行启动 P5（fork）设计（利用 CCB 占位 tool_result 技巧）。

## 五、哲学红线（不可违反）

1. 所有发送侧改动先问「会不会改变前缀字节？」会 = 破坏 prefix-cache = 否决或缓存收益补偿
2. 自动投递/steer 只能 append 在 turn 尾部（位置固定），绝不插入历史
3. profile 名、动态参数、worktree 路径等运行时值绝不进入 tool schema 或父 system prompt
4. P7（窄化 claim）若采纳是「安全 vs 协作」路线分叉，必须设计评审 + 用户拍板
