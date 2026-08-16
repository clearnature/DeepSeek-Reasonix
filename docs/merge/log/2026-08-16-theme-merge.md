# 2026-08-16 主题合并（v1.25.3 部分吸收）

## 决策
方案 A 主题合并：🟢 低风险先行 + 🟡 核对 + 🔴 hold（纪律红区不碰）。

## 已吸收（12/17）
| 提交 | 主题 | 冲突处理 |
|------|------|----------|
| 105e7853f | #8795 README 中文 | 无冲突 |
| 183fae98b | #8933 更新日志 | 无冲突 |
| 1a16cb7be | #8934 release blockers | constraints.go 采用上游（scope 约束解析）；constraints_test.go 删（依赖 #8930 类型——吸收后恢复） |
| b8f4eb1ba | #8937 更新日志 | 无冲突 |
| 86819d73d | #8916 ACP | 无冲突 |
| ff44cff71 | #8935 DOMPurify 安全 | 无冲突 |
| e4ce5343f | #8886 SCNet 预设 | 无冲突 |
| a733c4243 | #8920 标题统一 | 无冲突 |
| d4c87774a | #8913 SSH 窗口 | baseline.json theirs |
| 1359c7e6f | #8922 嵌套工作区 | 无冲突 |
| 6d9e776d0 | #8824 侧边栏本地化 | 无冲突 |
| 91bde0657 | #8925 滚动回跳 | 无冲突（与本地 isPinned 共存——测试过） |

## hold（5——红区纪律）
- 314beed8f #8930 fact-driven 重构（与 #8866 观察期叠加）
- 352d171b3 #8874 溢出恢复（与本地 BlockedInputHash 双实现——需深入）
- d473b5a9b #8921 预算抽象（与本地 est 实测——需深入）
- 515026e11 #8924 推理回放（社区 #8942/#8943 报回归——hold）
- f60ab17b0 #8923 投影保留（与本地第三态/语义哈希——需深入）

## 验证
- Go：build/vet/agent 88.7s/config/acp/runtimepolicy 全过
- 前端：tsc clean + 滚动 20/20 + 行 53/53（虚拟化 #8903 已知待排）
- repolint clean（1441）
- wails build（后台进行中）

## 备注
- #8934 的 constraints_test.go 依赖 #8930（hold）——恢复条件：#8930 吸收后 git checkout 上游测试
- 已排除 3 个等价提交（#8903/#8901/#8884——之前吸收）
