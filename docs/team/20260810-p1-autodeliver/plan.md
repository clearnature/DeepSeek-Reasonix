# 定稿：P1 后台结果自动投递（规划小组一致性裁决）

> 事务：docs/team/20260810-p1-autodeliver/ · 规划小组：plan-1.md / plan-2.md / plan-3.md（3 个 team-planner 独立产出）
> 一致性裁决：父代理汇总 3 份方案 → 共识采纳 + 分歧裁决 → 本定稿

## 一、一致性结论

3 份独立方案在以下 4 点**完全共识**：
1. **信封结构**：`<background-job-result>` 结构化信封，字段 task_id/status/label/artifact/结果正文（bounded）/usage 预留
2. **bounded 策略**：三层上限——4096 字节/条（rune 截断 + `[truncated…]`）、8 条/次（余留下轮）、16KB/块（丢最旧 + `[N more queued]`）
3. **wait/bash_output 兼容**：只读快照，不消费 `readOffset`/`resultRead`、不触碰 evidence lease
4. **锁序约束**：recordCompletion 内 `j.mu` 临界区拷贝、`m.mu` 独占 append；`-race` 验证 `writeJobMetaLocked` 无反向嵌套

## 二、分歧裁决（注入落点）

| 方案 | 落点 | 裁决 |
|------|------|------|
| plan-1/2 | 移到父 turn 尾部（recall 之后） | ❌ 否决：引入新剥离逻辑 + `stripTrailingMemoryRecall` 顺序陷阱风险 |
| **plan-3** | **保持 `input.go:191-195` 的 `<background-jobs>` 容器与注入位置不变，容器内升级为结构化信封** | ✅ **采纳**：preview/strip 剥离路径零改动、消解 UI 泄漏、最小侵入 |

**裁决理由**：缓存红线要求「位置固定、不插入历史」——保持现有 compose 布局完全满足，且避免 plan-1/2 必须新增的尾部剥离逻辑（memory-recall 泄漏同类事故风险）。

## 三、定稿设计

### 信封格式
```xml
<background-jobs>
<background-job-result task_id="task-1" status="completed" label="调研" artifact="…path…">
<output>…bounded 结果正文（XML 转义）…</output>
</background-job-result>
<result-overflow count="2"/>
</background-jobs>
```
- 属性/正文全部 XML 转义（防伪造关闭标签）
- 失败任务用 `<error>` 替代 `<output>`
- `usage` 字段预留省略（真实 usage 采集归 P2）
- 降级摘要保留 `use bash_output or wait for full output` 指引

### 注入语义
- 替换 `input.go:191-195` 当前的一行 `<background-jobs>` 摘要块（`DrainCompletedNoteForSession`）
- 每轮最多 8 条（部分 drain，余留下轮），超 16KB 丢最旧 + overflow 计数
- 位置固定、不修改 canonical 历史、`ComposeSynthetic` 不投递

### 只读快照
- 新增 `ResultSnapshotForSession`（jobs 包）：同源构造信封正文，不消费 `readOffset`/`resultRead`，不触碰 evidence lease
- status 用参数 `st`（`recordCompletion` 置终态前调用），`st.String()` 保 `jobs_test.go:325 Contains("Done")` 兼容

## 四、任务分解（执行小队）

| 任务 | 内容 | 文件 | 验证 |
|------|------|------|------|
| T1 | jobs 结构化完成记录 + 只读快照函数 | `internal/jobs/jobs.go` | `go test ./internal/jobs/ -race` |
| T2 | input 注入接线（容器内信封渲染） | `internal/control/input.go` | `go test ./internal/control/` |
| T3 | bounded 单测（截断/条数/总量/丢弃/XML 转义） | `internal/jobs/jobs_test.go` | 新增单测通过 |
| T4 | 剥离回归（preview/strip 零改动验证无泄漏）+ wait/bash_output 兼容回归 | `internal/agent/preview.go` 回归 | `go test ./internal/agent/ -run Preview` |
| T5 | 端到端（后台 job 完成 → 下一轮信封注入） | 集成测试 | 定向 e2e 测试通过 |
| T6 | 文档更新（本事务状态） | `docs/team/…/` | 文件存在 |

## 五、缓存/纪律检查点

- 发送侧仅新 user turn 的 `<background-jobs>` 块内容变化（一行摘要 → 结构化信封）；稳定历史前缀零变化 ✅
- 不插入历史、不重写 canonical ✅
- 锁序经 `-race` 验证 ✅
- 执行队不越权：T1/T2 分工 disjoint，T3-T5 依赖前序
