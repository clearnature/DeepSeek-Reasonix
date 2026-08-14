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

func TestProjectionValidRejectsEditedPrefix(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task-v1"},
		{Role: provider.RoleAssistant, Content: "done"},
		{Role: provider.RoleUser, Content: "next"},
	}
	st := CompactionState{
		TranscriptVersion: 2,
		PromptCacheKey:    "ws|sess|model",
		Projection: ContextProjection{
			Messages: []provider.Message{
				{Role: provider.RoleSystem, Content: "sys"},
				{Role: provider.RoleUser, Content: "summary"},
			},
			TranscriptVersion: 2,
			CoveredCount:      3,
			CoveredPrefixHash: coveredPrefixHash(msgs, 3),
		},
	}
	if !projectionValid(st, msgs, 2, "ws|sess|model") {
		t.Fatal("expected valid projection for matching prefix")
	}
	// Append-only growth still valid.
	grown := append(append([]provider.Message(nil), msgs...), provider.Message{Role: provider.RoleAssistant, Content: "more"})
	if !projectionValid(st, grown, 3, "ws|sess|model") {
		t.Fatal("append-only growth should keep projection valid")
	}
	// Prefix edit invalidates.
	edited := append([]provider.Message(nil), msgs...)
	edited[1].Content = "task-EDITED"
	if projectionValid(st, edited, 4, "ws|sess|model") {
		t.Fatal("edited covered prefix must invalidate projection")
	}
}

func TestCoveredPrefixHashIncludesProviderVisibleFields(t *testing.T) {
	base := []provider.Message{{
		Role:               provider.RoleAssistant,
		Content:            "answer",
		ReasoningContent:   "think",
		ReasoningID:        "rid-1",
		ReasoningStatus:    "completed",
		ReasoningSignature: "sig-1",
		Images:             []string{"data:image/png;base64,AAA"},
		ToolCalls: []provider.ToolCall{{
			ID: "c1", Name: "f", Arguments: `{}`, ThoughtSignature: "ts-1",
		}},
		ResponsesItems: []json.RawMessage{json.RawMessage(`{"type":"web_search_call"}`)},
	}}
	h1 := coveredPrefixHash(base, 1)
	if h1 == "" {
		t.Fatal("empty fingerprint")
	}
	// Each provider-visible field change must move the hash.
	cases := []struct {
		name string
		mut  func([]provider.Message)
	}{
		{"images", func(m []provider.Message) { m[0].Images = []string{"data:image/png;base64,BBB"} }},
		{"reasoning_id", func(m []provider.Message) { m[0].ReasoningID = "rid-2" }},
		{"reasoning_status", func(m []provider.Message) { m[0].ReasoningStatus = "in_progress" }},
		{"reasoning_signature", func(m []provider.Message) { m[0].ReasoningSignature = "sig-2" }},
		{"thought_signature", func(m []provider.Message) { m[0].ToolCalls[0].ThoughtSignature = "ts-2" }},
		{"responses_items", func(m []provider.Message) {
			m[0].ResponsesItems = []json.RawMessage{json.RawMessage(`{"type":"other"}`)}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := []provider.Message{base[0]}
			mutated[0].ToolCalls = append([]provider.ToolCall(nil), base[0].ToolCalls...)
			mutated[0].Images = append([]string(nil), base[0].Images...)
			mutated[0].ResponsesItems = append([]json.RawMessage(nil), base[0].ResponsesItems...)
			tc.mut(mutated)
			if coveredPrefixHash(mutated, 1) == h1 {
				t.Fatalf("%s change did not alter coveredPrefixHash", tc.name)
			}
		})
	}
}

