package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/internal/tool"
)

// plan_approval_request lets a teammate submit its plan for the leader's
// approval before executing (P9). The plan rides a fixed envelope to the
// leader's next turn; the leader answers with /team-approve, whose verdict
// comes back on the P3 steer queue. Only teammate fork contexts carry the
// mailbox, so the tool is invisible to the leader and unrelated sub-agents.

func init() {
	tool.RegisterBuiltin(planApprovalRequest{})
}

type planApprovalRequest struct{}

// NewPlanApprovalRequestTool registers the teammate plan-approval tool.
func NewPlanApprovalRequestTool() tool.Tool {
	return planApprovalRequest{}
}

func (planApprovalRequest) Name() string { return "plan_approval_request" }

func (planApprovalRequest) ProviderVisible(ctx context.Context) bool {
	_, ok := MailboxFromContext(ctx)
	return ok
}

func (planApprovalRequest) ReadOnly() bool { return false }

func (planApprovalRequest) SkipEmit() bool { return false }

func (planApprovalRequest) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"request_id":{"type":"string","description":"Unique id for this approval request (generate a random string)."},"plan":{"type":"string","description":"The plan to execute, as markdown. Bounded to ~4KB."}},"required":["request_id","plan"]}`)
}

func (planApprovalRequest) Description() string {
	return "Submit your plan to the leader for approval before executing (P9). " +
		"The leader reviews the plan and replies allow or deny via /team-approve; " +
		"wait for the verdict (it arrives as a steer message) before doing the work."
}

func (planApprovalRequest) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		RequestID string `json:"request_id"`
		Plan      string `json:"plan"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("plan_approval_request: %w", err)
	}
	in.RequestID = strings.TrimSpace(in.RequestID)
	in.Plan = strings.TrimSpace(in.Plan)
	if in.RequestID == "" || in.Plan == "" {
		return "", fmt.Errorf("plan_approval_request: request_id and plan are required")
	}
	m, ok := MailboxFromContext(ctx)
	if !ok || m == nil {
		return "", fmt.Errorf("plan_approval_request: no teammate mailbox in this context")
	}
	sender, _ := ctx.Value(mailboxIdentityKey{}).(string)
	if sender == "" {
		return "", fmt.Errorf("plan_approval_request: needs a teammate identity")
	}
	if err := m.RequestApproval(sender, in.RequestID, in.Plan); err != nil {
		return "", fmt.Errorf("plan_approval_request: %w", err)
	}
	return "approval requested — waiting for the leader's verdict (arrives as a steer message)", nil
}
