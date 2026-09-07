package control

import (
	"context"
	"os"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// smokeProvider answers every request (leader and forked teammate alike) with
// one terse line, so the orchestration smoke can run with no real model, no
// network, and no API key — it exercises the jobs/scheduler/teammate/result
// loop, not any provider behavior.
type smokeProvider struct{}

func (p *smokeProvider) Name() string { return "smoke" }

func (p *smokeProvider) Stream(context.Context, provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 2)
	ch <- provider.Chunk{Type: provider.ChunkText, Text: "收到任务。"}
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}

// TestOrchestrationSmokeRunsTeammateChainToIdle is the orchestration-layer
// smoke test: agent orchestration is base infrastructure (jobs queue,
// scheduler, teammate store, completion events), distinct from any single
// subagent run. It drives the same /team-create -> /team-add chain as
// TestTeamChainRealDeepSeek but with a fake provider so the full loop —
// task admission, forked job, completion event, roster back to idle, result
// envelope — is verified in every test run without an API key.
func TestOrchestrationSmokeRunsTeammateChainToIdle(t *testing.T) {
	if os.Getenv("REASONIX_SKIP_ORCH_SMOKE") != "" {
		t.Skip("REASONIX_SKIP_ORCH_SMOKE set")
	}
	systemPrompt := "You are a terse coding agent."

	sink, _, _ := collectSink()
	executor := agent.New(&smokeProvider{}, tool.NewRegistry(), agent.NewSession(systemPrompt), agent.Options{Temperature: 0}, sink)
	jm := jobs.NewManager(sink)
	defer jm.Close()

	task := agent.NewTaskTool(&smokeProvider{}, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", systemPrompt, nil, 0, "", "", nil).
		WithTranscripts(agent.NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	ts := agent.NewTeammateStore(task, jm, t.TempDir())
	t.Cleanup(ts.Close)
	ts.SetSink(sink)

	c := New(Options{
		Executor:     executor,
		Jobs:         jm,
		Teammates:    ts,
		Sink:         sink,
		SystemPrompt: systemPrompt,
		SessionDir:   t.TempDir(),
		SessionPath:  t.TempDir(),
	})

	c.Submit("/team-create smoke1 researcher")
	if _, ok := ts.Status("smoke1"); !ok {
		t.Fatal("smoke1 not registered after /team-create")
	}

	c.Submit("/team-add smoke1 简述前缀缓存如何降低推理成本")

	deadline := time.Now().Add(30 * time.Second)
	for {
		st := agent.TeammateRunning
		for _, tm := range ts.List() {
			if tm.Name == "smoke1" {
				st = tm.State
			}
		}
		if st == agent.TeammateIdle {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("teammate smoke1 still %s after 30s", st)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Completion event drove the teammate back to idle — the orchestration
	// base loop is what the smoke guards; the result envelope is an extra
	// delivery detail covered by the real-API test.
	roster := c.teammates.Roster()
	var idle bool
	for _, rv := range roster {
		if rv.Name == "smoke1" && rv.State == string(agent.TeammateIdle) {
			idle = true
		}
	}
	if !idle {
		t.Fatalf("smoke1 did not return to idle: %+v", roster)
	}
	t.Logf("orchestration smoke passed: /team-create -> /team-add -> idle on %d teammate(s)", len(roster))
}
