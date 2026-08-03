# MiMo-V2.5 Series Inference Full-Link Optimization: Pushing Hybrid SWA Efficiency to the Extreme

The MiMo V2.5 series models (including MiMo-V2.5, MiMo-V2.5-Pro, etc.) integrate multiple architectural features: Hybrid Sliding Window Attention (Hybrid SWA) reduces KVCache storage to approximately 1/7 of Full Attention through hybrid window attention; MoE reduces the computational cost per token while maintaining model capacity through sparse activation; MultiModal Machine Learning Encoder supports cross-modal understanding of vision, audio, video, etc. The combination of the three endows the MiMo-V2.5 series of models with significant potential for effectiveness and efficiency in long-context and multi-modal scenarios.

From the very beginning of the design of the MiMo-V2 model, we have had a very clear goal: to train a model that is both sufficiently powerful and efficient in long text reasoning scenarios. However, these two goals inherently have tension in engineering. Strong reasoning ability means that the model needs to have the ability to model long-range dependencies, which usually corresponds to larger-scale attention computation and higher KVCache overhead. In the traditional Full Attention architecture, both the computational complexity of Attention and the storage of KVCache grow rapidly with the context length, making the cost of long context training and inference quickly become unacceptable. The core idea of Hybrid SWA is to perform hierarchical mixing between local window attention (Sliding Window Attention, SWA) and global attention (Full Attention): the vast majority of layers only compute attention within local windows, and only a small number of key layers retain a global view. Theoretically, this structure can reduce the computational complexity of Attention to nearly linear while still maintaining the ability to model long-range dependencies. 

However, the theoretical architectural advantages do not naturally translate into efficiency advantages in real online systems. On the one hand, Hybrid SWA significantly increases the complexity of KVCache cache hits, prefix matching, and the maintenance of Full / SWA semantic consistency. On the other hand, in real engineering systems, data transfer in multi-level storage, inconsistencies between asynchronous prefetch and scheduling, and the difficulty of aligning distributed cache states together make it difficult to directly realize the theoretical benefits.

In addition to Hybrid SWA, MoE places higher demands on distributed scheduling and load balance; the throughput bottleneck of MultiModal Machine Learning Encoder in large graph and long video scenarios also urgently needs to be broken through. Moreover, the optimization of general components such as scheduling strategies and Prefill/Decode execution links is also indispensable. What this article records is precisely a whole-link engineering practice of an inference system centered around the MiMo-V2.5 series of models, covering KVCache management, hierarchical cache systems, SWA prefix cache trees, scheduling strategies, Prefill/Decode execution links, and MultiModal Machine Learning optimization, systematically translating the efficiency potential of the architecture (especially Hybrid SWA) into the production environment.

## 1. Inference Efficiency Advantages of the Hybrid SWA Architecture

Before delving into specific discussions on optimization work, it is necessary to first quantify the efficiency upper bound of Hybrid SWA - this serves not only as the starting point for architecture selection but also as the theoretical benchmark and theoretical upper bound for all subsequent system optimizations. 

### 1.1 Computational Complexity Analysis

Taking MiMo-V2.5-Pro as an example, this model has a total of 70 layers, among which 10 layers are Full Attention, the remaining 60 layers are SWA, and the sliding window size of SWA is 128. Compared with Full Attention, the computational complexity of Hybrid SWA is shown in the figure. Since the proportion of SWA layers is 6/7, the computational complexity of the Hybrid SWA architecture is approximately 1/7 that of Full Attention. In the Chunk Prefill scenario, Prefill is approximately compute-bound, and this gap is basically equivalent to the theoretical reduction in Prefill cost. 

### 1.2 KVCache Storage Analysis

Since the SWA layer only needs to retain KV within the sliding window and does not need to store the full sequence, the KVCache occupancy also drops to nearly 1/7. The Decode phase is approximately memory-bandwidth-bound, and its latency is proportional to the read volume of model parameters plus KVCache. In the case of long sequences, the volume of KVCache may far exceed the model parameters, so the reduction in KVCache storage is almost directly equivalent to the reduction of decode cost in long sequence scenarios. 

There are significant differences in the KVCache storage of different model architectures, and there are also differences in memory access patterns. The following is an estimate of the KVCache size for each domestic model. It can be seen that MiMo-V2.5-Pro and MiMo-V2.5 rank second among domestic models in terms of KVCache, second only to DeepSeek-V4-Pro and Flash. 

