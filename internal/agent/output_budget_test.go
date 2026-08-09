package agent

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// sharedWindowBudgetProvider simulates DeepSeek: a large default output
// budget that shares the context window with the prompt input.
type sharedWindowBudgetProvider struct {
	budget int
}

func (*sharedWindowBudgetProvider) Name() string { return "shared-window" }
func (p *sharedWindowBudgetProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	return nil, nil
}
func (p *sharedWindowBudgetProvider) OutputBudget() int       { return p.budget }
func (*sharedWindowBudgetProvider) SharesContextWindow() bool { return true }

var _ provider.Provider = (*sharedWindowBudgetProvider)(nil)
var _ provider.OutputBudgetProvider = (*sharedWindowBudgetProvider)(nil)
var _ provider.SharedWindowOutputProvider = (*sharedWindowBudgetProvider)(nil)

// independentBudgetProvider simulates MiMo/OpenAI: a large output budget that
// does NOT compete with the prompt input.
type independentBudgetProvider struct {
	budget int
}

func (*independentBudgetProvider) Name() string { return "independent" }
func (p *independentBudgetProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	return nil, nil
}
func (p *independentBudgetProvider) OutputBudget() int { return p.budget }

var _ provider.Provider = (*independentBudgetProvider)(nil)
var _ provider.OutputBudgetProvider = (*independentBudgetProvider)(nil)

// sharedWindowOnlyProvider implements SharesContextWindow but no output budget.
type sharedWindowOnlyProvider struct{}

func (*sharedWindowOnlyProvider) Name() string { return "shared-no-budget" }
func (*sharedWindowOnlyProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	return nil, nil
}
func (*sharedWindowOnlyProvider) SharesContextWindow() bool { return true }

var _ provider.Provider = (*sharedWindowOnlyProvider)(nil)
var _ provider.SharedWindowOutputProvider = (*sharedWindowOnlyProvider)(nil)

// sharedFakeProvider embeds fakeProvider (which can summarize via its Stream
// reply) and advertises a shared context window like DeepSeek.
type sharedFakeProvider struct {
	*fakeProvider
	budget int
}

func (p *sharedFakeProvider) OutputBudget() int       { return p.budget }
func (*sharedFakeProvider) SharesContextWindow() bool { return true }

var _ provider.Provider = (*sharedFakeProvider)(nil)
var _ provider.OutputBudgetProvider = (*sharedFakeProvider)(nil)
var _ provider.SharedWindowOutputProvider = (*sharedFakeProvider)(nil)

// bigTokenString returns a string that estimates to roughly n tokens under
// estimateTextTokens (CJK-heavy: ~1 rune per token).
func bigTokenString(n int) string {
	const rune = "字"
	b := make([]byte, 0, n*3)
	for len(b)/3 < n {
		b = append(b, rune...)
	}
	return string(b)
}

func newSessionWithMsgs(msgs []provider.Message) *Session {
	s := NewSession("")
	for _, m := range msgs {
		s.Add(m)
	}
	return s
}

func TestMaybeCompactOnResumeUnsharedWindowNoop(t *testing.T) {
	a := &Agent{prov: &independentBudgetProvider{budget: 128 * 1024}, contextWindow: 1_048_576, sink: event.Discard}
	a.session = newSessionWithMsgs([]provider.Message{{Role: provider.RoleUser, Content: "x"}})
	a.MaybeCompactOnResume(context.Background())
	if st := a.compactionState; len(st.Projection.Messages) != 0 {
		t.Fatal("independent vendor must not compact on resume")
	}
}

