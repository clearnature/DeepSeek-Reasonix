# guides/token-plan-mcp-guide.md

> 来源: https://platform.minimaxi.com/docs/guides/token-plan-mcp-guide.md

# Token Plan MCP

> Token Plan MCP 提供了两个专属工具：**网络搜索** 和 **图片理解**，帮助开发者在编码过程中快速获取信息和理解图片内容。 

推荐使用 [MiniMax CLI](/docs/token-plan/minimax-cli) 替代 MCP，配置更简单、使用更高效。

## 工具说明

 根据搜索查询词进行网络搜索，返回搜索结果和相关搜索建议。

 | 参数 | 类型 | 必需 | 说明 |
 | :---- | :----- | :-: | :---- |
 | query | string | ✓ | 搜索查询词 |

 对图片进行理解和分析，支持多种图片输入方式。

 | 参数 | 类型 | 必需 | 说明 |
 | :--------- | :----- | :-: | :----------------------------- |
 | prompt | string | ✓ | 对图片的提问或分析要求 |
 | image\_url | string | ✓ | 图片来源，支持 HTTP/HTTPS URL 或本地文件路径 |

 **支持格式**：JPEG、PNG、GIF、WebP（最大 20MB）

## 前置准备

 访问 [订阅管理 > Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan) 查看您的订阅 Key。该 Key 需要拥有 Token Plan 席位或已购积分权限后，才能使用付费资源。

 ```bash theme={null}
 curl -LsSf https://astral.sh/uv/install.sh | sh
 ```

 ```powershell theme={null}
 powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"
 ```

 其他安装方式可参考 [uv 仓库](https://github.com/astral-sh/uv)。

 ```bash theme={null}
 which uvx
 ```

 ```powershell theme={null}
 (Get-Command uvx).source
 ```

 若正确安装，会显示路径（如 `/usr/local/bin/uvx`）。若报错 `spawn uvx ENOENT`，需配置绝对路径。

## 在 Claude Code 中使用

 在 [Claude Code 官网](https://www.claude.com/product/claude-code)下载并安装 Claude Code

 在终端运行以下命令，将`api_key`替换为您的 API Key：

 ```bash theme={null}
 claude mcp add -s user MiniMax --env MINIMAX_API_KEY=api_key --env MINIMAX_API_HOST=https://api.minimaxi.com -- uvx minimax-coding-plan-mcp -y
 ```

 编辑配置文件 `~/.claude.json`，添加以下 MCP 配置：

 ```json theme={null}
 {
 "mcpServers": {
 "MiniMax": {
 "command": "uvx",
 "args": ["minimax-coding-plan-mcp", "-y"],
 "env": {
 "MINIMAX_API_KEY": "MINIMAX_API_KEY",
 "MINIMAX_API_HOST": "https://api.minimaxi.com"
 }
 }
 }
 }
 ```

 进入 Claude Code 后输入 `/mcp`，能看到 `web_search` 和 `understand_image`，说明配置成功。

 如果您在 IDE（如 TRAE）中使用 MCP，还需要在对应 IDE 的 MCP 配置中进行设置

## 在 Cursor 中使用

 通过 [Cursor 官网](https://cursor.com/) 下载并安装 Cursor

 前往 `Cursor -> Preferences -> Cursor Settings -> Tools & Integrations -> MCP -> Add Custom MCP`

 ![Cursor MCP 配置](https://filecdn.minimax.chat/public/61982fde-6575-4230-94eb-798f35a60450.png)

 在 `mcp.json` 文件中添加以下配置：

 ```json theme={null}
 {
 "mcpServers": {
 "MiniMax": {
 "command": "uvx",
 "args": ["minimax-coding-plan-mcp"],
 "env": {
 "MINIMAX_API_KEY": "填写你的 API Key",
 "MINIMAX_MCP_BASE_PATH": "本地输出目录路径，需保证路径存在且有写入权限",
 "MINIMAX_API_HOST": "https://api.minimaxi.com",
 "MINIMAX_API_RESOURCE_MODE": "可选，资源提供方式：url 或 local，默认 url"
 }
 }
 }
 }
 ```

## 在 OpenCode 中使用

 通过 [OpenCode 官网](https://opencode.ai/) 下载并安装 OpenCode

 编辑配置文件 `~/.config/opencode/opencode.json`，添加以下 MCP 配置：

 ```json theme={null}
 {
 "$schema": "https://opencode.ai/config.json",
 "mcp": {
 "MiniMax": {
 "type": "local",
 "command": ["uvx", "minimax-coding-plan-mcp", "-y"],
 "environment": {
 "MINIMAX_API_KEY": "MINIMAX_API_KEY",
 "MINIMAX_API_HOST": "https://api.minimaxi.com"
 },
 "enabled": true
 }
 }
 }
 ```

 进入 OpenCode 后，输入 `/mcp`，能看到 `MiniMax connected`，说明配置成功。