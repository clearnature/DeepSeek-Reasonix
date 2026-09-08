package control

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/config"
	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

// teamTool is the leader's model-callable orchestration tool: it exposes team
// management (create member, assign work, roster status, remove, broadcast)
// to the model so an agent can act as its own team leader — the orchestration
// layer's primary caller, per the 2026-09-07 architecture report.
// ProviderVisible mirrors send_message: leader contexts carry a job manager,
// sub-agents and teammates have it cleared, so they never see this tool.
type teamTool struct {
	ts            *agent.TeammateStore
	workspaceRoot string
	// sched backs the loop_wakeup/cron_* actions (one instance per tool so the
	// persisted schedule file has a single writer).
	sched *wakeupScheduler
}

// NewTeamLeaderTool wraps a TeammateStore as the leader orchestration tool.
func NewTeamLeaderTool(ts *agent.TeammateStore, workspaceRoot string) tool.Tool {
	persist := ""
	if snap := ts.SnapshotPath(); snap != "" {
		persist = filepath.Join(filepath.Dir(snap), "wakeups.json")
	}
	sched := newWakeupScheduler(func(prompt string) error {
		return ts.PostMailToLeader("wakeup", prompt)
	}, persist)
	sched.onError = ts.EmitNotice
	return &teamTool{ts: ts, workspaceRoot: workspaceRoot, sched: sched}
}

func (t *teamTool) Name() string { return "team" }

func (t *teamTool) Description() string {
	return "Orchestrate your team of sub-agents (you are the leader). Actions: " +
		"group_create (name your team), group_delete (dissolve: stop and drop " +
		"every member), create (register a member), add (assign a task — it " +
		"forks your prefix as a background job), tasks (list the team's task " +
		"board), mail (send a message to one member's inbox, delivered on its " +
		"next assignment), status (list members), remove (drop a member), " +
		"shutdown (ask one member to wind down its work), approve (answer a " +
		"member's plan request: request_id + decision allow/deny), broadcast " +
		"(mail all active members), grant (restricted-writer write paths or " +
		"worktree mode for a member), revoke (drop a member's write token). " +
		"Running members are steered with the " +
		"send_message tool by job_id; mail reaches idle members."
}

