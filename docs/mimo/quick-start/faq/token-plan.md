# Token Plan

### Packages and Prices

- **What packages are available for the Token Plan?** 

{/* feishu-style:text-align:left */}
Currently, four packages are offered: Lite, Standard, Pro, and Max, along with two subscription cycles: continuous monthly and continuous annual. Each package includes different usage quotas and special privileges.

- **Which models does Token Plan support?** 

{/* feishu-style:text-align:left */}
Supports a total of 6 models from the MiMo-V2.5 series, and can be used with all-tier packages.

> mimo-v2.5-pro：Flagship reasoning model
>
> mimo-v2.5：Omnipotent MultiModal Machine Learning Model
>
> mimo-v2.5-asr: Speech Recognition Model
>
> mimo-v2.5-tts/ v2.5-tts-voiceclone / v2.5-tts-voicedesign：Speech synthesis model (free for a limited time)

- **How much is the set meal? Are there any discounts?** 

{/* feishu-style:text-align:left */}
The specific prices of the four packages shall be subject to the display on the landing page. The platform is currently offering the following time-limited promotional activities:
- First Purchase Discount: Enjoy 12% off on your first purchase, available only once per account.
- Continuous annual subscription: Enjoy an 12% discount compared to continuous monthly subscription; the first purchase/first activation auto-renewal discount does not apply to annual subscriptions.
- Nighttime discount rate: During off-peak hours (0:00-8:00 Beijing Time, i.e., 16:00-24:00 UTC), the consumption coefficient is 0.8x.

- **Does it support continuous subscription?** 

{/* feishu-style:text-align:left */}
Support. It supports two subscription cycles: continuous monthly and continuous annual, with automatic renewal after expiration. You can cancel automatic renewal at any time on the Token Plan page. Continuous annual subscription enjoys an 12% discount, offering higher savings compared to continuous monthly subscription.

- **Can I purchase multiple packages or upgrade a package?** 

{/* feishu-style:text-align:left */}
Currently, the platform only supports purchasing 1 package at a time. If you wish to obtain more credits before the package expires, you can convert the used Credit amount into an equivalent amount, and then top up the price difference on this basis to upgrade to a higher package and obtain more Credits. Cross-level package upgrades by topping up the price difference are supported, while package downgrades are not. If you have already upgraded to the highest-tier Max package, further upgrades are not possible. After the package expires, you can purchase a package of any tier again.

> Price difference = New package price - (Remaining amount of the original package / Total amount of the original package) * Original package price

{/* feishu-style:text-align:left */}
<br></br>

### Validity Period and Expiration

- **How long is the package valid after purchase?** 

{/* feishu-style:text-align:left */}
It takes effect immediately after purchase, and the validity period of the package is "the day of purchase + 30 complete natural days (as of 23:59:59 UTC)". Effective immediately upon purchase, valid for one calendar month/year starting from the date of purchase.

{/* feishu-style:text-align:left */}
For example, if you subscribe to a monthly plan on March 28, the plan will expire at 23:59:59 (UTC) on April 28.

- **Will the package automatically renew after it expires?** 

{/* feishu-style:text-align:left */}
It depends on whether you have enabled auto-renewal. If auto-renewal is enabled, the system will automatically deduct payment via Alipay, WeChat Pay or Xiaomi Pay (domestic users), or via Waffo / Stripe (overseas users) on the package expiration date. After successful deduction, it will automatically enter the next subscription cycle without manual operation; if auto-renewal is not enabled, the service will stop after the package expires, and manual re-subscription is required.

- **Can I still use it if the quota is used up but not expired?** 

{/* feishu-style:text-align:left */}
No. The service will stop when either the "expiration" or "all Credits used up" condition is met. **The system will not continue to consume your bonus or account balance.**   We support the package upgrade feature. Regardless of your package consumption, we support automatically converting your remaining Credits into an equivalent amount, and you can upgrade your package by paying the price difference to obtain more Credits.  **If you need to continue using it, please upgrade your package or switch to the pay-as-you-go API.**  

- **If the package has expired but there are still unused amounts, can it still be carried forward?**  

- **Will I receive a reminder when the package expires or is almost used up?** 

{/* feishu-style:text-align:left */}
Yes. If auto-renewal is enabled for your package, you will receive renewal reminders via SMS, email, and in-app notifications from payment apps (WeChat Pay / Alipay) five days before the expiration date; if your plan does not have auto-renewal enabled, you will receive reminders via SMS and email two days before and on the expiration date.

- **Will there be a reminder when the package quota is almost used up?** 

