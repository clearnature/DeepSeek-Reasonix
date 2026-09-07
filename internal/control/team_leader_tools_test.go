package control

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

func newSmokeTeamTool(t *testing.T) (tool.Tool, *jobs.Manager) {
	t.Helper()
	sink, _, _ := collectSink()
	jm := jobs.NewManager(sink)
	t.Cleanup(jm.Close)
	systemPrompt := "You are a terse coding agent."
	task := agent.NewTaskTool(&smokeProvider{}, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", systemPrompt, nil, 0, "", "", nil).
		WithTranscripts(agent.NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	ts := agent.NewTeammateStore(task, jm, t.TempDir())
	t.Cleanup(ts.Close)
	ts.SetSink(sink)
	return NewTeamLeaderTool(ts), jm
}

// TestTeamToolCreatesAssignsReportsRemoves locks in the model-callable
// orchestration surface: the leader agent can run the full member lifecycle
// (create -> add -> status shows idle -> remove) without host commands,
// which is the architectural first principle from the 2026-09-07 report.
func TestTeamToolCreatesAssignsReportsRemoves(t *testing.T) {
	tt, jm := newSmokeTeamTool(t)
	ctx := jobs.WithSession(jobs.WithManager(context.Background(), jm), "test-session")

	call := func(action, name, task, role string) string {
		args := map[string]string{"action": action}
		if name != "" {
			args["name"] = name
		}
		if task != "" {
			args["task"] = task
		}
		if role != "" {
			args["role"] = role
		}
		raw, _ := json.Marshal(args)
		out, err := tt.Execute(ctx, raw)
		if err != nil {
			t.Fatalf("team %s: %v", action, err)
		}
		return out
	}

	if out := call("group_create", "writers", "", ""); !strings.Contains(out, `"writers" created`) {
		t.Fatalf("group_create = %q", out)
	}
	if out := call("create", "alice", "", "researcher"); !strings.Contains(out, `"alice" created`) {
		t.Fatalf("create = %q", out)
	}
	if out := call("add", "alice", "write a note about cache", ""); !strings.Contains(out, "job dispatched") {
		t.Fatalf("add = %q", out)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		out := call("status", "", "", "")
		if strings.Contains(out, "alice: idle") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("alice never returned to idle in roster: %q", out)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if out := call("group_delete", "", "", ""); !strings.Contains(out, "team dissolved") {
		t.Fatalf("group_delete = %q", out)
	}
	if out := call("status", "", "", ""); !strings.Contains(out, "no teammates") {
		t.Fatalf("members survived group_delete: %q", out)
	}
}

// TestTeamToolVisibilityRequiresJobManagerContext verifies ProviderVisible:
// only leader contexts (carrying a jobs manager) expose the tool; plain or
// child contexts stay hidden, mirroring send_message.
func TestTeamToolVisibilityRequiresJobManagerContext(t *testing.T) {
	tt, _ := newSmokeTeamTool(t)
	pt, ok := tt.(interface{ ProviderVisible(context.Context) bool })
	if !ok {
		t.Fatal("team tool must implement ProviderVisible")
	}
	if pt.ProviderVisible(context.Background()) {
		t.Fatal("team tool must be hidden without a jobs-manager context (leader-only)")
	}
}
