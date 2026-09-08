package boot

import (
	"context"
	"reflect"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
)

// topLevelToolGolden is the provider-visible tool surface every session pays
// for on every turn. It must stay byte-stable: adding or removing a top-level
// tool rewrites the cached prompt prefix and re-bills whole conversations, so
// this list changes only deliberately (and with a Cache-impact: high note).
// Optional capabilities (team orchestration, MCP, task) live behind
// use_capability and never enter this surface.
var topLevelToolGolden = []string{
	"ask", "bash", "bash_output", "complete_step", "compress", "edit_file",
	"kill_shell", "read_file", "todo_write", "update_goal", "use_capability",
	"wait", "write_file",
}

// TestTopLevelToolSurfaceGolden locks the top-level surface against silent
// growth: a new top-level tool invalidates the cached prefix for every
// session, so it must be a reviewed, deliberate change.
func TestTopLevelToolSurfaceGolden(t *testing.T) {
	isolateConfigHome(t)
	dir := robustTempDir(t)
	t.Chdir(dir)
	rec := &effectRecordingProvider{}
	provider.Register("boot-surface-golden", func(provider.Config) (provider.Provider, error) {
		return rec, nil
	})
	writeFile(t, dir, "reasonix.toml", `
default_model = "test-model"

[agent]
system_prompt = "BASE"

[environment]
enabled = false

[[providers]]
name = "test-model"
kind = "boot-surface-golden"
model = "x"
`)
	ctrl, err := Build(context.Background(), Options{Sink: event.Discard})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer ctrl.Close()
	if err := ctrl.Run(context.Background(), "reply ok"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rec.reqs) == 0 {
		t.Fatal("no provider request recorded")
	}
	got := toolSchemaNames(rec.reqs[0].Tools)
	if !reflect.DeepEqual(got, topLevelToolGolden) {
		t.Fatalf("top-level tool surface changed — this rewrites the cached prompt prefix for every session.\n got: %v\nwant: %v\nIf deliberate, update topLevelToolGolden and note Cache-impact: high.", got, topLevelToolGolden)
	}
}
