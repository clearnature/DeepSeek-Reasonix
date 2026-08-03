# Cherry Studio Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support Cherry Studio. Refer to this guide for configuration and usage.

## Prerequisites

### Obtain Credentials 

{/* feishu-style:text-align:left */}
Supports two usage methods, but the corresponding credential acquisition methods are different:

## Install Cherry Studio

{/* feishu-style:text-align:left */}
Cherry Studio is a desktop AI client that supports multi-model conversations.

- Official website: https://www.cherry-ai.com

- Github: https://github.com/CherryHQ/cherry-studio

## Configure Basic Settings

{/* feishu-style:text-align:left */}
**1.** **Find provider** `Xiaomi MiMo`

{/* feishu-style:text-align:left */}
Click the settings icon in the upper right corner, go to the Model Services page, and search for `Xiaomi MiMo` in the search box.

{/* feishu-style:text-align:left */}
**2.** **Configure basic settings**

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API**

{/* feishu-style:text-align:left */}
Since the `Xiaomi MiMo` model service is already provided by Cherry Studio officially, you only need to provide the API Key obtained through this method. Keep the API Host unchanged.

{/* feishu-style:text-align:left */}
**Token Plan**

{/* feishu-style:text-align:left */}
After successfully subscribing to Token Plan, replace with the dedicated Token Plan API Key and API Host (BASE_URL).

Token Plan is temporarily unavailable in Agent mode.

## Use Cherry Studio

{/* feishu-style:text-align:left */}
Select the model you need to use from the model list, and you can have a normal conversation.

### Enable Thinking Mode (Optional)

{/* feishu-style:text-align:left */}
Click assistant settings and add a custom parameter: `"thinking": {"type": "enabled"}`.

{/* feishu-style:text-align:left */}
You can also adjust temperature, context window, and other parameters as needed.