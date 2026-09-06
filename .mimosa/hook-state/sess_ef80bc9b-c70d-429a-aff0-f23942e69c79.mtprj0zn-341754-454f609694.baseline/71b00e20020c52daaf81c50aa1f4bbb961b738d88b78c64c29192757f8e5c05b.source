package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestForkReadOnlyGatePolicy: the P5 fork execution gate allows read-only tools
// and permission-classified read-only bash commands, and rejects every writer
// (bash included when the concrete command is not read-only).
func TestForkReadOnlyGatePolicy(t *testing.T) {
	g := forkReadOnlyGate{}
	cases := []struct {
		name      string
		tool      string
		args      json.RawMessage
		readOnly  bool
		wantAllow bool
	}{
		{name: "read_only_tool", tool: "read_file", args: json.RawMessage(`{"path":"/a"}`), readOnly: true, wantAllow: true},
		{name: "bash_readonly_git_status", tool: "bash", args: json.RawMessage(`{"command":"git status --short"}`), readOnly: false, wantAllow: true},
		{name: "bash_readonly_grep", tool: "bash", args: json.RawMessage(`{"command":"grep -n Fork x"}`), readOnly: false, wantAllow: true},
		{name: "bash_writer_rm", tool: "bash", args: json.RawMessage(`{"command":"rm -rf /tmp/x"}`), readOnly: false, wantAllow: false},
		{name: "writer_tool", tool: "write_file", args: json.RawMessage(`{"path":"/a"}`), readOnly: false, wantAllow: false},
		{name: "writer_bash_run_in_background", tool: "bash", args: json.RawMessage(`{"command":"ls","run_in_background":true}`), readOnly: false, wantAllow: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			allow, reason, err := g.Check(context.Background(), tc.tool, tc.args, tc.readOnly)
			if err != nil {
				t.Fatalf("Check returned error: %v", err)
			}
			if allow != tc.wantAllow {
				t.Fatalf("allow = %v (reason %q), want %v", allow, reason, tc.wantAllow)
			}
			if !allow && !strings.Contains(reason, "read-only") {
				t.Fatalf("deny reason should explain the read-only boundary, got %q", reason)
			}
		})
	}
}

// TestForkReadOnlyGateExecutionBlocksWriterAndAllowsReadOnly proves the gate is
// consulted at execute time (executeOne): the writer is blocked with a
// "blocked:" result, while a read-only tool and a read-only bash command run.
func TestForkReadOnlyGateExecutionBlocksWriterAndAllowsReadOnly(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(fakeTool{name: "bash", readOnly: false})
	reg.Add(fakeTool{name: "read_file", readOnly: true})
	reg.Add(fakeTool{name: "write_file", readOnly: false})

	a := New(nil, reg, NewSession(""), Options{Gate: forkReadOnlyGate{}}, event.Discard)
	turn := &turnRuntime{}

	blocked := a.executeOne(context.Background(), turn, provider.ToolCall{Name: "write_file", Arguments: `{"path":"/a"}`})
	if !strings.HasPrefix(blocked.output, "blocked:") || !blocked.blocked {
		t.Errorf("writer call = %q/%+v, want a blocked result", blocked.output, blocked)
	}

	blockedBash := a.executeOne(context.Background(), turn, provider.ToolCall{Name: "bash", Arguments: `{"command":"rm -rf /tmp/x"}`})
	if !strings.HasPrefix(blockedBash.output, "blocked:") || !blockedBash.blocked {
		t.Errorf("writer bash call = %q/%+v, want a blocked result", blockedBash.output, blockedBash)
	}

	okBash := a.executeOne(context.Background(), turn, provider.ToolCall{Name: "bash", Arguments: `{"command":"git status --short"}`})
	if strings.HasPrefix(okBash.output, "blocked:") {
		t.Errorf("read-only bash call should run, got %q", okBash.output)
	}

	okRead := a.executeOne(context.Background(), turn, provider.ToolCall{Name: "read_file", Arguments: `{"path":"/a"}`})
	if strings.HasPrefix(okRead.output, "blocked:") {
		t.Errorf("read-only tool should run, got %q", okRead.output)
	}
}

// TestForkReadOnlyGateContext verifies the WithForkReadOnlyGate /
// forkReadOnlyGateFromContext round-trip used to ride the gate to the child.
func TestForkReadOnlyGateContext(t *testing.T) {
	if _, ok := forkReadOnlyGateFromContext(context.Background()); ok {
		t.Fatalf("plain ctx unexpectedly carries a fork gate")
	}
	ctx := WithForkReadOnlyGate(context.Background(), forkReadOnlyGate{})
	g, ok := forkReadOnlyGateFromContext(ctx)
	if !ok || g == nil {
		t.Fatalf("forkReadOnlyGateFromContext = (%v, %v), want non-nil gate", g, ok)
	}
}

// TestForkDepthGuard: fork-of-fork is allowed only while depth+1 stays within
// max_subagent_depth (plan §二 裁决).
func TestForkDepthGuard(t *testing.T) {
	cases := []struct {
		name      string
		depth     int
		maxDepth  int
		wantError bool
	}{
		{name: "root_fork_at_depth0", depth: 0, maxDepth: 2},
		{name: "fork_at_depth1_of2", depth: 1, maxDepth: 2},
		{name: "fork_at_limit", depth: 2, maxDepth: 2, wantError: true},
		{name: "fork_beyond_limit", depth: 3, maxDepth: 2, wantError: true},
		{name: "unset_maxdepth_normalizes_to1_root_ok", depth: 0, maxDepth: 0},
		{name: "unset_maxdepth_normalizes_to1_depth1_rejected", depth: 1, maxDepth: 0, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkForkDepthGuard(WithSubagentDepth(context.Background(), tc.depth), tc.maxDepth)
			if tc.wantError && err == nil {
				t.Fatalf("depth guard allowed depth %d at max %d", tc.depth, tc.maxDepth)
			}
			if !tc.wantError && err != nil {
				t.Fatalf("depth guard rejected depth %d at max %d: %v", tc.depth, tc.maxDepth, err)
			}
		})
	}
}

// TestForkSizeGuard: the inherited prefix may not exceed 80% of the child
// context window (estimated tokens); a disabled window enforces nothing.
func TestForkSizeGuard(t *testing.T) {
	// estimateMessagesTokens counts ASCII text by rune (1 rune ≈ 1 token) plus
	// 4 per-message framing, so content length is the dominant term.
	big := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("a", 1024)}} // ~1028 tokens
	small := []provider.Message{{Role: provider.RoleUser, Content: "hi"}}

	if err := checkForkSizeGuard(1000, big); err == nil {
		t.Fatalf("size guard allowed a %d-token prefix in a %d-token window (80%% = 800)", estimateMessagesTokens(big), 1000)
	}
	if err := checkForkSizeGuard(1000, small); err != nil {
		t.Fatalf("size guard rejected a small prefix: %v", err)
	}
	// Exactly at the 80% boundary is allowed (guard triggers only on exceed).
	atLimit := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("a", 796)}} // 796+4 = 800
	if err := checkForkSizeGuard(1000, atLimit); err != nil {
		t.Fatalf("size guard rejected an at-limit prefix: %v", err)
	}
	overLimit := []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("a", 797)}} // 797+4 = 801
	if err := checkForkSizeGuard(1000, overLimit); err == nil {
		t.Fatalf("size guard allowed a %d-token prefix over the 80%% budget", estimateMessagesTokens(overLimit))
	}
	// No window configured: no ceiling.
	if err := checkForkSizeGuard(0, big); err != nil {
		t.Fatalf("size guard with no window rejected: %v", err)
	}
}
