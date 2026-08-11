package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/internal/tool"
)

// Mailbox is the P6.2 teammate-to-teammate mail channel implemented by the
// agent-side TeammateStore. The interface lives here (not in agent) so the
// builtin package never imports agent — the store is injected through the
// context under mailboxKey by the agent's teammate sub-agent builder.
type Mailbox interface {
	PostMail(name, text string) error
	// PostMailToLeader routes a message from a teammate into the leader's
	// inbox (P8). from names the sending teammate; the leader's next turn
	// drains the inbox as a <team-messages> envelope.
	PostMailToLeader(from, text string) error
	// RequestApproval submits a plan for the leader's approval (P9). The
	// teammate produces a plan and calls plan_approval_request; the leader
	// sees it as a <plan-approval-request> envelope and answers with
	// /team-approve, whose verdict rides the P3 steer queue back.
	RequestApproval(from, requestID, plan string) error
}

type mailboxKey struct{}
type mailboxIdentityKey struct{}

// WithMailbox attaches a teammate mailbox to a context.
func WithMailbox(ctx context.Context, m Mailbox) context.Context {
	return context.WithValue(ctx, mailboxKey{}, m)
}

// WithMailboxIdentity records the sending teammate's name so the
// team_message tool can stamp mail from="<name>" without trusting model
// text (P8). The leader's own contexts carry no identity.
func WithMailboxIdentity(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, mailboxIdentityKey{}, name)
}

// MailboxFromContext returns the attached mailbox, if any.
func MailboxFromContext(ctx context.Context) (Mailbox, bool) {
	m, ok := ctx.Value(mailboxKey{}).(Mailbox)
	return m, ok
}

// team_message delivers mail to another teammate's inbox (P6.2 teammate
// direct-connect). The target flushes it into its job's steer queue on the
// next assignment. Only teammate sub-agent contexts carry the mailbox, so the
// leader and unrelated sub-agents see this tool hidden.
func init() {
	tool.RegisterBuiltin(teamMessage{})
}

type teamMessage struct{}

// NewTeamMessageTool registers the teammate-to-teammate mail tool.
func NewTeamMessageTool() tool.Tool {
	return teamMessage{}
}

func (teamMessage) Name() string { return "team_message" }

func (teamMessage) ProviderVisible(ctx context.Context) bool {
	_, ok := MailboxFromContext(ctx)
	return ok
}

func (teamMessage) ReadOnly() bool { return false }

func (teamMessage) SkipEmit() bool { return false }

func (teamMessage) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"target":{"type":"string","description":"Teammate name to deliver the mail to, or the reserved target \"leader\" to route it to the leader's inbox (P8)."},"text":{"type":"string","description":"Message body for the teammate."}},"required":["target","text"]}`)
}

func (teamMessage) Description() string {
	return "Deliver mail to another teammate's inbox (teammate-to-teammate). " +
		"The recipient flushes it into its next assignment's steer queue. " +
		"Target \"leader\" is reserved (P8): it routes the mail to the leader's inbox."
}

func (teamMessage) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Target string `json:"target"`
		Text   string `json:"text"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("team_message: %w", err)
	}
	in.Target = strings.TrimSpace(in.Target)
	in.Text = strings.TrimSpace(in.Text)
	if in.Target == "" || in.Text == "" {
		return "", fmt.Errorf("team_message: target and text are required")
	}
	m, ok := MailboxFromContext(ctx)
	if !ok || m == nil {
		return "", fmt.Errorf("team_message: no teammate mailbox in this context")
	}
	// P8: target "leader" routes to the leader's inbox. The sender identity
	// comes from the fork context (WithMailboxIdentity), never from model
	// text, so the envelope's from= is trustworthy.
	if in.Target == "leader" {
		sender, _ := ctx.Value(mailboxIdentityKey{}).(string)
		if sender == "" {
			return "", fmt.Errorf("team_message: leader routing needs a teammate identity (not available in this context)")
		}
		if err := m.PostMailToLeader(sender, in.Text); err != nil {
			return "", fmt.Errorf("team_message: %w", err)
		}
		return fmt.Sprintf("mail delivered to leader (from %q)", sender), nil
	}
	if err := m.PostMail(in.Target, in.Text); err != nil {
		return "", fmt.Errorf("team_message: %w", err)
	}
	return fmt.Sprintf("mail delivered to teammate %q (flushes on its next assignment)", in.Target), nil
}
