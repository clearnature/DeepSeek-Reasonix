package control

import (
	"testing"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestControllerCloseStopsTeammateAutoWorker pins the P6 close wiring: closing
// a controller that owns a teammate registry must stop the auto-advance worker
// goroutine (merge audit found DestroyAll alone leaked it — only Close() closes
// doneCh). The package goleak verifier fails if the worker outlives the test,
// so this guards the c.teammates.Close() call in Controller.Close.
func TestControllerCloseStopsTeammateAutoWorker(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()

	rec := &recordingProvider{streams: [][]provider.Chunk{{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}}
	executor := agent.New(rec, tool.NewRegistry(), agent.NewSession("sys"), agent.Options{}, event.Discard)
	task := agent.NewTaskTool(rec, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil)
	ts := agent.NewTeammateStore(task, jm, t.TempDir())
	t.Cleanup(ts.Close)

	c := New(Options{
		Executor:     executor,
		Jobs:         jm,
		Teammates:    ts,
		Sink:         event.Discard,
		SystemPrompt: "sys",
	})
	c.Close()
	// No explicit assertion: the goleak verifier (TestMain) proves the
	// auto-advance worker exited with Close. A leaked worker fails the package.
}
