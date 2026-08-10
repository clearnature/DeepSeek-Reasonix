# P4 前台→后台动态降级：执行与审查报告

> 事务：docs/team/20260810-p4-backgroundize/ · 分支：team · 状态：✅ 已执行并通过纪律审查

## 一、流程记录（按团队章程）

| 阶段 | 参与者 | 产出 | 结果 |
|------|--------|------|------|
| 规划 | 3 × team-planner（独立） | plan-1/2/3.md | 共识：方案 A（可后台化前台 job，现场续跑）优于 CCB signal-race 重启 |
| 定稿 | 父代理 | plan.md | 裁决：检查点 sentinel 交接、复用 StartForSession、REASONIX_AUTO_BACKGROUND_MS |
| 执行 | 4 × team-executor（T1→T2→T3→T4 依赖链） | jobs 原语 / 检查点+交接 / controller+slash / config+ablation | 实现落盘，诚实报告无 shell 未验证 + T4 write_paths 缺口 |
| 补执行 | 父代理 | 真实验证 + T4 接线（agent Options/boot/自动触发）+ 2 处根因修复 | 全部通过 |

## 二、父代理补跑发现的真实缺陷（防幻觉价值）

1. **UnixMilli 单位错误**：`activeTurnCreatedAt` 存毫秒，自动触发误用 `time.Unix(0, ...)`（纳秒）→ 解析成 1970 → 所有 turn 立即后台化（boot 测试全挂）→ 修复为 `time.UnixMilli`
2. **测试清理竞态**：`TestBackgroundizeIdempotent` 缺 `waitIdle`，turn goroutine 未退出致 TempDir 清理失败 → 补 waitIdle
3. **boot 调用点错误**：`cfg.Agent.ForegroundBackgroundize()` 应为 `cfg.ForegroundBackgroundize()`
4. **banner 违规**：执行队 5 处 `// ---- xxx ----` 分隔线 → 全部清理

## 三、最终实现

- **jobs**：`StartForeground(ForSession)` + `foregroundClaimPending`（抑制 P1 信封）+ `ClaimForegroundResult`（证据桥接，复用 collectBackgroundEvidence 模式）
- **agent**：`BackgroundizeSignal`（幂等 one-shot）+ runToolLoop 迭代边界检查点（`errBackgroundizeRequested` sentinel）+ 交接串行点（同 goroutine：MarkRunning → StartForSession → resume 续跑同一内存 Session，跳过 prompt add）+ 父 cancel 传导（AfterFunc 不短路）
- **自动后台化**：`REASONIX_AUTO_BACKGROUND_MS`（默认 120000/0 禁用）+ `ablation.AutoBackground` + config `foreground_backgroundize_seconds`
- **入口**：`/background` slash + `Controller.Backgroundize` + `ForegroundTaskState` 状态暴露（idle/running/backgroundize_requested）

## 四、验证（全部真实执行）

```
go build ./...                                          ✅
go test ./internal/jobs/ -race                           ✅（foreground 原语）
go test ./internal/agent/ -run 'Backgroundize|Task' -race ✅（检查点/交接/续跑）
go test ./internal/control/ -count=1                    ✅（/background + 状态）
go test ./internal/config/ ./internal/ablation/          ✅（阈值 + 开关）
go test ./internal/boot/ -count=1                        ✅（自动触发不误判 + 契约）
go vet + gofmt + repolint                                ✅（banner 清零，功能增长入 baseline）
```

## 五、缓存红线确认

- 续跑同一内存 Session + 跳过 beginRunTurn prompt add → 前缀逐字节稳定、0 额外 cache miss ✅
- 交接串行点无双跑（前台 run loop 完全 return 后才 StartForSession）✅
- 禁用开关（ablation/config 0）时字节级等价旧路径 ✅

## 六、遗留（后续阶段）

- T0 探查结论：turn-busy 时 `/background` 走 steer 式紧急命令通道（已确认 TrySteer 直通，接线完成）
- 自动阈值激进性（首轮采样超时立即后台化）为已知接受项，增强项「至少 2 轮」未纳入
- UI 接线（TUI/desktop 按钮）后续；ACP session_backgroundize 可选
- 下一事务：P2（任务管理面板）或 P5（fork）

## 七、fable5 独立审查（2026-08-10 追加）——2 个严重缺陷修复

独立审查子代理对 P1-P4 全 diff 审核，评分 6.5/10，发现 2 个 blocking 缺陷（均在 P4）：

1. **P4 后台任务被父 turn 生命周期绑架**（task.go 原 `AfterFunc(ctx, cancelRun)`）：turn 正常结束即杀后台任务，与「runs across turns」矛盾。
   → 修复：job runCtx 只基于 jobCtx（kill/close 传播），detach 父 turn；测试改为 `TestBackgroundizeSurvivesParentTurnCancel`（父结束存活 + kill 生效）。
2. **P4 sentinel 在主 agent 无捕获者**（run_loop 返回 errBackgroundizeRequested，仅 task.go 两处捕获）：默认 120s 阈值下主 agent 长对话 turn 直接报错。
   → 修复：`WithForegroundTask` 标记只注入前台 task 子代理，`backgroundizeRequested` 非前台 task 恒 false；子代理继承阈值（TaskToolOptions.AutoBackgroundizeAfter）；补 `TestBackgroundizeCheckpointNoopOnMainAgent`。

警告修复：stalled 活动复位（jobWriter.Write 清 stalled）；desktop 错误文案。

验证：agent -race 全绿（含新测试）、jobs/control/boot 全绿、repolint clean。