{/* feishu-style:text-align:left */}
Yes. When your current plan usage reaches 50%, 90%, and 100%, you will receive text message and email reminders.

{/* feishu-style:text-align:left */}
<br></br>

### Usage and Quota

- Usage and Quota

- **Are the quotas of different models consumed independently?** 

{/* feishu-style:text-align:left */}
No.  The available models in the package are consumed in parallel according to different  proportions, not independently. TTS  all models  are free for a limited time and do not consume package credits.  ASR models deduct credits based on the duration of the input audio (duration is counted accurately to the second and finally converted to hours for statistics) The following table lists the package deduction points corresponding to each token for cache, input, and output of each model **.**  

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

**Note:** `mimo-v2-pro`, `mimo-v2-omni` and `mimo-v2-tts` **have been officially deprecated on June 30 00:00, 2026. Please switch to the new models as soon as possible.**

</div>

{/* feishu-style:text-align:left */}
Language Model

<table>
<colgroup>
<col style="width: 136px" />
<col style="width: 200px" />
<col style="width: 219px" />
<col style="width: 147px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: left;">Model</span></th>
<th><span style="display: inline-block; text-align: left;">Input (Cache Hit) Token </span></th>
<th><span style="display: inline-block; text-align: left;">Input (Cache Miss) Token </span></th>
<th><span style="display: inline-block; text-align: left;">Output Token </span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">mimo-v2.5-pro</span></td>
<td><span style="display: inline-block; text-align: left;">2.5 Credits</span></td>
<td><span style="display: inline-block; text-align: left;">300 Credits</span></td>
<td><span style="display: inline-block; text-align: left;">600 Credits</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">mimo-v2.5</span></td>
<td><span style="display: inline-block; text-align: left;">2 Credits</span></td>
<td><span style="display: inline-block; text-align: left;">100 Credits</span></td>
<td><span style="display: inline-block; text-align: left;">200 Credits</span></td>
</tr>
</tbody>
</table>

{/* feishu-style:text-align:left */}
ASR Model

<table>
<colgroup>
<col style="width: 136px" />
<col style="width: 200px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: left;">Model</span></th>
<th><span style="display: inline-block; text-align: left;">Input audio duration (h)</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">mimo-v2.5-asr</span></td>
<td><span style="display: inline-block; text-align: left;">30M Credits</span></td>
</tr>
</tbody>
</table>

{/* feishu-style:text-align:left */}
TTS Series models are free for a limited time and do not consume package credits.

{/* feishu-style:text-align:left */}
For example, if you have subscribed to the Lite Plan (4.1B Credits), you can call the MiMo-V2.5 series models individually or in combination. After you have used 10M input (cache miss) tokens of mimo-v2.5-pro, it is equivalent to consuming 3000M Credits.You can still enjoy 1100M Credits of mimo-v2.5.If you subscribe to the Lite plan and only use the ASR model, you can use 4100M ÷ 30M/hour = 136.6 hours per month (equivalent to processing 4.5 hours of audio per day for 1 consecutive month).  You can check the quota and usage of your current plan in [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage). 

- **What is the 0.8x coefficient for off-peak periods?** 

{/* feishu-style:text-align:left */}
To balance resource pressure and provide benefits to users, when using the model during off-peak hours (0:00-8:00 Beijing Time, i.e., 16:00-24:00 UTC), the credit consumption coefficient is 0.8 times.

{/* feishu-style:text-align:left */}
For example, in a scenario where you use the mimo-v2.5-pro model and consume 10M Credits during peak hours, it will only consume 8M during off-peak hours.

- **When using Token to calculate usage, I'm worried that Credit will be consumed very quickly. What should I do?** 

   - We recommend that you check your past token usage in each AI Agent framework before making a purchase, and choose a suitable package based on your experience. 

   - We have designed a Progress Bar system, and you can view the progress and plan ahead in [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage).

   - We support the package upgrade feature. Regardless of your package consumption status, we support automatically converting your remaining Credits into an equivalent amount. You can upgrade your package by paying the price difference to obtain more Credits. 

- **What is the remaining value of a package?** 

{/* feishu-style:text-align:left */}
When you renew or upgrade from a currently unused/expired package, the system calculates the equivalent value based on the Credits consumption of the current package, and this remaining value will be used to offset a portion of the payment amount for the new package.

- **Why are there still compensation Credits on my Token Plan page?** 

{/* feishu-style:text-align:left */}
During your renewal for the current package, since the remaining value of the previous package is higher than the value of the current package, the platform compensates you with Credits equivalent to the difference in value.
