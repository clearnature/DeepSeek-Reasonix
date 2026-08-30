# 摘要缓存打穿终局修复（2026-08-31）

> 与 `docs/compaction-cache-tools-freeze-20260831.md` 同步的 team 决策记录。
> 技能沉淀：`team-discipline` SKILL.md 缓存红线/合并纪律/命中率判读三节已更新。

## 结论（永久事实）

1. **摘要请求的缓存单元 = system+tools+messages 一体**。冻结前缀必须三者完整——
   只冻结 messages 不冻结 tools，MCP 异步注册/拦截器改写会使摘要 tools 与主请求
   分叉，前缀在 tools 处打穿（06:30:32 实锤：in=256,122 仅命中 system 16,896）。
2. **legacy sidecar 缺失字段必须回退 live 值**。旧 sidecar 无 `last_wire_tools`
   时发送空工具列表会被 commit 写回，造成永久打穿（fd34b1008）。
3. **同字节重复重放渐进命中是服务器正常特性**（40%→68%→99%+），不是 bug；
   验证修复用「同视图前后对照」（view_fp 相同、版本不同），如 6.6%→97.5%。
4. **面板「摘要调用开销」= 最近 11 次移动窗口**，低百分比先看构成
   （10 次修复前 + 1 次修复后 = 26.07%），再归因。

## 修复提交链

- `51e5d6e9b` messages 冻结 + sidecar `last_wire_messages` + resume 恢复
- `f1cce4a6b` tools 冻结 + sidecar `last_wire_tools` + RawMessage 字节保真
- `fd34b1008` legacy sidecar 无 tools 回退 live registry

## 验证样本（desktop 实测，同视图 192bace1）

```
06:30:32（0514） in=256,122 hit=16,896（6.6%）  system-only
07:20:36（0700） in=255,795 hit=249,472（97.5%） 完整单元（miss 6,323≈fold+指令）
```
