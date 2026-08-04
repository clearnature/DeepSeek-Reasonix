# #7234 数据链全程追踪（dev 分支，与 #7168/#4711/#7357 交叉锚点）

## 环节1 配置层：preset → provider entry → vendor 能力表

```
provider_presets.go:  deepseek-responses(kind=responses, api.deepseek.com)
                     [mimo-api 仍 kind=openai——responses 绑定未配置（配置层缺口，与 #7357 同层）]
vendor.go:           capabilitiesFor(baseURL) → {stateless, summaryRequired, toolCallReasoning,
                     ignoresTemperature, defaultMaxOutputTokens, singleSegmentReasoning}
DetectVendor:        api.xiaomimimo.com→mimo / dashscope.aliyuncs.com→dashscope /
                     api.deepseek.com→deepseek / 默认 OpenAI 行为
问题锚点：          #7357（配置合并缺陷——用户 provider 继承默认，同层同源）
                     #7273（effort 硬编码覆盖第三方代理——同一"官方判定"模式）
```

## 环节2 请求层：buildRequestBody（vendor 分支）

```
buildRequestBody:
  max_output_tokens → caps.defaultMaxOutputTokens（mimo 65536 / deepseek 32K / 表驱动）
  temperature/top_p → caps.ignoresTemperature 跳过（mimo 思考模式强制 1.0/0.95）
  reasoning.effort → mimo 支持 none（effort.go isMimoEntry）
  text.format=json_object → ResponseFormat 支持（结构化输出）
问题锚点：          MiMo 截断（da3aadd34 预算提升）；#7273（effort 校验）
```

## 环节3 wire 层：messagesToInput（string vs array + summary 隔离）

```
messagesToInput(messages, vision, summary):
  纯文本 → content 为 string（OpenAI base；用户明确：array 是图像形态）
  含图片 + vision → content 为 array（input_text + input_image×N）
  reasoning item → summary 仅 caps.summaryRequired（dashscope=true）发送
                    [MiMo 无 summary 字段：发送会折回上下文膨胀——b0c1d8cb5 修复]
  tool message name → *string 始终序列化（#4711：空 name 也发，MiMo 严格校验）
问题锚点：          b0c1d8cb5（summary 膨胀截断根因）；#4711（name 400）
```

## 环节4 流式层：readStream（reasoning meta + 完成语义）

```
readStream:
  response.output_item.added/done → 捕获 reasoning id/status
  response.completed → terminal 后发射 meta chunk（ReasoningID/Status）
  FinishReason=stop 合成（#7168 语义；缺失时补）
  zero-usage 抑制（TotalTokens==0 && stop → 不发 ChunkUsage，防计费污染）
问题锚点：          #7168 评审第 3 点；#6259（缺 thinking 降级——服务端偶发）
```

## 环节5 会话层：Chunk meta → Agent.stream → run_loop → 回传

```
provider.Chunk.ReasoningID/Status → Agent.stream 收集（返回值扩展 11 值）
run_loop 持久化进 session message → 下一轮 messagesToInput 回传 id/status
broker 路径（Host↔Desktop）：BrokerProviderChunk 双向转换 + 生成产物
问题锚点：          #7234 评审第 1 点（reasoning 贯通）；#7168 评审第 2 点（重放）
```

## 环节6 工具层：web_search（检索系统依赖此管道）

```
WebSearchTool(false) → tool_choice=web_search → knowledge_extract json_schema
responses.Retrieve（本地缓存 L1/L2 → 服务端 web_search → 蒸馏落盘）
问题锚点：          #6259（工具轮缺 thinking 时 web_search 同样受影响）
                     检索系统（本地 dev 独有）是 responses 管道的最大消费者
```
