package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"reasonix/internal/provider"
)

// TestSidecarWireBytesSurviveRoundTrip pins the lossless projection inverse:
// the frozen main-request bytes stored in the sidecar must survive the JSON
// round trip byte-exact, so a resumed process can replay the same prefix.
func TestSidecarWireBytesSurviveRoundTrip(t *testing.T) {
	wire := []provider.Message{
		{Role: provider.RoleSystem, Content: "system"},
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "t1", Name: "read", Arguments: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "t1", Name: "read", Content: "result"},
		{Role: provider.RoleUser, Content: "u2"},
	}
	st := CompactionState{LastWireMessages: wire}
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back CompactionState
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.LastWireMessages) != len(wire) {
		t.Fatalf("round-trip messages = %d, want %d", len(back.LastWireMessages), len(wire))
	}
	for i := range wire {
		m, w := back.LastWireMessages[i], wire[i]
		if m.Role != w.Role || m.Content != w.Content || m.ToolCallID != w.ToolCallID || m.Name != w.Name {
			t.Fatalf("message %d diverged after round trip: %+v vs %+v", i, m, w)
		}
		if len(m.ToolCalls) != len(w.ToolCalls) {
			t.Fatalf("message %d tool calls diverged: %d vs %d", i, len(m.ToolCalls), len(w.ToolCalls))
		}
		for j := range w.ToolCalls {
			if m.ToolCalls[j].ID != w.ToolCalls[j].ID || m.ToolCalls[j].Name != w.ToolCalls[j].Name || m.ToolCalls[j].Arguments != w.ToolCalls[j].Arguments {
				t.Fatalf("message %d tool call %d diverged", i, j)
			}
		}
	}
}

// TestLoadProjectionSidecarRestoresWireBytes simulates a resume: a sidecar
// written by a parent process (with LastWireMessages) is loaded by a fresh
// agent, and the frozen prefix is restored so the first post-resume compaction
// replays the provider-cached unit instead of falling back to system-only.
func TestLoadProjectionSidecarRestoresWireBytes(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")
	wire := []provider.Message{
		{Role: provider.RoleSystem, Content: "system prompt"},
		{Role: provider.RoleUser, Content: "user one"},
		{Role: provider.RoleAssistant, Content: "assistant one"},
		{Role: provider.RoleUser, Content: "user two"},
	}
	canonical := append([]provider.Message(nil), wire...)
	covered := len(canonical)
	st := CompactionState{
		SchemaVersion:    compactionStateSchemaCurrent,
		LastWireMessages: wire,
		LastReceipt:      &ContextMaintenanceReceipt{Status: "applied"},
		PromptCacheKey:   "lineage",
		Projection: ContextProjection{
			Messages: []provider.Message{
				{Role: provider.RoleSystem, Content: "system prompt"},
				{Role: provider.RoleUser, Content: "SUMMARY: everything was folded"},
			},
			CoveredCount:      covered,
			CoveredPrefixHash: coveredPrefixHash(canonical, covered),
		},
	}
	if err := SaveCompactionState(sessionPath, st); err != nil {
		t.Fatalf("save sidecar: %v", err)
	}

	// Fresh process: new agent, no frozen bytes.
	prov := &countingProvider{reply: "digest"}
	a := newFoldAgent(t, 200000, prov)
	if got := a.savedMainRequest(); len(got) != 0 {
		t.Fatalf("fresh agent must start without frozen bytes, got %d", len(got))
	}
	a.sess.conversation = &Session{Messages: canonical}
	a.LoadProjectionSidecar(sessionPath)

	restored := a.savedMainRequest()
	if len(restored) != len(wire) {
		t.Fatalf("restored wire bytes = %d messages, want %d", len(restored), len(wire))
	}
	for i := range wire {
		if restored[i].Role != wire[i].Role || restored[i].Content != wire[i].Content {
			t.Fatalf("restored message %d diverged: %+v vs %+v", i, restored[i], wire[i])
		}
	}

	// First post-resume compaction must replay the restored bytes, not the
	// cropped fallback.
	prefix, extra, anchors := a.summaryFoldPlan(canonical, 1, len(canonical)-1)
	if len(prefix) != len(wire) {
		t.Fatalf("first post-resume prefix = %d messages, want the restored %d", len(prefix), len(wire))
	}
	for i := range wire {
		if prefix[i].Content != wire[i].Content {
			t.Fatalf("prefix message %d = %q, want restored %q", i, prefix[i].Content, wire[i].Content)
		}
	}
	if len(extra) != 0 || anchors == "" {
		t.Fatalf("extra=%d anchors=%q, want fold located by anchors, not re-sent", len(extra), anchors)
	}
}

// TestLoadProjectionSidecarWithoutWireBytesFallsBack covers old sidecars that
// predate the lossless field: no restore, cropped fallback stays.
func TestLoadProjectionSidecarWithoutWireBytesFallsBack(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.jsonl")
	canonical := []provider.Message{
		{Role: provider.RoleSystem, Content: "system"},
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "u2"},
	}
	st := CompactionState{
		SchemaVersion:  compactionStateSchemaCurrent,
		LastReceipt:    &ContextMaintenanceReceipt{Status: "applied"},
		PromptCacheKey: "lineage",
		Projection: ContextProjection{
			Messages: []provider.Message{
				{Role: provider.RoleSystem, Content: "system"},
				{Role: provider.RoleUser, Content: "SUMMARY"},
			},
			CoveredCount:      3,
			CoveredPrefixHash: coveredPrefixHash(canonical, 3),
		},
	}
	if err := SaveCompactionState(sessionPath, st); err != nil {
		t.Fatalf("save sidecar: %v", err)
	}
	a := newFoldAgent(t, 200000, &countingProvider{reply: "digest"})
	a.sess.conversation = &Session{Messages: canonical}
	a.LoadProjectionSidecar(sessionPath)
	if got := a.savedMainRequest(); len(got) != 0 {
		t.Fatalf("legacy sidecar must not restore bytes, got %d", len(got))
	}
	os.Remove(sessionPath)
}