func TestMaybeCompactOnResumeOversizedPromptCompacts(t *testing.T) {
	fp := &fakeProvider{reply: "GOAL: fit inside window"}
	sess := NewSession("sys")
	for i := 0; i < 40; i++ {
		sess.Add(provider.Message{Role: provider.RoleUser, Content: "user turn " + strings.Repeat("x", 200)})
		sess.Add(provider.Message{Role: provider.RoleAssistant, Content: "assistant " + strings.Repeat("y", 400)})
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	// Tiny window so the oversized prompt exceeds window - minBudget - reserve.
	a := New(fp, nil, sess, Options{
		ContextWindow: 30_000,
		RecentKeep:    2,
		ArchiveDir:    dir,
		SessionPath:   path,
		ModelRef:      "test/model",
	}, event.Discard)
	// Shared-window vendor: fakeProvider must advertise it for the gate to run.
	// Re-point the agent's provider to one that shares the window and can
	// summarize (fakeProvider streams the reply text).
	a.prov = &sharedFakeProvider{fakeProvider: fp, budget: 128 * 1024}
	a.outputBudget = 128 * 1024

	a.MaybeCompactOnResume(context.Background())
	if len(a.compactionState.Projection.Messages) == 0 {
		t.Fatal("oversized prompt must install a projection on resume")
	}
}

func TestMaybeCompactOnResumeColdLargePromptUntouched(t *testing.T) {
	// Deferred compaction: a cold cache with a large-but-fitting prompt is left
	// untouched — replay pays the miss price once, cheaper than rewriting the
	// prefix on every resume. Only an input-overflow (would-400) prompt compacts.
	fp := &fakeProvider{reply: "GOAL: cold replay avoided"}
	sess := NewSession("sys")
	for i := 0; i < 20; i++ {
		sess.Add(provider.Message{Role: provider.RoleUser, Content: "user turn " + strings.Repeat("x", 200)})
		sess.Add(provider.Message{Role: provider.RoleAssistant, Content: "assistant " + strings.Repeat("y", 400)})
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	a := New(fp, nil, sess, Options{
		ContextWindow: 100_000,
		RecentKeep:    2,
		ArchiveDir:    dir,
		SessionPath:   path,
		ModelRef:      "test/model",
	}, event.Discard)
	a.prov = &sharedFakeProvider{fakeProvider: fp, budget: 128 * 1024}
	a.outputBudget = 128 * 1024
	a.cacheState = CacheStateCold

	a.MaybeCompactOnResume(context.Background())
	if st := a.compactionState; len(st.Projection.Messages) != 0 {
		t.Fatal("cold cache + fitting prompt must NOT compact on resume (deferred policy)")
	}
}

func TestMaybeCompactOnResumeWarmSmallPromptUntouched(t *testing.T) {
	a := &Agent{
		prov:              &sharedWindowBudgetProvider{budget: 128 * 1024},
		contextWindow:     1_048_576,
		outputBudgetState: outputBudgetState{outputBudget: 128 * 1024},
		cacheState:        CacheStateWarm,
		sink:              event.Discard,
	}
	a.session = newSessionWithMsgs([]provider.Message{{Role: provider.RoleUser, Content: "short"}})
	a.MaybeCompactOnResume(context.Background())
	if st := a.compactionState; len(st.Projection.Messages) != 0 {
		t.Fatal("warm small resume must keep the cached prefix untouched")
	}
}

// TestMaybeCompactOnResumeWarmMidPromptUntouched guards the resume gate against
// the overflow-guard safety factor: the real-shape estimate (~49% of the
// window) must NOT be doubled into an apparent ~98% that folds a warm cached
// prefix. Over-estimating here destroys the server prefix cache of a healthy
// session; under-estimating only defers compaction to the request path.
func TestMaybeCompactOnResumeWarmMidPromptUntouched(t *testing.T) {
	sink := &recordSink{}
	a := &Agent{
		prov:              &sharedWindowBudgetProvider{budget: 128 * 1024},
		contextWindow:     100_000,
		outputBudgetState: outputBudgetState{outputBudget: 128 * 1024},
		cacheState:        CacheStateWarm,
		sink:              sink,
	}
	// ~49K tokens real shape (ASCII: 1 rune ≈ 1 token), well inside the
	// 100K - 8K - 8K = 83.6K resume threshold.
	a.session = newSessionWithMsgs([]provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: strings.Repeat("a", 49_000)},
	})
	a.MaybeCompactOnResume(context.Background())
	if got := len(sink.kinds(event.CompactionStarted)); got != 0 {
		t.Fatalf("warm resume at ~49%% of the window compacted (%d starts), want none", got)
	}
}

