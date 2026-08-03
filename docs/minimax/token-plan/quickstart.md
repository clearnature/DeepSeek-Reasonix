# token-plan/quickstart.md

> 来源: https://platform.minimaxi.com/docs/token-plan/quickstart.md

# 快速接入

> 快速了解 Token Plan 订阅及接入

## 开始使用

 访问 [订阅管理 > Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan) 查看您的 **订阅 Key**。

 重要提示：

 * 订阅 Key 用于 Token Plan 订阅套餐和已购积分。
 * 订阅 Key 与按量计费 API Key 不互通。
 * 这把 Key 可以在您尚未拥有付费资源时就存在；当您拥有 Token Plan 席位或积分权限后才可实际使用资源。
 * 请妥善保存您的 API Key ，建议将其导出为环境变量或保存到配置文件

 在默认团队中购买个人 Plus、Max 或 Ultra Token Plan 订阅，或购买积分套餐；也可以使用团队 Owner / Admin 分配给您的资源。

 通过 Claude SDK 快速测试 **MiniMax M3**

 **1. 安装 Claude SDK**

 ```bash Python theme={null}
 pip install anthropic
 ```

 ```bash Node.js theme={null}
 npm install @anthropic-ai/sdk
 ```

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

 您可参考以下内容选择您常用的 AI 编程工具，体验最新 **MiniMax M 系列**模型能力

## 接入 MCP

快速接入 **Token Plan MCP**，获取 **网络搜索** 能力

 了解如何配置和使用 Token Plan MCP

## 了解更多

 查看订阅和积分规则。

 查看用量、计费、切换和退款等高频问题。

## 最佳实践

快速查看 MiniMax Token Plan 模型的 Prompt 模板、工具调用和长上下文实践

 掌握 Token Plan 模型的 Prompt 模板、工具调用和长上下文工作流

 使用 M 系列模型构建 Agent