# guides/mcp-guide.md

> 来源: https://platform.minimaxi.com/docs/guides/mcp-guide.md

# MiniMax MCP

> 本文档介绍模型上下文协议（MCP）的原理与应用，提供 Python 与 JS 版本工具说明，支持语音合成、音色克隆、图像与视频生成等多模态能力，助力开发者高效集成 AI 功能。

推荐使用 [MiniMax CLI](/docs/token-plan/minimax-cli) 替代 MCP，配置更简单、使用更高效。

## 模型上下文协议（MCP）简介

[**模型上下文协议(MCP）**](https://modelcontextprotocol.io/docs/getting-started/intro) 是一个开放协议，标准化应用程序向大语言模型提供工具和上下文的方式。它类似于 AI 领域的 USB‑C 接口，提供一个稳定和标准化的接入点，让模型能访问数据库、API、插件或其他工具。通过 MCP 工具，开发者可以让模型访问托管在远程 MCP 服务器上的各种工具。

**MiniMax 提供官方的 [Python 版本](https://github.com/MiniMax-AI/MiniMax-MCP)和 [JavaScript 版本](https://github.com/MiniMax-AI/MiniMax-MCP-JS) 模型上下文协议(MCP)**、声音克隆、图像生成、视频生成等多模态能力。 开发者可自行部署 MCP 服务，并通过 MCP 客户端（如 Claude Desktop、Cursor、Windsurf、OpenAI Agents 等）调用，从而快速集成语音、图像和视频相关功能。在传输方面，Python 版本提供 stdio 和 SSE 两种标准传输方式，JS 版本提供 stdio 、REST 和 SSE 三种标准传输方式。

## MiniMax MCP 工具和参数介绍

### MCP 工具清单

 该工具可将将输入的文本合成为自然流畅的语音

 该工具可查询所有可用音色

 该工具可根据指定音频文件克隆音色

 该工具可根据指定提示词生成音色和试听文本

 该工具用于播放一个音频文件

 该工具可根据指定提示词和歌词生成音乐

 该工具可根据指定文本或图片进行视频生成生成

 该工具用于使用首帧图像生成视频

 该工具用于查询异步视频生成任务的状态

 该工具可根据指定提示词生成图片

## 工具与参数详情

### 1. text\_to\_audio

该工具可将输入的文本合成为自然流畅的语音。

| 参数 | 含义 | 格式及说明 | 默认值 |
| :----------------- | :------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | :---------------- |
| `text` | **必需**，待合成的文本 | 字符串，长度限制 \ Settings > Developer > Edit Config > claude_desktop_config.json` ，添加以下配置。完成配置后，重启 Claude Desktop。

注意：如果使用 Windows，需要在 Claude Desktop 中启用"开发者模式"才能使用 MCP 服务器。 若出现报错 `spawn uvx ENOENT` ，请在 `command` 中配置 `uvx` 的绝对路径

```json theme={null}
{
 "mcpServers": {
 "MiniMax": {
 "command": "uvx",
 "args": ["minimax-mcp"],
 "env": {
 "MINIMAX_API_KEY": "填写你的 API Key",
 "MINIMAX_MCP_BASE_PATH": "本地输出目录路径，如/User/xxx/Desktop",
 "MINIMAX_API_HOST": "填写API Host, https://api.minimaxi.com 或 https://api.minimax.io",
 "MINIMAX_API_RESOURCE_MODE": "可选配置，资源生成后的提供方式, 可选项为 [url|local], 默认为 url"
 },
 "transport": "可选配置，传输方式，可选项为 [studio|SSE]，默认为 studio"
 }
 }
}
```

### 在 Cursor 中使用

1. 通过 [Cursor 官网](https://cursor.com/) 下载并安装 Cursor
2. 前往 `Cursor -> Preferences -> Cursor Settings -> Tools & Inrgrations -> MCP -> Add Custom MCP` ，打开 MCP 工具配置文件

![Cursor MCP 配置](https://filecdn.minimax.chat/public/61982fde-6575-4230-94eb-798f35a60450.png)

3. 在 `mcp.json` 文件中，增加 MiniMax 账户配置信息

```json theme={null}
{
 "mcpServers": {
 "MiniMax": {
 "command": "uvx",
 "args": ["minimax-mcp"],
 "env": {
 "MINIMAX_API_KEY": "填写你的 API Key",
 "MINIMAX_MCP_BASE_PATH": "本地输出目录路径，如/User/xxx/Desktop，需要保证路径存在且具有写入权限",
 "MINIMAX_API_HOST": "填写API Host, https://api.minimaxi.com 或 https://api.minimax.io",
 "MINIMAX_API_RESOURCE_MODE": "可选配置，资源生成后的提供方式, 可选项为 [url|local], 默认为 url"
 },
 "transport": "可选配置，传输方式，可选项为 [studio|SSE]，默认为 studio"
 }
 }
}
```

### 在 Cherry Studio 中使用

1. 通过 [Cherry Studio 官网](https://www.cherry-ai.com/) 下载客户端
2. 前往 `Settings -> MCP Settings -> Add Server -> Import from JSON` ，将以下代码粘贴到代码框中，确认

```json theme={null}
{
 "name": "minimax-mcp",
 "isActive": true,
 "command": "uvx",
 "args": [
 "minimax-mcp"
 ],
 "env": {
 "MINIMAX_API_KEY": "填写你的 API Key",
 "MINIMAX_MCP_BASE_PATH": "本地输出目录路径，如/User/xxx/Desktop，需要保证路径存在且具有写入权限",
 "MINIMAX_API_HOST": "填写API Host, https://api.minimaxi.com 或 https://api.minimax.io",
 "MINIMAX_API_RESOURCE_MODE": "可选配置，资源生成后的提供方式, 可选项为 [url|local], 默认为 url"
 }
 "transport": "可选配置，传输方式，可选项为 [studio|SSE]，默认为 studio"
 }
```

3. 在对话框中，点击 `MCP Settings` 后，选择完成配置的“MiniMax MCP”即可使用

![Cherry Studio 配置](https://filecdn.minimax.chat/public/10883349-c9ff-4c92-b1d7-bccf15a5f28f.png)

## 在客户端使用 MiniMax MCP JS 服务器

### 获取 API Key

* 访问 [MiniMax 开放平台](https://platform.minimaxi.com/user-center/payment/token-plan)
* 点击“**Create new secret key**”按钮，输入项目名称以创建新的 API Key。
* 创建成功后，系统将展示 API Key。**请务必复制并妥善保存**，该密钥**只会显示一次**，无法再次查看。

![获取 API Key](https://filecdn.minimax.chat/public/155646cf-6e99-4ecd-8843-8232668d5f67.png)

### Node.js 与 npm 安装

Node.js 是一个开源的 JavaScript 运行时环境，可以在浏览器之外运行 JavaScript 代码。它基于 Google 的 V8 引擎，具有高性能、事件驱动和非阻塞 I/O 的特点，适合构建高并发的网络服务、实时应用和微服务等场景。 npm 是随 Node.js 一起安装的默认包管理器，也是全球最大的软件注册中心。开发者可以通过 npm 搜索、安装、更新和管理依赖包（包括前端和后端代码模块），大幅简化开发流程。

1. [安装 Node.js 与 npm](https://nodejs.org/en/download)
2. **验证安装是否完成**

执行以下命令，若正确安装，会显示 Node.js 和 npm 的版本

```python theme={null}
node -v
npm -v
```

### 传输方式说明

MiniMax-MCP-JS 提供 stdio、REST 和 SSE 三种传输方式，在使用时可按需选择

| 特性 | stdio (默认) | REST | SSE |
| :--- | :------------------ | :-------------------- | :-------------------- |
| 运行环境 | 本地运行 | 可本地或云端部署 | 可本地或云端部署 |
| 通信方式 | 通过标准输入输出通信 | 通过 HTTP 请求通信 | 通过服务器发送事件通信 |
| 适用场景 | 本地 MCP 客户端集成 | API 服务，跨语言调用 | 需要服务器推送的应用 |
| 输入限制 | 支持处理本地文件或有效的 URL 资源 | 当部署在云端时，建议使用 URL 作为输入 | 当部署在云端时，建议使用 URL 作为输入 |

### 在 Claude Desktop 中使用

1. 在 Claude 官网下载 [Claude Desktop](https://claude.ai/download)
2. 前往 `Claude > Settings > Developer > Edit Config > claude_desktop_config.json` ，添加以下配置。完成配置后，重启 Claude Desktop。

注意：如果使用 Windows，需要在 Claude Desktop 中启用"开发者模式"才能使用 MCP 服务器。

```json theme={null}
{
 "mcpServers": {
 "minimax-mcp-js": {
 "command": "npx",
 "args": ["-y", "minimax-mcp-js"],
 "env": {
 "MINIMAX_API_HOST": "https://api.minimaxi.com",
 "MINIMAX_API_KEY": "",
 "MINIMAX_MCP_BASE_PATH": "",
 "MINIMAX_RESOURCE_MODE": ""
 },
 "transport": "可选配置，传输方式，可选项为 [studio|REST|SSE]，默认为 studio"
 }
 }
}
```

### 在 Cursor 中使用

1. 通过 [Cursor 官网](https://cursor.com/) 下载并安装 Cursor
2. 前往 `Cursor -> Preferences -> Cursor Settings -> Tools & Inrgrations -> MCP -> Add Custom MCP` ，打开 MCP 工具配置文件

![Cursor MCP 配置](https://filecdn.minimax.chat/public/61982fde-6575-4230-94eb-798f35a60450.png)

3. 在 `mcp.json` 文件中，增加 MiniMax 账户配置信息

```json theme={null}
{
 "mcpServers": {
 "MiniMax": {
 "command": "uvx",
 "args": ["minimax-mcp"],
 "env": {
 "MINIMAX_API_KEY": "填写你的 API Key",
 "MINIMAX_MCP_BASE_PATH": "本地输出目录路径，如/User/xxx/Desktop，需要保证路径存在且具有写入权限",
 "MINIMAX_API_HOST": "填写API Host, https://api.minimaxi.com 或 https://api.minimax.io",
 "MINIMAX_API_RESOURCE_MODE": "可选配置，资源生成后的提供方式, 可选项为 [url|local], 默认为 url"
 },
 "transport": "可选配置，传输方式，可选项为 [studio|REST|SSE]，默认为 studio"
 }
 }
}
```

4. 完成配置后，可以查看 MiniMax 目前支持的 mcp 工具

![MCP 工具列表](https://filecdn.minimax.chat/public/72ceca39-5edc-4f18-84cc-599715d3475f.png)

### 在 Cherry Studio 中使用

1. 通过 [Cherry Studio 官网](https://www.cherry-ai.com/) 下载客户端
2. 前往 `Settings -> MCP Settings -> Add Server -> Import from JSON` ，将以下代码粘贴到代码框中，确认

```json theme={null}
{
 "name": "minimax-mcp",
 "isActive": true,
 "command": "npx",
 "args": ["y", "minimax-mcp"],
 "env": {
 "MINIMAX_API_KEY": "填写你的 API Key",
 "MINIMAX_MCP_BASE_PATH": "本地输出目录路径，如/User/xxx/Desktop，需要保证路径存在且具有写入权限",
 "MINIMAX_API_HOST": "填写API Host, https://api.minimaxi.com 或 https://api.minimax.io",
 "MINIMAX_API_RESOURCE_MODE": "可选配置，资源生成后的提供方式, 可选项为 [url|local], 默认为 url"
 },
 "transport": "可选配置，传输方式，可选项为 [studio|REST|SSE]，默认为 studio"
}
```

3. 在对话框中，点击 `MCP Settings` 后，选择完成配置的“MiniMax MCP”即可使用

![Cherry Studio 配置](https://filecdn.minimax.chat/public/10883349-c9ff-4c92-b1d7-bccf15a5f28f.png)

## MiniMax MCP 使用示例

### 音频工具使用

1. 选择合适的声音信息，播报晚间新闻片段

参考提示词

```python theme={null}
choose a voice, and broadcast a segment of the evening news
```

生成内容

思考过程

![思考过程](https://filecdn.minimax.chat/public/e9f12d7f-e7d8-4f3d-b273-181a51433606.png)

2. 根据指定音频克隆声音，并指定克隆音色的 id

参考提示词

```python theme={null}
clone the voice from the audio file named Marketing_Voice.sav, the id is test_vlone_voice
```

来源音频

结果音频

思考过程

![思考过程](https://filecdn.minimax.chat/public/3e73f8ca-39e6-420f-8b90-8773d4db6e14.png)

3. 按照要求设计音色，并给定示例文本生成音频

参考提示词

```python theme={null}
Design a voice, the requirement is "Mysterious narrator with a deep, magnetic voice, suspenseful tone, moderate pace, subtle reverb". Then use it in the sample Text: "In the shadows of the old manor, secrets whisper through the walls. Beware what you seek…"
```

生成内容

思考过程

![思考过程](https://filecdn.minimax.chat/public/a44cade5-1f33-4d54-a70a-2fec83b73a0a.png)

### 音乐生成工具使用

参考提示词

```python theme={null}
generate a song, the background music is gentle ambient piano and warm pad synth, soft reverb and subtle field recordings of wind chimes. The musical style: calm and reflective
Lyrics:
‘In the stillness of the midnight air,
Find the echoes of dreams we share.
Softly drifting ‘neath pale moonlight,
Whispering hearts drifting into night.’"
```

生成内容

思考过程

![思考过程](https://filecdn.minimax.chat/public/a53e06e9-e10e-46d4-82b2-2c829261709c.png)

### 图片生成工具使用

参考提示词

```python theme={null}
generate a hyperreal style picture, the requirement is "Ultra‑detailed digital painting of a serene mountain lake at sunrise, ultra-realistic, soft golden light, mist over the water"
```

生成内容

![生成的图片](https://filecdn.minimax.chat/public/09b2c787-8c37-4f80-94b5-c1855ce05e97.jpeg)

思考过程

![思考过程](https://filecdn.minimax.chat/public/2017abf5-8168-43dd-b938-fb60179d3562.png)

### 视频生成工具使用

参考提示词及图片

```python theme={null}
From the existing image of a kitten perched on a diving board, create a short video showing the kitten crouching, leaping off into the pool, and making a small splash—adorable and playful. Use MiniMax-Hailuo-02 model, and resolution is 1080P
```

![输入图片](https://filecdn.minimax.chat/public/eeee4d8d-7834-4370-bd55-e5579e739a3e.jpeg)

生成内容

思考过程

![思考过程](https://filecdn.minimax.chat/public/02b88e6b-63b1-4989-a821-285809d45eae.png)

## 如何做出自己的贡献

若希望对 MiniMax MCP 项目进行改进或修复错误，欢迎通过以下方式提交建议或代码贡献：

1. 在 GitHub 项目主页（ [Python 版](https://github.com/MiniMax-AI/MiniMax-MCP/issues)或 [JS 版](https://github.com/MiniMax-AI/MiniMax-MCP-JS/issues) ）开一个新的 Issue，简要描述您建议的更改或问题。
2. 获取反馈后，请按照项目贡献指南创建一个对应的 **Pull Request (PR)** ，附上修改说明及必要的背景信息。
3. 项目维护者会对你的 PR 进行代码审查，并给予合并建议或进一步修改意见。