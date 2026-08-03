# token-plan/other-tools.md

> 来源: https://platform.minimaxi.com/docs/token-plan/other-tools.md

# 其他工具

> 在任意支持自定义 OpenAI 兼容或 Anthropic 兼容端点的 AI 编程工具中接入最新的 MiniMax M 系列模型。

上面已经覆盖了主流 AI 编程工具的接入步骤。如果你使用的工具不在列表里，但支持自定义 Base URL + API Key，照下面的值填即可。

## 配置参考

MiniMax 同时提供两种兼容协议，你的工具支持哪种就选哪种——大多数现代工具至少支持其中一种。

### OpenAI 兼容协议

| 字段 | 值 |
| ------------ | ------------------------------------------------------------------------ |
| **Provider** | `OpenAI Compatible`（有的工具叫 `Custom` 或 `OpenAI-format`） |
| **Base URL** | `https://api.minimaxi.com/v1` |
| **API Key** | [获取订阅 Key](https://platform.minimaxi.com/user-center/payment/token-plan) |
| **Model ID** | `MiniMax-M3` |

### Anthropic 兼容协议

| 字段 | 值 |
| ------------ | ------------------------------------------------------------------------ |
| **Provider** | `Anthropic Compatible`（有的工具叫 `Claude` 或 `Custom Anthropic`） |
| **Base URL** | `https://api.minimaxi.com/anthropic` |
| **API Key** | [获取订阅 Key](https://platform.minimaxi.com/user-center/payment/token-plan) |
| **Model ID** | `MiniMax-M3` |

## 该选哪种协议

| 工具类型 | 推荐协议 | 常见环境变量 |
| ----------------------------------------------- | --------------------------------------- | --------------------------------------------- |
| Claude Code 风格（为 Anthropic 设计的 TUI/CLI） | Anthropic 兼容 | `ANTHROPIC_BASE_URL` + `ANTHROPIC_AUTH_TOKEN` |
| Cursor / Continue / Aider / 各类 OpenAI 格式 IDE 插件 | OpenAI 兼容 | `OPENAI_BASE_URL` + `OPENAI_API_KEY` |
| 两种都支持的工具 | 任选——推荐 Anthropic 兼容（享受 prompt cache 优势） | |

## Dify

[**Dify**](https://github.com/langgenius/dify) 是一个开源 LLM 应用开发平台，集成工作流、RAG、Agent 与可观测性。

 打开 [Dify Cloud](https://cloud.dify.ai)，登录后进入 **设置 → 工作空间 → 模型供应商**，在列表中找到 **Minimax** 并点 **安装**。

 安装完成后点 **添加 API Key**，填入 [订阅 Key](https://platform.minimaxi.com/user-center/payment/token-plan)、**API Base** `https://api.minimaxi.com/anthropic`、**Group ID** 留空。

 保存后即可在工作流中调用 MiniMax-M3 系列模型。

## Cherry Studio

[**Cherry Studio**](https://github.com/CherryHQ/cherry-studio) 是一个开源桌面客户端，支持 50+ LLM 服务商，内置 MCP 服务器和 300+ 智能助手。

 按 [Cherry Studio 官方文档](https://docs.cherry-ai.com/cherry-studio/download) 完成安装。

 打开 Cherry Studio → 点 **Choose other Providers** → 搜索框输入 `MiniMax`，选 **MiniMax CN**。

 填入 [订阅 Key](https://platform.minimaxi.com/user-center/payment/token-plan)（API Host 已预填），点 **Check** 验证连接。

 模型列表中选 `MiniMax-M3` 即可使用。

## Xcode

[**Xcode**](https://developer.apple.com/xcode/) 是 Apple 官方 IDE，支持 macOS / iOS / iPadOS / watchOS / visionOS 开发，内置 Coding Intelligence。

 从 [Mac App Store](https://apps.apple.com/us/app/xcode/id497799835) 安装 Xcode 26 或更新版本。

 打开 Xcode → 顶部菜单 **Xcode → Settings → Intelligence → Add a Model Provider**，选 **Internet Hosted** 标签页，填：

 * **URL**：`https://api.minimaxi.com`（裸 host，不带任何 path）
 * **API Key Header**：`Authorization`（手动键入，覆盖默认的 `x-api-key`）
 * **API Key**：`Bearer `（`Bearer` 后**只有一个空格**再接 `sk-cp-…` key，[去获取](https://platform.minimaxi.com/user-center/payment/token-plan)）
 * **Description**：`MiniMax`（任意）

 点 **Add**，回 Intelligence 面板进入新加的 MiniMax provider，启用 `MiniMax-M3`。

 打开任意项目，按 **⌘+0** 唤出 Coding Assistant，左上角编辑图标里选 `MiniMax-M3`。

## Pi

[**Pi**](https://github.com/earendil-works/pi) 是 earendil-works 出品的开源终端编程 Agent，通过统一 LLM API 接入任意服务商。

 ```bash theme={null}
 npm install -g @earendil-works/pi-coding-agent
 ```

 把 [订阅 Key](https://platform.minimaxi.com/user-center/payment/token-plan)（前缀 `sk-cp-…`）设到环境变量后启动 Pi：

 ```bash theme={null}
 export MINIMAX_CN_API_KEY=sk-cp-...
 pi --provider minimax-cn --model MiniMax-M3
 ```

## Kilo Code

[**Kilo Code**](https://github.com/Kilo-Org/kilocode) 是开源的 VS Code AI 编程 Agent 插件，支持多家 LLM 服务商和 MCP 服务器。

使用前先清空 `ANTHROPIC_AUTH_TOKEN` 和 `ANTHROPIC_BASE_URL` 环境变量，否则会覆盖配置。

 在 VS Code 扩展面板搜索 `Kilo Code` 安装。

 打开 Kilo Code → **Settings**：

 * **API Provider** 选 `MiniMax`
 * **MiniMax Entrypoint** 选 `api.minimaxi.com`
 * **MiniMax API Key** 填入 [订阅 Key](https://platform.minimaxi.com/user-center/payment/token-plan)
 * **Model** 选 `MiniMax-M3`

 依次点 **Save** + **Done** 保存。

## Zed

[**Zed**](https://github.com/zed-industries/zed) 是 Atom 创始团队打造的开源高性能多人协同代码编辑器，由 Rust 编写。

 按 [Zed 官方文档](https://zedhub.org/getting-started) 完成安装。

 设置 → **LLM Provider** → **+Add Provider** → 选 **OpenAI**，填：

 * **API URL**：`https://api.minimaxi.com/v1`
 * **API Key**：[Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan) 获取
 * **Model Name**：`MiniMax-M3`

 点 **Save Provider** 保存，回 LLM Provider 列表点击新加的 MiniMax 条目，**再次输入 API Key 并按回车**确认。

 回智能体面板右下角 **Select a Model** 选 `MiniMax-M3` 即可使用。

## OpenCode

[**OpenCode**](https://github.com/sst/opencode) 是 SST 出品的开源终端 AI 编程 Agent，支持多服务商接入和 LSP 集成。

OpenCode 已**内置 MiniMax-M3**，无需额外配置文件。

 ```bash theme={null}
 curl -fsSL https://opencode.ai/install | bash
 # 或 npm i -g opencode-ai
 ```

 运行 `opencode auth login`，提示选 provider 时搜并选 **MiniMax Token Plan（minimaxi.com）**，填入 [Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan) API Key。

 回到命令行 `opencode` 启动即可。

## Grok CLI

[**Grok CLI**](https://github.com/superagent-ai/grok-cli) 是开源终端编程 Agent，可接入 xAI Grok 与任意 OpenAI 兼容服务商。

不推荐做 Agent 工作流，推荐使用 **Claude Code** 或 **Cursor**。

使用前先清空 `OPENAI_API_KEY` 和 `OPENAI_BASE_URL` 环境变量。

 ```bash theme={null}
 npm install -g @vibe-kit/grok-cli
 ```

 ```bash theme={null}
 export GROK_BASE_URL=https://api.minimaxi.com/v1
 export GROK_API_KEY=sk-cp-... # 从 Token Plan 获取
 grok --model MiniMax-M3
 ```

## Droid

[**Droid**](https://factory.ai) 是 Factory 官方终端编程 Agent，可与 IDE 和团队协作工具集成。

必须先清空 `ANTHROPIC_AUTH_TOKEN` 环境变量（会覆盖 config.json 里的 key）。注意配置文件路径是 `~/.factory/config.json`（**不是** `settings.json`）。

 ```bash theme={null}
 curl -fsSL https://app.factory.ai/cli | sh # macOS / Linux
 # Windows: irm https://app.factory.ai/cli/windows | iex
 ```

 在 `~/.factory/config.json` 加入：

 ```json theme={null}
 {
 "custom_models": [{
 "model_display_name": "MiniMax-M3",
 "model": "MiniMax-M3",
 "base_url": "https://api.minimaxi.com/anthropic",
 "api_key": "",
 "provider": "anthropic",
 "max_tokens": 64000
 }]
 }
 ```

 启动 `droid`，`/model` 选 `MiniMax-M3` 即可使用。

## MonkeyCode

[**MonkeyCode**](https://github.com/chaitin/MonkeyCode) 是长亭科技 (Chaitin) 出品的企业级 AI 开发平台，Code Agent 兼容 Codex / Claude Code / OpenCode。

MonkeyCode 内置免费的 MiniMax-M3，但资源池在高峰期可能排队，建议长期使用自己配 key。

 访问 [MonkeyCode 官网](https://monkeycode-ai.com/?ic=019b4f38-64b2-7dee-959c-ec02691c290d) 登录。

 右下角 **配置** → **AI 大模型** → **绑定**。

 * **API 地址**：`https://api.minimaxi.com/anthropic`
 * **API Key**：[Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan) 获取
 * **接口格式**：`anthropic`
 * **模型名称**：`MiniMax-M3`

 **保存** 后回主界面即可使用 MiniMax-M3。

## Qwen Code

[**Qwen Code**](https://github.com/QwenLM/qwen-code) 是阿里巴巴开源的终端编程 Agent，针对 Qwen 模型族优化。

 Linux / macOS：

 ```bash theme={null}
 bash -c "$(curl -fsSL https://qwen-code-assets.oss-cn-hangzhou.aliyuncs.com/installation/install-qwen.sh)"
 ```

 Windows（Command Prompt 与 PowerShell 通用）：

 ```powershell theme={null}
 powershell -Command "Invoke-WebRequest 'https://qwen-code-assets.oss-cn-hangzhou.aliyuncs.com/installation/install-qwen.bat' -OutFile (Join-Path $env:TEMP 'install-qwen.bat'); & (Join-Path $env:TEMP 'install-qwen.bat')"
 ```

 终端运行 `qwen` 启动客户端，依次选 **Third-party Providers** → **MiniMax API Key** → 区域选 **China**。

 填入从 [Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan) 获取的 API Key（前缀 `sk-cp-…`）后回车。

 模型 ID 应为 `MiniMax-M3`，按回车提交即可开始对话。

## Open WebUI

[**Open WebUI**](https://github.com/open-webui/open-webui) 是自托管的开源 AI 聊天平台，支持离线运行、RAG 和多模型 runner。

 通过 Python pip 安装（要求 **Python 3.11**，避免兼容性问题）：

 ```bash theme={null}
 pip install open-webui
 open-webui serve
 ```

 浏览器打开 [http://localhost:8080](http://localhost:8080)，按提示创建本地管理员账号。

 右上角头像 → **Admin Panel** → 顶部 **Settings** → 左侧 **Connections**，在 **OpenAI** 一栏点 **➕ Add Connection**，填：

 * **URL**：`https://api.minimaxi.com/v1`
 * **Auth**（Bearer 模式）：从 [Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan) 获取的 API Key（前缀 `sk-cp-…`）

 点 **Save** 后回到聊天界面，顶部模型选择器选 `MiniMax-M3` 即可对话。

## nanobot

[**nanobot**](https://github.com/HKUDS/nanobot) 是 HKUDS 出品的轻量级开源个人 AI Agent CLI，内置对话频道、记忆和 MCP 支持。

 ```bash theme={null}
 uv tool install nanobot-ai
 # 或：pipx install nanobot-ai
 # 或：pip install nanobot-ai # macOS 上可能需要 --user 或 --break-system-packages
 ```

 ```bash theme={null}
 nanobot onboard
 ```

 把 `sk-cp-...` 换成你的 [Token Plan API Key](https://platform.minimaxi.com/user-center/payment/token-plan)：

 ```bash theme={null}
 python3 -c '
 import json, os, sys
 p = os.path.expanduser("~/.nanobot/config.json")
 c = json.load(open(p))
 c["providers"]["minimax"]["apiKey"] = sys.argv[1]
 c["providers"]["minimax"]["apiBase"] = "https://api.minimaxi.com/v1"
 c["agents"]["defaults"]["provider"] = "minimax"
 c["agents"]["defaults"]["model"] = "MiniMax-M3"
 json.dump(c, open(p, "w"), indent=2)
 ' sk-cp-...
 ```

 ```bash theme={null}
 nanobot agent
 ```

## OpenHands

[**OpenHands**](https://github.com/All-Hands-AI/OpenHands)（前身 OpenDevin）是 All-Hands-AI 出品的开源 AI 编程 Agent，提供 TUI、网页 GUI 和 IDE 集成。

 ```bash theme={null}
 pipx install openhands
 ```

 把 `sk-cp-...` 换成你的 [Token Plan API Key](https://platform.minimaxi.com/user-center/payment/token-plan)：

 ```bash theme={null}
 pipx run --spec openhands python -c '
 import sys
 from openhands_cli.stores.agent_store import AgentStore
 AgentStore().create_and_save_from_settings(
 llm_api_key=sys.argv[1],
 settings={"llm_model": "openai/MiniMax-M3",
 "llm_base_url": "https://api.minimaxi.com/v1"},
 )
 ' sk-cp-...
 ```

 ```bash theme={null}
 openhands
 ```

## LangChain

[**LangChain**](https://github.com/langchain-ai/langchain) 是 LangChain Inc. 出品的开源 LLM 应用开发框架，提供模型适配器、检索、Agent 与可观测性能力。

 ```bash theme={null}
 pip install langchain-openai
 ```

 把 `sk-cp-...` 换成你的 [Token Plan API Key](https://platform.minimaxi.com/user-center/payment/token-plan)：

 ```python theme={null}
 from langchain_openai import ChatOpenAI

 llm = ChatOpenAI(
 model="MiniMax-M3",
 api_key="sk-cp-...",
 base_url="https://api.minimaxi.com/v1",
 )
 print(llm.invoke("Hello").content)
 ```