# P2 任务管理面板：执行与审查报告

> 事务：docs/team/20260810-p2-taskpanel/ · 分支：team · 状态：✅ 已执行并通过纪律审查

## 一、流程记录（按团队章程）

| 阶段 | 参与者 | 产出 | 结果 |
|------|--------|------|------|
| 社区调研 | 2 子代理（GitHub API + 内置检索） | 调研报告 | 确认无现成 TaskList/TaskStop；基础设施完备；#7962 要求 /status 扩展 + stop |
| 规划 | 3 × team-planner（独立） | plan-1/2/3.md | 共识：jobs 只读快照 + stop 收紧 KillForSession + 增强 TaskMonitorPanel |
| 定稿 | 父代理 | plan.md | 裁决：独立 Snapshot 类型、jobs>store 合并、tail 4KiB bounded |
| 执行 | 6 × team-executor（T0-T6 依赖链） | jobs/control/desktop/cli/前端 | 实现落盘，诚实上报无 shell 未验证 |
| 补执行 | 父代理 | 真实验证 + 3 处真实缺陷修复 | 全部通过 |

## 二、父代理补跑发现的真实缺陷（防幻觉价值）

1. **前端 locale 回归**（测试全挂根因）：Node 22 全局 `navigator.language` 为系统 zh；T4 给 zh 字典补键后，面板从英文 fallback 变中文 label → 既有测试期望英文全挂。修复：测试固定 `navigator.language = "en-US"`
2. **前端竞态（产品 bug）**：TaskMonitorPanel `useEffect` 依赖含 `expanded`——展开行触发 effect 重跑 → `fetchJobs` 重新 setJobs **覆盖** fetchJobOutput 刚刷新的 tail。修复：`expandedRef` 读展开态，effect 不重跑
3. **测试时序**：detail tail 刷新测试缺 flush（异步 setState）
4. **既有失败确认**：`TestClearMCPAuthenticationUsesControllerWorkspace`（anthropic provider 注册）为基线环境问题，该测试文件零改动，与 P2 无关

## 三、最终实现

- **jobs**：`JobSnapshot`（9 字段：id/kind/label/session/status/tail/stalled/interrupted）+ `JobSnapshotsForSession`（非消费 tail，锁序 m.mu→j.mu 不嵌套，含 running/terminal/tombstone）
- **control**：`JobSnapshots()` 包装（session 过滤）+ CancelJob 已确认收紧 KillForSession
- **desktop bridge**：`JobPanelJobsForTab`/`JobOutputForTab`（tab 路由 + 鸭子接口）
- **前端**：TaskMonitorPanel 增 kind badge/label/tail 行（512B 截断）/stalled 高亮（仅修饰 running）；types/bridge/useController 接线；三语 i18n
- **CLI**：/status 在 jobs 行后追加任务明细段（向下兼容，T2 未合入时自动降级）

## 四、验证（全部真实执行）

```
go build ./...                                        ✅
go test ./internal/jobs/ -race                         ✅（快照/非消费/stalled）
go test ./internal/control/ -count=1                  ✅（session 边界 + e2e）
go test ./internal/cli/ -run Status -count=1           ✅（明细段 + 降级 + 截断）
go test ./internal/boot/ ./internal/agent/ ...         ✅（全量回归）
desktop/frontend: tsc --noEmit                        ✅
desktop/frontend: TaskMonitorPanel.test.tsx           ✅ 23/23
go vet + gofmt + repolint                              ✅（essay 清零，功能增长入 baseline）
```

## 五、缓存红线确认

- **零发送侧变化**：面板/快照纯读取 + UI；不触 input.go/compose/system prompt/tools ✅
- **快照绝不消费 readOffset/resultRead/evidence lease**（T1 单测锁定）✅
- 不引入调度器（#3799）；queued 仅展示枚举 ✅
- 不转发 raw reasoning（#7372 原样保留）✅

## 六、遗留

- T0 结论：进度 tracker childID 与 job.ID 不同号（后续进度关联需映射）
- stalled 恢复检测（job 恢复可见输出后解除 stalled）列后续
- bot /status 不在范围（#3799 一致）
- 下一事务：P5（fork，省 80% token）或 P6（team 组织）