It should be noted that the actual cost difference is not strictly equivalent to the KVCache scale ratio, because there are also fixed computational and memory access overheads that are independent of the sequence length. However, in the long context scenario, the overall trend remains consistent: **the cost-effectiveness of short texts is similar, and the longer the sequence, the greater the inference cost advantage** . 

## 2. KVCache System Refactoring

As models that were among the first to adopt the Hybrid SWA architecture, the MiMo-V2 and MiMo-V2.5 series faced the issue that, at the time, neither mainstream open-source inference frameworks nor caching systems provided complete support for SWA. At the beginning of the MiMo API launch, we chose SGLang v0.5.5 as the codebase for the service backend. Immediately afterwards, we encountered a severe test. In the then version, SGLang's HiCache did not support SWA, or rather, the early SWA support was implemented in a way that "compatible with SWA at the cost of storing Full KVCache". Although there are some temporary solutions to make SWA more usable, we still hope to develop a KVCache system with a higher ceiling and better usability. 

### 2.1 SWA KVCache Management

#### KVCache Dual Pool 

Hybrid SWA introduces a core storage contradiction: the Full Attention layer needs to retain full-sequence KV (O(N)), while the SWA layer only needs to maintain KV within the sliding window (O(W)). However, under the traditional single KV pool design, the system must uniformly allocate video memory for all layers according to O(N), preventing the window sparsity of SWA from being utilized, and the actual storage efficiency degrades to an approximate implementation of Full KVCache.

To address this issue, a natural approach is to split KVCache into two independent pools, Full Attention and SWA, and perform unified abstraction at the system level: 

- **Physical Level**: Maintain Full KV pool and SWA KV pool separately. The SWA pool only configures its capacity according to the window size and supports independent eviction based on the window, thereby strictly limiting SWA storage to the O(W) scale; this mechanism also extends to the L2 and L3 storage levels.

- **Logical Level**: Still exposes a single sequence view to the upper layer (prefix tree, scheduler, and transport protocol), with the Full Attention index as the authoritative index, and maintains the mapping relationship from Full to SWA to achieve transparent hierarchical storage.

- **Scheduling Constraints** : When the system receives an access request, it simultaneously verifies the capacity constraints of Full KV and SWA KV to avoid misjudgment of resources in a single dimension. 

- **Data Transfer**: Cross-layer transfer is performed solely based on the SWA mask to ensure that only data within the valid window is transferred, thereby avoiding redundant bandwidth occupation.

Through the above design, SWA KVCache achieves strict O(W) storage constraints at the system level, increasing the overall KVCache capacity efficiency by approximately 7×, thereby truly unleashing the structural advantages of Hybrid SWA. Mainstream inference frameworks have also adopted similar implementation schemes. 

#### KVCache asynchronous layer-by-layer pulling 

After the implementation of SWA KVCache storage optimization, the SWA layer only needs to prefetch a very small amount of KVCache, which enables the process of prefetching KVCache from the Host to the Device to achieve perfect overlap through layerwise granularity scheduling. As a result, the cost of Cache reads during the inference process is close to zero. 

#### SWA-aware prefix cache tree 

The hit rule of traditional RadixAttention is based on a simple assumption: **equal token sequences → equal KV** . This assumption holds true in Full Attention mode - as long as two requests share the same token id, their corresponding KV must still be in the pool and can be directly reused. 

However, this assumption is broken in SWA mode. The reason is that the logical lifecycle of the prefix tree does not align with the physical lifecycle of SWA KV. The length of prefix tree nodes is not constrained by the SWA window. The sequence length of a node can be either shorter or much longer than the window; moreover, nodes change continuously with request merging, splitting, and removal. Thus, although a prefix tree node still logically represents a complete token sequence, its corresponding SWA KV **may only have its last part remaining, or may even have completely disappeared.** If the prefix tree still gives the reuse length according to the rule of "matching when tokens are equal", what the scheduler receives may be a pseudo-match where "the tail KV has evaporated" - subsequent attention calculations will read invalid or overwritten slots, directly affecting the model's performance. 

