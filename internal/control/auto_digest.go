package control

import (
	"context"
	"strings"
	"time"

	"reasonix/internal/jobs"
)

// The leader digest round is the qwen waitForTeammateActivity analog: a
// teammate job completion wakes the leader model without waiting for the next
// user turn. The worker drains completed-job notes and runs one synchronous
// turn that asks the model to digest them and report. Design:
// docs/team/20260909-live-feedback-design.md (6.1-6.7).
const digestRoundTimeout = 10 * time.Minute

// teamDigestInstruction frames the auto round as a digest-and-report turn and
// forbids redispatch, so a completion cannot cascade into new background work
// from an unattended model turn (loop guard; mechanism gate is prompt-level in
// v1, see design 6.3).
const teamDigestInstruction = "Digest the results and report; if a result drifts from its task you may say so. This is an automatic digest round — do not dispatch new background tasks now."

// startDigestRound wires the completion callback and launches the single
// digest worker. Only when orchestration exists and DigestAuto is on (tests
// and headless harnesses stay off so a completion never runs an unattended
// model turn). Called from New; Close shuts the worker down.
func (c *Controller) startDigestRound() {
	if c.teammates == nil || !c.autoDigest {
		return
	}
	c.digestCh = make(chan struct{}, 1)
	c.digestDone = make(chan struct{})
	c.digestOnce.Do(func() {
		c.teammates.SetCompletionCallback(func(name string, _ jobs.Status) {
			// Cap 1 collapses a burst of completions into one round.
			select {
			case c.digestCh <- struct{}{}:
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
		case <-c.digestCh:
			// Batch: drain every completion signalled while this round waited.
		loop:
			for {
				select {
				case <-c.digestCh:
				default:
					break loop
				}
			}
			c.runDigestRound()
		}
	}
}

// runDigestRound runs one model turn that digests the completed jobs. The
// completion notes are drained here (the user-turn compose path drains them
// only when a user submits), so an idle leader still learns the results.
// ErrTurnRunning means a user turn owns the controller — it already injected
// the notes via compose, so this round is dropped, not retried.
func (c *Controller) runDigestRound() {
	ctx, cancel := context.WithTimeout(context.Background(), digestRoundTimeout)
	defer cancel()
	input := c.digestPrompt()
	if input == "" {
		return
	}
	if err := c.RunTurn(ctx, input); err != nil {
		if err == ErrTurnRunning {
			return // the user turn that owns the controller carries the notes
		}
		c.notice("team digest failed: " + err.Error())
	}
}

// digestPrompt assembles the auto-round input from the drained completion
// notes plus the live roster. Empty when there is nothing to digest.
func (c *Controller) digestPrompt() string {
	var note string
	if c.jobs != nil {
		note = c.jobs.DrainCompletedNoteForSession(c.parentSessionID())
	}
	if strings.TrimSpace(note) == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("<team_digest>\n")
	b.WriteString("The following teammate task(s) completed since your last turn. ")
	b.WriteString(teamDigestInstruction + "\n")
	b.WriteString(note)
	b.WriteString("\n</team_digest>")
	return b.String()
}
