package agent

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// countingProvider records every summarizer call so tests can assert that a
// fold costs exactly one request.
type countingProvider struct {
	reply string
	got   []provider.Request
}

type deadlineInspectProvider struct {
	hadDeadline bool
}

type summaryChunksProvider struct {
	chunks []provider.Chunk
}

func (p *summaryChunksProvider) Name() string { return "summary-chunks" }
func (p *summaryChunksProvider) Stream(context.Context, provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, len(p.chunks))
	for _, chunk := range p.chunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

func (p *deadlineInspectProvider) Name() string { return "deadline-inspect" }
func (p *deadlineInspectProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	_, p.hadDeadline = ctx.Deadline()
	ch := make(chan provider.Chunk, 1)
	ch <- provider.Chunk{Type: provider.ChunkText, Text: "digest"}
	close(ch)
	return ch, nil
}

func TestSummaryDoesNotAddInternalWallClockDeadline(t *testing.T) {
	prov := &deadlineInspectProvider{}
	a := New(prov, tool.NewRegistry(), &Session{Messages: []provider.Message{{Role: provider.RoleSystem, Content: "sys"}}}, Options{}, event.Discard)
	if _, err := a.foldToSummary(context.Background(), nil, []provider.Message{{Role: provider.RoleUser, Content: "old"}}, ""); err != nil {
		t.Fatal(err)
	}
	if prov.hadDeadline {
		t.Fatal("summary provider context unexpectedly has an internal deadline")
	}
}

func TestSummaryCollectorStoresOnlyVisibleText(t *testing.T) {
	prov := &summaryChunksProvider{chunks: []provider.Chunk{
		{Type: provider.ChunkReasoning, Text: "PRIVATE REASONING"},
		{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "call-1", Name: "read_file", Arguments: `{}`}},
		{Type: provider.ChunkText, Text: "VISIBLE DIGEST"},
		{Type: provider.ChunkDone},
	}}
	a := New(prov, tool.NewRegistry(), NewSession("system"), Options{}, event.Discard)
	got, _, err := a.summarize(context.Background(), nil, []provider.Message{{Role: provider.RoleUser, Content: "old"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "VISIBLE DIGEST" {
		t.Fatalf("summary = %q, want visible text only", got)
	}
}

func TestSummaryCollectorRejectsEmptyAndLengthLimitedOutput(t *testing.T) {
	for _, tc := range []struct {
		name   string
		chunks []provider.Chunk
		want   string
	}{
		{
			name: "reasoning and tool call are empty",
			chunks: []provider.Chunk{
				{Type: provider.ChunkReasoning, Text: "PRIVATE REASONING"},
				{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "call-1", Name: "read_file"}},
			},
			want: "empty output",
		},
		{
			name: "length finish accepts partial output",
			chunks: []provider.Chunk{
				{Type: provider.ChunkText, Text: "partial"},
				{Type: provider.ChunkUsage, Usage: &provider.Usage{FinishReason: "length"}},
			},
			want: "partial",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prov := &summaryChunksProvider{chunks: tc.chunks}
			a := New(prov, tool.NewRegistry(), NewSession("system"), Options{}, event.Discard)
			result, _, err := a.summarize(context.Background(), nil, []provider.Message{{Role: provider.RoleUser, Content: "old"}}, "")
			if tc.want == "partial" {
				// length finish now accepts partial output
				if err != nil {
					t.Fatalf("summarize error = %v, want success", err)
				}
				if !strings.Contains(result, tc.want) {
					t.Fatalf("summarize result = %q, want %q", result, tc.want)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("summarize error = %v, want %q", err, tc.want)
				}
			}
		})
	}
}

