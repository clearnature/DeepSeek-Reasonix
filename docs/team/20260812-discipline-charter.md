# 纪律团审查规范：前置开发纪律规范 + 后置验收标准

> 生成：2026-08-12 · 分支：dev/clearnature（唯一本地开发分支）· 状态：生效
> 依据：`docs/team-org.md`（团队章程 §二/§四/§六）、`fable5-thinking` 九原则、
> `REASONIX.md`（前缀缓存第一原则 + 注释/repolint 规范）、`AGENTS.md`（DoD 与 CI 门）、
> `scripts/check-cache-impact.sh`（缓存影响检查）、既有审查先例
> （`docs/team/20260810-p6-complete/review-*.md` 驳回链）。
> 定位：本文件是纪律检查团（6 人）每次交付前审查的**统一检查表**——前置规范约束
> 开发过程，后置标准约束验收判定。两者任一违反均有明确处置，不依赖审查员个人裁量。

---

## 第一部分：前置开发纪律规范（开发前/开发中强制）

> 违规即审查 FAIL 项；本部分供规划小组与执行小队在开发开始前对照自检。

### A. 职能与流程

- **A1 三职能硬分离**：本次事务的规划/执行/纪律由不同智能体承担，任一智能体不得身兼
  两职（章程 §二）。纪律团成员不得参与实现，执行队不得自审自判。
- **A2 事务文档完备**：事务必须有 `plan.md`（拓扑扫描 + 多路径推演 + 任务分解，
  fable5 Step 1-3）、`execution.md`（执行 + 沙盒验证 + 证据链，fable5 Step 4-6）、
  `review.md`（纪律团结论，fable5 Step 7-9）（章程 §七）。缺文档 = 流程不完整。
- **A3 偏离需显式裁决**：放弃计划中的硬约束、验收矩阵项、测试项必须留下裁决记录
  （仲裁/驳回记录），不得擅自降级（先例：P6 D2 硬约束未落实 → 驳回）。
- **A4 事务记录位置**：所有团队文档写入 `docs/team/{date}-{topic}/`（章程 §七）；
  绝不写入 main-v2 只读镜像。

### B. fable5 九原则（全体强制，不可跳过）

| # | 原则 | 执行队必须留下可验证痕迹 |
|---|------|--------------------------|
| B1 | 任务分解 | >3 步任务先分解为子任务 + 每步验证点 |
| B2 | 拓扑扫描 | 改文件前 grep 引用者/被引用者，识别级联风险 |
| B3 | 多路径推演 | >50 行改动给出 ≥2 条路径并标注取舍 |
| B4 | 自我验证 | 写完以审稿人视角审视（边界/简化/命名） |
| B5 | 沙盒验证 | 写完必须测试；失败捕获完整日志 → 定位根因 → 修复 → 重测 |
| B6 | 分级路由 | 按复杂度选择合适 agent |
| B7 | 持久记忆 | 关键决策写入 memory |
| B8 | 对抗自检 | 以 devil's advocate 攻击自己的方案 |
| B9 | 防虚假完成 | 每个完成声明附带可验证证据（命令输出/diff/数字） |

### C. 分支纪律（项目特定——详见 docs/merge/merge-discipline.md §11，不重复表述）

- **C1 main-v2 = 只读镜像**：只允许 `git fetch origin` + `git merge --ff-only origin/main-v2`
  更新；**绝不在 main-v2 上提交任何开发内容**（文档/代码都不行）。
- **C2 唯一开发分支**：所有本地开发/文档/实验 → `dev/clearnature`（唯一本地成果载体）。
- **C3 禁止 checkout 覆盖**：禁止 `git checkout` 其他分支覆盖本地文件——需用
  `diff` + `edit_file` 精确补齐。
- **C4 PR 纪律**：PR 面向 main-v2；一个 review 回合只允许一次 force-push；
  PR diff 最小化（只含本事务相关文件）；review 反馈用 amend 而非追加 commit。

