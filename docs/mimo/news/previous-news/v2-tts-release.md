# Xiaomi MiMo-V2-TTS: Versatile Voice Agent that Speaks and Sings

**Xiaomi** **mimo-v2-tts** is a large-scale speech synthesis model independently developed by Xiaomi. Built on a proprietary audio tokenizer and a multi-codebook joint speech–text modeling architecture, it has been trained on hundreds of millions of hours of speech data with large-scale pretraining and multi-dimensional reinforcement learning, enabling highly controllable, fine-grained speech style generation. mimo-v2-tts supports precise control ranging from global style setting to nuanced local emotional expression. It can perform tone shifts and gradual emotional transitions within a single utterance, faithfully reproducing the natural prosody of human speech. When singing, it can also accurately render pitch and rhythm, delivering natural and expressive performance.

The mimo-v2-tts model is now available through the Xiaomi MiMo API open platform (https://platform.xiaomimimo.com), **with free access for a limited time**.

### Text Control
Flexible and customizable style control

mimo-v2-tts supports free-form natural language descriptions instead of being limited to predefined keywords. The model can understand and follow arbitrary descriptive instructions.

- Emotion control: happy, sad, angry, gentle, excited, calm…

- Dialect support: Northeastern Mandarin, Sichuan dialect, Henan dialect, Cantonese, Taiwanese accent…

- Role play: Monkey King, Lin Daiyu, Iron Man…

- Freely combined phrases — true natural language control: “cute and coquettish, soft ‘baby voice’,” “lazy, just woke up, slightly husky,” “deeply affectionate, slow speaking pace,” “passionate and powerful”

### Fine-grained control of vocal events

mimo-v2-tts can naturally insert and control various paralinguistic vocal events in speech, making the generated audio more realistic and expressive.

Supported vocal events: laughter, coughing, pauses, thinking/hesitation, sighing, etc.

## Deep Text Understanding

The model can intelligently recognize formatting cues in text and convert them into corresponding speech expressions—such as tone and punctuation—without requiring extra annotations.

Format awareness → speech rendering:

- ALL CAPS text (e.g., “THIS IS IMPORTANT”) → automatically adds emphasis;

- Repeated words or characters (e.g., “no no no no no”) → automatically mapped to matching rhythm and emotion.

During pretraining, the model learned from large-scale text–speech aligned data, enabling it to convert written formatting signals into natural-sounding speech.

## Beyond Speech: Dialects · Characters · Singing

mimo-v2-tts goes beyond standard speech synthesis with rich and versatile expressive capabilities. It supports natural pronunciation across multiple dialects, enables role-playing with stylized character performances, and delivers high-quality singing synthesis—allowing a single model to speak, act, and sing with ease.

## Open API

mimo-v2-tts is now officially available via API. **Free access is available for a limited time.** 

Visit [https://platform.xiaomimimo.com](https://platform.xiaomimimo.com/) to get started.
