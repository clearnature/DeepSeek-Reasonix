# token-plan/quickstart.md

> 来源: https://platform.minimaxi.com/docs/token-plan/quickstart.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# 快速接入

> 快速了解 Token Plan 订阅及接入

## 开始使用

<Steps>
  <Step title="获取订阅 Key">
    访问 [订阅管理 > Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan) 查看您的 **订阅 Key**。

    <Callout icon="album" color="#A8A8A8" iconType="regular">
      重要提示：

      * 订阅 Key 用于 Token Plan 订阅套餐和已购积分。
      * 订阅 Key 与按量计费 API Key 不互通。
      * 这把 Key 可以在您尚未拥有付费资源时就存在；当您拥有 Token Plan 席位或积分权限后才可实际使用资源。
      * 请妥善保存您的 API Key ，建议将其导出为环境变量或保存到配置文件
    </Callout>
  </Step>

  <Step title="获得资源">
    在默认团队中购买个人 Plus、Max 或 Ultra Token Plan 订阅，或购买积分套餐；也可以使用团队 Owner / Admin 分配给您的资源。
  </Step>

  <Step title="测试 API 调用（可选）">
    通过 Claude SDK 快速测试 **MiniMax M3**

    **1. 安装 Claude SDK**

    <CodeGroup>
      ```bash Python theme={null}
      pip install anthropic
      ```

      ```bash Node.js theme={null}
      npm install @anthropic-ai/sdk
      ```
    </CodeGroup>

    **2. 配置环境变量**

    ```bash theme={null}
    export ANTHROPIC_BASE_URL=https://api.minimaxi.com/anthropic
    export ANTHROPIC_API_KEY=${YOUR_API_KEY}
    ```

    **3. 调用 API**

    ```python Python theme={null}
    import anthropic

    client = anthropic.Anthropic()

    message = client.messages.create(
        model="MiniMax-M3",
        max_tokens=1000,
        system="You are a helpful assistant.",
        messages=[
            {
                "role": "user",
                "content": [
                    {
                        "type": "text",
                        "text": "Hi, how are you?"
                    }
                ]
            }
        ]
    )

    for block in message.content:
        if block.type == "thinking":
            print(f"Thinking:\n{block.thinking}\n")
        elif block.type == "text":
            print(f"Text:\n{block.text}\n")
    ```
  </Step>

  <Step title="接入 AI 编程工具">
    您可参考以下内容选择您常用的 AI 编程工具，体验最新 **MiniMax M 系列**模型能力

    <Columns cols={3}>
      <Card title="Claude Code" icon="sparkles" href="/docs/token-plan/claude-code" arrow="true" />

      <Card title="Cursor" icon="mouse-pointer" href="/docs/token-plan/cursor" arrow="true" />

      <Card title="Trae" icon="layers" href="/docs/token-plan/trae" arrow="true" />
    </Columns>

    <Columns cols={3}>
      <Card title="OpenCode" icon="folder-open" href="/docs/token-plan/opencode" arrow="true" />

      <Card title="Kilo Code" icon="zap" href="/docs/token-plan/kilo-code" arrow="true" />
    </Columns>

    <Columns cols={3}>
      <Card title="Grok CLI" icon="terminal" href="/docs/token-plan/grok-cli" arrow="true" />

      <Card title="Codex CLI" icon="file-code" href="/docs/token-plan/codex-cli" arrow="true" />
    </Columns>

    <Columns cols={3}>
      <Card title="Droid" icon="bot" href="/docs/token-plan/droid" arrow="true" />
    </Columns>
  </Step>
</Steps>

## 接入 MCP

快速接入 **Token Plan MCP**，获取 **网络搜索** 能力

<Card title="MCP 使用指南" icon="plug" href="/docs/token-plan/mcp-guide" arrow="true">
  了解如何配置和使用 Token Plan MCP
</Card>

## 了解更多

<Columns cols={2}>
  <Card title="Token Plan 定价" icon="coins" href="/docs/guides/pricing-token-plan" arrow="true">
    查看订阅和积分规则。
  </Card>

  <Card title="常见问题" icon="circle-help" href="/docs/token-plan/faq" arrow="true">
    查看用量、计费、切换和退款等高频问题。
  </Card>
</Columns>

## 最佳实践

快速查看 MiniMax Token Plan 模型的 Prompt 模板、工具调用和长上下文实践

<Columns cols={2}>
  <Card title="M 系列模型使用技巧" icon="lightbulb" href="/docs/token-plan/prompting-best-practices" arrow="true">
    掌握 Token Plan 模型的 Prompt 模板、工具调用和长上下文工作流
  </Card>

  <Card title="Mini Agent" icon="bot" href="/docs/token-plan/mini-agent" arrow="true">
    使用 M 系列模型构建 Agent
  </Card>
</Columns>
