# MiMo Code Configuration

{/* feishu-style:text-align:left */}
[MiMo Code](https://mimo.xiaomi.com/zh/mimocode) is an AI-powered coding assistant developed by Xiaomi, available as a CLI tool in the terminal. Both **pay-as-you-go MiMo API** and **Token Plan** are supported by MiMo Code. Refer to this guide for configuration and usage.

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

**Limited-Time Offer**

After completing authorization for MiMo Code, each user is entitled to **1000 free** web search queries per day. Once the quota is exceeded, please activate the relevant service on the open platform and ensure your account has sufficient balance.

</div>

## Prerequisites

{/* feishu-style:text-align:left */}
MiMo Code supports direct redirection to the [Xiaomi MiMo API Platform](https://platform.xiaomimimo.com/docs/en-US/welcome) for authorization login, or you can log in using the authorization code returned by the platform. No manual API Key configuration is required. The system provides two key types for you to choose from based on your usage scenario.

> Note: Before authorizing and logging into MiMo Code, please ensure your account on the open platform has sufficient balance or a valid Token Plan. Otherwise, the authorization will fail.

<table>
<colgroup>
<col style="width: 152px" />
<col style="width: 242px" />
<col style="width: 659px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: center;">Usage</span></th>
<th><span style="display: inline-block; text-align: center;">Description</span></th>
<th><span style="display: inline-block; text-align: center;">How to Obtain and Manage</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: center;">Pay-as-you-go API</span></td>
<td><span style="display: inline-block; text-align: left;">Billed by actual usage, suitable for light usage</span></td>
<td><span style="display: inline-block; text-align: left;">After authorization, the platform will automatically create a new API Key prefixed with `mimo-code-cli-key`. You can view and manage it on the [API Keys](https://platform.xiaomimimo.com/#/console/api-keys) page.</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: center;">Token Plan</span></td>
<td><span style="display: inline-block; text-align: left;">Fixed subscription fee with limited calls per plan</span></td>
<td><span style="display: inline-block; text-align: left;">You can view the current Token Plan quota, usage, and expiration on the [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage) page.</span></td>
</tr>
</tbody>
</table>

## Install MiMo Code

{/* feishu-style:text-align:left */}
MiMo Code supports two installation methods.

{/* feishu-style:text-align:left */}
**Method 1: Official Script Installation (for macOS/Linux)** 

> For a better user experience, Mac users are strongly recommended to use iTerm or VSCode Terminal.

```bash
curl -fsSL https://mimo.xiaomi.com/install | bash
```

{/* feishu-style:text-align:left */}
**Method 2: npm Installation (for Windows)**  

{/* feishu-style:text-align:left */}
Requires Node.js 18 or later.

```bash
npm install -g @mimo-ai/cli
```

{/* feishu-style:text-align:left */}
**Verify Installation (a version number output indicates successful installation):** 

```bash
mimo --version
```

## Connect Provider

{/* feishu-style:text-align:left */}
You can connect to the Xiaomi MiMo provider in two ways:

{/* feishu-style:text-align:left */}
**1. MiMo Code Already Running**

{/* feishu-style:text-align:left */}
Run the `/connect` or `/login` command in the interactive interface, and select `Xiaomi` as the provider.

{/* feishu-style:text-align:left */}
**2. MiMo Code Not Yet Running**

{/* feishu-style:text-align:left */}
Run the following command directly in the terminal and select `MiMo` to complete authorization.

```bash
mimo auth login
```

{/* feishu-style:text-align:left */}
After confirmation, the MiMo authorization login popup will automatically appear. Follow the prompts to complete the login, and then select your preferred authorization key type based on your usage scenario:

<img src="./images/OaltbhLKMo8siQx06ZZc1pKnnYe.png" alt="图片" style="margin: 16px auto;" />

## Use MiMo Code

### Quick Start

{/* feishu-style:text-align:left */}
Follow these steps to use MiMo Code in your project:

```bash
# 1. Navigate to your project directory
cd /path/to/your/project

# 2. Launch MiMo Code
mimo

# 3. (Recommended) Initialize project configuration on first use
/init
```

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

It is strongly recommended to run the `/init` command on first use:

- It automatically analyzes your project structure and coding conventions

- Generates an `AGENTS.md` file in the project root directory

- MiMo Code will use this file to better understand your project context, improving interaction quality

</div>

{/* feishu-style:text-align:left */}
For more commands and detailed usage, please refer to the [MiMo Code Official Documentation](https://mimo.xiaomi.com/mimocode/interaction).

<img src="./images/Xe4WbBAjmoAjowxcPyNcdtnznfd.png" alt="图片" style="margin: 16px auto;" />

### Model Selection

{/* feishu-style:text-align:left */}
Run the `/models` command to view and select from the currently available models.

## FAQ

### What should I do if I encounter the following error when verifying the installation on Windows? 

> It seems that your package manager failed to install the right version of the mimocode CLI for your platform. You can try manually installing "@mimo-ai/mimocode-windows-x64" or "@mimo-ai/mimocode-windows-x64-baseline" package

{/* feishu-style:text-align:left */}
Answer: Run the command `npm install -g @mimo-ai/mimocode-windows-x64` as indicated to resolve the issue.

### Why can't I see the model's reasoning content?

{/* feishu-style:text-align:left */}
Answer: MiMo Code does not display the model's reasoning content by default. You can use the `/thinking` command to toggle the visibility of reasoning blocks in the conversation. Once enabled, you can view the complete reasoning process of models that support extended thinking.

> Note: This command is not a toggle for the model's thinking function. It cannot enable or disable the model's thinking process.

### Why does authorization fail?

{/* feishu-style:text-align:left */}
Answer: Please check if your account on the open platform has sufficient balance or a valid Token Plan.
