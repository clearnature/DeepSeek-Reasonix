# 执行报告：output_budget admission 信任观察值（反复压缩修复）

> 事务：docs/team/20260812-output-budget-admission/ · 分支：dev/clearnature · 状态：✅ 已执行
> 执行：Team Executor（fable5 Step 4-6）· 依据：plan.md + 纪律章程

## 一、实现

### T2 修复（internal/agent/output_budget.go:246-254）
`effectiveOutputBudget` 在无校准时，若 `lastUsage` 存在且 `LatestPromptTokens() > 0`，用观察值覆盖 wire-char 估算：

```go
est := a.estimatedRequestTokens(req)
// Admission trusts the last observed prompt size over the wire-char
// estimate: fresh fork agents lack calibration and the 0.25 fallback
// inflates dense sessions ~2x, falsely reporting shared-window overflow.
if u := a.lastUsage.Load(); u != nil {
    if pt := u.LatestPromptTokens(); pt > 0 {
        est = pt
    }
}
```

### T3 测试（internal/agent/output_budget_test.go:654）
`TestEffectiveOutputBudgetUsesObservedTokensWhenCalibrationAbsent`：无校准 agent + lastUsage 观察值 858K + 500K 字符 code/JSON 消息 → 不误报 overflow。

## 二、验证（全部真实执行，可重跑）

| 验证 | 命令 | 结果 |
|------|------|------|
| 定向测试 11/11 | `go test ./internal/agent/ -run 'TestEffectiveOutputBudget\|TestCalibratedOutputBudget\|TestSharedWindow\|TestPrepareSamplingRequest' -count=1 -v` | PASS (0.448s) |
| agent 全量 | `go test ./internal/agent/ -count=1` | ok (23.6s，3/5 次通过；2 次 `TestTeammateConcurrencyBurst` 30s 超时为负载 flaky，见下) |
| control 依赖方 | `go test ./internal/agent/ ./internal/control/ -count=1` | ok (23.9s + 10.9s) |
| 全仓编译 | `go build ./...` | exit=0 |
| vet | `go vet ./internal/agent/` | 无告警 |
| gofmt | `gofmt -l internal/agent/output_budget.go internal/agent/output_budget_test.go` | 空（合规） |
| repolint | `go run ./tools/repolint` | clean (2121 baselined findings) |

### TestTeammateConcurrencyBurst flaky 甄别（防虚假完成）
- 单独跑 5 次：全部通过（0.771s）
- 干净 HEAD（bfd2c8888 worktree）全量 2 次：通过（22.5s / 22.6s）
- 当前工作区全量 5 次：3 通过（23.6-23.9s）、2 失败（`TestTeammateConcurrencyBurst` 30.05s 超时）
- 结论：该测试为 6 个 teammate 并发 Assign + 30s 等待窗口 + worktree/git 操作，对系统负载（load 2.0+）敏感；本次改动仅触及 output_budget.go（admission 侧），与 teammate 并发无数据路径交集，非本改动引入

## 三、变更文件

```
internal/agent/output_budget.go      | 8 ++++++++   （admission 信任观察值）
internal/agent/output_budget_test.go | 12 ++++++++-- （新增回归测试 + gofmt 双空行修复）
docs/team/20260812-output-budget-admission/plan.md + execution.md（本事务文档）
```

## 四、缓存红线确认

- D1 前缀字节：admission 侧本地估算，零发送侧改动 → 安全
- 与 `ContextPreparePolicy.ObservedInputTokens`（context_manager.go:83）语义一致，非新机制

## 五、遗留事项（交纪律团关注）

1. `TestTeammateConcurrencyBurst` 全量负载 flaky（30s 窗口过紧）——既有问题，非本次引入，建议后续加长窗口或加 `-p 1` 隔离
2. HEAD 遗留 gofmt 问题：`internal/agent/run_usage.go`、`internal/agent/team_worktree.go` 在 HEAD 即不合规（非本事务引入，未扩大范围处理）
3. 未跟踪文件 `cmd/team-ga2/`（GA 引擎实验）与 `docs/team/20260812-discipline-charter.md`（纪律章程）非本事务范围，未纳入提交
