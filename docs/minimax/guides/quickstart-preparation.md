# guides/quickstart-preparation.md

> 来源: https://platform.minimaxi.com/docs/guides/quickstart-preparation.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# 前置准备

> 在开始使用 MiniMax API 之前，需要完成账户注册和 API Key 获取。

<Steps>
  <Step title="账户注册/登录">
    API 调用前，需在 MiniMax 开放平台进行[账户注册](https://platform.minimaxi.com/login)，企业团队注册请参考页面底部说明.
  </Step>

  <Step title="获取 Key">
    * **按量付费**：通过 [接口密钥 > 创建新的 API Key](https://platform.minimaxi.com/user-center/basic-information/interface-key)，获取 **API Key**
      <Note>按量付费支持使用所有模态模型，包括语言、视频、语音、图像等</Note>
    * **Token Plan**：通过 [订阅管理 > Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan)，查看 **订阅 Key**
      <Note>订阅 Key 用于 Token Plan 订阅套餐和已购积分。它可以在付费资源可用之前就存在；当您拥有 Token Plan 席位或积分权限后才可实际使用资源。详情见 [Token Plan 概要](https://platform.minimaxi.com/docs/token-plan/intro)</Note>

    生成 API Key 后，建议将其存储为环境变量或保存到 `.env` 文件中：

    * 推荐使用 Anthropic API 兼容，具体查看：[Anthropic SDK](/docs/api-reference/text-anthropic-api)

      ```bash theme={null}
      export ANTHROPIC_BASE_URL=https://api.minimaxi.com/anthropic
      export ANTHROPIC_API_KEY=${YOUR_API_KEY}
      ```

    * 使用 OpenAI API 兼容，具体查看：[OpenAI SDK](/docs/api-reference/text-openai-api)

      ```bash theme={null}
      export OPENAI_BASE_URL=https://api.minimaxi.com/v1
      export OPENAI_API_KEY=${YOUR_API_KEY}
      ```

    * 使用 AI SDK 兼容，具体查看：[AI SDK](/docs/api-reference/text-ai-sdk)

      ```bash theme={null}
      export MINIMAX_API_KEY=${YOUR_API_KEY}
      ```
  </Step>

  <Step title="添加资源">
    按量付费可通过 [账户管理 > 余额](https://platform.minimaxi.com/user-center/payment/balance) 按需充值。Token Plan 可购买订阅或积分，也可以使用团队分配给您的资源。
  </Step>
</Steps>

***

<Accordion title="企业团队注册说明">
  建议采用**主账号+子账号**的形式创建和管理。

  1. 在 [MiniMax 开放平台](https://platform.minimaxi.com/user-center/basic-information) 注册一个账号（此账号即为主账号，注册时填写的姓名与手机号会成为本企业账号的管理员信息）
  2. 登录该主账号，在 [账户管理 > 子账号](https://platform.minimaxi.com/user-center/basic-information/child-account)，创建您所需要数量的子账户（子账号的创建数量暂时没有限制）
  3. 为您企业的人员，分配不同的子账户，进行登录使用

  **子账户权限说明：**

  * 子账号和主账号享用相同的使用权益与速率限制，子账号和主账号的 API 消耗共享，统一结算
  * 子账号无查看和管理"支付"权限
</Accordion>
