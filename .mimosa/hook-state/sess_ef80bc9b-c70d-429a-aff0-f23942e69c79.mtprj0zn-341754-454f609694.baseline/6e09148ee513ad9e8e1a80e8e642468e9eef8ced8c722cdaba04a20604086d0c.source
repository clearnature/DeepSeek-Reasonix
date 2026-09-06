package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"reasonix/internal/tool"

	"reasonix/internal/tool/builtin"
)

// askGate wraps a tool so a teammate whose leader configured the tool for
// approval (P11 /team-ask) gets its call auto-submitted instead of executed:
// the request rides the approvals channel, the leader answers with
// /team-approve, and the verdict comes back on the P3 steer queue. Non-team
// contexts (the leader, unrelated sub-agents) pass straight through — the
// gate is inert without a mailbox + identity.

type askGate struct {
	inner tool.Tool
}

func (g askGate) Name() string            { return g.inner.Name() }
func (g askGate) Description() string     { return g.inner.Description() }
func (g askGate) Schema() json.RawMessage { return g.inner.Schema() }
func (g askGate) ReadOnly() bool          { return g.inner.ReadOnly() }
func (g askGate) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	m, ok := builtin.MailboxFromContext(ctx)
	if !ok || m == nil {
		return g.inner.Execute(ctx, args)
	}
	name := builtin.MailboxIdentityFromContext(ctx)
	if name == "" || !m.AskRequiredForTool(name, g.Name()) {
		return g.inner.Execute(ctx, args)
	}
	if err := m.AskForTool(name, g.Name(), string(args)); err != nil {
		return "", fmt.Errorf("%s requires leader approval: %w", g.Name(), err)
	}
	return fmt.Sprintf("tool %q requires leader approval — request submitted; wait for the /team-approve verdict (arrives as a steer message)", g.Name()), nil
}

// wrapAskGates wraps every tool in a sub-registry with askGate so teammate
// forks honor /team-ask tool approvals. Non-teammate callers pass through.
func wrapAskGates(sub *tool.Registry) *tool.Registry {
	if sub == nil {
		return sub
	}
	for _, name := range sub.Names() {
		if t, ok := sub.Get(name); ok {
			sub.Add(askGate{inner: t})
		}
	}
	return sub
}

// ProviderVisible forwards the inner tool's contextual visibility (registry
// checks ContextualTool) so wrapping never leaks teammate-only tools to the
// leader or unrelated sub-agents.
func (g askGate) ProviderVisible(ctx context.Context) bool {
	if ct, ok := g.inner.(tool.ContextualTool); ok {
		return ct.ProviderVisible(ctx)
	}
	return true
}
