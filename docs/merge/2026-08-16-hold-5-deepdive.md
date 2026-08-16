# 2026-08-16 hold 5 个深入分析（fable5）

## 压缩域 3 个（重点）

### #8923 重启保留投影（f60ab17b0）——✅ 可吸收
- 上游：InvalidateProjectionIfStale（回退时保持）+ migrateLegacyCoveredPrefixHash + CoveredCount 冻结 fold 边界
- 对比：key 软条件（空/不兼容跳过——与我们 model-switch 兼容）；migrate 不动我们语义哈希字段；
  projectionContentValid 上游失配即失效（无 prune 容忍）——我们语义哈希增强叠加
- 关系：【互补】有效保持（上游）+ 失效降级（我们第三态）+ prune 容忍（我们语义哈希）
- 合并方式：逐函数（projection.go 两边 60+ 行）——投影测试全套验证
- 风险：🔴 高（投影核心双实现）→ 需投影全套测试（projection_restart + 我们 ToleratesPruneRewrite + 第三态）

### #8921 共享窗口预算 provider-aware（d473b5a9b）——⏸ 可吸收（useObserved 叠加）
- 上游：contextAdmission 新架构（learned 窗口/ContextBudgetPolicy/LastRecovery）+ 桌面 ContextBudgetCard
- 对比：上游 3 参 effectiveOutputBudget（无 useObserved）vs 我们 4 参（summarize false/主 true——8/12 修复）
- 关系：上游 learned 是"窗口观察"（模型能力）——我们 useObserved 是"prompt 实测"（请求大小）——不同概念
- 合并方式：output_budget.go +316 vs 我们 +40——逐块（上游架构 + useObserved 叠加）
- 风险：🔴 高（D7 红线 max_output_tokens 语义专项——发送侧字节）

### #8874 连续溢出恢复（352d171b3）——✅ 可吸收（互补）
- 上游：sameTurnCompactionBlocked（lastTurn 同轮 + mustFree 需 prior 进展）
- 对比：我们 BlockedInputHash（generation-scoped 内容门——失败后同 summary 不重复付费）
- 关系：【互补】同轮进度门（上游）+ 同内容门（我们）——不同维度
- 合并方式：context_receipt/context_manager/compact_projection 逐块 + 四方审计（OutputHash vs BlockedInputHash 共存）
- 风险：🔴 高（抑制双实现——静默失效风险——8/13 教训）

## 内核 2 个

### #8924 推理回放（515026e11）——⛔ hold（社区 #8942/#8943 报回归——上游没修好）
### #8930 fact-driven 重构（314beed8f）——⏸ hold（与 #8866 观察期叠加；#8934 测试依赖其类型）

## 决策矩阵
| 提交 | 决策 | 前置条件 |
|------|------|----------|
| #8923 | ✅ 吸收 | 投影全套测试绿 + 逐函数合并 |
| #8921 | ⏸ 吸收 | useObserved 叠加 + D7 专项 |
| #8874 | ✅ 吸收 | 四方审计 + 失败落盘保持 |
| #8924 | ⛔ hold | 上游修好 #8942/#8943 |
| #8930 | ⏸ hold | #8866 观察期结束 |

## 执行结果（2026-08-16 完成）

- **#8923** ✅ 吸收（445c28db8）：上游 InvalidateProjectionIfStale/migrateLegacy/onProjection 边界 + 我们语义哈希/第三态/孤儿锁保留（上游清投影弃用、key 失配测试弃用）
- **#8921** ✅ 吸收（0b4b23e63）：上游 contextAdmission/admitOutputBudget + 我们 useObserved（4 参——summarize false/主 true）+ summaryInputBudget 扣 prefix + jobs import 恢复
- **#8874** ✅ 吸收（9cab85a3a）：上游 sameTurnCompactionBlocked（同轮进度门）+ 我们 BlockedInputHash（内容门）共存——四方审计无冲突
- **验证**：agent 全量 28.5s + 前端（滚动 20/行 53/ContextBudgetCard 13）+ vet/repolint（1442）+ wails
- **遥测**：失败落盘/status=failed/est 覆盖/校准链全保持（用户"不要丢点位"）
