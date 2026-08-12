# 20260812 fork 继承校准（问题 2）

## 背景

8/12 凌晨 24 次 overflow 反复压缩（间隔最短 24s）。根因两层：
- 问题 1（已修，f105a0454）：`effectiveOutputBudget` 忽略 lastUsage 实测，fallback 高估 2.18x
- 问题 2（本事务）：fork 子代理继承父 Session 原始快照（未压缩全量 858K）+ 新 agent 无 lastUsage/calibration → 首请求 fallback 高估 → 反复误压缩

## 改动

| 文件 | 改动 |
|---|---|
| `internal/agent/subagent_fork.go` | ① `captureForkPrefix` 改用 `parent.modelVisibleMessages()`（投影有效则投影视图，与 Prepare 发送同源）替代 `session.Snapshot()` 原始快照；② 新增 `captureForkInheritance(parent, modelRef)`——model 一致才继承父 lastUsage/promptCalibration（值拷贝） |
| `internal/agent/task.go` | `RunSubAgentWithSession` 子 Agent 创建后，fork ctx（`ForkSourceFromContext`）存在时注入继承值 |
| `internal/agent/subagent_fork_test.go` | 新增 `TestCaptureForkPrefixUsesProjectedView`（父已压缩 → 前缀=投影视图含摘要）+ `TestCaptureForkInheritance`（model gate / 值拷贝 / 空值） |

## 设计要点

- **byte-identical 前提保持**：`modelVisibleMessages` = 父 Prepare 发送同源视图；投影有效时=投影（父压缩后发送的就是投影）
- **跨 model 不继承**：tokenizer 属性不可移植，`parent.modelRef != 子 model` 时冷启动
- **值拷贝**：继承的是原子快照拷贝，父后续更新不影响子
- **只读路径覆盖**：`RunReadOnlySubAgentWithSession` 委托 `RunSubAgentWithSession`，注入点全覆盖
- **父零改动**：captureForkPrefix 深拷贝保留；继承只读不写父

## 验证

- 定向：TestCaptureForkPrefix{ByteIdentical,UsesProjectedView,ParentSessionUntouched} + TestCaptureForkInheritance + TestTruncateUnfinishedTurn + TestCloneForkMessages 全 PASS
- fork 相关全量（TestCaptureFork|TestFork|TestSubagent）：ok 2.6s
- agent 全量：唯一 FAIL 为已知 flaky（TestTeammateConcurrencyBurst 30s 负载敏感，非本次引入，单独跑 PASS）
- go build ./... / go vet / golangci-lint 0 issues / gofmt clean

## 遗留（范围外）

- P3：共享 Session 投影持久化到 Session 层（多 agent 并发共享写的一致性）——后续事务
- 投影占用/释放纳入 team 组队/解散生命周期（用户指出为独立状态维护成本话题）
