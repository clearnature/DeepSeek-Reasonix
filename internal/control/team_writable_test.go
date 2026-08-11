package control

import (
	"strings"
	"testing"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestTeamCreateWritableParsing pins the /team-create <name> [role]
// [writable] syntax: the trailing "writable" token must opt the teammate
// out of the default read-only execution gate (P6.1), and omitting it must
// keep the teammate read-only.
func TestTeamCreateWritableParsing(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()

	rec := &recordingProvider{streams: [][]provider.Chunk{{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}}
	executor := agent.New(rec, tool.NewRegistry(), agent.NewSession("sys"), agent.Options{}, event.Discard)
	task := agent.NewTaskTool(rec, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil)
	ts := agent.NewTeammateStore(task, jm, t.TempDir())

	var sinkText []string
	sink := event.FuncSink(func(e event.Event) {
		if e.Kind == event.Notice {
			sinkText = append(sinkText, e.Text)
		}
	})
	c := New(Options{
		Executor:     executor,
		Jobs:         jm,
		Teammates:    ts,
		Sink:         sink,
		SystemPrompt: "sys",
	})
	defer c.Close() // stops the teammate auto-advance worker (goleak verifier)

	// Explicit writable opt-in.
	c.Submit("/team-create writer coder writable")
	var writer *agent.Teammate
	for i := range ts.List() {
		tm := &ts.List()[i]
		if tm.Name == "writer" {
			writer = tm
		}
	}
	if writer == nil || !writer.Writable {
		t.Fatalf("teammate writer: Writable = false, want true (writable token must opt out of the read-only gate)")
	}

	// Default stays read-only.
	c.Submit("/team-create reader coder")
	var reader *agent.Teammate
	for i := range ts.List() {
		tm := &ts.List()[i]
		if tm.Name == "reader" {
			reader = tm
		}
	}
	if reader == nil || reader.Writable {
		t.Fatalf("teammate reader: Writable = true, want false (default must stay read-only)")
	}

	// A plain role (no writable token) must not accidentally flip writable.
	c.Submit("/team-create strict coder")
	var strict *agent.Teammate
	for i := range ts.List() {
		tm := &ts.List()[i]
		if tm.Name == "strict" {
			strict = tm
		}
	}
	if strict == nil || strict.Writable {
		t.Fatalf("teammate strict: Writable = true, want false (no writable token)")
	}

	joined := strings.Join(sinkText, "\n")
	if !strings.Contains(joined, `"writer" created`) || !strings.Contains(joined, `"reader" created`) {
		t.Fatalf("missing created notices; sink=%v", sinkText)
	}
}
