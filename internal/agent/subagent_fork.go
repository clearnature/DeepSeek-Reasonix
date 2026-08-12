package agent

import (
	"context"
	"encoding/json"

	"reasonix/internal/provider"
)

// captureForkPrefix 构造 fork 子代理预填前缀：父 system + 历史截断（去当前
// 未完成 assistant 轮次，尾部未配对剔除）+ 深拷贝——零发送、父零改动。
// 前缀与父已发送字节 byte-identical：子代理首请求命中父已建缓存的硬前提
// （plan §一.1/§五）。取父 modelVisibleMessages（投影有效则投影视图，与
// Prepare 发送同源）而非 Session 原始快照，让子代理继承父的压缩视图而非
// 未压缩全量——否则共享大上下文反复触发 overflow 压缩（8/12 凌晨 24 次）。
// 调用方（RunProfileSpec fork 分支）预填进子代理 session。
func captureForkPrefix(parent *Agent, ctx context.Context) []provider.Message {
	if parent == nil || parent.session == nil {
		return nil
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil
		}
	}
	msgs := parent.modelVisibleMessages()
	// 深拷贝先行：返回的切片完全独立于父 Session，任何修改（含 ToolCalls /
	// Images / Receipts 等内嵌 slice）都不会回流到父会话。
	msgs = cloneForkMessages(msgs)
	return truncateUnfinishedTurn(msgs)
}

// cloneForkMessages 深拷贝消息日志：外层 slice 与每条 Message 内嵌的
// slice/指针目标一并复制，保证 fork 前缀与父 Session 零共享可变状态；
// 纯值字段（Role/Content/Reasoning* 等 string、int64/bool）按值复制。
// captureForkInheritance 捕获子代理继承的 admission 校准：父的 lastUsage
// 实测与 promptCalibration 值快照，仅同 model 时继承（跨 model 的 tokenizer
// 属性不同，calibration 不可移植）；父无可用值或 model 不一致则 ok=false，
// 子代理保持冷启动（fallback 估算 + 首轮自校准）。
func captureForkInheritance(parent *Agent, modelRef string) (*provider.Usage, *promptTokenCalibration, bool) {
	if parent == nil || parent.modelRef != modelRef {
		return nil, nil, false
	}
	var usage *provider.Usage
	if lu := parent.lastUsage.Load(); lu != nil {
		cp := *lu
		usage = &cp
	}
	var cal *promptTokenCalibration
	if c := parent.promptCalibration.Load(); c != nil {
		cc := *c
		cal = &cc
	}
	if usage == nil && cal == nil {
		return nil, nil, false
	}
	return usage, cal, true
}

func cloneForkMessages(msgs []provider.Message) []provider.Message {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]provider.Message, len(msgs))
	for i, m := range msgs {
		if len(m.ToolCalls) > 0 {
			m.ToolCalls = append([]provider.ToolCall(nil), m.ToolCalls...)
		}
		if len(m.Images) > 0 {
			m.Images = append([]string(nil), m.Images...)
		}
		if len(m.ResponsesItems) > 0 {
			items := make([]json.RawMessage, len(m.ResponsesItems))
			for j, it := range m.ResponsesItems {
				items[j] = append(json.RawMessage(nil), it...)
			}
			m.ResponsesItems = items
		}
		if len(m.MemoryCitations) > 0 {
			m.MemoryCitations = append([]provider.MemoryCitation(nil), m.MemoryCitations...)
		}
		if len(m.DecisionReceipts) > 0 {
			receipts := make([]*provider.DecisionReceipt, len(m.DecisionReceipts))
			for j, r := range m.DecisionReceipts {
				if r != nil {
					cp := *r
					receipts[j] = &cp
				}
			}
			m.DecisionReceipts = receipts
		}
		if m.DecisionReceipt != nil {
			cp := *m.DecisionReceipt
			m.DecisionReceipt = &cp
		}
		if m.ToolExecution != nil {
			te := *m.ToolExecution
			if te.ExitCode != nil {
				ec := *te.ExitCode
				te.ExitCode = &ec
			}
			m.ToolExecution = &te
		}
		if m.InterruptedTurn != nil {
			it := *m.InterruptedTurn
			if len(it.CompletedTools) > 0 {
				it.CompletedTools = append([]provider.InterruptedToolSummary(nil), it.CompletedTools...)
			}
			if len(it.InterruptedTools) > 0 {
				it.InterruptedTools = append([]string(nil), it.InterruptedTools...)
			}
			m.InterruptedTurn = &it
		}
		out[i] = m
	}
	return out
}

// truncateUnfinishedTurn 从尾部剔除「当前未完成的 assistant 轮次及其后」，
// 使前缀停在父已发送的最后一个完整请求边界（缓存命中硬前提）：LocalOnly
// 消息一律剔除；带 tool_calls 但未被后续结果全部配对的轮次整轮作废；纯文本
// assistant 与配对完整的工具轮次保留。返回原切片前缀视图（len 缩短）。
func truncateUnfinishedTurn(msgs []provider.Message) []provider.Message {
	end := len(msgs)
	for {
		// 1. 剔除尾部 LocalOnly（未完成/中断记录）。
		for end > 0 && msgs[end-1].LocalOnly {
			end--
		}
		if end == 0 {
			break
		}
		// 2. 向前找截断窗口内最后一个 assistant 消息（跨越其后的 tool 结果）。
		j := end - 1
		for j >= 0 && msgs[j].Role != provider.RoleAssistant {
			j--
		}
		if j < 0 {
			break // 无 assistant（纯 system/user 或孤儿 tool 前缀），无需截断
		}
		// 3. 该 assistant 的 calls 全部配对 → 前缀就绪。
		if forkToolCallsAnswered(msgs[j], msgs[j+1:end]) {
			break
		}
		// 4. 未配对 → 剔除该 assistant 及其后的部分配对结果，继续向前检查。
		end = j
	}
	return msgs[:end]
}

// forkToolCallsAnswered 报告 assistant 消息的每个 tool call 是否都已被后续
// tool 结果消息配对（RoleTool 且 ToolCallID 命中）。无 tool_calls 的纯文本
// 回复视为配对完整。
func forkToolCallsAnswered(a provider.Message, following []provider.Message) bool {
	if len(a.ToolCalls) == 0 {
		return true
	}
	answered := 0
	for _, call := range a.ToolCalls {
		for _, f := range following {
			if f.Role == provider.RoleTool && f.ToolCallID == call.ID {
				answered++
				break
			}
		}
	}
	return answered == len(a.ToolCalls)
}

// forkSourceKey 是 WithForkSource 的 context 键（包内私有，仿 evidence 的
// WithSessionMessages 惰性访问模式：ctx 只携带父 Agent 引用，不携带数据）。
type forkSourceKey struct{}

// WithForkSource 把父 Agent 挂到工具执行上下文，使 task 工具的 fork 分支
// （T2）能通过 ForkSourceFromContext 取到父 Agent 并调用 captureForkPrefix，
// 而无需在 Tool 注册表里持有父引用（T0-A：TaskTool 无父 Agent 引用，需此通道，
// 仿 evidence.WithSessionMessages 的注入模式）。
func WithForkSource(ctx context.Context, parent *Agent) context.Context {
	return context.WithValue(ctx, forkSourceKey{}, parent)
}

// ForkSourceFromContext 解析 WithForkSource 挂载的父 Agent。
func ForkSourceFromContext(ctx context.Context) (*Agent, bool) {
	parent, ok := ctx.Value(forkSourceKey{}).(*Agent)
	return parent, ok
}
