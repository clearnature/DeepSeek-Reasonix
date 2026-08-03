# guides/local-deploy.md

> 来源: https://platform.minimaxi.com/docs/guides/local-deploy.md

# 本地部署指南

> 使用 vLLM、SGLang（Linux GPU）或 MLX（Mac Studio）本地部署 MiniMax-M2.7，包含不同硬件配置的使用指南。

 本指南介绍如何使用三种推理框架本地部署 MiniMax-M2.7：[vLLM](https://docs.vllm.ai/en/stable/) 和 [SGLang](https://github.com/sgl-project/sglang) 适用于 Linux GPU 服务器，[MLX](https://github.com/ml-explore/mlx) 适用于 Apple Silicon Mac Studio。如需使用 MiniMax API 服务，请参阅[语言模型](/docs/guides/text-generation)。

***

## GPU 服务器部署（vLLM / SGLang）

### 环境要求

* **操作系统**：Linux
* **Python**：3.10 – 3.12
* **GPU**：Compute capability 7.0 或更高

#### 推荐配置

| 配置 | 总 KV Cache 容量 |
| :----------------- | :------------- |
| **96 GB x 4** GPU | 最高 40 万 token |
| **144 GB x 8** GPU | 最高 300 万 token |

 以上数值为硬件支持的最大并发缓存总量。模型单序列（Single Sequence）长度上限仍为 **196K** token。

### 使用 vLLM 部署

vLLM 是一个高性能的 LLM 推理与服务库，通过 PagedAttention 高效管理注意力键值缓存，结合连续批处理（Continuous Batching）和前缀缓存等技术实现卓越的服务吞吐量。

#### 安装

```bash theme={null}
uv venv
source .venv/bin/activate
uv pip install vllm --torch-backend=auto
```

#### 启动服务

vLLM 会自动从 Hugging Face 下载并缓存模型。

 ```bash 4 卡部署 theme={null}
 SAFETENSORS_FAST_GPU=1 vllm serve \
 MiniMaxAI/MiniMax-M2.7 --trust-remote-code \
 --tensor-parallel-size 4 \
 --enable-auto-tool-choice --tool-call-parser minimax_m2 \
 --reasoning-parser minimax_m2_append_think
 ```

 ```bash 8 卡部署 theme={null}
 SAFETENSORS_FAST_GPU=1 vllm serve \
 MiniMaxAI/MiniMax-M2.7 --trust-remote-code \
 --enable_expert_parallel --tensor-parallel-size 8 \
 --enable-auto-tool-choice --tool-call-parser minimax_m2 \
 --reasoning-parser minimax_m2_append_think
 ```

### 使用 SGLang 部署

SGLang 是一个高性能的大模型推理服务框架，利用 RadixAttention 实现高效的前缀缓存和调度，具备低延迟、高吞吐的特点。

#### 安装

```bash theme={null}
uv venv
source .venv/bin/activate
uv pip install sglang
```

#### 启动服务

SGLang 会自动从 Hugging Face 下载并缓存模型。

 ```bash 4 卡部署 theme={null}
 python -m sglang.launch_server \
 --model-path MiniMaxAI/MiniMax-M2.7 \
 --tp-size 4 \
 --tool-call-parser minimax-m2 \
 --reasoning-parser minimax-append-think \
 --host 0.0.0.0 \
 --trust-remote-code \
 --port 8000 \
 --mem-fraction-static 0.85
 ```

 ```bash 8 卡部署 theme={null}
 python -m sglang.launch_server \
 --model-path MiniMaxAI/MiniMax-M2.7 \
 --tp-size 8 \
 --ep-size 8 \
 --tool-call-parser minimax-m2 \
 --reasoning-parser minimax-append-think \
 --host 0.0.0.0 \
 --trust-remote-code \
 --port 8000 \
 --mem-fraction-static 0.85
 ```

### 验证部署

vLLM 和 SGLang 均提供 OpenAI 兼容接口。启动后，可以通过 curl 或 OpenAI SDK 调用：

 ```bash curl theme={null}
 curl http://localhost:8000/v1/chat/completions \
 -H "Content-Type: application/json" \
 -d '{
 "model": "MiniMaxAI/MiniMax-M2.7",
 "messages": [
 {"role": "system", "content": [{"type": "text", "text": "You are a helpful assistant."}]},
 {"role": "user", "content": [{"type": "text", "text": "Who won the world series in 2020?"}]}
 ]
 }'
 ```

 ```python Python theme={null}
 from openai import OpenAI

 client = OpenAI(
 base_url="http://localhost:8000/v1",
 api_key="not-needed",
 )

 response = client.chat.completions.create(
 model="MiniMaxAI/MiniMax-M2.7",
 messages=[
 {"role": "system", "content": "You are a helpful assistant."},
 {"role": "user", "content": "Who won the world series in 2020?"},
 ],
 temperature=1.0,
 top_p=0.95,
 max_tokens=8192,
 )

 print(response.choices[0].message.content.strip())
 ```

***

## Mac Studio 部署（MLX）

得益于 Apple Silicon 的统一内存架构，Mac Studio 可以将模型完整加载到内存中并在设备端完成推理 — 无需 GPU 集群，数据完全私有。

### 环境要求

* **Mac Studio**，搭载 Apple Silicon（M4 Max 或 M3 Ultra）
* **macOS 15.0**（Sequoia）或更高版本
* **Python 3.10+**
* **足够的统一内存** — 参见下方[模型选择指南](#模型选择指南)

### 模型选择指南

所有 MLX 模型变体均可从 Hugging Face 上的 [mlx-community](https://huggingface.co/mlx-community/models?search=minimax-m2.7) 获取。更高位数的量化保留更多模型质量，但需要更多内存。

#### 可用模型变体

| 模型变体 | 量化方式 | 模型大小 | 最低内存 |
| :-------------------------------------------------------------------------------------- | :------------ | :----- | :----- |
| [MiniMax-M2.7](https://huggingface.co/mlx-community/MiniMax-M2.7) | BF16（全精度） | 457 GB | 512 GB |
| [MiniMax-M2.7-8bit-gs32](https://huggingface.co/mlx-community/MiniMax-M2.7-8bit-gs32) | 8-bit（组大小 32） | 257 GB | 288 GB |
| [MiniMax-M2.7-8bit](https://huggingface.co/mlx-community/MiniMax-M2.7-8bit) | 8-bit | 243 GB | 256 GB |
| [MiniMax-M2.7-6bit](https://huggingface.co/mlx-community/MiniMax-M2.7-6bit) | 6-bit | 186 GB | 200 GB |
| [MiniMax-M2.7-5bit](https://huggingface.co/mlx-community/MiniMax-M2.7-5bit) | 5-bit | 157 GB | 172 GB |
| [MiniMax-M2.7-4bit](https://huggingface.co/mlx-community/MiniMax-M2.7-4bit) | 4-bit | 129 GB | 140 GB |
| [MiniMax-M2.7-nvfp4](https://huggingface.co/mlx-community/MiniMax-M2.7-nvfp4) | NVFP4 | 129 GB | 140 GB |
| [MiniMax-M2.7-4bit-mxfp4](https://huggingface.co/mlx-community/MiniMax-M2.7-4bit-mxfp4) | 4-bit MXFP4 | 122 GB | 136 GB |
| [MiniMax-M2.7-3bit](https://huggingface.co/mlx-community/MiniMax-M2.7-3bit) | 3-bit | 100 GB | 112 GB |

 "最低内存" = 模型大小 + 约 10–15 GB 余量（用于 KV 缓存、激活值和系统开销）。实际内存使用量随上下文长度增长。

#### 推荐配置

MLX 框架下最小变体（3-bit）需要约 112 GB 内存。统一内存低于 128 GB 的 Mac Studio 配置（M4 Max 36/48/64 GB、M3 Ultra 96 GB）无法通过 MLX 运行 MiniMax-M2.7，但仍可尝试使用 [llama.cpp](https://github.com/ggerganov/llama.cpp) 等支持更激进量化策略的推理框架。

| Mac Studio 配置 | 推荐变体 | 说明 |
| :---------------- | :--------------------------------- | :---------------------------------------------------------------- |
| M4 Max — 128 GB | 3-bit（100 GB）或 4-bit-mxfp4（122 GB） | 3-bit 较为充裕；4-bit-mxfp4 内存紧张，上下文受限 |
| M3 Ultra — 256 GB | 6-bit（186 GB） | **推荐 6-bit**；8-bit（243 GB）仅剩 13 GB 余量，易触发 Swap 或被系统终止进程 |
| M3 Ultra — 512 GB | 8-bit-gs32（257 GB） | **推荐 8-bit-gs32**，质量接近无损且留有充足余量；BF16（457 GB）仅剩约 55 GB，长上下文场景易 OOM |

### 快速开始

#### 安装

```bash theme={null}
pip install -U mlx-lm
```

#### 使用方式

使用 `mlx_lm` 在终端直接生成文本，或通过 Python 脚本调用：

 ```bash CLI theme={null}
 mlx_lm.generate \
 --model mlx-community/MiniMax-M2.7-6bit \
 --prompt "解释神经网络中混合专家模型的概念。" \
 --max-tokens 8192 \
 --temp 1.0
 ```

 ```python Python theme={null}
 from mlx_lm import load, generate
 from mlx_lm.sample_utils import make_sampler

 # 根据内存选择合适的变体 — 参见上方「模型选择指南」
 model, tokenizer = load("mlx-community/MiniMax-M2.7-6bit")

 sampler = make_sampler(temp=1.0, top_p=0.95, top_k=40)

 messages = [{"role": "user", "content": "用 Python 写一个查找素数的函数。"}]
 prompt = tokenizer.apply_chat_template(
 messages,
 tokenize=False,
 add_generation_prompt=True,
 )

 response = generate(
 model,
 tokenizer,
 prompt=prompt,
 max_tokens=8192,
 sampler=sampler,
 verbose=True,
 )
 print(response)
 ```

### API 服务部署

如需将 MiniMax-M2.7 作为本地 API 服务（兼容 OpenAI SDK），可以使用 **mlx\_lm.server** 或第三方工具如 **LM Studio**。

#### mlx\_lm.server

启动 OpenAI 兼容的 API 服务：

```bash theme={null}
mlx_lm.server \
 --model mlx-community/MiniMax-M2.7-6bit \
 --port 8080
```

`mlx_lm.server` 会自动将模型的思考过程（`` 标签内容）分离到 `reasoning` 字段，`content` 仅包含最终回复：

```json theme={null}
{
 "choices": [{
 "message": {
 "role": "assistant",
 "content": "\n\n你好！很高兴见到你！",
 "reasoning": "用户用中文打招呼，我应该友好地回复。"
 }
 }]
}
```

启动服务后，可通过 curl 或 OpenAI SDK 调用：

 ```bash curl theme={null}
 curl http://localhost:8080/v1/chat/completions \
 -H "Content-Type: application/json" \
 -d '{
 "model": "mlx-community/MiniMax-M2.7-6bit",
 "messages": [{"role": "user", "content": "你好！"}],
 "temperature": 1.0,
 "top_p": 0.95,
 "max_tokens": 8192
 }'
 ```

 ```python Python theme={null}
 from openai import OpenAI

 client = OpenAI(
 base_url="http://localhost:8080/v1",
 api_key="not-needed",
 )

 response = client.chat.completions.create(
 model="mlx-community/MiniMax-M2.7-6bit",
 messages=[{"role": "user", "content": "你好！"}],
 temperature=1.0,
 top_p=0.95,
 max_tokens=8192,
 )

 # content 已自动分离，strip() 去除前导换行
 print(response.choices[0].message.content.strip())
 ```

#### LM Studio

[LM Studio](https://lmstudio.ai/) 提供了带有内置 MLX 支持的 Mac 桌面应用：

1. 下载并安装 [LM Studio](https://lmstudio.ai/)
2. 在模型浏览器中搜索 `MiniMax-M2.7`
3. 选择适合您内存的量化变体
4. 点击 **Load** 即可开始对话

LM Studio 同样提供 OpenAI 兼容的本地服务，便于与其他工具集成。

### 微调（LoRA）

`mlx-lm` 支持使用 LoRA / QLoRA 对模型进行参数高效微调（PEFT）。对于 MiniMax-M2.7 这种 MoE 模型，建议使用**量化模型 + QLoRA** 以降低内存需求。

 在 Mac Studio 上微调 MiniMax-M2.7 对内存要求极高。即使使用 QLoRA，也需要至少 **192 GB 统一内存**（M3 Ultra），并采用非常保守的训练参数。128 GB 及以下配置不建议尝试微调。

#### 安装训练依赖

```bash theme={null}
pip install "mlx-lm[train]"
```

#### 准备训练数据

训练数据使用 JSONL 格式。在数据目录中创建 `train.jsonl`、`valid.jsonl` 和（可选的）`test.jsonl` 文件。

**对话格式（推荐）：**

```json theme={null}
{"messages": [{"role": "user", "content": "你的问题"}, {"role": "assistant", "content": "期望的回答"}]}
```

**补全格式：**

```json theme={null}
{"prompt": "输入文本", "completion": "期望的输出"}
```

#### 启动微调

使用量化模型进行 QLoRA 微调，并启用内存优化选项：

```bash theme={null}
mlx_lm.lora \
 --model mlx-community/MiniMax-M2.7-4bit \
 --train \
 --data ./your-data-dir \
 --batch-size 1 \
 --num-layers 4 \
 --grad-checkpoint \
 --iters 600 \
 --adapter-path ./adapters
```

关键参数说明：

| 参数 | 建议值 | 说明 |
| :------------------ | :---- | :------------------- |
| `--batch-size` | `1` | 大模型必须使用最小 batch size |
| `--num-layers` | `4` | 仅微调少量层以节省内存 |
| `--grad-checkpoint` | — | 用计算换内存，降低峰值占用 |
| `--iters` | `600` | 根据数据量调整 |

#### 使用微调后的模型

**加载适配器生成文本：**

```bash theme={null}
mlx_lm.generate \
 --model mlx-community/MiniMax-M2.7-4bit \
 --adapter-path ./adapters \
 --prompt "你的提示词" \
 --max-tokens 2048
```

**将适配器合并到模型（可选）：**

```bash theme={null}
mlx_lm.fuse \
 --model mlx-community/MiniMax-M2.7-4bit \
 --adapter-path ./adapters
```

合并后可直接加载融合模型，无需额外指定适配器路径。

***

## 推荐参数

以下为 MiniMax-M2.7 的推荐参数，适用于所有推理框架：

| 参数 | 推荐值 |
| :------------ | :----- |
| `temperature` | `1.0` |
| `top_p` | `0.95` |
| `top_k` | `40` |

***

## 相关链接

 官方模型权重、文档和基准测试

 MLX Community 上的全部量化变体

 vLLM 官方文档和部署指南

 SGLang 官方文档和部署指南