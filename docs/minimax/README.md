# MiniMax（稀宇科技）技术文档

> MiniMax 开放平台全量技术文档（来源: platform.minimaxi.com，抓取于 2026-08-03）
> 组织原则：按技术关联度分层——对话 LLM 核心 → 多模态扩展 → 平台能力 → 接入指南 → 资费

## 一、对话 LLM 核心（Reasonix 直接相关）

### text/ — 文本对话 API（OpenAI/Anthropic/Responses 三协议 + 缓存）

| 文档 | 说明 |
|------|------|
| [responses-create](api-reference/text/responses-create.md) | OpenAI Responses API 兼容主接口（流式/非流式，MiniMax-M3 推理控制） |
| [responses-create-html](api-reference/text/responses-create-html.md) | Responses API（HTML 渲染版，含 schema 详情） |
| [responses-input-tokens](api-reference/text/responses-input-tokens.md) | Responses input tokens 计费说明 |
| [text-chat-openai](api-reference/text/text-chat-openai.md) | OpenAI Chat Completions 兼容 |
| [text-chat-anthropic](api-reference/text/text-chat-anthropic.md) | Anthropic Messages 兼容 |
| [text-openai-api](api-reference/text/text-openai-api.md) | OpenAI 原生 API 参考 |
| [text-anthropic-api](api-reference/text/text-anthropic-api.md) | Anthropic 原生 API 参考 |
| [text-post](api-reference/text/text-post.md) | 文本对话主接口 |
| [text-prompt-caching](api-reference/text/text-prompt-caching.md) | 提示词缓存（自动/主动两模式） |
| [anthropic-api-compatible-cache](api-reference/text/anthropic-api-compatible-cache.md) | Anthropic 缓存兼容 |
| [text-ai-sdk](api-reference/text/text-ai-sdk.md) | 官方 SDK |

### models/ — 模型查询

- [list-models](api-reference/models/openai/list-models.md)（OpenAI 兼容）
- [retrieve-model](api-reference/models/openai/retrieve-model.md)
- [list-models](api-reference/models/anthropic/list-models.md)（Anthropic 兼容）
- [retrieve-model](api-reference/models/anthropic/retrieve-model.md)

### 基础

- [api-overview](api-reference/api-overview.md) — 接口总览
- [api-overview-html](api-reference/api-overview-html.md) — 接口总览（HTML 渲染版）
- [errorcode](api-reference/errorcode.md) — 错误码

## 二、多模态扩展（语音/图像/视频/音乐/文件）

### media/speech/ — 语音合成

- [speech-t2a-http](api-reference/media/speech/speech-t2a-http.md) · [speech-t2a-websocket](api-reference/media/speech/speech-t2a-websocket.md)
- [speech-t2a-async-create](api-reference/media/speech/speech-t2a-async-create.md) · [speech-t2a-async-query](api-reference/media/speech/speech-t2a-async-query.md)

### media/voice/ — 声音克隆与设计

- [voice-cloning-clone](api-reference/media/voice/voice-cloning-clone.md) · [voice-cloning-uploadcloneaudio](api-reference/media/voice/voice-cloning-uploadcloneaudio.md) · [voice-cloning-uploadprompt](api-reference/media/voice/voice-cloning-uploadprompt.md)
- [voice-design-design](api-reference/media/voice/voice-design-design.md)
- [voice-management-get](api-reference/media/voice/voice-management-get.md) · [voice-management-delete](api-reference/media/voice/voice-management-delete.md)

### media/image/ — 图像生成

- [image-generation-t2i](api-reference/media/image/image-generation-t2i.md) · [image-generation-i2i](api-reference/media/image/image-generation-i2i.md)

### media/video/ — 视频生成

- [video-generation-t2v](api-reference/media/video/video-generation-t2v.md) · [video-generation-i2v](api-reference/media/video/video-generation-i2v.md) · [video-generation-s2v](api-reference/media/video/video-generation-s2v.md) · [video-generation-fl2v](api-reference/media/video/video-generation-fl2v.md)
- [video-generation-v2-create](api-reference/media/video/video-generation-v2-create.md) · [video-generation-v2-query](api-reference/media/video/video-generation-v2-query.md) · [video-generation-v2-regeneration](api-reference/media/video/video-generation-v2-regeneration.md) · [video-generation-v2-delete](api-reference/media/video/video-generation-v2-delete.md) · [video-generation-v2-list](api-reference/media/video/video-generation-v2-list.md)
- [video-generation-v2-h3-context-ir](api-reference/media/video/video-generation-v2-h3-context-ir.md) · [video-generation-query](api-reference/media/video/video-generation-query.md) · [video-generation-download](api-reference/media/video/video-generation-download.md)
- [video-agent-create](api-reference/media/video/video-agent-create.md) · [video-agent-query](api-reference/media/video/video-agent-query.md)

### media/music/ — 音乐与歌词

- [music-generation](api-reference/media/music/music-generation.md) · [music-cover-preprocess](api-reference/media/music/music-cover-preprocess.md) · [lyrics-generation](api-reference/media/music/lyrics-generation.md)

### media/file/ — 文件管理

- [file-management-upload](api-reference/media/file/file-management-upload.md) · [file-management-list](api-reference/media/file/file-management-list.md) · [file-management-retrieve](api-reference/media/file/file-management-retrieve.md) · [file-management-retrieve-content](api-reference/media/file/file-management-retrieve-content.md) · [file-management-delete](api-reference/media/file/file-management-delete.md)

## 三、接入指南

- [guides/](guides/) — 24 篇（模型介绍/限速/推理/MCP/Agent/Claude Code/Cursor 接入）
- [token-plan/](token-plan/) — 16 篇（Codex/Claude Code/Cline/MCP/迁移等）

## 四、资费与公告

- [pricing](pricing/) · [faq](faq/) · [release-notes](release-notes/)
