# Codex Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support Codex. Refer to this guide for configuration and usage.

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

MiMo now supports the Responses API. Users who are using the Chat Completions API can follow this document to update.

</div>

## Prerequisites

### Obtain Credentials

{/* feishu-style:text-align:left */}
Two usage methods are supported, but the corresponding credential acquisition methods differ:

<table>
<colgroup>
<col style="width: 157px" />
<col style="width: 191px" />
<col style="width: 608px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: center;">Usage Method</span></th>
<th><span style="display: inline-block; text-align: center;">Description</span></th>
<th><span style="display: inline-block; text-align: center;">Acquisition Method (BASE_URL and API Key below are examples)</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: center;">Pay-as-you-go API</span></td>
<td><span style="display: inline-block; text-align: left;">Charged based on actual usage, suitable for light use</span></td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://api.xiaomimimo.com/v1`</li></ul></li><li>API Key<ul><li>Format: `sk-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">Go to [API Keys](https://platform.xiaomimimo.com/#/console/api-keys) to create an API Key</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Token Plan</span></td>
<td><span style="display: inline-block; text-align: left;">Fixed subscription fee, with limited calls based on the package</span></td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://token-plan-cn.xiaomimimo.com/v1`</li></ul></li><li>API Key<ul><li>Format: `tp-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">After successful subscription, go to [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage) to obtain the exclusive Base URL and API Key</span></td>
</tr>
</tbody>
</table>

## Install Codex

{/* feishu-style:text-align:left */}
**Prerequisites:**  Node.js 18 or a later version must be installed first.

{/* feishu-style:text-align:left */}
**Installation command:** 

```bash
npm install -g @openai/codex
```

{/* feishu-style:text-align:left */}
**Verify the installation (a version number output indicates success):** 

```bash
codex --version
```

## Edit Configuration File

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

**Notes**

- When configuring basic information, first check if the `MIMO_API_KEY` environment variable exists. If it does, please clear it or replace the value with the API Key obtained through the corresponding usage method.

- To switch models directly via the `/model` command, refer to the "Further Reading - Custom MiMo Model Configuration" section and configure the `model-catalogs.json` file.

</div>

{/* feishu-style:text-align:left */}
Configuration file paths:

- macOS/Linux: `~/.codex/config.toml`

- Windows: `User directory\.codex\config.toml`

- Below are the complete configuration examples. The `model` field can be changed to other supported models (e.g., `mimo-v2.5`) as needed.

### Pay-as-you-go

{/* feishu-style:text-align:left */}
**1. Edit or create the configuration file** `config.toml`

```bash
model = "mimo-v2.5-pro"
model_provider = "mimo"
model_reasoning_effort = "high"

# Enable model reasoning summaries; if set to false, model_reasoning_effort will not take effect even if configured
model_supports_reasoning_summaries = true
model_reasoning_summary = "none"
model_context_window = 1048576

web_search = "disabled"

[model_providers.mimo]
name = "mimo"
base_url = "https://api.xiaomimimo.com/v1"
env_key = "MIMO_API_KEY"
wire_api = "responses"
```

{/* feishu-style:text-align:left */}
**2. Configure the environment variable** `MIMO_API_KEY`

{/* feishu-style:text-align:left */}
Pay-as-you-go uses an API Key starting with `sk-`.

- macOS/Linux

   ```bash
   echo 'export MIMO_API_KEY="sk-your-api-key-here"' >> ~/.bashrc
   source ~/.bashrc
   ```

- Windows (CMD)

   ```bash
   # Run the following command in CMD
   setx MIMO_API_KEY "sk-your-api-key-here"
   
   # After success, open a new command prompt and run the following to verify the variable is set.
   echo %MIMO_API_KEY%
   ```

### Token Plan

{/* feishu-style:text-align:left */}
**1. Edit or create the configuration file** `config.toml`

```bash
model = "mimo-v2.5-pro"
model_provider = "mimo"
model_reasoning_effort = "high"

# Enable model reasoning summaries; if set to false, model_reasoning_effort will not take effect even if configured
model_supports_reasoning_summaries = true
model_reasoning_summary = "none"
model_context_window = 1048576

web_search = "disabled"

[model_providers.mimo]
name = "mimo"
base_url = "https://token-plan-cn.xiaomimimo.com/v1"
env_key = "MIMO_API_KEY"
wire_api = "responses"
```

