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
}

type mailboxKey struct{}

// WithMailbox attaches a teammate mailbox to a context.
func WithMailbox(ctx context.Context, m Mailbox) context.Context {
	return context.WithValue(ctx, mailboxKey{}, m)
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
	return json.RawMessage(`{"type":"object","properties":{"target":{"type":"string","description":"Teammate name to deliver the mail to."},"text":{"type":"string","description":"Message body for the teammate."}},"required":["target","text"]}`)
}

func (teamMessage) Description() string {
	return "Deliver mail to another teammate's inbox (teammate-to-teammate). " +
		"The recipient flushes it into its next assignment's steer queue."
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
	if err := m.PostMail(in.Target, in.Text); err != nil {
		return "", fmt.Errorf("team_message: %w", err)
	}
	return fmt.Sprintf("mail delivered to teammate %q (flushes on its next assignment)", in.Target), nil
}
