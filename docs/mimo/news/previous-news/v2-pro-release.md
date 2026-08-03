# Xiaomi MiMo-V2-Pro: Flagship Foundation Model towards Agent Era

Today, we are releasing Xiaomi mimo-v2-pro, Xiaomi’s flagship foundation model for the agent era.

Xiaomi mimo-v2-pro is built for demanding real-world Agent workflows. It has over **1T** total parameters, with **42B** active parameters, uses an innovative hybrid attention architecture, and supports an ultra-long context window of up to **1M** tokens. Based on the strong foundation model, we continue to scale compute across a broader range of agent scenarios, further expanding the action space of intelligence and achieving an important generalization leap from Coding to Claw.

On the global authoritative model intelligence ranking by Artificial Analysis, mimo-v2-pro ranks eighth worldwide and second in China.

<img src="./images/FZw6bk5M6odYsSxODBOcZds6nAb.png" alt="图片" style="margin: 16px auto;" />

In agent frameworks such as OpenClaw and Claude Code, mimo-v2-pro shows excellent end-to-end task completion ability. It can handle complex workflow orchestration, long-horizon planning, and precise tool use without human intervention, while reliably delivering final results. In overall hands-on experience, it has surpassed Claude Sonnet 4.6 and is approaching Opus 4.6, while its API pricing is only one-fifth of theirs, lowering the barrier to using frontier intelligence.

## A major leap in foundation capabilities

By scaling both parameters and compute, mimo-v2-pro reaches to a larger and stronger model foundation.

- **Trillion-parameter scale, efficient architecture**: Total parameters exceed 1T, with 42B active parameters, about 3x larger than the previous mimo-v2-flash. It continues to use the innovative Hybrid Attention mechanism introduced in mimo-v2-flash, with the hybrid ratio further increased from 5:1 to 7:1. This keeps inference efficient even with the large increase in model size, while also supporting 1M-token context. A lightweight MTP (Multi-Token Prediction) layer enables fast generation.

- **From Chat to Agent**: By scaling during post-training across a broader set of Agent tasks, the model is no longer limited to “answering questions” or “generating polished demos.” It is built to complete tasks. We aim to integrate it deeply into productivity scenarios so it can serve as the “brain” behind working systems and continuously deliver results with real-world impact.

- **Real-world experience beyond benchmark rankings**: mimo-v2-pro performs strongly across benchmarks that measure key model capabilities. In Coding Agent, general Agent, and Tool Use tasks, it is in the same tier as Claude 4.5 Sonnet, GPT5.2, and Gemini 3.0 Pro, showing leading intelligence. We remain focused on training and optimization guided by actual user experience, always paying close attention to how the model performs in real applications.

<img src="https://mimo.mi.com/static/IAt5buS6To3zKExg1RZcsGhhnPc.70149dcc5bf23184.png" alt="图片" style="margin: 16px auto;" />

## A flagship model built for Agents

mimo-v2-pro is deeply optimized specifically for Agent scenarios.

### The native brain for OpenClaw

OpenClaw is a general-purpose agent framework that has recently gained strong attention in the open-source community. As the core engine behind frameworks like this, the upper limit of the underlying model directly determines the system’s real-world performance. mimo-v2-pro is trained with SFT and RL on complex and diverse Agent scaffolds, giving it stronger tool-use and multi-step reasoning abilities. 

On OpenClaw’s standard benchmark leaderboards, PinchBench and ClawEval, mimo-v2-pro ranks among the best in the world. At the same time, with its 1M-token context window, mimo-v2-pro can comfortably support demanding real-world Claw application flows. Hunter Alpha shown below is an early anonymous version of mimo-v2-pro.

<img src="https://mimo.mi.com/static/XDC4bTG01opiD1xNgphcTj9gnih.58a0a54d2a1ba8e0.png" alt="图片" style="margin: 16px auto;" />

### Continuous Evolution of Coding Capabilities

Going beyond mere "Vibe Coding", mimo-v2-pro is capable of participating in more rigorous code engineering construction.

In in-depth evaluations by internal engineers at Xiaomi, mimo-v2-pro's user experience has approached that of Claude Opus 4.6, demonstrating advanced code intelligence: it boasts superior system design and task planning capabilities, more elegant coding styles, and more efficient, direct problem-solving pathways.

During the "Hunter Alpha" anonymous testing phase, the most frequently called apps were mostly programming-specific tools, which confirm mimo-v2-pro's high usability and reliability in real-world R&D scenarios.

<img src="https://mimo.mi.com/static/RxMnbgF97owiMPxcclPcrF5CnJd.5b6aa5473ce2cf13.png" alt="图片" style="margin: 16px auto;" />

## 1M Context Window, Open API

The mimo-v2-pro model is now officially available via API with pricing:

- Within 256K: Input at $1 / 1M tokens, Output at $3 / 1M tokens

- 256K ~ 1M: Input at $2 / 1M tokens, Output at $6 / 1M tokens

Visit [https://platform.xiaomimimo.com](https://platform.xiaomimimo.com/) to get started.
