package agent

import (
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
)

// TestFitFoldToWindowKeepsFoldBounded pins the manual /compact dead stall on
// a session whose fold estimate dwarfs the window: the second bound loop
// compared moved tokens against maxCompactFoldTokens (false on entry), ran to
// cut=0, emptied the fold, and planFold returned Noop — "compacted" with zero
// summarizer calls (user-observed 2026-08-09).
func TestFitFoldToWindowKeepsFoldBounded(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "initial request"},
		{Role: provider.RoleAssistant, Content: "work"},
	}
	// ReasoningContent inflates the estimate (818K in the real session); each
	// part stays under maxCompactFoldTokens so the first loop stops mid-fold.
	for i := 0; i < 40; i++ {
		id := string(rune('a' + i))
		msgs = append(msgs,
			provider.Message{Role: provider.RoleAssistant, Content: "", ReasoningContent: strings.Repeat("r", 45_000), ToolCalls: []provider.ToolCall{{ID: id, Name: "read_file", Arguments: "{}"}}},
			provider.Message{Role: provider.RoleTool, ToolCallID: id, Name: "read_file", Content: strings.Repeat("t", 8_000)},
		)
	}
	a := &Agent{
		prov:              &capturingBudgetProvider{sharedFakeProvider: sharedFakeProvider{fakeProvider: &fakeProvider{reply: "SUMMARY"}, budget: 128 * 1024}},
		contextWindow:     1_000_000,
		outputBudgetState: outputBudgetState{outputBudget: 128 * 1024},
		compactRatio:      0.8,
		sink:              event.Discard,
	}
	a.session = &Session{Messages: msgs}
	outcome, err := a.compactToProjection(t.Context(), "manual", "", true, false)
	if err != nil {
		t.Fatalf("manual compact: %v", err)
	}
	if outcome != CompactionInstalled {
		t.Fatalf("outcome=%v, want Installed: fold must survive the bound loop", outcome)
	}
}
