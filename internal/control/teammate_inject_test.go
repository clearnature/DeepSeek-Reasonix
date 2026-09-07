package control

import (
	"strings"
	"testing"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

// TestTeammateMessageRidesComposedTurn locks in the qwen <teammate_message>
// injection: mail a teammate sent to the leader is drained and injected ahead
// of the composed user text on the next turn — appended inside the same
// message, never touching the cache-stable prefix (memory-update /
// background-jobs pattern).
func TestTeammateMessageRidesComposedTurn(t *testing.T) {
	systemPrompt := "You are a terse coding agent."
	sink, _, _ := collectSink()
	jm := jobs.NewManager(sink)
	defer jm.Close()
	task := agent.NewTaskTool(&smokeProvider{}, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", systemPrompt, nil, 0, "", "", nil).
		WithTranscripts(agent.NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	ts := agent.NewTeammateStore(task, jm, t.TempDir())
	t.Cleanup(ts.Close)
	ts.SetSink(sink)

	c := New(Options{
		Executor:     agent.New(&smokeProvider{}, tool.NewRegistry(), agent.NewSession(systemPrompt), agent.Options{Temperature: 0}, sink),
		Jobs:         jm,
		Teammates:    ts,
		Sink:         sink,
		SystemPrompt: systemPrompt,
		SessionDir:   t.TempDir(),
		SessionPath:  t.TempDir(),
	})

	if err := ts.PostMailToLeader("alice", "found the root cause"); err != nil {
		t.Fatalf("PostMailToLeader: %v", err)
	}
	composed := c.Compose("continue")
	if !strings.Contains(composed, "<teammate_message>") || !strings.Contains(composed, "From alice: found the root cause") {
		t.Fatalf("composed = %q, want teammate_message block", composed)
	}
	// Drained: the second compose carries nothing (no duplicate injection).
	if again := c.Compose("continue"); strings.Contains(again, "found the root cause") {
		t.Fatalf("second compose re-injected mail: %q", again)
	}
	_ = event.Discard
}
