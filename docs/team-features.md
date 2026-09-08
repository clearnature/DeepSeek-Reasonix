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

### 任务板（qwen task_* 对齐）

- `task_create` 发布无主任务（open）→ 任意 teammate `task_list` 找到后
  `task_update owner=<自己>` 认领 → 完成后 `task_update status=done`。
- 仲裁：仅 owner 可完成；二次认领被拒（同 owner 幂等）；`task_stop`
  停成员工作时自动把其认领的任务释放回 open。
- 任务板随团队快照持久化（重启恢复）；成员完成时若有 open 任务，
  自动发看板唤醒通知。

### Worktree 隔离（qwen enter/exit_worktree 对齐）

- `enter_worktree {name}`：为 teammate 建独立 git worktree（新分支）并把
  其写 token 绑定到该路径——后续写入全部落在 worktree 内（我们架构下
  替代 qwen 的 cwd 切换，更严格）。
- `exit_worktree {name, action}`：`keep` 保留 checkout 与分支；`remove`
  （需 `confirm=true`）删除两者并把成员恢复为只读。
- 桌面 Team 面板显示成员的 worktree 路径/分支。

### 调度（qwen loop_wakeup + cron 对齐）

- `loop_wakeup {delay_seconds, prompt}`：一次性延迟唤醒——到期把 prompt
  写入 leader inbox，随下一轮到达（不跑无人自动轮，符合 qwen 模式）。
- `cron_create {schedule_seconds, prompt}` / `cron_list` / `cron_delete`：
  周期唤醒，同一调度器。

### 子会话（qwen create_sub_session 对齐）

- `create_sub_session {prompt, completion}`：仅在 `reasonix serve` 下可用
  （daemon-only，与 qwen 一致）；`sent` 投递即返回、`first-turn` 等待首轮
  并返回输出。无会话桥时原样返回 daemon-only 提示。
