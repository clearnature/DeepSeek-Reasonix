# Reasonix Architecture

> 本文档描述 Reasonix 的系统架构：以 [`SPEC.md`](./SPEC.md) 为契约骨架，以 `internal/` 实际代码为校验基准。
> 契约（SPEC）与实现的偏差已在 §0 质疑表中逐条裁决；本文档正文以**实际代码**为准。
> 核心接口签名直接取自源码（`internal/provider/provider.go`、`internal/tool/tool.go`、
> `internal/permission/permission.go`、`internal/agent/coordinator.go`），验证日期见 git history。

## 0. 规格质疑（PHASE 0）与裁决

| # | 严重级 | 发现 | 裁决 |
|---|--------|------|------|
| S1 | P1 | SPEC §2 目录结构严重过时：只列 11 个包，实际 `internal/` 有 80+ 包；`control`、`boot`、`capability`、`memory`、`skill`、`guardian`、`serve`、`bot` 等核心包未出现；`provider` 已有 `anthropic`/`responses` 子包 | 架构文档以实际代码为准（§2），并给出 SPEC→实际的映射；建议 SPEC §2 在下次契约修订时重写 |
| S2 | P1 | SPEC §1.3 "TOML parsing is the one accepted dependency" 与实际 `go.mod` 不符（Bubble Tea、tree-sitter、goldmark、mvdan.cc/sh、pkg/sftp 等 30+ 直接依赖） | 原则已被实现实际放宽；本文档 §6 记录真实依赖分组与各自理由，建议 SPEC 修订 §1.3 措辞 |
| S3 | P2 | SPEC §3.2 的 `Tool` 接口只列 4 个方法，实际有第 5 个 `ReadOnly()` 及多个 optional 接口（`Previewer`、`ImageTool`、`PlanModeClassifier`、`MCPMetadata` 等） | 采用实际签名；SPEC §3.3 已引用 `Tool.ReadOnly()`，§3.2 列表属于简写而非矛盾 |
| S4 | P2 | SPEC §9 路线图把 "Anthropic-native provider kind" 列为未实现，实际 `internal/provider/anthropic` 已实现并注册 | 标注为已完成；架构文档覆盖该 kind（§3.1、§6） |
| S5 | P3 | SPEC §4 数据类型为最小子集，实际另有 `StreamInterruptedError` 分类、`schema_dialect`/`schema_validate`/`schema_canonicalize` 扩展 | 超集扩展，不冲突；本文档 §3.1 补充说明 |
| S6 | P3 | SPEC §3.7 说 agent 咨询一个 `Gate` 接口；实际 `permission.Gate` 是 struct，以结构化类型满足 agent 的 Gate 接口 | 无冲突；本文档记录 `Gate.Check` 精确签名 |

未发现 FR 之间不可调和的冲突；无需 BLOCKED。

## 1. 总体架构与分层

Reasonix 是"薄 harness + 配置/插件驱动能力"的 coding agent。实际代码演化出三层结构，
分层由 `tools/repolint/layers.go` 在 CI 中强制（REASONIX.md 契约）：

```text
        host（入口）：cmd/…  desktop/…
            │
   frontends（仅这 6 个包可导入 control）：
   internal/cli · internal/serve · internal/acp · internal/bot · internal/botruntime · internal/boot
            │
   ┌────────▼─────────┐
   │ internal/control │  transport-agnostic Controller：唯一会话/回合/权限编排入口
   └────────┬─────────┘
            ▼
   内核：agent · tool · provider · plugin · permission · config · command · memory …
            ▲
   leaves（工具层，不得导入内核）：diff · fileutil · retrieval · shellparse · store · textutil …
```

- **host**（`cmd/`、`desktop/`）是唯一允许导入 frontends 与 control 的入口层。
- **frontends** 只做 UI/传输适配；所有行为（turn 编排、goal、plan、权限、compaction、memory）
  都放在 `control.Controller` 里，三个前端（chat TUI、HTTP/SSE serve、Wails desktop）共享同一实现。
- **内核**（kernel）包不反向导入 frontends 或 control；`provider/openai`、`provider/anthropic`、
  `tool/builtin` 等子包通过 `init()` 导入父包完成自注册，父包不反向导入子包。
