# faq/about-apis.md

> 来源: https://platform.minimaxi.com/docs/faq/about-apis.md

# 接口相关

> 本文档解答 MiniMax 开放平台接口使用的核心问题，包括 API Key 获取与管理、资源保障提升方案及声音复刻服务使用条件。帮助开发者高效集成与调用平台API服务。

### 问：如何获取 API Key

**答：** 您可前往[账户管理 > 接口密钥](https://platform.minimaxi.com/user-center/basic-information/interface-key)创建并管理自己的 **按量计费 API Key**。前往[订阅管理 > Token Plan](https://platform.minimaxi.com/user-center/payment/token-plan)查看您的 **订阅 Key**，它用于 Token Plan 订阅套餐和已购积分。请注意，API Key 是您调用接口的重要凭证，请不要与他人共享您的 API Key，或将其暴露在浏览器或其他客户端代码中。

### 问：如何提高速率限制

**答：** MiniMax 开放平台为您提供不同的资源保障方案，可前往[速率限制](/docs/guides/rate-limits)页面查看具体内容。如您需要获得更高的资源保障，您可通过[api@minimaxi.com](mailto:api@minimaxi.com)与我们的商务获得联系。

### 问：语言模型的 TPS（Tokens Per Second）是如何计算的

**答：** TPS 表示模型每秒生成的 token 数量，用于衡量模型的推理输出速度。计算公式为：

$$
\text{TPS} = \frac{\text{输出 token 数量}}{\text{最后一个 token 的生成时间} - \text{第一个 token 的生成时间}}
$$

即从模型输出第一个 token 开始计时，到最后一个 token 输出完成为止，期间生成的 token 总数除以这段时间（秒）。

 TPS 在实际使用中可能存在波动，各模型页面标注的 TPS 为参考值。

### 问：如何才能使用声音复刻服务

**答：** 基于法律法规的要求，如您需要使用声音克隆服务，请先前往[账户管理 > 账户信息](https://platform.minimaxi.com/user-center/basic-information)中的认证信息中，完成**个人实名认证**或者**企业认证**完成认证后，即刻可以通过 [API 调试台 > Voice Cloning](https://platform.minimaxi.com/examination-center/voice-experience-center/voiceCloning) 页面，或者通过快速复刻接口使用声音复刻服务。