{/* feishu-style:text-align:left */}
**2. Configure the environment variable** `MIMO_API_KEY`

{/* feishu-style:text-align:left */}
Token Plan uses an API Key starting with `tp-`.

- macOS/Linux

   ```bash
   echo 'export MIMO_API_KEY="tp-your-api-key-here"' >> ~/.bashrc
   source ~/.bashrc
   ```

- Windows (CMD)

   ```bash
   # Run the following command in CMD
   setx MIMO_API_KEY "tp-your-api-key-here"
   
   # After success, open a new command prompt and run the following to verify the variable is set.
   echo %MIMO_API_KEY%
   ```

## Use Codex CLI

{/* feishu-style:text-align:left */}
After completing the above configuration, open a new terminal and run the following command to start Codex.

```bash
codex
```

## Use Codex IDE Plugin

{/* feishu-style:text-align:left */}
Codex provides a VS Code plugin. Search for "Codex" in the VS Code Extension Marketplace to install it. The plugin automatically reuses the existing local Codex configuration. If you have never used the Codex command-line tool, please follow the specifications in the "Edit Configuration File" section to complete the configuration.

<img src="https://mimo.mi.com/static/NaLobyEGloR0MuxkU2HcQlU8nEg.0f0413942ff26459.png" alt="图片" style="margin: 16px auto;" />

## Further Reading

### Custom MiMo Model Configuration

{/* feishu-style:text-align:left */}
Codex supports custom model parameter configuration, allowing fine-grained and personalized adaptation for MiMo models. After configuration, type `/model` in the Codex CLI to see MiMo models and their corresponding reasoning levels in the model list, and switch between them at any time.

#### Key Field Descriptions

<table>
<colgroup>
<col style="width: 298px" />
<col style="width: 942px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: left;">Field</span></th>
<th><span style="display: inline-block; text-align: left;">Description</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">`slug`</span></td>
<td><span style="display: inline-block; text-align: left;">Unique internal identifier for the model; must exactly match the backend API model name</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`display_name`</span></td>
<td><span style="display: inline-block; text-align: left;">Model name displayed in the frontend UI; can be the same as slug or a custom display name</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`description`</span></td>
<td><span style="display: inline-block; text-align: left;">Model description text, used for hover tooltips in the model list and model detail views</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`default_reasoning_level`</span></td>
<td><span style="display: inline-block; text-align: left;">Default reasoning intensity level for new sessions. For MiMo models, `none` disables thinking; other values enable thinking</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`supported_reasoning_levels`</span></td>
<td><span style="display: inline-block; text-align: left;">List of reasoning intensity configurations available for users to switch between, containing effort identifiers and description text. `none` and `high` are provided as examples</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`shell_type`</span></td>
<td><span style="display: inline-block; text-align: left;">Tool execution environment type. The example uses `"shell_command"`, indicating support for Shell command tools</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`visibility`</span></td>
<td><span style="display: inline-block; text-align: left;">Model visibility strategy in the client UI; `list` means normal display</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`supported_in_api`</span></td>
<td><span style="display: inline-block; text-align: left;">Whether the model is open for standard API calls; `true` means external programs can call it via API</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`priority`</span></td>
<td><span style="display: inline-block; text-align: left;">UI sorting weight; smaller numbers appear earlier, `1` represents high priority pinned at the top</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`base_instructions`</span></td>
<td><span style="display: inline-block; text-align: left;">System prompt</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`supports_reasoning_summaries`</span></td>
<td><span style="display: inline-block; text-align: left;">Whether the model supports reasoning summary output. When `true`, requests carry reasoning parameters; when `false`, all reasoning fields are cleared and the reasoning level (`reasoning_level`) configuration becomes ineffective</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`default_reasoning_summary`</span></td>
<td><span style="display: inline-block; text-align: left;">Default reasoning summary output mode; `none` means no separate reasoning summary is output</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`support_verbosity`</span></td>
<td><span style="display: inline-block; text-align: left;">Whether output verbosity control is supported; `false` means text verbosity level switching is not yet supported</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`truncation_policy`</span></td>
<td><span style="display: inline-block; text-align: left;">Context truncation strategy; mode is the truncation unit, limit is the maximum byte limit per context</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`supports_parallel_tool_calls`</span></td>
<td><span style="display: inline-block; text-align: left;">Whether parallel multi-tool calls are supported; `false` means only sequential calls are supported</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`supports_image_detail_original`</span></td>
<td><span style="display: inline-block; text-align: left;">Whether original high-resolution image parsing is supported for image input; `true` means using the full original resolution</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`context_window`</span></td>
<td><span style="display: inline-block; text-align: left;">Model nominal total context window size; `1048576` = 1M context window</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`max_context_window`</span></td>
<td><span style="display: inline-block; text-align: left;">Maximum supported context window limit</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`effective_context_window_percent`</span></td>
<td><span style="display: inline-block; text-align: left;">Actual usable context percentage; `95` means reserving a 5% safety buffer to prevent exceeding the limit</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`experimental_supported_tools`</span></td>
<td><span style="display: inline-block; text-align: left;">Experimental feature tool whitelist; an empty array `[]` means no additional beta tools are currently available</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`input_modalities`</span></td>
<td><span style="display: inline-block; text-align: left;">Supported input modality types; `["text", "image"]` means text + image input is supported</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`supports_search_tool`</span></td>
<td><span style="display: inline-block; text-align: left;">Whether built-in web search tool capability is supported; `false` means web search is not supported</span></td>
</tr>
</tbody>
</table>