- **leaves**（utility 层）不得导入任何内核包，保证可被任何层安全复用。
- Remote-SSH 保持 `cli → remote/bootstrap → remote → {forward, sftpfs, …}` 的独立分层，
  `remote` 及其子包不依赖 `cli`、`agent`、`serve`，host-key/secret 等交互通过 callback 暴露。

依赖方向整体无环：`host → frontends → control → kernel → leaves`。

## 2. 模块结构

以实际代码为准，按职责分组（SPEC §2 的原始 11 包均在其中，标注 `(SPEC §2)`）：

```text
cmd/                                    # host：可执行入口
├── reasonix/                           # 主 CLI；blank-import 全部 built-in providers/tools
├── reasonix-plugin-example/            # 参考 MCP stdio server（echo / wordcount），e2e 测试驱动
├── reasonix-launcher/  reasonix-legacy-migrator/  signpath-contract/  e2ebench/  dashscope-bench/
internal/
├── control/            # (演化) transport-agnostic Controller：turn 编排、goal/plan 状态机、
│                       #   权限接入、compaction 调度、memory 召回、checkpoint、extensions、approval
├── cli/                # (SPEC §2) 子命令路由、flags、TUI(chat_tui.go)、组装、exit code
├── serve/              # (演化) HTTP/SSE browser frontend（[serve] 配置）
├── acp/  bot/  botruntime/  boot/   # (演化) 其余 frontends：ACP 协议、headless bot、启动装配
├── agent/              # (SPEC §2) Session + Agent.Run 主循环；coordinator.go 双模型协作；
│                       #   capability_gate.go、todo/进度租约、cancel、cache_shape
├── provider/           # (SPEC §2) Provider 接口 + 数据类型 + kind→factory registry
│   ├── openai/         # OpenAI-compatible /chat/completions（含 streaming tool-call 聚合）
│   ├── anthropic/      # (演化) Anthropic wire format（SPEC §9 已完成项）
│   ├── responses/      # (演化) OpenAI Responses API 兼容 kind
│   └── resolver.go · retry.go(退避) · schema_*.go(dialect/canonicalize/validate) · stream_error.go
├── tool/               # (SPEC §2) Tool 接口 + Registry；contract.go 文档契约校验
│   └── builtin/        # read_file/write_file/edit_file/multi_edit/move_file/bash/ls/glob/grep/… 自注册
├── permission/         # (SPEC §2) Policy(纯函数规则) + Gate(执行时裁决)；bash_approval.go 动态
│                       #   Bash 分级、bash_decompose.go 复合命令分解、bash_readonly.go 只读判定
├── plugin/             # (SPEC §2) MCP client：stdio/http/sse transport、initialize/tools/list/call、
│                       #   lazy 冷启动、launcher_lock、hotadd、install
├── pluginpkg/          # (演化) 插件包（bundled MCP server）管理
├── command/            # (SPEC §2) slash command：built-in action / .md 自定义 / MCP prompt
├── config/             # (SPEC §2) TOML 加载：flag > reasonix.toml > ~/.reasonix/config.toml > 默认
├── sandbox/            # (演化) OS 级执行强制：macOS Seatbelt / Linux bubblewrap / Windows off
├── memory/             # (演化) 自动记忆：recall/auto_recall（BM25 召回）、remember/forget 存储
├── retrieval/          # (演化) BM25 检索内核（history/memory/productdocs 共用）
├── history/            # (演化) history tool：session/archive JSONL 的 BM25 检索与 around 读取
├── skill/              # (演化) Skill 存储：内置/项目/全局、runAs=subagent profile 校验
├── capability/         # (演化) 能力声明与调度（read-only 分类、MCP capability 代理）
├── guardian/           # (演化) 安全复核（Guardian review）
├── checkpoint/         # (演化) 会话 checkpoint / 恢复
├── remote/             # (SPEC §2) SSH 传输：forward(端口转发) · sftpfs(SFTP 文件层) · bootstrap(远端 serve)
├── billing/  stats/  telemetry/  crashreport/      # (演化) 用量/统计/遥测/崩溃上报
├── extension/  event/  eventwire/  hook/           # (演化) 扩展协议与事件/钩子
├── secrets/  store/  filelock/  workspacelease/     # (演化) 凭据、状态存储、锁与租约
├── shellparse/  shellsafe/  shellrun/              # (演化) shell 命令解析/安全化/执行
├── lsp/  diff/  fileref/  fileutil/  gitcmd/       # (演化) 代码理解与文件/git 操作
├── i18n/  outputstyle/  textutil/  nilutil/        # (演化) 本地化、输出渲染与通用工具
└── 其余：ablation/ autoresearch/ boundedllm/ capdiag/ desktoplauncher/ doctor/ environment/
         evidence/ frontmatter/ goaleval/ installlayout/ installsource/ instruction/ jobs/
         mcpdiag/ mcplaunch/ mcpregistry/ migration/ netclient/ notify/ planmode/
         proc/ productdocs/ recovery/ releaseasset/ repair/ sessiontemp/ subagent?/ sysproxy/
         taskintent/ taskmonitor/ testenv/ worktree/ 等——每包单一职责，长解释在 doc.go
```

