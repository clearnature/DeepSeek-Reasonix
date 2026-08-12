package agent

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// foldableSessionOverForce builds a transcript whose bulk is assistant text, so
// the free prune pass cannot reclaim it and Prepare must reach the summarizer.
func foldableSessionOverForce(turns int) *Session {
	big := strings.Repeat("word ", 400)
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "standing constraint: never change the public API"},
	}
	for range turns {
		msgs = append(msgs,
			provider.Message{Role: provider.RoleAssistant, Content: big},
			provider.Message{Role: provider.RoleUser, Content: "continue"},
		)
	}
	return &Session{Messages: msgs}
}

func agentOverForce(t *testing.T, prov provider.Provider, sess *Session) *Agent {
	t.Helper()
	return agentOverForceWindow(t, prov, sess, 5000)
}

// agentOverForceWindow sits the session above the force ratio. A folded
// transcript lands back under the trigger, so a blocked turn can only mean the
// fold itself failed.
func agentOverForceWindow(t *testing.T, prov provider.Provider, sess *Session, window int) *Agent {
	t.Helper()
	return New(prov, tool.NewRegistry(), sess, Options{
		ContextWindow:     window,
		CompactRatio:      0.5,
		CompactForceRatio: 0.5,
		RecentKeep:        2,
		ArchiveDir:        t.TempDir(),
	}, event.Discard)
}

// degradedFold reports whether a fold was committed with the mechanical digest
// standing in for the summary. The receipt is the host record that the
// projection was installed; the digest text is what the model is actually told.
func degradedFold(a *Agent) bool {
	r := a.sess.compactionState.LastReceipt
	return r != nil && r.Status == "applied" &&
		strings.Contains(latestDigest(a.sess.compactionState.Projection.Messages), "summary was unavailable")
}

func prepareContext(ctx context.Context, a *Agent, trigger string) error {
	_, err := a.contextManager().Prepare(ctx, ContextPreparePolicy{Trigger: trigger})
	return err
}

// foldRegionOf is the region the next compaction would hand the summarizer.
func foldRegionOf(a *Agent) []provider.Message {
	canonical, version := a.sess.conversation.snapshotMessagesVersion()
	msgs := a.visibleInputForFold(a.sess.compactionState, canonical, version)
	head, start, ok := a.planFoldRegion(msgs, false)
	if !ok {
		return nil
	}
	_, fold, _ := a.partitionFoldForProjection(msgs[head:start])
	return fold
}

// latestDigest returns the text of the last compaction digest in a projection.
func latestDigest(msgs []provider.Message) string {
	for _, m := range slices.Backward(msgs) {
		if isCompactionSummary(m) {
			return m.Content
		}
	}
	return ""
}

// projectionTokens reports what the model would actually see.
func projectionTokens(a *Agent) int {
	msgs, _ := a.sess.conversation.snapshotMessagesVersion()
	return estimateMessagesTokens(provider.ModelMessages(modelVisibleFromProjection(a.sess.compactionState.Projection, msgs)))
}

// The 90s summary bound is deliberately not retried, so a summarizer that never
// answers is the original reason a mechanical fold exists. Where the fold is the
// only way out, its timeout must free the context rather than strand the turn.
func TestSummarizerTimeoutWhereFoldIsTheOnlyWayOutDegrades(t *testing.T) {
	sess := foldableSessionOverForce(6)
	a := agentOverForce(t, &fakeProvider{hang: true}, sess)
	before := estimateMessagesTokens(provider.ModelMessages(sess.Messages))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := prepareContext(ctx, a, CompactionTriggerOverflow); err != nil {
		t.Fatalf("prepare = %v, want a degraded fold instead of a hard failure", err)
	}
	after := projectionTokens(a)
	t.Logf("degraded fold: est %d -> %d tokens", before, after)
	if after >= before {
		t.Fatalf("degraded fold freed no context: %d -> %d", before, after)
	}
	if !degradedFold(a) {
		t.Errorf("no degraded fold committed: receipt=%+v digest=%q",
			a.sess.compactionState.LastReceipt, latestDigest(a.sess.compactionState.Projection.Messages))
	}
}

// Overflow is the trigger that reports ErrCompactionRequired, so it is where a
// failed summary turns into "context exceeds provider limit and compaction
// failed" and blocks every further message. A mechanical fold answers it.
func TestOverflowSummarizerFailureDegradesInsteadOfBlockingTheTurn(t *testing.T) {
	sess := foldableSessionOverForce(6)
	a := agentOverForce(t, &fakeProvider{streamErr: errors.New("provider down")}, sess)
	before := estimateMessagesTokens(provider.ModelMessages(sess.Messages))

	if err := prepareContext(context.Background(), a, CompactionTriggerOverflow); err != nil {
		t.Fatalf("prepare = %v, want a degraded fold instead of ErrCompactionRequired", err)
	}
	if after := projectionTokens(a); after >= before {
		t.Fatalf("degraded fold freed no context: %d -> %d", before, after)
	}
	if !degradedFold(a) {
		t.Errorf("no degraded fold committed: receipt=%+v", a.sess.compactionState.LastReceipt)
	}
}

