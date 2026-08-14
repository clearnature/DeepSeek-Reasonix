# 执行记录（execution.md）——方案 B 分阶段

日期：2026-08-14 | 分支：dev/develop

## 阶段 B1：✅ 完成（62cf18df0）

#8731 滚动部分（4 文件）：
- Transcript.tsx footer effect + useTranscriptVirtuosoScroll.ts viewport 检测
- 2 测试；tsc clean；提交 62cf18df0

## 阶段 B2：✅ 完成（b74a04591）

#8731 水合部分（场景比对 → 移植）：
- **比对结论**：dev 只有加载侧门控（hasReusableCachedTranscript——决定
  "要不要取新页"），缺应用侧拒绝——Retry/failed-clear 后新 fetch 返回
  更短同指纹页会回滚屏上长转录（#8731 场景——dev 未覆盖）
- **移植**：isStaleResidentProjection（上游 40 行）到 dev bool API——
  HydrateProjection/sameHydrateFingerprint/isStaleResidentProjection +
  HydrateLiveState 补 historyRevision/historyDigest + useController.ts
  应用点守卫 + dev 版测试 4 场景
- 验证：tsc clean；tsx 测试 4/4；提交 b74a04591

## 阶段 B3：⛔ 回退（#8758 hold——依赖链确认）

#8758 cherry-pick 无冲突（自动合并），但编译失败：
- `continuePathForOpen` undefined——该方法是上游**后加**的会话恢复
  retarget 逻辑（continuePathForOpen/continuePathForMissingParent——
  dev 的 session_canonical_retarget.go 只有旧版 resolveCanonicalSessionPath）
- **依赖链**：这些 retarget 方法属于主题 A 会话恢复域（#8728/#8736 等）
  = **#8739 域**——未吸收
- **结论**：#8758 不独立（同 #8814 教训）——移植依赖 = 部分吸收 #8739 域
  （违反 hold 纪律——#8739 根因未清）——**回退**（reset --hard b74a04591）

## 汇总

| 阶段 | 结果 | 提交 |
|---|---|---|
| B1（#8731 滚动） | ✅ | 62cf18df0 |
| B2（#8731 水合） | ✅ | b74a04591 |
| B3（#8758 持久化） | ⛔ hold（依赖 #8739 域 retarget） | — |

#8731 全部吸收完成；#8758 解锁条件 = #8739 域 retarget 逻辑先吸收
（或 #8739 修复后整体评估主题 A）。
