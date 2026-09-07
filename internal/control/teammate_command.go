package control

import (
	"context"
	"fmt"
	"strings"

	"reasonix/internal/agent"
	"reasonix/internal/jobs"
)

var ctxTODO = context.Background

// applyTeamCommand implements the P6 /team-* management verbs: /team-create
// registers a teammate; /team-add dispatches one job (first run forks the
// leader prefix, later runs continue the teammate's own transcript);
// /team-status lists the roster; /team-remove kills and drops a member. All
// are host commands: output rides Notices, never the provider surface. It
// lives in its own upstream-absent file so a controller.go convergence cannot
// silently drop the implementation (v1.38 did exactly that — the options
// field survived, the dispatch was gone).
func (c *Controller) applyTeamCommand(cmd, trimmed string) {
	if c.teammates == nil {
		c.notice("team commands are disabled (no TeammateStore configured)")
		return
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, cmd))
	switch cmd {
	case "/team-create":
		name, rest2, _ := strings.Cut(rest, " ")
		role, rest3, _ := strings.Cut(strings.TrimSpace(rest2), " ")
		writable := strings.TrimSpace(rest3) == "writable"
		if err := c.teammates.Create(name, role, writable); err != nil {
			c.notice("team-create: " + err.Error())
			return
		}
		c.notice(fmt.Sprintf("teammate %q created (role %q) — assign work with /team-add", name, role))
	case "/team-grant":
		fields := strings.Fields(rest)
		if len(fields) < 2 {
			c.notice("usage: /team-grant <name> <path...> | <name> worktree")
			return
		}
		if fields[1] == "worktree" {
			if err := c.teammates.GrantWorktree(fields[0]); err != nil {
				c.notice("team-grant: " + err.Error())
				return
			}
			c.notice(fmt.Sprintf("teammate %q granted worktree mode — dedicated branch on next assignment", fields[0]))
			return
		}
		ws, err := agent.NormalizeWritePaths(c.workspaceRoot, fields[1:])
		if err != nil {
			c.notice("team-grant: " + err.Error())
			return
		}
		if err := c.teammates.Grant(fields[0], ws); err != nil {
			c.notice("team-grant: " + err.Error())
			return
		}
		c.notice(fmt.Sprintf("teammate %q granted write token over %d path(s) — now a restricted writer", fields[0], len(ws.Paths)))
	case "/team-revoke":
		name := strings.TrimSpace(rest)
		if err := c.teammates.Revoke(name); err != nil {
			c.notice("team-revoke: " + err.Error())
			return
		}
		c.notice(fmt.Sprintf("teammate %q write token revoked — back to read-only", name))
	case "/team-remove":
		name := strings.TrimSpace(rest)
		if err := c.teammates.Remove(name); err != nil {
			c.notice("team-remove: " + err.Error())
			return
		}
		c.notice(fmt.Sprintf("teammate %q removed", name))
	case "/team-stop":
		name := strings.TrimSpace(rest)
		if err := c.teammates.TeamStop(name); err != nil {
			c.notice("team-stop: " + err.Error())
			return
		}
		c.notice(fmt.Sprintf("teammate %q stopping", name))
	case "/team-status":
		c.notice(c.teamRosterText())
	case "/team-group":
		c.applyTeamGroup(rest)
		return
	case "/team-add":
		name, task, _ := strings.Cut(rest, " ")
		if strings.TrimSpace(task) == "" {
			c.notice("usage: /team-add <name> <task>")
			return
		}
		if _, err := c.teammates.Assign(jobs.WithSession(ctxTODO(), c.parentSessionID()), name, task); err != nil {
			c.notice("team-add: " + err.Error())
			return
		}
		c.notice(fmt.Sprintf("job dispatched to %q — /team-status to watch it finish", name))
	case "/team-broadcast":
		text := strings.TrimSpace(rest)
		if text == "" {
			c.notice("usage: /team-broadcast <message>")
			return
		}
		for _, tm := range c.teammates.List() {
			if tm.State == agent.TeammateIdle {
				continue
			}
			if err := c.teammates.PostMail(tm.Name, text); err != nil {
				c.notice("team-broadcast: " + err.Error())
				return
			}
		}
		c.notice("broadcast mailed to active teammates")
	case "/team-ask":
		c.notice("/team-ask: ask-forwarding is managed automatically for running members")
	case "/team-approve":
		c.applyTeamApprove(rest)
	case "/team-spawn":
		c.notice("/team-spawn is not wired in this build — create members individually with /team-create")
	default:
		c.notice("unknown team command: " + cmd)
	}
}

