package jobs

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
)

// TestLeaseExemptSkipsStartObserver locks in the A1/A2 contract: read-only
// background jobs must not retain the workspace lease (StartForSessionReadonly
// skips the start observer), while writer jobs still do — otherwise a qwen-style
// read-only researcher locks the leader out of its own workspace.
func TestLeaseExemptSkipsStartObserver(t *testing.T) {
	m := NewManager(nil)
	t.Cleanup(m.Close)
	var retains atomic.Int32
	m.onJobStart = func(<-chan struct{}) { retains.Add(1) }

	run := func(ctx context.Context, _ io.Writer) (string, error) { return "ok", nil }
	// Writer job retains (observer fires synchronously at start).
	w := m.StartForSession("s1", "task", "writer", run)
	_ = w
	if retains.Load() != 1 {
		t.Fatalf("writer retains = %d, want 1", retains.Load())
	}
	// Read-only job does not retain.
	r := m.StartForSessionReadonly("s1", "task", "reader", run)
	_ = r
	if retains.Load() != 1 {
		t.Fatalf("readonly retains = %d, want still 1 (exempt)", retains.Load())
	}
}
