package agent

import (
	"context"

	"reasonix/internal/provider"
)

type parentAgentContextKey struct{}

// withParentAgent stamps ctx with the agent executing the current tool round
// so skill sub-agent runners can capture the parent's fork prefix.
func withParentAgent(ctx context.Context, a *Agent) context.Context {
	return context.WithValue(ctx, parentAgentContextKey{}, a)
}

// ContextParentAgent returns the agent executing the current tool round.
func ContextParentAgent(ctx context.Context) *Agent {
	a, _ := ctx.Value(parentAgentContextKey{}).(*Agent)
	return a
}

// forkPrefillSession reports whether sess was prefilled with a parent fork
// prefix; sub-agent start context still applies to it.
func forkPrefillSession(sess *Session) bool {
	if sess == nil {
		return false
	}
	sess.mu.RLock()
	defer sess.mu.RUnlock()
	return sess.forkPrefill
}

// EphemeralSubagentRunWithPrefix builds an ephemeral run whose session is
// prefilled with a fork prefix (system + committed history), like
// PrepareParentFork but without a persisted transcript owner.
func EphemeralSubagentRunWithPrefix(prefix []provider.Message) *SubagentRun {
	sess := NewSession("")
	for _, m := range prefix {
		sess.Add(m)
	}
	sess.MarkForkPrefill()
	return &SubagentRun{Session: sess}
}