#### Configuration Steps

{/* feishu-style:text-align:left */}
1. Edit or create the `.codex/model-catalogs/model-catalogs.json` file for detailed MiMo model configuration (Take `mimo-v2.5-pro` and `mimo-v2.5` as examples)

```json
{
  "models": [
    {
      "slug": "mimo-v2.5-pro",
      "display_name": "mimo-v2.5-pro",
      "description": "MiMo-v2.5-Pro: Trillion-parameter Flagship Agent Foundation",
      "default_reasoning_level": "high",
      "supported_reasoning_levels": [
        {
          "effort": "none",
          "description": "Disable Thinking"
        },
        {
          "effort": "high",
          "description": "Enabled Thinking"
        }
      ],
      "shell_type": "shell_command",
      "visibility": "list",
      "supported_in_api": true,
      "priority": 0,
      "base_instructions": "You are MiMo, an AI assistant developed by Xiaomi. Today's date: {date} {week}. Your knowledge cutoff date is December 2024.",
      "supports_reasoning_summaries": true,
      "default_reasoning_summary": "none",
      "support_verbosity": false,
      "truncation_policy": {
        "mode": "bytes",
        "limit": 10000
      },
      "supports_parallel_tool_calls": false,
      "supports_image_detail_original": false,
      "context_window": 1048576,
      "max_context_window": 1048576,
      "effective_context_window_percent": 95,
      "experimental_supported_tools": [],
      "input_modalities": ["text"],
      "supports_search_tool": false
    },
    {
      "slug": "mimo-v2.5",
      "display_name": "mimo-v2.5",
      "description": "MiMo-V2.5: Native Omni-modal Perception Model",
      "default_reasoning_level": "high",
      "supported_reasoning_levels": [
        {
          "effort": "none",
          "description": "Disable Thinking"
        },
        {
          "effort": "high",
          "description": "Enabled Thinking"
        }
      ],
      "shell_type": "shell_command",
      "visibility": "list",
      "supported_in_api": true,
      "priority": 1,
      "base_instructions": "You are MiMo, an AI assistant developed by Xiaomi. Today's date: {date} {week}. Your knowledge cutoff date is December 2024.",
      "supports_reasoning_summaries": true,
      "default_reasoning_summary": "none",
      "support_verbosity": false,
      "truncation_policy": {
        "mode": "bytes",
        "limit": 10000
      },
      "supports_parallel_tool_calls": false,
      "supports_image_detail_original": true,
      "context_window": 1048576,
      "max_context_window": 1048576,
      "effective_context_window_percent": 95,
      "experimental_supported_tools": [],
      "input_modalities": ["text", "image"],
      "supports_search_tool": false
    }
  ]
}

```

{/* feishu-style:text-align:left */}
2. Add the following configuration at the root level of the `config.toml` file

- macOS/Linux

   ```bash
   model_catalog_json = "~/.codex/model-catalogs/model-catalogs.json"
   ```

- Windows

   ```bash
   model_catalog_json = "User directory/.codex/model-catalogs/model-catalogs.json"
   ```
