# Merge 决策与纪律

上游同步（main-v2 → dev/clearnature）的**决策记录、风险分析、纪律清单**。
每次合并前先读本目录，按纪律执行。

## 目录

- `merge-discipline.md` — **合并纪律全文**（侦察/逐块/语义复查/生成文件/验证链/遥测四方审计/分支纪律）
- `20260814-v125-merge-risk.md` — v1.25.x（26 提交）↔ dev（508 提交）合并风险分析（按主题）
- `../telemetry-audit-20260813.md` — 遥测四方审计纪律（合并涉及遥测文件后必做）

> 完整纪律见 `merge-discipline.md`（本目录主文档）；以下为速查。

## 合并纪律速查（长期）

0. **前置风险分析（必做）**：使用 team-planner 技能，按四域（内核安全/压缩/缓存/统计准确性）识别风险，三级别预警（🟢低常规验证 / 🟡中定向核对 / 🔴高深入分析+守卫全量+四方审计），归档风险文档。
1. **合并前侦察**：`git merge-base` → `git rev-list --count` 落后量 → `git merge-tree --write-tree` dry-run 冲突面。
2. **0 文本冲突 ≠ 0 语义风险**：同一文件双方都改 = 逐字段核对（参考 telemetry-audit 纪律）；同域修复（压缩/估算/遥测）合并后必须跑守卫测试套件。
3. **逐块精确处理**：禁整文件/整包 checkout、禁 `git merge <branch>` 到 PR 分支；冲突用 edit_file 逐块解决。
4. **合并后验证链**：`gofmt -l .` → `go build ./...` → 核心包测试（provider/agent/control/config/tool/protocol）→ `remote-protocol-gen -check` → `repolint` → desktop 边界（tsc + bundle budget + `wails build -tags webkit2_41`）。
5. **遥测四方审计**（合并涉及 `internal/agent/*telemetry*.go`、`internal/stats/*` 后必做）：`CompactionTelemetry` 结构 ↔ emit detail 键 ↔ recorder 解析 ↔ `CompactionRecord` JSON tag 一次对照；失败路径必须落盘（`status=failed` + `err_type=`）。
6. **分支纪律**：main-v2 只读镜像（ff-only）；dev/clearnature 是唯一开发分支；文档/决策一律提交到 dev。