## 3. 接口设计

### 3.1 Provider（`internal/provider`）

```go
type Provider interface {
    Name() string                                            // 实例名，如 "deepseek"
    Stream(ctx context.Context, req Request) (<-chan Chunk, error) // 流式补全；ctx 取消即中止
}

type Factory func(cfg Config) (Provider, error)
func Register(kind string, f Factory)   // init() 自注册
func New(kind string, cfg Config) (Provider, error)
```

- kind 现有 `openai`（OpenAI-compatible chat）、`anthropic`、`responses`；vendor 差异
  （`base_url`/`model`/`api_key_env`）都是配置实例，不加代码。
- 可选能力接口（type-assert 发现，缺省视为 false）：`ToolCallReasoningPolicy`
  （assistant tool_calls 回合会重放 reasoning，agent 需归档原文）、`ReasoningRoundTripPolicy`
  （所有 assistant 消息保留并重放 reasoning）、`MissingToolCallReasoningWarningPolicy` 及其
  `…IdentityPolicy`（身份哈希后限频恢复）。
- 数据契约（§4 超集）：`Role`/`Message`/`ToolCall`/`ToolSchema`/`Request` 与 SPEC §4 一致；
  `Chunk` 区分 text/tool-call/done/error；流错误细分为 `StreamInterruptedError`
  （premature EOF / connection reset），供上层决定是否可重试。
- schema 进 registry 前 canonicalize；`schema_dialect` 处理 MiMo/DashScope/DeepSeek 端点方言差异。

### 3.2 Tool（`internal/tool`）

```go
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage                 // JSON Schema（params）
    Execute(ctx context.Context, args json.RawMessage) (string, error)  // 自行解析 args
    ReadOnly() bool                          // 无可观察副作用 → 并行批处理/只读默认
}

type Registry struct { … }
func NewRegistry() *Registry
func (r *Registry) Add(t Tool)               // 插入时 canonicalize schema
func (r *Registry) Get(name string) (Tool, bool)
func (r *Registry) ResolveCall(name string) (resolved Tool, canonical string, candidates []string)
func (r *Registry) RemovePrefix(p string) int   // 插件热卸载：mcp__<server>__ 前缀
func (r *Registry) SuspendPrefix(p string) int  // 插件停用/恢复
func (r *Registry) MCPBindings() []MCPBinding
func (r *Registry) ContractEntries() []ContractEntry   // TOOL_CONTRACT 文档契约校验
```

- built-in 通过 `init()` 注册到进程级集合；每次运行按配置启用子集 + 插件工具组装新 `*Registry`。
- 可选能力接口：`Previewer.Preview(args) (diff.Change, error)`（写前预览，不动磁盘）、
  `ImageTool.ExecuteWithImages`（MCP 截图等结构图像通道）、`PlanModeClassifier.PlanModeSafe()`
  （如 `complete_step` 只读但只属批准后执行阶段）、`ReadOnlyExecutionHostMutation`（如按需拉起
  MCP 进程，严格只读拒绝）、`MCPMetadata`/`MCPVisibleMetadata`/`MCPPackageMetadata`（MCP 身份溯源）。
- 错误语义：`Execute` 错误返回给模型供自纠，不终止 loop。

### 3.3 权限（`internal/permission`）

```go
type Decision int
const (Allow Decision = iota; Ask; Deny)

type Rule struct { … }                       // "Tool" | "Tool(specifier)" | "Bash=<literal>"
func ParseRule(s string) Rule

type Policy struct { Mode Decision; Allow, Ask, Deny []Rule }
func (p Policy) Decide(toolName string, readOnly bool, args json.RawMessage) Decision  // 纯函数，无 I/O
func (p Policy) MatchedRule(toolName string, d Decision, args json.RawMessage) (Rule, bool)

type Approver interface { … }                // 交互式批准（allow once/session/always/deny）

type Gate struct { Policy Policy; Approver Approver; OnRemember func(rule string) }
func (g *Gate) Check(ctx context.Context, toolName string, args json.RawMessage, readOnly bool) (bool, string, error)
```