// applyTeamGroup implements /team-group (qwen team_create/team_delete
// analog): create names the singleton group, delete dissolves it.
func (c *Controller) applyTeamGroup(rest string) {
	sub := strings.TrimSpace(rest)
	name, _, _ := strings.Cut(sub, " ")
	switch {
	case strings.HasPrefix(sub, "create"):
		if err := c.teammates.CreateGroup(name); err != nil {
			c.notice("team-group create: " + err.Error())
			return
		}
		c.notice(fmt.Sprintf("team %q created — register members with /team-create or /team-add", name))
	case strings.HasPrefix(sub, "delete"), strings.HasPrefix(sub, "remove"):
		if err := c.teammates.DeleteGroup(); err != nil {
			c.notice("team-group delete: " + err.Error())
			return
		}
		c.notice("team dissolved — all members stopped and removed")
	case sub == "" || sub == "status":
		c.notice("team group: " + c.teamGroupText())
	default:
		c.notice("usage: /team-group create <name> | delete | status")
	}
}

// applyTeamApprove answers a teammate plan-approval request (qwen
// team_plan_approval analog).
func (c *Controller) applyTeamApprove(rest string) {
	fields := strings.Fields(strings.TrimSpace(rest))
	if len(fields) != 2 || (fields[1] != "allow" && fields[1] != "deny") {
		c.notice("usage: /team-approve <request_id> allow|deny")
		return
	}
	if err := c.teammates.Approve(fields[0], fields[1] == "allow", c.parentSessionID()); err != nil {
		c.notice("team-approve: " + err.Error())
		return
	}
	c.notice(fmt.Sprintf("plan %q %s — verdict sent to teammate", fields[0], fields[1]))
}

// teamRosterText renders the P6 roster for /team-status notices.
func (c *Controller) teamRosterText() string {
	if c.teammates == nil {
		return "team commands are disabled (no TeammateStore configured)"
	}
	roster := c.teammates.Roster()
	if len(roster) == 0 {
		return "no teammates — /team-create <name> to start"
	}
	var b strings.Builder
	for _, rv := range roster {
		fmt.Fprintf(&b, "%s %s\n", rv.Name, rv.State)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// teamGroupText renders the group label for /team-group status.
func (c *Controller) teamGroupText() string {
	if c.teammates == nil {
		return "team commands are disabled (no TeammateStore configured)"
	}
	name := c.teammates.Name()
	if name == "" {
		return "no named team — /team-group create <name>"
	}
	n := len(c.teammates.Roster())
	return fmt.Sprintf("%s (%d member(s))", name, n)
}

// TeamRosterView exposes the live roster for headless harnesses and the
// desktop team panel.
func (c *Controller) TeamRosterView() []agent.RosterView {
	if c.teammates == nil {
		return nil
	}
	return c.teammates.Roster()
}

// JobSnapshots returns the background-job state for the desktop job panel.
func (c *Controller) JobSnapshots() []jobs.JobSnapshot {
	if c.jobs == nil {
		return nil
	}
	return c.jobs.JobSnapshotsForSession(c.parentSessionID())
}

// SendTaskMessage routes a human message to a background job's worker via the
// job mailbox (a running teammate consumes it on its next loop turn).
func (c *Controller) SendTaskMessage(jobID, text string) error {
	if c.jobs == nil {
		return fmt.Errorf("no job manager")
	}
	return c.jobs.SendMessageForSession(c.parentSessionID(), jobID, text)
}