### D. 缓存红线（Reasonix 领域最硬红线）

- **D1 前缀字节第一**：任何发送侧改动（序列化/字段/顺序/重放/compaction）先问
  「会不会改变前缀字节？」——会 = 破坏服务端前缀缓存 = 用户付全价，否决或必须有
  缓存收益补偿（REASONIX.md 思想钢印）。
- **D2 只 append 尾部**：自动投递/steer/mailbox 投递只 append turn 尾部
  （消息位置固定），零插入历史、零重写 canonical、零重排既有消息。
- **D3 位置固定保留窗口**：小 turns 保留窗口必须按「消息位置固定」`[head, head+N]`
  实现，禁止「最新 N 条」动态保留——否则每次 compaction 前缀漂移打穿缓存。
- **D4 重放门控（C1）**：打开历史会话/recovery 恢复先压缩再发送，把「赌前缀恰好
  命中」变成「确定的小前缀 + 后续稳定命中」。
- **D5 运行值不进 schema**：profile 名/任务文本/动态值绝不进 schema/system prompt；
  fork 等参数必须是静态 bool。发送侧 byte-identical 是硬前提。
- **D6 缓存影响申报**：改动触及 `scripts/check-cache-impact.sh` 列出的敏感路径时，
  PR body 必须含 `Cache-impact:`、`Cache-guard:`（触及 system-prompt 敏感路径另加
  `System-prompt-review:`），值不得为 todo/tbd/none/n/a。
- **D7 打穿后稳定**：一次性打穿（如工具面新增）属 deliberate，但打穿后必须字节稳定
  （golden 同步更新），不得每轮漂移。

### E. 代码与回归纪律

- **E1 CI 双门**：`gofmt` + `go vet` 由 CI 强制，push 前必须跑。
- **E2 repolint ratchet**：`go run ./tools/repolint` 对 ratchet baseline 强制；
  不得 widening baseline 来落地变更（`-update` 仅限改名/抽取且须在 PR 中 justify）。
- **E3 注释规范**：doc 注释 ≤5 行；不写代码复述/阶段叙事/横幅；`FIXME` 禁止；
  `TODO(#nnn):` 需 issue 锚点。
- **E4 分层纪律**：只前端/host 可 import `control`（`tools/repolint/layers.go`）；
  新增包须同步 `ARCHITECTURE.md`（check-arch-sync 守卫）。
- **E5 effect test**：性能/缓存特性必须带最终边界的 effect test
  （`internal/boot/effect_test.go` 模式）——组件正确 ≠ 系统生效。
- **E6 锁纪律**：禁止 `m.mu → j.mu` 嵌套；持锁路径内禁止调用任何 `m.*` 查询方法
  （先例：P1 自死锁、P6 依赖门自死锁均由此类问题引发）。

---

## 第二部分：后置验收标准（纪律团审查清单）

> 每条验收项必须附**可验证证据**；审查员不得无证据通过。交叉验证优先用
> 重跑命令/`git diff`/`read_file` 核对行号——不替代执行队，只独立核验。

### 验收项 1：fable5 合规（执行队是否跳步）

- 证据要求：`plan.md` 中有任务分解 + 拓扑扫描 + 多路径推演；`execution.md` 中有
  沙盒验证记录（真实命令 + 输出）；B8 对抗自检痕迹。
- **FAIL 条件**：发现跳步（未拓扑扫描直接改、未沙盒验证即声称完成）、放弃矩阵项
  但无裁决记录。

### 验收项 2：幻觉检测（证据真实性）

- 证据要求：测试输出**可重跑**（命令 + 真实输出）；文件变更 `git diff` 可核对；
  行号 `read_file`/`grep` 可核对；数字可复核；测试名与场景**名实相符**。
