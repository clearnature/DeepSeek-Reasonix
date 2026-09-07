# Team 编排实现 × 文档对齐审核（2026-09-07）

> fable5 Step 9 对抗自检：qwen 移植（T1-T11）完成后，逐项核对实现 vs
> 架构报告（20260907-team-architecture-report.md）/ 差距登记（gap）/ 调用配套（call-surface-audit）。
> 结论：主体对齐，3 个真差距（1 已修，2 记录为 debt）。

## 一、对齐确认（✅）

| 文档承诺 | 实现 | 证据 |
|---|---|---|
| 组实体（qwen TeamFile.name）| TeammateStore.Name/CreateGroup/DeleteGroup | 0ebb7b995 |
| /team-group 命令 + 工具 | applyTeamGroup + team tool group_* | dd67e4917 |
| cap（MAX_TEAMMATES）| MaxTeamTeammates=10 | 5b0de4182 |
| send_message teammate 路由 | team tool mail action | abda4bdca |
| 协作 shutdown | RequestShutdown + shutdown action | d9d5d883a |
| 停滞 abort | checkStalled + **boot 启用 600s**（本审核修复 76b2b572a）| 76b2b572a |
| plan 审批应答 | /team-approve 接线（库已有 Approve）| 90979de19 |
| mailbox 磁盘 + 状态快照 | boot 接 inboxRoot + snapshotPath | 6bcf000a5 |
| 任务板可见性 | team tool tasks action | be085812d |
| 实时注入 | compose `<teammate_message>` ride-the-turn | be085812d |
| 模型工具面第一性 | team 工具（leader-only ProviderVisible）| fcd039906 |
| 缓存红线 | 注入在 append 消息内，前缀零触碰（memory/background 先例）| 注入测试 |

## 二、审核发现（按严重度）

### G1（已修）：T6 abort 是死代码
- 现象：SetStallAbort 定义存在但**从未配置**——stallAbort=0 = abort 禁用（仅 jobs 层警告）。早前判"T6 已实现"只核验了 checkStalled 存在，未核验**启用**。
- 修复：boot SetStallAbort(600s)（76b2b572a）。教训：功能存在 ≠ 生效，须验装配启用（同 saveMainRequest 教训族）。

### G2（已评估，维持 debt）：调用面双份语义
- 现象：team 工具 Execute 与 /team-* applyTeamCommand **各自实现校验/文案**（都调 ts 方法但中间逻辑重复），非"工具封装同一 Controller 路径"。
- 影响：改语义需两处同步；文案漂移风险。
- 缓解：两者都薄（薄封装 ts 方法），重复面小。彻底共享 = 抽 team 命令语义层（applyTeamCommand 供工具调）——成本高，标记 debt，下阶段做。

### G3（已修 8da9a2e5c）：工具形态补齐
- 现象：audit 记"leader→teammate 直投缺 🟡 / send_message teammate 路由"——实现为 **team tool mail action**（send_message 未扩展）；"approve 应答工具/命令"——/team-approve 命令做了但 **team 工具无 approve action**（半）。
- approve 模型工具 action 已补（8da9a2e5c），工具面与 /team-approve 对齐；mail/team_message 形态差异记录为设计选择（send_message 保持 task 路由，mail 走 team 工具）。

## 三、测试与规范基线

- 每原子守卫测试在（组生命周期/cap/mail/注入 ×2/工具全链）；control 12.4s、agent 31s、boot 16s 全绿
- repolint clean（1368 baselined）、gofmt/vet 干净
- 新代码全部上游无路径文件（teammate_assembly/team_group/team_shutdown/team_leader_tools）→ merge 防丢模板满足

## 四、结论

工程主体与文档对齐（11 原子 9 commit 全验证）；审核抓到 1 个真功能缺口（G1 abort 未启用，已修——这类"存在未启用"与 saveMainRequest"定义在调用丢"同族，归因于只验存在不验生效）与 2 个结构性 debt（G2 双份语义、G3 文档形态滞后），建议下一阶段消化。

## 五、记忆/交接

- G2/G3 debt 待下一阶段：抽 team 命令语义共享层；决定 approve 的模型工具形态；文档同步。
- 全程 Cache-impact：模型工具 schema 一次性变更（升级后稳定），注入走 ride-the-turn（无前缀漂移）。
