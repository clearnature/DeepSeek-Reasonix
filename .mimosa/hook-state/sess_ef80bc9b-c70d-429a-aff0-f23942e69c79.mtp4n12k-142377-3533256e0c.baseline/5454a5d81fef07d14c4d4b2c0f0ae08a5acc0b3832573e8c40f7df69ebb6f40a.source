package agent

import (
	"context"
	"encoding/json"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// TestTeamAskGateBlocksApprovalTools verifies the P11 AskGate: a tool the
// leader marked for approval is intercepted (request submitted, not
// executed), while unmarked tools pass straight through.
func TestTeamAskGateBlocksApprovalTools(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	dir := t.TempDir()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm, dir)
	defer ts.Close()

	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.SetAskTools("alpha", []string{"write_file"}); err != nil {
		t.Fatalf("SetAskTools: %v", err)
	}

	if !ts.AskRequiredForTool("alpha", "write_file") {
		t.Fatal("write_file should require approval for alpha")
	}
	if ts.AskRequiredForTool("alpha", "read_file") {
		t.Fatal("read_file should not require approval")
	}

	// AskForTool submits a "tool" approval request.
	if err := ts.AskForTool("alpha", "write_file", `{"path":"/tmp/x"}`); err != nil {
		t.Fatalf("AskForTool: %v", err)
	}
	pending := ts.PendingApprovals()
	if len(pending) != 1 || pending[0].Kind != "tool" {
		t.Fatalf("pending after tool ask = %+v, want one kind=tool", pending)
	}
	if len(pending[0].Plan) < len("tool write_file ") {
		t.Fatalf("tool ask body should name the tool; got %q", pending[0].Plan)
	}
}

// TestAskGatePassthroughWithoutMailbox verifies the gate is inert outside
// teammate contexts: a plain (non-team) call executes normally.
func TestAskGatePassthroughWithoutMailbox(t *testing.T) {
	// The askGate wrapper is exercised through buildSubReg; with no mailbox
	// in ctx it must forward to the inner tool. We assert the wrapper's
	// existence and passthrough via a direct unit check:
	g := askGate{inner: askEchoTool{}}
	out, err := g.Execute(context.Background(), json.RawMessage(`{"text":"hi"}`))
	if err != nil {
		t.Fatalf("passthrough execute: %v", err)
	}
	if out != "echo:hi" {
		t.Fatalf("passthrough output = %q, want echo:hi", out)
	}
}

// askEchoTool is a minimal Tool for the passthrough test.
type askEchoTool struct{}

func (askEchoTool) Name() string            { return "echo" }
func (askEchoTool) Description() string     { return "echo" }
func (askEchoTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (askEchoTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(args, &in)
	return "echo:" + in.Text, nil
}
func (askEchoTool) ReadOnly() bool { return true }
