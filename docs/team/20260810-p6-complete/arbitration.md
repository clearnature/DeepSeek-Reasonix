# P6 完成事件驱动 — 父代理仲裁记录

> 12 人 planner 队设计完毕（2026-08-10）。以下为父代理对 §8 争议点的最终裁决，
> 执行小队与纪律团以本记录为准。

## 仲裁 1：完成事件回调签名
**裁决：3 参** `WithJobDoneObserver(func(id string, st jobs.Status, err error))`。
- 理由：id 定位 job、st 区分 done/failed/killed、err 携带失败详情——handler 只依赖参数，
  禁止反查 jm（时序陷阱：recordCompletion 触发时 j.status 仍 Running）。结构式 6 参过度设计。

## 仲裁 2：回调执行方式
**裁决：同步回调 + recover 隔离**，不做 done-channel 异步分发。
- 理由：挂点（recordCompletion L975-977 / L1000-1002）已在 m.mu 之外（与
  taskRecorder.RecordDone 并列，已核实无锁）；handler 全为内存级操作；异步分发
  拿不到 st 参数、startInvalid 路径会错过窗口。订阅者 panic 用 defer recover 隔离，
  不破坏 job 收尾管线。

## 仲裁 3：依赖「自动推进」语义（最大分歧点）
**裁决：实现自动 Assign**（完整完成事件驱动，符合用户意图）。
- 完成事件到达 → 找出 dependsOn 全部终态且未启动的等待任务 → 自动 Assign 到原 owner。
- 缓解措施（planner 已论证）：
  1. ctx 重建：recordTask 补存 `SessionID string`，自动 Assign 用 store 内
     `assignContext(sessionID)` 重建最小 ctx（jobs.WithSession 语义），**绝不缓存 ctx 对象**；
  2. 依赖环：pendingDependenciesLocked 只查终态 → 环死等不自动推进（安全，无死循环）；
  3. owner 忙/被删：跳过该任务（保留登记，Tasks() 可见），留待 leader 手动处理；
  4. 自动 Assign 的 prompt = 登记时原文（字节稳定，缓存红线不破）；
  5. 同步回调内调 Assign：挂点在 m.mu 外（仲裁 2），startForSession 重入 m.mu 是
     不同 goroutine 的独立临界区，无死锁（planner-8 锁序分析确认）。
- 被 killed 的 job 算终态（st=Killed），依赖放行（生命周期设计裁决）。

## 仲裁 4：mailbox 唤醒与完成 Notice
**裁决：完成 Notice 本身不发**（jobs 层 closing Notice 已覆盖正常 done，再发=第 4 个
重复通道+风暴源）；**mailbox 积压唤醒是唯一例外**——置 idle 后若 inbox 有新信
（N 封聚合），notifyMail 一次（计数 N，防风暴）。PostMail 时 teammate 已 idle 也
立即唤醒（信件到达即通知，不等完成事件）。

## 仲裁 5：置 idle 语义
**裁决：严格 LastJobID 匹配 + 仅 Running→Idle 翻转**（Complete 既有语义）；
惰性 syncStateLocked 保留为兜底（observer 未注册/时序窗口时仍能自愈）。

## 仲裁 6：destroy 窗口 / Remove orphan
**裁决：destroy 窗口对齐 RecordDone（吞事件，TeammateStore.DestroyAll 兜底）**；
**Remove 时清理该 teammate 的登记任务**（防 Tasks() 悬空——planner 发现的隐藏缺口，
本次并入）。

## 仲裁 7：tm.Ref 断点
**裁决：不并入本次**（独立跟踪——续轮 continue 语义修复，需配套 fork 前缀验证）。

## 缓存红线（全队一致）
发送侧字节零变化：改动均为进程内回调 + event.Notice（UI 层，不进 provider 输入）；
teammate fork/continue 前缀稳定；自动 Assign prompt 字节不变。
