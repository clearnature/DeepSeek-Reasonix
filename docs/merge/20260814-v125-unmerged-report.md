# v1.25.x 剩余未合并清单——详细技术报告（2026-08-14）

基线 merge-base `1ac7817ff`；上游 26 提交；**已吸收 9**（主题合并 + release/test）；
**剩余未合 17**。已吸收：#8779/#8756/#8675/#8674/goal/v1.25.0-notes/v1.25.1-notes×2/#8790-bench。

---

## 主题 A：desktop 会话树/归档（15 个）——🔴 hold（#8739 同域未清）

| 提交 | issue | 改动 | 归属子域 |
|---|---|---|---|
| 9aaf8d381 | #8728 | 更新后会话消失且只剩前半段 | 会话恢复 |
| 42a9de71d | #8727 | 中途打开会话丢历史 + 隐藏 Auto Guard | 会话恢复 |
| ebdd48354 | #8736 | 点活会话附着当前 runtime（不再开旧副本） | 会话恢复 |
| c7bc2f3e1 | #8730 | delivery-check 暂停与 recovery 暂停分开标注 | 会话恢复 |
| 511d7ff74 | #8760 | 历史会话滚不到最后几行 | 会话恢复 |
| f525db811 | #8694 | 旧会话反复变成未读 | 会话恢复 |
| de3618b27 | #8812 | 归档对话时侧栏整组消失 | 归档/侧栏 |
| 0cc808a88 | #8766 | 归档后侧栏冒空白新会话 | 归档/侧栏 |
| 15abb0394 | #8772 | load-more 与整理历史反复循环 | 归档/侧栏 |
| 3acb3c5b0 | #8813 | phantom transcript bottom | 归档/侧栏 |
| b58e3e353 | #8808 | 工作台新建空白项目 | 工作台 |
| d42cfd194 | #8814 | 项目内会话排序即时刷新 | 树/排序 |
| 49f24d197 | #8758 | 关闭待办写入会话文件 | 待办 |
| 345a71dc7 | #8731 | 关闭待办后会话画面回跳 | 待办 |
| 6e8459c0a | #8669 | 流式列表中间闪源码 | 渲染 |

**hold 依据**：#8739 open（8/13 11:52 v1.25.0 启动 replace rev432 从 08-09 快照重建
510 msgs——recovery 分支有完整 1047 msgs——**UI 显示旧态数据丢失仍复现**）。
v1.25.1 的 #8728/#8727 是相关修复尝试，**但 #8739 明确仍复现**（其根因机制
"启动重建 recovery-aware"未实现）。**合入这 15 个 ≠ 修复 #8739**，且引入
"修复未修好"的中间状态 + 前端验证成本。

**合并条件**：① #8739 关闭（或 v1.25.2 实现启动重建 recovery-aware）；
② 或社区确认 #8728/#8727 实际缓解（新增复现数据）；③ 合并时跑
前端边界验证（tsc + Playwright bench + 侧栏场景）。

**文件面**：desktop/frontend 63 文件 + internal/sessioncatalog 9 + internal/agent 25（部分）。

---

## 主题 B：压缩估算（1 个）——🔴 第一风险域 hold（红区）

| 提交 | issue | 改动 |
|---|---|---|
| 7b82fc3bc | #8718 | web_search 重放排除出 compact 估算（compact.go + output_budget.go） |

**hold 依据（红区）**：与本地保护清单 5 文件交集（compact.go / output_budget.go /
sampling_request.go / projection.go / compact_projection.go）。2.1 判定 = **同向互补**
（上游 web_search 排除 + 本地 est 实测优先 f105a0454——不同缺口）——**可合**，
但按第 0 条（压缩/缓存 = 第一风险，触及必红）需 **team-planner 深入分析 +
守卫套件全量**（cachehit_e2e / healthy-window / RejectsExhaustedSharedWindow /
投影/校准测试）+ 四方审计。

**合并条件**：独立执行一次深入分析（双修复共存确认 + 守卫全量）——影响面小
（web_search 重放估算）但触及第一风险域，故不并入批量。

---

## 主题 C：inbox 隔离/恢复（1 个）——🟡 待评估

| 提交 | issue | 改动 |
|---|---|---|
| fd8d85ed6 | #8701 | 隔离会话队列 + 恢复暂停 inbox（internal/sessioninbox 4 + internal/agent） |

**待评估**：inbox 域与 dev 的 P6 team inbox（sessionDir/team-inbox）可能重叠——
需 2.1 差别识别（上游"隔离会话队列" vs dev 的 team mailbox 语义）——若互补可合。

---

## 汇总

| 主题 | 数量 | 风险 | 状态 | 解锁条件 |
|---|---|---|---|---|
| A desktop 会话树/归档 | 15 | 🔴（#8739 同域） | hold | #8739 关闭 / v1.25.2 / 社区确认缓解 |
| B 压缩估算 #8718 | 1 | 🔴（第一风险域红区） | hold | 深入分析 + 守卫套件全量 |
| C inbox #8701 | 1 | 🟡 | 待评估 | 2.1 差别识别（与 dev team inbox） |

**原则**：主题合并模式下，17 个未合 = "合了风险大于收益"（A：数据丢失域未清；
B：第一风险域需专门验证；C：需先差别识别）——**不是永远不合，是解锁条件未满足**。
