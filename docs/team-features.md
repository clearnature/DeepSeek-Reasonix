# Team & background-task features

多智能体协作与后台任务功能（P1–P6.2，2026-08 落地）。协作流程纪律见
`docs/team-org.md`；设计证据链在 `docs/team/`。

## Background tasks

- 后台任务：`task` 工具 `run_in_background`，结果经 `<background-job-result>`
  信封自动投递到父会话下一轮（P1）；`wait` / `bash_output` 仍可显式取结果。
- Steer：`send_message` 工具（父侧）/ `/task-message <job_id> <text>` 给运行中
  的后台 agent 发中途指令（P3），下一工具轮次边界投递。
- 任务面板：`/status` 明细段（CLI）与桌面 Task Monitor 面板（P2）展示
  job/task 快照、尾部输出、停止按钮（session 内停止）。
- 前台→后台交接：`/background` 立即转后台；或超过
  `foreground_backgroundize_seconds`（默认 **120s**，`0` 关闭）在迭代边界自动
  转后台（P4）。注意：相对上游 v1.23.0 基线这是**默认开启的行为变化**——
  长前台任务会自动转后台，如不需要请显式配置为 `0`。

## Team

- 唯一编排入口 = `team` 模型工具（qwen 对齐——2026-09-08 起不再有
  `/team-*` 命令）：`group_create`（命名团队）/ `create`（注册成员）/
  `add`（派任务——fork 父前缀为后台 job）/ `status`（成员状态）/
  `tasks`（任务板）/ `mail`（成员信箱）/ `remove` / `shutdown` /
  `approve`（计划审批）/ `broadcast` / `grant`（写路径或 worktree 授权）/
  `revoke`。
- 模型是编排的主要调用方：说"派 N 个智能体并行研究 X"→ leader 自主
  `team group_create + create + add` 建团分发；桌面 Team 面板仍可查看。
- 每个 teammate 是独立后台 agent：首次派活 fork 父会话前缀（共享 prompt
  缓存），续轮 continue 同一 transcript（前缀稳定）；完成事件驱动置
  idle / 依赖自动推进（`depends_on` 链）/ mailbox 积压唤醒（P6）。
- teammate 的信件（mailbox）持久化于会话目录 `team-inbox/`；`team mail`
  在 teammate 运行时作为 steer 投递，空闲时留存并在其完成时通知 leader。
- 完成反馈默认走用户下一轮（qwen 模式）；`team_mail`（teammate→leader）
  与运行中 steering 经 `send_message`（job_id）投递。