To ensure that prefix reuse remains correct and efficient in SWA mode, it is necessary to modify the semantics of the prefix tree in three aspects: 

1. **The matching rule has been upgraded to "window safety length"** : In addition to token equality, it is also necessary to ensure that at least W tokens at the end still have valid slots in the SWA pool. The matching length is trimmed to this new boundary - the part beyond it is treated as a miss. This ensures that all KV retrieved from the hit segment must be valid. 

1. **The elimination path is bound to the request lifecycle** : Each chunk of long prefill completion, request end, and decode generating a certain number of tokens all trigger a release of out-of-window SWA. This ensures that the SWA pool occupancy remains constant at the W or chunk level in long context/long output tasks, rather than growing with the sequence length. 

1. **The node simultaneously carries two sets of indexes** : Each prefix tree node records two pieces of information - the Full Attention segment index (determines the logical order and participates in the Full Attention layer computation) and the SWA segment mapping (determines window security). During eviction, they also need to be managed separately: the SWA segments outside the window can be evicted individually while retaining the Full Attention segments (so that the prefix can still be reused by the Full Attention layer), or the entire segment can be evicted. 

SWA compressing the KV volume to 1/7 is **the gain at the capacity level** , while the hit rate is **the gain at the reuse level** . Only when the two are multiplied can we obtain the curve of the actual computational cost during the prefill phase. After introducing the "window safety length" matching rule, the hit rate of KVCache with the same token capacity theoretically decreases slightly, but the number of tokens under the same storage capacity reaches several times, resulting in a substantial increase in the actual hit rate. 

#### Optimization of KVCache Hit Rate Improvement

After all three levels of HiCache are transformed into SWA-aware, the device, host, and storage backend each maintain a set of states indicating "which locations have valid SWA". However, HiCache's data transfer link is asynchronous, cross-deployment caches vary, and the length of shared prefixes across sessions also varies - which means that inconsistencies are likely to occur between the Full Attention Cache and valid SWA indices on each end. According to the matching rules of the SWA-aware prefix cache tree, if a sequence can be hit on the Full Attention Cache but not on the SWA Cache, it will lead to severe matching length truncation. The more truncation there is, the longer the length that needs to be recalculated, and the lower the optimization effect of the SWA Cache. Therefore, we have optimized distributed consistency and cache hit rate for different scenarios:

- **Device side complete, host side missing**: When L3→L2 prefetching only pulls in the tail segment of the sequence due to bandwidth and latency trade-offs, or when the L1 prefix tree is not synchronized to L2/L3 after reorganization, the problem of a complete device side but a missing host side may occur.In response, we proactively check the difference set of the occupancy of the two sets of SWA (device/host) at time points such as node merging in the prefix tree and completion of prefill, allocate additional slots in the SWA pool on the host side, and asynchronously write the SWA KV of the device from D2H. 

- **Host side is complete, device side is missing**: Wait until the next H2D completes natural alignment, no need for active patching.

- **High-frequency sequence L3 prefix expiration**: The head of the long sequence persists in L1/L2 due to high-frequency access, and Cache affinity routes requests with the same prefix to the same node. As a result, the Cache in L3 may be cleaned up by the storage eviction policy due to long-term lack of direct access, leading to the premature release of the L3 Cache of the global high-frequency sequence and a significant reduction in cross-machine reuse. To address this, when accessing the Cache on L1/L2, we query the L3 Cache at a certain interval to avoid eviction.

- **SWA Retention Strategy for Medium and Short Sequences** : For SWA of medium and short sequences, we fix relatively dense SWA KV Caches at certain lengths based on user request patterns. Although increasing the density of SWA will raise the proportion of SWA in the overall KV Cache, it can directly benefit scenarios such as multi-user sharing of system prompts. 

Through the above optimizations, we have truly transformed the expansion at the KV Cache capacity level into a high effective hit length, making it possible to reuse long prefixes across sessions, which is particularly beneficial for scenarios such as long conversations of agents, multi-user sharing of system prompts, and tool calls with multiple accesses to the same codebase. 

### 2.2 GCache: High-performance Distributed Cache Infrastructure

