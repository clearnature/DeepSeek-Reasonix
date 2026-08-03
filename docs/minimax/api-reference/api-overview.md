# api-reference/api-overview.md

> 来源: https://platform.minimaxi.com/docs/api-reference/api-overview.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# 接口概览

> MiniMax 开放平台 API 接口能力概览，包括语言、视频、语音、图像、音乐和文件管理等多模态能力。

## 获取 API Key

* **按量付费**：通过 [接口密钥 > 创建新的 API Key](https://platform.minimaxi.com/user-center/basic-information/interface-key)，获取 **API Key**
  <Note>按量付费支持使用所有模态模型，包括语言、视频、语音、图像等</Note>

* **Token Plan**：通过 [订阅管理 > Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan)，查看 **订阅 Key**
  <Note>订阅 Key 用于 Token Plan 订阅套餐和已购积分，并与按量计费 API Key 相互独立。详情见 [Token Plan 概要](https://platform.minimaxi.com/docs/token-plan/intro)</Note>

***

## 语言模型

语言模型接口使用 **MiniMax M3**，**MiniMax M2.7**，**MiniMax M2.7-highspeed**，**MiniMax M2.5**，**MiniMax M2.5-highspeed**，**MiniMax M2.1**，**MiniMax M2.1-highspeed**，**MiniMax M2** 根据输入的上下文，让模型生成对话内容、工具调用。

可通过 **HTTP** 请求、**Anthropic SDK**（推荐） 或 **OpenAI SDK** 接入。

**支持模型**

| 模型名称                   | 输入输出总 token | 模型介绍                                        |
| :--------------------- | :---------: | :------------------------------------------ |
| MiniMax-M3             |  1,000,000  | **最新 M 系列语言模型，适用于 Agent 推理、工具调用、代码和长上下文任务** |
| MiniMax-M2.7           |    204800   | **开启模型的自我迭代 (输出速度约60tps)**                  |
| MiniMax-M2.7-highspeed |    204800   | **M2.7 极速版：效果不变，更快，更敏捷  (输出速度约100tps)**     |
| MiniMax-M2.5           |    204800   | **顶尖性能与极致性价比，轻松驾驭复杂任务 (输出速度约60tps)**        |
| MiniMax-M2.5-highspeed |    204800   | **M2.5 极速版：效果不变，更快，更敏捷  (输出速度约100tps)**     |
| MiniMax-M2.1           |    204800   | **强大多语言编程能力，全面升级编程体验 (输出速度约60tps)**         |
| MiniMax-M2.1-highspeed |    204800   | **M2.1 极速版：效果不变，更快，更敏捷  (输出速度约100tps)**     |
| MiniMax-M2             |    204800   | **专为高效编码与Agent工作流而生**                       |

如果在使用模型过程中遇到任何问题：

* 通过邮箱 [Model@minimaxi.com](mailto:Model@minimaxi.com) 等官方渠道联系我们的技术支持团队
* 在我们的 [Github](https://github.com/MiniMax-AI/MiniMax-M2/issues) 仓库提交Issue

<Columns cols={2}>
  <Card title="Anthropic API 兼容（推荐）" icon="book-open" href="/docs/api-reference/text-anthropic-api" cta="查看文档">
    通过 Anthropic SDK 调用 MiniMax 模型
  </Card>

  <Card title="OpenAI API 兼容" icon="book-open" href="/docs/api-reference/text-openai-api" cta="查看文档">
    通过 OpenAI SDK 调用 MiniMax 模型
  </Card>
</Columns>

***

## MiniMax-H3 \<NEW>

本接口基于 MiniMax-H3 模型，支持文本、图片、视频、音频等多模态输入进行视频生成，覆盖文生视频、图生视频、首尾帧生成视频、多模态参考生视频等场景。

**支持模型**

| 模型         | 功能                                                       |
| :--------- | :------------------------------------------------------- |
| MiniMax-H3 | 多模态视频生成模型，支持文生 / 图生 / 首尾帧 / 多模态参考，768P / 2K 分辨率，4–15s 时长 |

**接口说明**

MiniMax-H3 任务采用异步方式，提供**创建视频生成任务**、**创建 H3-Context-IR 任务**和**创建视频再生成任务**三个创建入口，并共用查询、列表以及取消或删除接口。使用步骤如下：

1. 根据场景创建视频生成任务，使用相同的多模态输入创建 H3-Context-IR 任务，或对符合 MiniMax-H3 768P 输出规格的源视频创建视频再生成任务；再生成请求必须且只能包含一个 `role=base_video` 的源视频项。成功后均返回 `task_id`。
2. 使用**查询任务**接口按 `task_id` 获取状态与结果；视频任务成功后从 `content.url` 获取成片地址，H3-Context-IR 任务成功后从 `content.prompt` 获取增强提示词。也可以使用**查询任务列表**接口批量查看，并通过 `task_type` 区分 `generation`、`h3_context_ir` 与 `regeneration`。
3. 对排队中的任务，可使用**取消或删除任务**接口取消；对成功或失败的任务，可使用同一接口删除任务记录。

<Columns cols={2}>
  <Card title="创建视频生成任务" icon="circle-play" href="/docs/api-reference/video-generation-v2-create" cta="查看文档">
    基于多模态 content 输入创建视频生成任务
  </Card>

  <Card title="创建 H3-Context-IR 任务" icon="pen-to-square" href="/docs/api-reference/video-generation-v2-h3-context-ir" cta="查看文档">
    深度理解视频生成的多模态上下文并生成结构化增强提示词
  </Card>

  <Card title="创建视频再生成任务" icon="wand-magic-sparkles" href="/docs/api-reference/video-generation-v2-regeneration" cta="查看文档">
    将符合 MiniMax-H3 768P 输出规格的视频再生成为 2K 视频
  </Card>

  <Card title="查询任务" icon="search" href="/docs/api-reference/video-generation-v2-query" cta="查看文档">
    按 task\_id 查询任务状态并获取成片下载地址
  </Card>

  <Card title="查询任务列表" icon="list" href="/docs/api-reference/video-generation-v2-list" cta="查看文档">
    分页查询最近 7 天内的任务并按任务类型过滤
  </Card>

  <Card title="取消或删除任务" icon="trash" href="/docs/api-reference/video-generation-v2-delete" cta="查看文档">
    取消排队中的任务，或删除成功和失败的任务记录
  </Card>
</Columns>

***

## 同步语音合成

本接口支持基于文本到语音的同步生成，单次可处理最长 **10,000 字符** 的文本。

接口本身为无状态接口，即单次调用时，模型仅处理单次传入内容，不涉及业务逻辑，同时模型也不存储您传入的数据。 该接口支持以下功能：

1. 支持 300+ 系统音色、复刻音色自主选择；
2. 支持音量、语调、语速、输出格式调整；
3. 支持按比例混音功能；
4. 支持固定间隔时间控制；
5. 支持多种音频规格、格式，包括：mp3, pcm, flac, wav；
6. 支持流式输出。

**支持模型**

以下为 MiniMax 提供的语音模型及其特性说明。

| 模型               | 特性                             |
| :--------------- | :----------------------------- |
| speech-2.8-hd    | 最新的 HD 模型，情绪渲染融合语气词，重塑自然听感     |
| speech-2.8-turbo | 最新的 Turbo 模型，极致生成速度，更自然逼真的音频效果 |
| speech-2.6-hd    | HD 模型，韵律表现出色，极致音质与韵律表现，生成更快更自然 |
| speech-2.6-turbo | Turbo 模型，音质优异，超低时延，响应更灵敏       |
| speech-02-hd     | 拥有出色的韵律、稳定性和复刻相似度，音质表现突出       |
| speech-02-turbo  | 拥有出色的韵律和稳定性，小语种能力加强，性能表现出色     |

**接口说明**

同步语音合成功能，共包含 2 个接口，可根据需求，选择使用。

* HTTP 同步语音合成
* WebSocket 同步语音合成

### 支持语言

MiniMax 的语音合成模型具备卓越的跨语言能力，全面支持 40 种全球广泛使用的语言。我们致力于打破语言壁垒，构建真正意义上的全球通用人工智能模型。

目前支持的语言包含：

| 支持语种                |                      |                       |
| :------------------ | :------------------- | :-------------------- |
| 1. 中文（Chinese）      | 15. 土耳其语（Turkish）    | 28. 马来语（Malay）        |
| 2. 粤语（Cantonese）    | 16. 荷兰语（Dutch）       | 29. 波斯语（Persian）      |
| 3. 英语（English）      | 17. 乌克兰语（Ukrainian）  | 30. 斯洛伐克语（Slovak）     |
| 4. 西班牙语（Spanish）    | 18. 泰语（Thai）         | 31. 瑞典语（Swedish）      |
| 5. 法语（French）       | 19. 波兰语（Polish）      | 32. 克罗地亚语（Croatian）   |
| 6. 俄语（Russian）      | 20. 罗马尼亚语（Romanian）  | 33. 菲律宾语（Filipino）    |
| 7. 德语（German）       | 21. 希腊语（Greek）       | 34. 匈牙利语（Hungarian）   |
| 8. 葡萄牙语（Portuguese） | 22. 捷克语（Czech）       | 35. 挪威语（Norwegian）    |
| 9. 阿拉伯语（Arabic）     | 23. 芬兰语（Finnish）     | 36. 斯洛文尼亚语（Slovenian） |
| 10. 意大利语（Italian）   | 24. 印地语（Hindi）       | 37. 加泰罗尼亚语（Catalan）   |
| 11. 日语（Japanese）    | 25. 保加利亚语（Bulgarian） | 38. 尼诺斯克语（Nynorsk）    |
| 12. 韩语（Korean）      | 26. 丹麦语（Danish）      | 39. 泰米尔语（Tamil）       |
| 13. 印尼语（Indonesian） | 27. 希伯来语（Hebrew）     | 40. 阿非利卡语（Afrikaans）  |
| 14. 越南语（Vietnamese） |                      |                       |

<Columns cols={2}>
  <Card title="HTTP 同步语音合成" icon="globe" href="/docs/api-reference/speech-t2a-http" cta="查看文档">
    通过 HTTP 请求进行语音合成
  </Card>

  <Card title="WebSocket 同步语音合成" icon="plug" href="/docs/api-reference/speech-t2a-websocket" cta="查看文档">
    通过 WebSocket 进行流式语音合成
  </Card>
</Columns>

***

## 异步长文本语音生成

该 API 支持基于文本到语音的异步生成，单次文本生成传输最大支持 **100 万字符**，生成的完整音频结果支持异步的方式进行检索。

该接口支持以下功能：

1. 支持 100+系统音色、复刻音色自主选择；
2. 支持语调、语速、音量、比特率、采样率、输出格式自主调整；
3. 支持音频时长、音频大小等返回参数；
4. 支持时间戳（字幕）返回，精确到句；
5. 支持直接传入字符串与上传文本文件 file\_id 两种方式进行待合成文本的输入；
6. 支持非法字符检测：非法字符不超过 10%（包含 10%），音频会正常生成并返回非法字符占比；非法字符超过 10%，接口不返回结果（返回报错码），请检测后再次进行请求【非法字符定义：ascii 码中的控制符（不含制表符和换行符）】。

提交长文本语音合成请求后，会生成 file\_id，生成任务完成后，可通过 file\_id 使用文件检索接口进行下载。

⚠️ 注意：返回的 url 的有效期为：自 url 返回开始的 **9 个小时**（即 32400 秒），超过有效期后 url 便会失效，生成的信息便会丢失，请注意下载信息的时间。

**适用场景：整本书籍等长文本的语音生成。**

**支持模型**

以下为 MiniMax 提供的语音模型及其特性说明。

| 模型               | 特性                             |
| :--------------- | :----------------------------- |
| speech-2.8-hd    | 最新的 HD 模型，情绪渲染融合语气词，重塑自然听感     |
| speech-2.8-turbo | 最新的 Turbo 模型，情绪渲染融合语气词，重塑自然听感  |
| speech-2.6-hd    | HD 模型，韵律表现出色，极致音质与韵律表现，生成更快更自然 |
| speech-2.6-turbo | Turbo 模型，音质优异，超低时延，响应更灵敏       |
| speech-02-hd     | 拥有出色的韵律、稳定性和复刻相似度，音质表现突出       |
| speech-02-turbo  | 拥有出色的韵律和稳定性，小语种能力加强，性能表现出色     |

**接口说明**

整体包含 2 个 API：创建**语音生成任务**、**查询语音生成任务状态**。使用步骤如下：

1. 创建语音生成任务得到 task\_id（如果选择以 file\_id 的形式传入待合成文本，需要前置使用 File(Upload)接口进行文件上传）；
2. 基于 taskid 查询语音生成任务状态；
3. 如果发现任务生成成功，那么可以使用本接口返回的 file\_id 通过 File API 进行结果查看和下载。

<Columns cols={2}>
  <Card title="创建异步语音任务" icon="circle-play" href="/docs/api-reference/speech-t2a-async-create" cta="查看文档">
    创建长文本语音生成任务
  </Card>

  <Card title="查询任务状态" icon="search" href="/docs/api-reference/speech-t2a-async-query" cta="查看文档">
    查询语音生成任务状态
  </Card>
</Columns>

***

## 音色快速复刻

本接口支持基于用户上传需要复刻音频的音频，以及示例音频，进行音色的复刻。

使用本接口需要完成个人认证及企业认证用户后，方可调用。 请在 [账户管理 -> 账户信息](https://platform.minimaxi.com/user-center/basic-information) 中，完成个人用户认证或企业用户认证，以确保可以正常使用本功能。

本接口适用场景：IP 音色复刻、音色克隆等需要快速复刻某一音色的相关场景。

本接口支持单、双声道复刻声音，支持按照指定音频文件快速复刻相同音色的语音。

**支持模型**

以下为 MiniMax 提供的语音模型及其特性说明。

| 模型               | 特性                             |
| :--------------- | :----------------------------- |
| speech-2.8-hd    | 最新的 HD 模型，情绪渲染融合语气词，重塑自然听感     |
| speech-2.8-turbo | 最新的 Turbo 模型，情绪渲染融合语气词，重塑自然听感  |
| speech-2.6-hd    | HD 模型，韵律表现出色，极致音质与韵律表现，生成更快更自然 |
| speech-2.6-turbo | Turbo 模型，音质优异，超低时延，响应更灵敏       |
| speech-02-hd     | 拥有出色的韵律、稳定性和复刻相似度，音质表现突出       |
| speech-02-turbo  | 拥有出色的韵律和稳定性，小语种能力加强，性能表现出色     |

**接口说明**

1. **上传待克隆音频** 调用 [上传复刻音频](/docs/api-reference/voice-cloning-uploadcloneaudio) 上传待克隆的音频文件并获取 `file_id`。
2. **上传示例音频 (可选)** 若需要提供示例音频以增强克隆效果，需要再次调用 [上传示例音频](/docs/api-reference/voice-cloning-uploadprompt) 上传示例音频文件并获得对应的 `file_id`。填写在`clone_prompt`中的`prompt_audio`中。
3. **调用复刻接口** 基于获取的 `file_id` 和自定义的 `voice_id` 作为输入参数，调用 [快速复刻](/docs/api-reference/voice-cloning-clone) 克隆音色。

### 注意事项

* 调用本接口进行音色克隆时，不会立即收取音色复刻费用。音色的复刻费用将在首次使用此复刻音色进行语音合成时收取（不包含本接口内的试听行为）。
* 本接口产出的快速复刻音色为临时音色，若希望永久保留某复刻音色，请于 168 小时（7 天）内在任意 T2A 语音合成接口中调用该音色（不包含本接口内的试听行为）。若超过时限，该音色将被删除。
* 接口采用无状态设计：每次调用仅处理传入数据，且不存储用户上传内容，不涉及任何业务逻辑状态。

<Columns cols={2}>
  <Card title="上传复刻音频" icon="upload" href="/docs/api-reference/voice-cloning-uploadcloneaudio" cta="查看文档">
    上传待克隆的音频文件
  </Card>

  <Card title="快速复刻" icon="mic" href="/docs/api-reference/voice-cloning-clone" cta="查看文档">
    执行音色克隆
  </Card>
</Columns>

***

## 音色设计

该 API 支持基于用户输入的声音描述 prompt，生成个性化定制音色。

本接口支持使用生成的音色（voice\_id）在[同步语音合成接口](/docs/api-reference/speech-t2a-intro)和[异步长文本语音合成接口](/docs/api-reference/speech-t2a-async-intro)中进行语音生成

**支持模型**

> 推荐使用 speech-02-hd 以获得最佳效果

| 模型               | 特性                             |
| :--------------- | :----------------------------- |
| speech-2.8-hd    | 最新的 HD 模型，情绪渲染融合语气词，重塑自然听感     |
| speech-2.8-turbo | 最新的 Turbo 模型，情绪渲染融合语气词，重塑自然听感  |
| speech-2.6-hd    | HD 模型，韵律表现出色，极致音质与韵律表现，生成更快更自然 |
| speech-2.6-turbo | Turbo 模型，音质优异，超低时延，响应更灵敏       |
| speech-02-hd     | 拥有出色的韵律、稳定性和复刻相似度，音质表现突出       |
| speech-02-turbo  | 拥有出色的韵律和稳定性，小语种能力加强，性能表现出色     |

### 注意事项

> * 调用本接口获得音色时，不会立即收取生成音色的费用，生成音色的费用将在首次使用此音色进行语音合成时收取（不包含本接口内的试听行为）。
> * 本接口产出的音色为临时音色，如您希望永久保留某音色，请于 168 小时（7 天）内在任意语音合成接口中调用该音色（不包含本接口内的试听行为），超过有效期未被使用的音色将自动删除。

<Card title="音色设计接口" icon="wand-magic-sparkles" href="/docs/api-reference/voice-design-design" cta="查看文档">
  基于描述生成个性化音色
</Card>

***

## 图像生成

本接口支持基于用户提供的文本或参考图片，进行创意图像生成。支持设置不同图片比例和长宽像素设置，满足不同场景下图像需求。

**接口说明**

通过创建图片生成任务接口，使用文本描述和参考图片，进行图像生成。

**模型列表**

| 模型名称          | 简介                              |
| :------------ | :------------------------------ |
| image-01      | 图像生成模型，画面表现细腻，支持文生图、图生图（人物主体参考） |
| image-01-live | 图像生成模型，在 image-01 基础上额外支持多种画风设置 |

<Columns cols={2}>
  <Card title="文生图" icon="file-text" href="/docs/api-reference/image-generation-t2i" cta="查看文档">
    基于文本描述生成图像
  </Card>

  <Card title="图生图" icon="image-plus" href="/docs/api-reference/image-generation-i2i" cta="查看文档">
    基于参考图片生成图像
  </Card>
</Columns>

***

## 音乐生成

本接口根据歌曲描述（prompt）和歌词（lyrics），生成一首人声的歌曲。

**支持模型**

| 模型名称      | 使用方法                            |
| :-------- | :------------------------------ |
| music-3.0 | 最新音乐生成模型，支持用户输入音乐灵感和歌词，生成 AI 音乐 |

<Card title="音乐生成接口" icon="music" href="/docs/api-reference/music-generation" cta="查看文档">
  根据描述和歌词生成音乐
</Card>

***

## 文件管理

本接口是作为文件管理接口，配合 MiniMax 开放平台的其他接口使用。

**接口说明**

本接口是作为文件管理接口，配合其他接口使用。共包含 5 个接口：**上传**、**列出**、**检索**、**下载**、**删除**。

文件的支持格式、容量及大小限制以**上传文件**接口文档为准，详见 [上传文件](/docs/api-reference/file-management-upload)。

<Columns cols={2}>
  <Card title="上传文件" icon="upload" href="/docs/api-reference/file-management-upload" cta="查看文档">
    上传文件到平台
  </Card>

  <Card title="文件列表" icon="list" href="/docs/api-reference/file-management-list" cta="查看文档">
    获取已上传的文件列表
  </Card>
</Columns>

***

## 官方 MCP

MiniMax 提供官方的 [Python 版本](https://github.com/MiniMax-AI/MiniMax-MCP) 和 [JavaScript 版本](https://github.com/MiniMax-AI/MiniMax-MCP-JS) 模型上下文协议（MCP）服务器实现代码，支持语音合成、音色克隆、视频生成、音乐生成等功能，详细说明请参考 [MiniMax MCP 使用指南](/docs/guides/mcp-guide)

## 语音调试台

<Columns cols={2}>
  <Card title="语音合成调试台" icon="audio-lines" href="https://platform.minimaxi.com/examination-center/voice-experience-center/t2a_v2" cta="立即体验语音合成能力" />

  <Card title="音色快速复刻调试台" icon="mic" href="https://platform.minimaxi.com/examination-center/voice-experience-center/voiceCloning" cta="立即体验音色快速复刻能力" />
</Columns>
