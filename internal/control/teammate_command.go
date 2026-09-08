package control

import (
	"fmt"

	"reasonix/internal/agent"
	"reasonix/internal/jobs"
)

// Desktop/UI surface over the TeammateStore and job manager. Team
// orchestration is model-driven through the "team" tool (qwen alignment —
// sole entry point; /team-* host commands removed 2026-09-08). These
// read-only views keep the desktop panels alive without a command parser.

// TeamRosterView returns the teammate roster for the desktop team panel.
func (c *Controller) TeamRosterView() []agent.RosterView {
	if c.teammates == nil {
		return nil
	}
	return c.teammates.Roster()
}

// JobSnapshots returns the background-job state for the desktop job panel.
func (c *Controller) JobSnapshots() []jobs.JobSnapshot {
	if c.jobs == nil {
		return nil
	}
	return c.jobs.JobSnapshotsForSession(c.parentSessionID())
}

// SendTaskMessage routes a human message to a background job's worker via the
// job mailbox (a running teammate consumes it on its next loop turn).
func (c *Controller) SendTaskMessage(jobID, text string) error {
	if c.jobs == nil {
		return fmt.Errorf("no job manager")
	}
	return c.jobs.SendMessageForSession(c.parentSessionID(), jobID, text)
}