// TestMaybeCompactOnResumeProjectionSmallCanonicalHugeUntouched guards the
// resume gate against the canonical transcript: a valid projection plus tail
// is the exact sent shape, so a huge canonical history behind a small
// projection must NOT trigger the gate (Copilot review 8/8: "estimate the
// model-visible messages instead").
func TestMaybeCompactOnResumeProjectionSmallCanonicalHugeUntouched(t *testing.T) {
	sink := &recordSink{}
	a := &Agent{
		prov:              &sharedWindowBudgetProvider{budget: 128 * 1024},
		contextWindow:     100_000,
		outputBudgetState: outputBudgetState{outputBudget: 128 * 1024},
		cacheState:        CacheStateWarm,
		sink:              sink,
		workspaceID:       "ws",
		sessionPath:       "sess.jsonl",
		modelRef:          "model",
	}
	// Canonical history alone (~90K ASCII) exceeds the 100K - 8K - 8K gate.
	canonical := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: strings.Repeat("a", 90_000)},
	}
	a.session = newSessionWithMsgs(canonical)
	msgs, version := a.session.snapshotMessagesVersion()
	// A valid projection keeps the model-visible shape tiny.
	a.compactionState = CompactionState{
		TranscriptVersion: version,
		PromptCacheKey:    "ws|sess|model",
		Projection: ContextProjection{
			Messages: []provider.Message{
				{Role: provider.RoleSystem, Content: "sys"},
				{Role: provider.RoleUser, Content: "summary"},
			},
			TranscriptVersion: version,
			CoveredCount:      len(msgs),
			CoveredPrefixHash: coveredPrefixHash(msgs, len(msgs)),
		},
	}
	a.MaybeCompactOnResume(context.Background())
	if got := len(sink.kinds(event.CompactionStarted)); got != 0 {
		t.Fatalf("resume with a small valid projection compacted (%d starts), want none", got)
	}
}

// capturingBudgetProvider embeds sharedFakeProvider and records the MaxTokens
// of the last streamed request plus how many streams ran.
type capturingBudgetProvider struct {
	sharedFakeProvider
	maxTokens int
	streams   int
}

func (p *capturingBudgetProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.maxTokens = req.MaxTokens
	p.streams++
	return p.sharedFakeProvider.Stream(ctx, req)
}

var _ provider.Provider = (*capturingBudgetProvider)(nil)
var _ provider.OutputBudgetProvider = (*capturingBudgetProvider)(nil)
var _ provider.SharedWindowOutputProvider = (*capturingBudgetProvider)(nil)

// TestSummarizeClipsSharedWindowBudget verifies the compaction summarizer clips
// its own MaxTokens for a shared-window vendor: an unclipped default made
// compaction itself fail with HTTP 400 near the window edge.

// TestSummarizeRejectsTooLargeInput verifies the summarizer refuses a fold
// that cannot fit beside a usable output budget: the request would 400 again
// and retrying it every turn just re-runs the failure.

func TestMaybePredictOverflowFires(t *testing.T) {
	var notices []string
	sink := event.FuncSink(func(e event.Event) {
		if e.Kind == event.Notice {
			notices = append(notices, e.Text)
		}
	})
	a := &Agent{
		contextWindow: 1_048_576,
		sink:          sink,
	}
	// est 1M, max 128K → headroom = 1M - 1M - 128K = -128K < 8K → fires.
	a.maybePredictOverflow(1_048_576, 131_072)
	found := false
	for _, n := range notices {
		if n == "context window nearly full" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected overflow notice, got %v", notices)
	}
}

// TestMaybePredictOverflowDoesNotFire when there is adequate headroom.
func TestMaybePredictOverflowDoesNotFire(t *testing.T) {
	count := 0
	sink := event.FuncSink(func(e event.Event) {
		count++
	})
	a := &Agent{
		contextWindow: 1_048_576,
		sink:          sink,
	}
	// est 100K, max 128K → headroom = 1M - 100K - 128K = ~820K >> 8K → no fire.
	a.maybePredictOverflow(100_000, 131_072)
	if count > 0 {
		t.Fatalf("overflow predicted on a small prompt (%d events)", count)
	}
}

type sharedWindowTestProvider struct {
	budget int
	shared bool
	policy provider.SharedWindowInputPolicy
	last   provider.Request
	calls  int
	finish string
}

func (*sharedWindowTestProvider) Name() string { return "shared-window-test" }

