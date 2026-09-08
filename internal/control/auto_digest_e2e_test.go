package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestAutoDigestRoundRunsAfterTeammateCompletion is the orchestration
// boundary effect test for the live-feedback round: with DigestAuto on, a
// teammate job completion wakes an unattended leader turn whose input carries
// the <team_digest> block (drained completion note + no-redispatch guard).
// The recordingProvider asserts the digest round's input actually reached the
// executor — the full 6.1 chain (HandleJobDone → completion callback →
// digestCh → worker → RunTurn) with a real jobs manager, no API key.
func TestAutoDigestRoundRunsAfterTeammateCompletion(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	systemPrompt := "You are a terse coding agent."
	sink, _, events := collectSink()
	rec := &recordingProvider{streams: [][]provider.Chunk{{
		{Type: provider.ChunkText, Text: "收到任务。"},
		{Type: provider.ChunkDone},
	}}}
	executor := agent.New(rec, tool.NewRegistry(), agent.NewSession(systemPrompt), agent.Options{Temperature: 0}, sink)
	jm := jobs.NewManager(sink)
	defer jm.Close()

	roReg := tool.NewRegistry()
	roReg.Add(fakeControlTool{})
	task := agent.NewTaskTool(rec, nil, roReg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", systemPrompt, nil, 0, "", "", nil).
		WithTranscripts(agent.NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	ts := agent.NewTeammateStore(task, jm, t.TempDir())
	t.Cleanup(ts.Close)
	ts.SetSink(sink)

	c := New(Options{
		Executor:     executor,
		Runner:       executor,
		Jobs:         jm,
		Teammates:    ts,
		Sink:         sink,
		SystemPrompt: systemPrompt,
		SessionDir:   t.TempDir(),
		SessionPath:  t.TempDir(),
		DigestAuto:   true,
	})
	defer c.Close()

	if err := ts.Create("smoke2", "researcher"); err != nil {
		t.Fatalf("team create: %v", err)
	}
	ctx := jobs.WithSession(jobs.WithManager(context.Background(), jm), c.parentSessionID())
	if _, err := ts.Assign(ctx, "smoke2", "简述前缀缓存如何降低推理成本"); err != nil {
		t.Fatalf("team add: %v", err)
	}

	// Wait for the teammate job to finish AND the auto digest round to have
	// run: the last recorded leader turn must carry the digest block.
	deadline := time.Now().Add(30 * time.Second)
	var last string
	var n int
	for {
		reqs := rec.recorded()
		n = len(reqs)
		if n > 0 {
			last = lastUserMessage(reqs[n-1].Messages)
		}
		if strings.Contains(last, "<team_digest>") {
			break
		}
		if time.Now().After(deadline) {
			var notices []string
			for _, e := range *events {
				if e.Kind == event.Notice {
					notices = append(notices, e.Text)
				}
			}
			t.Fatalf("auto digest round not observed; %d leader turns, last tail=%q, notices=%q", n, last, notices)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !strings.Contains(last, "do not dispatch new background tasks") {
		t.Fatalf("digest round input must forbid redispatch: %q", last)
	}
}
