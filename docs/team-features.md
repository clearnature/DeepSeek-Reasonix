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

- `/team-create <name> <role>`、`/team-add <name> <prompt>`、`/team-status`、
  `/team-stop <name>`、`/team-message <name> <text>`、`/team-remove <name>`。
- 每个 teammate 是独立后台 agent：首次派活 fork 父会话前缀（共享 prompt
  缓存），续轮 continue 同一 transcript（前缀稳定）；完成事件驱动置
  idle / 依赖自动推进（`depends_on` 链）/ mailbox 积压唤醒（P6）。
- teammate 的信件（mailbox）持久化于会话目录 `team-inbox/`；`/team-message`
  在 teammate 运行时作为 steer 投递，空闲时留存并在其完成时通知 leader。