func (p *sharedWindowTestProvider) Stream(_ context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.last = req
	p.calls++
	ch := make(chan provider.Chunk, 3)
	ch <- provider.Chunk{Type: provider.ChunkText, Text: "summary"}
	if p.finish != "" {
		ch <- provider.Chunk{Type: provider.ChunkUsage, Usage: &provider.Usage{FinishReason: p.finish}}
	}
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}

func TestSharedWindowFoldUsesGuardedInputBudget(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	a := &Agent{
		prov:              prov,
		contextWindow:     100_000,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget},
		sink:              event.Discard,
	}
	fold := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("字", 50_000)}}

	if _, err := a.foldToSummary(context.Background(), fold, ""); err != nil {
		t.Fatalf("foldToSummary: %v", err)
	}
	if prov.calls == 0 || len(prov.last.Messages) < 2 {
		t.Fatalf("guarded fold produced no summarizer request: calls=%d request=%+v", prov.calls, prov.last)
	}
	if got := prov.last.Messages[1].Content; len(got) >= len(renderTranscript(fold)) || !strings.Contains(got, "omitted") {
		t.Fatalf("cold CJK fold was not bounded before summarize: len=%d original=%d", len(got), len(renderTranscript(fold)))
	}
	if got := prov.last.MaxTokens; got < summaryOutputReserve {
		t.Fatalf("summarizer MaxTokens = %d, below summaryOutputReserve %d", got, summaryOutputReserve)
	}
}

func (p *sharedWindowTestProvider) OutputBudget() int         { return p.budget }
func (p *sharedWindowTestProvider) SharesContextWindow() bool { return p.shared }

func (p *sharedWindowTestProvider) SharedWindowInputPolicy() provider.SharedWindowInputPolicy {
	return p.policy
}

func TestEffectiveOutputBudgetClipsSharedWindowRequest(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	a := &Agent{prov: prov, contextWindow: 1_048_576,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget}}
	msgs := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("字", 950_000)}}
	// Calibrate this session at one token per rune. The 950K prompt fits, but
	// not beside the provider's full 128K output default.
	a.lastUsage.Store(&provider.Usage{PromptTokens: 950_000})
	a.setPromptTokenCalibration(950_000, requestCalibrationShapeOf(provider.Request{Messages: msgs}))

	got, clipped, err := a.effectiveOutputBudget(provider.Request{Messages: msgs})
	if err != nil {
		t.Fatalf("effectiveOutputBudget: %v", err)
	}
	if !clipped {
		t.Fatal("near-window request kept the provider's full output budget")
	}
	if got <= 0 || got >= prov.budget {
		t.Fatalf("clipped budget = %d, want 0 < budget < %d", got, prov.budget)
	}
	if got+950_000 > a.contextWindow-outputBudgetReserve {
		t.Fatalf("input + output = %d, exceeds reserved shared window %d", got+950_000, a.contextWindow-outputBudgetReserve)
	}
}

func TestCalibratedOutputBudgetIncludesReplayedReasoning(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	a := &Agent{prov: prov, contextWindow: 200_000,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget}}
	previous := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("x", 300_000)}}
	a.setPromptTokenCalibration(75_000, requestCalibrationShapeOf(provider.Request{Messages: previous}))
	current := append(previous, provider.Message{
		Role:             provider.RoleAssistant,
		ReasoningContent: strings.Repeat("r", 400_000),
		ToolCalls:        []provider.ToolCall{{ID: "call_1", Name: "bash", Arguments: `{}`}},
	})

	before := a.estimatedPromptTokens(previous)
	after := a.estimatedPromptTokens(current)
	if after < before+99_000 {
		t.Fatalf("400K replayed reasoning was not calibrated: before=%d after=%d", before, after)
	}
	budget, clipped, err := a.effectiveOutputBudget(provider.Request{Messages: current})
	if err != nil {
		t.Fatalf("effectiveOutputBudget: %v", err)
	}
	if !clipped || budget > 20_000 {
		t.Fatalf("replayed reasoning budget = %d clipped=%v, want a clipped budget <= 20000", budget, clipped)
	}
}

