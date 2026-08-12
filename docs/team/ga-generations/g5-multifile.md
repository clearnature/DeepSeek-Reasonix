# GA 代际记录：G5（多文件任务复测 path-grant 缓存污染）

## 背景

G4（单文件任务）结论 path-grant 无缓存污染，但样本小（~70 请求/代）、复杂任务未覆盖。G5 用**多文件任务**（order.go + order_test.go ≥3 测试 + README.md）复测。

## G5 实验（2026-08-12，natural ask × 多文件冒泡排序包）

| 组 | grant | fitness | 命中率 | miss | prefix 唯一 |
|---|---|---|---|---|---|
| G5-A | worktree | **1/3**（仅 g2 完成 3 文件） | 99.64% | 11,058 | 3 |
| G5-B | path | **3/3 全完成**（fitness 1.0） | 99.06% | 20,751 | 3 |

## 结论

1. **path-grant 多文件任务下仍无缓存污染**：prefix_hash 稳定（3 = 每 teammate 一个）、命中率 99.06%——G4 结论巩固
2. **意外**：path-grant 完成率 3/3 优于 worktree 1/3——复杂任务下 path-grant 反而更稳（worktree 的 git 隔离可能引入额外障碍；g1/g3 worktree 失败原因待查，可能子代理步骤耗尽/随机）
3. 两代 0 ask（natural 判断决策明确）

## 局限与下一步

- G5-A 的 1/3 完成率引入样本偏差（A 缓存数据来自 g2 单个体）
- worktree 多文件失败原因未深挖（建议单独排查：子代理 max_steps？worktree 交互？）
- **G6 候选**：容器隔离 vs path 的成本/隔离深度对比；或 worktree 失败根因排查

## 代际路径

G2(worktree+force) → G3(worktree+natural) → G4(path 单文件无污染) → **G5(path 多文件无污染 + 完成率更优)** → G6(容器对比 / worktree 失败排查)