func TestSummaryRequestReplaysSystemToolsAndSelectedPrefix(t *testing.T) {
	prov := &countingProvider{reply: "digest"}
	reg := tool.NewRegistry()
	reg.Add(echoTool{})
	system := provider.Message{Role: provider.RoleSystem, Content: "stable system", CreatedAt: 11}
	fold := []provider.Message{
		{Role: provider.RoleUser, Content: "old task", CreatedAt: 12},
		{Role: provider.RoleAssistant, Content: "old work", CreatedAt: 13},
	}
	a := New(prov, reg, &Session{Messages: append([]provider.Message{system}, fold...)}, Options{ContextWindow: 100_000, MaxOutputTokens: 1024}, event.Discard)

	if _, err := a.foldToSummary(context.Background(), nil, fold, "keep exact identifiers"); err != nil {
		t.Fatalf("foldToSummary: %v", err)
	}
	if len(prov.got) != 1 {
		t.Fatalf("summary requests = %d, want 1", len(prov.got))
	}
	req := prov.got[0]
	if len(req.Messages) != 4 {
		t.Fatalf("summary messages = %d, want system + 2 prefix messages + instruction", len(req.Messages))
	}
	wantPrefix := []provider.Message{system, fold[0], fold[1]}
	for i := range wantPrefix {
		wantPrefix[i].CreatedAt = 0
		if !reflect.DeepEqual(req.Messages[i], wantPrefix[i]) {
			t.Fatalf("prefix message %d = %+v, want %+v", i, req.Messages[i], wantPrefix[i])
		}
	}
	last := req.Messages[len(req.Messages)-1]
	if last.Role != provider.RoleUser || !strings.Contains(last.Content, "keep exact identifiers") || !strings.Contains(last.Content, "Do not call tools") {
		t.Fatalf("final compaction instruction = %+v", last)
	}
	if len(req.Tools) != 1 || req.Tools[0].Name != "echo" {
		t.Fatalf("summary tools = %+v, want normal echo schema", req.Tools)
	}
	if req.MaxTokens != summaryOutputMaxTokens {
		t.Fatalf("summary max tokens = %d, want fixed cap %d", req.MaxTokens, summaryOutputMaxTokens)
	}
}

func (p *countingProvider) Name() string { return "counting" }

func (p *countingProvider) Stream(_ context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.got = append(p.got, req)
	ch := make(chan provider.Chunk, 2)
	ch <- provider.Chunk{Type: provider.ChunkText, Text: fmt.Sprintf("%s %d", p.reply, len(p.got))}
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}

func foldOfToolResults(n, size int) []provider.Message {
	fold := make([]provider.Message, 0, n*2)
	for i := range n {
		fold = append(fold,
			provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: fmt.Sprint(i), Name: "read_file", Arguments: "{}"}}},
			provider.Message{Role: provider.RoleTool, ToolCallID: fmt.Sprint(i), Name: "read_file", Content: strings.Repeat(fmt.Sprintf("line %d filler\n", i), size)},
		)
	}
	return fold
}

func newFoldAgent(t *testing.T, window int, prov provider.Provider) *Agent {
	t.Helper()
	return New(prov, nil, &Session{}, Options{ContextWindow: window}, event.Discard)
}

func TestFoldUnderBudgetIsSummarizedVerbatimInOneCall(t *testing.T) {
	prov := &countingProvider{reply: "digest"}
	a := newFoldAgent(t, 200000, prov)
	fold := foldOfToolResults(3, 40)

	res, err := a.foldToSummary(context.Background(), nil, fold, "")
	if err != nil {
		t.Fatalf("foldToSummary: %v", err)
	}
	if len(prov.got) != 1 || res.Spans != 1 {
		t.Fatalf("requests=%d spans=%d, want a single call", len(prov.got), res.Spans)
	}
	if body := joinContents(prov.got[0].Messages); strings.Contains(body, snippedMarker) {
		t.Fatal("an under-budget fold must reach the summarizer unshortened")
	}
}

func TestManualFoldDoesNotPrivatelyShortenToolResults(t *testing.T) {
	prov := &countingProvider{reply: "digest"}
	a := newFoldAgent(t, 24000, prov)
	fold := foldOfToolResults(6, 300)

	res, err := a.foldToSummary(context.Background(), nil, fold, "")
	if err != nil {
		t.Fatalf("foldToSummary: %v", err)
	}
	if len(prov.got) != 1 || res.Spans != 1 {
		t.Fatalf("requests=%d spans=%d, want exactly one call", len(prov.got), res.Spans)
	}
	body := joinContents(prov.got[0].Messages)
	if strings.Contains(body, snippedMarker) || strings.Contains(body, toolPruneMarker) {
		t.Fatalf("manual summary input was privately pruned:\n%.300q", body)
	}
	if !strings.Contains(body, "line 5 filler") {
		t.Fatalf("complete tool results did not reach summarizer:\n%.300q", body)
	}
}

