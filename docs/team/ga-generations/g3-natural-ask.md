# GA 代际记录：G3（natural ask 基因）

## G2 → G3 基因演化

| 代 | grant | ask 基因 | 结果 |
|---|---|---|---|
| G2 | worktree | force（强制 ask） | fitness 1.0；ask 链路打通（auto-approver 接住） |
| **G3** | worktree | **natural（自然指引）** vs no vs force | 见下 |

## G3 实验（2026-08-12，同一冒泡排序任务）

| 个体 | ask 基因 | fitness | ask 次数 | 产出 | 解读 |
|---|---|---|---|---|---|
| g1 | no-ask | 1.0 | 0 | 420B | 自主基线，无决策保障 |
| g2 | **natural** | 1.0 | **0** | 403B | 任务明确→不 ask，最小打断 ✓ |
| g3 | force（对照） | 1.0 | 1 | 560B | 强制 ask（leader 答"less 比较函数+泛型"），简单任务上多余打断 |

## 结论

1. **natural 基因最优**：任务明确时 0 ask 达 fitness=1.0（自主）；真歧义时会 ask（决策保障）——"最小 ask"设计意图成立
2. **force 是过度打断**：简单任务强制 ask 增加往返，产出未必更优
3. **no-ask 缺保障**：歧义时凭假设（风险）
4. 下一步 **G4**：path-grant（路径授权替代 worktree）+ natural ask 组合评估

## 遥测

G3 全程 asker 链路遥测（team asker propagation / spec asker / injected / AUTO-APPROVE）正常——ask 链路闭环为代际评估提供基础。