func TestCalibratedOutputBudgetKeepsCJKConservativeFloor(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	a := &Agent{prov: prov, contextWindow: 1_048_576,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget}}
	previous := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("x", 300_000)}}
	a.setPromptTokenCalibration(75_000, requestCalibrationShapeOf(provider.Request{Messages: previous}))
	current := append(append([]provider.Message(nil), previous...), provider.Message{
		Role:             provider.RoleAssistant,
		ReasoningContent: strings.Repeat("字", 430_000),
		ToolCalls:        []provider.ToolCall{{ID: "call_1", Name: "bash", Arguments: `{}`}},
	})

	calibrated := a.estimatedPromptTokens(current)
	wantFloor := 75_000 + 2*430_000
	if calibrated < wantFloor {
		t.Fatalf("calibrated estimate %d fell below mixed-script safety floor %d", calibrated, wantFloor)
	}

	budget, clipped, err := a.effectiveOutputBudget(provider.Request{Messages: current})
	if err != nil {
		t.Fatalf("effectiveOutputBudget: %v", err)
	}
	if !clipped || budget <= 0 || budget >= prov.budget {
		t.Fatalf("mixed-script request budget = %d clipped=%v, want a clipped positive budget below %d", budget, clipped, prov.budget)
	}
}

func TestCalibrationIgnoresNonReplayableOrdinaryReasoning(t *testing.T) {
	a := &Agent{}
	previous := []provider.Message{
		{Role: provider.RoleUser, Content: strings.Repeat("x", 300_000)},
		{Role: provider.RoleAssistant, ReasoningContent: strings.Repeat("hidden", 150_000)},
	}
	a.setPromptTokenCalibration(75_000, requestCalibrationShapeOf(provider.Request{Messages: previous}))
	current := append(append([]provider.Message(nil), previous...), provider.Message{
		Role:             provider.RoleAssistant,
		ReasoningContent: strings.Repeat("r", 400_000),
		ToolCalls:        []provider.ToolCall{{ID: "call_1", Name: "bash", Arguments: `{}`}},
	})

	if got := a.estimatedPromptTokens(current); got < 160_000 {
		t.Fatalf("replayable reasoning estimate = %d, want ordinary local reasoning excluded from calibration denominator", got)
	}
}

func TestCalibratedResponsesBudgetIncludesNewOrdinaryReasoning(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true,
		policy: provider.SharedWindowInputPolicy{ReplaysOrdinaryReasoning: true}}
	a := &Agent{prov: prov, contextWindow: 200_000,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget}}
	previous := provider.Request{Messages: []provider.Message{{
		Role: provider.RoleUser, Content: strings.Repeat("x", 300_000),
	}}}
	a.setPromptTokenCalibration(75_000, a.requestCalibrationShape(previous))
	current := previous
	current.Messages = append(append([]provider.Message(nil), previous.Messages...), provider.Message{
		Role: provider.RoleAssistant, ReasoningContent: strings.Repeat("r", 400_000),
	})

	if got := a.estimatedRequestTokens(current); got < 174_000 {
		t.Fatalf("Responses ordinary reasoning estimate = %d, want newly replayed reasoning included", got)
	}
	if budget, clipped, err := a.effectiveOutputBudget(current); err != nil || !clipped || budget >= prov.budget {
		t.Fatalf("Responses ordinary reasoning budget = %d clipped=%v err=%v, want a clipped budget", budget, clipped, err)
	}
}

func TestCalibratedResponsesBudgetIncludesNewReplayItems(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true,
		policy: provider.SharedWindowInputPolicy{ReplaysResponsesItems: true}}
	a := &Agent{prov: prov, contextWindow: 200_000,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget}}
	previous := provider.Request{Messages: []provider.Message{{
		Role: provider.RoleUser, Content: strings.Repeat("x", 300_000),
	}}}
	a.setPromptTokenCalibration(75_000, a.requestCalibrationShape(previous))
	item := json.RawMessage(`{"id":"ws_1","type":"web_search_call","status":"completed","action":{"query":"` + strings.Repeat("q", 400_000) + `"}}`)
	current := previous
	current.Messages = append(append([]provider.Message(nil), previous.Messages...), provider.Message{
		Role: provider.RoleAssistant, ResponsesItems: []json.RawMessage{item},
	})

	if got := a.estimatedRequestTokens(current); got < 174_000 {
		t.Fatalf("Responses replay-item estimate = %d, want newly replayed item included", got)
	}
	if budget, clipped, err := a.effectiveOutputBudget(current); err != nil || !clipped || budget >= prov.budget {
		t.Fatalf("Responses replay-item budget = %d clipped=%v err=%v, want a clipped budget", budget, clipped, err)
	}
}

