# Qwen Agent Team 对比研究报告（代码级验证版）

> 生成：2026-08-10 · 基线：main-v2（team 分支 07fcf585b 基点）· 关联 memory：team-multiagent-qwen-comparison
> 研究源：`/data/training/cli/qwen-code`（packages/core/src/agents/team/、utils/forkedAgent.ts、memory/writeContextFile.ts）
> 我方源码：`/home/yanli/work/DeepSeek-Reasonix`（internal/agent/、internal/jobs/、internal/control/）

## 一、Qwen 侧关键机制（源码行号验证）

### 1.1 磁盘 mailbox 消息通道 —— `agents/team/mailbox.ts`（357 行）
- 存储：`~/.qwen/teams/{teamName}/inboxes/{agentName}.json`（L10-11）
- **双层并发锁**：per-inbox 进程内 `Mutex`（L99-108，keyed by 绝对路径）+ `proper-lockfile` 文件锁（L71-88，10 次重试、5–100ms 随机退避防惊群、5s stale、crash 日志降噪）
- **热路径零锁竞争**：读不锁（L173-178），写走 `atomicWriteJSON`（tmp+rename），读者永远看到写前/写后完整快照；只有轮询发现未读消息才加锁 `consumeUnread` 原子标记已读（L245-258）
- **有界文件**：已读消息 5 分钟保留窗口后压缩，未读永不丢弃（L200-210）
- **损坏隔离**：解析失败的 inbox rename 为 `.corrupt-{ts}` 并视为空（L337-356），防止坏文件永久阻塞后续消息（含 shutdown 指令）

### 1.2 500ms 轮询自动注入 leader —— `agents/team/TeamManager.ts`
- `ensureLeaderInboxPolling`（L875-882）：`setInterval(..., 500)`，teammate 存活期间持续运行
- `pollLeaderInbox`（L905-932）：`consumeLeaderInbox` → `callback(formatLeaderEnvelope(...))` 注入 leader 会话
- **注入信封**：`<teammate_message from="...">`（L151），防伪造是**结构性转义**（L960-978 escapeEnvelopeTags，只转义信封开头 `<`），不用 secret
- **退出前兜底**：`drainLeaderInbox` 一次性排空（L894-899），防止最后一条 teammate 消息在写盘后立即 IDLE 时丢失
- **注入落点**：`enqueueWithIdentity`（L1640-1655）在 `AsyncLocalStorage` teammate identity 下 `agent.enqueueMessage(message)` 进 runLoop

### 1.3 send_message 即时投递 + 背压 —— `TeamManager.sendMessage`（L503-570）
- 发给 leader → 写磁盘 mailbox；发给 teammate → 按 agentId 入队 `pendingMessages`，**agent IDLE 立即 flush**（L565-569）
- 背压：`MAX_PENDING_MESSAGES` 上限，超出直接抛错拒绝（L570-577），防单 teammate 淹没他人内存
- `broadcast` 用 `Promise.allSettled`（L603-615）：单接收者终止不连累全员
- shutdown 响应**类型化**（L520-547）：仅当 leader 真的 `_shutdownPending` 且回复以结构化 token 开头才分类为 approved/rejected，防误 abort；check-and-act 二次核对（L540-557）防并发竞态

### 1.4 无整工作区互锁（安全哲学的根本差异）
- 搜索 `writeClaim / workspaceLock / writeLock`：**Qwen 没有任何 workspace 级 write claim 概念**
- 唯一并发原语是 **per-file Mutex**（`jsonl-utils.ts` L41-50、`writeContextFile.ts` L32-60，带 30s timeout）+ `atomicFileWrite.ts` 的 tmp+rename 原子写（含 EPERM/EACCES 重试 3 次 + fsync）
- 即：并发写同一文件靠"每文件锁 + 原子替换"解决，**不同文件互不阻塞**

### 1.5 fork 子代理共享父前缀 —— `utils/forkedAgent.ts`
- `saveCacheSafeParams`（L30-60）：保存当前 `generationConfig + history + model`，检测 sysInstruction/tools 变化才失效
- fork 出的子代理复用父前缀 → 共享前缀命中缓存 → 省 token（memory 记录为 80%+）

## 二、我方四大痛点锚点验证（行号对上）

