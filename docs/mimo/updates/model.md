# Model Release

## 2026-06-02 mimo-v2.5-asr Released

Model Introduction:

- **Bilingual & Dialects:** Supports Chinese, English, code-switching, and various regional dialects (Wu, Cantonese, Minnan, Sichuanese).

- **Lyrics Transcription:** High-accuracy Chinese/English lyrics transcription in mixed vocal-instrumental tracks.

- **Robust in Complex Audio:** Excels in challenging environments (high noise, far-field, multi-speaker).

- **Knowledge-Intensive AI:** Pinpoint accuracy for classical poetry, jargon, and proper nouns, with auto-punctuation.

## 2026-04-23 mimo-v2.5-pro Released

Model Introduction:

- **Trillion parameters, efficient architecture:** 1T total parameters | 42B activations | 1M ultra-long context

- **Ultimate Agent Performance:** In high-intensity agent scenarios, it performs comparably to Claude Opus4.6

## 2026-04-23 mimo-v2.5 Released

Model Introduction:

- **Native full-modal perception + 1M context:** Supports native understanding of images, videos, audio, and text, enabling cross-modal precise perception and long-range reasoning, with comprehensive perception capabilities ranking among the industry's forefront 

- **Powerful full-modal Agent capabilities:** It has native Agent execution capabilities, enabling it to efficiently complete complex tasks such as browsing, understanding, reasoning, and operation, with its performance in daily tasks comparable to that of **mimo-v2.5-pro**

- **Combining Performance and Efficiency:** While maintaining leading capabilities, achieving superior token efficiency, and positioned at the Pareto frontier of performance and efficiency

## 2026-04-23 MiMo-V2.5-TTS Series Release

Model Introduction:

- **Premium Voice TTS:** Built-in with multiple high-quality premium voices, it has strong capabilities in understanding and adhering to style instructions, supports fine-grained control over speech rate, emotion, tone, etc., and meets the expression needs of multiple scenarios 

- **Timbre Design:** Supports quickly defining and generating new timbres through a single sentence, making timbre creation more intuitive and efficient

- **Timbre Cloning:** Based on a small number of audio samples, it can reproduce the target timbre with high fidelity, while maintaining the consistency of timbre characteristics and possessing good generalization and stability

## 2026-03-18 mimo-v2-pro Release

**Model overview:**

- Uses hybrid architecture with a 1:7 ratio of Global Attention to Sliding Window Attention (SWA);

- 1T total parameters, with 42B active parameters;

- Supports an ultra-long context window of 1M tokens.

**Model details:** https://platform.xiaomimimo.com/#/docs/news/v2-pro-release

## 2026-03-18 mimo-v2-omni Release

**Model overview:**

- Supports up to 256K context length;

- Supports text, vision, and speech modalities.

**Model details:** https://platform.xiaomimimo.com/#/docs/news/v2-omni-release

## 2026-03-18 mimo-v2-tts Release

**Model overview:**

- Pretrained on over 100 million hours of data, using a self-developed multi-codebook speech modeling architecture;

- Offers unique capabilities such as style control, singing, and voice cloning.

**Pricing:** free for a limited time.

**Model details:** https://platform.xiaomimimo.com/#/docs/news/v2-tts-release

## 2026-02-04 mimo-v2-flash Update

1. **Upgraded Coding Capabilities in Thinking Mode:**
Specifically optimized for programming scenarios, the Thinking Mode now achieves a score of **78.6** on SWE-Bench Verified. Both the resolution rate and the quality of code generation have been significantly improved.

1. **Substantial Boost in Tool Calling Accuracy:**
Stability issues regarding tool usage have been resolved. Tool calling accuracy in Thinking Mode has surged from 64% to **97.0%**, greatly enhancing execution reliability in Agent scenarios.

1. **Enhanced Instruction Following & Reduced Hallucinations:**

- **Instruction Following:** Improved adherence to specific instructions, achieving an **AA-IFBench score of 72**.

- **Factuality:** Enhanced rigor in factual responses, with the **Non-Hallucination Rate updated to 52%**.

1. **Optimized Handling of Complex Tasks:**
Performance on Arena-Hard (Hard Prompts) in Thinking Mode has been strengthened, with the score rising to **60.6**. The model now demonstrates superior performance when handling high-difficulty logic problems.

1. **More Efficient Chain-of-Thought (CoT):**
By optimizing CoT generation strategies, the consumption of redundant tokens has been significantly reduced. In benchmarks such as AIME25 and HMMT, the average generation length has decreased by **13% to 30%**. This effectively lowers latency and token costs while maintaining model performance.

> Note: The model API call method and model name remain unchanged

## 2026-01-12 mimo-v2-flash Update

1. **Enhanced general capabilities:** Improved the model’s performance on a wide range of general-purpose tasks.

1. **Upgraded coding performance in Thinking mode:** Strengthened code generation quality in Thinking mode, especially for programming scenarios.

1. **Deep integration with Claude Code:** Fully supports using Thinking mode in Claude Code.

 - **Best practice:** Set Thinking as the default mode to achieve more stable, higher-quality code generation.

1. **Optimized Experience for Other Code Agents:** Synchronized improvements to the interaction experience and generation quality across code assistant tools (Code Scaffolds) such as Kilo, Cline, and Roo.

1. **Improved Stability & Instruction Following:** Enhanced output stability and significantly improved adherence to specific output formats.

> Note: The model API call method and model name remain unchanged

## 2025-12-16 mimo-v2-flash Release

**Model overview:**

- Uses hybrid architecture with a 1:5 ratio of Global Attention to Sliding Window Attention (SWA), a window size of 128, native 32K context, and extended training up to 256K;

- Introduces 3 MTP layers, delivering 2.5 to 3.7× faster inference.

**Pricing:** input 0.1/M tokens, output 0.3/M tokens.

**Model details:** [mimo-v2-flash](https://platform.xiaomimimo.com/#/docs/news/news20251216)[: High-Efficiency Inference, Code & Agent Foundation Model](https://platform.xiaomimimo.com/#/docs/news/news20251216)

**Usage guide:** [First API call](https://platform.xiaomimimo.com/#/docs/quick-start/first-api-call)