- 优先级：`deny > ask > allow > fallback`；fallback = 只读工具 `Allow`，写工具 `Mode`。
- `Check` 对 `bash` 先做只读判定（`BashCommandIsReadOnly`），把可证明只读的命令升级为只读路径。
- 动态 Bash 分级：`BashSubjectRequiresExplicitApproval` 将命令分为可复用/仅精确字面量/必须人工
  三类（`DecomposeBashCommand` 处理复合命令，`shellparse` 支撑结构解析）；嵌套/间接执行
  （eval、`-c`、命令替换、动态命令名）默认要求人工，`allow_dynamic_bash = true` 可放宽 Allow fallback。
- 无 approver（headless/subagent）时 `Ask` fail-closed；`auto` 只放行普通 writer fallback；
  `yolo` 可越过普通 Ask，不能越过 deny/Sandbox/强制新鲜人工审批。

### 3.4 插件 / MCP（`internal/plugin`）

```go
type transport interface { call / notify / close }   // stdio | http(streamable-http) | sse
```

- 协议统一 JSON-RPC 2.0；生命周期 `initialize → notifications/initialized → tools/list → tools/call`。
- `${VAR}`/`${VAR:-default}` 在 command/args/env/url/headers 中展开（secret 留在环境）。
- 远程工具适配 `Tool`，命名 `mcp__<server>__<tool>`；`readOnlyHint` → `Tool.ReadOnly()`（默认 false）。
- 冷启动与调用分离：调用方只等短暂冷启动，`initialize`/`tools/list` 后台继续至
  `mcp_startup_timeout_seconds`（默认 30s）；调用超时从连接就绪后起算。
- 安装/项目声明即授权其全部工具；`reasonix.toml` > `.mcp.json` > 用户全局的覆盖顺序。

### 3.5 Agent 与双模型（`internal/agent`）

```go
type Runner interface { Run(ctx context.Context, input string) error }   // Agent 与 Coordinator 都实现

type Session struct { … }        // []Message
func (a *Agent) Run(ctx context.Context, input string) error             // agent.go:1307
type Coordinator struct { … }    // 双模型协作，coordinator.go:19/347
```

- CLI/控制层只依赖 `Runner`，单模型与双模型无差别使用。
- `Coordinator`：planner 与 executor 各持独立 Session，prefix 只追加不互混（cache 稳定）；
  确定性路由（不调 classifier）选择 executor-only / Light / Full / plan-for-approval / plan-only。

### 3.6 配置（`internal/config`）

- 优先级 `flag > ./reasonix.toml > ~/.reasonix/config.toml（或 %AppData%\reasonix）> 内置默认`。
- provider key 存 Reasonix home `.env`（CLI/desktop 共享）；项目 `.env` 只做 workspace 范围
  非 provider 的 `${VAR}` 展开。
- `Config.ResolveModel`：provider 名 / 裸模型名 / `provider/model`；`context_window` 与
  `max_output_tokens` 支持 `model_overrides.<model>` 覆盖。

## 4. 关键算法

