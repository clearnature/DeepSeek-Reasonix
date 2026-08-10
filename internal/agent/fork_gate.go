package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"reasonix/internal/permission"
	"reasonix/internal/provider"
)

// forkHistoryWindowRatio is the size guard ceiling for a fork's inherited
// prefix: the parent's committed history (in estimated tokens) may not exceed
// this fraction of the child's context window, otherwise the fork is rejected
// (plan §二 裁决：深度 guard（≤maxSubagentDepth）+ 大小 guard（父历史 ≤窗口 80%））。
const forkHistoryWindowRatio = 0.8

// forkReadOnlyGate is the P5 fork execution-layer gate (T3): the fork child
// keeps the parent's full writer-capable schema (prompt-cache prefix identity,
// plan §一.4 安全双轨的「前缀要求」) but executions are read-only — read-only
// tools and permission-classified read-only bash pass; every other call is
// rejected with a reason fed back to the model. The gate rides the run context
// (WithForkReadOnlyGate), mirroring WithForkSource, since TaskTool is shared
// across concurrent runs.
type forkReadOnlyGate struct{}

func (forkReadOnlyGate) Check(_ context.Context, toolName string, args json.RawMessage, readOnly bool) (bool, string, error) {
	// Bash is schema-level writer-capable but a concrete invocation can be
	// permission-classified as read-only. Allow exactly those; anything else
	// bash-shaped falls through to the writer rejection below.
	if toolName == "bash" && permission.BashCommandIsReadOnly(args) {
		return true, "", nil
	}
	if readOnly {
		return true, "", nil
	}
	return false, fmt.Sprintf("fork sub-agents are read-only: %q is a writer and is blocked — only read-only tools and permission-classified read-only bash commands are allowed", toolName), nil
}

// forkReadOnlyGateKey is the context key for the fork child's execution gate.
type forkReadOnlyGateKey struct{}

// WithForkReadOnlyGate stamps the fork execution gate onto a run context so
// RunSubAgentWithSession can install it on the child's Options (T3). Only the
// P5 fork branch sets it; every other sub-agent path finds no key and keeps
// its existing gate.
func WithForkReadOnlyGate(ctx context.Context, g Gate) context.Context {
	return context.WithValue(ctx, forkReadOnlyGateKey{}, g)
}

// forkReadOnlyGateFromContext resolves the fork execution gate, if any.
func forkReadOnlyGateFromContext(ctx context.Context) (Gate, bool) {
	g, ok := ctx.Value(forkReadOnlyGateKey{}).(Gate)
	return g, ok
}

// checkForkDepthGuard is the recursion guard (T3): a fork child inherits the
// parent prefix and may itself fork (fork-of-fork) only while the chain stays
// within max_subagent_depth. SubagentDepth(ctx) is the parent's depth; the
// fork child would run at depth+1.
func checkForkDepthGuard(ctx context.Context, maxDepth int) error {
	maxDepth = NormalizeMaxSubagentDepth(maxDepth)
	if SubagentDepth(ctx)+1 > maxDepth {
		return fmt.Errorf("subagent fork depth limit reached (max_subagent_depth=%d)", maxDepth)
	}
	return nil
}

// checkForkSizeGuard is the size guard (T3): the fork's inherited prefix (the
// parent's committed history, already truncated by captureForkPrefix) must fit
// inside forkHistoryWindowRatio of the child context window, estimated in
// tokens the same way compaction budgets the conversation. Oversized history is
// rejected up front rather than collapsing the child's first request (which
// would break the byte-identical prefix the provider cache depends on).
func checkForkSizeGuard(contextWindow int, prefix []provider.Message) error {
	if contextWindow <= 0 {
		return nil // no window configured: no size ceiling enforced
	}
	tokens := estimateMessagesTokens(prefix)
	limit := int(float64(contextWindow) * forkHistoryWindowRatio)
	if tokens > limit {
		return fmt.Errorf("fork prefix is too large for the context window: ~%d estimated tokens exceed 80%% of the %d-token window; compact the conversation or trim history before forking", tokens, contextWindow)
	}
	return nil
}
