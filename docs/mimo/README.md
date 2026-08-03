# MiMo（小米）技术文档

> Xiaomi MiMo API 全量技术文档（来源: mimo.mi.com + mimo.xiaomi.com，抓取于 2026-08-03）
> 组织原则：按技术关联度分层——对话 LLM 核心 → 快速开始 → 官方博客/论文 → 接入计划 → 公告

## 一、对话 LLM 核心（Reasonix 直接相关）

### api/chat/ — 三协议对话 API

| 文档 | 说明 |
|------|------|
| [responses](api/chat/responses.md) / [responses.zh-CN](api/chat/responses.zh-CN.md) | **OpenAI Responses API**（多轮工具/思考模式/多模态，厂商主推协议） |
| [openai-api](api/chat/openai-api.md) / [.zh-CN](api/chat/openai-api.zh-CN.md) | OpenAI Chat Completions 兼容 |
| [anthropic-api](api/chat/anthropic-api.md) / [.zh-CN](api/chat/anthropic-api.zh-CN.md) | Anthropic Messages 兼容 |

### api/guidance/ — 平台指引

- [rate-limit](api/guidance/rate-limit.md) — 速率限制（mimo-v2.5/pro: RPM=100, TPM=10M）
- [model-hyperparameters](api/guidance/model-hyperparameters.md) — 模型超参（max_output_tokens 131072 / temperature 1.5）
- [error-codes](api/guidance/error-codes.md) — 错误码

### api/ — 其他

- [responses-api-full](api/responses-api-full.md) — Responses API 完整 schema（400 行 curl + 1151 行字段）
- [model](api/model/) — 模型列表
- [audio](api/audio/) — 语音识别（ASR）

## 二、快速开始

- [quick-start/](quick-start/) — 19 篇（思考模式/工具调用/多模态/结构化输出/FAQ/条款）

## 三、官方博客与研究（推理路线参考）

### blog/ — 官方博客

- [v2-5-inference-blog](blog/v2-5-inference-blog.md) — MiMo-v2.5 推理技术博客
- [inference-optimization-blog-analysis](blog/inference-optimization-blog-analysis.md) — 推理优化分析（长前缀缓存 95% 命中）
- [mimo-code-long-horizon](blog/mimo-code-long-horizon.md) — 长程代码智能体
- [mimo-tilert-1000tps](blog/mimo-tilert-1000tps.md) — TILERT 1000 TPS 推理
- [mimo-v2-flash-hss](blog/mimo-v2-flash-hss.md) · [mimo-v2-flash-safety](blog/mimo-v2-flash-safety.md) — v2-flash

### paper/ — 研究论文摘要

- [mimo-reasoning](paper/mimo-reasoning.md) — 推理论文
- [mimo-audio](paper/mimo-audio.md) — 音频
- [hysparse](paper/hysparse.md) — HySparse（混合稀疏注意力，缓存 TTL 长设计依据）
- [arl-tangram](paper/arl-tangram.md) — ARL-Tangram

## 四、接入计划与资费

- [tokenplan/](tokenplan/) — 12 篇（Codex/Claude Code/Cline 等接入）
- [price/](price/) · [news/](news/)（24 篇公告）· [updates/](updates/)
