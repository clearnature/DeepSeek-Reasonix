# 20260812 子代理 asker 注入 + ask 权限模式感知（team 自主完成任务）

## 背景（实验验证）

GA 引擎（cmd/team-ga2）评估"ask 基因"（子代理在歧义时用 ask 工具向 leader 提问）。两次实验（含强制 ask 任务）均无 AUTO-APPROVE 日志、无 AskRequest 事件——**自动审批空转**。

**根因（代码级）**：子代理（teammate）spawn 时 `a.asker = nil`（New 不设；Options 无 Asker 字段）。`AskTool.Call`（ask.go:119）`asker == nil` → 返回 headless model-assumption fallback（不阻塞、不产生 AskRequest）——ask 基因的评估从未走审批链。

## 用户设计意图（2026-08-12 确认）

系统权限三模式（询问 ask / 自动 auto / yolo）应驱动 **ask 工具的审批策略**，且 **team 继承父权限**：
- 用户设定任务 → AI 自主完成（auto/yolo 模式 ask 自动批准，风险由权限模式控制）
- 询问模式保留真人审批（AskRequest → 真人）
- 目标：GA 引擎的 auto-approver（实验 hack）产品化为"auto 模式自动批准"

## 现状 asker 注入

| 对象 | asker | 权限 gate | 来源 |
|---|---|---|---|
| executor（主 agent） | Controller | 交互 gate | controller.go:2467 / SetGate |
| planner | Controller | 继承 | coordinator.go:801 |
| **子代理（teammate/subagent）** | **nil** ❌ | **headless variant（继承，deny 规则生效）** ✓ | task.go:2061 New 不设 asker / NewTaskTool gate |

**结论**：权限 gate 已继承（子代理 deny 规则生效）；**缺 asker 注入 + Controller.Ask 权限模式感知**。

## 方案（两部分）

### 部分 A：子代理 asker 注入（已立项）
TaskTool 从工具执行 ctx（`CallContext`，execute_one.go:671 带 a.asker=Controller）取 asker → 经 ctx 传 `RunSubAgentWithSession` → `sub := New(...)` 后 `sub.SetAsker(a)`（与 fork 继承注入点同模式）。

### 部分 B：Controller.Ask 权限模式感知（产品化）
`Controller.Ask` 读 `approvalManager.toolApprovalMode`（已有字段）三分支：
- **ask（询问）**：现状——emit AskRequest → 真人 AnswerQuestion
- **auto**：emit AskRequest（可审计）+ **自动答 first option**（GA auto-approver 产品化；仍可被真人覆盖/查看）
- **yolo**：不阻塞——返回模型自决（等效 headless fallback，yolo = 全自动）
- 子代理经 asker 注入后自动继承该策略（teammate ask → leader Controller.Ask → 按父权限模式）

## 任务分解

- [x] T1 拓扑扫描：注入点（RunSubAgentWithSession sub 后）+ asker 来源（工具执行 ctx）+ 权限模式读取（approvalManager.toolApprovalMode，approval.go:43/220）
- [ ] T2 TaskTool spawn 传 asker（ctx withValue）
- [ ] T3 RunSubAgentWithSession `sub.SetAsker(askerFromContext(ctx))`
- [ ] T4 Controller.Ask 权限模式三分支（ask/auto/yolo）
- [ ] T5 验证：GA 实验重跑（auto 模式 g3 强制 ask → 自动批准日志、不挂起）+ 询问模式回归（AskRequest → 真人路径不变）+ 单元测试（auto/yolo 分支）
- [ ] T6 文档：本 plan + 实验证据

## 验收标准

1. teammate 子代理调用 ask → 产生 AskRequest（captureSink 收到）
2. auto 模式：AskRequest 自动批准 → 子代理收到答案继续（不挂起）
3. yolo 模式：ask 不阻塞 → 模型自决（等效 fallback）
4. 询问模式：AskRequest → 真人（主 agent 回归不变）
5. 缓存红线：纯注入/审批逻辑，零发送字节改动