func (t *teamTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "action":{"type":"string","enum":["group_create","group_delete","create","add","tasks","mail","status","shutdown","approve","remove","broadcast","grant","revoke","task_create","task_list","task_update","task_stop","enter_worktree","exit_worktree","loop_wakeup","cron_create","cron_list","cron_delete"]},
  "name":{"type":"string","description":"team name for group_create / member name for create/add/mail/remove/grant/revoke"},
  "role":{"type":"string","description":"role for create, default researcher"},
  "task":{"type":"string","description":"task prompt for add"},
  "text":{"type":"string","description":"message for broadcast"},
  "paths":{"type":"array","items":{"type":"string"},"description":"workspace-relative write paths for grant (restricted writer token)"},
  "worktree":{"type":"boolean","description":"grant worktree mode (dedicated branch) instead of explicit paths"},
  "prompt":{"type":"string","description":"task text for task_create / wake-up prompt for loop_wakeup and cron_create"},
  "task_id":{"type":"string","description":"board task id for task_update (from task_list)"},
  "owner":{"type":"string","description":"claiming teammate name for task_update"},
  "status":{"type":"string","description":"task_update status: done to complete, open to release"},
  "disposition":{"type":"string","description":"exit_worktree disposition: keep (default) or remove"},
  "confirm":{"type":"boolean","description":"required true when exit_worktree disposition=remove"},
  "delay_seconds":{"type":"integer","description":"one-shot delay for loop_wakeup"},
  "schedule_seconds":{"type":"integer","description":"repeat interval for cron_create"},
  "id":{"type":"string","description":"wake-up id for cron_delete (from cron_list)"},
  "request_id":{"type":"string","description":"approval request id for approve"},
  "decision":{"type":"string","enum":["allow","deny"],"description":"verdict for approve"}
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

func (t *teamTool) executeTasks() (string, error) {
	tasks := t.ts.Tasks()
	if len(tasks) == 0 {
		return "no tasks", nil
	}
	var b strings.Builder
	for _, tk := range tasks {
		fmt.Fprintf(&b, "- [%s] %s → %s\n", tk.Status, tk.Owner, truncateText(tk.Prompt, 80))
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (t *teamTool) executeApprove(requestID, decision string) (string, error) {
	requestID = strings.TrimSpace(requestID)
	decision = strings.TrimSpace(decision)
	if requestID == "" || (decision != "allow" && decision != "deny") {
		return "", fmt.Errorf("team approve: request_id and decision (allow|deny) are required")
	}
	if err := t.ts.Approve(requestID, decision == "allow", ""); err != nil {
		return "", err
	}
	return fmt.Sprintf("plan %q %s — verdict sent to teammate", requestID, decision), nil
}

func (t *teamTool) executeMail(name, text string) (string, error) {
	name = strings.TrimSpace(name)
	text = strings.TrimSpace(text)
	if name == "" || text == "" {
		return "", fmt.Errorf("team mail: name and text are required")
	}
	if err := t.ts.PostMail(name, text); err != nil {
		return "", err
	}
	return fmt.Sprintf("mail queued for %q — delivered on its next assignment", name), nil
}

func (t *teamTool) executeShutdown(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("team shutdown: name is required")
	}
	if err := t.ts.RequestShutdown(name); err != nil {
		return "", err
	}
	return fmt.Sprintf("shutdown requested for %q — it will wind down its current work", name), nil
}

// teamToolParams is the union of every action's arguments (single tool, many
// actions — qwen tool-surface convergence).
type teamToolParams struct {
	Action          string   `json:"action"`
	Name            string   `json:"name"`
	Role            string   `json:"role"`
	Task            string   `json:"task"`
	Text            string   `json:"text"`
	Paths           []string `json:"paths"`
	Worktree        bool     `json:"worktree"`
	RequestID       string   `json:"request_id"`
	Decision        string   `json:"decision"`
	Prompt          string   `json:"prompt"`
	TaskID          string   `json:"task_id"`
	Owner           string   `json:"owner"`
	Status          string   `json:"status"`
	Disposition     string   `json:"disposition"`
	Confirm         bool     `json:"confirm"`
	DelaySeconds    int      `json:"delay_seconds"`
	ScheduleSeconds int      `json:"schedule_seconds"`
	ID              string   `json:"id"`
}

func (t *teamTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p teamToolParams
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
	case "tasks":
		return t.executeTasks()
	case "mail":
		return t.executeMail(p.Name, p.Text)
	case "shutdown":
		return t.executeShutdown(p.Name)
	case "approve":
		return t.executeApprove(p.RequestID, p.Decision)
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
	case "grant":
		return t.executeGrant(p.Name, p.Worktree, p.Paths)
	case "revoke":
		return t.executeRevoke(p.Name)
	default:
		return t.executeExtended(ctx, p)
	}
}

// executeExtended routes the board/worktree/scheduler actions folded into the
// team tool (qwen tool-surface convergence), keeping Execute's own switch small.
func (t *teamTool) executeExtended(ctx context.Context, p teamToolParams) (string, error) {
	switch p.Action {
	case "task_create":
		id, err := t.ts.BoardCreate(p.Prompt)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("task %q published on the board — teammates claim it with team task_update", id), nil
	case "task_list":
		return t.executeBoardList()
	case "task_update":
		return t.executeBoardUpdate(p.TaskID, p.Owner, p.Status)
	case "task_stop":
		return t.executeTaskStop(p.Name)
	case "enter_worktree":
		path, branch, err := t.ts.EnterWorktree(ctx, p.Name, config.DeliveryWorktreeDir())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("teammate %q entered worktree %s on branch %s — its writes are bound there until exit_worktree", strings.TrimSpace(p.Name), path, branch), nil
	case "exit_worktree":
		remove := strings.TrimSpace(p.Disposition) == "remove"
		if remove && !p.Confirm {
			return "", fmt.Errorf("team exit_worktree: disposition=remove deletes the worktree and branch; pass confirm=true")
		}
		return t.ts.ExitWorktree(ctx, p.Name, remove)
	case "loop_wakeup":
		return t.executeLoopWakeup(p.DelaySeconds, p.Prompt)
	case "cron_create":
		return t.executeCronCreate(p.ScheduleSeconds, p.Prompt)
	case "cron_list":
		return t.executeCronList()
	case "cron_delete":
		return t.executeCronDelete(p.ID)
	default:
		return "", fmt.Errorf("team: unknown action %q", p.Action)
	}
}

func (t *teamTool) executeGrant(name string, worktree bool, paths []string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("team grant: name is required")
	}
	if worktree {
		if err := t.ts.GrantWorktree(name); err != nil {
			return "", err
		}
		return fmt.Sprintf("teammate %q granted worktree mode — dedicated branch on next assignment", name), nil
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("team grant: paths (or worktree) is required")
	}
	ws, err := agent.NormalizeWritePaths(t.workspaceRoot, paths)
	if err != nil {
		return "", err
	}
	if err := t.ts.Grant(name, ws); err != nil {
		return "", err
	}
	return fmt.Sprintf("teammate %q granted write token over %d path(s) — now a restricted writer", name, len(ws.Paths)), nil
}

