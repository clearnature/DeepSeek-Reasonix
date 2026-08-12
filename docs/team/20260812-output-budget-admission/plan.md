# 计划：output_budget admission 信任观察值（反复压缩修复）

> 事务：docs/team/20260812-output-budget-admission/ · 分支：dev/clearnature · 状态：已批准
> 关联问题：8/12 凌晨 00:06-00:58 出现 24 次 overflow 反复压缩（stats 实证，间隔最短 24s）
> 依据纪律章程：docs/team/20260812-discipline-charter.md（B1-B9 + 验收项 1-6）

## 一、问题与根因（拓扑扫描 B2）

### 数据实证（~/.reasonix/stats/2026-08-12.jsonl）
- 00:06:39 起 24 次 `trigger=overflow mode=summarized src≈1.55M`，压缩后**立即**又溢出
- 与压缩同秒存在恒定 `prompt≈858K` 的请求（hit 从 447K 增长到 875K，唯一 ph=None）
- 858K 真实 prompt 被 wire-char × 0.25 估算成 ~1.57M → 超过 1M context window → 误报 shared-window overflow → 触发 compaction → 压缩后该请求仍以 858K 重发 → 反复

### 代码路径
- `effectiveOutputBudget`（internal/agent/output_budget.go:238）：无校准时走 `estimatedRequestTokens` → `estimatedShapeTokens` → wire-char × `fallbackTokPerChar`(0.25)
- fresh fork agent（teammate 子代理）无校准（`promptCalibration` 为空）→ 落入 0.25 回退
- 对照：`ContextPreparePolicy.ObservedInputTokens`（context_manager.go:83-84）在 Prepare 已用观察值覆盖估算

### 级联风险（引用者扫描）
- `effectiveOutputBudget` 3 处调用：sampling_request.go:59/71/75（overflow recovery 路径）、compact.go:568（summary 预算）
- `lastUsage` 原子指针，sampling 后由 `storeLatestRequestUsage` 更新（run_usage.go:205）
- `resetOutputBudgetState`（rebind/resume）清空 lastUsage → 回到保守估算，无残留

## 二、多路径推演（B3）

### 方案 A：admission 信任 lastUsage 观察值（采纳）
- 改 `effectiveOutputBudget`：`lastUsage` 存在且 `LatestPromptTokens()>0` 时用观察值替代估算
- 复杂度：低（10 行）；性能：无；可维护性：与 Prepare 的 ObservedInputTokens 语义一致；风险：低估场景有 provider 层兜底
- 取舍：观察值来自上一真实请求，同一会话内是渐进近似；fresh agent 首请求后即有值

### 方案 B：修 estimatedShapeTokens 的 0.25 回退系数
- 调低回退系数（如 0.5）或按内容类型自适应
- 风险：全局影响所有无校准场景，代码/JSON 与中文混排难以一个系数覆盖；改动面大
- 否决：无法精确，且改变所有 cold-start 估算语义

### 方案 C：仅调高 context window 阈值
- 掩盖问题，不解决误报；用户窗口真实受限
- 否决

## 三、任务分解（B1）

- [x] T1 拓扑扫描：确认 3 处调用方 + lastUsage 生命周期 → 验证: grep + read_file
- [x] T2 实现修复：effectiveOutputBudget 信任观察值 → 验证: go build
- [x] T3 单元测试：无校准 + 观察值 858K → 不误报 overflow → 验证: go test -run TestEffectiveOutputBudgetUsesObservedTokensWhenCalibrationAbsent
- [x] T4 回归验证：gofmt/vet/repolint/全量 agent+control 测试 → 验证: 命令全绿
- [x] T5 事务文档 + 提交 → 验证: git log

## 四、缓存/纪律检查点（D 组 + E 组）

- D1 前缀字节：本改动是 admission 侧（本地估算），**零发送侧改动** → 安全
- D2-D7：不涉及投递/steer/重放/schema
- E1：gofmt + go vet 已跑
- E2：repolint clean（2121 baselined findings，无新增）
- E3：注释 ≤3 行（floating）
- 对抗自检（B8）：低估风险 → 若观察值 < 真实当前请求（增量大），不触发本地 overflow，由 provider 400 兜底，与 Prepare 一致，接受
