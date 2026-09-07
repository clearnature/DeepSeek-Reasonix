package jobs

import (
	"context"
	"io"
)

// StartForSessionReadonly starts a background job that skips the
// workspace-lease retain observer. Read-only background tasks (qwen-style
// parallel researchers: teammates, read_only_task) never write the workspace,
// so retaining the lease would lock the leader out of its own workspace for
// the whole run. Writer jobs keep upstream lease semantics. Lives in its own
// file so jobs.go does not drift past its repolint budgets.
func (m *Manager) StartForSessionReadonly(parentSession, kind, label string, run func(ctx context.Context, out io.Writer) (string, error)) *Job {
	return m.startForSession(parentSession, kind, label, run, false, false, true)
}