func TestLoadProjectionSidecarRebindsMatchingContentAcrossLineage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
	}
	hash := coveredPrefixHash(msgs, 2)
	if err := SaveCompactionState(path, CompactionState{
		SchemaVersion:     compactionStateSchemaV1,
		PromptCacheKey:    "ws|s|other-model",
		TranscriptVersion: 1,
		Projection: ContextProjection{
			Messages:          []provider.Message{{Role: provider.RoleSystem, Content: "sys summary"}},
			CoveredCount:      2,
			CoveredPrefixHash: hash,
		},
	}); err != nil {
		t.Fatal(err)
	}
	sess := NewSession("sys")
	sess.Add(provider.Message{Role: provider.RoleUser, Content: "task"})
	a := New(nil, nil, sess, Options{
		SessionPath: path,
		WorkspaceID: "ws",
		ModelRef:    "this-model",
	}, event.Discard)
	// New() already called LoadProjectionSidecar; the projection body matches
	// the canonical covered prefix, so it must be rebound to the current key
	// instead of being dropped (upgrade / model-switch path).
	if len(a.sess.compactionState.Projection.Messages) == 0 {
		t.Fatal("matching projection body was dropped on lineage change")
	}
	wantKey := promptCacheKey("ws", BranchID(path), "this-model")
	if a.sess.compactionState.PromptCacheKey != wantKey {
		t.Fatalf("PromptCacheKey = %q, want %q", a.sess.compactionState.PromptCacheKey, wantKey)
	}
	if a.sess.checkpointState != "restored" {
		t.Fatalf("checkpointState = %q, want restored", a.sess.checkpointState)
	}
	// The rebind must be persisted so the next launch does not re-downgrade.
	disk, ok, err := LoadCompactionState(path)
	if err != nil || !ok {
		t.Fatalf("sidecar should remain on disk: ok=%v err=%v", ok, err)
	}
	if disk.PromptCacheKey != wantKey {
		t.Fatalf("persisted PromptCacheKey = %q, want %q", disk.PromptCacheKey, wantKey)
	}
}

// TestLoadProjectionSidecarKeepsBodyWithoutTranscript pins the resume-order
// guard: binding the sidecar before the conversation finished loading must not
// discard a valid projection body — modelVisible re-validates later (8/13:
// a cold start re-reported a window-full session as 2.4M/1M after a projection
// was dropped this way).
func TestLoadProjectionSidecarKeepsBodyWithoutTranscript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "q2"},
	}
	if err := SaveCompactionState(path, CompactionState{
		SchemaVersion:     compactionStateSchemaV1,
		PromptCacheKey:    "ws|s|this-model",
		TranscriptVersion: 1,
		Projection: ContextProjection{
			Messages:          []provider.Message{{Role: provider.RoleSystem, Content: "summary"}},
			CoveredCount:      4,
			CoveredPrefixHash: coveredPrefixHash(msgs, 4),
		},
	}); err != nil {
		t.Fatal(err)
	}
	// Transcript entirely missing: keep the projection body.
	a := &Agent{agentConfig: agentConfig{workspaceID: "ws", modelRef: "this-model"}, sess: sessionRuntime{}}
	a.LoadProjectionSidecar(path)
	if len(a.sess.compactionState.Projection.Messages) == 0 {
		t.Fatal("projection body dropped when transcript was not loaded yet")
	}
	// Partial UI-provided slice (resumeWithFreshSystemPrompt fallback path):
	// shorter than CoveredCount, so it cannot judge the covered prefix —
	// keep the projection instead of dropping it.
	partial := &Agent{agentConfig: agentConfig{workspaceID: "ws", modelRef: "this-model"}, sess: sessionRuntime{}}
	partial.sess.conversation = NewSession("sys")
	partial.sess.conversation.Add(provider.Message{Role: provider.RoleUser, Content: "task"})
	partial.LoadProjectionSidecar(path)
	if len(partial.sess.compactionState.Projection.Messages) == 0 {
		t.Fatal("projection body dropped when transcript was only partially loaded")
	}
	// Once the full transcript is attached the projection is usable again.
	a.sess.conversation = NewSession("sys")
	for _, m := range msgs[1:] {
		a.sess.conversation.Add(m)
	}
	if vis := a.modelVisibleMessages(); len(vis) != 1 || vis[0].Content != "summary" {
		t.Fatalf("modelVisible after transcript attach = %+v, want the projection", vis)
	}
}

