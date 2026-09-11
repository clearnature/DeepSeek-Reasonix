package control

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestCreateSubSessionWithoutSpawnerReportsDaemonOnly locks the qwen
// daemon-only contract: with no session bridge the tool fails with the
// documented message instead of pretending to work.
func TestCreateSubSessionWithoutSpawnerReportsDaemonOnly(t *testing.T) {
	tt := NewCreateSubSessionTool(nil)
	if _, err := tt.Execute(context.Background(), json.RawMessage(`{"prompt":"hi"}`)); err == nil ||
		!strings.Contains(err.Error(), "only available when running under `reasonix serve`") {
		t.Fatalf("err = %v, want daemon-only message", err)
	}
}

type fakeSpawner struct {
	gotPrompt, gotMode string
}

func (f *fakeSpawner) SpawnSubSession(_ context.Context, prompt, completion string) (string, error) {
	f.gotPrompt, f.gotMode = prompt, completion
	return "spawned " + completion, nil
}

// TestCreateSubSessionDelegatesToSpawner locks the bridge contract: prompt and
// completion mode reach the spawner, defaulting to first-turn.
func TestCreateSubSessionDelegatesToSpawner(t *testing.T) {
	sp := &fakeSpawner{}
	tt := NewCreateSubSessionTool(sp)
	out, err := tt.Execute(context.Background(), json.RawMessage(`{"prompt":"do work"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sp.gotPrompt != "do work" || sp.gotMode != "first-turn" {
		t.Fatalf("spawner got (%q,%q), want (do work, first-turn)", sp.gotPrompt, sp.gotMode)
	}
	if !strings.Contains(out, "first-turn") {
		t.Fatalf("out = %q", out)
	}
	if _, err := tt.Execute(context.Background(), json.RawMessage(`{"prompt":"x","completion":"bogus"}`)); err == nil {
		t.Fatal("bogus completion should be rejected")
	}
	var _ = tt
}
