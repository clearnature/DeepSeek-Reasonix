package agent

import (
	"os"
	"testing"
	"time"

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

// TestBoardPersistsAcrossSnapshot round-trips the board through the crash
// snapshot: a new store loading the same snapshot sees the same items.
func TestBoardPersistsAcrossSnapshot(t *testing.T) {
	snap := t.TempDir() + "/team-snapshot.json"
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	ts.SetSnapshotPath(snap)
	t.Cleanup(ts.Close)
	id, err := ts.BoardCreate("persisted task")
	if err != nil {
		t.Fatalf("BoardCreate: %v", err)
	}
	if err := ts.BoardClaim(id, "alpha"); err != nil {
		t.Fatalf("BoardClaim: %v", err)
	}
	// saveSnapshot is async; wait for the file, then a second store loads it.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(snap); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("snapshot never written")
		}
		time.Sleep(20 * time.Millisecond)
	}
	ts2 := NewTeammateStore(testTaskToolForTeam(t), jm)
	ts2.SetSnapshotPath(snap)
	t.Cleanup(ts2.Close)
	items := ts2.BoardList()
	if len(items) != 1 || items[0].ID != id || items[0].Owner != "alpha" || items[0].Status != BoardClaimed {
		t.Fatalf("restored board = %+v, want one claimed item %s", items, id)
	}
}

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

// TestReleaseBoardTasksReturnsClaimedToOpen locks the task_stop release: a
// stopped owner's claimed items go back to the open pool.
func TestReleaseBoardTasksReturnsClaimedToOpen(t *testing.T) {
	ts := testBoardStore()
	a, _ := ts.BoardCreate("one")
	b, _ := ts.BoardCreate("two")
	_ = ts.BoardClaim(a, "alpha")
	_ = ts.BoardClaim(b, "beta")
	if n := ts.ReleaseBoardTasks("alpha"); n != 1 {
		t.Fatalf("released = %d, want 1", n)
	}
	for _, v := range ts.BoardList() {
		if v.ID == a && (v.Status != BoardOpen || v.Owner != "") {
			t.Fatalf("released item = %+v, want open unowned", v)
		}
		if v.ID == b && v.Owner != "beta" {
			t.Fatalf("other owner's item changed: %+v", v)
		}
	}
}