| 算法 | 位置 | 选择理由 | 复杂度 |
|------|------|----------|--------|
| 分级上下文压缩（snip→prune→summary→force） | `control/controller.go`、`agent/agent.go` | cache-first：低频改写，只有越过阈值才动 prefix；snip/prune 不删消息保证 tool_calls 配对 | 每回合 O(待归档消息数) 扫描；摘要为罕见路径（预算内 LLM 调用） |
| BM25 检索 | `retrieval/bm25.go`（history/memory/productdocs 共用） | 无依赖、确定性、够用；带相对分数下限裁剪常见词噪音，0 结果指导重试 | 单查询 O(词项×文档数) 打分；召回预算受限（结果/字符上限） |
| 权限决策 `Policy.Decide` | `permission/permission.go` | 纯函数无 I/O，deny>ask>allow>fallback 一次扫描 | O(规则数)；规则解析在加载期完成 |
| 动态 Bash 分类/分解 | `permission/bash_approval.go`、`bash_decompose.go`、`shellparse` | 静态结构判定（展开/赋值/heredoc/重定向/glob vs eval/替换/动态名），无执行即分级 | 单命令 O(长度)；复合命令按分隔符分解为段后逐段裁决 |
| streaming tool-call 聚合 | `provider/openai|anthropic|responses` 内部 | 按 index 累积 delta，只发完整 ToolCall | 每 delta O(1)，按 index 索引 |
| 网络退避 | `provider/retry.go` | 429/5xx 有界指数退避，`maxBackoff = 15s`，尊重 Retry-After | 指数级尝试次数 |
| 模型解析 `Config.ResolveModel` | `provider/resolver.go` | 三态（provider 名/裸名/provider/model）确定性解析，无分类器 | O(1) 查表 |
| 进度租约（todo 活跃期） | `agent/`（todo） | 新完成/唯一读/命令/变更续租，精确重复不续；8 轮无进展追加 nudge、16 轮暂停 | O(1) 每工具轮 |

弃用方案：compaction 的强制折叠（`compact_force_ratio`）在折叠经济学不划算时允许不执行；
`context_window = 0` 关闭整个压缩路径（回到无状态朴素实现）。

## 5. 数据流

### 5.1 单模型 agent loop（SPEC §3.4）

```text
用户输入 → control.Controller
  → 组装 Request（历史 + 启用工具的 canonical schema）
  → provider.Stream（text delta 实时输出；tool-call delta 按 index 聚合）
  → 无 tool call ⇒ 回合结束
  → 有 tool call ⇒ 逐调用过 permission.Gate.Check（Deny 回"blocked"结果）
      → sandbox 强制层 → tool.Execute（built-in 或 mcp__ adapter）
      → 结果 + 进度事件回填 Session → 继续，直到完成或安全边界
```

### 5.2 双模型 Coordinator（SPEC §3.5）

```text
宿主：原始用户文本 + 可信回合元数据 → 确定性路由（无 classifier）
  → executor-only | Light | Full | plan-for-approval | plan-only
Planner（独立 session，只读调研工具集）→ 结构化计划文本
  → plan-only：持久化并结束回合（headless 可续）
  → plan-for-approval：宿主强制审批边界，批准后交接
  → 普通 plan-first：直接交接
Executor（独立 session，完整工具）→ 验证候选触点 → 执行
两 session 永不互混；prefix 只追加 ⇒ prefix cache 保持
```

### 5.3 前端统一入口

```text
chat TUI (cli/chat_tui.go) ─┐
HTTP/SSE (serve) ───────────┤→ control.Controller（唯一编排者）
Wails desktop ──────────────┘      ↓
                            内核：agent → tool → provider / plugin → permission → sandbox
```

### 5.4 MCP 生命周期

```text
配置声明（reasonix.toml / .mcp.json / 用户全局）
  → 启动：${VAR} 展开 → 拉起 transport（stdio 子进程 / http POST / sse GET）
  → initialize（roots 能力声明）→ notifications/initialized → tools/list（readOnlyHint/destructiveHint）
  → 工具注入 registry（mcp__<server>__<tool>）
  → 每次调用：_meta.progressToken → notifications/progress 流进既有进度事件链路
  → 后台启动继续至 startup timeout；调用超时自连接就绪起算
```

### 5.5 记忆与归档

```text
每真实用户回合前：原始消息 → 有预算 BM25 自动召回（memory/retrieval）
  → 命中作为低权限 user-turn 后缀追加（不改稳定 system prompt / tool schema）
compaction 移除的原文 → reasonix/archive/<timestamp>.jsonl（每行一条消息）
history tool：scope=project|global 的 BM25 检索 + operation=around 读取窗口
remember/forget：事实带不变 ID + 单调 revision + type/scope；forget 归档并从活动索引移除
```

## 6. 技术栈

- 语言/运行时：**Go**（`go.mod` 声明 `go 1.25.0`，toolchain `go1.26.5`），`CGO_ENABLED=0` 静态单二进制。
- 构建/分发：Makefile；`-ldflags "-s -w -X main.version=$(VERSION)"`（`git describe --tags --always`）；
  目标矩阵 `darwin|linux|windows × amd64|arm64`；预编译二进制 / `go install` / Homebrew。
