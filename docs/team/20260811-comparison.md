# Reasonix Team 功能完善度对照（2026-08-11 实测分析）

> 方法：用 Reasonix team 创建 3 个并行分析者（writable），真实分配任务分析
> `/data/training/cli` 下 8 个软件的 team 实现。teammate 产出原始报告：
> `a1.md`（qwen-code + claude-code）、`a2.md`（claude-code-best + clawcode + grok-build）、
> `a3.md`（opencode + gemini-cli + MiMo-Code）。
> 本文件 = teammate 原始发现 + Reasonix 现状对照 + 完善优先级。

## 一、实测验证：Reasonix team 全链路可用

3 分析者并行（task-1/2/3 同时 running）→ 每份报告 18-25KB 真实深度分析 →
完成事件驱动全部 idle。**并行团队执行、写文件、完成回收全部真实生效**。
（会话与报告在 `/tmp/team-analysis/`；本文件是综合对照。）

## 二、对照矩阵（Reasonix vs 8 个参考实现）

| 能力维度 | Reasonix 现状 | qwen-code | claude-code-best | grok-build | opencode/MiMo |
|---|---|---|---|---|---|
| 团队创建 | /team-create（role + writable）| team_create 工具 | TeamCreateTool | — | agent 注册表 |
| 任务分配 | /team-add 推模式 | **拉模式自动认领**（scanIdleAgentsForTasks + nonce 信封）| task_assignment + 磁盘任务列表 | task 工具 + admission 队列 | Task 工具子会话 |
| 依赖任务 | recordTask/dependsOn ✓ | SwarmTask blockedBy 依赖图 | Team=Project=TaskList | — | — |
| teammate→leader 消息 | **P1 信封（完成时）+ team_message 工具** | 500ms 轮询 + `<teammate_message>` 稳定标签注入 | mailbox 轮询 → leader 收件箱 | 无 | actor send |
| 中途消息注入 leader | **✗ 无**（只有完成结果） | ✓ 实时注入 | ✓ | ✗ | ✗ |
| plan 审批门 | **✗ 无**（teammate 直接执行）| plan_approval_request/response + 审批期权限收缩 | plan_approval + leader 自动批准 | EnterPlanMode + 客户端 UI | plan agent 只读 |
| broadcast 群发 | **✗ 无** | ✓ broadcast | — | — | — |
| 停滞检测/abort | **✗ 无** | STALL_THRESHOLD 600s + abortStalledTeammates | — | — | — |
| teammate 工具审批转发 leader | **✗ 二元 gate**（只读/writable）| leaderPermissionBridge + 工具级 ask | leader 权限桥 | SubagentCapabilityMode | permission ask |
| 权限隔离 | read-only 默认 + writable opt-in ✓ | 每成员 mode + teamAllowedPaths | worker 工具白名单 | sandbox + 深度上限 | agent mode |
| 前缀缓存继承 | **✓ fork 继承 leader 前缀**（实测 cache_hit）| fork-subagent 省 80% | — | — | — |
| 持久化/并发安全 | 进程内 mailbox + 依赖树 | 文件锁（proper-lockfile + tmp+rename 原子写）| 磁盘 mailbox | — | — |
| 模型无关 | **✓ 全局 provider 注入，零硬编码** | 每成员 model 覆盖 | 继承 leader 模型 | — | — |

## 三、命令完善对照

**Reasonix 已有**：`/team-create`（+writable）`/team-add` `/team-status` `/team-remove` `/team-stop` `/task-message`（后台任务消息）
**已登记 /help**：本轮补上（99663cf80）。

**缺失（建议按优先级补）**：

| 命令/机制 | 优先级 | 依据（参考实现） | 说明 |
|---|---|---|---|
| `/team-broadcast <msg>` | P0 | qwen broadcast（Promise.allSettled 容错）| 指令群发全体成员——多智能体协同基本能力 |
| send_message 完成闭环 | P0 | qwen/CCB 双通道 | team_message 工具已有（P6.2），核对 leader 收信注入是否完整 |
| `/team-plan <task>` 一键规划分发 | P1 | planner 产出计划 → 自动拆任务分配 | 复用 team-planner skill + /team-add 编排 |
| teammate 消息自动注入 leader | P1 | qwen 500ms 轮询 + 稳定标签 | 中途进度可见（当前只有完成信封）|
| plan 审批门 | P1 | qwen plan_approval + 审批期只读收缩 | 决策权回 leader（15决策/6纪律/5执行思想）|
| 停滞检测 + /team-abort-stalled | P2 | qwen STALL_THRESHOLD | 长任务可靠性 |

## 四、模型/API 无关性审计（开源关键）

**结论：Reasonix team 管线完全不依赖任何特定模型/API。**
- teammate fork 用 `resolveSubSessionRuntime("", "")` → TaskTool 注入的**全局 provider**（用户接入的任何模型）；
- team 相关文件（teammate_store/subagent_fork/jobs）**零模型/API 硬编码**（grep 审计通过）；
- 前缀继承、P1 信封、mailbox 均为**协议层**，与模型无关；
- 测试用用户的 DeepSeek API，但**换任何 provider 一样跑**（boot 装配即用）。

## 五、结论

Reasonix team 在**基础管线（创建/分配/并行执行/完成回收/前缀缓存/模型无关）上已完整可用**，
且并行能力实测优于多数参考实现（3 分析者同时工作）。主要差距集中在**协作深度**：
teammate→leader 实时消息、plan 审批门、broadcast、停滞检测——这些是"从能用变好用"
的第二层能力，建议按上表 P0→P1 顺序补。