func TestHugeFoldNeverMultiSpan(t *testing.T) {
	// Even a very large fold gets at most one complete-prefix provider request.
	// If it cannot fit, the transaction fails rather than shortening or splitting.
	prov := &countingProvider{reply: "digest"}
	a := newFoldAgent(t, 32000, prov)
	fold := foldOfToolResults(80, 800)

	res, err := a.foldToSummary(context.Background(), nil, fold, "focus on the parser")
	if err != nil {
		// Failure without a second attempt is acceptable for an unfittable fold.
		if len(prov.got) != 0 {
			t.Fatalf("failed fold still made %d provider requests", len(prov.got))
		}
		return
	}
	if len(prov.got) != 1 || res.Spans != 1 {
		t.Fatalf("requests=%d spans=%d, want at most one call", len(prov.got), res.Spans)
	}
	if !strings.Contains(prov.got[0].Messages[len(prov.got[0].Messages)-1].Content, "focus on the parser") {
		t.Fatal("focus instructions lost")
	}
}

func TestNoContextWindowLeavesTheFoldUnbounded(t *testing.T) {
	prov := &countingProvider{reply: "digest"}
	a := New(prov, nil, &Session{}, Options{}, event.Discard)
	fold := foldOfToolResults(40, 400)

	res, err := a.foldToSummary(context.Background(), nil, fold, "")
	if err != nil {
		// Without a window the input budget is 0 and the single-call path
		// refuses before paying for a request.
		if len(prov.got) != 0 {
			t.Fatalf("no-window failure still called provider %d times", len(prov.got))
		}
		return
	}
	if len(prov.got) != 1 || res.Spans != 1 {
		t.Fatalf("requests=%d spans=%d, want one unbounded call", len(prov.got), res.Spans)
	}
}