func (t *teamTool) executeRevoke(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("team revoke: name is required")
	}
	if err := t.ts.Revoke(name); err != nil {
		return "", err
	}
	return fmt.Sprintf("teammate %q write token revoked — back to read-only", name), nil
}

func (t *teamTool) executeBoardList() (string, error) {
	items := t.ts.BoardList()
	if len(items) == 0 {
		return "task board is empty — publish work with team task_create", nil
	}
	var b strings.Builder
	for _, v := range items {
		fmt.Fprintf(&b, "- %s [%s] owner=%s: %s\n", v.ID, v.Status, v.Owner, v.Prompt)
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

func (t *teamTool) executeBoardUpdate(taskID, owner, status string) (string, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "", fmt.Errorf("team task_update: task_id is required")
	}
	switch strings.TrimSpace(status) {
	case agent.BoardDone:
		if err := t.ts.BoardUpdate(taskID, owner, agent.BoardDone); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %q marked done", taskID), nil
	case agent.BoardOpen:
		if err := t.ts.BoardUpdate(taskID, owner, agent.BoardOpen); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %q released to open", taskID), nil
	default:
		if err := t.ts.BoardClaim(taskID, owner); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %q claimed by %q", taskID, owner), nil
	}
}

func (t *teamTool) executeTaskStop(name string) (string, error) {
	name = strings.TrimSpace(name)
	if err := t.ts.TeamStop(name); err != nil {
		return "", err
	}
	released := t.ts.ReleaseBoardTasks(name)
	return fmt.Sprintf("stopped %q (job killed, member idle); %d claimed board task(s) released", name, released), nil
}

func (t *teamTool) executeLoopWakeup(delaySeconds int, prompt string) (string, error) {
	if delaySeconds <= 0 || strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("team loop_wakeup: delay_seconds and prompt are required")
	}
	e := &wakeupEntry{ID: fmt.Sprintf("wake-%d", time.Now().UnixNano()), Prompt: prompt,
		Delay: (time.Duration(delaySeconds) * time.Second).String(), Created: time.Now()}
	if err := t.sched.add(e); err != nil {
		return "", err
	}
	return fmt.Sprintf("wake-up %s armed for %ds", e.ID, delaySeconds), nil
}

func (t *teamTool) executeCronCreate(scheduleSeconds int, prompt string) (string, error) {
	if scheduleSeconds <= 0 || strings.TrimSpace(prompt) == "" {
		return "", fmt.Errorf("team cron_create: schedule_seconds and prompt are required")
	}
	e := &wakeupEntry{ID: fmt.Sprintf("cron-%d", time.Now().UnixNano()), Prompt: prompt,
		Schedule: (time.Duration(scheduleSeconds) * time.Second).String(), Created: time.Now()}
	if err := t.sched.add(e); err != nil {
		return "", err
	}
	return fmt.Sprintf("cron %s created (every %ds)", e.ID, scheduleSeconds), nil
}

func (t *teamTool) executeCronList() (string, error) {
	items := t.sched.list()
	if len(items) == 0 {
		return "no scheduled wake-ups", nil
	}
	var b strings.Builder
	for _, v := range items {
		kind, every := "wake", v.Delay
		if v.Schedule != "" {
			kind, every = "cron", v.Schedule
		}
		fmt.Fprintf(&b, "- %s [%s every %s]: %s\n", v.ID, kind, every, v.Prompt)
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

func (t *teamTool) executeCronDelete(id string) (string, error) {
	id = strings.TrimSpace(id)
	if !t.sched.remove(id) {
		return "", fmt.Errorf("team cron_delete: unknown id %q", id)
	}
	return fmt.Sprintf("wake-up %s cancelled", id), nil
}