GCache is a high-performance general-purpose cache developed by Xiaomi's Storage Team, and it is an important part of building the "training and inference integrated" storage system. In the early days of the training scenario, the Storage Team realized that some open-source cache projects had limited acceleration effects on distributed file systems and could not fully unleash their performance, so they embarked on the path of self-developed. Later, with the release of the MiMo large model and the launch of inference services, the team also transformed GCache into an independent storage product for model distribution and as the L3 KVCache of the inference engine. 

GCache supports both file and KV semantics, multi-level caching of memory/disk/remote, has shm memory persistence and whole-link zero-copy capabilities, supports advanced features such as high-concurrency non-blocking IO and RDMA communication, meets the performance requirements of upper-layer services for high throughput and low latency, and has good scalability. 

#### Architecture Design 

GCache has several characteristics: 

1. **The decentralized metadata management approach allows the cluster scale to expand without limitations:** 

 - Calculate the consistent hash for the key to determine the storage location. 

 - Master uses Raft high-availability deployment. However, Master is only responsible for managing heartbeats and Service Discovery, and the IO path does not pass through Master.

1. **The server supports both memory and disk caching simultaneously:** 

 - Cold data in memory will be evicted to disk, while hot data on disk will be promoted to memory. This mode is very friendly to inference scenarios, automatically ensuring the performance of active sessions and reducing the cost of sessions that have not been started for a long time. 

 - Memory supports persistence to shm, ensuring that cached data is not lost when the service is restarted.

 - Supports smooth scaling up or down of machines without losing cache during the process. 

1. **Provide multi-language SDK, start a dedicated thread, slice and dispatch user requests:** 

 - Does not occupy the resources of user threads; slicing improves concurrency and controls the IO size within a range friendly to RDMA.

 - The thread callback operates in Asynchronous Mode, allowing flexible control of callback granularity, such as single kv level, batch level, or CUDA stream level. 

#### Network Optimization

Currently, mainstream GPU models are all equipped with eight 400G high-performance network cards. However, even when considering the separate deployment of PD, the current inference frameworks still struggle to fully utilize the network bandwidth, leading to voices in the industry calling for reducing the configuration of network cards to cut costs.

To fully leverage the capabilities of high-speed networks, GCache preferentially uses GPU network cards instead of front-end network cards for communication, and has made extensive optimizations on the communication module, such as NUMA binding and same-track affinity. In terms of specific performance metrics, when using 1MB-sized IOs, the RDMA read throughput of a single process can reach 170 GB/s, with a latency of only 280 us; in the GDR scenario, due to the higher bandwidth of HBM, a single process can reach approximately 350 GB/s, which is sufficient to meet the communication performance requirements of inference frameworks.

#### Storage Cost Optimization

2026 will undoubtedly be a year when the industry is highly sensitive to storage costs. Unlike other competitors that use dedicated storage models, GCache preferentially adopts the method of co-locating on GPU machines, taking over part of the memory of Prefill and Decode nodes, as well as several NVMe SSDs that come with the machine, with additional storage costs being 0.

#### Stability Assurance

Due to the mixed fabric, the high failure rate of GPU machines has become a hurdle in front of stability. Since its launch, GCache has basically encountered machine failures where the server is located every day. First, the team spent a great deal of time and effort strengthening the fault handling logic of the code; second, since the keys are fully scattered by consistent hashing, by performing pre-logical grouping on session IDs, the fault radius has also been reduced; finally, by combining the hardware detection capabilities provided by the underlying platform, faults are detected in advance, and automated processes are used for data migration. For extremely rare sudden downtime events that cannot be addressed in advance, by setting a lower SDK timeout, the inference framework can promptly detect misses and perform recalculation, ensuring that the front-end inference process is not significantly affected. 

Based on the above work, GCache has been able to maintain single-copy storage in the hybrid deployment state without using multi-copy means to improve availability, which is also one of the important reasons for the relatively low storage cost.

### 2.3 Discussion on Cache Hit Rate

