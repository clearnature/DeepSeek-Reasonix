# Cline Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support Cline. Refer to this guide for configuration and usage.

## Prerequisites

### Obtain Credentials 

{/* feishu-style:text-align:left */}
Supports two usage methods, but the corresponding credential acquisition methods are different:

## Use Cline CLI

### Install Cline CLI

{/* feishu-style:text-align:left */}
**Prerequisites:** Node.js 20 or later is required (Node.js 22 recommended).

{/* feishu-style:text-align:left */}
**Installation command:** 

```bash
npm install -g cline
```

{/* feishu-style:text-align:left */}
**Verify installation (if a version number is displayed, the installation was successful):** 

```bash
cline --version
```

### Configure Basic Settings

{/* feishu-style:text-align:left */}
Cline CLI uses the `cline auth` command to configure API providers. Run the following command to configure MiMo model:

```bash
cline auth -p openai -k MIMO_API_KEY -b BASE_URL -m mimo-v2.5-pro
```

{/* feishu-style:text-align:left */}
Parameter descriptions:

- `-p openai`: Select OpenAI-compatible provider

- `-k`: Enter the API Key obtained from the corresponding usage method

- `-b`: Enter the BASE_URL obtained from the corresponding usage method

- `-m`: Enter the model ID, e.g. `mimo-v2.5-pro`

For more detailed configuration information, visit the [Cline CLI Official Documentation](https://docs.cline.bot/cline-cli/cli-reference).

{/* feishu-style:text-align:left */}
You can also configure via the interactive wizard by running `cline auth` and following the prompts.

### Use Cline CLI

{/* feishu-style:text-align:left */}
After completing the configuration, open a new terminal and run the following command to start Cline CLI.

> If you prefer the classic terminal interface, select `Exit` and run `cline --tui` to return to the familiar command-line environment.

```bash
cline
```

{/* feishu-style:text-align:left */}
After starting, you can use MiMo models in Cline CLI.

## Use Cline IDE Plugin

### Install Plugin

{/* feishu-style:text-align:left */}
Search for and install the **Cline** plugin in the VS Code Extensions marketplace.

### Configure Basic Settings

{/* feishu-style:text-align:left */}
Open the Cline plugin in VS Code and fill in the following configuration:

- Required settings:

 - **API Provider**: Select `OpenAI Compatible`

 - **Base URL**: Fill in the BASE_URL obtained through the corresponding usage method

 - **API Key**: API Key obtained from the corresponding usage method

 - **Model ID**: Enter the model name `mimo-v2.5-pro`

- Optional settings:

 - Uncheck **Supports Images**

 - Set **Context Window Size** to `1048576`

 - Set **Temperature** to `1.0`, adjustable based on task requirements

{/* feishu-style:text-align:left */}
Other parameters not mentioned can be adjusted as needed.

### Use Cline Plugin

{/* feishu-style:text-align:left */}
After successful configuration, enter your request in the input box, for example to generate code: