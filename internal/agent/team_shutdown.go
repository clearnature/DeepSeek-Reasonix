package agent

import (
	"fmt"
	"log/slog"
)

// RequestShutdown asks a teammate to shut down cooperatively (qwen
// requestShutdown analog): a shutdown_request mail lands in its inbox
// (delivered on its next assignment) and, when the member is mid-job, a
// stop instruction is steered into the running job so it winds down instead
// of being hard-killed. TeamStop remains the hard-stop path (DeleteGroup /
// emergencies).
func (ts *TeammateStore) RequestShutdown(name string) error {
	ts.mu.Lock()
	tm, ok := ts.teammates[name]
	if !ok {
		ts.mu.Unlock()
		return fmt.Errorf("unknown teammate %q", name)
	}
	jobID := tm.LastJobID
	running := tm.State != TeammateIdle
	ts.mu.Unlock()

	if err := ts.PostMail(name, "shutdown_request: please finish your current work and stop. Reply via team_message to leader with shutdown_approved when done."); err != nil {
		return fmt.Errorf("request shutdown mail: %w", err)
	}
	if running && jobID != "" && ts.jm != nil {
		if err := ts.jm.SendMessageForSession("", jobID, "SHUTDOWN: stop all further work and end your turn now."); err != nil {
			return fmt.Errorf("request shutdown steer: %w", err)
		}
	}
	slog.Info("team shutdown requested", "name", name, "running", running, "job", jobID)
	return nil
}
