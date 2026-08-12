# GA 代际记录：G4（path-grant vs worktree 缓存污染评估）

## 前提修复（本代前置）

subagent Usage 事件此前被 event.Discard 丢弃（job ctx 无 CallContext，subSink 回退）——teammate 请求不落 stats，缓存行为不可观测（G4 初测暴露：实验窗口 stats 0 行）。

修复（31f9bc8dc）：`ProfileExecSpec.Sink` 携带 leader Recorder-wrapped sink → tracker 用它 → 子代理 Usage 落 stats。

## G4 实验（2026-08-12，natural ask × 同一冒泡排序任务）

| 组 | grant | 请求 | prefix_hash 唯一 | 命中率 | miss | fitness |
|---|---|---|---|---|---|---|
| G4-A | worktree | 67 | 4 | 98.09% | 59,940 | 3×1.0 |
| G4-B | path（WritePathSet） | 71 | 3 | **99.33%** | 32,056 | 3×1.0 |

## 结论

1. **path-grant 无缓存污染**：prefix_hash 稳定（path=3、worktree=4，均每 teammate 一个），命中率 path 不劣于 worktree（99.33% vs 98.09%）——共享工作区的工具输出未抖前缀（授权路径隔离有效）
2. **功能等价**：path-grant fitness 1.0，无越权写入（隔离正确性 OK）
3. **成本**：path-grant 省 worktree 创建/清理（D1 开销）
4. **worktree 命中率略低**可能来自 git 操作/独立 checkout 的冷启动，非污染

## 局限

- 样本 ~70 请求/代（单任务），结论初步
- 复杂任务（多文件/长会话）的污染风险未覆盖——G5 建议用多文件任务复测

## 代际路径

G2（worktree+force）→ G3（worktree+natural 最优）→ **G4（path 可行，无污染）** → G5（容器隔离 vs path 的成本/隔离深度对比）
