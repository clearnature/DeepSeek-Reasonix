# 定稿：P6 team 组织（规划小组一致性裁决）

> 事务：docs/team/20260810-p6-team/ · 规划小组：plan-1/2/3.md（3 个 team-planner 独立产出）
> 一致性裁决：父代理汇总 → 共识采纳 + 分歧裁决

## 一、共识（3/3，方案 A：消息驱动轮次）

1. **teammate = 持久身份 transcript + 每次派活一个后台 job**——否决 Qwen 常驻 run-loop（与我们的 job 终态语义冲突）
2. **首轮派活 = fork 前缀继承 leader**（captureForkPrefix → PrepareParentFork，命中 leader 缓存）+ **续轮 = PrepareContinue**（teammate 自身 transcript 续跑，前缀稳定）
3. **结果回 leader = P1 信封**（teammate job kind="task" 非静默，label 带 teammate 名）
4. **运行中指挥 = P3 steer 零改动生效**
5. **teammate 只读**（绕开 P7 写冲突分叉）；**只读必须 schema 全量 + forkReadOnlyGate**（禁用 readOnly registry——否则 ToolsHash 断前缀）
6. **头号纪律点：三要素钉死**（model/effort/工具集注册时快照，续轮 validateMeta 强制一致 → 前缀稳定缓存命中）

## 二、分歧裁决

| 分歧 | 方案 | 裁决 |
|------|------|------|
| 未运行消息 | plan-1/2: 磁盘 inbox + 唤醒转投 / plan-3: 显式拒绝 | 采纳 **plan-3 简化（MVP）**：运行中走 P3 队列指挥；未运行派活显式拒绝（诚实语义）。磁盘 inbox 列 P6.1 增强 |
| 派活入口 | plan-2: create_team 强约束 + spawn_teammate 工具 / plan-3: /team-add + TeamMessage 工具 | 采纳 **plan-3**：`/team-create` `/team-add` `/team-status` `/team-remove` slash + `team_message` 工具（teammate 会话可见、leader 会话隐藏） |
| 生命周期 | 三份一致：destroy 钩子清理（kill job + 删 transcript + 清注册表） | 采纳 |

## 三、关键改动点（执行前 T0）

- **T0-A fork/silent 解耦**：task.go fork 分支硬绑 StartSilentForSession——teammate 需要「非静默 fork」（结果经 P1 信封回 leader）；P5 默认行为必须保持（TestStartSilentUnscopedAndNormalJobStillDelivers 零回归）
- **T0-B PrepareContinue 前缀延续**：续轮 system/tools/model/effort 一致 → validateMeta 强制（架构防线）
- **T0-C job session 归属**：teammate 轮 job 挂 leader session → P1 信封 drain 进 leader 下轮（复用零新通道）

## 四、任务分解

| 任务 | 内容 | 文件 | 验证 |
|------|------|------|------|
| T0 | 三项前置探查（A/B/C） | task.go/subagent_store | 结论记录 |
| T1 | fork/silent 解耦：fork 分支参数化静默（默认 true 保 P5）+ teammate 非静默路径 | `internal/agent/task.go` | P5 测试零回归 |
| T2 | TeammateStore（name/role/三要素快照/LastJobID/idle 状态，内存态） | `internal/team/`（新包） | 单测 |
| T3 | 派活管线：Assign(name, task) → 首轮 fork 前缀 + PrepareParentFork、续轮 PrepareContinue → 后台 job（非静默、forkReadOnlyGate）→ 终态回调置 idle | `internal/agent/task.go` | 单测（首轮 ref/续轮 ref） |
| T4 | slash：`/team-create` `/team-add` `/team-status` `/team-remove` + Controller 方法 + destroy 钩子 | `internal/control/controller.go` | `go test ./internal/control/` |
| T5 | `team_message` 工具（teammate 会话可见/leader 隐藏） | `internal/tool/builtin/` | 单测 + schema 断言 |
| T6 | e2e：注册×2 → 派活 → 首轮前缀 byte-identical（cache_hit>0）→ 续轮前缀稳定 → P1 信封回 leader → 只读 gate 拒绝写 → destroy 清理 | 集成测试 | 定向 e2e + `go build ./...` |
| T7 | 文档：plan.md 汇总 + execution.md + SPEC 小节 | docs | 文件存在 + repolint |

## 五、缓存/纪律检查点

- 首轮 fork 复用 captureForkPrefix（零发送、leader 零改动）✅
- teammate 轮消息只 append 自身 transcript；leader 前缀零变化 ✅
- role prompt 固定字节 → 续轮前缀严格 append、命中 teammate 缓存 ✅
- 只读 = schema 全量 + forkReadOnlyGate（禁用 readOnly registry）✅
- fork/silent 解耦默认保持 P5 静默（teammate 显式开启投递）✅
- 运行值（name/role/任务文本）不进 schema/system prompt ✅
