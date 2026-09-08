package agent

import (
	"context"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

func TestPressureCompactionDoesNotCallChunkedFold(t *testing.T) {
	prov := &extractStubProvider{failFirst: 64, reply: "digest"}
	a := agentOverForce(t, prov, foldableSessionOverForce(12))
	err := prepareContext(context.Background(), a, CompactionTriggerPressure)
	if err == nil {
		t.Fatal("truncated summary must fail without installing a chunked projection")
	}
	if degradedFold(a) {
		t.Fatal("pressure compaction must not install a fabricated summary")
	}
	if prov.calls > 2 {
		t.Fatalf("provider calls = %d, want at most one summary plus one retry, not chunked/tree-reduce", prov.calls)
	}
}

func TestExplicitChunkedFallbackStillRuns(t *testing.T) {
	prov := &extractStubProvider{failFirst: 1, reply: "digest"}
	a := New(prov, tool.NewRegistry(), extractStubSession(), Options{}, event.Discard)
	fold := a.Session().Snapshot()
	if _, _, err := a.foldSummaryWithChunkedFallback(context.Background(), CompactionTriggerManual, nil, fold, "focus", 321, SummaryInputCachePrefix); err != nil {
		t.Fatalf("explicit chunked fallback: %v", err)
	}
	if prov.calls < 2 {
		t.Fatalf("provider calls = %d, want the failed summary plus chunked recovery", prov.calls)
	}
}

// contextLimitOnceProvider rejects the first summarize with a 400 context-
// length overflow (exactly what a >window fold hits) and succeeds after,
// proving the chunked fallback engages on provider context-limit errors.
type contextLimitOnceProvider struct {
	calls int
}

func (p *contextLimitOnceProvider) Name() string { return "context-limit-once" }

func (p *contextLimitOnceProvider) Stream(_ context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	p.calls++
	ch := make(chan provider.Chunk, 2)
	if p.calls == 1 {
		ch <- provider.Chunk{Type: provider.ChunkError, Err: &provider.ContextLimitError{APIError: &provider.APIError{Status: 400, Body: "maximum context length is 1048576 tokens, requested 1167124"}}}
	} else {
		ch <- provider.Chunk{Type: provider.ChunkText, Text: "digest"}
	}
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}

func TestContextLimitOverflowFallsBackToChunkedFold(t *testing.T) {
	prov := &contextLimitOnceProvider{}
	a := New(prov, tool.NewRegistry(), extractStubSession(), Options{}, event.Discard)
	fold := a.Session().Snapshot()
	if _, _, err := a.foldSummaryWithChunkedFallback(context.Background(), CompactionTriggerManual, nil, fold, "focus", 321, SummaryInputCachePrefix); err != nil {
		t.Fatalf("chunked fallback after context-limit overflow: %v", err)
	}
	if prov.calls < 2 {
		t.Fatalf("provider calls = %d, want the 400 attempt plus chunked recovery", prov.calls)
	}
}