// A fold too large for one request is shortened before it is sent, which is a
// second way into the same failure. That path must degrade like a plain
// summarizer failure rather than strand the turn.
func TestSummarizerFailureOnOversizedFoldDegrades(t *testing.T) {
	sess := foldableSessionOverForce(120)
	a := agentOverForceWindow(t, &fakeProvider{streamErr: errors.New("provider exploded")}, sess, 60000)
	if tokens, budget := a.guardedSummaryInputTokens(foldRegionOf(a)), a.summaryInputBudget(nil, ""); budget <= 0 || tokens <= budget {
		t.Fatalf("fixture fold is %d tokens against a %d budget; the shortening path is not exercised", tokens, budget)
	}

	if err := prepareContext(context.Background(), a, CompactionTriggerOverflow); err != nil {
		t.Fatalf("prepare = %v, want a degraded fold", err)
	}
	if !degradedFold(a) {
		t.Errorf("no degraded fold committed: receipt=%+v", a.sess.compactionState.LastReceipt)
	}
}

// Below the hard ceiling the turn still goes out, so a failed summary must stay
// a failure: degrading here would fold a recoverable view without a summary and
// spend the mechanical fold on a turn that never needed it.
func TestPressureBelowHardCeilingKeepsTheFailure(t *testing.T) {
	sess := foldableSessionOverForce(6)
	a := agentOverForce(t, &fakeProvider{streamErr: errors.New("provider down")}, sess)
	if est, hard := a.estimatedPromptTokens(sess.Messages), a.hardInputCeiling(); est >= hard {
		t.Fatalf("fixture estimates %d tokens against a %d ceiling; it is not below it", est, hard)
	}

	if err := prepareContext(context.Background(), a, CompactionTriggerPressure); err != nil {
		t.Fatalf("prepare = %v, want the turn to proceed unfolded", err)
	}
	if degradedFold(a) {
		t.Error("a recoverable view was folded without a summary")
	}
	if r := a.sess.compactionState.LastReceipt; r == nil || (r.Status != "blocked" && r.Status != "failed") {
		t.Errorf("receipt = %+v, want the failure recorded so the summary is not paid for twice", r)
	}
}

// Cancellation is the user's decision, not a summarizer failure: it must keep
// its error and leave the projection alone.
func TestCallerCancellationDoesNotDegrade(t *testing.T) {
	sess := foldableSessionOverForce(6)
	a := agentOverForce(t, &fakeProvider{hang: true}, sess)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := prepareContext(ctx, a, CompactionTriggerOverflow); err == nil {
		t.Fatal("cancelled prepare reported success")
	}
	if degradedFold(a) {
		t.Error("cancellation installed a degraded fold; it should change nothing")
	}
}

// TestSummaryBudgetAccountsForPrefix guards the 2026-08-12 overflow: the
// summarizer request rides the main-request prefix, so the fold budget must
// shrink as the prefix grows. Without the deduction a ~900k-token prefix plus
// a window-sized fold still overflowed and forced a degraded mechanical fold.
func TestSummaryBudgetAccountsForPrefix(t *testing.T) {
	a := agentOverForceWindow(t, &fakeProvider{}, foldableSessionOverForce(2), 1_000_000)
	base := a.summaryInputBudget(nil, "")
	if base <= 0 {
		t.Fatalf("baseline budget = %d, want > 0", base)
	}
	big := make([]provider.Message, 0, 600)
	for range 600 {
		big = append(big, provider.Message{Role: provider.RoleUser, Content: strings.Repeat("x", 1500)})
	}
	withPrefix := a.summaryInputBudget(big, "")
	if withPrefix >= base {
		t.Fatalf("budget with 900k-char prefix = %d, want < baseline %d (prefix must be deducted)", withPrefix, base)
	}
	if withPrefix <= 0 {
		t.Fatalf("budget with prefix = %d, want > 0 (window can still host a small fold)", withPrefix)
	}
}

// TestKeepFoldWithinSummaryBudgetKeepsVerbatim proves the budget trimmer moves
// overflow messages back to kept verbatim instead of dropping them, so a
// summarize-side trim can never shed a user turn.
func TestKeepFoldWithinSummaryBudgetKeepsVerbatim(t *testing.T) {
	a := agentOverForceWindow(t, &fakeProvider{}, foldableSessionOverForce(2), 1_000_000)
	prefix := make([]provider.Message, 0, 500)
	for range 500 {
		prefix = append(prefix, provider.Message{Role: provider.RoleUser, Content: strings.Repeat("p", 1500)})
	}
	var fold []provider.Message
	for range 2500 {
		fold = append(fold, provider.Message{Role: provider.RoleUser, Content: "keep-me-" + strings.Repeat("z", 400)})
	}
	kept, out := a.keepFoldWithinSummaryBudget(prefix, nil, fold, 0, 0)
	if len(out) == 0 || len(out) == len(fold) {
		t.Fatalf("fold %d → %d messages, want a partial trim", len(fold), len(out))
	}
	for _, m := range kept {
		if !strings.HasPrefix(m.Content, "keep-me-") {
			t.Fatalf("kept message %q is not verbatim overflow text", m.Content[:16])
		}
	}
	if got := a.guardedSummaryInputTokens(out); got > a.summaryInputBudget(prefix, "") {
		t.Fatalf("trimmed fold still exceeds the prefix-aware budget")
	}
}

