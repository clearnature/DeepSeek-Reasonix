package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

// send_message is the P3 parent-side steer channel: queues a message for a
// running background task job, drained as a mid-turn steer next tool round.
// Parent-only (jobs.WithoutManager hides it from sub-agents).

func init() {
	tool.RegisterBuiltin(sendMessage{})
}

type sendMessage struct{}

func (sendMessage) Name() string { return "send_message" }

func (sendMessage) Description() string {
	return "Queue a message for a running background task job started with task(run_in_background=true). The message is delivered to the background agent on its next turn as a user instruction, letting you steer a long-running task mid-flight. The job must still be running; finished or unknown jobs are rejected, and the queue is bounded (16 messages / 8KB) — overflow is rejected, never silently dropped. Only the parent agent can call this; it is hidden from sub-agents."
}

func (sendMessage) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"job_id":{"description":"The background task job id (e.g. \"task-1\") returned when it was started.","type":"string"},"text":{"description":"The instruction to send to the background agent. Delivered as a user message on the job's next turn.","type":"string"}},"required":["job_id","text"],"type":"object"}`)
}

func (sendMessage) ReadOnly() bool { return false }

// ProviderVisible restricts send_message to contexts carrying a job manager:
// the parent agent has one (withAgentContext), sub-agents and planners have it
// explicitly cleared, so their tool schemas never include this tool.
func (sendMessage) ProviderVisible(ctx context.Context) bool {
	_, ok := jobs.FromContext(ctx)
	return ok
}

func (sendMessage) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		JobID string `json:"job_id"`
		Text  string `json:"text"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(p.JobID) == "" {
		return "", fmt.Errorf("job_id is required")
	}
	if strings.TrimSpace(p.Text) == "" {
		return "", fmt.Errorf("text is required")
	}
	jm, ok := jobs.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("background jobs are not available in this context")
	}
	if err := jm.SendMessageForSession(jobs.SessionFromContext(ctx), p.JobID, p.Text); err != nil {
		return "", err
	}
	return fmt.Sprintf("Message queued for background job %q; it will be delivered on the job's next turn.", p.JobID), nil
}
