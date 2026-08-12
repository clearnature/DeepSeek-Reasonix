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

### P3 降级评估（2026-08-12 结论：降级为观察项，不立即实现）

**理由（与上游差异度 / 合并风险 / 维护难度 / 必要性四维）：**
1. **与上游结构性偏离**：投影归属双方均为 per-Agent `CompactionState`（上游 projection.go:111，我们合并保留的结构）；Session 层投影是 dev 独有、上游无先例（上游 Session 无投影字段）。
2. **合并风险高**：P3 改动文件（session.go/projection.go/compact_projection.go/context_manager.go）正是上游 151 commits 中 #8112/#8244/#8419 密集迭代区——改投影归属后，下次上游同步冲突面从"功能叠加"变"架构分歧"，违背并轨方针（上游 #8112 事务收敛为正式实现）。
3. **维护难度高**：Session 层投影需引用计数/失效传播（多 agent 并发 append → InputHash 失配）/与 per-Agent CompactionState 双轨并存——状态维护成本高（用户此前已指出此顾虑）。
4. **必要性存疑**：8/12 现象已由问题 1（est 实测 f105a0454）+ 问题 2（fork 继承 27d6cb0f5）闭环；teammate 实际是 fork 前缀 + 自己 transcript（顺序/只读），不存在真实的并发共享写 leader Session 路径——P3 是为假设场景付架构成本。

**触发条件（满足任一才重新评估）：**
- 出现真实的多 agent **并发共享写**同一 Session 场景（如 team 子代理共享写上下文、并行 teammate 共享 leader transcript）
- 现有 per-Agent 投影 + LastReceipt/InputHash 失效检测（重建成本）被证实无法兜底（如重建频率导致成本失控）
- 上游 main-v2 出现 Session 层投影/共享写架构（届时跟随上游，而非 dev 单方引入）

**过渡保障**：现有投影失效检测（LastReceipt/InputHash）已能兜底多 agent 场景（代价是重建），不阻塞任何功能。

### 投影占用/释放 × team 组队/解散生命周期
- 当前：组队（Create）无上下文；任务（Assign）fork 父投影视图（问题 2 已修）；解散（Remove）删元数据+tasks+worktree，transcript 由 SubagentStore 管理
- 观察：如 team 上下文资源池化需求出现（显式分配/释放/引用计数），按生命周期设计单独评估，不并入本事务