Thanks to the aforementioned optimizations for SWA on KVCache - lower storage footprint, supplemented by a more stable large-capacity GCache as L3 storage - we have been able to significantly extend the TTL (Time-To-Live) of the cache, thereby substantially increasing the hit rate of the KV Cache. The eviction of KVCache essentially stems from storage capacity constraints. When the capacity approaches saturation, the system must prioritize retaining the KV Cache generated by new requests and evict historically accessed entries according to strategies such as LRU, which directly results in the fact that a certain context often fails to be hit when reused after several hours. The extremely small storage footprint of SWA enables the cache capacity for concurrent requests that can be supported to increase exponentially under the same cost, while the large-capacity L3 further expands the available capacity at low cost—the more abundant the storage space, the less pressure there is for KVCache to be evicted, and naturally the longer its retention duration.The longer the TTL, the wider the hit window for historical context, and the cache hit rate rises accordingly. Additionally, although the smaller bandwidth transmission pressure of SWA does not directly affect TTL, it significantly reduces the data transfer overhead between multi-level storage, providing a guarantee for the stable and efficient operation of the entire cache system. 

Since the model went live, we have continuously observed on the server side that under the mainstream high-quality harness framework, the server-side KV Cache hit rate can reach an average of **93%** ; for individual users with high-intensity and long-term usage, this indicator can even climb to **95%** or higher. In the future, we will continue to iterate on the KV Cache management logic of SWA and collaborate with more harness frameworks to promote the co-design of harness-inference to further optimize the upper limit of the cache hit rate. 

## III. Scheduling Optimization

In the early days of the SGLang community, the router service was not yet fully mature, and there was no data sharing among multiple instances. If a router service unexpectedly fails or requests are routed to different router services, the issue of KVCache scheduling fallback will occur. To address this issue and ensure high availability in large-scale Clustered Deployment, Xiaomi developed the dynamically scalable stateless scheduler LLM-Router. By using Redis as a centralized storage, it avoids the KVCache scheduling fallback phenomenon after a single-service failure, thereby ensuring a more stable cache hit rate. 

### 3.1 KVCache and Load Affinity Scheduling

Since HiCache is highly sensitive to the L2 hit rate, if there is a miss in L2, it is necessary to search in L3 and fetch the KVCache, and only after the fetching is completed can the request be inferred. Increasing the L2 hit rate from the router side can reduce unnecessary synchronous waiting, thereby directly improving throughput performance.

Router implements KVCache affinity scheduling by maintaining distributed requests in a Radix prefix tree. It preferentially selects among multiple Prefill instances**nodes that have cached the prefix of the current request**, and at the same time**takes load balance into account**to avoid hot spot skew. After the strategy was launched, it increased the cache hit rate of L2 by approximately**25%** , and the single-machine input throughput by approximately**30%** . Its core formula is roughly:

```bash

# 选择 score 最大的 worker，含义：缓存命中率高 + 负载低 = 得分高 = 优先选择score(worker) = matchWeight × prefix_match_percentage − normalized_load
```

### 3.2 TTFT Optimization

When queuing occurs in model services, the traditional First Come First Serve strategy does not consider the priority relationship between requests with high and low hit rates, causing requests with more cache hits but fewer real computational tokens to potentially wait for requests with lower cache hit rates to finish inference before they can start. The TTFT P99 of the overall service becomes extremely long, slowing down the average performance of the overall throughput. 

To address this issue, when the waiting queue selects to prioritize the execution of the prefill service, the Router side preferentially schedules requests with fewer real computational tokens, avoiding the problem of P99 degradation caused by blocking requests that originally had short computation times. Meanwhile, this strategy may lead to the starvation phenomenon where some requests are not scheduled for a long time, so we have also added a waiting time penalty mechanism to balance this phenomenon. The results show that this strategy does not degrade the service quality for shorter requests, while for longer requests, it can reduce the P90 metric of TTFT by up to **30%** . 

## 4. Prefill Optimization

### 4.1 Distributed Configuration

Theoretically, the smaller the EP (Expert Parallelism) during the prefill phase, the better the performance and throughput, which is mainly reflected in three aspects: fewer machines are involved, resulting in lower cross-machine communication overhead; fewer DP (Data Parallelism) numbers, reducing the impact of attention load differences between DPs on performance; and each machine can carry more experts, leading to better MoE load balance. However, the size of EP is constrained by video memory and needs to meet the video memory requirements of model parameters and KVCache. Early SWA KVCache needed to store the KVCache of all tokens, resulting in an excessively large EP; after optimization, only the KVCache of partial SWA tokens needs to be stored, and we have reduced the EP to 1/2 of its original size.**End-to-end performance has improved by approximately 40%.** Subsequently, we will continue to explore PP (Pipeline Parallelism) optimization for the Hybrid SWA architecture to further reduce the EP size and improve overall throughput.

