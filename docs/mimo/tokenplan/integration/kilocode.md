# Kilo Code Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support Kilo Code. Refer to this guide for configuration and usage.

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

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

When Kilo Code uses MiMo under the Anthropic protocol, since the assistant containing tool calls is missing `reasoning_content`, the API will return a 400 error. For details, see [Multi-turn Conversation Pass-through Requirements](https://mimo.mi.com/docs/en-US/quick-start/usage-guide/text-generation/deep-thinking#:~:text=.-,Multi%2Dturn%20Conversation%20Pass%2Dthrough%20Requirements,-When%20deep%20thinking).

</div>

## Use Kilo Code CLI

### Install Kilo Code CLI

{/* feishu-style:text-align:left */}
Node.js 18 or later is required.

{/* feishu-style:text-align:left */}
**Installation command:** 

```bash
npm install -g @kilocode/cli
```

{/* feishu-style:text-align:left */}
**Verify installation (success if version number is displayed):** 

```bash
kilocode --version
```

### Configure Basic Settings

{/* feishu-style:text-align:left */}
Edit or create the `config.json` configuration file at the following paths:

- **macOS/Linux**: `~/.config/kilo/config.json`

- **Windows**: `User Directory\.config\kilo\config.json`

{/* feishu-style:text-align:left */}
Copy the following content into the configuration file (replace `BASE_URL` and `MIMO_API_KEY` as needed):

```bash
{
  "$schema": "https://kilo.ai/config.json",
  "disabled_providers": [],
  "provider": {
    "mimo": {
      "name": "MiMo",
      "npm": "@ai-sdk/openai-compatible",
      "models": {
        "mimo-v2.5-pro": {
          "name": "mimo-v2.5-pro",
          "options": {
            "thinking": {
              "type": "enabled"
            }
          }
        }
      },
      "options": {
        "apiKey": "MIMO_API_KEY",
        "baseURL": "BASE_URL"
      }
    }
  },
  "permission": {
    "bash": "allow"
  }
}
```

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

For more detailed configuration information, visit the [Kilo Code CLI Official Documentation](https://kilo.ai/docs/cli).

</div>

### Use Kilo Code CLI

{/* feishu-style:text-align:left */}
After completing the above configuration, open a new terminal and run the following command to start Kilo Code CLI:

```bash
kilocode
```

{/* feishu-style:text-align:left */}
Once started, enter `/models` to switch models, and you can use MiMo models in Kilo Code CLI.

## Use Kilo Code IDE Plugin

### Install Plugin

{/* feishu-style:text-align:left */}
Search for and install the **Kilo Code** plugin in the VS Code Extensions marketplace.

<img src="https://mimo.mi.com/static/LwoAbbDLOo5YtextdBGcnmwUnpd.21832d886c9eef6d.png" alt="图片" style="margin: 16px auto;" />

### Configure a Predefined Provider (Recommended)

{/* feishu-style:text-align:left */}
Click Providers --> Show more providers, search for `Xiaomi`, select the corresponding Provider, and fill in the API Key.

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

When using the **Xiaomi Token Plan**, you need to select the Provider corresponding to the Base URL displayed on the [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage) page.

- `https://token-plan-cn.xiaomimimo.com/*`: Xiaomi Token Plan (China)

- `https://token-plan-sgp.xiaomimimo.com/*`: Xiaomi Token Plan (Singapore)

- `https://token-plan-ams.xiaomimimo.com/*`: Xiaomi Token Plan (Europe)

</div>

<img src="https://mimo.mi.com/static/FHKXbyRaMoX4C0x3jiEcjuKInVh.5c1ed74fff419710.png" alt="图片" style="margin: 16px auto;" />

### Configure a Custom Provider

{/* feishu-style:text-align:left */}
Fill in the relevant information according to the following configuration.

{/* feishu-style:text-align:left */}
**1.**  **Select Custom Provider**

<img src="https://mimo.mi.com/static/EVQHbk7Zlo8XnQxNXT0c4VZAneh.2f0c877959e06668.png" alt="图片" style="margin: 16px auto;" />

{/* feishu-style:text-align:left */}
**2.**  **Fill in configuration details**

- **Provider ID** and **Display name**: Fill in as needed

- **Base URL**: Enter the BASE_URL obtained from your usage method

- **API Key**: Enter the API Key obtained from your usage method

- **Models**: Add as needed, e.g. `mimo-v2.5-pro`

<img src="https://mimo.mi.com/static/DdUBbS2OHoBJb7xQuHZcF8zmngc.8af0bda1aa20f54c.png" alt="图片" style="margin: 16px auto;" />

{/* feishu-style:text-align:left */}
Other unmentioned parameters can be adjusted as needed.

### Use Kilo Code Plugin

{/* feishu-style:text-align:left */}
After successful configuration, switch to the configured model and enter your requirements in the input box to start using.

<img src="https://mimo.mi.com/static/Se1ZbimQhonqsoxJYEecFOY5npB.00013cebc9fa9f98.png" alt="图片" style="margin: 16px auto;" />

## FAQ

### When verifying installation on Windows, I encounter the following error. How to resolve? 

> It seems that your package manager failed to install the right version of the Kilo CLI for your platform. You can try manually installing "@kilocode/cli-windows-x64" or "@kilocode/cli-windows-x64-baseline" package

{/* feishu-style:text-align:left */}
Run the command `npm install -g @kilocode/cli-windows-x64` as suggested to resolve the issue.
