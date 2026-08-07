# 中转路由 + 语义缓存服务器设计方案

> 状态: v1.1（2026-08-08）— 决策点 1 已定: **A: LiteLLM 一体化**（用户确认）
> 目标服务器: 110.41.86.205（4 核 2.0GHz / 7.5G 内存 / 150G 磁盘 / Redis 8.2.8 / podman 5.8.2）
> 关联: [[prefix-stability-core-value]]（缓存哲学）、[[billing-transparency-official-api-only]]（计费透明）

## 1. 目标与背景

### 1.1 要解决的问题

团队多人使用同一个 DeepSeek key 时：
- **服务端 prefix cache**（DeepSeek 自动）已让"公共前缀"（系统提示 + 工具 schema）跨人命中，输入费降至 ¥0.02/M——**这部分免费，无需自建**。
- **语义缓存**（本方案核心）：把 `(prompt → response)` 存到中转服务器硬盘，**语义相似**的后续请求直接返回缓存响应，**完全不走上游 API**——命中时 0 上游成本。

### 1.2 两种缓存的分工

| 维度 | DeepSeek prefix cache | 语义缓存（本方案） |
|---|---|---|
| 存在哪 | DeepSeek 服务端 | 中转服务器硬盘 |
| 命中条件 | 前缀字节完全相同 | prompt 语义相似（相似度 > 阈值） |
| 命中省什么 | 输入费（仍打 API） | 整个调用（0 上游成本） |
| 谁提供 | DeepSeek 免费自动 | 需自建 |

**两者互补**：prefix cache 省"每次请求的输入费"，语义缓存省"重复问题的整个调用"。

## 2. 架构设计

```
客户端（团队多人，Reasonix/任意 OpenAI 兼容客户端）
      │  （api.prajna.wiki 统一入口）
      ▼
┌──────────────────────────────────────────────────────┐
│ nginx (443, 已有)                                     │
│   api.prajna.wiki → 127.0.0.1:3003                    │
└──────────────────────┬───────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────┐
│ 语义缓存层（新增，端口 4000）                          │
│  ┌────────────────────────────────────────────────┐  │
│  │ 请求拦截: prompt → embedding → 向量相似度检索     │  │
│  │  命中 (sim ≥ 0.85) → 返回缓存响应（0 上游）       │  │
│  │  未命中 → 转发网关 → 拿响应 → 写入缓存 → 返回      │  │
│  └────────────────────────────────────────────────┘  │
└──────────────────────┬───────────────────────────────┘
                       ▼
┌──────────────────────────────────────────────────────┐
│ 路由/网关层（new-api 或 LiteLLM，端口 3003）           │
│  计费 / 密钥管理 / 多上游路由 / 日志                    │
└──────────────────────┬───────────────────────────────┘
                       ▼
                  上游（DeepSeek / OpenAI / 其他）
```

### 2.1 组件选型（推荐：LiteLLM 一体化）

| 组件 | 推荐 | 备选 | 资源占用 |
|---|---|---|---|
| 网关 + 语义缓存 | **LiteLLM**（内置 Redis/Qdrant 语义缓存 + 计费 + 100+ 上游） | new-api + GPTCache 外挂 | ~500MB-1GB |
| 向量检索 | **Redis 8.2.8 + RediSearch 模块**（需装模块） | Qdrant（独立容器 ~200MB） | ~200-400MB |
| embedding | **本地 bge-small-zh-v1.5**（ONNX，~100MB 模型） | 调 API（DeepSeek embedding，未公开；OpenAI embedding 付费） | ~500MB + CPU |
| 响应存储 | Redis（String，TTL 可控） | 硬盘目录（JSON 文件） | 磁盘充足 |

### 2.2 为什么不选 new-api + GPTCache 外挂为主方案

- new-api 本身**无语义缓存**，需额外写适配层（OpenAI 兼容请求 ↔ GPTCache），维护两套系统。
- LiteLLM 语义缓存是**内置一等公民**（`config.yaml` 配置即用），含 Redis 精确缓存 + Qdrant/Redis 语义缓存。
- new-api 优势（完整计费面板）对团队内部场景是锦上添花，不是必需。

## 3. 缓存策略（核心决策）

### 3.1 缓存键与隔离

```
缓存键 = hash(model + 用户/项目 + prompt归一化文本)
隔离维度:
  - model: 不同模型响应不同，必须隔离
  - 用户/项目: 防 A 的私有上下文污染 B（团队保密要求）
  - prompt: 归一化（去空白/大小写）后做 embedding
```

### 3.2 相似度阈值

| 阈值 | 效果 | 适用 |
|---|---|---|
| 0.90+ | 高精度，命中少 | 代码/精确问答 |
| **0.85（推荐默认）** | 平衡 | 通用 |
| 0.75-0.80 | 高命中，有错答风险 | 非关键场景 |

### 3.3 TTL（缓存有效期）

| 场景 | TTL |
|---|---|
| 团队活跃项目（每天用） | 7-30 天（活跃即续期） |
| 一般问答 | 24h-7 天 |
| 有版本/知识更新风险 | 1-24h |

### 3.4 流式请求处理

- **非流式**：完整响应可缓存，命中直接返回。
- **流式（SSE）**：命中时仍需以 SSE 格式**重放**缓存内容（保持客户端协议一致）；未命中时透传上游流并**边收边存**（存最终完整响应）。

## 4. 部署步骤（LiteLLM 路径）

### 4.1 环境准备（服务器已具备）

