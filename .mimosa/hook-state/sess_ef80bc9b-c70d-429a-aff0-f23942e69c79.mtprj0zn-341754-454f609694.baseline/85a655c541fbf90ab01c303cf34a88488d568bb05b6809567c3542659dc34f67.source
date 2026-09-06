package boot

import (
	"context"

	"reasonix/internal/agent"
	"reasonix/internal/skill"
)

// skillForkSession builds the skill sub-agent session: the parent's fork
// prefix (system + committed history, byte-identical → warm cache) with the
// skill body appended to the system tail, or a cold start when inheritance is
// unavailable. runAs=subagent skills used to cold-start every invocation,
// paying full price for each child turn (team 三职能 each round).
func skillForkSession(sctx context.Context, sysPrompt string) *agent.Session {
	if sess := agent.PrepareSkillForkSession(agent.ContextParentAgent(sctx), sysPrompt); sess != nil {
		return sess
	}
	return agent.NewSession(sysPrompt)
}

// ephemeralSkillRun mirrors skillForkSession for headless runs that have no
// persisted transcript owner.
func ephemeralSkillRun(sctx context.Context, body string) *agent.SubagentRun {
	run := agent.EphemeralSubagentRun(body)
	if forkPrefix := agent.CaptureSkillForkPrefix(agent.ContextParentAgent(sctx), body); len(forkPrefix) > 0 {
		run = agent.EphemeralSubagentRunWithPrefix(forkPrefix)
	}
	return run
}

// prepareSkillRun prepares a persisted skill sub-agent run (writer path) with
// fork inheritance when available, falling back to a fresh cold start.
func prepareSkillRun(sctx context.Context, sk skill.Skill, spec agent.SubagentSpec, store *agent.SubagentStore) (*agent.SubagentRun, error) {
	if forkPrefix := agent.CaptureSkillForkPrefix(agent.ContextParentAgent(sctx), sk.Body); len(forkPrefix) > 0 {
		forkSpec := spec
		forkSpec.SystemPrompt = ""
		run, err := store.PrepareParentFork(forkPrefix, forkSpec)
		if err != nil {
			return nil, err
		}
		run.Session.MarkForkPrefill()
		return run, nil
	}
	return store.PrepareFresh(spec)
}
