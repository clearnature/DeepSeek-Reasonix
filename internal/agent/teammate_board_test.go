package agent

import (
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// testBoardStore builds a store with no task tool (board methods don't need
// job execution) but a manager for job-status constants.
func testBoardStore() *TeammateStore {
	jm := jobs.NewManager(event.Discard)
	_ = jm
	ts := &TeammateStore{board: make(map[string]*TaskBoardItem)}
	return ts
}

// TestBoardClaimThenOnlyOwnerCompletes locks the arbitration: an open item is
// claimable by any teammate; once claimed, only that owner can complete it;
// a second claimer is rejected; completion flips it to done.
func TestBoardClaimThenOnlyOwnerCompletes(t *testing.T) {
	ts := testBoardStore()
	id, err := ts.BoardCreate("research the compression theory")
	if err != nil {
		t.Fatalf("BoardCreate: %v", err)
	}
	// Open -> any member claims.
	if err := ts.BoardClaim(id, "alpha"); err != nil {
		t.Fatalf("claim by alpha: %v", err)
	}
	// Second claimer rejected.
	if err := ts.BoardClaim(id, "beta"); err == nil {
		t.Fatal("second claimer should be rejected")
	}
	// Non-owner cannot complete.
	if err := ts.BoardUpdate(id, "beta", BoardDone); err == nil {
		t.Fatal("non-owner completion should be rejected")
	}
	// Owner completes.
	if err := ts.BoardUpdate(id, "alpha", BoardDone); err != nil {
		t.Fatalf("owner completion: %v", err)
	}
	items := ts.BoardList()
	if len(items) != 1 || items[0].Status != BoardDone {
		t.Fatalf("board item status = %v, want done", items)
	}
}

// TestBoardIdempotentSameOwnerClaim allows a re-claim by the same owner (a
// teammate re-checking its own task), matching qwen TaskUpdate tolerance.
func TestBoardIdempotentSameOwnerClaim(t *testing.T) {
	ts := testBoardStore()
	id, _ := ts.BoardCreate("write the report")
	_ = ts.BoardClaim(id, "alpha")
	if err := ts.BoardClaim(id, "alpha"); err != nil {
		t.Fatalf("same-owner re-claim should be allowed: %v", err)
	}
}

// TestBoardListTaskBoard unmarshals the claimable board for tool display.
func TestBoardListTaskBoard(t *testing.T) {
	ts := testBoardStore()
	_, _ = ts.BoardCreate("task one")
	_, _ = ts.BoardCreate("task two")
	got := ts.BoardList()
	if len(got) != 2 {
		t.Fatalf("BoardList len = %d, want 2", len(got))
	}
	for _, v := range got {
		if v.Status != BoardOpen {
			t.Fatalf("new board item status = %s, want open", v.Status)
		}
	}
}
