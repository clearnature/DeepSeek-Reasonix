# Token Plan

{/* feishu-style:text-align:left */}
**Token Plan** is a dedicated subscription plan launched for AI programming scenarios. You can use the cost-effective subscription resource package to call the MiMo flagship large model in various mainstream AI development tools.

## Core Advantages

- **Covers the flagship model** - Supports mimo-v2.5-pro, mimo-v2.5, mimo-v2.5-asr, mimo-v2.5-tts-voiceclone, mimo-v2.5-tts-voicedesign, mimo-v2.5-tts. A total of 6 models. Adopts a Token conversion mechanism, with transparent and controllable quotas 

- **Flexible Subscription Plan** - Four-tier Gradient Package, Meeting the Needs from Individual Development to Enterprise-level Development

- **Multi-ecosystem Out Of The Box** - Compatible with mainstream development toolchains such as OpenCode, OpenClaw, Codex,and Claude Code

## Usage Quota

#### Monthly Package

#### Annual Package

#### Applicable Scenarios

> The above is the scenario scope of the monthly package, and the order of magnitude of task processing for the annual package is approximately 12 times that of the monthly package. 

- **Discounts: 0.8x consumption at night, first purchase of a package enjoys 12% discount, consecutive annual subscription enjoys 12% discount, Token Plan existing users exclusively enjoy once the "Credits Usage Refresh and Reset" event after TokenPlan upgrade.** 

 - First Purchase Discount: Enjoy 12% off on your first purchase, available only once per account;

 - Continuous annual subscription: Enjoy an 88% discount compared to continuous monthly subscription; first purchase discounts do not apply to annual subscriptions;

 - Nighttime Discount Rate: Off-peak hours (Beijing Time 0:00-8:00, i.e., UTC 16:00-24:00) with a consumption coefficient of 0.8x.

- **Supported Models:** All packages support **mimo-v2.5-pro、mimo-v2.5、mimo-v2.5-asr、mimo-v2.5-tts-voiceclone、mimo-v2.5-tts-voicedesign、mimo-v2.5-tts** a total of 6 models.

- **Quota Consumption: Language models deduct Credit quota based on the number of Tokens. Available models in the package are consumed in parallel at different ratios, not independently. TTS series models are free for a limited time and do not consume package Credit. ASR models deduct Credit quota based on the duration of the input audio (duration is counted accurately to the second and finally converted to hours for statistics). The following table lists the types of models cache, input, and output for each Token's corresponding package deduction quota .** 

**Note:** `mimo-v2-pro`, `mimo-v2-omni` and `mimo-v2-tts` **have been officially deprecated on June 30 00:00, 2026. Please switch to the new models as soon as possible.**

{/* feishu-style:text-align:left */}
Language Model

{/* feishu-style:text-align:left */}
ASR Model

{/* feishu-style:text-align:left */}
TTS Series models are free for a limited time and do not consume package credits.

{/* feishu-style:text-align:left */}
For example, if you have ordered the Lite Package (4.1B Credits), you can call MiMo-V 2.5 series models individually or in combination mimo-v2.5-pro input (cache miss) tokens, which is equivalent to consuming 3000 M Credits, you can still enjoy 1100M mimo-v2.5 Credits quota.If you subscribe to the Lite plan and only use the ASR model, you can use 4100M ÷ 30M/hour = 136.6 hours per month (equivalent to processing 4.5 hours of audio per day for 1 consecutive month). You can check the quota and usage of your current plan in [ Subscription Management ](https://platform.xiaomimimo.com/#/console/plan-manage). 

- **Quota Exhausted:** When the monthly total quota of the package is exhausted, the system will stop service and will not continue to consume your bonus or account balance. 

- **If you need to continue using:** Please purchase an upgrade package to unlock new package resources; or switch to the regular API, which is billed by the unit price of tokens, and you can continue using it without usage limits. 

## Package Purchase

- **Support for upgrading a package by paying the price difference: Currently, the platform only supports purchasing 1 package at a time. If you wish to obtain more credits before the package expires** , you can convert the used credit amount into an equivalent amount, and on this basis, pay the price difference to upgrade to a higher package and obtain more credits. Support for upgrading packages across levels by paying the price difference, but package downgrading is not supported. If you have already upgraded to the highest Max package, you cannot continue to upgrade. **After the package expires, you can purchase a package of any level again.** 

> Price Difference = New Package Price - (Remaining Amount of Original Package / Total Amount of Original Package) * Original Package Price

- **Supports Auto-renewal**: The auto-renewal feature has been launched, the first time you activate continuous subscription enjoys a discount , please stay tuned. 

- **Refunds are not currently supported**: Please note that once a subscription service is purchased, it becomes effective immediately, and refunds are not supported. Unused credits within the package will not be refunded. Please carefully select a suitable subscription plan based on your own usage needs.

- **Invoice Support**: Domestic users can apply for invoices based on the transaction orders in the recharge details, with the actual invoiceable amount being the actual payment amount. Overseas users can directly download invoices after purchase or download them from the recharge details page.

## Package Usage

{/* feishu-style:text-align:left */}
The Token Plan package quota can only be used in programming tools (such as OpenClaw, OpenCode, etc.), and is prohibited from being used in the form of API calls for request behaviors in clearly non-Coding scenarios such as automated scripts and custom application backends. 

{/* feishu-style:text-align:left */}
If the API Key corresponding to the package is used for calls beyond the permitted scope, it will be considered a violation or abuse, and the platform has the right to take measures such as suspending service and banning the API Key for the relevant subscription.