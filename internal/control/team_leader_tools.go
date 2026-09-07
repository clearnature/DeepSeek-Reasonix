package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/internal/agent"
	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

// teamTool is the leader's model-callable orchestration tool: it exposes team
// management (create member, assign work, roster status, remove, broadcast)
// to the model so an agent can act as its own team leader — the orchestration
// layer's primary caller, per the 2026-09-07 architecture report. Host
// /team-* commands remain the human fallback over the same TeammateStore.
// ProviderVisible mirrors send_message: leader contexts carry a job manager,
// sub-agents and teammates have it cleared, so they never see this tool.
type teamTool struct {
	ts *agent.TeammateStore
}

// NewTeamLeaderTool wraps a TeammateStore as the leader orchestration tool.
func NewTeamLeaderTool(ts *agent.TeammateStore) tool.Tool {
	return &teamTool{ts: ts}
}

func (t *teamTool) Name() string { return "team" }

func (t *teamTool) Description() string {
	return "Orchestrate your team of sub-agents (you are the leader). Actions: " +
		"group_create (name your team — one active group), group_delete " +
		"(dissolve the team: stop and drop every member), create (register a " +
		"member), add (assign one task to a member — it forks your prefix and " +
		"runs as a background job), status (list members), remove (drop a " +
		"member), broadcast (mail all active members). Members run " +
		"concurrently; a member returns to idle when its job completes."
}

func (t *teamTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "action":{"type":"string","enum":["group_create","group_delete","create","add","status","remove","broadcast"]},
  "name":{"type":"string","description":"team name for group_create / member name for create/add/remove"},
  "role":{"type":"string","description":"role for create, default researcher"},
  "task":{"type":"string","description":"task prompt for add"},
  "text":{"type":"string","description":"message for broadcast"}
},
"required":["action"]
}`)
}

func (t *teamTool) ReadOnly() bool { return false }

// ProviderVisible hides the tool from sub-agents and teammates: leader
// contexts carry a jobs manager (jobs.FromContext), child contexts clear it.
func (t *teamTool) ProviderVisible(ctx context.Context) bool {
	if t.ts == nil {
		return false
	}
	_, ok := jobs.FromContext(ctx)
	return ok
}

func (t *teamTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action string `json:"action"`
		Name   string `json:"name"`
		Role   string `json:"role"`
		Task   string `json:"task"`
		Text   string `json:"text"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("team: invalid args: %w", err)
	}
	switch p.Action {
	case "group_create":
		if err := t.ts.CreateGroup(p.Name); err != nil {
			return "", err
		}
		return fmt.Sprintf("team %q created — register members with team create or team add", p.Name), nil
	case "group_delete":
		if err := t.ts.DeleteGroup(); err != nil {
			return "", err
		}
		return "team dissolved — all members stopped and removed", nil
	case "create":
		if strings.TrimSpace(p.Name) == "" {
			return "", fmt.Errorf("team create: name is required")
		}
		if err := t.ts.Create(strings.TrimSpace(p.Name), p.Role); err != nil {
			return "", err
		}
		return fmt.Sprintf("teammate %q created (role %q) — assign work with team add", strings.TrimSpace(p.Name), p.Role), nil
	case "add":
		name := strings.TrimSpace(p.Name)
		task := strings.TrimSpace(p.Task)
		if name == "" || task == "" {
			return "", fmt.Errorf("team add: name and task are required")
		}
		if _, err := t.ts.Assign(ctx, name, task); err != nil {
			return "", err
		}
		return fmt.Sprintf("job dispatched to %q — team status to watch it finish", name), nil
	case "status":
		var b strings.Builder
		roster := t.ts.Roster()
		if len(roster) == 0 {
			return "no teammates — team create <name> to start", nil
		}
		for _, rv := range roster {
			fmt.Fprintf(&b, "- %s: %s", rv.Name, rv.State)
			if rv.Role != "" {
				fmt.Fprintf(&b, " (%s)", rv.Role)
			}
			b.WriteString("\n")
		}
		return strings.TrimSuffix(b.String(), "\n"), nil
	case "remove":
		name := strings.TrimSpace(p.Name)
		if name == "" {
			return "", fmt.Errorf("team remove: name is required")
		}
		if err := t.ts.Remove(name); err != nil {
			return "", err
		}
		return fmt.Sprintf("teammate %q removed", name), nil
	case "broadcast":
		text := strings.TrimSpace(p.Text)
		if text == "" {
			return "", fmt.Errorf("team broadcast: text is required")
		}
		sent := 0
		for _, tm := range t.ts.List() {
			if tm.State == agent.TeammateIdle {
				continue
			}
			if err := t.ts.PostMail(tm.Name, text); err != nil {
				return "", err
			}
			sent++
		}
		return fmt.Sprintf("broadcast mailed to %d active teammate(s)", sent), nil
	default:
		return "", fmt.Errorf("team: unknown action %q (create|add|status|remove|broadcast)", p.Action)
	}
}