### 4.2 Length Bucketing Strategy

Compared to the pure GQA architecture, the Hybrid architecture of the MiMo-V2.5 series significantly improves computational efficiency, but throughput still decreases significantly as the sequence length increases. The following figure shows the throughput in Chunked Prefill when computing 16K tokens with different prefix lengths:

In the Agentic scenario, most ultra-long requests originate from multi-round agent interactions and generally carry a large amount of prefix cache. When requests with significantly different lengths are scheduled to the same model instance, short requests will be dragged down by long requests, reducing overall throughput, which is mainly reflected in two scenarios: 

1. **DP-Attention Synchronization**: Multiple DPs need to perform collective communication after each layer of attention computation to synchronize the computation entering the MoE stage. If both long and short requests exist simultaneously on multiple DPs within the same EP group, the short requests will be slowed down by the computation of the long requests;

1. **Chunked Prefill Interference**: When requests with different prefix lengths are grouped into the same chunk for inference, short-prefix requests are slowed down by the computation of long-prefix requests.

To alleviate the above load imbalance issue, we adopted**the three-level length bucketing strategy**(0–64K / 64K–256K / 256K–1M), aggregating requests with similar load characteristics into the same bucket for computation,**significantly improving the average throughput of online prefill**. On this basis, we are currently exploring a more fine-grained and flexible bucketing mechanism to adapt to the dynamic changes of online loads.

### 4.3 MoE Load Balance

All models in the MiMo-V2.5 series adopt the MoE architecture, and the issue of expert load balance during the prefill phase needs to be considered. Since a load balance training objective was introduced during the pre-training phase and the training was relatively stable, the model has learned a relatively uniform expert allocation strategy during training. During the inference phase, without enabling any expert load balance strategy, **the average expert load per layer (the ratio of the average number of tokens across all ranks in a layer to the maximum number of tokens in a rank of that layer) is approximately 0.85, which is already at a relatively optimal distribution level**. Therefore, we currently have not introduced any expert load balance strategy. Subsequently, we will continuously monitor this indicator and introduce relevant optimizations as needed based on the dynamic evolution of the online load pattern.

### 4.4 Resolve NUMA Conflicts

In some Ubuntu systems, the kernel numa_balancing parameter conflicts with the numa-node configuration of SGLang, resulting in occasional large execution gaps between computing kernels during model inference. In a multi-node multi-GPU deployment, the occurrence positions of these gaps on each rank are random, and each time synchronization occurs between ranks, the overall computing process is slowed down by the slowest rank, thereby significantly affecting the overall inference efficiency. After disabling the numa_balancing parameter of the system kernel, the issue was resolved, **and end-to-end performance increased by approximately 10%** . 

## 5. Decode Optimization

### 5.1 Video Memory Optimization

In the Agentic scenario, multi-round conversations cause the context to continuously grow, and the video memory occupancy of KVCache has become the main bottleneck in decoding. After the video memory is fully occupied by KVCache, the batch size cannot be expanded, resulting in underutilized computation and limited decoding throughput. As a result, we can only increase the number of nodes to handle concurrency, which drives up inference costs. To increase the concurrency of a single node, we have implemented multiple video memory optimization measures: 

1. **Decode KVCache fully supports SWA**: The effective capacity of KVCache has increased by nearly 5 times**.** 

1. **Optimization of KVCache Preallocation in PD Separation**: Move the prealloc process of requests that have not yet started from GPU memory to CPU memory, and only move it into GPU memory when the decode actually starts, eliminating waste caused by resource preoccupation.

1. **CUDA Graph Memory Optimization**: Optimize CUDA Graph parameters to reduce space waste and increase available memory.

### 5.2 MTP Optimization

The MiMo-V2.5 series models natively support 3-layer MTP accelerated decode output, but MTP was not enabled during the prefill phase previously, resulting in the MTP using invalid KVCache for the initial 128 output tokens of decode and a very low prediction acceptance rate. Since most output sequences are short in Agentic scenarios, this defect significantly reduces the effectiveness of MTP acceleration. By introducing MTP support during the prefill phase and performing specialized adaptation and optimization on HiCache L2/L3, the acceleration effect of MTP in the early stage of decoding has been significantly improved: **The acceleration ratio reaches 2.3× for the 0–128th tokens and 1.5× for the 128–256th tokens** , effectively reducing the real decoding cost in agentic scenarios. 