```
Redis 8.2.8 ✅（需装 RediSearch 模块或用 Qdrant）
podman 5.8.2 ✅
Python 3.12 ✅
内存 6.8G 可用 ✅
磁盘 150G ✅
```

### 4.2 安装清单

```bash
# 1. 语义缓存依赖
podman run -d --name qdrant -p 6333:6333 qdrant/qdrant     # 向量库（或 Redis + RediSearch）
pip install sentence-transformers onnxruntime              # 本地 embedding

# 2. LiteLLM 网关
podman run -d --name litellm -p 3003:4000 \
  -v /opt/litellm/config.yaml:/app/config.yaml \
  ghcr.io/berriai/litellm:main-latest

# 3. nginx 反代（已有 api-prajna.conf，改指向 4000）
```

### 4.3 config.yaml 核心片段

```yaml
model_list:
  - model_name: deepseek-v4-flash
    litellm_params:
      model: deepseek/deepseek-v4-flash
      api_key: os.environ/DEEPSEEK_API_KEY
  - model_name: gpt-4o            # 若做路由（可选，透明标注）
    litellm_params:
      model: deepseek/deepseek-v4-pro

litellm_settings:
  cache: true
  cache_params:
    type: redis-semantic        # 或 redis（精确）/ qdrant-semantic
    host: localhost
    port: 6379
    similarity_threshold: 0.85
    ttl: 604800                # 7 天
```

## 5. 验证方案

### 5.1 功能验证

```bash
# 1. 首次请求（未命中 → 上游 → 写缓存）
curl api.prajna.wiki/v1/chat/completions -d '{"model":"deepseek-v4-flash","messages":[{"role":"user","content":"什么是CRT?"}]}'

# 2. 语义相似请求（应命中缓存，0 上游调用，响应 <100ms）
curl api.prajna.wiki/v1/chat/completions -d '{"model":"deepseek-v4-flash","messages":[{"role":"user","content":"请解释一下CRT理论"}]}'

# 3. 验证差异（改模型/用户应隔离，不命中）
```

### 5.2 降本量化指标

| 指标 | 测量方式 | 目标 |
|---|---|---|
| 缓存命中率 | LiteLLM 日志/Redis 统计 | 团队场景 ≥ 30% |
| 上游调用减少 | 对比网关日志 | ≥ 30% |
| 平均响应延迟 | 命中 vs 未命中 | 命中 < 100ms（vs 上游 ~1-2s） |
| 月费用 | 账单对比 | 视命中率 |

### 5.3 与现有 DeepSeek prefix cache 叠加

- 未命中语义缓存的请求 → 仍打 DeepSeek → **prefix cache 继续生效**（公共前缀命中，输入 ¥0.02/M）
- 两层叠加：语义缓存省"重复问题整体"，prefix cache 省"每请求输入"

## 6. 安全与隐私

| 项 | 措施 |
|---|---|
| 缓存内容 | 可能含团队私有上下文 → 按用户/项目隔离，**绝不清除隔离维度** |
| API key | 主 key 只存服务器（.env / os.environ），客户端只拿网关子 key |
| 传输 | nginx TLS（已有 certbot 证书） |
| 敏感查询 | 可配置跳过缓存（如含"密码/token"关键词的请求不缓存） |
| 合规 | 缓存的数据属于用户 → 提供清除接口 + 明确 TTL |

## 7. 风险与权衡

| 风险 | 影响 | 缓解 |
|---|---|---|
| **缓存返回过时答案** | 团队拿到旧信息 | 短 TTL + 关键域跳过缓存 + 响应头标注 `x-cache: hit` |
| **语义误命中**（相似但不等价） | 错答 | 阈值 0.85+ 起步，按域调优 |
| LiteLLM 内存泄漏（8GB 实测） | 需重启 | systemd 每周定时重启 / 监控 RSS |
| embedding CPU 推理慢 | 每请求 +50-100ms | bge-small 已够小；可换 ONNX 优化 |
| 缓存容量增长 | 磁盘/内存 | LRU + TTL + 上限（如 10 万条） |

## 8. 决策点（已定 + 待定）

**已定（2026-08-08）**:
- ✅ 方案 = **A: LiteLLM 一体化**（用户确认）

**待定**:

1. **方案 A（推荐）：LiteLLM 一体化** vs **方案 B：new-api + GPTCache 外挂**
2. **向量库**：Qdrant（独立，简单） vs Redis + RediSearch（复用现有）
3. **embedding**：本地 bge-small-zh（免费，+50ms） vs API embedding（快但付费）
4. **对外 vs 对内**：仅团队内部（简单） vs 对外开放（需计费/注册/限额，new-api 更合适）
5. **路由替换**：是否做"透明模型路由"（明确标注，非欺诈）——建议不做模型伪装

## 9. 实施顺序

```
Phase 1: 搭 LiteLLM + Redis 精确缓存（验证网关 + 缓存链路，1 天）
Phase 2: 加语义缓存（embedding + 向量检索，1-2 天）
Phase 3: 团队接入（发子 key、配置 Reasonix provider 指向、观察命中率）
Phase 4: 调优（阈值/TTL/隔离维度，持续）
```

## 10. 相关

- [[prefix-stability-core-value]] — 缓存哲学（语义缓存是请求级扩展）
- [[knowledge-cache-governance-language]] — 已有知识缓存的容量治理经验可复用（LRU/TTL/隔离）
- [[billing-transparency-official-api-only]] — 计费透明原则（若对外开放）
