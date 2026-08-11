package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// P9 plan-approval gate: a teammate submits its plan, the leader reviews it
// as a <plan-approval-request> envelope on the next turn and answers with
// /team-approve; the verdict rides the P3 steer queue back to the teammate,
// which continues only on allow. Decisions rest with the leader/user (15
// decisions / 6 discipline / 5 execution team philosophy) — never with the
// executing teammate.

// approvalReq is one pending plan-approval request. The verdict channel is
// write-once: Approve/Reject delivers exactly one decision.
// ApprovalRequest is the exported view of a pending P9 plan-approval
// request, consumed by the controller's turn injection (input.go).
type ApprovalRequest = approvalReq

type approvalReq struct {
	RequestID string
	Teammate  string
	Plan      string
	At        int64
}

// RequestApproval records a teammate's plan for the leader (P9). The request
// is persisted under the leader's approvals inbox so it survives restarts;
// a duplicate request_id is rejected (no replay).
func (ts *TeammateStore) RequestApproval(from, requestID, plan string) error {
	from = strings.TrimSpace(from)
	requestID = strings.TrimSpace(requestID)
	plan = strings.TrimSpace(plan)
	if from == "" || requestID == "" || plan == "" {
		return fmt.Errorf("approval request needs a teammate, request_id, and plan")
	}
	ts.mu.Lock()
	if _, ok := ts.teammates[from]; !ok {
		ts.mu.Unlock()
		return fmt.Errorf("teammate %q not found", from)
	}
	if _, dup := ts.approvals[requestID]; dup {
		ts.mu.Unlock()
		return fmt.Errorf("duplicate approval request id %q", requestID)
	}
	root := ts.inboxRoot
	ts.mu.Unlock()

	req := &approvalReq{
		RequestID: requestID,
		Teammate:  from,
		Plan:      plan,
		At:        time.Now().Unix(),
	}
	// Persist under <inboxRoot>/leader/approvals/<id>.json so a restart keeps
	// the pending request visible to the leader.
	if root != "" {
		dir := filepath.Join(root, "leader", "approvals")
		if err := os.MkdirAll(dir, 0o755); err == nil {
			payload, _ := json.Marshal(req)
			_ = os.WriteFile(filepath.Join(dir, sanitizeMailName(requestID)+".json"), payload, 0o644)
		}
	}

	ts.mu.Lock()
	ts.approvals[requestID] = req
	ts.mu.Unlock()
	ts.saveSnapshot()
	return nil
}

// PendingApprovals returns the pending plan-approval requests for the
// leader's turn injection (P9), sorted by submission time (FIFO).
func (ts *TeammateStore) PendingApprovals() []ApprovalRequest {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	out := make([]approvalReq, 0, len(ts.approvals))
	for _, r := range ts.approvals {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At < out[j].At })
	return out
}

// Approve answers a plan-approval request (P9). decision=true means allow:
// the verdict rides the teammate's P3 steer queue so its next turn sees
// <plan-approval-verdict decision="allow"|"deny" request_id="...">. The
// request is consumed (removed from pending) regardless of the decision.
func (ts *TeammateStore) Approve(requestID string, decision bool, session string) error {
	requestID = strings.TrimSpace(requestID)
	ts.mu.Lock()
	req, ok := ts.approvals[requestID]
	if !ok {
		ts.mu.Unlock()
		return fmt.Errorf("no pending approval request %q", requestID)
	}
	delete(ts.approvals, requestID)
	tm, hasTM := ts.teammates[req.Teammate]
	jm := ts.jm
	root := ts.inboxRoot
	ts.mu.Unlock()

	// Consume the persisted request (best effort).
	if root != "" {
		_ = os.Remove(filepath.Join(root, "leader", "approvals", sanitizeMailName(requestID)+".json"))
	}

	verdict := "deny"
	if decision {
		verdict = "allow"
	}
	text := fmt.Sprintf(`<plan-approval-verdict decision="%s" request_id="%s"/>`, verdict, requestID)
	if jm != nil && hasTM && tm.LastJobID != "" {
		_ = jm.SendMessageForSession(session, tm.LastJobID, text)
	}
	ts.saveSnapshot()
	return nil
}
