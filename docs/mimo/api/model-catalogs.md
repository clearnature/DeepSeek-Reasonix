# MiMo-Code Model Catalogs 配置

> 来源: MiMo-Code CLI 工具 (`/data/training/cli/MiMo-Code`) 的 model-catalogs.json 配置
> 日期: 2026-08-07

## 概述

MiMo-Code 支持通过 `.codex/model-catalogs/model-catalogs.json` 文件自定义模型参数配置，
可针对 MiMo 模型进行精细化个性化适配。配置完成后，在 Codex CLI 中输入 `/model` 即可在
模型列表中看到 MiMo 模型及其对应的推理档位，并支持随时切换使用。

## 关键字段说明

| 字段 | 作用 |
|------|------|
| `slug` | 模型唯一内部标识，必须和后端 API 模型名称完全一致 |
| `display_name` | 前端界面展示用模型名称 |
| `description` | 模型简介文案 |
| `default_reasoning_level` | 新建会话默认推理强度档位，none 为关闭思考，其他为开启思考 |
| `supported_reasoning_levels` | 客户端允许用户切换的推理强度配置列表 |
| `supports_reasoning_summaries` | **总开关**：true 时请求携带推理参数；false 则清空所有推理字段 |
| `default_reasoning_summary` | 推理摘要默认输出模式，none 为不单独输出推理摘要 |
| `input_modalities` | 支持的输入模态类型 |
| `supports_image_detail_original` | 图片输入是否支持原图高清解析 |
| `context_window` | 模型标称总上下文窗口大小 |
| `supports_parallel_tool_calls` | 是否支持并行多工具调用 |

## mimo-v2.5-pro vs mimo-v2.5 对比

| 配置项 | mimo-v2.5-pro | mimo-v2.5 |
|--------|---------------|-----------|
| `input_modalities` | `["text"]` | `["text", "image"]` |
| `supports_image_detail_original` | false | **true** |
| `context_window` | 1048576 (1M) | 1048576 (1M) |
| `default_reasoning_level` | "high" | "high" |
| `supported_reasoning_levels` | none, high | none, high |
| `supports_reasoning_summaries` | true | true |
| `default_reasoning_summary` | "none" | "none" |
| `supports_parallel_tool_calls` | false | false |
| `supports_search_tool` | false | false |

**关键差异**：mimo-v2.5 支持图像输入（多模态），mimo-v2.5-pro 是纯文本深度推理模型。

## Reasonix 对齐状态

| MiMo-Code 配置 | Reasonix vendor.go | 状态 |
|----------------|-------------------|------|
| `default_max_output_tokens` (MIMO_OUTPUT_TOKEN_MAX=128000) | `defaultMaxOutputTokens: 128000` | ✅ |
| `supports_reasoning_summaries = true` | effort != "" 时发送 reasoning 对象 | ✅ |
| `default_reasoning_summary = "none"` | `summaryMode: "none"` | ✅ |
| `default_reasoning_level = "high"` | 用户配置 effort | ✅ |
| reasoning 模型不发 temperature/top_p | `ignoresTemperature: true` | ✅ |
| `singleSegmentReasoning` | `false`（MiMo 支持多段推理） | ✅ |

## 完整配置示例

```json
{
  "models": [
    {
      "slug": "mimo-v2.5-pro",
      "display_name": "mimo-v2.5-pro",
      "description": "MiMo-v2.5-Pro: Trillion-parameter Flagship Agent Foundation",
      "default_reasoning_level": "high",
      "supported_reasoning_levels": [
        {"effort": "none", "description": "Disable Thinking"},
        {"effort": "high", "description": "Enabled Thinking"}
      ],
      "shell_type": "shell_command",
      "visibility": "list",
      "supported_in_api": true,
      "priority": 0,
      "base_instructions": "You are MiMo, an AI assistant developed by Xiaomi. Today's date: {date} {week}. Your knowledge cutoff date is December 2024.",
      "supports_reasoning_summaries": true,
      "default_reasoning_summary": "none",
      "support_verbosity": false,
      "truncation_policy": {"mode": "bytes", "limit": 10000},
      "supports_parallel_tool_calls": false,
      "supports_image_detail_original": false,
      "context_window": 1048576,
      "max_context_window": 1048576,
      "effective_context_window_percent": 95,
      "experimental_supported_tools": [],
      "input_modalities": ["text"],
      "supports_search_tool": false
    },
    {
      "slug": "mimo-v2.5",
      "display_name": "mimo-v2.5",
      "description": "MiMo-V2.5: Native Omni-modal Perception Model",
      "default_reasoning_level": "high",
      "supported_reasoning_levels": [
        {"effort": "none", "description": "Disable Thinking"},
        {"effort": "high", "description": "Enabled Thinking"}
      ],
      "shell_type": "shell_command",
      "visibility": "list",
      "supported_in_api": true,
      "priority": 1,
      "base_instructions": "You are MiMo, an AI assistant developed by Xiaomi. Today's date: {date} {week}. Your knowledge cutoff date is December 2024.",
      "supports_reasoning_summaries": true,
      "default_reasoning_summary": "none",
      "support_verbosity": false,
      "truncation_policy": {"mode": "bytes", "limit": 10000},
      "supports_parallel_tool_calls": false,
      "supports_image_detail_original": true,
      "context_window": 1048576,
      "max_context_window": 1048576,
      "effective_context_window_percent": 95,
      "experimental_supported_tools": [],
      "input_modalities": ["text", "image"],
      "supports_search_tool": false
    }
  ]
}
```