- 配置：`BurntSushi/toml v1.6.0`（SPEC §1 唯一"接受"的基础依赖，实际已放宽，见 §0 S2）。
- CLI/TUI：`charm.land/bubbles/v2`、`bubbletea/v2`、`lipgloss/v2`、`charmbracelet/colorprofile`、
  `x/ansi`、`mattn/go-runewidth`、`rivo/uniseg`、`x/term`、`muesli/cancelreader`。
- Shell 解析/执行：`mvdan.cc/sh/v3`（结构解析）、tree-sitter（go/js/python/rust/typescript，代码理解）。
- 文件/远端：`pkg/sftp`、`kevinburke/ssh_config`、`golang.org/x/crypto`、`bmatcuk/doublestar/v4`（glob）、
  `sabhiram/go-gitignore`、`aymanbagabas/go-udiff`（预览 diff）。
- MCP/HTTP：`golang.org/x/net`、`gorilla/websocket`（间接）。
- 数据/渲染：`yuin/goldmark`（markdown）、`alecthomas/chroma/v2`（高亮）、
  `santhosh-tekuri/jsonschema/v6`（schema 校验）、`joho/godotenv`、`zalando/go-keyring`。
- 系统集成：`godbus/dbus/v5`、`git.sr.ht/~jackmordaunt/go-toast/v2`、`larksuite/oapi-sdk-go/v3`、
  `atotto/clipboard`、`golang.org/x/sys`/`x/image`/`x/text`/`x/mod`。
- 测试：`go.uber.org/goleak`（goroutine 泄漏检测）。
- 质量门：`gofmt`、`go vet`、`go run ./tools/repolint`（注释/行数/分层）、`golangci-lint`（CI）。

## 7. 决策记录（ADR）

1. **transport-agnostic `control.Controller` 统一前端**（AGENTS.md/REASONIX.md，repolint 强制）。
   为什么：TUI/serve/desktop 三前端共享全部行为，避免行为漂移；代价是 frontend 集合被显式枚举。
2. **cache-first 与双模型并存**（SPEC §3.5-3.6）。为什么：双模型若共享一个会话会破坏 prefix cache，
   故 planner/executor 各持独立 session、prefix 只追加；compaction 是唯一的"cache-reset point"。
3. **接口优先 + registry 自注册**（SPEC §1.5）。为什么：核心不硬编码 `switch model`；
   built-in 用 `init()` 自注册，父包不反向导入子包，保持依赖无环。
4. **权限为纯函数策略 + 可插拔 Approver**（SPEC §3.7）。为什么：策略可单测、无 I/O；
   交互性只出现在 Approver；headless 无 approver 时 fail-closed。
5. **MCP 信任模型 = 安装即授权**（SPEC §3.3/§3.7）。为什么：不再叠加第二套 server/raw-tool 审批；
   `readOnlyHint`/`destructiveHint` 只用于调度与只读边界，真正边界是显式 deny + 进程 sandbox。
6. **分级压缩（snip→prune→summary）而非频繁摘要**。为什么：低频改写保持 prefix cache 命中率；
   摘要只折叠 assistant/tool 工作，用户回合与既有 digest 原样保留，被移除原文归档可追溯。
7. **lean dependencies 原则被实现放宽但保持克制**（SPEC §1.3 vs go.mod）。为什么：TUI/MCP/SSH/语法
   分析需要成熟库；所有依赖仍为纯 Go（除 cgo 禁用下可用的），未引入重量级框架。
8. **单一静态二进制 + 版本 ldflags 注入**（SPEC §8）。为什么：跨平台一条命令构建，分发体验统一。

## 8. 风险

- **R1（中）**：SPEC §1.3/§2 与实际代码的偏差（见 §0 S1/S2）若长期不修订，新维护者会误判依赖策略
  与目录归属；repolint 是实际分层强制者，SPEC 应尽快同步。
- **R2（中）**：`Tool`/`Provider` 的 optional 接口持续膨胀（§3.1/§3.2），新增能力以 type-assert 发现，
  缺失时静默降级——需要契约测试锁定（`contract_test.go` 已覆盖 schema 层面）。
- **R3（低）**：SPEC §9 路线图部分项已完成（Anthropic kind）而未从 roadmap 移除，规划时可能重复投入。
- **R4（低）**：Windows 无 OS 级 Bash sandbox（SPEC §5），file-tool 路径限制是唯一强制层，属已知平台差异。
