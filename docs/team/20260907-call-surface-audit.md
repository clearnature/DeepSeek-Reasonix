# Reasonix vs qwen-code Team：功能 + 调用配套全面审核（2026-09-07）

> 方法：两侧调用面逐项对照（qwen 工具名 = tool-names.ts + tools/*.ts 实现；
> Reasonix = controller host 命令 + tool/builtin 工具 + TeammateStore/TeammateStore 方法）。
> 结论分三层：功能等价 / 功能有但调用面缺 / 整体缺失。

## 一、调用面清单对照

### qwen 调用面（全部模型工具，leader 对话中直接可用）

| 工具 | 作用 |
|---|---|
| `agent` | 派 agent 执行任务 |
| `task_create / task_update / task_list / task_stop` | **任务板 CRUD**（磁盘 SwarmTask；teammate 侧拉模式认领的接口，teammate 也能用）|
| `team_create` | 创建组（TeamFile；已有组拒绝 / stale 回收）|
| `team_delete` | **解散组**（仅 leader；删 TeamFile+任务板+inbox）|
| `send_message` | 双路由：`task_id`（后台任务 steer）或 teammate 名（成员指挥）|
| `team_plan_approval` | 计划审批应答 |
| mailbox 消息类型 | shutdown_request/approved/rejected、plan_approval_request/response、task_assignment |

### Reasonix 调用面

| 调用 | 形态 | 可用方 |
|---|---|---|
| `/team-create /team-add /team-status /team-remove /team-stop /team-grant /team-revoke /team-broadcast` | **host 命令** | 仅用户（模型不可调）|
| `task`（run_in_background 子代理 job）| 模型工具 | agent ✓ |
| `send_message`（job_id steer 后台 job）| 模型工具 | agent ✓（leader）|
| `team_message`（mailbox 邮件）| 模型工具 | teammate（target leader/成员）|
| `plan_approval_request` | 模型工具 | teammate 提交 ✓ |
| `/team-approve` 应答 | host 命令 | 仅用户（模型无应答工具）|

## 二、功能 × 调用配套差距定级

| # | 能力 | qwen | Reasonix 功能 | 调用面差距 | 级 |
|---|---|---|---|---|---|
| 1 | 创建组 | team_create 工具 | 无组概念 | **组功能整体缺**（本会话登记新需求）| 🔴 |
| 2 | 解散组 | team_delete 工具 | /team-remove 单成员 host | 组解散缺 + 无工具面 | 🔴 |
| 3 | 任务板（teammate 认领）| task_create/list 等 + 拉模式 | 推模式 /team-add（host）| 任务板工具缺（teammate 无法自助/leader 模型无法派）| 🔴 |
| 4 | leader→teammate 消息 | send_message（teammate 路由）| team_message 是 teammate→…，leader→teammate 无直投（/team-add 才派活）| 直投工具缺 | 🟡 |
| 5 | 中途实时注入 leader | `<teammate_message>` 500ms | 仅 P1 完成信封 | 实时注入缺（已登记 P1）| 🟡 |
| 6 | 计划审批应答 | team_plan_approval | plan_approval_request 提交有；应答 = /team-approve 降级 notice | 应答工具缺 | 🟡 |
| 7 | 后台任务 steer | send_message task_id | send_message job_id ✓ | 等价（同工具名）| ✅ |
| 8 | teammate 工作历史 | 会话 transcript | SubagentStore jsonl ✓ | 等价 | ✅ |
| 9 | 依赖任务自动推进 | blockedBy 图 + 拉 | recordTask/dependsOn + 完成事件自动推进 | 等价（机制不同：拉 vs 事件）| ✅ |
| 10 | 前缀缓存继承 | fork-subagent | fork 前缀 ✓ | 等价 | ✅ |
| 11 | mailbox 磁盘化 | 磁盘 inbox + 锁 | 实现有（inboxRoot）boot 未接线 | 接线缺 | 🟡 |
| 12 | 状态持久化 | TeamFile 原子写 | snapshotPath 声明未实现 | 实现缺 | 🔴 |

## 三、核心结论：功能层大半有对应，**调用面系统性不配套**

**根因**：qwen 的 team 编排设计为**模型工具面**（leader 在对话流里直接 `team_create`/`send_message`/`task_list`/`team_delete` 完成整个编排，无 host 命令）；我们的 team 编排从 P6 裁决起就落在 **host 命令层**（用户敲 /team-*），模型工具只有 send_message/team_message/task 三个零散件。

后果（用户实测触发点）：
- agent（模型）**无法自主**创建组/派活/解散——必须用户敲命令（此前"我用编排创建 teammate"失败即此）
- teammate 侧无任务板自助（qwen teammate 可 task_list 看板认领/回报进度）
- 解散 = 无协作协议无组概念

## 四、对齐路线（调用配套补齐优先级）

| 步 | 补什么 | 形态 |
|---|---|---|
| P0 | leader 组管理工具：`team_create` / `team_group_dissolve`（模型工具，封装 TeammateStore 批量 create/remove）| 解决"模型无法自主编排" |
| P0 | 任务板：`task_list`（模型可查 teammate 状态/任务）+ 批量分派 | 调用面与 /team-status 对齐 |
| P1 | leader→teammate 直投（send_message 双路由加 teammate target，或 team_message 开放 leader 源）| 中途指挥 |
| P1 | 实时进度注入（teammate 消息 500ms → leader 下一轮）| comparison 已登记 |
| P2 | 协作 shutdown 协议 + 组事务释放（用户需求）| 新 |
| 持久化 | inboxRoot 接线（1 行）+ P10 snapshot 实现 | 组载体 |

## 五、约束

- 模型工具新增 = **工具 schema 变化** → Cache-impact: low（一次性，升级后稳定）
- 新工具注册走 builtin/boot（防丢模板：上游无路径文件）
- host 命令保留（用户手动编排仍可用），工具是**补充面**非替换