// TestDegradedFoldKeepsAllUserMessages is the data-safety contract: when the
// summarizer fails and the fold is committed mechanically, every user turn
// keeps its text verbatim in the projection — no turn may survive "through
// the summary" that never existed.
func TestDegradedFoldKeepsAllUserMessages(t *testing.T) {
	// Large window: every user turn fits under the trigger ceiling, so all of
	// them must survive verbatim — none may depend on a summary that never
	// existed.
	sess := foldableSessionOverForce(120)
	a := agentOverForceWindow(t, &fakeProvider{streamErr: errors.New("provider exploded")}, sess, 200000)
	if err := prepareContext(context.Background(), a, CompactionTriggerOverflow); err != nil {
		t.Fatalf("prepare = %v, want a degraded fold", err)
	}
	if !degradedFold(a) {
		t.Fatalf("fixture did not degrade; test premise broken")
	}
	wantUsers := 0
	for _, m := range sess.Messages {
		if m.Role == provider.RoleUser {
			wantUsers++
		}
	}
	projUsers := 0
	for _, m := range a.sess.compactionState.Projection.Messages {
		if m.Role == provider.RoleUser && !isCompactionSummary(m) {
			projUsers++
		}
	}
	if projUsers != wantUsers {
		t.Fatalf("projection keeps %d user turns, want all %d verbatim", projUsers, wantUsers)
	}

	// Tight window: the same safety loop must still land under the trigger
	// ceiling — a candidate at/above it would be rejected and re-trigger
	// every turn instead of stabilizing.
	tight := agentOverForceWindow(t, &fakeProvider{streamErr: errors.New("provider exploded")}, foldableSessionOverForce(120), 60000)
	if err := prepareContext(context.Background(), tight, CompactionTriggerOverflow); err != nil {
		t.Fatalf("tight prepare = %v, want a degraded fold", err)
	}
	if !degradedFold(tight) {
		t.Fatalf("tight fixture did not degrade; test premise broken")
	}
	tightUsers := 0
	for _, m := range tight.sess.compactionState.Projection.Messages {
		if m.Role == provider.RoleUser && !isCompactionSummary(m) {
			tightUsers++
		}
	}
	if tightUsers == 0 {
		t.Fatalf("tight projection kept no user turns")
	}
	if got := tight.estimatedPromptTokens(tight.sess.compactionState.Projection.Messages); got >= tight.compactTrigger() {
		t.Fatalf("tight degraded projection %d still at/above trigger %d (would re-trigger)", got, tight.compactTrigger())
	}
}

func TestSummarizerBudgetIgnoresMainRequestObservedSize(t *testing.T) {
	// Regression for 2026-08-12 22:56: the summarizer request carries its own
	// shape (system + prefix + fold, no retained tail), so admission must not
	// substitute the main request's observed prompt size — a 1.38M observed
	// main prompt (tail included) falsely rejected a ~300k summarizer request
	// and degraded the fold, dropping 98 user turns.
	prov := &sharedWindowTestProvider{budget: 128 * 1024, shared: true}
	a := &Agent{
		agentConfig: agentConfig{contextWindow: 1_048_576, maxOutputTokens: 128 * 1024},
		svc:         agentServices{prov: prov},
		sess:        sessionRuntime{output: outputBudgetState{outputBudget: 128 * 1024}},
	}
	// Simulate the observed main prompt (large, includes the retained tail).
	a.storeLatestRequestUsage(&provider.Usage{PromptTokens: 1_385_656})
	// The summarizer request is small: system + a modest fold.
	req := provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: summarySystemPrompt},
			{Role: provider.RoleUser, Content: strings.Repeat("fold region text ", 10_000)},
		},
		MaxTokens: 4096,
	}
	est := a.estimatedRequestTokens(req)
	t.Logf("est=%d shapeChars=%d", est, a.requestCalibrationShape(req).requestChars)
	got, clipped, err := a.effectiveOutputBudget(req, false)
	if err != nil {
		t.Fatalf("summarizer rejected by main-request observed size: %v", err)
	}
	// 0 + !clipped is the pass signal: the request fits, no clipping needed.
	if clipped {
		t.Fatalf("summarizer should not be clipped: got=%d", got)
	}
	// The same request with useObserved=true is what the main request does;
	// the observed substitution is intentional there (fresh-agent fallback).
	if _, _, err := a.effectiveOutputBudget(req, true); err == nil {
		t.Fatal("useObserved=true should report overflow against the 1.38M observed prompt")
	}
}