## 6. MultiModal Machine Learning Inference Optimization

Based on the SGLang Community v0.5.7 EPD solution, we have carried out**a large number of engineering optimizations and stability fixes in EPD separation**around MiMo-V2.5, increasing the Encoder throughput to **2 times**while keeping the latency unchanged. Currently, we are feeding back the work results to the SGLang Community (issues#24945). The differences in Encoder performance before and after optimization are shown in the following table:

**6.1 Architecture Optimization**

- **MultiModal Machine Learning Embedding data replication and inference parallel** : In the Prefill scheduler main loop, support MultiModal Machine Learning Embedding data **asynchronous replication** between TPs, overlap with Prefill inference, and reduce GPU idle time.

- **Encoder supports Data Parallelism**: Since the Encoder model is relatively small, setting TP>1 will actually lead to performance degradation. Therefore, we deploy Encoder with TP=1 and at the same time **support Data Parallelism**, which facilitates deployment and operation and maintenance on a single machine with 8 GPUs.

- **Encoder supports cross-request group batching**: We introduced cross-request batching for the EPD Encoder Server, aggregating concurrent requests by modality through the Encoder scheduler, where the image/audio of multiple requests **is merged into a single Encoder Forward** and then split and returned according to the requests, thus solving the problem of low GPU utilization in per-request encoding.

### 6.2 Preprocessing Optimization

- **GPU Preprocessing of Images**: In large image scenarios, performing resize/normalize/patchify on the CPU will significantly increase end-to-end latency. Therefore, we move the preprocessing to the GPU,**eliminating the CPU bottleneck**. 

- **Parallel Image Download**: We use **multi-process parallelism** to download images and perform PIL decoding, avoiding the latency caused by sequential downloading and GIL lock contention.

- **MultiModal Machine Learning Download and Forward Parallelism**: In the early Encoder implementation, both data download and inference between batches and within batches were sequential, resulting in the GPU being idle during the download; we decoupled download and inference through a message queue to achieve**overlap between download and inference within batches**.

- **Video parallel decoding**: evenly divide the frame extraction indices into N chunks, and create an independent VideoDecoder instance for each chunk to perform multi-threaded parallel decoding, reducing the end-to-end latency of the 1-hour video Encoder **from 156s to 23s**.

### 6.3 Cache Optimization

- **Encoder Consistent Hashing**: In scenarios with multiple Encoders, Prefill uses polling to select Encoders, resulting in a decrease in the MultiModal Machine Learning cache hit rate. Through consistent hashing, we route the same requests to the same Encoder, **increasing the cache hit rate by 30%** . 

- **In-machine Embedding Cache Sharing** : Through shared memory, multi-modal cached data is shared among multiple cards within the Encoder machine, improving cache hit rate. 

## VII. Postscript

Looking back at the entire project, the inference efficiency of the MiMo-V2.5 series models does not stem from a single-point breakthrough in one particular stage, but rather from the result of multi-dimensional collaborative optimization. Hybrid SWA benefits both prefill and decode, but the under-optimized implementation of KVCache actually increases costs at each stage.Around this goal, we systematically reconstructed KVCache management, hierarchical caching, and prefix cache trees, tackled the core issues of SWA KVCache, optimized the scheduling strategy and Prefill/Decode links, and after verification in real online scenarios, finally translated its theoretical efficiency advantages into actual production environments. Only then did Hybrid SWA demonstrate its architectural advantages of combining strength and efficiency in **long text inference** . By further combining MoE configuration and various optimizations of MultiModal Machine Learning inference, the performance of online inference services has been significantly improved. 

We hereby present the first large-scale engineering implementation plan that comprehensively covers the Hybrid SWA + MoE + MultiModal Machine Learning combined architecture, and will pass on the cost savings achieved thereby to users through API price cuts. Meanwhile, we have already contributed some optimizations to the SGLang open-source community in the form of PRs, and will continue to advance more open-source initiatives, hoping to ensure that engineering optimization no longer serves as a barrier as soon as possible, enabling this type of composite architecture that combines strength and efficiency to be more widely explored and applied.