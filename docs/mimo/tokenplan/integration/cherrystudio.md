# Cherry Studio Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support Cherry Studio. Refer to this guide for configuration and usage.

## Prerequisites

### Obtain Credentials 

{/* feishu-style:text-align:left */}
Supports two usage methods, but the corresponding credential acquisition methods are different:

<table>
<colgroup>
<col style="width: 143px" />
<col style="width: 191px" />
<col style="width: 700px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: center;">Usage Method</span></th>
<th><span style="display: inline-block; text-align: center;">Description</span></th>
<th><span style="display: inline-block; text-align: center;">Acquisition Method (BASE_URL and API Key below are examples) </span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">Pay-as-you-go MiMo API</span></td>
<td><span style="display: inline-block; text-align: left;">Charged based on actual usage, suitable for light use</span></td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://api.xiaomimimo.com/v1`</li></ul></li><li>API Key<ul><li>Format: `sk-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">Go to [API Keys](https://platform.xiaomimimo.com/#/console/api-keys) to create an API Key</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Token Plan</span></td>
<td><span style="display: inline-block; text-align: left;">Fixed subscription fee, with limited calls based on the package</span></td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://token-plan-cn.xiaomimimo.com/v1`</li></ul></li><li>API Key<ul><li>Format: `tp-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">After successful subscription, go to [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage) to obtain the exclusive Base URL and API Key </span></td>
</tr>
</tbody>
</table>

## Install Cherry Studio

{/* feishu-style:text-align:left */}
Cherry Studio is a desktop AI client that supports multi-model conversations.

- Official website: https://www.cherry-ai.com

- Github: https://github.com/CherryHQ/cherry-studio

## Configure Basic Settings

{/* feishu-style:text-align:left */}
**1.**  **Find provider** `Xiaomi MiMo`

{/* feishu-style:text-align:left */}
Click the settings icon in the upper right corner, go to the Model Services page, and search for `Xiaomi MiMo` in the search box.

<img src="https://mimo.mi.com/static/CZhDbutx9o7LeYxOjEoc0RBQnxe.47c68ff9b3c21fe0.png" alt="图片" style="margin: 16px auto;" />

{/* feishu-style:text-align:left */}
**2.**  **Configure basic settings**

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API**

{/* feishu-style:text-align:left */}
Since the `Xiaomi MiMo` model service is already provided by Cherry Studio officially, you only need to provide the API Key obtained through this method. Keep the API Host unchanged.

<img src="https://mimo.mi.com/static/IfjXbN3Tcozm8MxS9XTcDWoDnLf.fbde74dd47e268d3.png" alt="图片" style="margin: 16px auto;" />

{/* feishu-style:text-align:left */}
**Token Plan**

{/* feishu-style:text-align:left */}
After successfully subscribing to Token Plan, replace with the dedicated Token Plan API Key and API Host (BASE_URL).

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

Token Plan is temporarily unavailable in Agent mode.

</div>

## Use Cherry Studio

{/* feishu-style:text-align:left */}
Select the model you need to use from the model list, and you can have a normal conversation.

<img src="https://mimo.mi.com/static/QXSmbNqVlohMGExFuqfcvjrHnwc.61d9aa62e308fdc4.png" alt="图片" style="margin: 16px auto;" />

### Enable Thinking Mode (Optional)

{/* feishu-style:text-align:left */}
Click assistant settings and add a custom parameter: `"thinking": {"type": "enabled"}`.

{/* feishu-style:text-align:left */}
You can also adjust temperature, context window, and other parameters as needed.

<img src="https://mimo.mi.com/static/M7Fpb4Gn4omCO4xCbmTcvzFwnKg.6d085917235f446e.png" alt="图片" style="margin: 16px auto;" />

<img src="https://mimo.mi.com/static/NHoTbq2eoo1EVfxkrFRcMy8wnNe.09dc8179ff6c009e.png" alt="图片" style="margin: 16px auto;" />
