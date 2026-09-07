package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestFastCompressManualElidesStaleToolResults locks in the /compress-fast
// contract: manual elision rewrites the model-visible projection (no API
// call), never the canonical transcript, and reports the affected count.
func TestFastCompressManualElidesStaleToolResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	bigTool := strings.Repeat("界", toolPruneThresholdRunes+1)
	sess := &Session{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "t1", Name: "read_file", Arguments: "{}"}}},
		{Role: provider.RoleTool, ToolCallID: "t1", Name: "read_file", Content: bigTool},
		{Role: provider.RoleAssistant, Content: "work"},
		{Role: provider.RoleUser, Content: "tail"},
	}}
	a := New(nil, tool.NewRegistry(), sess, Options{ContextWindow: 60_000, SessionPath: path}, event.Discard)

	stats, err := a.FastCompressManual()
	if err != nil {
		t.Fatalf("FastCompressManual: %v", err)
	}
	if stats.Results != 1 {
		t.Fatalf("elided results = %d, want 1", stats.Results)
	}
	a.sess.compactionMu.Lock()
	proj := a.sess.compactionState.Projection
	a.sess.compactionMu.Unlock()
	if proj.ProjectionVersion == 0 {
		t.Fatal("prune projection was not installed")
	}
	if proj.Messages[3].Content == bigTool {
		t.Fatal("projection still carries the full tool result")
	}
	for _, m := range sess.Snapshot() {
		if m.Role == provider.RoleTool && m.Content != bigTool {
			t.Fatal("canonical tool result was rewritten by manual prune")
		}
	}

	again, err := a.FastCompressManual()
	if err != nil {
		t.Fatalf("second FastCompressManual: %v", err)
	}
	if again.Results != 0 {
		t.Fatalf("second pass elided %d, want 0 (projection already pruned)", again.Results)
	}
}
