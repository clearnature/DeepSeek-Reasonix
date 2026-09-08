package agent

import "reasonix/internal/jobs"

// SetCompletionCallback registers a handler fired outside the store lock
// after each teammate job reaches a terminal state (HandleJobDone flipped the
// member back to idle). The leader digest round hooks here so completion
// feedback reaches the model without waiting for the next user turn; nil
// disables. Lives in its own file so the callback API can grow without
// pushing teammate_store.go past its repolint ceiling.
func (ts *TeammateStore) SetCompletionCallback(fn func(name string, status jobs.Status)) {
	ts.mu.Lock()
	ts.completionFn = fn
	ts.mu.Unlock()
}
