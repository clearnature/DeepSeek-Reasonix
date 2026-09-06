package agent

import (
	"context"
	"sync"
)

// BackgroundizeSignal is the P4 foreground→backgroundize signal for the in-flight
// foreground turn. spawnGuardedTurn stamps a fresh signal into the turn context
// and records it here; finishGuardedTurn clears it. The signal is injected into the
// turn context with WithBackgroundizeSignal; the run-loop checkpoint consumes
// it exactly once (Request returns true only for the first request, so a
// double-click or an auto+manual pair collapses to a single handoff). The
// signal mutex is a short, independent critical section and never nests jobs
// Manager or job locks.
type BackgroundizeSignal struct {
	mu        sync.Mutex
	requested bool
}

// NewBackgroundizeSignal returns a fresh, unrequested backgroundize signal.
func NewBackgroundizeSignal() *BackgroundizeSignal { return &BackgroundizeSignal{} }

// Request marks the signal requested and reports whether this is the first
// request. Repeated requests are idempotent no-ops returning false.
func (s *BackgroundizeSignal) Request() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	first := !s.requested
	s.requested = true
	return first
}

// Requested reports whether the signal has been requested.
func (s *BackgroundizeSignal) Requested() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requested
}

type backgroundizeSignalKey struct{}

// WithBackgroundizeSignal injects the signal a foreground task's run loop
// checks at its iteration boundary. It inherits down the whole sub-agent ctx
// chain (withAgentContext only rebinds jobs/memory/planmode), so a signal
// stamped on the parent turn context reaches the nested task's run loop.
func WithBackgroundizeSignal(ctx context.Context, s *BackgroundizeSignal) context.Context {
	return context.WithValue(ctx, backgroundizeSignalKey{}, s)
}

// BackgroundizeSignalFromContext retrieves the signal from the context.
func BackgroundizeSignalFromContext(ctx context.Context) *BackgroundizeSignal {
	s, _ := ctx.Value(backgroundizeSignalKey{}).(*BackgroundizeSignal)
	return s
}
