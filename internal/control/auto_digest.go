package control

import (
	"context"
	"strings"
	"time"

	"reasonix/internal/jobs"
)

// The leader digest round is the qwen waitForTeammateActivity analog: a
// teammate job completion wakes the leader model without waiting for the next
// user turn. The round is a lightweight completion notice — the teammate's
// result envelope already rides the leader transcript prefix (appended when
// the fork finished), so the digest turn's model can read it and report.
// Design: docs/team/20260909-live-feedback-design.md (6.1-6.7).
const digestRoundTimeout = 10 * time.Minute

// teamDigestInstruction frames the auto round as a digest-and-report turn and
// forbids redispatch, so a completion cannot cascade into new background work
// from an unattended model turn (loop guard; mechanism gate is prompt-level in
// v1, see design 6.3).
const teamDigestInstruction = "Summarize the completed teammate task and its delivered result into the running work state, then report concisely. This is an automatic digest round — do not dispatch new background tasks now."

// startDigestRound wires the completion callback and launches the single
// digest worker. Only when orchestration exists and DigestAuto is on (tests
// and headless harnesses stay off so a completion never runs an unattended
// model turn). Called from New; Close shuts the worker down.
func (c *Controller) startDigestRound() {
	if c.teammates == nil || !c.autoDigest {
		return
	}
	c.digestCh = make(chan string, 1)
	c.digestDone = make(chan struct{})
	c.digestOnce.Do(func() {
		c.teammates.SetCompletionCallback(func(name string, _ jobs.Status) {
			// Cap 1 collapses a burst of completions into one round.
			select {
			case c.digestCh <- name:
			default:
			}
		})
		go c.digestWorker()
	})
}

// stopDigestRound closes the worker from Close.
func (c *Controller) stopDigestRound() {
	if c.digestDone == nil {
		return
	}
	close(c.digestDone)
}

func (c *Controller) digestWorker() {
	for {
		select {
		case <-c.digestDone:
			return
		case name := <-c.digestCh:
			// Batch: drain every completion signalled while this round waited.
			for {
				select {
				case n := <-c.digestCh:
					name = n
				default:
					goto run
				}
			}
		run:
			c.runDigestRound(name)
		}
	}
}

// runDigestRound runs one model turn that digests the completed teammate job.
// ErrTurnRunning means a user turn owns the controller — it already injected
// the completion context, so this round is dropped, not retried.
func (c *Controller) runDigestRound(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), digestRoundTimeout)
	defer cancel()
	if err := c.RunTurn(ctx, c.digestPrompt(name)); err != nil {
		if err == ErrTurnRunning {
			return // the user turn that owns the controller carries the context
		}
		c.notice("team digest failed: " + err.Error())
	}
}

// digestPrompt assembles the auto-round input for a completed teammate.
func (c *Controller) digestPrompt(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("<team_digest>\n")
	b.WriteString("Teammate ")
	b.WriteString(name)
	b.WriteString(" finished its task. ")
	b.WriteString(teamDigestInstruction)
	b.WriteString("\n</team_digest>")
	return b.String()
}
