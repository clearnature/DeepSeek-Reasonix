# guides/mcp-guide.md

> 来源: https://platform.minimaxi.com/docs/guides/mcp-guide.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# MiniMax MCP

> 本文档介绍模型上下文协议（MCP）的原理与应用，提供 Python 与 JS 版本工具说明，支持语音合成、音色克隆、图像与视频生成等多模态能力，助力开发者高效集成 AI 功能。

<Tip>推荐使用 [MiniMax CLI](/docs/token-plan/minimax-cli) 替代 MCP，配置更简单、使用更高效。</Tip>

## 模型上下文协议（MCP）简介

[**模型上下文协议(MCP）**](https://modelcontextprotocol.io/docs/getting-started/intro) 是一个开放协议，标准化应用程序向大语言模型提供工具和上下文的方式。它类似于 AI 领域的 USB‑C 接口，提供一个稳定和标准化的接入点，让模型能访问数据库、API、插件或其他工具。通过 MCP 工具，开发者可以让模型访问托管在远程 MCP 服务器上的各种工具。

**MiniMax 提供官方的 [Python 版本](https://github.com/MiniMax-AI/MiniMax-MCP)和 [JavaScript 版本](https://github.com/MiniMax-AI/MiniMax-MCP-JS) 模型上下文协议(MCP)**、声音克隆、图像生成、视频生成等多模态能力。 开发者可自行部署 MCP 服务，并通过 MCP 客户端（如 Claude Desktop、Cursor、Windsurf、OpenAI Agents 等）调用，从而快速集成语音、图像和视频相关功能。在传输方面，Python 版本提供 stdio 和 SSE 两种标准传输方式，JS 版本提供 stdio 、REST 和 SSE 三种标准传输方式。

## MiniMax MCP 工具和参数介绍

### MCP 工具清单

<Columns cols={3}>
  <Card title="text_to_audio" icon="volume-2">
    该工具可将将输入的文本合成为自然流畅的语音
  </Card>

  <Card title="list_voices" icon="volume-2">
    该工具可查询所有可用音色
  </Card>

  <Card title="voice_clone" icon="volume-2">
    该工具可根据指定音频文件克隆音色
  </Card>

  <Card title="voice_design" icon="volume-2">
    该工具可根据指定提示词生成音色和试听文本
  </Card>

  <Card title="play_audio" icon="volume-2">
    该工具用于播放一个音频文件
  </Card>

  <Card title="music_generation" icon="music">
    该工具可根据指定提示词和歌词生成音乐
  </Card>

  <Card title="generate_video" icon="video">
    该工具可根据指定文本或图片进行视频生成生成
  </Card>

  <Card title="image_to_video" icon="video">
    该工具用于使用首帧图像生成视频
  </Card>

  <Card title="query_video_generation" icon="video">
    该工具用于查询异步视频生成任务的状态
  </Card>

  <Card title="text_to_image" icon="image">
    该工具可根据指定提示词生成图片
  </Card>
</Columns>

## 工具与参数详情

### 1. text\_to\_audio

该工具可将输入的文本合成为自然流畅的语音。

| 参数                 | 含义                         | 格式及说明                                                                                                                                                                                                                                                                                                                                       | 默认值               |
| :----------------- | :------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | :---------------- |
| `text`             | **必需**，待合成的文本              | 字符串，长度限制 \< 10000 字符，段落切换用换行符替代。                                                                                                                                                                                                                                                                                                            | 无                 |
| `output_directory` | 保存音频文件的目录                  | 字符串 (文件路径)                                                                                                                                                                                                                                                                                                                                  | 配置文件中设置的路径        |
| `voice_id`         | 请求的音色编号                    | 字符串，可选值可参考 [API 手册](/docs/faq/system-voice-id)                                                                                                                                                                                                                                                                                                   | `"female-shaonv"` |
| `model`            | 请求的模型版本                    | 字符串，可选值可参考 [API 手册](/docs/api-reference/speech-t2a-http)                                                                                                                                                                                                                                                                                         | `"speech-02-hd"`  |
| `speed`            | 生成音频的语速                    | 浮点数，范围 \[0.5 - 2.0]                                                                                                                                                                                                                                                                                                                         | `1.0`             |
| `vol`              | 生成音频的音量                    | 浮点数，范围                                                                                                                                                                                                                                                                                                                                      | `1.0`             |
| `pitch`            | 生成音频的音调                    | 整数，范围 \[-12,12]                                                                                                                                                                                                                                                                                                                             | `0` (原音色输出)       |
| `emotion`          | 控制合成语音的情绪                  | 字符串，可选值范围`["happy", "sad", "angry", "fearful", "disgusted", "surprised", "calm", "whipser"]`，该参数仅对 speech-2.8-hd, speech-2.8-turbo, speech-2.6-hd, speech-2.6-turbo, speech-02-hd, speech-02-turbo, speech-01-turbo, speech-01-hd 模型生效；可选值范围`["fluent", "whipser"]`仅对 speech-2.8-turbo, speech-2.8-hd, speech-2.6-turbo, speech-2.6-hd 模型生效 | `happy`           |
| `sample_rate`      | 生成音频的采样率                   | 整数，可选范围 `[8000, 16000, 22050, 24000, 32000, 44100]`                                                                                                                                                                                                                                                                                         | `32000`           |
| `bitrate`          | 生成音频的比特率                   | 整数，可选范围 `[32000, 64000, 128000, 256000]`，该参数仅对 mp3 格式的音频生效                                                                                                                                                                                                                                                                                  | `128000`          |
| `channel`          | 生成音频的声道数                   | 整数，可选值为 1：单声道, 2：双声道                                                                                                                                                                                                                                                                                                                        | `1`               |
| `format`           | 生成的音频文件格式                  | 字符串，可选范围`["pcm", "mp3", "flac", "wav"]`                                                                                                                                                                                                                                                                                                     | `mp3`             |
| `language_boost`   | 语言增强选项，提升在指定小语种或方言场景下的语音表现 | 字符串，可选值可参考 [API 手册](/docs/api-reference/speech-t2a-http)                                                                                                                                                                                                                                                                                         | `null`            |

### 2. list\_voices

该工具可查询所有可用音色

| 参数          | 含义         | 格式及说明                                                                                                                                                  | 默认值   |
| :---------- | :--------- | :----------------------------------------------------------------------------------------------------------------------------------------------------- | :---- |
| voice\_type | 需要查询的音色类型。 | 字符串, 可选值范围 \["system" (系统音色) , "voice\_cloning" (快速复刻的音色) , "voice\_generation" (文生音色接口生成的音色) , "music\_generation" (音乐生成产生的人声或者伴奏音色) , "all" (以上全部) ] | "all" |

### 3. voice\_clone

该工具可根据指定音频文件克隆音色

| 参数                | 含义                | 格式及说明                                                                                                        | 默认值        |
| :---------------- | :---------------- | :----------------------------------------------------------------------------------------------------------- | :--------- |
| voice\_id         | **必需**, 音色复刻的声音编号 | 字符串, 可自定义, 注意事项包括 1. 自定义的 voice\_id 长度范围 2. 首字符必须为英文字母 3. 允许数字、字母、-、\_4. 末位字符不可为-、\_5. 创建的 voice\_id 不可与之前重复 | 无          |
| file              | **必需**, 用于克隆的音频文件 | 字符串, 可选范围\["mp3", "m4a", "wav"]                                                                              | 无          |
| text              | 生成克隆音色演示音频的文本     | 字符串, 限制 2000 字符以内                                                                                            | 无          |
| output\_directory | 保存音频文件的目录         | 字符串 (文件路径)                                                                                                   | 配置文件中设置的路径 |
| is\_url           | 用于克隆的音频文件来源是否 url | 布尔值 (True / False)                                                                                           | FALSE      |

### 4. voice\_design

该工具可根据指定提示词生成音色和试听文本

| 参数                | 含义                | 格式及说明      | 默认值        |
| :---------------- | :---------------- | :--------- | :--------- |
| prompt            | **必需**, 生成音色的描述   | 字符串        | 无          |
| preview\_text     | **必需**, 生成试听音频的文本 | 字符串        | 无          |
| voice\_id         | 自定义生成音色的 ID       | 字符串        | 自动生成唯一值    |
| output\_directory | 保存试听音频文件的目录       | 字符串 (文件路径) | 配置文件中设置的路径 |

### 5. play\_audio

该工具用于播放一个音频文件。

| 参数                | 含义                         | 格式及说明              | 默认值   |
| :---------------- | :------------------------- | :----------------- | :---- |
| input\_file\_path | **必需**, 要播放的音频文件的本地路径或 URL | 字符串 (文件路径或 URL)    | 无     |
| is\_url           | 需要播放的音频文件来源是否 url          | 布尔值 (True / False) | FALSE |

### 6. music\_generation

该工具可根据指定提示词和歌词生成音乐

| 参数                | 含义                                   | 可选值/格式                                                                              | 默认值        |
| :---------------- | :----------------------------------- | :---------------------------------------------------------------------------------- | :--------- |
| prompt            | **必需**, 表示音乐创作的灵感，通常是对音乐风格、情绪、场景等的描述 | 字符串, 字符取值范围 \[10-300]                                                               | 无          |
| lyrics            | **必需**, 音乐生成的歌词                      | 字符串, 使用换行符 (`\n`) 分隔每行歌词, 支持结构标签 \[Intro]\[Verse]\[Chorus]\[Bridge]\[Outro], 字符取值范围 | 无          |
| sample\_rate      | 生成音乐的采样率                             | 整数, 可选值范围                                                                           | 32000      |
| bitrate           | 生成音乐的比特率                             | 整数, 可选值范围                                                                           | 128000     |
| format            | 生成音乐的格式                              | 字符串, 可选值范围 \["mp3", "wav", "pcm"]                                                   | "mp3"      |
| output\_directory | 保存音乐文件的目录                            | 字符串 (文件路径)                                                                          | 配置文件中设置的路径 |

### 7. generate\_video

该工具可根据指定提示词生成视频, `prompt` 和 `first_frame_image` 两个参数至少要有一个。

| 参数                  | 含义                   | 格式及说明                                                                                                                                                                                                                                              | 默认值        |
| :------------------ | :------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :--------- |
| `prompt`            | 生成视频的描述              | 字符串, 最大支持 2000 字符                                                                                                                                                                                                                                  | 无          |
| `model`             | 请求的模型版本              | 字符串, 可选范围 `MiniMax-Hailuo-02`, `T2V-01-Director`, `I2V-01-Director`, `S2V-01`, `I2V-01-live`, `I2V-01`, `T2V-01`                                                                                                                                   | `T2V-01`   |
| `first_frame_image` | 视频的首帧画面              | 支持传入图片的 "data:image/jpeg;base64,data" 格式的 Base64 编码字符串, 或可通过公网访问的 URL                                                                                                                                                                              | 无          |
| `duration`          | 生成视频的持续时间 (秒)        | 整数, 可选值, 与分辨率和模型相关:<br />**01 系列** (包括 T2V-01, I2V-01,T2V-01-Director, I2V-01-Director, I2V-01-live, S2V-01): 可选值: 6<br />**02 系列** (MiniMax-Hailuo-02) : 512P: 可选值: 6, 10 768P: 可选值: 6, 10 1080P: 可选值: 6                                          | `6`        |
| `resolution`        | 生成视频的分辨率             | 字符串, 与选择模型及设置的视频时长相关:<br />**01 系列** (包括 T2V-01, I2V-01,T2V-01-Director, I2V-01-Director, I2V-01-live, S2V-01): 不支持设置本参数。<br />**02 系列** <br />6s 时长: 默认值为"720P", 可选范围 `["512P", "768P", "1080P"]`<br />10s 时长: 默认值"768P", 可选范围 `["512P", "768P"]` | `None`     |
| `output_directory`  | 保存视频文件的目录            | 字符串 (文件路径)                                                                                                                                                                                                                                         | 配置文件中设置的路径 |
| `async_mode`        | 是否使用异步模式, 启用则返回任务 ID | 布尔值 (True / False)                                                                                                                                                                                                                                 | `FALSE`    |

### 8. image\_to\_video

该工具用于使用首帧图像生成视频, 其中 `prompt` 和 `first_frame_image` 两个参数至少有其一, 该 MCP 工具仅在 **JavaScript/TypeScript 版本**中可用

| 参数                  | 含义                   | 格式及说明                                                                                                               | 默认值        |
| :------------------ | :------------------- | :------------------------------------------------------------------------------------------------------------------ | :--------- |
| prompt              | **必需**, 生成视频的描述。     | 字符串, 最大支持 2000 字符                                                                                                   | 无          |
| model               | 请求的模型版本              | 字符串, 可选范围 \["MiniMax-Hailuo-02", "T2V-01-Director", "I2V-01-Director", "S2V-01", "I2V-01-live", "I2V-01", "T2V-01"] | "T2V-01"   |
| first\_frame\_image | **必需**, 视频的首帧画面      | 支持传入图片的 "data:image/jpeg;base64,data" 格式的 Base64 编码字符串, 或可通过公网访问的 URL                                               | 无          |
| output\_directory   | 保存视频文件的目录            | 字符串 (文件路径)                                                                                                          | 配置文件中设置的路径 |
| async\_mode         | 是否使用异步模式, 启用则返回任务 ID | 布尔值 (True / False)                                                                                                  | FALSE      |

### 9. query\_video\_generation

该工具用于查询异步视频生成任务的状态。

| 参数                | 含义                 | 格式及说明      | 默认值        |
| :---------------- | :----------------- | :--------- | :--------- |
| task\_id          | **必需**, 需要查询的任务 ID | 字符串        | 无          |
| output\_directory | 保存视频文件的目录          | 字符串 (文件路径) | 配置文件中设置的路径 |

### 10. text\_to\_image

该工具可根据指定提示词生成图片

| 参数                | 含义              | 格式及说明                                                                  | 默认值        |
| :---------------- | :-------------- | :--------------------------------------------------------------------- | :--------- |
| prompt            | **必需**, 生成图像的描述 | 字符串, 最大支持 1500 字符                                                      | 无          |
| model             | 请求的模型版本         | 字符串, 可选范围 \["image-01", "image-01-live"]                               | "image-01" |
| aspect\_ratio     | 生成图像的宽高比        | 字符串, 可选范围 \["1:1", "16:9", "4:3", "3:2", "2:3", "3:4", "9:16", "21:9"] | "1:1"      |
| n                 | 单次请求生成的图片数量     | 整数, 可选范围                                                               | 1          |
| prompt\_optimizer | 是否开启提示词自动优化提示词  | 布尔值 (True / False)                                                     | TRUE       |
| output\_directory | 保存图片文件的目录       | 字符串 (文件路径)                                                             | 配置文件中设置的路径 |

## 在客户端使用 MiniMax MCP 服务

### 获取 API Key

* 访问 [MiniMax 开放平台](https://platform.minimaxi.com/user-center/payment/token-plan)
* 点击“**Create new secret key**”按钮，输入项目名称以创建新的 API Key。
* 创建成功后，系统将展示 API Key。**请务必复制并妥善保存**，该密钥**只会显示一次**，无法再次查看。

![获取 API Key](https://filecdn.minimax.chat/public/0b03e12d-191d-45f4-826a-5e997fa845c2.png)

### UVX 安装和配置

**MiniMax‑MCP** 是一个 Python 实现的 MCP 服务，为了让 MCP 客户端顺利调用，该服务必须通过 `uvx` 启动和执行。 `uvx` 和 `uv` 提供的命令行工具，类似于 `npm exec` ，用于运行包内部定义的可执行程序，确保环境隔离与依赖可控。

1. 安装 `uv` 以便获取 `uvx`
   * macOS / Linux 用户：

```python theme={null}
curl -LsSf https://astral.sh/uv/install.sh | sh
```

* Windows 用户：

```python theme={null}

powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"
```

其他安装方式可参考 [uv 仓库](https://github.com/astral-sh/uv)。安装完成后，会在 Python 环境的 `Scripts` 或 `bin` 目录中生成 `uv` 和 `uvx` 可执行文件。

2. 验证 `uvx` 是否可用

执行命令：

* macOS / Linux 用户：

```python theme={null}
which uvx
```

* Windows 用户：

```python theme={null}
(Get-Command uvx).source
```

* 若正确安装，会显示路径（如 `/usr/local/bin/uvx` ）。
* 若系统报错 `spawn uvx ENOENT` ，说明没有安装 `uvx` 或 `uvx` 不在系统路径中，需要配置绝对路径。

### 传输方式说明

MiniMax-MCP 提供 stdio 和 SSE 两种传输方式，在使用时可按需选择

| 特性     | stdio（默认）         | SSE（Server‑Sent Events）       |
| :----- | :---------------- | :---------------------------- |
| 运行环境   | 本地部署运行            | 本地部署或云端部署均可                   |
| 通信方式   | 通过 stdout 进行通信    | 通过网络通信                        |
| 适用场景   | 本地 MCP 客户端集成      | 需要服务器推送的应用                    |
| 输入资源支持 | 支持本地文件或有效的 URL 资源 | 支持本地文件或 URL；部署在云端时推荐使用 URL 输入 |

### 在 Claude Desktop 中使用

1. 在 [Claude 官网](https://claude.ai/download)下载 Claude Desktop
2. 前往 `Claude > Settings > Developer > Edit Config > claude_desktop_config.json` ，添加以下配置。完成配置后，重启 Claude Desktop。

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

| 特性   | stdio (默认)          | REST                  | SSE                   |
| :--- | :------------------ | :-------------------- | :-------------------- |
| 运行环境 | 本地运行                | 可本地或云端部署              | 可本地或云端部署              |
| 通信方式 | 通过标准输入输出通信          | 通过 HTTP 请求通信          | 通过服务器发送事件通信           |
| 适用场景 | 本地 MCP 客户端集成        | API 服务，跨语言调用          | 需要服务器推送的应用            |
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
        "MINIMAX_API_KEY": "<您的 API Key>",
        "MINIMAX_MCP_BASE_PATH": "<本地输出目录路径，如/User/xxx/Desktop>",
        "MINIMAX_RESOURCE_MODE": "<可选配置，资源生成后的提供方式, [url|local], 默认为 url>"
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

<video controls src="https://filecdn.minimax.chat/public/87394f73-1ed6-4507-8eb6-d1e7abb65e23.mp3" className="audio-container" />

思考过程

![思考过程](https://filecdn.minimax.chat/public/e9f12d7f-e7d8-4f3d-b273-181a51433606.png)

2. 根据指定音频克隆声音，并指定克隆音色的 id

参考提示词

```python theme={null}
clone the voice from the audio file named Marketing_Voice.sav, the id is test_vlone_voice
```

来源音频

<video controls src="https://filecdn.minimax.chat/public/cd3c2e0c-8cb8-48ff-999c-96d5f070765f.wav" className="audio-container" />

结果音频

<video controls src="https://filecdn.minimax.chat/public/678a8b1c-fd02-48a5-b250-e225119d9448.mp3" className="audio-container" />

思考过程

![思考过程](https://filecdn.minimax.chat/public/3e73f8ca-39e6-420f-8b90-8773d4db6e14.png)

3. 按照要求设计音色，并给定示例文本生成音频

参考提示词

```python theme={null}
Design a voice, the requirement is "Mysterious narrator with a deep, magnetic voice, suspenseful tone, moderate pace, subtle reverb". Then use it in the sample Text: "In the shadows of the old manor, secrets whisper through the walls. Beware what you seek…"
```

生成内容

<video controls src="https://filecdn.minimax.chat/public/98488c0c-c9e5-47f0-bab0-3b1b3f4dae3a.mp3" className="audio-container" />

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

<video controls src="https://filecdn.minimax.chat/public/737fa5dc-0ecf-4fd6-9821-197e9490d3e3.mp3" className="audio-container" />

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

<video controls src="https://filecdn.minimax.chat/public/515dec59-459c-4560-8557-00a81e630138.mp4" />

思考过程

![思考过程](https://filecdn.minimax.chat/public/02b88e6b-63b1-4989-a821-285809d45eae.png)

## 如何做出自己的贡献

若希望对 MiniMax MCP 项目进行改进或修复错误，欢迎通过以下方式提交建议或代码贡献：

1. 在 GitHub 项目主页（ [Python 版](https://github.com/MiniMax-AI/MiniMax-MCP/issues)或 [JS 版](https://github.com/MiniMax-AI/MiniMax-MCP-JS/issues) ）开一个新的 Issue，简要描述您建议的更改或问题。
2. 获取反馈后，请按照项目贡献指南创建一个对应的 **Pull Request (PR)** ，附上修改说明及必要的背景信息。
3. 项目维护者会对你的 PR 进行代码审查，并给予合并建议或进一步修改意见。