func TestSummarizeOnceNoRetry(t *testing.T) {
	prov := &failOnceProvider{}
	a := newFoldAgent(t, 200000, prov)
	_, _, err := a.summarizeOnce(context.Background(), nil, []provider.Message{
		{Role: provider.RoleUser, Content: "hello"},
	}, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if prov.calls != 1 {
		t.Fatalf("provider calls = %d, want exactly 1 (no application-layer retry)", prov.calls)
	}
}

type failOnceProvider struct{ calls int }

func (p *failOnceProvider) Name() string { return "fail-once" }

func (p *failOnceProvider) Stream(_ context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	p.calls++
	ch := make(chan provider.Chunk, 1)
	ch <- provider.Chunk{Type: provider.ChunkError, Err: fmt.Errorf("network glitch")}
	close(ch)
	return ch, nil
}

func TestMaximumSafeSummaryPrefixEndTrimsOversizedFold(t *testing.T) {
	// Manual compaction with an oversized fold (2026-08-30: 1.15M fold vs a
	// 1,048,576-token model) used to reach the provider as-is when mustFree
	// was false (manual trigger, under-estimated input) → provider 400. The
	// fold region must now be trimmed to the window's physical input ceiling
	// on every path.
	prov := &countingProvider{reply: "digest"}
	a := newFoldAgent(t, 100000, prov)
	fold := foldOfToolResults(300, 400) // ~120k tokens, window 100k
	msgs := append([]provider.Message{}, fold...)
	adm := a.lastAdmission()
	adm.ObservedWindow = 100000
	a.storeAdmission(adm)
	head, start, ok := a.planFoldRegion(msgs, false)
	if !ok || head >= start {
		t.Fatalf("planFoldRegion ok=%v head=%d start=%d", ok, head, start)
	}
	end := a.maximumSafeSummaryPrefixEnd(msgs, head, start, "")
	if end >= start {
		t.Fatalf("oversized fold not trimmed: end=%d start=%d", end, start)
	}
	if end > head {
		// Trimmed fold must still fit the summary input budget, measured with
		// the real request shape: verbatim head precedes the fold region.
		folded := msgs[head:end]
		req := a.summaryRequest(msgs[0:head], folded, "")
		if est := a.estimatedRequestTokens(req); est > a.hardInputCeiling() {
			t.Fatalf("trimmed fold est=%d exceeds hard input ceiling %d", est, a.hardInputCeiling())
		}
	}
}

func TestSummaryRequestForcesNoReasoningEffort(t *testing.T) {
	// Summary requests must not inherit the user's reasoning effort:
	// DeepSeek thinking would consume the 8192 output budget and truncate
	// the digest (errSummaryOutputTruncated), failing compaction.
	a := &Agent{}
	req := a.summaryRequest(nil, []provider.Message{{Role: provider.RoleUser, Content: "x"}}, "")
	if req.EffortOverride != "none" {
		t.Fatalf("summaryRequest EffortOverride = %q, want none", req.EffortOverride)
	}
	if req.MaxTokens != summaryOutputMaxTokens {
		t.Fatalf("summaryRequest MaxTokens = %d, want %d", req.MaxTokens, summaryOutputMaxTokens)
	}
}

type compactionBudgetProvider struct {
	provider.Provider
	budget int
}

func (p compactionBudgetProvider) CompactionOutputTokens() int { return p.budget }

func TestSummaryOutputBudgetUsesVendorCompactionTokens(t *testing.T) {
	a := &Agent{}
	if got := a.summaryOutputBudget(); got != summaryOutputMaxTokens {
		t.Fatalf("default budget = %d, want %d", got, summaryOutputMaxTokens)
	}
	a.svc.prov = compactionBudgetProvider{budget: 16384}
	if got := a.summaryOutputBudget(); got != 16384 {
		t.Fatalf("vendor budget = %d, want 16384", got)
	}
}

func TestForkCaptureProviderPassesCompactionBudget(t *testing.T) {
	f := &forkCaptureProvider{inner: compactionBudgetProvider{budget: 16384}}
	if got := f.CompactionOutputTokens(); got != 16384 {
		t.Fatalf("fork CompactionOutputTokens = %d, want 16384", got)
	}
	f = &forkCaptureProvider{inner: compactionBudgetProvider{}}
	if got := f.CompactionOutputTokens(); got != 0 {
		t.Fatalf("non-provider inner = %d, want 0", got)
	}
}

func TestCompactToProjectionTrimsOversizedFoldEvenWithoutMustFree(t *testing.T) {
	// Regression for the manual-compaction 400: compactToProjection with
	// mustFree=false (the pre-fix manual-trigger path) must still bound the
	// fold input instead of sending an oversized summary request.
	prov := &countingProvider{reply: "digest"}
	a := newFoldAgent(t, 100000, prov)
	fold := foldOfToolResults(300, 400)
	msgs := append([]provider.Message{}, fold...)
	adm := a.lastAdmission()
	adm.ObservedWindow = 100000
	a.storeAdmission(adm)
	a.sess.conversation = &Session{Messages: msgs}
	_, err := a.compactToProjection(context.Background(), CompactionTriggerManual, "", false, false)
	if err != nil {
		// Trimmed to nothing is acceptable (explicit rejection); a provider
		// request for an oversized fold is not.
		if len(prov.got) != 0 {
			t.Fatalf("failed compaction still sent %d provider requests", len(prov.got))
		}
		return
	}
	if len(prov.got) != 1 {
		t.Fatalf("requests=%d, want exactly one bounded summary call", len(prov.got))
	}
	if est := a.estimatedVisibleRequestTokens(prov.got[0].Messages); est > a.hardInputCeiling() {
		t.Fatalf("summary request est=%d exceeds hard input ceiling %d", est, a.hardInputCeiling())
	}
}

func TestSummaryRequestPrefixMatchesOrdinaryRequestBytes(t *testing.T) {
	// Summary must reproduce the ordinary request's byte prefix; divergence = full-price cache miss (2026-08-30: 3.6% hit).
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
		{Role: provider.RoleAssistant, Content: "reply-1", ToolCalls: []provider.ToolCall{{ID: "c1", Name: "read", Arguments: `{"p":"a"}`}}},
		{Role: provider.RoleTool, ToolCallID: "c1", Content: `{"ok":true}`},
		{Role: provider.RoleUser, Content: "more"},
		{Role: provider.RoleAssistant, Content: "reply-2"},
	}
	a := &Agent{}
	head := 1
	ordinary := a.normalizeModelRequestMessages(msgs)
	summary := a.summaryRequest(msgs[0:head], msgs[head:], "").Messages
	// Both end with the compaction instruction; the shared prefix must match
	// up to the start of the instruction.
	prefix := len(summary) - 1
	if prefix > len(ordinary) {
		prefix = len(ordinary)
	}
	if prefix != len(ordinary) {
		t.Fatalf("ordinary messages = %d, summary prefix = %d (want ordinary fully reproduced)", len(ordinary), prefix)
	}
	for i := 0; i < prefix; i++ {
		o, s := ordinary[i], summary[i]
		if o.Role != s.Role || o.Content != s.Content || o.ToolCallID != s.ToolCallID {
			t.Fatalf("byte divergence at message %d: ordinary (%s %q tc=%s) vs summary (%s %q tc=%s)", i, o.Role, o.Content, o.ToolCallID, s.Role, s.Content, s.ToolCallID)
		}
		if len(o.ToolCalls) != len(s.ToolCalls) {
			t.Fatalf("tool-call count divergence at message %d: %d vs %d", i, len(o.ToolCalls), len(s.ToolCalls))
		}
	}
}
