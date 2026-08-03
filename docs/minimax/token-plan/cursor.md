# token-plan/cursor.md

> 来源: https://platform.minimaxi.com/docs/token-plan/cursor.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Cursor

> 在 Cursor 中使用最新的 MiniMax M 系列模型进行 AI 编程。

<div style={{background:"#fffbeb",borderLeft:"4px solid #d97706",padding:"12px 16px",borderRadius:"6px",margin:"16px 0"}}>[**Cursor**](https://cursor.com) 是 Anysphere 出品的 AI-first IDE，基于 VS Code 二次开发，内置 Agent 与代码库理解能力。</div>

## 安装 Cursor

1. 通过 [Cursor 官网](https://cursor.com/) 下载并安装 Cursor
2. 打开 Cursor，右上角"设置"按钮，进入设置界面。点击"Sign in"按钮，登录自己的 Cursor 账户

![](https://filecdn.minimax.chat/public/ebf6631f-1d76-432c-b307-59b0b805e5cd.png)

## 配置 MiniMax API

<Warning>
  **重要提示：使用前请先清除 OpenAI 环境变量**

  在配置前，请确保清除以下 OpenAI 相关的环境变量，以免影响 MiniMax API 的正常使用：

  * `OPENAI_API_KEY`
  * `OPENAI_BASE_URL`
</Warning>

<Warning>
  **注意：Cursor 仅支持订阅高级会员及以上的用户配置自定义模型**

  若非Cursor高级会员，配置时将出现以下错误:

  `The model MiniMax-M3 does not work with your current plan or api key`
</Warning>

<Warning>
  **已知问题："Override OpenAI Base URL" 是全局设置**

  开启后会作用到 Cursor 中**所有**已配置的 API Key——包括 Cursor 自带模型用的 Anthropic / GPT Key。Cursor 官方已确认此问题：当你设置 base URL 时，所有 API Key 都会受影响（[社区帖](https://forum.cursor.com/t/anthropic-models-break-when-override-openai-baseurl-is-set/144899)）。

  如果开启 Override 后 Cursor 自带的 Claude / GPT 模型停止工作，请在不使用 MiniMax 时关闭 Override。Cursor 目前不支持按模型设置不同 base URL（[open feature request](https://forum.cursor.com/t/custom-base-urls-for-each-custom-model/147219)）。
</Warning>

1. 点击左侧栏的 **"Models"**，进入模型配置页面
2. 展开 **"API Keys"** 部分，配置 API 信息：
   * 勾选 **"Override OpenAI Base URL"**
   * 在下方输入 MiniMax 的调用地址 `https://api.minimaxi.com/v1`
3. 在 **OpenAI API Key** 输入框，输入从 [MiniMax 开放平台](https://platform.minimaxi.com/user-center/payment/token-plan) (国际用户可访问 [MiniMax Developer Platform](https://platform.minimax.io/user-center/payment/token-plan)) 获取的 **API Key**
4. 点击 **"OpenAI API Key"** 栏右侧的按钮

![](https://filecdn.minimax.chat/public/ebf48187-f00f-48df-a479-d9c17df67f71.jpeg)

5. 在弹出的窗口中点击 **"Enable OpenAI API Key"** 按钮，完成设置验证

![](https://filecdn.minimax.chat/public/67d40b7c-1a18-4553-a0f8-9ceb2dc610a4.png)

6. 在 **Models** 板块中，点击 **"View All Models"** 按钮，并点击 **"Add Custom Model"** 按钮

![](https://filecdn.minimax.chat/public/0c4bebfb-2859-482e-872b-11d6d361bc63.jpeg)

7. 输入模型名称"MiniMax-M3"后，点击"Add"按钮

<Note>
  模型名称需严格输入 `MiniMax-M3`（注意大小写，与 MiniMax 文档中保持一致）。
</Note>

![](https://filecdn.minimax.chat/public/d2623e83-bf5d-4bf3-9c72-b1993823e10c.png)

8. 启用刚添加的 "MiniMax-M3" 模型
9. 在聊天面板中选择 **"MiniMax-M3"** 模型，开始使用

<Note>
  **Cursor Tab 自动补全无法由 MiniMax 驱动。** Tab 是 Cursor Pro 自带能力，始终使用 Cursor 自家模型，不受你的自定义 API Key 影响。Custom Model 仅在 Chat / Composer / Edit 等模式下生效。
</Note>

<Note>
  如果您出现模型没有返回任何内容的问题，可尝试通过将 Cursor 中的 **"Network"** 设置更改为 **HTTP/1.0** 解决。
</Note>

<img src="https://filecdn.minimax.chat/public/00abf9ab-12b6-4374-9cef-09074d21d23d.png" width="100%" />
