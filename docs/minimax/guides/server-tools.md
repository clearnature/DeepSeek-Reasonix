# guides/server-tools.md

> 来源: https://platform.minimaxi.com/docs/guides/server-tools.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# 服务端工具（Server Tools）

## 什么是服务端工具

**服务端工具（Server Tools）** 是由 MiniMax 在服务端托管并自动执行的内置工具。与传统的 Function Call（客户端工具）不同，你无需自行实现工具的执行逻辑，也无需在多轮对话中手动回传工具结果——模型在生成回复的过程中会自动调用这些工具、获取结果并继续生成，整个流程在一次 API 请求内完成。

<Info>
  该能力目前处于 **Beta** 阶段，功能与参数可能会调整。
</Info>

<CardGroup cols={2}>
  <Card title="客户端工具（Function Call）" icon="wrench">
    模型返回 `tool_use`，由**你的代码**执行，再把 `tool_result` 回传给模型。需要多轮交互。
  </Card>

  <Card title="服务端工具（Server Tools）" icon="server">
    模型在 MiniMax **服务端**自动执行工具并获取结果，**一次请求**内完成，你只需读取最终回复。
  </Card>
</CardGroup>

## 支持范围

| 能力   | 支持状态                                                   |
| :--- | :----------------------------------------------------- |
| 接口   | 仅 **Anthropic Messages API**（`/anthropic/v1/messages`） |
| 可用工具 | `web_search`（联网搜索）                                     |
| 调用方式 | 在请求的 `tools` 数组中声明服务端工具                                |

## web\_search

`web_search` 让模型在生成回复时自动进行联网搜索，获取实时信息，并基于搜索结果作答。适用于时效性强、需要最新事实的问题（如新闻、行情、文档查询等）。

### 声明工具

在 `tools` 中加入一个 `type` 为 `web_search_20250305` 的工具即可：

```json theme={null}
{
  "type": "web_search_20250305",
  "name": "web_search"
}
```

<Info>
  `web_search_20250305` 是该服务端工具的**版本化类型标识**，沿用 Anthropic 官方命名约定：`web_search` 为工具名，后缀 `20250305`（即 2025-03-05）标识工具的发布版本。当工具能力发生变化时，Anthropic 会以新的日期后缀发布新版本，请以此处声明的类型为准。
</Info>

### 调用示例

<CodeGroup>
  ```bash cURL theme={null}
  curl https://api.minimaxi.com/anthropic/v1/messages \
    -H "Content-Type: application/json" \
    -H "x-api-key: ${YOUR_API_KEY}" \
    -H "anthropic-version: 2023-06-01" \
    -d '{
      "model": "MiniMax-M3",
      "max_tokens": 8192,
      "messages": [
          {
              "role": "user",
              "content": "帮我搜一下今天上海的天气"
          }
      ],
      "tools": [
          {
              "type": "web_search_20250305",
              "name": "web_search"
          }
      ]
  }'
  ```
</CodeGroup>

### 响应说明

启用 `web_search` 后，模型会在服务端自动执行搜索并将结果用于生成最终回复。你无需处理 `tool_use` / `tool_result` 的多轮回传，直接读取 `message.content` 中的 `text` 内容块即可获取答案。

`message.content` 会按模型的执行顺序返回多个内容块，一次完整的搜索问答通常包含以下类型：

| 内容块类型                    | 说明                                                                                     |
| :----------------------- | :------------------------------------------------------------------------------------- |
| `text`                   | 模型生成的文本。搜索前的引导语与搜索后的最终答案都属于此类型                                                         |
| `server_tool_use`        | 模型在服务端发起的工具调用，`name` 为 `web_search`，`input.query` 为实际检索的关键词                            |
| `web_search_tool_result` | 服务端返回的搜索结果，`content` 为 `web_search_result` 列表，含 `title`、`url`、`page_age`、`content` 等字段 |

