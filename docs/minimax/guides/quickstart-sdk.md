# guides/quickstart-sdk.md

> 来源: https://platform.minimaxi.com/docs/guides/quickstart-sdk.md

# 通过 SDK 接入

> 使用 Anthropic SDK 快速接入 MiniMax API，开始调用 MiniMax-M3 模型。

 ```bash Python theme={null}
 pip install anthropic
 ```

 ```bash Node.js theme={null}
 npm install @anthropic-ai/sdk
 ```

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

 ```json theme={null}
 {
 "thinking": "The user is just greeting me casually. I should respond in a friendly, professional manner.",
 "text": "Hi there! I'm doing well, thanks for asking. I'm ready to help you with whatever you need today—whether it's coding, answering questions, brainstorming ideas, or just chatting. What can I do for you?"
 }
 ```

## 探索更多

 探索 MiniMax 最新语言模型

 使用 Hailuo 2.3 创建图生视频任务

 使用 Hailuo 2.3 创建文生视频任务

 使用 Speech 2.8 进行同步语音合成

 使用 Speech 2.8 进行异步语音合成

 创建音色复刻任务

 使用 Music 3.0 进行音乐创作