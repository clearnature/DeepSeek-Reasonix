# 本地实现整体评估报告（2026-08-14）

> 语境：dev/develop 主题合并吸收上游 v1.25.x 13 项 + 本地独有实现。
> 只读评估（T1-T6 验证），无实现变更。

## 1. 同步态势（T1）

```
上游 26 提交 → ✅ 已吸收 13 / ⏸ hold 13
已吸收：#8779（384K）、#8756（stop_reason）、#8675（ACP）、#8674（卡片）、
        goal、#8701（inbox）、#8669（渲染）、#8731（待办×2）、
        #8718（改造吸收）、release notes×3、bench #8790
hold 13：desktop 会话树/归档（#8739 闸门——会话恢复 6 + 归档 5 + 待办 1）
```

## 2. 验证链结果（T2/T3/T4/T5）——全绿

| 项 | 结果 |
|---|---|
| gofmt -l | 干净 |
| go build ./... | ✓ |
| provider/acp/config 测试 | ✓（6.6s/4.4s/3.9s） |
| 守卫套件（cachehit_e2e/healthy-window/RejectsExhausted/投影/校准/压缩） | ✓（1.1s） |
| P6 team/检索/遥测测试 | ✓（60.6s） |
| repolint | clean（2116 baselined） |
| desktop tsc | ✓ |

## 3. 红区结论（T3——压缩域）

- **#8718 改造吸收**（c3cd09cd5）：4 处估算遍历统一为 Walk（含 Raw——官方
  口径——与上游相反）+ 补全 Title/URL——**守卫套件全过**（无触发漂移）
- **四方遥测审计**：telemetry 91 字段 ↔ detail 22 键 ↔ recorder 37 case ↔
  record 73 tag——**字段齐全**（tpc/est/elapsed_ms/pref_hash/action/results/
  saved_chars 均覆盖）
- **官方口径复核**：含 Raw 断言与 Anthropic 文档一致（重放搜索计 input tokens）

## 4. 中区结论（T4）

- **#8731/#8669**：tsc clean + 滚动/水合守卫——**待办回跳修复闭环**
- **#8701 差别识别**：sessioninbox（15 文件——会话队列服务端）vs dev P6
  team inbox（teammate_store.go——agent 间通信）——**不同层，无重叠**

## 5. 本地独有健康度（T5）——通过

P6 team（Teammate/Assign）、检索（Retrieval）、遥测（Recorder）测试全过。

## 6. 实验物处置建议（T6——待用户确认）

| 项 | 性质 | 建议 |
|---|---|---|
| cmd/team-test1/ | P6 team 驱动（test1 任务） | 保留（入库 or 忽略——用户决定） |
| benchmarks/deepseek_v3_tokenizer* | DeepSeek 官方 tokenizer（离线标定用） | 保留（入库 or 忽略——用户决定） |

## 7. 发布就绪度

- **内核 + desktop 前端**：验证链全绿——可编译发布
- **剩余 13 hold 解锁条件**：#8739 上游修复（启动重建 recovery-aware）或本地
  实现后整体评估——**非阻塞**（hold 域独立于已吸收功能）

## 8. 结论

本地实现（13 吸收 + 本地独有）健康：无回归、红区深挖通过、四方审计完整。
剩余 hold 全部由 #8739 闸门控制——不影响当前发布。