// TestProjectionUsableRejectsOverflowingBody pins the window-fit guard: a
// projection whose body alone overflows the physical ceiling (a prune/snip
// rebuilt full-history one) must not be sent — modelVisible falls back to
// canonical so the request path folds it down (8/13: 7.2K messages ≈ 2.2M).
func TestProjectionUsableRejectsOverflowingBody(t *testing.T) {
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	huge := provider.Message{Role: provider.RoleUser, Content: strings.Repeat("字", 1_200_000)} // 1.2M CJK runes
	projMsgs := []provider.Message{huge, huge}                                                 // official est ≈ 1.44M
	sess := &Session{Messages: append([]provider.Message{{Role: provider.RoleSystem, Content: "sys"}}, projMsgs...)}
	a := New(prov, tool.NewRegistry(), sess, Options{
		SessionPath:   filepath.Join(t.TempDir(), "s.jsonl"),
		ContextWindow: 1_048_576,
		WorkspaceID:   "ws",
		ModelRef:      "m",
	}, event.Discard)
	a.sess.compactionState.Projection = ContextProjection{
		Messages:          projMsgs,
		CoveredCount:      2,
		CoveredPrefixHash: coveredPrefixHash(sess.Messages, 2),
	}
	if a.projectionUsable(a.sess.compactionState, sess.Messages, 0) {
		t.Fatal("overflowing projection body accepted as usable")
	}
	if vis := a.modelVisibleMessages(); len(vis) != 3 {
		t.Fatalf("modelVisible = %d messages, want canonical fallback (3)", len(vis))
	}
}

func TestLoadProjectionSidecarDropsForeignCacheKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	msgs := []provider.Message{{Role: provider.RoleSystem, Content: "sys"}}
	// Content validation must fail despite a model-only key change: lineage
	// rebinding cannot resurrect a projection whose canonical prefix differs.
	foreign := []provider.Message{{Role: provider.RoleSystem, Content: "sys-old"}}
	if err := SaveCompactionState(path, CompactionState{
		SchemaVersion:  compactionStateSchemaV1,
		PromptCacheKey: "ws|s|other-model",
		Projection: ContextProjection{
			Messages:          msgs,
			CoveredCount:      1,
			CoveredPrefixHash: coveredPrefixHash(foreign, 1),
		},
	}); err != nil {
		t.Fatal(err)
	}
	a := New(nil, nil, NewSession("sys"), Options{
		SessionPath: path,
		WorkspaceID: "ws",
		ModelRef:    "this-model",
	}, event.Discard)
	// New() already called LoadProjectionSidecar; a mismatched content body is
	// kept for the third-state degraded view (digest + tail), not a replay.
	if len(a.sess.compactionState.Projection.Messages) == 0 {
		t.Fatalf("foreign body dropped; third state cannot splice")
	}
	if a.sess.checkpointState != "degraded" {
		t.Fatalf("checkpointState = %q, want degraded", a.sess.checkpointState)
	}
	if _, ok, err := LoadCompactionState(path); err != nil || !ok {
		t.Fatalf("sidecar should remain on disk: ok=%v err=%v", ok, err)
	}
}

func TestForceThresholdNoopReturnsCompactionRequired(t *testing.T) {
	// Huge tool result is entirely in the recent tail → no fold region, but
	// estimate exceeds force; preflight must refuse (not mid-turn).
	huge := strings.Repeat("word ", 5000)
	sess := &Session{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "1", Name: "read", Arguments: "{}"}}},
		{Role: provider.RoleTool, ToolCallID: "1", Name: "read", Content: huge},
	}}
	a := New(&fakeProvider{reply: "unused"}, tool.NewRegistry(), sess, Options{
		ContextWindow:     200,
		CompactRatio:      0.5,
		CompactForceRatio: 0.6,
		RecentKeep:        2,
	}, event.Discard)

	_, err := a.contextManager().Prepare(context.Background(), ContextPreparePolicy{Trigger: CompactionTriggerPressure})
	if err == nil {
		t.Fatal("expected ErrCompactionRequired when force threshold has no fold region")
	}
	if !errors.Is(err, ErrCompactionRequired) {
		t.Fatalf("err = %v, want ErrCompactionRequired", err)
	}
}

