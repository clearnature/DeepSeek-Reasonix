# token-plan/codex.md

> 来源: https://platform.minimaxi.com/docs/token-plan/codex.md

# Codex

> 在 Codex 桌面客户端中使用最新的 MiniMax M 系列模型进行 AI 编程。

[**Codex**](https://developers.openai.com/codex/) 是 OpenAI 官方的桌面端 AI 编程 Agent。

## 安装 Codex

从 [OpenAI Codex 页面](https://developers.openai.com/codex/) 下载并安装 Codex 桌面客户端。

## 配置 MiniMax API

 打开 `~/.codex/config.toml`，加入以下内容，并将 `` 替换为你从 [MiniMax 开放平台](https://platform.minimaxi.com/user-center/payment/token-plan) 获取的 Key：

 ```toml theme={null}
 model = "MiniMax-M3"
 model_provider = "minimax"
 model_context_window = 1000000

 [model_providers.minimax]
 name = "MiniMax"
 base_url = "https://api.minimaxi.com/v1"
 experimental_bearer_token = ""
 wire_api = "responses"
 ```

 重启 Codex —— 即可开始使用 MiniMax-M3。

## 配置模型能力目录（可选）

Codex 可以通过自定义模型目录识别 MiniMax-M3 的多模态输入、reasoning effort（thinking 开关）、system prompt、工具类型以及其他详细参数。配置完成后，在 Codex CLI 中输入 `/model`，即可在模型列表中看到 MiniMax-M3 及其可选 reasoning level。

在 `~/.codex/config.toml` 中增加一行：

```toml theme={null}
model_catalog_json = "~/.codex/model-catalogs/custom-catalog.json"
```

然后新建 `~/.codex/model-catalogs/custom-catalog.json`，写入模型详细配置：

```json theme={null}
{
 "models": [
 {
 "slug": "MiniMax-M3",
 "display_name": "MiniMax-M3",
 "description": "MiniMax",
 "default_reasoning_level": "high",
 "supported_reasoning_levels": [
 { "effort": "none", "description": "Think-Off" },
 { "effort": "high", "description": "Deep" }
 ],
 "shell_type": "shell_command",
 "visibility": "list",
 "supported_in_api": true,
 "priority": 0,
 "base_instructions": "You are Codex, a coding agent based on MiniMax-M3. You and the user share the same workspace and collaborate to achieve the user's goals.",
 "supports_reasoning_summaries": true,
 "default_reasoning_summary": "none",
 "support_verbosity": false,
 "truncation_policy": { "mode": "bytes", "limit": 10000 },
 "supports_parallel_tool_calls": true,
 "experimental_supported_tools": [],
 "input_modalities": ["text", "image"]
 }
 ]
}
```

其中常用字段含义如下：

* `slug` / `display_name`：模型在 Codex 配置与 `/model` 列表中的标识和展示名称，需与 API 中使用的模型名保持一致。
* `default_reasoning_level`：默认 reasoning effort。对于 MiniMax-M3，任意非 `none` 值都会开启 Adaptive Thinking；该值不用于调节 reasoning 深度。
* `supported_reasoning_levels`：在 `/model` 中可切换的 reasoning 选项。`none` 表示关闭 thinking；`high` 表示开启 Adaptive Thinking。
* `base_instructions`：Codex 使用该模型时附加的基础 system prompt，可用于声明模型身份和协作方式。
* `supports_reasoning_summaries`：开启 Codex 对该模型的 Responses API reasoning 路径。设置为 `true` 后，Codex 才会发送 `reasoning.effort`；否则即使配置了 `default_reasoning_level`，Codex 也会省略 `reasoning` 字段。示例中将 `default_reasoning_summary` 设为 `none`，表示不额外请求 reasoning summary。
* `shell_type`：声明模型适配的 shell 工具调用类型，示例中使用 `shell_command`。
* `visibility` / `supported_in_api` / `priority`：控制模型是否出现在列表中、是否可通过 API 使用，以及在模型列表中的排序优先级。
* `supports_parallel_tool_calls`：声明模型支持并行工具调用，便于 Codex 处理多个工具请求。
* `experimental_supported_tools`：预留的实验性工具能力列表；没有额外工具时保持空数组即可。
* `input_modalities`：声明模型支持的输入模态。`["text", "image"]` 表示支持文本和图片输入。
* `truncation_policy`：控制上下文截断策略，示例中按字节数限制工具或上下文保留内容。

修改后重启 Codex，使新的模型目录生效。