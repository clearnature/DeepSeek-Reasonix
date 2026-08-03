# guides/quickstart-sdk.md

> 来源: https://platform.minimaxi.com/docs/guides/quickstart-sdk.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# 通过 SDK 接入

> 使用 Anthropic SDK 快速接入 MiniMax API，开始调用 MiniMax-M3 模型。

<Steps>
  <Step title="安装 Anthropic SDK（推荐）">
    <CodeGroup>
      ```bash Python theme={null}
      pip install anthropic
      ```

      ```bash Node.js theme={null}
      npm install @anthropic-ai/sdk
      ```
    </CodeGroup>
  </Step>

  <Step title="调用 MiniMax-M3">
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

  <Step title="示例输出">
    ```json theme={null}
    {
      "thinking": "The user is just greeting me casually. I should respond in a friendly, professional manner.",
      "text": "Hi there! I'm doing well, thanks for asking. I'm ready to help you with whatever you need today—whether it's coding, answering questions, brainstorming ideas, or just chatting. What can I do for you?"
    }
    ```
  </Step>
</Steps>

## 探索更多

<Columns cols={3}>
  <Card title="语言模型" icon="book-open" href="/docs/guides/text-generation" cta="点击查看">
    探索 MiniMax 最新语言模型
  </Card>

  <Card title="图生视频" icon="video" href="/docs/guides/video-generation-i2v-refer" cta="点击查看">
    使用 Hailuo 2.3 创建图生视频任务
  </Card>

  <Card title="文生视频" icon="video" href="/docs/guides/video-generation-t2v-refer" cta="点击查看">
    使用 Hailuo 2.3 创建文生视频任务
  </Card>

  <Card title="同步语音合成" icon="mic" href="/docs/guides/speech-t2a-websocket" cta="点击查看">
    使用 Speech 2.8 进行同步语音合成
  </Card>

  <Card title="异步语音合成" icon="mic" href="/docs/guides/speech-t2a-async" cta="点击查看">
    使用 Speech 2.8 进行异步语音合成
  </Card>

  <Card title="音色复刻" icon="mic" href="/docs/guides/speech-voice-clone" cta="点击查看">
    创建音色复刻任务
  </Card>

  <Card title="音乐生成" icon="music" href="/docs/guides/music-generation" cta="点击查看">
    使用 Music 3.0 进行音乐创作
  </Card>
</Columns>