func TestCalibratedOutputBudgetCountsToolSchemasOnce(t *testing.T) {
	a := &Agent{}
	req := provider.Request{
		Messages: []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("x", 100_000)}},
		Tools: []provider.ToolSchema{{
			Name: "lookup", Description: strings.Repeat("y", 100_000), Parameters: []byte(`{"type":"object"}`),
		}},
	}
	a.setPromptTokenCalibration(60_000, requestCalibrationShapeOf(req))

	if got := a.estimatedRequestTokens(req); got != 60_000 {
		t.Fatalf("calibrated request tokens = %d, want tool schema counted once in 60000", got)
	}
}

func TestPrepareSamplingRequestClipsSharedWindowOutput(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	msgs := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("字", 950_000)}}
	sess := NewSession("")
	sess.Replace(msgs)
	a := &Agent{
		prov:              prov,
		tools:             tool.NewRegistry(),
		session:           sess,
		contextWindow:     1_048_576,
		compactRatio:      2,
		compactForceRatio: 2,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget},
	}
	a.lastUsage.Store(&provider.Usage{PromptTokens: 950_000})
	a.setPromptTokenCalibration(950_000, requestCalibrationShapeOf(provider.Request{Messages: msgs}))

	prepared, err := a.prepareSamplingRequest(context.Background())
	if err != nil {
		t.Fatalf("prepareSamplingRequest: %v", err)
	}
	if prepared.req.MaxTokens <= 0 || prepared.req.MaxTokens >= prov.budget {
		t.Fatalf("prepared MaxTokens = %d, want a clipped positive budget below %d", prepared.req.MaxTokens, prov.budget)
	}
}

func TestEffectiveOutputBudgetRejectsExhaustedSharedWindow(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	a := &Agent{prov: prov, contextWindow: 1_048_576,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget}}
	msgs := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("字", 1_045_000)}}
	a.lastUsage.Store(&provider.Usage{PromptTokens: 1_045_000})
	a.setPromptTokenCalibration(1_045_000, requestCalibrationShapeOf(provider.Request{Messages: msgs}))

	_, _, err := a.effectiveOutputBudget(provider.Request{Messages: msgs})
	if !errors.Is(err, ErrCompactionRequired) {
		t.Fatalf("effectiveOutputBudget error = %v, want ErrCompactionRequired", err)
	}
}

func TestEffectiveOutputBudgetLeavesIndependentProviderUnchanged(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: false}
	a := &Agent{prov: prov, contextWindow: 1_048_576,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget}}
	got, clipped, err := a.effectiveOutputBudget(provider.Request{
		Messages: []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("字", 950_000)}},
	})
	if err != nil || clipped || got != 0 {
		t.Fatalf("independent provider changed: budget=%d clipped=%v err=%v", got, clipped, err)
	}
}

func TestEffectiveOutputBudgetHonorsExplicitOmit(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	a := &Agent{prov: prov, contextWindow: 1_048_576,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget}}
	got, clipped, err := a.effectiveOutputBudget(provider.Request{
		Messages:  []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("字", 950_000)}},
		MaxTokens: -1,
	})
	if err != nil || clipped || got != 0 {
		t.Fatalf("explicit omit changed: budget=%d clipped=%v err=%v", got, clipped, err)
	}
}

func TestSummarizeClipsSharedWindowOutputBudget(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	a := &Agent{
		prov:              prov,
		contextWindow:     100_000,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget},
		sink:              event.Discard,
	}
	region := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("字", 50_000)}}
	a.lastUsage.Store(&provider.Usage{PromptTokens: 50_000})
	a.setPromptTokenCalibration(50_000, requestCalibrationShapeOf(provider.Request{Messages: region}))

	if _, _, err := a.summarize(context.Background(), region, ""); err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if prov.last.MaxTokens <= 0 || prov.last.MaxTokens >= prov.budget {
		t.Fatalf("summarizer MaxTokens = %d, want a clipped positive budget below %d", prov.last.MaxTokens, prov.budget)
	}
}

