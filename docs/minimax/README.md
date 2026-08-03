# MiniMax（稀宇科技）技术文档

> MiniMax API 官方技术文档全集（来源: platform.minimaxi.com llms.txt，Mintlify 原生 md，抓取于 2026-08-03）

## 目录

### api-reference
API 参考（Chat openai/anthropic、Responses、缓存、文件、语音、图像、视频、音乐、错误码）

- [anthropic-api-compatible-cache](api-reference/anthropic-api-compatible-cache.md)
- [api-overview](api-reference/api-overview.md)
- [errorcode](api-reference/errorcode.md)
- [file-management-delete](api-reference/file-management-delete.md)
- [file-management-list](api-reference/file-management-list.md)
- [file-management-retrieve-content](api-reference/file-management-retrieve-content.md)
- [file-management-retrieve](api-reference/file-management-retrieve.md)
- [file-management-upload](api-reference/file-management-upload.md)
- [image-generation-i2i](api-reference/image-generation-i2i.md)
- [image-generation-t2i](api-reference/image-generation-t2i.md)
- [lyrics-generation](api-reference/lyrics-generation.md)
- [list-models](api-reference/models/anthropic/list-models.md)
- [retrieve-model](api-reference/models/anthropic/retrieve-model.md)
- [list-models](api-reference/models/openai/list-models.md)
- [retrieve-model](api-reference/models/openai/retrieve-model.md)
- [music-cover-preprocess](api-reference/music-cover-preprocess.md)
- [music-generation](api-reference/music-generation.md)
- [responses-create](api-reference/responses-create.md)
- [responses-input-tokens](api-reference/responses-input-tokens.md)
- [speech-t2a-async-create](api-reference/speech-t2a-async-create.md)
- [speech-t2a-async-query](api-reference/speech-t2a-async-query.md)
- [speech-t2a-http](api-reference/speech-t2a-http.md)
- [speech-t2a-websocket](api-reference/speech-t2a-websocket.md)
- [text-ai-sdk](api-reference/text-ai-sdk.md)
- [text-anthropic-api](api-reference/text-anthropic-api.md)
- [text-chat-anthropic](api-reference/text-chat-anthropic.md)
- [text-chat-openai](api-reference/text-chat-openai.md)
- [text-openai-api](api-reference/text-openai-api.md)
- [text-post](api-reference/text-post.md)
- [text-prompt-caching](api-reference/text-prompt-caching.md)
- [video-agent-create](api-reference/video-agent-create.md)
- [video-agent-query](api-reference/video-agent-query.md)
- [video-generation-download](api-reference/video-generation-download.md)
- [video-generation-fl2v](api-reference/video-generation-fl2v.md)
- [video-generation-i2v](api-reference/video-generation-i2v.md)
- [video-generation-query](api-reference/video-generation-query.md)
- [video-generation-s2v](api-reference/video-generation-s2v.md)
- [video-generation-t2v](api-reference/video-generation-t2v.md)
- [video-generation-v2-create](api-reference/video-generation-v2-create.md)
- [video-generation-v2-delete](api-reference/video-generation-v2-delete.md)
- [video-generation-v2-h3-context-ir](api-reference/video-generation-v2-h3-context-ir.md)
- [video-generation-v2-list](api-reference/video-generation-v2-list.md)
- [video-generation-v2-query](api-reference/video-generation-v2-query.md)
- [video-generation-v2-regeneration](api-reference/video-generation-v2-regeneration.md)
- [voice-cloning-clone](api-reference/voice-cloning-clone.md)
- [voice-cloning-uploadcloneaudio](api-reference/voice-cloning-uploadcloneaudio.md)
- [voice-cloning-uploadprompt](api-reference/voice-cloning-uploadprompt.md)
- [voice-design-design](api-reference/voice-design-design.md)
- [voice-management-delete](api-reference/voice-management-delete.md)
- [voice-management-get](api-reference/voice-management-get.md)

### guides
指南（模型介绍/限速/MCP/Agent 接入）

- [image-generation](guides/image-generation.md)
- [local-deploy](guides/local-deploy.md)
- [mcp-guide](guides/mcp-guide.md)
- [models-intro](guides/models-intro.md)
- [music-generation](guides/music-generation.md)
- [pricing-paygo](guides/pricing-paygo.md)
- [pricing-speech](guides/pricing-speech.md)
- [pricing-token-plan-team](guides/pricing-token-plan-team.md)
- [pricing-token-plan](guides/pricing-token-plan.md)
- [pricing-video](guides/pricing-video.md)
- [privacy-policy](guides/privacy-policy.md)
- [quickstart-preparation](guides/quickstart-preparation.md)
- [quickstart-sdk](guides/quickstart-sdk.md)
- [rate-limits](guides/rate-limits.md)
- [server-tools](guides/server-tools.md)
- [speech-t2a-async](guides/speech-t2a-async.md)
- [speech-t2a-websocket](guides/speech-t2a-websocket.md)
- [speech-voice-clone](guides/speech-voice-clone.md)
- [terms-of-service](guides/terms-of-service.md)
- [text-generation](guides/text-generation.md)
- [text-m3-function-call](guides/text-m3-function-call.md)
- [token-plan-mcp-guide](guides/token-plan-mcp-guide.md)
- [video-generation](guides/video-generation.md)
- [video-prompt](guides/video-prompt.md)

### token-plan
Token Plan

- [claude-code](token-plan/claude-code.md)
- [codex](token-plan/codex.md)
- [cursor](token-plan/cursor.md)
- [faq](token-plan/faq.md)
- [hermes-agent](token-plan/hermes-agent.md)
- [intro](token-plan/intro.md)
- [mcp-guide](token-plan/mcp-guide.md)
- [migration](token-plan/migration.md)
- [mini-agent](token-plan/mini-agent.md)
- [minimax-cli](token-plan/minimax-cli.md)
- [openclaw](token-plan/openclaw.md)
- [other-tools](token-plan/other-tools.md)
- [promotion](token-plan/promotion.md)
- [prompting-best-practices](token-plan/prompting-best-practices.md)
- [quickstart](token-plan/quickstart.md)
- [trae](token-plan/trae.md)

### pricing
定价

- [overview](pricing/overview.md)

### faq
FAQ

- [about-account](faq/about-account.md)
- [about-apis](faq/about-apis.md)
- [contact-us](faq/contact-us.md)
- [history-query](faq/history-query.md)
- [system-voice-id](faq/system-voice-id.md)

### release-notes
发布记录

- [apis](release-notes/apis.md)
- [models](release-notes/models.md)