<Accordion title="完整响应示例">
  ```json theme={null}
  {
      "id": "069d492820d3562155e88b67fe988b42",
      "type": "message",
      "role": "assistant",
      "model": "MiniMax-M3",
      "content": [
          {
              "text": "我来帮您搜索今天上海的天气信息。",
              "type": "text"
          },
          {
              "type": "server_tool_use",
              "id": "call_function_aa733q961ql2_1",
              "name": "web_search",
              "input": {
                  "query": "今天上海天气"
              }
          },
          {
              "type": "web_search_tool_result",
              "tool_use_id": "call_function_aa733q961ql2_1",
              "content": [
                  /// ...
                  {
                      "type": "web_search_result",
                      "title": "上海天气预报",
                      "url": "http://www.weather.com.cn/textFC/shanghai.shtml",
                      "page_age": "2026-07-07 18:00:00",
                      "content": "地图版国内城市天气预报 天气预报 >国内> 上海 上海 今天周二(7月7日) 周三(7月8日) 周四(7月9日) 周五(7月10日) 周六(7月11日) 周日(7月12日) 周一(7月13日) 市 区/县 周二(7月7日)白天 周二(7月7日)夜间 天气现象 风向风力 最高气温 天气现象 风向风力 最低气温 上海 上海 - - - - 小雨 南风 <3级 28 详情 闵行 - - - - 小雨 "
                  },
                  /// ...
              ]
          },
          {
              "text": "根据最新搜索到的信息，今天（7月8日）上海的天气情况如下：\n\n## 🌤️ 上海今日天气\n\n- **天气**：多云到阴为主，局部地区有**短时阵雨或雷雨**\n- **气温**：早间最低气温约 30.3℃，白天最高气温可达 **35~36℃**\n- **风力**：南到西南风 3~4 级（沿江沿海地区 4~5 级），夜里转南到东南风\n- **湿度**：相对湿度 95%~60%，**体感闷热**\n- **预警**：上海市气象台已发布**中心城区高温黄色预警**\n\n## ⚠️ 温馨提示\n\n1. **防暑降温**：气温高、湿度大，闷热感明显，外出注意防晒、多补水。\n2. **防雨防雷**：局部有短时雷阵雨，建议随身带把伞，遮阳又挡雨。\n3. **台风消息**：今年第 14 号台风\"**巴威**\"强度超强，预计 11-13 日将对上海及周边海域产生明显的风雨影响，建议持续关注后续预报。\n\n## 📅 未来几天\n\n- **明天（7月9日）**：多云到阴，局部短时雷雨，气温 28~34℃，略有缓解\n- **周五起**：受台风\"巴威\"外流环流影响，多阵雨天气，气温下降并伴东南大风\n\n如需更详细的逐小时预报，可以访问 [中国天气网-上海](http://www.weather.com.cn/weather/101020100.shtml) 查询。",
              "type": "text"
          }
      ],
      "usage": {
          "input_tokens": 3206,
          "output_tokens": 365,
          "cache_creation_input_tokens": 0,
          "cache_read_input_tokens": 228,
          "service_tier": "standard"
      },
      "stop_reason": "end_turn",
      "base_resp": {
          "status_code": 0,
          "status_msg": ""
      }
  }
  ```
</Accordion>

<Tip>
  搜索行为完全在服务端完成，因此单次请求的耗时可能比不启用工具时更长，请合理设置客户端超时时间。
</Tip>

## 注意事项

1. 服务端工具处于 **Beta** 阶段，行为与参数可能随时调整。
2. 目前仅 **Anthropic Messages API** 支持服务端工具，且仅提供 `web_search` 一个工具。
3. 使用前请确认将 `ANTHROPIC_BASE_URL` 配置为 `https://api.minimaxi.com/anthropic`，详见 [Anthropic SDK](/docs/api-reference/text-anthropic-api) 文档。

如在使用过程中遇到问题，可通过邮箱 [Model@minimaxi.com](mailto:Model@minimaxi.com) 联系我们的技术支持团队。
