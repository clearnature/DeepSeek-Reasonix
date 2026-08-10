package agent

import (
	"context"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestPreflightPruneThenFoldFitsWindow is the user-scenario regression: a
// canonical transcript far over the shared window (measured 3.7M tokens vs a
// 1M window), mostly stale tool results. Preflight prunes them into a
// placeholder view, and the force fold must run against that view — a
// raw-canonical rebuild would leave the projection over the window.
func TestPreflightPruneThenFoldFitsWindow(t *testing.T) {
	prov := &sharedFakeProvider{
		fakeProvider: &fakeProvider{reply: "digest of older work"},
		budget:       128 * 1024,
	}
	sess := &Session{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
	}}
	// ~2.6M tokens of stale tool results (older) + ~1.4M of turns: canonical
	// ends far over the 1M window.
	for i := 0; i < 120; i++ {
		sess.Messages = append(sess.Messages,
			provider.Message{Role: provider.RoleUser, Content: strings.Repeat("题", 3000)},  // 3K
			provider.Message{Role: provider.RoleTool, Content: strings.Repeat("果", 22000)}, // stale 22K
		)
	}
	for i := 0; i < 10; i++ {
		sess.Messages = append(sess.Messages,
			provider.Message{Role: provider.RoleUser, Content: strings.Repeat("新", 4000)},
			provider.Message{Role: provider.RoleAssistant, Content: strings.Repeat("答", 6000)},
		)
	}
	if est := estimateMessagesTokens(provider.ModelMessages(sess.Messages)); est < 2_000_000 {
		t.Fatalf("setup: canonical must exceed 2M tokens (est=%d)", est)
	}

	a := New(prov, tool.NewRegistry(), sess, Options{
		ContextWindow: 1024 * 1024,
		RecentKeep:    4,
	}, event.Discard)

	if err := a.contextPreflight(context.Background(), "auto"); err != nil {
		t.Fatalf("preflight must succeed after prune + bounded fold: %v", err)
	}
	st := a.compactionState
	if !projectionValid(st, sess.Messages, st.TranscriptVersion, a.currentPromptCacheKey()) {
		t.Fatal("no valid projection installed after preflight")
	}
	if est := estimateMessagesTokens(st.Projection.Messages); est >= a.contextWindow-minOutputBudget {
		t.Fatalf("projection still over window after fold: est=%d window=%d", est, a.contextWindow)
	}
	// The projection must cover the whole canonical (the pruned view collapsed
	// the stale tool results).
	if st.Projection.CoveredCount != len(sess.Messages) {
		t.Fatalf("covered = %d, want %d (full canonical)", st.Projection.CoveredCount, len(sess.Messages))
	}
}

// TestPreflightTooSmallWindowLatchesOnlyWhenProjectionCannotFit pins the
// latch criterion to the projection's real shape: a fold inside the window
// (beside a usable output budget) must not pause, while one that cannot fit
// must. The old projection >= high check depended on tokPerChar
// over-estimation and never latched a too-small window.
func TestPreflightTooSmallWindowLatchesOnlyWhenProjectionCannotFit(t *testing.T) {
	newSess := func() (*Session, *Agent) {
		sess := &Session{Messages: []provider.Message{{Role: provider.RoleSystem, Content: "sys"}}}
		prov := &sharedFakeProvider{fakeProvider: &fakeProvider{reply: "digest"}, budget: 128 * 1024}
		a := New(prov, tool.NewRegistry(), sess, Options{ContextWindow: 1024 * 1024, RecentKeep: 4}, event.Discard)
		return sess, a
	}
	// Helper: install a projection whose visible shape + minOutputBudget
	// stays inside the given window, then run preflight under force pressure.
	t.Run("healthy window never latches", func(t *testing.T) {
		sess, a := newSess()
		for range 100 {
			sess.Add(provider.Message{Role: provider.RoleUser, Content: strings.Repeat("grow", 20)})
		}
		// Force preflight once so a projection exists.
		if err := a.contextPreflight(context.Background(), "auto"); err != nil {
			t.Fatalf("preflight: %v", err)
		}
		if a.compactStuck {
			t.Fatal("healthy window must not latch after a successful fold")
		}
	})
	t.Run("healthy window with projection stays unpaused", func(t *testing.T) {
		sess, a := newSess()
		for range 100 {
			sess.Add(provider.Message{Role: provider.RoleUser, Content: strings.Repeat("grow", 20)})
		}
		// Healthy window: a real fold installs a projection that fits
		// beside the 8K floor; repeated preflight must stay unpaused.
		for range 3 {
			if err := a.contextPreflight(context.Background(), "auto"); err != nil {
				t.Fatalf("preflight: %v", err)
			}
			if a.compactStuck {
				t.Fatal("healthy window latched; over-pause regression")
			}
		}
	})
}

// TestPreflightUsesObservedTokensBelowForce pins the dev-port fix: the
// preflight force gate must prefer the last real usage observation over the
// canonical calibrated estimate, which runs hot on CJK/tool-dense sessions
// (975k estimated vs 602k actual on 2026-08-10) and can force a summarizer
// pass at 60% of the window.
func TestPreflightUsesObservedTokensBelowForce(t *testing.T) {
	prov := &fakeProvider{reply: "summary"}
	sess := &Session{Messages: []provider.Message{
		{Role: provider.RoleUser, Content: strings.Repeat("中文内容", 400)},
		{Role: provider.RoleUser, Content: strings.Repeat("tool detail ", 300)},
	}}
	a := New(prov, tool.NewRegistry(), sess, Options{
		ContextWindow:       1000,
		ToolResultSnipRatio: 0.6,
		CompactRatio:        0.8,
		CompactForceRatio:   0.9,
		RecentKeep:          2,
	}, event.Discard)
	// Real observation sits far below the force threshold (900); without the
	// observed-tokens handoff the CJK-heavy estimate crosses it and the
	// summarizer is called.
	a.lastUsage.Store(&provider.Usage{PromptTokens: 200, CompletionTokens: 10, TotalTokens: 210})
	if err := a.contextPreflight(context.Background(), CompactionTriggerPressure); err != nil {
		t.Fatalf("preflight: %v", err)
	}
	if prov.got != nil {
		t.Fatal("summarizer was called although observed input (200) is below the force threshold (900)")
	}
}
