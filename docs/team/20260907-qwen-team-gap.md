# qwen-code Team 编排深度对照 + 差距登记（2026-09-07）

> 方法：深度阅读 `/data/training/cli/qwen-code/packages/core/src/agents/team/`（TeamManager 1825 行、
> types/mailbox/tasks/teamHelpers + tools/team-create）+ 核实 Reasonix 现状（teammate_store/jobs/boot 装配）。
> 目的：登记"小组按事务创建/解散/释放"需求的设计依据与差距，供实现前决策。

## 一、qwen 架构速记（源码实证）

```
工具面    team_create / spawn teammate / send_message / task 工具
编排中枢  TeamManager —— 单 leader 单组（config.getTeamManager 单例，已有组则拒绝）
任务层    tasks.ts —— SwarmTask 文件板（pending/in_progress/completed + blockedBy 依赖图）
通信层    mailbox.ts —— 磁盘 inbox/agent（proper-lockfile + Mutex + atomicWriteJSON）
持久化    TeamFile ~/.qwen/teams/{name}/config.json（members/leadSessionId/leadPid）
运行载体  Backend（InProcess/Tmux/ITerm）—— teammate = subagent 运行时实例
```

关键机制：
- **全磁盘协议**：消息/任务/组都是磁盘文件 + 锁 → 进程崩溃 teammate/mail/task 全存活
- **拉模式任务分发**：`scanIdleAgentsForTasks`——IDLE 且无 shutdown_pending 的 teammate 自动认领 pending 任务；per-agent claim Mutex 防 TOCTOU
- **双向 500ms 轮询**：leader `pollLeaderInbox` → `<teammate_message>` 包络实时注入 leader 下一轮模型流（完整版 + display 短版双文本）；teammate `flushNextMessage` 逐条注入
- **协作解散**：`requestShutdown(name)` → `shutdown_request` 消息 → 成员 finish 后回 `shutdown_approved/rejected` → 终止；`_shutdownPending` per-agent 门（防伪造 shutdown 扩爆炸半径）
- **组生命周期**：创建 = TeamFile + EEXIST → `tryReclaimStaleTeam`（leadPid 存活检测回收 stale）；**组不自动消解**——"nothing deletes team dirs on normal exit (only an explicit team_delete)"；完成判定 = `hasActiveTeammates/allTeammatesTerminated/waitForTeammateActivity`
- 消息类型：shutdown_request/approved/rejected、plan_approval_request/response、task_assignment
- 其他：abortStalledTeammates（停滞中止）、leaderPermissionBridge（工具级审批转发）、plan 审批期权限收缩、MAX_TEAMMATES=10

## 二、Reasonix 差距登记（核实现状）

| 维度 | qwen | Reasonix 现状 | 差距等级 |
|---|---|---|---|
| 组聚合（TeamFile/members） | ✅ 单组文件 + 显式 team_delete | ❌ 无组概念（teammate 独立注册表） | **新增需求**（用户：小组按事务创建/解散） |
| 任务分发 | 拉（IDLE 自动认领） | 推（/team-add 显式指派）| 设计差异（可保留推 + 补拉） |
| 中途消息实时注入 leader | ✅ `<teammate_message>` 500ms | ❌ 仅完成 P1 信封（comparison 已登记 P1）| P1 |
| 协作解散 shutdown 协议 | ✅ requestShutdown 消息 | ❌ /team-stop 硬停 | 待补（若做组释放） |
| broadcast | ✅ | ✅ 已补（teammate_command.go）| 闭合 |
| plan 审批门 | ✅ | ❌（approve 通道不在当前 TeammateStore）| P1（已登记）|
| 停滞检测/abort | ✅ | ❌ | P2（已登记）|
| **mailbox 磁盘化** | ✅ 磁盘 inbox | ⚠️ **实现存在**（inboxRoot writeFile）但 **boot 装配未传 inboxRoot → 生产 ephemeral**（测试才落盘）| **接线 1 行** |
| **状态/组持久化** | ✅ TeamFile 原子写 | ❌ **snapshotPath 字段声明（P10 crash recovery）但零实现代码**；注册表纯内存，重启即失 | **P10 待实现** |
| teammate transcript | — | ✅ SubagentStore fork jsonl 落盘 | 已闭合 |
| 依赖任务/自动推进 | ✅ blockedBy 图 + 拉 | ✅ recordTask/dependsOn + 完成事件自动推进（P6）| 已闭合 |
| 前缀缓存继承 | ✅ fork-subagent | ✅ fork 前缀 | 已闭合 |
| 模型无关 | ✅ | ✅ 全局 provider | 已闭合 |

## 三、用户需求与待决语义

需求：**小组（编排写作小组）按事务创建，事务结束释放小组**。

待决（实现前必须拍板）：
1. **释放语义**：(a) 同 qwen 显式解散（组文件保留直到显式删）vs (b) 事务完成自动释放（组内全部 idle + 无 pending → 自动清组——比 qwen 更进一步）
2. **组模型落点**：命令层（/team-group start/end 批量管理现有 teammate）vs TeammateStore 内 TeamFile 化（组 = 持久化实体）
3. **持久化接线范围**：只接 mailbox（inboxRoot 1 行）vs 补 P10 snapshot（状态原子落盘 + 重启恢复）vs 两者
4. 组内协作命令集：/team-add 批量、/team-status 组视图、组内 broadcast（已可用）

## 四、建议实现顺序（待确认）

1. **P10 snapshot 实现**（状态持久化——组/成员/任务落盘 + 构造加载）——组概念的载体
2. **boot 接线 inboxRoot**（mailbox 落盘，1 行）
3. **组命令**：/team-group start <name> <n> <task...>（创建组 + 批量分派）/team-group status /team-group end（解散释放）
4. 释放语义决策后：(b) 自动释放 = 完成观察者检测组全 idle → 自动清组（含 mailbox/transcript 归档）

## 五、约束提醒

- 前缀缓存：组生命周期改动不涉发送侧字节（纯编排状态机）→ Cache-impact none
- 防丢模板：实现放上游无路径文件（teammate 域本就本地）
- 现有守卫测试：team_smoke_test（命令桥）、teammate e2e——组功能补对应测试
