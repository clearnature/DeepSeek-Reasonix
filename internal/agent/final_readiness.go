package agent

import (
	"fmt"
	"strings"

	"reasonix/internal/ablation"
	"reasonix/internal/evidence"
	"reasonix/internal/instruction"
	"reasonix/internal/runtimepolicy"
	"reasonix/internal/taskcontract"
)

// Final readiness: whether the fact contract and ledger allow the turn to stop.

type finalReadinessCheck struct {
	applies                    bool
	reason                     string
	continuationGeneric        bool
	continuationHighConfidence bool
	continuationUnsafe         bool
	missingProjectChecks       int
	incompleteTodos            int
	missingAcceptanceCriteria  int
	missingVerification        int
	missingReview              int
	missingSignoff             int
	missingActionEvidence      int
	missingMutation            int
	missingCapabilities        int
}

func (c finalReadinessCheck) continuationClass() ReadinessContinuationClass {
	if c.reason == "" || c.continuationUnsafe || c.missingActionEvidence > 0 ||
		c.missingMutation > 0 || c.missingCapabilities > 0 {
		return ReadinessContinuationNone
	}
	if c.continuationHighConfidence {
		return ReadinessContinuationHighConfidence
	}
	if c.continuationGeneric {
		return ReadinessContinuationGeneric
	}
	return ReadinessContinuationNone
}

func (c *finalReadinessCheck) observeObligation(o taskcontract.Obligation) {
	if o.Enforcement != taskcontract.EnforcementAdvisory {
		switch o.Kind {
		case taskcontract.ObligationActionReceipt:
			// Repeating an external/destructive action to manufacture a
			// receipt is never an automatic readiness operation.
			c.continuationUnsafe = true
		case taskcontract.ObligationTodo, taskcontract.ObligationCriteria,
			taskcontract.ObligationFullVerify, taskcontract.ObligationIndependentReview,
			taskcontract.ObligationSecurityReview, taskcontract.ObligationSignoff:
			switch {
			case o.Kind == taskcontract.ObligationTodo || o.Kind == taskcontract.ObligationCriteria:
				c.continuationHighConfidence = true
			case o.Enforcement == taskcontract.EnforcementStrict:
				c.continuationHighConfidence = true
			case o.Enforcement == taskcontract.EnforcementRecoverable:
				c.continuationGeneric = true
			}
		case taskcontract.ObligationTargetedVerify, taskcontract.ObligationDiffReview:
			switch o.Enforcement {
			case taskcontract.EnforcementStrict:
				c.continuationHighConfidence = true
			case taskcontract.EnforcementRecoverable:
				c.continuationGeneric = true
			}
		default:
			c.continuationUnsafe = true
		}
	}
	switch o.Kind {
	case taskcontract.ObligationTargetedVerify, taskcontract.ObligationFullVerify:
		c.missingVerification++
	case taskcontract.ObligationDiffReview, taskcontract.ObligationIndependentReview, taskcontract.ObligationSecurityReview:
		c.missingReview++
	case taskcontract.ObligationSignoff:
		c.missingSignoff++
	case taskcontract.ObligationActionReceipt:
		c.missingActionEvidence++
	case taskcontract.ObligationCriteria:
		c.missingAcceptanceCriteria++
	}
}

func (c finalReadinessCheck) progressSignature() string {
	return fmt.Sprintf("%d/%d/%d/%d/%d/%d/%d/%d/%d/%d\x00%s",
		c.missingProjectChecks,
		c.incompleteTodos,
		c.missingAcceptanceCriteria,
		c.missingVerification,
		c.missingReview,
		c.missingSignoff,
		c.missingActionEvidence,
		c.missingMutation,
		c.missingCapabilities,
		boolInt(c.applies),
		c.reason,
	)
}

func (c finalReadinessCheck) missingIDs() []string {
	missing := make([]string, 0, 9)
	add := func(id string, count int) {
		if count > 0 {
			missing = append(missing, id)
		}
	}
	add("project_check", c.missingProjectChecks)
	add("todo", c.incompleteTodos)
	add("criteria", c.missingAcceptanceCriteria)
	add("verification", c.missingVerification)
	add("review", c.missingReview)
	add("signoff", c.missingSignoff)
	add("action", c.missingActionEvidence)
	add("mutation", c.missingMutation)
	add("capability", c.missingCapabilities)
	return missing
}

