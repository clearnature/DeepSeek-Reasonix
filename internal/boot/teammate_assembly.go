package boot

import (
	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// newTeammateOrchestration assembles the P6 TeammateStore for the controller.
// It lives outside build() (already past its repolint budgets) so the
// orchestration wiring can grow without pushing build further over, and in an
// upstream-absent file so a convergence cannot silently drop the wiring.
func newTeammateOrchestration(sink event.Sink, jm *jobs.Manager, store *agent.SubagentStore, root, baseModel, baseEffort string, newTask func() *agent.TaskTool) *agent.TeammateStore {
	teammateTask := newTask()
	if store != nil {
		teammateTask = teammateTask.WithTranscripts(store, root, baseModel, baseEffort)
	}
	ts := agent.NewTeammateStore(teammateTask, jm)
	ts.SetSink(sink)
	if root != "" {
		ts.SetWorkspaceRoot(root)
	}
	return ts
}
