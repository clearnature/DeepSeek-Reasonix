package agent

import (
	"context"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// Automatic prune/snip projections are gone from the auto path; the manual
// rescue command /compress-fast (PruneStaleToolResults) elides stale tool
// results into placeholders without a summarizer call. SnipStaleToolResults
// stays a no-op (auto snip is gone). The canonical transcript is untouched;
// only the projection view changes.
func TestPruneAndSnipAreNoOps(t *testing.T) {
	sess := &Session{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "t1", Name: "read_file", Arguments: "{}"}}},
		{Role: provider.RoleTool, ToolCallID: "t1", Name: "read_file", Content: strings.Repeat("x", 8000)},
		{Role: provider.RoleUser, Content: "next"},
	}}
	a := New(nil, tool.NewRegistry(), sess, Options{ContextWindow: 100_000, RecentKeep: 2}, event.Discard)
	st, err := a.PruneStaleToolResults()
	if err != nil {
		t.Fatal(err)
	}
	if st.Results != 1 {
		t.Fatalf("prune results = %d, want 1 (stale tool result elided)", st.Results)
	}
	st, err = a.SnipStaleToolResults()
	if err != nil {
		t.Fatal(err)
	}
	if st.Results != 0 {
		t.Fatalf("snip results = %d, want 0 (auto snip is gone)", st.Results)
	}
	for _, m := range sess.Snapshot() {
		if m.Role == provider.RoleTool && m.Content != strings.Repeat("x", 8000) {
			t.Fatal("canonical tool result was rewritten")
		}
	}
}

// At the sole compact_ratio trigger, maintenance calls the summarizer once —
// it does not install an intermediate prune projection.
func TestMaintenanceUsesSummaryNotPruneAtFoldTrigger(t *testing.T) {
	big := strings.Repeat("tool body line\n", 400)
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
	}
	for i := range 12 {
		id := "t" + string(rune('a'+i))
		msgs = append(msgs,
			provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: id, Name: "read_file", Arguments: "{}"}}},
			provider.Message{Role: provider.RoleTool, ToolCallID: id, Name: "read_file", Content: big},
		)
	}
	msgs = append(msgs,
		provider.Message{Role: provider.RoleUser, Content: "tail"},
		provider.Message{Role: provider.RoleAssistant, Content: "ok"},
	)
	prov := &countingProvider{reply: "digest"}
	a := New(prov, tool.NewRegistry(), &Session{Messages: msgs}, Options{
		ContextWindow: 20_000, CompactRatio: 0.5, RecentKeep: 2,
	}, event.Discard)
	if _, err := a.contextManager().Prepare(context.Background(), ContextPreparePolicy{Trigger: CompactionTriggerPressure}); err != nil {
		t.Fatal(err)
	}
	if a.currentProjectionVersion() != 1 {
		t.Fatalf("projection version = %d, want 1", a.currentProjectionVersion())
	}
	if len(prov.got) != 1 {
		t.Fatalf("summarizer calls = %d, want 1", len(prov.got))
	}
	if got := countToolResultsWithPrefix(a.modelVisibleMessages(), prunedMarker); got != 0 {
		t.Fatalf("prune markers in projection = %d, want 0", got)
	}
}

func TestSnipStrategyStillAvailableForFirstVisibleAndSummaryInput(t *testing.T) {
	a := &Agent{tools: tool.NewRegistry()}
	s := a.snipStrategyFor("read_file")
	if s.head <= 0 || s.tail <= 0 {
		t.Fatalf("snip strategy for read_file = %+v", s)
	}
	body, notice := truncateToolOutputFor(strings.Repeat("x", maxToolOutputBytes+100), "read_file", "call-1")
	if notice == "" || !strings.Contains(body, "call_id=call-1") {
		t.Fatalf("first-visible truncation missing marker: notice=%q body=%.200q", notice, body)
	}
	if len(body) > maxToolOutputBytes+200 {
		t.Fatalf("bounded body still oversized: %d", len(body))
	}
}
