package agent

import (
	"fmt"
	"strings"
	"time"
)

// TaskBoardItem is one entry on the qwen-style claimable team task board.
// Unlike the dependency-tree tasks map (assign-or-wait), the board holds
// work that any teammate can claim: Open tasks await an owner; Claimed ones
// run; Done is set by the teammate via task_update. Lives in its own file (an
// upstream-absent path) so a convergence cannot silently drop the board.
type TaskBoardItem struct {
	ID        string
	Prompt    string
	Owner     string
	Status    string // open | claimed | done
	CreatedAt time.Time
}

const (
	BoardOpen    = "open"
	BoardClaimed = "claimed"
	BoardDone    = "done"
)

// BoardCreate publishes an unassigned task for teammate self-claiming.
func (ts *TeammateStore) BoardCreate(prompt string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", fmt.Errorf("board create: prompt is required")
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	id := fmt.Sprintf("board-%d", time.Now().UnixNano())
	ts.board[id] = &TaskBoardItem{ID: id, Prompt: prompt, Status: BoardOpen, CreatedAt: time.Now()}
	if ts.snapshotPath != "" {
		ts.saveSnapshot()
	}
	return id, nil
}

// BoardList returns the claimable board snapshot.
func (ts *TeammateStore) BoardList() []TaskBoardItem {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	out := make([]TaskBoardItem, 0, len(ts.board))
	for _, v := range ts.board {
		out = append(out, *v)
	}
	return out
}

// BoardClaim assigns an open item to a teammate. Returns an error if the item
// is already claimed (idempotent re-claim by the same owner is allowed).
func (ts *TeammateStore) BoardClaim(id, owner string) error {
	owner = strings.TrimSpace(owner)
	if id == "" || owner == "" {
		return fmt.Errorf("board claim: id and owner are required")
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	item, ok := ts.board[id]
	if !ok {
		return fmt.Errorf("board item %q not found", id)
	}
	if item.Status == BoardDone {
		return fmt.Errorf("board item %q is done", id)
	}
	if item.Status == BoardClaimed && item.Owner != owner {
		return fmt.Errorf("board item %q already claimed by %q", id, item.Owner)
	}
	item.Status = BoardClaimed
	item.Owner = owner
	if ts.snapshotPath != "" {
		ts.saveSnapshot()
	}
	return nil
}

// BoardUpdate sets an item's status (done by the completing teammate, or
// reopened to open). Only the current owner can change a claimed item.
func (ts *TeammateStore) BoardUpdate(id, owner, status string) error {
	status = strings.TrimSpace(status)
	ts.mu.Lock()
	defer ts.mu.Unlock()
	item, ok := ts.board[id]
	if !ok {
		return fmt.Errorf("board item %q not found", id)
	}
	switch status {
	case BoardDone:
		if item.Status == BoardClaimed && item.Owner != strings.TrimSpace(owner) {
			return fmt.Errorf("board item %q owned by %q, not %q", id, item.Owner, owner)
		}
		item.Status = BoardDone
	case BoardOpen:
		item.Status = BoardOpen
		item.Owner = ""
	default:
		return fmt.Errorf("board update: unknown status %q", status)
	}
	if ts.snapshotPath != "" {
		ts.saveSnapshot()
	}
	return nil
}
