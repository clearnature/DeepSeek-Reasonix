package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// pruneObservation records whether a prune pass ran before the fold, so the
// manual-compaction ordering can be asserted without a live provider.
type pruneObservation struct {
	requests []provider.Request
}

func (p *pruneObservation) Name() string { return "prune-observation" }

func (p *pruneObservation) Stream(_ context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.requests = append(p.requests, req)
	return chunks(
		provider.Chunk{Type: provider.ChunkText, Text: "## Goal\nsummarised"},
		provider.Chunk{Type: provider.ChunkUsage, Usage: &provider.Usage{PromptTokens: 1000, CompletionTokens: 50, TotalTokens: 1050, RequestCount: 1}},
		provider.Chunk{Type: provider.ChunkDone},
	), nil
}

// toolHeavySession builds a view dominated by oversized tool results, which is
// what prune can reclaim. Bodies must exceed the provider-visible Content limit
// (see pruneToolResultsToProjectionLocked) or there is nothing to prune.
func toolHeavySession(turns int) *Session {
	msgs := []provider.Message{{Role: provider.RoleSystem, Content: "system prompt"}}
	for i := range turns {
		body := strings.Repeat(fmt.Sprintf("stale tool output %d. ", i), 2000)
		msgs = append(msgs,
			provider.Message{Role: provider.RoleUser, Content: fmt.Sprintf("turn %d", i)},
			provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: fmt.Sprintf("c%d", i), Name: "read_file", Arguments: "{}"}}},
			provider.Message{Role: provider.RoleTool, ToolCallID: fmt.Sprintf("c%d", i), Name: "read_file", Content: body, RawContent: body},
		)
	}
	return &Session{Messages: msgs}
}

// TestManualCompactPrunesBeforeFold pins the ordering fixed for manual
// compaction: /compact on a large view must reclaim stale tool results first.
// Summarising the raw fold drives the 8192-token digest to its ceiling and
// drops the run into the full-price fragment path; pruning first leaves the
// summarizer a fold small enough to describe.
func TestManualCompactPrunesBeforeFold(t *testing.T) {
	probe := &pruneObservation{}
	a := New(probe, tool.NewRegistry(), toolHeavySession(160), Options{
		ContextWindow: 200_000,
		CompactRatio:  0.8,
		ArchiveDir:    t.TempDir(),
	}, event.Discard)

	before := a.currentProjectionVersion()
	beforeTokens := a.ContextUsedTokens()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := a.CompactNow(ctx, ""); err != nil {
		t.Fatalf("CompactNow = %v", err)
	}

	afterVersion := a.currentProjectionVersion()
	afterTokens := a.ContextUsedTokens()
	largestSummaryPrompt := 0
	summaries := 0
	for _, req := range probe.requests {
		if !isSummaryRequest(req) {
			continue
		}
		summaries++
		if n := denseTokens(req); n > largestSummaryPrompt {
			largestSummaryPrompt = n
		}
	}
	t.Logf("manual compact: version %d→%d tokens %d→%d summaryCalls=%d largestSummaryPrompt=%d",
		before, afterVersion, beforeTokens, afterTokens, summaries, largestSummaryPrompt)

	if afterVersion == before {
		t.Fatalf("manual compact installed nothing: version stayed %d", before)
	}
	if afterTokens >= beforeTokens {
		t.Fatalf("manual compact did not reclaim: %d → %d", beforeTokens, afterTokens)
	}
	// The point of pruning first: the summarizer must never be handed the raw
	// view, which is what drives its digest into truncation and the fragment
	// fallback.
	if largestSummaryPrompt >= beforeTokens/2 {
		t.Fatalf("summary ran on an unpruned view: largest summary prompt %d vs pre-compact %d",
			largestSummaryPrompt, beforeTokens)
	}
}
