# token-plan/mini-agent.md

> 来源: https://platform.minimaxi.com/docs/token-plan/mini-agent.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Mini-Agent

> Mini-Agent 是一个极简但专业的项目，旨在展示使用 MiniMax M3 构建 Agent 的最佳实践。项目通过兼容 Anthropic 的 API，完全支持交错思维链（interleaved thinking），从而解锁模型在处理长而复杂的任务时强大的推理能力。

<Card title="Mini-Agent" icon="github" href="https://github.com/MiniMax-AI/Mini-Agent">
  查看 GitHub 仓库
</Card>

## 核心特性

* **完整的 Agent 执行循环**：一个完整可靠的执行框架，配备了文件系统和 Shell 操作的基础工具集
* **持久化记忆**：通过内置的 Session Note Tool，Agent 能够在多个会话中保留关键信息
* **智能上下文管理**：自动对会话历史进行摘要，可处理长达可配置 Token 上限的上下文，从而支持无限长的任务
* **集成 Claude Skills**：内置 15 种专业技能，涵盖文档处理、设计、测试和开发等领域
* **集成 MCP 工具**：原生支持 MCP 协议，可轻松接入知识图谱、网页搜索等工具
* **全面的日志记录**：为每个请求、响应和工具执行提供详细日志，便于调试
* **简洁明了的设计**：美观的命令行界面和易于理解的代码库，使其成为构建高级 Agent 的理想起点

***

## 使用示例

### 任务执行

要求 Agent 创建一个简洁美观的网页并在浏览器中显示它，展示基础的工具使用循环。
![CreateWeb](https://filecdn.minimax.chat/public/9d284dc8-fff4-4fbc-a232-12efa94e731b.gif)

### 使用 Claude Skill（如 PDF 生成）

Agent 利用 Claude Skill 根据用户请求创建专业文档（如 PDF 或 DOCX），展示了其强大的高级能力。
![Claude Skill](https://filecdn.minimax.chat/public/a0a51586-af59-4349-8010-f734d63eb54b.gif)

### 网页搜索与摘要（MCP 工具）

Agent 使用网页搜索工具在线查找最新信息，并为用户进行总结。
![Web Search](https://filecdn.minimax.chat/public/d3690081-a887-431c-8b7e-e61a0454f5ca.gif)

***

## 快速开始

### 1. 安装 uv

<CodeGroup>
  ```bash macOS/Linux/WSL theme={null}
  curl -LsSf https://astral.sh/uv/install.sh | sh

  # 安装完成后，重启终端或运行：
  source ~/.bashrc  # 或 ~/.zshrc
  ```

  ```powershell Windows theme={null}
  # 安装后需要重启 PowerShell
  powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"
  ```
</CodeGroup>

### 2. 安装 Mini Agent

```bash theme={null}
uv tool install git+https://github.com/MiniMax-AI/Mini-Agent.git
```

### 3. 运行配置脚本

<CodeGroup>
  ```bash macOS/Linux theme={null}
  curl -fsSL https://raw.githubusercontent.com/MiniMax-AI/Mini-Agent/main/scripts/setup-config.sh | bash
  ```

  ```powershell Windows theme={null}
  $r=Invoke-WebRequest -Uri "https://raw.githubusercontent.com/MiniMax-AI/Mini-Agent/main/scripts/setup-config.ps1" -UseBasicParsing;[IO.File]::WriteAllText("$env:TEMP\setup-config.ps1",$r.Content,(New-Object Text.UTF8Encoding $true));powershell -ExecutionPolicy Bypass -File "$env:TEMP\setup-config.ps1"
  ```
</CodeGroup>

### 4. 配置 API Key

配置脚本会在 `~/.mini-agent/config/` 目录下创建配置文件，请编辑该文件：

```bash theme={null}
nano ~/.mini-agent/config/config.yaml
```

填入您的 API Key 和对应的 API Base：

```yaml theme={null}
api_key: "YOUR_API_KEY_HERE"          # 填入您的 API Key
api_base: "https://api.minimaxi.com"  # 国内版
# api_base: "https://api.minimax.io"  # 海外版（如使用海外平台，请取消本行注释）
model: "MiniMax-M3"
```

### 5. 开始使用

```bash theme={null}
mini-agent                                    # 使用当前目录作为工作空间
mini-agent --workspace /path/to/your/project  # 指定工作空间目录
mini-agent --version                          # 查看版本信息

# 管理命令
uv tool upgrade mini-agent                    # 升级到最新版本
uv tool uninstall mini-agent                  # 卸载工具
uv tool list                                  # 查看所有已安装的工具
```

***

## 开发模式

此模式适合需要修改代码、添加功能或进行调试的开发者。

**安装与配置步骤：**

```bash theme={null}
# 1. 克隆仓库
git clone https://github.com/MiniMax-AI/Mini-Agent.git
cd Mini-Agent

# 2. 同步依赖
uv sync

# 3. 初始化 Claude Skills（可选）
git submodule update --init --recursive

# 4. 复制配置模板
cp mini_agent/config/config-example.yaml mini_agent/config/config.yaml

# 5. 编辑配置文件
vim mini_agent/config/config.yaml
```

填入您的 API Key 和对应的 API Base：

```yaml theme={null}
api_key: "YOUR_API_KEY_HERE"          # 填入您的 API Key
api_base: "https://api.minimaxi.com"  # 国内版
# api_base: "https://api.minimax.io"  # 海外版
model: "MiniMax-M3"
max_steps: 100
workspace_dir: "./workspace"
```

**运行方式：**

```bash theme={null}
# 方式 1：作为模块直接运行（适合调试）
uv run python -m mini_agent.cli

# 方式 2：以可编辑模式安装（推荐）
uv tool install -e .
mini-agent
mini-agent --workspace /path/to/your/project
```

<Note>
  更多开发指引，请参阅 [开发指南](https://github.com/MiniMax-AI/Mini-Agent/blob/main/docs/DEVELOPMENT_GUIDE_CN.md)
</Note>

***

## ACP & Zed Editor 集成

Mini Agent 支持 Agent Communication Protocol（ACP），可与 Zed 等代码编辑器集成。

**在 Zed Editor 中设置：**

1. 以开发模式或工具模式安装 Mini Agent
2. 在您的 Zed `settings.json` 中添加：

```json theme={null}
{
  "agent_servers": {
    "mini-agent": {
      "command": "/path/to/mini-agent-acp"
    }
  }
}
```

**命令路径：**

* 通过 `uv tool install` 安装：使用 `which mini-agent-acp` 的输出结果
* 开发模式：`./mini_agent/acp/server.py`

**使用方法：**

* 使用 `Ctrl+Shift+P` → "Agent: Toggle Panel" 打开 Zed 的 Agent 面板
* 从 Agent 下拉列表中选择 "mini-agent"
* 直接在编辑器中开始与 Mini Agent 对话
