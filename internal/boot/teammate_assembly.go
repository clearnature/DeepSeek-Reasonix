package boot

import (
	"reasonix/internal/agent"
	"reasonix/internal/control"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

// newTeammateOrchestration assembles the P6 TeammateStore for the controller.
// It lives outside build() (already past its repolint budgets) so the
// orchestration wiring can grow without pushing build further over, and in an
// upstream-absent file so a convergence cannot silently drop the wiring.
// When the store is wired it also registers the leader model tool (team) on
// the shared registry — the orchestration layer's primary caller is the model.
func newTeammateOrchestration(sink event.Sink, jm *jobs.Manager, store *agent.SubagentStore, root, baseModel, baseEffort string, newTask func() *agent.TaskTool, reg *tool.Registry, inboxRoot, snapshotPath string) *agent.TeammateStore {
	teammateTask := newTask()
	if store != nil {
		teammateTask = teammateTask.WithTranscripts(store, root, baseModel, baseEffort)
	}
	ts := agent.NewTeammateStore(teammateTask, jm, inboxRoot)
	ts.SetSink(sink)
	if root != "" {
		ts.SetWorkspaceRoot(root)
	}
	if snapshotPath != "" {
		ts.SetSnapshotPath(snapshotPath)
	}
	if ts != nil {
		reg.Add(control.NewTeamLeaderTool(ts))
	}
	return ts
}