| # | 痛点 | 锚点 | 状态 |
|---|------|------|------|
| P0-1 | **后台任务强制整工作区 claim** | `internal/agent/task.go:745-753` `resolveWriterClaims` 无 write_paths 且 requireClaim → `WholeWorkspaceWriteClaim(workspaceRoot)`；互锁报错 `internal/agent/scheduler.go:180-195`（两条：后台子代理占用 / 父写占用）；波及 `internal/agent/fleet.go:193` | ✅ |
| P0-2 | **steer 排队滞留/不可转向** | `internal/control/admission_guard.go:23-63` steer 到达时另一轮在跑则 park 排队、普通 turn 静默丢弃；`internal/agent/run_loop.go:281-288` 每 stream 轮 `consumeSteer`，卡 wait 时滞留 | ✅ |
| P0-3 | **后台结果不投递** | `internal/jobs/jobs.go:778-814` `recordCompletion` 仅 emit Notice + 摘要（`background X finished: Y`），正文需 wait/bash_output；`internal/control/input.go:191-195` 下一轮开头仅注入一行 `<background-jobs>` 摘要 | ✅ |

## 三、P1-P4 可借鉴性排序与落地要点

### 🟢 P1 后台结果自动投递（bounded 注入 turn 尾部）
- **Qwen 做法**：500ms 轮询 → envelope → 随时注入 leader 会话（这是注入中间，我方红线不允许）
- **我方落地**：保持 `input.go:191-195` 的 turn 尾部位置不变，把"一行摘要"升级为 **bounded 完整正文**（如 ≤N 条/≤X KB，按 completions 队列 drain）
- 前缀字节不变（append 尾部、位置固定）→ 缓存零影响
- 新增：结果正文通过 drain 队列进 turn 尾部而非立即 Notice，替代 wait/bash_output 二次取正文

### 🟢 P2 job 消息通道 steer（背压 + 轮次边界投递）
- **Qwen 做法**：sendMessage 背压上限 + IDLE 立即 flush + 轮次边界投递
- **我方落地**：给 steerQueue 加背压/超时上限（参考 `internal/agent/agent.go:939` append 处），run_loop 每轮消费已有；卡 wait 滞留问题需在 wait 侧补"轮次边界唤醒"，或允许 steer 打断 wait 前的阻塞点
- 位置固定 append turn 尾部（已有 mid-turn steer 前缀标记），不改前缀

### 🟡 P3 窄化 write-claim（安全 vs 协作路线分叉，需用户拍板）
- **Qwen 做法**：完全无 workspace 锁，靠 per-file Mutex + 原子写兜底
- **我方现状**：无 write_paths 默认整工作区（含所有后台任务）
- **建议落地**：保留默认整工作区，但**显式提供窄化工具**（subtask 声明 write_paths 子集即只 claim 子集 + per-file 原子写兜底）；**不得**直接照搬 Qwen 无锁模式
- ⚠️ 动 `scheduler.go:180-195` 互锁语义前必须过设计评审（memory 红线）

### 🟢 P4 fork 子代理共享父前缀
- **Qwen 做法**：`saveCacheSafeParams` 保存父 generationConfig/history，fork 复用
- **与我们的相关性**：直接命中「前缀字节稳定=成本优势」同向目标；实现时注意 sys/tools 变化即失效（Qwen L33-46 的失效检测可照搬）

## 四、可直接复用的 Qwen 工程细节（免费赢）
1. `broadcast`/批量投递用 allSettled 而非 all —— 单接收者终止不连累全员
2. 已读消息按时间窗口压缩 + 未读永不丢弃 —— 消息通道有界且不丢
3. 损坏文件隔离（rename 到 `.corrupt-{ts}`）—— 坏文件不永久阻塞控制通道
4. check-and-act 二次核对 —— 异步投递中的竞态防护
5. 退出前 drain 一次性排空 —— 防"最后一条消息丢失"

## 五、哲学红线核对
- ✅ 所有 P1/P2 落地均 append turn 尾部（位置固定），不插入历史、不改前缀字节
- ⚠️ P3 若采纳需用户拍板 + 设计评审（安全哲学让步）
- 本报告研究过程未改动任何源码文件

## 六、遗留问题
- P3 窄化设计尚未出评审文档（需用户拍板后另立任务）
- 本文档为研究记录，落地实施另立任务追踪