func TestSummarizeOnceDoesNotRetry(t *testing.T) {
	fp := &retryUsageProvider{
		failOnce: errors.New("transient"),
		reply:    "digest body",
		usage1:   &provider.Usage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12, RequestCount: 1},
		usage2:   &provider.Usage{PromptTokens: 11, CompletionTokens: 3, TotalTokens: 14, RequestCount: 1},
	}
	a := New(fp, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
	_, _, err := a.summarizeOnce(context.Background(), nil, []provider.Message{
		{Role: provider.RoleUser, Content: "fold me"},
	}, "")
	if err == nil {
		t.Fatal("expected first-attempt failure to surface without retry")
	}
	if fp.calls != 1 {
		t.Fatalf("provider calls = %d, want exactly 1", fp.calls)
	}
}

// retryUsageProvider fails the first Stream, then returns reply + usage2.
type retryUsageProvider struct {
	calls    int
	failOnce error
	reply    string
	usage1   *provider.Usage
	usage2   *provider.Usage
}

func (p *retryUsageProvider) Name() string { return "retry-usage" }
func (p *retryUsageProvider) Stream(_ context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	p.calls++
	ch := make(chan provider.Chunk, 4)
	if p.calls == 1 && p.failOnce != nil {
		if p.usage1 != nil {
			ch <- provider.Chunk{Type: provider.ChunkUsage, Usage: p.usage1}
		}
		ch <- provider.Chunk{Type: provider.ChunkError, Err: p.failOnce}
		close(ch)
		return ch, nil
	}
	ch <- provider.Chunk{Type: provider.ChunkText, Text: p.reply}
	if p.usage2 != nil {
		ch <- provider.Chunk{Type: provider.ChunkUsage, Usage: p.usage2}
	}
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}

func TestCompactInstallsCoveredPrefixHash(t *testing.T) {
	fp := &fakeProvider{reply: "digest"}
	sess := NewSession("sys")
	for range 8 {
		sess.Add(provider.Message{Role: provider.RoleUser, Content: strings.Repeat("u", 80)})
		sess.Add(provider.Message{Role: provider.RoleAssistant, Content: strings.Repeat("a", 120)})
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	a := New(fp, tool.NewRegistry(), sess, Options{
		ContextWindow: 2000,
		RecentKeep:    2,
		ArchiveDir:    dir,
		SessionPath:   path,
		WorkspaceID:   "ws",
		ModelRef:      "m",
	}, event.Discard)
	if err := a.CompactNow(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	st := a.sess.compactionState
	if st.Projection.CoveredPrefixHash == "" {
		t.Fatal("CoveredPrefixHash not set")
	}
	if st.PromptCacheKey != promptCacheKey("ws", BranchID(path), "m") {
		t.Fatalf("PromptCacheKey = %q", st.PromptCacheKey)
	}
	msgs, ver := sess.snapshotMessagesVersion()
	if !projectionValid(st, msgs, ver, st.PromptCacheKey) {
		t.Fatal("fresh projection should validate")
	}
}

// TestProjectionContentValidVersionDriftSamePrefix pins the resume-after-replay
// case: covered prefix bytes identical but the version counter drifted (event
// replay does not restore it). With no append (n == len) the projection must
// stay valid so the first turn folds incrementally (8/13: full fold 3.3% hit
// vs incremental 99.4%).
func TestProjectionContentValidVersionDriftSamePrefix(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
		{Role: provider.RoleAssistant, Content: "plan"},
		{Role: provider.RoleUser, Content: "continue"},
	}
	n := len(msgs)
	st := CompactionState{
		TranscriptVersion: 1, // persisted before the restart
		Projection: ContextProjection{
			CoveredCount:      n,
			CoveredPrefixHash: coveredPrefixHash(msgs, n),
			Messages:          msgs,
		},
	}
	// Version drifted (in-memory counter started at 2 after reload) but the
	// covered prefix is byte-identical and nothing was appended.
	if !projectionContentValid(st, msgs, 2) {
		t.Fatalf("version drift with identical covered prefix must keep the projection valid")
	}
}

func TestProjectionContentValidToleratesPruneRewrite(t *testing.T) {
	base := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "user q"},
		{Role: provider.RoleTool, Content: strings.Repeat("tool body ", 50)},
		{Role: provider.RoleAssistant, Content: "assistant a"},
	}
	st := CompactionState{Projection: ContextProjection{
		Messages:           []provider.Message{{Role: provider.RoleSystem, Content: "summary"}},
		CoveredCount:       len(base),
		CoveredPrefixHash:  coveredPrefixHash(base, len(base)),
		SemanticPrefixHash: semanticPrefixHash(base, len(base)),
		TranscriptVersion:  1,
	}}
	// prune/snip shortens tool results only: covered hash changes, semantic
	// hash (non-tool) stays, projection must remain valid.
	pruned := append([]provider.Message(nil), base...)
	pruned[2] = provider.Message{Role: provider.RoleTool, Content: "tool (trimmed)"}
	if !projectionContentValid(st, pruned, 1) {
		t.Fatalf("prune rewrite (tool-only) must keep the projection valid")
	}
	// Real content change (user message edited) must invalidate.
	edited := append([]provider.Message(nil), base...)
	edited[1] = provider.Message{Role: provider.RoleUser, Content: "user q EDITED"}
	if projectionContentValid(st, edited, 1) {
		t.Fatalf("user content edit must invalidate the projection")
	}
	// Legacy sidecar without semantic hash stays fail-closed on covered drift.
	legacy := st
	legacy.Projection.SemanticPrefixHash = ""
	if projectionContentValid(legacy, pruned, 1) {
		t.Fatalf("legacy sidecar without semantic hash must reject covered drift")
	}
}