- **FAIL 条件**：无证据/编造证据的「完成了」；测试名不副实（先例：
  `TestJobDoneObserverNotFiredAfterClose` 场景与设计意图不符）；引用的行号/文件
  在工作区不存在。

### 验收项 3：缓存红线

- 证据要求：diff 证明发送侧前缀零变化；投递/steer 只 append 尾部；运行值不进
  schema；`Cache-impact`/`Cache-guard`（及需要时的 `System-prompt-review`）三行齐备。
- **FAIL 条件**：任何前缀字节变化、动态值进 schema、黄金文件未同步、三行缺失或
  值为 todo/tbd/none/n/a。

### 验收项 4：回归

- 证据要求（按改动范围选择）：`make test`（go test ./...）、`make vet`、
  `make lint`（golangci-lint + repolint + wails pin）、`gofmt -l .` 清洁；
  性能/缓存特性跑对应 effect test；依赖方（拓扑扫描识别）无破坏。
- **合并事务**：必须执行 `docs/merge/merge-discipline.md` §8.1 统一验证链
  （gofmt→build→核心包测试→protocol-check→repolint→desktop tsc+wails）。
- **FAIL 条件**：任一命令失败；依赖方回归未被处理。

### 验收项 5：对抗自检（devil's advocate）

- 证据要求：审查报告列出攻击点（锁序/竞态/时序窗口/幂等/幽灵事件/文档漂移等）
  及处置（修复或显式裁决为已接受风险）。
- **FAIL 条件**：存在 blocking 缺陷未被修复或裁决（先例：P4 sentinel 无捕获者、
  P4 后台任务被父 turn 生命周期绑架均为 blocking → 驳回）。

### 验收项 6：防虚假完成（证据链完整性）

- 证据要求：每个「完成/通过/已修复」声明对应一条可核验证据（命令输出、文件 diff、
  数字）；无证据声明被立即标记 FAIL。
- **FAIL 条件**：声称完成但证据缺失或无法重跑。

### 判定规则

> 以下为本次确定的治理规则：既有章程/规范中已明文的引用 A-E 组与验收项 1-6；
> 判定规则中的「一票否决」「驳回周期上限」为本次新增决策（用户授权确定规范），
> 与既有规则一并生效，后续若与章程冲突以章程修订为准。

- **通过**：验收项 1-6 全部 PASS。
- **驳回**（附驳回理由 + 必须修复项清单）：
  - 验收项 2（幻觉检测）或 3（缓存红线）FAIL → **无条件驳回**；
  - 验收项 1（fable5 跳步且未裁决）、4（回归 FAIL）、5（blocking 缺陷）→ 驳回；
  - 任一 FAIL 且执行队未留下裁决记录 → 驳回。
- **复审**：驳回后执行小队按「必须修复项」修复 → 纪律团复审，逐条核对修复项 →
  全部 PASS → 通过（先例：P6 4 驳回 + 复审全 PASS）。
- **驳回周期上限**：同一事务连续驳回 ≥2 轮仍未收敛 → 升级决策委员会裁决
  （章程 §五），不无限循环。

---

## 第三部分：审查输出格式（纪律团 6 人独立报告）

```markdown
# 纪律审查报告：<事务名>
## 审查项逐条
- [PASS/FAIL] fable5 合规（执行队是否跳步）
- [PASS/FAIL] 幻觉检测（证据真实性——重跑验证结果）
- [PASS/FAIL] 缓存红线（前缀字节稳定？）
- [PASS/FAIL] 回归（测试命令 + 结果）
- [PASS/FAIL] 对抗自检（攻击发现的缺陷）
- [PASS/FAIL] 防虚假完成（证据链完整性）
## 结论
**通过** / **驳回**（附驳回理由与必须修复项）
```

6 人独立审查（互不参考结论，防锚定）→ 汇总：任一 FAIL 即驳回（纪律从严，
与章程 §五 决策委员会 ≥90% 同意率的宽松标准不同——交付审查采「一票否决」）。
