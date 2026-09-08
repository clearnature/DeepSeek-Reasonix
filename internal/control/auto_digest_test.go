package control

import (
	"strings"
	"testing"
)

// TestDigestPromptEmptyWithoutName locks the auto-round gating: with no
// completed teammate there is nothing to digest, so the prompt is empty and
// the worker skips the round (no unattended model call).
func TestDigestPromptEmptyWithoutName(t *testing.T) {
	c := &Controller{}
	if p := c.digestPrompt(""); p != "" {
		t.Fatalf("digestPrompt with empty name = %q, want empty", p)
	}
}

// TestDigestPromptNamesTeammate locks the round shape: the input names the
// completed teammate and forbids redispatch (design 6.3; prompt-level gate in
// v1). The result envelope already rides the leader transcript, so the prompt
// stays lightweight — no duplicated payload.
func TestDigestPromptNamesTeammate(t *testing.T) {
	c := &Controller{}
	p := c.digestPrompt("worker-a")
	if !strings.Contains(p, "worker-a") {
		t.Fatalf("digest prompt must name the teammate: %q", p)
	}
	if !strings.Contains(p, "do not dispatch new background tasks") {
		t.Fatalf("digest instruction must forbid redispatch: %q", teamDigestInstruction)
	}
	if strings.Contains(p, "<team_digest>") == false {
		t.Fatalf("digest prompt must carry the block marker: %q", p)
	}
}
