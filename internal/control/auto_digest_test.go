package control

import (
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// TestDigestPromptEmptyWithoutNotes locks the auto-round gating: with no
// completion notes drained there is nothing to digest, so the prompt is empty
// and the worker skips the round (no unattended model call).
func TestDigestPromptEmptyWithoutNotes(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	c := &Controller{jobs: jm}
	if p := c.digestPrompt(); p != "" {
		t.Fatalf("digestPrompt with no drained notes = %q, want empty", p)
	}
}

// TestDigestInstructionForbidsRedispatch locks the loop guard wording: the
// automatic digest round must tell the model not to dispatch new background
// tasks (design 6.3; prompt-level gate in v1).
func TestDigestInstructionForbidsRedispatch(t *testing.T) {
	if !strings.Contains(teamDigestInstruction, "do not dispatch new background tasks") {
		t.Fatalf("digest instruction must forbid redispatch: %q", teamDigestInstruction)
	}
}