func TestSummarizeRejectsLengthTruncation(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true, finish: "length"}
	a := &Agent{
		prov:              prov,
		contextWindow:     1_048_576,
		outputBudgetState: outputBudgetState{outputBudget: prov.budget},
		sink:              event.Discard,
	}

	_, _, err := a.summarizeWithRetry(context.Background(), []provider.Message{{
		Role: provider.RoleUser, Content: "retain every durable fact",
	}}, "")
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("summarizeWithRetry error = %v, want truncation failure", err)
	}
	if prov.calls != 1 {
		t.Fatalf("length-truncated summary calls = %d, want no identical retry", prov.calls)
	}
}

func TestSetSessionResetsTokenCalibration(t *testing.T) {
	a := &Agent{}
	a.lastUsage.Store(&provider.Usage{PromptTokens: 200_000})
	active := requestCalibrationShape{requestChars: 900_000, compactChars: 850_000}
	a.activeReqShape.Store(&active)
	a.setPromptTokenCalibration(200_000, requestCalibrationShape{requestChars: 1_000_000, compactChars: 950_000})
	a.SetSession(NewSession("new"))

	if got := a.lastUsage.Load(); got != nil {
		t.Fatalf("lastUsage survived session switch: %+v", got)
	}
	if got := a.activeReqShape.Load(); got != nil {
		t.Fatalf("activeReqShape survived session switch: %+v", got)
	}
	if got := a.promptCalibration.Load(); got != nil {
		t.Fatalf("promptCalibration survived session switch: %+v", got)
	}
}

func TestLatestUsagePairsWithActiveRequestSize(t *testing.T) {
	a := &Agent{}
	active := requestCalibrationShape{requestChars: 222, compactChars: 111, cjkRunes: 22, cjkBytes: 66}
	a.activeReqShape.Store(&active)
	a.storeLatestRequestUsage(&provider.Usage{PromptTokens: 100})

	if got := a.promptCalibration.Load(); got == nil || got.promptTokens != 100 || got.requestChars != 222 || got.compactChars != 111 || got.cjkRunes != 22 || got.cjkBytes != 66 {
		t.Fatalf("promptCalibration = %+v, want promptTokens=100 requestChars=222 compactChars=111 cjkRunes=22 cjkBytes=66", got)
	}
}

func TestEstimatedUsageDoesNotReplacePromptCalibration(t *testing.T) {
	a := &Agent{}
	active := requestCalibrationShape{requestChars: 200_000, compactChars: 100_000}
	a.activeReqShape.Store(&active)
	a.setPromptTokenCalibration(50_000, requestCalibrationShape{requestChars: 100_000, compactChars: 80_000})

	a.storeLatestRequestUsage(&provider.Usage{
		PromptTokens: 10_000,
		TotalTokens:  10_100,
		Estimated:    true,
	})

	got := a.promptCalibration.Load()
	if got == nil || got.promptTokens != 50_000 || got.requestChars != 100_000 || got.compactChars != 80_000 {
		t.Fatalf("estimated usage replaced provider calibration: %+v", got)
	}
	if latest := a.lastUsage.Load(); latest == nil || !latest.Estimated {
		t.Fatalf("estimated usage was not retained for accounting: %+v", latest)
	}
}

func TestForkCaptureProviderPreservesOutputBudgetCapabilities(t *testing.T) {
	t.Setenv("REASONIX_EXPERIMENT_FORK_CAPTURE_DIR", t.TempDir())
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true,
		policy: provider.SharedWindowInputPolicy{ReplaysOrdinaryReasoning: true, ReplaysResponsesItems: true}}
	a := New(prov, tool.NewRegistry(), NewSession(""), Options{}, event.Discard)

	if !sharesContextWindow(a.prov) {
		t.Fatal("fork capture wrapper erased shared-window output capability")
	}
	if got := outputBudgetOf(a.prov); got != prov.budget {
		t.Fatalf("wrapped output budget = %d, want %d", got, prov.budget)
	}
	if got := sharedWindowInputPolicyOf(a.prov); got != prov.policy {
		t.Fatalf("wrapped input policy = %+v, want %+v", got, prov.policy)
	}
}
