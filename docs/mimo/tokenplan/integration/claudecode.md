# Claude Code Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support Claude Code. Refer to this guide for configuration and usage.

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
<td><ul><li>BASE_URL<ul><li>Anthropic Compatibility Protocol: `https://api.xiaomimimo.com/anthropic`</li></ul></li><li>API Key<ul><li>Format: `sk-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">Go to [API Keys](https://platform.xiaomimimo.com/#/console/api-keys) to create an API Key</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Token Plan</span></td>
<td><span style="display: inline-block; text-align: left;">Fixed subscription fee, with limited calls based on the package</span></td>
<td><ul><li>BASE_URL<ul><li>Anthropic Compatibility Protocol: `https://token-plan-cn.xiaomimimo.com/anthropic`</li></ul></li><li>API Key<ul><li>Format: `tp-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">After successful subscription, go to [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage) to obtain the exclusive Base URL and API Key </span></td>
</tr>
</tbody>
</table>

## Use Claude Code CLI

### Install Claude Code CLI

{/* feishu-style:text-align:left */}
Claude Code requires Node.js 18 or later.

- Linux/macOS: No additional setup needed, the default environment is sufficient.

- Windows: Install [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) or [Git for Windows](https://git-scm.com/install/windows), then run the command below in WSL or Git Bash.

{/* feishu-style:text-align:left */}
**Installation command:** 

```bash
npm install -g @anthropic-ai/claude-code
```

{/* feishu-style:text-align:left */}
**Verify the installation (a version number output indicates success):** 

```bash
claude --version
```

### Configure Basic Settings

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

Before configuring, make sure to clear the following Anthropic official environment variables to avoid API conflicts: `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`

</div>

{/* feishu-style:text-align:left */}
**1.**  **Create/edit** `settings.json`

> If the `.claude` directory does not exist, you can create it manually.

- macOS/Linux: `~/.claude/settings.json`

- Windows: `User directory\.claude\settings.json`

{/* feishu-style:text-align:left */}
Please replace `BASE_URL` (Anthropic Compatibility Protocol) and `MIMO_API_KEY` as needed.

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

For MiMo models that support **1M** context, you can append the `[1m]` suffix to the model ID to enable extended context capacity. Example: `mimo-v2.5-pro[1m]`. After configuration, restart Claude Code and run the `/context` command to verify whether the long context takes effect.

</div>

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "BASE_URL",
    "ANTHROPIC_AUTH_TOKEN": "MIMO_API_KEY",
    "ANTHROPIC_MODEL": "mimo-v2.5-pro",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "mimo-v2.5-pro",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "mimo-v2.5-pro",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "mimo-v2.5-pro"
  }
}
```

{/* feishu-style:text-align:left */}
**2.**  **Create/edit** `.claude.json`

- macOS/Linux: `~/.claude.json`

- Windows: `User directory\.claude.json`

   ```json
   {
     "hasCompletedOnboarding": true
   }
   ```

{/* feishu-style:text-align:left */}
**3.**  **Apply the configuration**

{/* feishu-style:text-align:left */}
After completing the configuration, **reopen the terminal window** for the changes to take effect.

### Use Claude Code CLI

{/* feishu-style:text-align:left */}
Navigate to your project directory and run:

```bash
claude
```

{/* feishu-style:text-align:left */}
On first launch, complete the following: select "**Trust This Folder**" to allow Claude Code to access project files. After startup, use the `/status` command to verify the current configuration and model status.

## Use the Claude Code IDE Plugin

{/* feishu-style:text-align:left */}
Claude Code provides a VS Code IDE plugin. For configuration reference, see the official documentation [Use Claude Code in VS Code](https://code.claude.com/docs/en/vs-code#vs-code-extension-vs-claude-code-cli).

### Install Plugin

{/* feishu-style:text-align:left */}
Search for and install the **Claude Code for VS Code** plugin from the VS Code Extensions marketplace.

<img src="https://mimo.mi.com/static/Cpd0bPIpdoHSNdxFu4Yc34W9n52.56a4227d1565e05a.png" alt="图片" style="margin: 16px auto;" />

### Configure the Model

{/* feishu-style:text-align:left */}
Open VS Code settings, search for `Claude Code: Environment Variables`, and then manually configure it in `settings.json`:

```json
{
  "claudeCode.preferredLocation": "panel",
  "claudeCode.selectedModel": "mimo-v2.5-pro",
  "claudeCode.environmentVariables": [
    {
      "name": "ANTHROPIC_BASE_URL",
      "value": "BASE_URL"
    },
    {
      "name": "ANTHROPIC_AUTH_TOKEN",
      "value": "MIMO_API_KEY"
    },
    {
      "name": "ANTHROPIC_DEFAULT_SONNET_MODEL",
      "value": "mimo-v2.5-pro"
    },
    {
      "name": "ANTHROPIC_DEFAULT_OPUS_MODEL",
      "value": "mimo-v2.5-pro"
    },
    {
      "name": "ANTHROPIC_DEFAULT_HAIKU_MODEL",
      "value": "mimo-v2.5-pro"
    }
  ]
}
```

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

If Claude Code CLI is already installed, the VS Code plugin will automatically reuse the CLI configuration. To configure independently, specify the environment variables in the plugin settings as shown above.

</div>

## FAQ

### Installation fails on Windows? 

{/* feishu-style:text-align:left */}
Ensure the following dependencies are installed:

- Node.js 18+

- Git for Windows

{/* feishu-style:text-align:left */}
If you encounter permission issues with npm, try running the terminal as administrator, or use nvm to manage Node.js versions.