func (c finalReadinessCheck) audit(result evidence.ReadinessAuditResult, recovered bool) evidence.ReadinessAudit {
	return evidence.ReadinessAudit{
		Result:                    result,
		Recovered:                 recovered,
		MissingProjectChecks:      c.missingProjectChecks,
		IncompleteTodos:           c.incompleteTodos,
		CommandMismatchMissing:    c.missingProjectChecks,
		MissingAcceptanceCriteria: c.missingAcceptanceCriteria,
		MissingVerification:       c.missingVerification,
		MissingReview:             c.missingReview,
		MissingSignoff:            c.missingSignoff,
		MissingActionEvidence:     c.missingActionEvidence,
		MissingMutation:           c.missingMutation,
		MissingCapabilities:       c.missingCapabilities,
	}
}

func (a *Agent) finalReadinessCheckFor() finalReadinessCheck {
	if a.task.ledger == nil || a.ablation.Off(ablation.Evidence) {
		return finalReadinessCheck{}
	}
	// Absorb wait/child receipts that arrived after the last commit. Do not
	// rebuild the whole contract here — that would re-impose an unused plan.
	if a.turn.engine != nil && a.task.ledger != nil {
		a.turn.engine.SyncReceipts(a.task.ledger.Receipts(), a.writeWorkspaceRoot, a.turn.constraints.ForbidTests)
	}
	var missing []string
	out := finalReadinessCheck{continuationUnsafe: a.turn.constraints.ForbidTests}
	if a.planMode.Load() {
		return out
	}
<<<<<<< HEAD
	a.escalatePolicyFromEvidence()
	closedLoop, qualityGated := a.finalReadinessPolicyState()
	// An unfinished todo is a direct contradiction of a final answer, not a
	// quality-evidence gap. Keep blocking it even on ordinary targeted turns.
=======
>>>>>>> origin/main-v2
	incomplete, hasTodos := a.task.ledger.IncompleteLatestTodos()
	if !hasTodos && a.task.ledger.HasAnySuccessfulReceipt() {
		incomplete, hasTodos = a.incompleteCanonicalTodos()
	}
<<<<<<< HEAD
	if hasTodos && len(incomplete) > 0 && a.task.ledger.HasSuccessfulTodoProgressReceipt() {
		out.applies = true
		out.incompleteTodos = len(incomplete)
		missing = append(missing, finalReadinessIncompleteTodos(incomplete))
=======
	if msg := a.capabilityGateFailure(); msg != "" {
		out.applies = true
		out.continuationUnsafe = true
		out.missingCapabilities++
		missing = append(missing, msg)
>>>>>>> origin/main-v2
	}
	writer, hasWriter := a.task.ledger.LatestSuccessfulWriterIndex()
	if mutation, ok := a.task.ledger.LatestSuccessfulMutationIndex(); ok {
		writer, hasWriter = mutation, true
	}
<<<<<<< HEAD
	deliveryMutation := false
	deliveryVerificationOnly := false
	checkpoint := a.task.checkpoint
	checkpointApplies := a.turn.deliveryScopeActive && checkpoint.ScopeID == a.task.scopeID
	if closedLoop {
		if mutation, ok := a.task.ledger.LatestSuccessfulMutationIndex(); ok {
			writer, hasWriter = mutation, true
			deliveryMutation = true
		} else if checkpointApplies && checkpoint.PendingMutation {
			// The mutation happened before a controller rebuild/restart. Treat it as
			// the baseline so this run can satisfy verification/review/sign-off
			// without manufacturing another write.
			writer, hasWriter = -1, true
			deliveryMutation = true
		} else if checkpointApplies && checkpoint.MutationObserved {
			deliveryMutation = true
		}
		workObserved := a.task.ledger.HasSuccessfulWorkReceipt() || (checkpointApplies && checkpoint.WorkObserved)
		if a.turn.deliveryTaskExpected && !a.turn.deliveryPersistentExpected && !workObserved {
			out.missingActionEvidence++
			missing = append(missing, "perform host-observable work for this technical task before answering")
		}
		if a.turn.deliveryPersistentExpected && !a.task.ledger.HasSuccessfulToolReceipt("remember") {
			out.missingMutation++
			missing = append(missing, "save the requested durable memory with the remember tool before answering")
		}
		if a.turn.deliveryMutationExpected && !deliveryMutation {
			out.missingMutation++
			missing = append(missing, "the request requires a state change, but no successful mutation was observed")
		}
		if !hasWriter && a.task.ledger.HasSuccessfulVerificationCommand() {
			writer, hasWriter = -1, true
			deliveryVerificationOnly = true
		}
		// Required/preferred capability gates apply before the no-writer fast
		// path below: a user-required Skill/MCP must not be skippable by
		// answering from ordinary reads alone.
		if msg := a.capabilityGateFailure(); msg != "" {
			out.applies = true
			out.missingCapabilities++
			missing = append(missing, msg)
		}
		if a.turn.deliveryPersistentExpected && !a.turn.deliveryMutationExpected && !a.task.ledger.HasSuccessfulMutationOtherThan("remember") {
			// A durable-memory-only request has its own concrete receipt contract.
			// It must not inherit code-delivery todo/test/diff/review ceremonies;
			// any unrelated mutation falls through to the full contract below.
			out.applies = true
			if len(missing) > 0 {
				out.reason = strings.Join(missing, "; ")
			}
			return out
		}
	}
	if !hasWriter {
		if len(missing) > 0 {
			if a.loopGuardAllowsFinal() && !atomicMutationMissing {
				return out
			}
			out.reason = strings.Join(missing, "; ")
		}
		return out
	}
	// Targeted standard turns report missing checks without opening a recovery
	// card. Hard readiness remains for high-assurance and Goal execution.
	if !qualityGated {
		if len(missing) > 0 {
			if a.loopGuardAllowsFinal() && !atomicMutationMissing {
				return finalReadinessCheck{}
			}
			out.reason = strings.Join(missing, "; ")
		}
		return out
	}
	hasProjectChecks := len(a.projectChecks) > 0
	hasTodoReceipt := a.task.ledger.HasSuccessfulTodoWrite()
	if !closedLoop && !hasProjectChecks && !hasTodoReceipt && len(missing) == 0 {
		return finalReadinessCheck{}
	}
	out.applies = true
	if closedLoop {
		missing = a.appendClosedLoopReadiness(&out, missing, writer, deliveryMutation, checkpointApplies, checkpoint)
=======
	// Incomplete todos contradict closed-loop delivery only. In open turns they
	// are cross-turn work plans and must not trigger the recovery ceremony.
	if a.closedLoopActive() && hasWriter && hasTodos && len(incomplete) > 0 {
		out.applies = true
		out.continuationHighConfidence = true
		out.incompleteTodos = len(incomplete)
		missing = append(missing, finalReadinessIncompleteTodos(incomplete))
>>>>>>> origin/main-v2
	}
	for _, check := range a.projectChecks {
		if !hasWriter {
			break
		}
		command := strings.TrimSpace(check.Command)
		if command == "" {
			continue
		}
		if !a.task.ledger.HasSuccessfulCommandAfter(command, writer) {
			out.continuationHighConfidence = true
			out.missingProjectChecks++
			missing = append(missing, fmt.Sprintf("run %q from %s after the latest write", command, finalReadinessCheckSource(check)))
		}
	}
	if hasWriter {
		outstanding := a.outstandingPlanCriteria()
		if len(outstanding) > 0 {
			out.continuationHighConfidence = true
		}
		out.missingAcceptanceCriteria += len(outstanding)
		missing = append(missing, outstanding...)
	}

	stop := runtimepolicy.StopDecision{Disposition: taskcontract.StopReady}
	if a.turn.engine != nil {
		stop = a.turn.engine.BeforeStop(runtimepolicy.StopContext{
			GoalActive:     a.turn.deliveryScopeActive,
			ApprovedPlan:   a.planContractSnapshot() != nil,
			IncompleteTodo: a.closedLoopActive() && hasTodos && len(incomplete) > 0,
			Opts: taskcontract.StopOptions{
				LoopGuard:        a.loopGuardAllowsFinal(),
				EnvUnavailable:   a.turn.constraints.ForbidTests,
				PermissionDenied: false,
			},
		})
		for _, o := range a.turn.engine.Snapshot().Unsatisfied() {
			missing = append(missing, obligationGap(o))
			out.observeObligation(o)
		}
	}
	if a.loopGuardAllowsFinal() {
		return finalReadinessCheck{applies: true}
	}
	if out.incompleteTodos > 0 && stop.Disposition == taskcontract.StopReady {
		// Incomplete todos are a fact contradiction, not an advisory gap.
		stop.Disposition = taskcontract.StopContinue
	}
	if len(missing) == 0 && stop.Disposition == taskcontract.StopReady {
		return out
	}
	out.applies = true
	switch stop.Disposition {
	case taskcontract.StopReady:
		if a.loopGuardAllowsFinal() {
			return out
		}
		if out.incompleteTodos == 0 && a.turn.engine != nil && len(a.turn.engine.Snapshot().AdvisoryGaps()) > 0 {
			return out
		}
	case taskcontract.StopPartial:
		out.reason = strings.Join(missing, "; ")
		return a.applyPartialCheckWaiver(out)
	case taskcontract.StopBlocked:
		out.continuationUnsafe = true
		out.reason = strings.Join(missing, "; ")
		return out
	case taskcontract.StopContinue:
		if a.loopGuardAllowsFinal() {
			return out
		}
		if a.turn.engine != nil {
			a.turn.engine.NoteRecoveryAttempt()
		}
		out.reason = strings.Join(missing, "; ")
		if !a.turn.automaticReadinessContinuation && !a.closedLoopActive() {
			return a.applyPartialCheckWaiver(out)
		}
		return out
	}
	if a.loopGuardAllowsFinal() {
		return out
	}
	out.reason = strings.Join(missing, "; ")
	return a.applyPartialCheckWaiver(out)
}

<<<<<<< HEAD
// finalReadinessPolicyState keeps Goal mutation closure runtime-only, preserving
// provider-visible prompt bytes and the explicit EvidenceNone test escape hatch.
func (a *Agent) finalReadinessPolicyState() (closedLoop, qualityGated bool) {
	_, goalMutationObserved := a.task.ledger.LatestSuccessfulMutationIndex()
	closedLoop = a.closedLoopActive() ||
		(a.turn.deliveryScopeActive && goalMutationObserved && a.turn.policy.Evidence != taskpolicy.EvidenceNone)
	return closedLoop, closedLoop || a.turn.deliveryScopeActive || !a.turn.policySet
}

func (a *Agent) appendClosedLoopReadiness(out *finalReadinessCheck, missing []string, writer int, deliveryMutation, checkpointApplies bool, checkpoint evidence.DeliveryCheckpoint) []string {
	a.emitTurnPhase(event.TurnPhaseVerifying)
	if !(a.turn.deliveryCriteriaEstablished || (checkpointApplies && checkpoint.CriteriaEstablished)) {
		out.missingAcceptanceCriteria++
		missing = append(missing, "establish concrete acceptance criteria with todo_write before changing state")
=======
func obligationGap(o taskcontract.Obligation) string {
	switch o.Kind {
	case taskcontract.ObligationTargetedVerify, taskcontract.ObligationFullVerify:
		return "run relevant verification after the latest mutation"
	case taskcontract.ObligationDiffReview:
		return "inspect the changed result after the latest mutation (read the touched file or run git diff/status)"
	case taskcontract.ObligationIndependentReview:
		return "run an independent review after the latest mutation"
	case taskcontract.ObligationSecurityReview:
		return "run a security review after the latest mutation"
	case taskcontract.ObligationSignoff:
		return "call complete_step after the latest mutation"
	case taskcontract.ObligationActionReceipt:
		return "record a successful action receipt for this operation"
	case taskcontract.ObligationCriteria:
		return "establish concrete acceptance criteria before changing state"
	case taskcontract.ObligationTodo:
		return "keep an in-progress todo for this write"
	default:
		return string(o.Kind)
>>>>>>> origin/main-v2
	}
}

func finalReadinessCheckSource(check instruction.VerifyCheck) string {
	source := strings.TrimSpace(check.SourcePath)
	if source == "" {
		source = "project memory"
	}
	if check.Line > 0 {
		return fmt.Sprintf("%s:%d", source, check.Line)
	}
	return source
}

func finalReadinessIncompleteTodos(items []evidence.TodoStepMatch) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		label := strings.TrimSpace(item.Content)
		if label == "" {
			label = fmt.Sprintf("todo %d", item.Index)
		}
		parts = append(parts, fmt.Sprintf("%s: %s", label, item.Status))
	}
	return "latest successful todo_write still has incomplete items: " + strings.Join(parts, ", ")
}
