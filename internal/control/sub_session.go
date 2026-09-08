package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/internal/tool"
)

// create_sub_session: qwen's daemon-only session spawner on the model surface.
// Delegates to a SubSessionSpawner from the serving frontend; without one it
// reports the daemon-only message, exactly like qwen with no session bridge.

// SubSessionSpawner spawns an independent session and delivers a prompt.
// completion "sent" returns once delivered; "first-turn" waits for the
// sub-session's first turn and returns its output.
type SubSessionSpawner interface {
	SpawnSubSession(ctx context.Context, prompt, completion string) (string, error)
}

const subSessionDaemonOnly = "create_sub_session is only available when running under `reasonix serve` (daemon mode); there is no session bridge in this environment, so a sub-session cannot be spawned"

type createSubSessionTool struct{ spawner SubSessionSpawner }

// NewCreateSubSessionTool wraps a spawner as the create_sub_session tool.
func NewCreateSubSessionTool(spawner SubSessionSpawner) tool.Tool {
	return &createSubSessionTool{spawner: spawner}
}

func (t *createSubSessionTool) Name() string { return "create_sub_session" }
func (t *createSubSessionTool) Description() string {
	return "Spawn an independent sub-session and deliver a prompt to it. completion=\"sent\" returns as soon as the prompt is delivered; completion=\"first-turn\" (default) waits for the sub-session's first turn and returns its output. Use to fan work out into a separate session that keeps its own transcript. Only available under `reasonix serve`. qwen create_sub_session analog."
}
func (t *createSubSessionTool) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"prompt":{"description":"The prompt to deliver to the new sub-session.","type":"string"},"completion":{"description":"sent (return on delivery) or first-turn (default: wait for the first turn and return its output).","type":"string"}},"required":["prompt"],"type":"object"}`)
}
func (t *createSubSessionTool) ReadOnly() bool { return false }
func (t *createSubSessionTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if t.spawner == nil {
		return "", fmt.Errorf("%s", subSessionDaemonOnly)
	}
	var p struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("create_sub_session: %w", err)
	}
	if strings.TrimSpace(p.Prompt) == "" {
		return "", fmt.Errorf("create_sub_session: prompt is required")
	}
	mode := strings.TrimSpace(p.Completion)
	if mode == "" {
		mode = "first-turn"
	}
	if mode != "sent" && mode != "first-turn" {
		return "", fmt.Errorf("create_sub_session: unknown completion %q (sent|first-turn)", mode)
	}
	return t.spawner.SpawnSubSession(ctx, p.Prompt, mode)
}

// SubSessionSpawnerFunc adapts a function to SubSessionSpawner so a frontend
// can wire a lazily-created server without a struct.
type SubSessionSpawnerFunc func(ctx context.Context, prompt, completion string) (string, error)

// SpawnSubSession implements SubSessionSpawner.
func (f SubSessionSpawnerFunc) SpawnSubSession(ctx context.Context, prompt, completion string) (string, error) {
	return f(ctx, prompt, completion)
}
