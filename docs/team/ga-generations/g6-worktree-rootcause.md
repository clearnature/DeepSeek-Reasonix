# GA 代际记录：G6（worktree 多文件失败根因排查）

## 背景

G5 意外：worktree 组多文件任务 1/3 完成，path 组 3/3——疑似 path 更优。需排查 worktree 失败根因。

## 根因（transcript 实证）

G5-A 未产出个体（g1）的 write_file 路径：**`.reasonix/worktrees/g1/sortx/order.go`**——子代理按 Go 惯例（"package sortx"→ sortx/ 子目录）**创建了 sortx/ 子目录**，文件写进子目录；evaluate 检查目录根 `order.go` 找不到 → fitness 0.0。

**结论：worktree 失败 = 任务措辞歧义（"package sortx"诱导建子目录），非 worktree 隔离缺陷。** G5 的"path 完成率更优（3/3 vs 1/3）"是实验设计巧合（path 组恰好写根），**不成立**。

## 修复

任务措辞明确："所有文件**直接放在该目录根，不要创建任何子目录**"——消除歧义。

## 结论

1. **隔离方式（worktree/path）不影响任务完成率**——G5 的完成率差异是措辞伪影
2. path-grant 无缓存污染结论（G4/G5 缓存指标）**不受影响**（缓存分析独立于完成率）
3. 修正措辞后 worktree/path 应等价——如需可复测

## 代际路径更新

G2-G5 结论保持（natural ask 最优、path 无缓存污染）；G5 完成率差异勘误（非隔离因素）。
