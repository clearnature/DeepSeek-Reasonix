# token-plan/intro.md

> 来源: https://platform.minimaxi.com/docs/token-plan/intro.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Token Plan 概要

> Token Plan 的订阅和使用概要

<img src="https://filecdn.minimax.chat/public/token-plan-hero-v4.png" alt="Token Plan" style={{ display: "block", margin: "0 auto", maxWidth: "100%", borderRadius: "12px" }} />

## 欢迎使用 Token Plan！

MiniMax 在语言、视频、语音、音乐和图像等方向研发模型。[Token Plan](https://platform.minimaxi.com/subscribe/token-plan) 通过订阅 Key 提供套餐内 Token Plan 用量额度，覆盖语言模型之外的更多资源。

## 核心优势

<CardGroup cols={3}>
  <Card title="全模态覆盖" icon="layers">
    一个订阅通过统一的用量进度条覆盖可用的 MiniMax 资源。
  </Card>

  <Card title="面向 Agent 工作流" icon="zap">
    套餐面向长上下文 Agent、编程和多模态工作流设计。
  </Card>

  <Card title="极具性价比" icon="piggy-bank">
    固定订阅费提供更广的资源覆盖，并让用量规则更清晰。
  </Card>
</CardGroup>

## 订阅 Key

每位用户在所属的每个团队中都有一把专属的 **订阅 Key**。这把 Key 可以在团队尚未购买 Token Plan 席位或积分时就存在。如果用户当前没有可用资源，这把 Key 暂时没有可用的付费资源。当用户被分配 Token Plan 席位，或获得积分使用权限后，同一把订阅 Key 即可使用这些资源。

订阅和积分规则请参考 [Token Plan 定价](/docs/guides/pricing-token-plan)。团队版请参考 [Token Plan 团队版](/docs/guides/pricing-token-plan-team)。

## 用量额度

[Token Plan](https://platform.minimaxi.com/subscribe/token-plan) 的用量额度在控制台以用量进度条展示。对于已有按量计费价格的 API 端点，用量会按对应按量计费价格扣减套餐内 Token Plan 额度。

|              | **Plus**     | **Max**           | **Ultra**           |
| :----------- | :----------- | :---------------- | :------------------ |
| **价格**       | **¥49 /月**   | **¥119 /月**       | **¥469 /月**         |
| **适合场景**     | 轻量个人开发与日常试用  | 高频编程 Agent 与多模态调用 | 重度 Agent 工作流与更长时间使用 |
| **额度窗口**     | 5 小时固定窗口和周窗口 | 5 小时固定窗口和周窗口      | 5 小时固定窗口和周窗口        |
| **Agent 用量** | 3-4 个 Agent  | 4-5 个 Agent       | 6-7 个 Agent         |

<div style={{ fontSize: "11px", color: "#6b7280", lineHeight: 1.35, marginTop: "-16px", marginBottom: "12px" }}>可用模型覆盖 MiniMax 全系模型（M3 / M2.7 / 图像 / 语音 / 音乐），少量特殊模型（MiniMax H3、音色设计、快速复刻等）暂不支持。</div>

<Tip>
  如需通过订阅 Key 调用支持的多模态资源，请参考 [MiniMax CLI 指南](https://platform.minimaxi.com/docs/token-plan/minimax-cli)。
</Tip>

## 快速指南

<Steps>
  <Step title="订阅或获得资源分配">
    访问 [Token Plan](https://platform.minimaxi.com/subscribe/token-plan) 订阅页面，在默认团队中购买个人订阅或积分；也可以加入团队，并使用团队分配的 Token Plan 席位或共享积分。
  </Step>

  <Step title="获取订阅 Key">
    前往 [账户管理 / Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan) 页面，查看您的可用资源并获取 **订阅 Key**。
  </Step>
</Steps>

<Info>
  **重要提示**

  * 订阅 Key 用于 Token Plan 订阅套餐和已购积分。
  * 订阅 Key 与按量计费 API Key 不可互换。
  * 订阅 Key 可以在付费资源可用之前就存在；当用户拥有 Token Plan 席位或积分权限后才可实际使用资源。
  * 请妥善保管您的 API Key，防止资源损失。
</Info>

## 在 AI Agent 与编程工具中使用

挑选你常用的工具，按对应教程接入：

<CardGroup cols={3}>
  <Card title="OpenClaw" icon="bot" href="/docs/token-plan/openclaw" />

  <Card title="Claude Code" icon="terminal" href="/docs/token-plan/claude-code" />

  <Card title="Cursor" icon="square-code" href="/docs/token-plan/cursor" />

  <Card title="TRAE" icon="code" href="/docs/token-plan/trae" />

  <Card title="Hermes Agent" icon="sparkles" href="/docs/token-plan/hermes-agent" />
</CardGroup>

其他工具的接入方式见 [其他工具](/docs/token-plan/other-tools)。

## 达到用量上限后

当您达到 5 小时固定窗口额度或周窗口额度后，可选择以下方式：

1. **使用已购积分**：
   如果已购积分可用，Token Plan 资源覆盖范围内的用量可由已购积分自动补充支付。
2. **升级或获得新的分配**：
   升级订阅，或请团队 Owner / Admin 分配更高额度的可用套餐。
3. **切换至按量计费**：
   如需继续使用且不受限制，您可将订阅 Key 替换为您的[按量计费 API Key](https://platform.minimaxi.com/user-center/basic-information/interface-key)，切换至按实际 Token 用量计费模式，费用将从您的 API 账户余额中扣除。
4. **等待额度窗口重置**：
   套餐内 Token Plan 额度按 5 小时固定窗口和周窗口控制；未使用完的订阅额度不会结转到下一个计费周期。

## 下一步

<CardGroup cols={3}>
  <Card title="快速开始" icon="rocket" href="/docs/token-plan/quickstart">
    用 5 分钟跑通你的第一次 MiniMax API 调用。
  </Card>

  <Card title="常见问题" icon="info" href="/docs/token-plan/faq">
    用量、计费、切换、退款等高频问题集合。
  </Card>

  <Card title="活动 & 优惠" icon="gift" href="/docs/token-plan/promotion">
    查看当前进行中的优惠活动。
  </Card>
</CardGroup>
