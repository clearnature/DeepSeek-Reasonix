package cli

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"reasonix/internal/agent"
	"reasonix/internal/i18n"
)

// compactDoneMsg reports that an async /compact pass returned. The card was
// already drawn from the CompactionDone event; this carries the terminal
// verdict — folded, declined, or failed.
type compactDoneMsg struct {
	verdict agent.CompactVerdict
	err     error
}

// runCompact sends the request off the Update loop: compaction makes a network
// summarizer call, and the TUI must not freeze behind it.
func (m chatTUI) runCompact(args string) tea.Cmd {
	req := compactRequestFrom(args)
	ctrl := m.ctrl
	return func() tea.Msg {
		verdict, err := ctrl.Compact(context.Background(), req)
		return compactDoneMsg{verdict: verdict, err: err}
	}
}

// compactRequestFrom reads what the user typed after /compact. A leading
// --force is the one word that means "spend even where the fold does not pay
// for itself"; everything else is focus guidance for the summary.
func compactRequestFrom(args string) agent.CompactRequest {
	req := agent.CompactRequest{}
	for {
		rest, ok := strings.CutPrefix(args, "--force")
		if !ok || (rest != "" && !strings.HasPrefix(rest, " ")) {
			break
		}
		req.IgnoreEconomics = true
		args = strings.TrimSpace(rest)
	}
	req.Instructions = args
	return req
}

// reportCompactDone says which of the three things happened. A request the host
// declined is not a failure and not silence: the reason it settled is printed,
// so nobody has to guess why a command they typed appeared to do nothing.
func (m chatTUI) reportCompactDone(msg compactDoneMsg) {
	switch {
	case msg.err != nil && agent.IsCompactionDeclined(msg.err):
		m.notice(fmt.Sprintf("%s — %s", i18n.M.SlashCompactDeclined, agent.CompactionDeclineReason(msg.err)))
	case msg.err != nil:
		m.notice(fmt.Sprintf("%s: %v", i18n.M.SlashCompactFailed, msg.err))
	case !msg.verdict.Compacted():
		m.notice(fmt.Sprintf("%s — %s", i18n.M.SlashCompactDeclined, agent.CompactDeclineText(msg.verdict.Reason)))
	default:
		_ = m.ctrl.Snapshot()
		m.followSessionLease()
	}
}
