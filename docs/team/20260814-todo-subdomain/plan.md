# 待办子域吸收计划（#8758 / #8731）——方案 B 分阶段

日期：2026-08-14 | 分支：dev/develop | 基座：1ac7817ff + #8669（307fd1515）

## 目标

上游 v1.25.x 主题 A（desktop 会话树/归档）中待办子域 2 个提交的分阶段吸收：
- **#8731**（345a71dc7）关闭待办后会话画面回跳——滚动/水合两部分
- **#8758**（49f24d197）关闭待办写入会话文件——Go 侧边持久化 + 前端传递

## 拓扑扫描（关键结论）

| 提交 | 滚动部分（🟢 独立） | 水合/持久化部分（🟠 依赖链） |
|---|---|---|
| #8731 | Transcript.tsx + useTranscriptVirtuosoScroll.ts + 2 测试——dev 零改动，符号齐备 | hydrateHistoryApply.ts（上游三态 API vs dev bool API）+ useController.ts（dev 平行机制）——依赖 #8727（未吸收） |
| #8758 | bridge.ts/todoVisibility.ts/types.ts/App.tsx 低冲突 | branch.go（写权限域）+ app.go Meta + useController.ts metaFromTab 语义合并 |

**依赖链教训（#8814）**：水合部分建立在 #8727 的 `hydratedHistoryApplyMode` 上——不独立；滚动部分真独立。

## 缓存红线声明（D1）

- #8731/#8758 **零发送侧前缀字节变化**：纯前端渲染/水合 + 会话 sidecar 元数据（`dismissed_todo_batches`，omitempty 向后兼容）
- `branch.go` 为会话写权限域（#8457/#8599 所在）——**改动必须跑会话守卫**（branch.go 全量测试 + #8457/#8599 回归）
- 不触碰 compact.go/output_budget.go/sampling_request.go/projection.go/prefix_hash

## 分阶段划分（每阶段独立验证点）

- **阶段 B1**：吸收 #8731 滚动部分（4 文件干净）——验证：tsc + transcript-scroll-release 测试
- **阶段 B2**：场景比对——dev `hasReusableCachedTranscript`（加载侧门控）vs 上游 `isStaleResidentProjection`（应用侧拒绝）是否覆盖「更短 same-fingerprint 页面回滚」；不覆盖则移植（~30 行 + 1 测试）——验证：水合测试 + 无双门控冲突
- **阶段 B3**：#8758 会话侧边（新增 Go 文件 + branch.go 字段 + app.go Meta）——验证：go test dismissed + branch 守卫
- **阶段 B4**：#8758 useController.ts 语义合并（与 dev #8701/#8665 演进共存）——验证：use-controller-meta 测试 + dev 独有回归
- **阶段 B5**：全量验证链 + 更新合并报告（主题 A 拆出待办子域）

## 对抗自检风险点（每阶段 execution 附）

1. 滚动部分"干净"是假象（stick 语义 dev 历史偏移）→ 跑 Playwright 滚动场景
2. "dev 已覆盖水合"可能自欺（加载侧 vs 应用侧竞态窗口）→ B2 场景实证
3. #8758 依赖 #8731（体验补全）→ 拆分边界不制造"修复一半"
4. branch.go 写权限域（dedup/上限与 revision/digest 合并交互）→ 跑全量守卫
5. repolint 预算（上游靠删注释压线）→ 每阶段 commit 内跑 repolint
