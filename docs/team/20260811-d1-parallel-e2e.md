# D1 分支并行端到端实战证据（2026-08-11）

> 目标：验证 D1「分支并行 + 令牌」——teammate 并行写不再被整工作区 write-claim 串行化。
> 方法：3 个 teammate 按工作量分配（小/中/大），全部 worktree 模式，无依赖并行执行真实写码任务（真实 DeepSeek API）。

## 一、执行记录（bash-12，真实 API）

```
19:27:27  /team-create dev1/2/3 coder
19:27:30  /team-grant ×3 worktree
19:27:30-31  /team-add ×3（按工作量：dev1=3 函数 utils.go / dev2=6 函数 stringsx.go / dev3=10 函数 mathx.go）
19:27:45  ✅ 并行确认：3 个 teammate 同时 running（分配后 15s）
19:28-32  dev1/2/3 各自写 worktree（utils.go 19:27 / stringsx.go 19:30 / mathx.go 19:30）
19:33:01  全部完成：task-1/2/3 → 3×idle
```

## 二、验证证据（4/4）

| 检查 | 结果 | 证据 |
|---|---|---|
| 并行执行 | ✅ | roster 同帧 `dev1/2/3  running`（job=task-1/2/3 同时 running） |
| 按工作量分配 | ✅ | 3 任务规模差异化（3/6/10 个函数），各自独立完成 |
| 产出 | ✅ 3/3 | dev1 `utils.go`（370B）、dev2 `stringsx.go`（1953B）、dev3 `mathx.go`（2033B）——均在各自 worktree `.reasonix/worktrees/<name>/` |
| 隔离 | ✅ | 主 checkout 零污染（3 个文件无一泄漏） |

## 三、D1 提交链（dev/clearnature）

| commit | 内容 |
|---|---|
| `a09b2ec61` | **D1 令牌**：TeammateStore.grant（MiMo grant 表模拟）+ Grant/Revoke + Assign 注入 `WritePathSet` 精确 claim + `/team-grant <name> <path...>` |
| `cccbe09ce` | **D1 worktree 隔离**（CCB worktree.ts 模拟）：`team_worktree.go`（create/remove/change-detect/sanitize）+ GrantWorktree + Assign 创建 + Remove 清理 + `/team-grant <name> worktree` |
| `57d76154a` | **修复 fork 写 gate**：令牌 teammate 需 `Context.Writable=true`（gate 放行）+ WritePathSet 绑定（路径限制）——端到端复现 `write_file blocked: fork sub-agents are read-only` 后修复 |

## 四、端到端过程中发现的问题与修复

**D1 令牌 teammate 的 fork 仍只读**（`57d76154a`）：
- 现象：worktree 创建 ✓、隔离 ✓，但 teammate 写文件被拒（`fork sub-agents are read-only`）
- 根因：D1 令牌设了 `Grant.WritePaths + ReadOnly=false`，但 fork 执行 gate 键控 `spec.Context.Writable`（漏设）——令牌 teammate 的 fork 仍走只读 gate
- 修复：`canWrite = tm.Writable || !paths.Empty()` → `Context.Writable`——gate 放行 + WritePathSet 绑定 = **受限写**（不是全放开）
- 验证：修复后 teammate 真实写入 worktree（quicksort.go 端到端 + 3 并行端到端）

## 五、使用方式（新版本 desktop）

```
/team-create dev1 coder && /team-create dev2 coder
/team-grant dev1 worktree && /team-grant dev2 worktree   # 分支并行模式
/team-add dev1 <任务A> && /team-add dev2 <任务B>          # 无依赖 → 并行
/team-status                                             # 看多个 running 同帧
/team-remove dev1                                        # 自动清理 worktree + team-dev1 分支
```

## 六、结论

D1 完整可用：**令牌（精确路径写）+ worktree（分支物理隔离）**——teammate 并行写不再被整工作区 claim 串行。
无依赖任务天然并行（P6.1 依赖树门控已有：有依赖走 dependsOn 串行，无依赖全并行）。

## 七、后续（D1 增强候选）

1. worktree 完成自动清理（CCB 完整语义：`worktreeHasChanges` 无变更删 / 有变更保留回报路径，当前 Remove 时清理）
2. 令牌/分支策略配置（per-task vs per-teammate、动态并发分配 D2）