// TestModelVisibleDegradedSplicesDigestTail covers the third state: a stale
// projection body splices with the canonical tail instead of a full replay.
func TestModelVisibleDegradedSplicesDigestTail(t *testing.T) {
	a := &Agent{agentConfig: agentConfig{contextWindow: 1024 * 1024}}
	canonical := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleUser, Content: "u2"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "u3"},
		{Role: provider.RoleAssistant, Content: "a2"},
		{Role: provider.RoleUser, Content: "u4"},
		{Role: provider.RoleAssistant, Content: "a3"},
		{Role: provider.RoleUser, Content: "u5"},
		{Role: provider.RoleAssistant, Content: "a4"},
	}
	st := CompactionState{Projection: ContextProjection{
		Messages:          []provider.Message{{Role: provider.RoleSystem, Content: "digest"}},
		CoveredCount:      4,
		ProjectionVersion: 1,
	}}
	visible, ok := a.modelVisibleDegraded(st, canonical)
	if !ok {
		t.Fatalf("degraded splice rejected")
	}
	if len(visible) != 1+6 { // digest + canonical[4:]
		t.Fatalf("degraded view len = %d, want 7 (digest + tail 6)", len(visible))
	}
	if visible[1].Content != "u3" {
		t.Fatalf("tail splice wrong: visible[1] = %q, want u3", visible[1].Content)
	}
}

// TestModelVisibleDegradedRejectsOverflowingCoverage: canonical shrank below
// covered count (rewind) — the stale body must NOT be spliced.
func TestModelVisibleDegradedRejectsOverflowingCoverage(t *testing.T) {
	a := &Agent{agentConfig: agentConfig{contextWindow: 1024 * 1024}}
	canonical := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleUser, Content: "u2"},
	}
	st := CompactionState{Projection: ContextProjection{
		Messages:     []provider.Message{{Role: provider.RoleSystem, Content: "digest"}},
		CoveredCount: 5,
	}}
	if _, ok := a.modelVisibleDegraded(st, canonical); ok {
		t.Fatalf("degraded splice must be rejected when covered > len(canonical)")
	}
}
