# P1 后台结果自动投递：执行与审查报告

> 事务：docs/team/20260810-p1-autodeliver/ · 分支：team · 状态：✅ 已执行并通过纪律审查

## 一、流程记录（按团队章程）

| 阶段 | 参与者 | 产出 | 结果 |
|------|--------|------|------|
| 规划 | 3 × team-planner（独立） | plan-1/2/3.md | 高度一致，分歧（注入落点）由父代理一致性裁决 |
| 定稿 | 父代理 | plan.md | 采纳 plan-3（容器位置不变 + 内部升级），否决移尾部方案 |
| 执行 | 3 × team-executor（T1→T2→T3/T4/T5 依赖链） | jobs.go / 测试 | T1 ✅、T2 发现接口缺口 blocked（诚实上报）、T3/T4 静态推演 |
| 纪律审查 | 6 × team-discipline（独立） | d1-d6/review.md | **6/6 一致裁决**：方案 A 采纳、事务驳回（半成品） |
| 补执行 | 父代理（按纪律团签名级方案） | jobs.go 完整实现 + 测试 | 全部通过 |

## 二、执行中发现并被纪律团确认的真实缺陷

1. **接口缺口**：T1 只做了单条信封函数（`ResultSnapshotForSession`），但 jobs 缺「部分 drain / 待投递列表 / 16KB 聚合」——T2 无法接线（执行队诚实 blocked，未越权）
2. **m.mu 自死锁风险**：drain 内调用 `ResultSnapshotForSession`（内部 `m.get` 持 m.mu）会自死锁——纪律团 d2/d6 独立发现
3. **XML 转义膨胀**：转义前截断 → 单信封最坏 ~20KB（纪律团 d3 发现）
4. **completion 缺结构化字段**：无 envelope 载体（d5 发现）
5. **编译错误**：`st.String()`（Status 是 string 类型无该方法）——父代理补跑验证发现并修复

## 三、最终实现（方案 A）

- `recordCompletion` / `recordStalled`：j.mu 临界区预渲染 `<background-job-result>` 信封（单真源）
- `DrainCompletedNoteForSession`：部分 drain（≤8 条/轮，余留下轮）+ 16KB 块上限（丢最旧 + `<result-overflow count>`），签名不变 → input.go 零改动
- `input.go`：`<background-jobs>` 容器与注入位置不变（preview/strip 剥离路径零改动）→ 缓存红线不破
- bounded：4096 字节/条（rune 安全 + `[truncated…]` marker 计入预算）、8 条/次、16KB/块

## 四、验证（全部真实执行）

```
go build ./...
go test ./internal/jobs/ -race -count=1          ✅（含新增 7 个测试）
go test ./internal/control/ -count=1              ✅（含 e2e：compose 自动注入信封）
go test ./internal/agent/ -run Preview -count=1   ✅（剥离回归）
go test ./internal/tool/builtin/ -run 'Bg|Wait'   ✅（wait/bash_output 兼容）
go run ./tools/repolint                            ✅（baseline 更新，essay 4099→4096）
```

## 五、缓存红线确认

- 发送侧仅新 user turn 的 `<background-jobs>` 块内容变化（一行摘要 → 结构化信封）
- 稳定历史前缀零字节变化；不插入历史、不重写 canonical ✅
- `ResultSnapshotForSession` 只读（不消费 readOffset/resultRead、不触碰 evidence lease）✅

## 六、遗留（后续阶段）

- usage 字段预留省略（真实 usage 采集归 P2）
- T5 e2e 覆盖 compose 单层；完整 boot 级 effect test 可后续补
- 下一次事务可用同一团队流程推进 P2（任务管理）
