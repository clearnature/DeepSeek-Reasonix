# token-plan/openclaw.md

> 来源: https://platform.minimaxi.com/docs/token-plan/openclaw.md

# OpenClaw

> 在 OpenClaw 中使用最新的 MiniMax M 系列模型。

[**OpenClaw**](https://github.com/openclaw/openclaw) 是本地运行的个人 AI 助手，可与多种通讯平台集成实现远程操控。

## 安装 OpenClaw

在终端中运行以下命令安装（老用户同样可以用此命令更新）：

 ```bash theme={null}
 curl -fsSL https://openclaw.bot/install.sh | bash
 ```

 ```powershell theme={null}
 iwr -useb https://openclaw.ai/install.ps1 | iex
 ```

***

## 配置 MiniMax 模型

安装完成后会自动进入配置引导。若没有自动开始，运行：

```bash theme={null}
openclaw configure
```

选择认证方式开始配置：

 * Where will the Gateway run? → 选择 **Local (this machine)**
 * Select sections to configure → 选择 **Model**
 * Model/auth provider → 选择 **MiniMax**
 * MiniMax auth method → 选择 **MiniMax CN — OAuth (minimaxi.com)**

 自动弹出登录页，登录并授权。

 系统默认勾选 `MiniMax-M2.7` 和 `MiniMax-M3`，并将 M3 设为默认，直接回车确认即可。

 输入 `openclaw tui`，能正常对话即配置成功。

 * Where will the Gateway run? → 选择 **Local (this machine)**
 * Select sections to configure → 选择 **Model**
 * Model/auth provider → 选择 **MiniMax**
 * MiniMax auth method：
 * 中国大陆用户 → **MiniMax CN — API Key (minimaxi.com)**
 * 海外用户 → **MiniMax Global — API Key (minimax.io)**

 填入你的 MiniMax API Key，然后回车使用默认模型选项。

 按需配置以下选项：

 * **Channel**：选择在哪个 App 中对话
 * **Skill**：按需安装技能
 * **Hooks**（可选）：
 * 💾 `session-memory`：执行 `/new` 时自动保存会话上下文
 * 📝 `command-logger`：记录所有命令到日志文件
 * 🚀 `boot-md`：网关启动时运行 BOOT.md

 输入 `openclaw tui`，能正常对话即配置成功。

***

## 开关思考

运行时用 `/think` 开关思考：`off` 关闭，`adaptive`（默认）让 M3 自行决定何时思考。

***

## 进阶能力配置方法

### 识图能力

 OAuth 登录方式会**自动接入** MiniMax [图像理解 MCP 服务](/docs/token-plan/mcp-guide)，OpenClaw 的 `image` 工具开箱即用，无需额外配置。

 若使用 **API Key 手动配置**，或 OpenClaw 不是官方最新版（图像理解工具未自动接入），需通过下列方式之一为 Agent 补上识图能力。

 [MiniMax CLI](https://github.com/MiniMax-AI/cli)（命令名 `mmx`）是官方多模态命令行工具，装好后可让 OpenClaw 通过 `mmx vision` 实现图像理解，**同时复用文本/图像/视频/语音/音乐/搜索等 7 类多模态能力**。

 确保 Node.js 18+ 已就绪：

 ```bash theme={null}
 # 检查 Node.js 版本（应 >= 18）
 node -v

 # 如未安装，推荐通过 nvm 安装 LTS 版本
 curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
 nvm install --lts
 ```

 在终端运行以下命令完成全局安装：

 ```bash theme={null}
 npm install -g mmx-cli
 ```

 ⭐️ 或让 OpenClaw 一次性完成安装、登录与 SKILL 接入，提示词（请将 `sk-xxxxx` 替换为你的实际密钥）：

 ```plaintext theme={null}
 请帮我接入 MiniMax CLI（https://github.com/MiniMax-AI/cli），按以下三步完成安装与配置：

 1. 全局安装 CLI：执行 `npm install -g mmx-cli`，完成后用 `mmx --version` 验证
 2. 登录并配置 API Key：执行 `mmx auth login --api-key sk-xxxxx`
 3. 安装官方 SKILL：执行 `npx skills add MiniMax-AI/cli -y -g`

 完成后请执行 `mmx quota` 查看我的 Token Plan 余额，确认整体配置生效。
 ```

 使用 API Key 完成鉴权（请将 `sk-xxxxx` 替换为你的 Key）：

 ```bash theme={null}
 mmx auth login --api-key sk-xxxxx
 ```

 最新版 mmx-cli 会根据 Key 自动检测服务区域，通常无需手动配置 region。

 服务区域根据所使用的 API 服务是在国内平台（`cn`，[MiniMax 国内版订阅](https://platform.minimaxi.com/subscribe/token-plan)）还是在海外平台（`global`，[MiniMax 国际版订阅](https://platform.minimax.io/subscribe/token-plan)）上购买所决定。

 若登录后调用接口报 401，大概率是 region 未自动匹配成功。可手动指定：

 ```bash theme={null}
 mmx config set --key region --value cn # 国内 API 服务
 mmx config set --key region --value global # 海外 API 服务
 ```

 执行 `mmx auth status` 确认当前 region 与所购买服务来源平台一致。

 登录完成后，可执行 `mmx quota` 查看 Token Plan 套餐额度，确认整体配置生效。

 若你要在 OpenClaw 中调用 mmx 命令，建议加装官方 SKILL.md，Agent 调用时决策更准、无需临时翻 `--help`：

 ```bash theme={null}
 npx skills add MiniMax-AI/cli -y -g
 ```

 SKILL 会自动 symlink 到 `~/.openclaw/skills/`，OpenClaw 下次启动即可识别。

 告知 OpenClaw 后续识图需求优先走 `mmx vision`：

 ```plaintext theme={null}
 记住，之后要进行图像理解时优先使用 mmx vision 工具，
 通过 `mmx vision describe --image ` 调用并解析返回结果。
 ```

 在 OpenClaw 中发送图片，检查 OpenClaw 是否会自动调用 `mmx vision describe --image ` 并返回正确的图像描述。

 通过 [Token Plan MCP](/docs/token-plan/mcp-guide) 接入 `understand_image` 工具。

 确保 uv（Python 包管理器）已就绪：

 ```bash theme={null}
 # 检查是否已安装
 which uv

 # 如未安装，通过 pip 安装
 pip install uv

 # 或通过 curl 安装
 curl -LsSf https://astral.sh/uv/install.sh | sh

 # 若上述安装失败或速度很慢，可使用国内镜像加速：
 export UV_INDEX_URL="https://pypi.tuna.tsinghua.edu.cn/simple"
 curl -LsSf https://astral.sh/uv/install.sh | sh
 ```

 手动配置请参考 [Token Plan MCP 接入指南](/docs/token-plan/mcp-guide)，核心步骤：

 1. 安装 `mcporter` skill（MCP server 管理工具）
 2. 按照官方文档配置 `minimax-coding-plan-mcp`
 3. 设置 `MINIMAX_API_KEY` 环境变量（Token Plan API Key 以 `sk-cp` 开头）：

 ```bash theme={null}
 export MINIMAX_API_KEY="sk-..."
 ```

 ⭐️ 或让 OpenClaw 一次性完成，提示词（**把 `sk-xxxxxxx` 替换为你的实际 API Key**）：

 ```plaintext theme={null}
 先安装 mcporter skill，再按照 https://platform.minimaxi.com/docs/token-plan/mcp-guide 安装和配置 MCP，我的 MiniMax API Key 是 sk-xxxxxxx，最后再通过 mcporter skill 将其接入使用。
 ```

 告知 OpenClaw 后续识图需求优先走 `understand_image`：

 ```plaintext theme={null}
 记住，之后要进行图像理解时需使用 minimax-coding-plan-mcp 的 understand_image 工具。
 ```

 在 OpenClaw 中发送图片，检查 OpenClaw 是否会自动调用 `understand_image` 工具并返回正确的图像描述。

***

### 网络搜索

OpenClaw 默认不带网络搜索能力，可通过下列方式之一为 Agent 补上（适用于 Token Plan 订阅用户，API Key 以 `sk-cp` 开头）。

 `mmx search` 是 MiniMax CLI 提供的网络检索命令，无需额外 MCP server 进程，直接通过 Bash 调用即可。

 确保 Node.js 18+ 已就绪：

 ```bash theme={null}
 # 检查 Node.js 版本（应 >= 18）
 node -v

 # 如未安装，推荐通过 nvm 安装 LTS 版本
 curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
 nvm install --lts
 ```

 在终端运行以下命令完成全局安装：

 ```bash theme={null}
 npm install -g mmx-cli
 ```

 ⭐️ 或让 OpenClaw 一次性完成安装、登录与 SKILL 接入，提示词（请将 `sk-xxxxx` 替换为你的实际密钥）：

 ```plaintext theme={null}
 请帮我接入 MiniMax CLI（https://github.com/MiniMax-AI/cli），按以下三步完成安装与配置：

 1. 全局安装 CLI：执行 `npm install -g mmx-cli`，完成后用 `mmx --version` 验证
 2. 登录并配置 API Key：执行 `mmx auth login --api-key sk-xxxxx`
 3. 安装官方 SKILL：执行 `npx skills add MiniMax-AI/cli -y -g`

 完成后请执行 `mmx quota` 查看我的 Token Plan 余额，确认整体配置生效。
 ```

 使用 API Key 完成鉴权（请将 `sk-xxxxx` 替换为你的 Key）：

 ```bash theme={null}
 mmx auth login --api-key sk-xxxxx
 ```

 最新版 mmx-cli 会根据 Key 自动检测服务区域，通常无需手动配置 region。

 服务区域根据所使用的 API 服务是在国内平台（`cn`，[MiniMax 国内版订阅](https://platform.minimaxi.com/subscribe/token-plan)）还是在海外平台（`global`，[MiniMax 国际版订阅](https://platform.minimax.io/subscribe/token-plan)）上购买所决定。

 若登录后调用接口报 401，大概率是 region 未自动匹配成功。可手动指定：

 ```bash theme={null}
 mmx config set --key region --value cn # 国内 API 服务
 mmx config set --key region --value global # 海外 API 服务
 ```

 执行 `mmx auth status` 确认当前 region 与所购买服务来源平台一致。

 登录完成后，可执行 `mmx quota` 查看 Token Plan 套餐额度，确认整体配置生效。

 若你要在 OpenClaw 中调用 mmx 命令，建议加装官方 SKILL.md，Agent 调用时决策更准、无需临时翻 `--help`：

 ```bash theme={null}
 npx skills add MiniMax-AI/cli -y -g
 ```

 SKILL 会自动 symlink 到 `~/.openclaw/skills/`，OpenClaw 下次启动即可识别。

 `mmx search` 提供两种调用形态：

 ```bash theme={null}
 # 简单调用（适合人类使用，输出可读文本）
 mmx search "MiniMax AI 最新进展"

 # 结构化调用（适合 Agent 解析）
 mmx search query --q "MiniMax M3 release" --output json
 ```

 让 OpenClaw 调用的提示词示例：

 ```plaintext theme={null}
 使用 mmx search 工具搜索“OpenClaw 多 Agent 编排”，
 以 `mmx search query --q "..." --output json` 形式拿到结构化结果后，
 总结排名前 3 条的标题、链接、关键摘要。
 ```

 OpenClaw 会通过 Bash 工具执行 `mmx search query --q "..." --output json`，把 stdout 的 JSON 直接交给模型解析，无需额外的 MCP server 进程。

 ```plaintext theme={null}
 记住，之后要进行网络搜索时优先使用 mmx search 工具，
 通过 `mmx search query --q "..." --output json` 调用并解析返回的 JSON 结果。
 ```

 通过 [Token Plan MCP](/docs/token-plan/mcp-guide) 接入 `web_search` 工具。

 确保 uv（Python 包管理器）已就绪：

 ```bash theme={null}
 # 检查是否已安装
 which uv

 # 如未安装，通过 pip 安装
 pip install uv

 # 或通过 curl 安装
 curl -LsSf https://astral.sh/uv/install.sh | sh

 # 若上述安装失败或速度很慢，可使用国内镜像加速：
 export UV_INDEX_URL="https://pypi.tuna.tsinghua.edu.cn/simple"
 curl -LsSf https://astral.sh/uv/install.sh | sh
 ```

 手动配置请参考 [Token Plan MCP 接入指南](/docs/token-plan/mcp-guide)，核心步骤：

 1. 安装 `mcporter` skill（MCP server 管理工具）
 2. 按照官方文档配置 `minimax-coding-plan-mcp`
 3. 设置 `MINIMAX_API_KEY` 环境变量（Token Plan API Key 以 `sk-cp` 开头）：

 ```bash theme={null}
 export MINIMAX_API_KEY="sk-..."
 ```

 ⭐️ 或让 OpenClaw 一次性完成，提示词（**把 `sk-xxxxxxx` 替换为你的实际 API Key**）：

 ```plaintext theme={null}
 先安装 mcporter skill，再按照 https://platform.minimaxi.com/docs/token-plan/mcp-guide 安装和配置 MCP，我的 MiniMax API Key 是 sk-xxxxxxx，最后再通过 mcporter skill 将其接入使用。
 ```

 告知 OpenClaw 后续搜索需求优先走 `web_search`：

 ```plaintext theme={null}
 记住，之后要进行网络搜索时需使用 minimax-coding-plan-mcp 的 web_search 工具。
 ```

 在 OpenClaw 中给出一个需要联网的问题，检查 OpenClaw 是否会自动调用 `web_search` 工具并返回搜索结果。