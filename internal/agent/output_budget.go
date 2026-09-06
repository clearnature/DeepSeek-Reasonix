package agent

import (
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"reasonix/internal/event"
	"reasonix/internal/nilutil"
	"reasonix/internal/provider"
)

const (
	outputBudgetReserve    = 8 * 1024
	minOutputBudgetReserve = protocolReserveTokens
)

const learnedOutputBudgetTTL = 24 * time.Hour

type learnedOutputBudgetCacheEntry struct {
	completionBudget int
	expiresAt        time.Time
}

var learnedOutputBudgetCache = struct {
	sync.Mutex
	entries map[string]learnedOutputBudgetCacheEntry
}{entries: make(map[string]learnedOutputBudgetCacheEntry)}

type outputBudgetState struct {
	// lastPersistedRatio de-dupes calibration writes: persist only when the
	// ratio moved meaningfully, so a request every few seconds does not hit
	// the disk on every turn.
	lastPersistedRatio float64
	outputBudget       int
	// lastUsage caches the latest provider telemetry for per-turn readouts.
	// The run loop writes it while a frontend reads it, so it is atomic.
	lastUsage         atomic.Pointer[provider.Usage]
	activeReqShape    atomic.Pointer[requestCalibrationShape]
	promptCalibration atomic.Pointer[promptTokenCalibration]
	contextUsage      atomic.Pointer[contextUsage] // gauge's memoised prompt size
	learned           atomic.Pointer[learnedContextBudget]
	admission         atomic.Pointer[contextAdmission]
}

// learnedContextBudget is an Agent-local observation of the live provider/model
// window. Completion limits are additionally shared through the short-lived
// provider/model cache below so a model rebuild does not immediately repeat a
// known over-limit request. The cache is deliberately in-memory and expires
// after one day; it is not a persisted global provider limit.
type learnedContextBudget struct {
	windowTokens     int
	completionBudget int
}

const (
	contextRecoveryNone          = "none"
	contextRecoveryProactiveClip = "proactive_clip"
	contextRecoveryLearnedRetry  = "learned_retry"
	contextRecoveryCompacted     = "compacted"
	contextRecoveryFailed        = "failed"
)

type contextAdmission struct {
	WindowMode            string
	LimitMode             string
	Source                string
	WindowTokens          int
	PromptTokens          int
	AutoOutputTokens      int
	MaxOutputTokens       int
	RequestedOutputTokens int
	EffectiveOutputTokens int
	ReserveTokens         int
	PhysicalRemaining     int
	Clipped               bool
	ApplyMaxTokens        bool
	LastRecovery          string
	ObservedWindow        int
	ObservedPrompt        int
	ObservedCompletion    int
}

type promptTokenCalibration struct {
	promptTokens int
	requestChars int64
	compactChars int64
	cjkRunes     int64
	cjkBytes     int64
}

// requestCalibrationShape pairs the conservative provider-visible text and CJK
// composition used for overflow protection with the legacy content-only shape
// used by fold economics. Keeping them in one immutable pointer ensures readers
// never combine calibration fields from different prepared requests.
type requestCalibrationShape struct {
	requestChars int64
	compactChars int64
	cjkRunes     int64
	cjkBytes     int64
}

// reset drops what belongs to the transcript being replaced. Prompt-token
// calibration and the learned window are properties of the bound model/provider,
// and a model switch rebuilds the Agent, so they outlive SetSession. Admission
// describes one transcript's latest request and must never bleed into the next.
func (o *outputBudgetState) reset() {
	o.lastUsage.Store(nil)
	o.activeReqShape.Store(nil)
	o.admission.Store(nil)
}

func (a *Agent) setPromptTokenCalibration(promptTokens int, shape requestCalibrationShape) {
	if a == nil || promptTokens <= 0 || shape.requestChars <= 0 {
		return
	}
	cal := &promptTokenCalibration{
		promptTokens: promptTokens,
		requestChars: shape.requestChars,
		compactChars: shape.compactChars,
		cjkRunes:     shape.cjkRunes,
		cjkBytes:     shape.cjkBytes,
	}
	a.sess.output.promptCalibration.Store(cal)
	// Best-effort persistence: write only when the ratio moved ≥2% vs the
	// last persisted value (first write always persists).
	if r := float64(promptTokens) / float64(shape.compactChars); r > 0 && r != a.sess.output.lastPersistedRatio &&
		(a.sess.output.lastPersistedRatio == 0 || absRatioDelta(r, a.sess.output.lastPersistedRatio) >= 0.02) {
		a.sess.output.lastPersistedRatio = r
		persistCalibration(a.calibrationKey(), cal)
	}
}

func (a *Agent) setPromptTokenCalibrationFromActive(promptTokens int) {
	if a == nil {
		return
	}
	if shape := a.sess.output.activeReqShape.Load(); shape != nil {
		a.setPromptTokenCalibration(promptTokens, *shape)
	}
}

// setPromptTokenCalibrationFromUsage trusts provider telemetry only;
// reconstructed usage remains available for accounting but not admission.
func (a *Agent) setPromptTokenCalibrationFromUsage(usage *provider.Usage) {
	if a == nil || usage == nil || usage.Estimated {
		return
	}
	a.setPromptTokenCalibrationFromActive(usage.LatestPromptTokens())
}

func outputBudgetOf(p provider.Provider) int {
	if nilutil.IsNil(p) {
		return 0
	}
	if budget, ok := p.(provider.OutputBudgetProvider); ok {
		return budget.OutputBudget()
	}
	return 0
}

func sharesContextWindow(p provider.Provider) bool {
	return contextBudgetPolicyOf(p).WindowMode == provider.ContextWindowShared
}

func contextBudgetPolicyOf(p provider.Provider) provider.ContextBudgetPolicy {
	if nilutil.IsNil(p) {
		return provider.ContextBudgetPolicy{}
	}
	return provider.ResolveContextBudgetPolicy(p)
}

func (a *Agent) learnOutputBudget(limit int) {
	if a == nil || limit <= 0 {
		return
	}
	cacheLearnedOutputBudget(outputBudgetCacheKey(a), limit)
	current := a.sess.output.learned.Load()
	if current != nil && current.completionBudget > 0 && current.completionBudget <= limit {
		return
	}
	learned := &learnedContextBudget{completionBudget: limit}
	if current != nil {
		learned.windowTokens = current.windowTokens
	}
	a.sess.output.learned.Store(learned)
}

func outputBudgetCacheKey(a *Agent) string {
	if a == nil {
		return ""
	}
	providerName := ""
	if !nilutil.IsNil(a.svc.prov) {
		providerName = strings.TrimSpace(a.svc.prov.Name())
	}
	modelRef := strings.TrimSpace(a.modelRef)
	if providerName == "" && modelRef == "" {
		return ""
	}
	// Provider names are route-specific for the built-in OpenCode Go entries;
	// retaining modelRef as a second component keeps custom routes isolated too.
	return providerName + "|" + modelRef
}

func cacheLearnedOutputBudget(key string, limit int) {
	if strings.TrimSpace(key) == "" || limit <= 0 {
		return
	}
	now := time.Now()
	learnedOutputBudgetCache.Lock()
	defer learnedOutputBudgetCache.Unlock()
	if current, ok := learnedOutputBudgetCache.entries[key]; ok && current.expiresAt.After(now) && current.completionBudget > 0 && current.completionBudget <= limit {
		return
	}
	learnedOutputBudgetCache.entries[key] = learnedOutputBudgetCacheEntry{
		completionBudget: limit,
		expiresAt:        now.Add(learnedOutputBudgetTTL),
	}
}

func cachedLearnedOutputBudget(key string) int {
	if strings.TrimSpace(key) == "" {
		return 0
	}
	now := time.Now()
	learnedOutputBudgetCache.Lock()
	defer learnedOutputBudgetCache.Unlock()
	entry, ok := learnedOutputBudgetCache.entries[key]
	if !ok {
		return 0
	}
	if !entry.expiresAt.After(now) {
		delete(learnedOutputBudgetCache.entries, key)
		return 0
	}
	return entry.completionBudget
}

func sharedWindowInputPolicyOf(p provider.Provider) provider.SharedWindowInputPolicy {
	if nilutil.IsNil(p) {
		return provider.SharedWindowInputPolicy{}
	}
	policy, ok := p.(provider.SharedWindowInputPolicyProvider)
	if !ok {
		return provider.SharedWindowInputPolicy{}
	}
	return policy.SharedWindowInputPolicy()
}

func requestCalibrationShapeOf(req provider.Request) requestCalibrationShape {
	return requestCalibrationShapeWithPolicy(req, provider.SharedWindowInputPolicy{})
}

func (a *Agent) requestCalibrationShape(req provider.Request) requestCalibrationShape {
	return requestCalibrationShapeWithPolicy(req, sharedWindowInputPolicyOf(a.svc.prov))
}

func requestCalibrationShapeWithPolicy(req provider.Request, policy provider.SharedWindowInputPolicy) requestCalibrationShape {
	requestChars, cjkRunes, cjkBytes := requestCalibrationTextShape(req, policy)
	return requestCalibrationShape{
		requestChars: requestChars,
		compactChars: int64(charsOfMessages(req.Messages)),
		cjkRunes:     cjkRunes,
		cjkBytes:     cjkBytes,
	}
}

// requestCalibrationTextShape counts common shared-window text plus only the
// adapter-specific replay fields declared by the active provider. This keeps
// omitted bytes out of the ratio without missing newly appended wire content.
func requestCalibrationTextShape(req provider.Request, policy provider.SharedWindowInputPolicy) (chars, cjkRunes, cjkBytes int64) {
	add := func(s string) {
		chars += int64(len(s))
		for _, r := range s {
			if isCJKRune(r) {
				cjkRunes++
				cjkBytes += int64(utf8.RuneLen(r))
			}
		}
	}
	for _, msg := range req.Messages {
		if msg.LocalOnly {
			continue
		}
		chars += 4
		add(string(msg.Role))
		add(msg.Content)
		if msg.Role == provider.RoleAssistant && (len(msg.ToolCalls) > 0 || policy.ReplaysOrdinaryReasoning) {
			add(msg.ReasoningContent)
		}
		add(msg.Name)
		add(msg.ToolCallID)
		for _, call := range msg.ToolCalls {
			chars += 8
			add(call.ID)
			add(call.Name)
			add(call.Arguments)
		}
		if policy.ReplaysResponsesItems {
			for _, item := range msg.ResponsesItems {
				add(string(item))
			}
		}
		for _, search := range msg.ServerSearch {
			provider.WalkServerSearchEstimate(search, add)
		}
	}
	for _, schema := range req.Tools {
		chars += 8
		add(schema.Name)
		add(schema.Description)
		add(string(schema.Parameters))
	}
	return chars, cjkRunes, cjkBytes
}

func (a *Agent) calibratedPromptTokens(shape requestCalibrationShape) (int, bool) {
	if shape.requestChars <= 0 {
		return 0, false
	}
	if cal := a.sess.output.promptCalibration.Load(); cal != nil && cal.requestChars > 0 {
		ratio := float64(cal.promptTokens) / float64(cal.requestChars)
		if ratio > 0.05 && ratio < 2 {
			trustedChars := shape.requestChars
			excessCJKBytes := int64(0)
			// A higher CJK share cannot safely reuse the aggregate ratio. Scale its
			// represented share and price only the excess at the cold rate,
			// preserving exact calibration for stable CJK sessions.
			if shape.cjkRunes*cal.requestChars > cal.cjkRunes*shape.requestChars {
				trustedCJKBytes := min(cal.cjkBytes*shape.requestChars/cal.requestChars, shape.cjkBytes)
				excessCJKBytes = shape.cjkBytes - trustedCJKBytes
				trustedChars -= excessCJKBytes
			}
			cold := math.Ceil(float64(excessCJKBytes) * fallbackTokPerChar)
			return int(math.Ceil(float64(trustedChars)*ratio) + cold), true
		}
	}
	return 0, false
}

// estimatedPromptTokens sizes the provider-visible messages in real tokens —
// the only unit comparable against the context window. Same-session usage
// calibrates it; before that the wire character count carries the ~4 chars per
// token shape. estimateMessagesTokens counts characters and is for internal
// planning budgets only; against the window it would compact 4x early.
func (a *Agent) estimatedPromptTokens(msgs []provider.Message) int {
	return a.estimatedShapeTokens(a.requestCalibrationShape(provider.Request{Messages: msgs}))
}

func (a *Agent) estimatedRequestTokens(req provider.Request) int {
	return a.estimatedShapeTokens(a.requestCalibrationShape(req))
}

func (a *Agent) estimatedShapeTokens(shape requestCalibrationShape) int {
	if shape.requestChars <= 0 {
		return 0
	}
	if calibrated, ok := a.calibratedPromptTokens(shape); ok {
		return calibrated
	}
	return int(float64(shape.requestChars) * fallbackTokPerChar)
}

// replayTailSlack bounds the projection-covered prefix comparison below; a
// message set that far exceeds the covered count is a canonical full replay.
const replayTailSlack = 8

// estimatedReplaySafeTokens prices the request like estimatedShapeTokens but
// guards the display/admission estimate against canonical full replay after
// projection loss: a whole-transcript estimate (e.g. gpu1 5.6M chars → 1.9M
// tokens) fakes the window and forces an unnecessary compaction on a warm
// replay. When the message set clearly exceeds the projection-covered prefix
// and a last observed prompt exists, it prices the covered history at the
// observed baseline and only the uncovered tail above it. The raw estimate is
// returned separately so anomaly diagnostics still see the inflation source.
func (a *Agent) estimatedReplaySafeTokens(req provider.Request, shape requestCalibrationShape) (est, raw int) {
	raw = a.estimatedShapeTokens(shape)
	if a == nil {
		return raw, raw
	}
	a.sess.compactionMu.Lock()
	covered := a.sess.compactionState.Projection.CoveredCount
	a.sess.compactionMu.Unlock()
	if covered <= 0 || len(req.Messages) <= covered+replayTailSlack {
		return raw, raw
	}
	last := 0
	if u := a.LastUsage(); u != nil {
		last = u.LatestPromptTokens()
	}
	if last <= 0 {
		return raw, raw
	}
	var tailChars int64
	for _, m := range req.Messages[covered:] {
		tailChars += int64(len(m.Content))
	}
	est = last + int(float64(tailChars)*fallbackTokPerChar)
	if est < raw {
		return est, raw
	}
	return raw, raw
}

func isCJKRune(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x3040 && r <= 0x30FF) ||
		(r >= 0xAC00 && r <= 0xD7AF)
}

func (a *Agent) effectiveContextWindow() int {
	if a == nil {
		return 0
	}
	cfg := a.contextWindow
	learned := 0
	if snap := a.sess.output.learned.Load(); snap != nil {
		learned = snap.windowTokens
	}
	switch {
	case cfg > 0 && learned > 0:
		return min(cfg, learned)
	case learned > 0:
		return learned
	default:
		return cfg
	}
}

func (a *Agent) learnedCompletionBudget() int {
	if a == nil {
		return 0
	}
	cached := cachedLearnedOutputBudget(outputBudgetCacheKey(a))
	if snap := a.sess.output.learned.Load(); snap != nil {
		if cached > 0 && (snap.completionBudget <= 0 || cached < snap.completionBudget) {
			a.learnOutputBudget(cached)
			return cached
		}
		return snap.completionBudget
	}
	if cached > 0 {
		a.learnOutputBudget(cached)
		return cached
	}
	return 0
}

func (a *Agent) learnContextBudget(window, completion int, omittedOutput bool) {
	if a == nil {
		return
	}
	cur := learnedContextBudget{}
	if prev := a.sess.output.learned.Load(); prev != nil {
		cur = *prev
	}
	if window > 0 {
		if cur.windowTokens <= 0 || window < cur.windowTokens {
			cur.windowTokens = window
		}
	}
	if omittedOutput && completion > 0 {
		cur.completionBudget = completion
	}
	next := cur
	a.sess.output.learned.Store(&next)
}

func (a *Agent) storeAdmission(adm contextAdmission) {
	if a == nil {
		return
	}
	cp := adm
	a.sess.output.admission.Store(&cp)
}

func (a *Agent) lastAdmission() contextAdmission {
	if a == nil {
		return contextAdmission{LastRecovery: contextRecoveryNone}
	}
	if snap := a.sess.output.admission.Load(); snap != nil {
		return *snap
	}
	return contextAdmission{LastRecovery: contextRecoveryNone}
}

func (a *Agent) setLastRecovery(kind string) {
	if a == nil {
		return
	}
	adm := a.lastAdmission()
	adm.LastRecovery = kind
	a.storeAdmission(adm)
}

func admissionSource(userMax int, policy provider.ContextBudgetPolicy, learnedWindow bool) string {
	if learnedWindow {
		return provider.ContextBudgetSourceLearned
	}
	if userMax > 0 {
		return provider.ContextBudgetSourceExplicit
	}
	switch {
	case policy.AutoOutputTokens == provider.DeepSeekMaxOutputTokens && policy.LimitMode == provider.OutputLimitOmitWhenSafe:
		return provider.ContextBudgetSourceOfficial
	case policy.LimitMode == provider.OutputLimitAlways && policy.MaxOutputTokens > 0:
		return provider.ContextBudgetSourceOpenCode
	case policy.WindowMode == provider.ContextWindowUnknown || policy.AutoOutputTokens <= 0:
		return provider.ContextBudgetSourceUnknown
	default:
		return provider.ContextBudgetSourceOfficial
	}
}

// effectiveOutputBudget clips completion tokens at send time only; it never
// moves compact_ratio. Calibrated exhausted windows fail locally; a cold
// estimate that differs from the provider tokenizer uses bounded 400 recovery.
func (a *Agent) effectiveOutputBudget(req provider.Request, useObserved bool) (int, bool, error) {
	adm, err := a.admitOutputBudget(req)
	if err != nil {
		return 0, false, err
	}
	if !adm.ApplyMaxTokens || !adm.Clipped {
		if adm.ApplyMaxTokens && adm.EffectiveOutputTokens > 0 && !adm.Clipped {
			return adm.EffectiveOutputTokens, false, nil
		}
		return 0, false, nil
	}
	return adm.EffectiveOutputTokens, true, nil
}

func (a *Agent) admitOutputBudget(req provider.Request) (contextAdmission, error) {
	return a.admitOutputBudgetWithReserve(req, outputBudgetReserve)
}

func (a *Agent) admitOutputBudgetWithReserve(req provider.Request, reserveTokens int) (contextAdmission, error) {
	adm := contextAdmission{
		ReserveTokens: reserveTokens,
		LastRecovery:  a.lastAdmission().LastRecovery,
		Source:        provider.ContextBudgetSourceUnknown,
	}
	if adm.LastRecovery == "" {
		adm.LastRecovery = contextRecoveryNone
	}
	if a == nil {
		return adm, nil
	}
	if learned := a.sess.output.learned.Load(); learned != nil {
		adm.ObservedWindow = learned.windowTokens
		adm.ObservedCompletion = learned.completionBudget
	}
	policy := contextBudgetPolicyOf(a.svc.prov)
	if policy.WindowMode == provider.ContextWindowUnknown && adm.ObservedWindow > 0 {
		policy.WindowMode = provider.ContextWindowShared
	}
	if policy.AutoOutputTokens <= 0 && a.learnedCompletionBudget() > 0 {
		policy.AutoOutputTokens = a.learnedCompletionBudget()
	}
	if learned := a.learnedCompletionBudget(); learned > 0 {
		if policy.AutoOutputTokens <= 0 || learned < policy.AutoOutputTokens {
			policy.AutoOutputTokens = learned
		}
		if policy.MaxOutputTokens <= 0 || learned < policy.MaxOutputTokens {
			policy.MaxOutputTokens = learned
		}
	}
	adm.WindowMode = policy.WindowMode.String()
	adm.LimitMode = policy.LimitMode.String()
	adm.AutoOutputTokens = policy.AutoOutputTokens
	adm.MaxOutputTokens = policy.MaxOutputTokens
	window := a.effectiveContextWindow()
	adm.WindowTokens = window
	learnedWindow := window > 0 && (a.contextWindow <= 0 || window < a.contextWindow)
	adm.Source = admissionSource(req.MaxTokens, policy, learnedWindow)
	if window <= 0 {
		a.storeAdmission(adm)
		return adm, nil
	}
	shape := a.requestCalibrationShape(req)
	est, rawEst := a.estimatedReplaySafeTokens(req, shape)
	a.emitEstimateAnomaly(req, shape, rawEst, window)
	adm.PromptTokens = est
	physical := window - est - outputBudgetReserve
	adm.PhysicalRemaining = physical
	shared := policy.WindowMode == provider.ContextWindowShared
	if !shared {
		a.applyLimitMode(&adm, req.MaxTokens, policy, physical)
		a.storeAdmission(adm)
		return adm, nil
	}
	if physical <= 0 {
		a.storeAdmission(adm)
		return adm, fmt.Errorf("%w: estimated prompt %d leaves no shared-window output budget", ErrCompactionRequired, est)
	}
	requested := 0
	switch {
	case req.MaxTokens > 0:
		requested = req.MaxTokens
	default:
		requested = policy.AutoOutputTokens
	}
	if policy.MaxOutputTokens > 0 && requested > policy.MaxOutputTokens {
		requested = policy.MaxOutputTokens
	}
	adm.RequestedOutputTokens = requested
	if req.MaxTokens < 0 {
		if requested > 0 && requested > physical {
			a.storeAdmission(adm)
			return adm, fmt.Errorf("%w: estimated prompt %d leaves no room for omitted auto output %d", ErrCompactionRequired, est, requested)
		}
		a.storeAdmission(adm)
		return adm, nil
	}
	if requested <= 0 {
		a.applyLimitMode(&adm, req.MaxTokens, policy, physical)
		a.storeAdmission(adm)
		return adm, nil
	}
	effective := requested
	if effective > physical {
		effective = physical
		adm.Clipped = true
	}
	adm.EffectiveOutputTokens = effective
	a.applyLimitMode(&adm, req.MaxTokens, policy, physical)
	if adm.Clipped {
		adm.ApplyMaxTokens = req.MaxTokens >= 0 && policy.LimitMode != provider.OutputLimitUnsupported
		adm.EffectiveOutputTokens = effective
	}
	if adm.Clipped && adm.LastRecovery == contextRecoveryNone {
		adm.LastRecovery = contextRecoveryProactiveClip
	}
	a.storeAdmission(adm)
	return adm, nil
}

func (a *Agent) applyAdmissionToRequest(req *provider.Request) error {
	if a == nil || req == nil {
		return nil
	}
	adm, err := a.admitOutputBudget(*req)
	if err != nil {
		return err
	}
	if adm.ApplyMaxTokens && adm.EffectiveOutputTokens > 0 {
		req.MaxTokens = adm.EffectiveOutputTokens
	}
	return nil
}

func (a *Agent) applySummaryAdmissionToRequest(req *provider.Request) error {
	if a == nil || req == nil {
		return nil
	}
	adm, err := a.admitOutputBudgetWithReserve(*req, protocolReserveTokens)
	if err != nil {
		return err
	}
	if adm.ApplyMaxTokens && adm.EffectiveOutputTokens > 0 {
		req.MaxTokens = adm.EffectiveOutputTokens
	}
	return nil
}

// estimateAnomalyReason reports why an admission-time prompt estimate looks
// unreliable. overflow means the estimate alone already crosses the shared
// window (admitOutputBudget will refuse or force-compact); inflated means the
// estimate is far above the last provider-observed prompt on a meaningful
// request, which would fake the desktop context percentage without necessarily
// overflowing. Returns "" when the estimate looks trustworthy.
func (a *Agent) estimateAnomalyReason(est, window int) string {
	if est >= window-outputBudgetReserve {
		return "overflow"
	}
	if obs := a.lastAdmission().ObservedPrompt; obs > 0 && est > obs*2 && est >= window*2/3 {
		return "inflated"
	}
	return ""
}

// emitEstimateAnomaly persists the request shape behind a suspicious estimate
// (overflow or inflated vs observed) so the culprit message can be pinned from
// the stats file. The estimate is otherwise only shown on the desktop and never
// written to disk. Observer-only: it never changes admission behavior.
func (a *Agent) emitEstimateAnomaly(req provider.Request, shape requestCalibrationShape, est, window int) {
	if a == nil || a.svc.sink == nil {
		return
	}
	reason := a.estimateAnomalyReason(est, window)
	if reason == "" {
		return
	}
	topRole, topChars := "", 0
	for _, msg := range req.Messages {
		if n := len(msg.Content); n > topChars {
			topChars, topRole = n, string(msg.Role)
		}
	}
	_, calibrated := a.calibratedPromptTokens(shape)
	detail := fmt.Sprintf("reason=%s est=%d window=%d obs=%d chars=%d cchars=%d cjk=%d cjkb=%d msgs=%d top_role=%s top_chars=%d cal=%t",
		reason, est, window, a.lastAdmission().ObservedPrompt, shape.requestChars, shape.compactChars,
		shape.cjkRunes, shape.cjkBytes, len(req.Messages), topRole, topChars, calibrated)
	slog.Warn("agent: estimated prompt anomaly", "detail", detail)
	a.svc.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Audience: event.NoticeAudienceOperator,
		Text: "estimate telemetry", Detail: detail})
}

func (a *Agent) applyLimitMode(adm *contextAdmission, userMax int, policy provider.ContextBudgetPolicy, physical int) {
	if userMax < 0 || policy.LimitMode == provider.OutputLimitUnsupported {
		adm.ApplyMaxTokens = false
		return
	}
	effective := adm.EffectiveOutputTokens
	if effective <= 0 {
		if userMax > 0 {
			effective = userMax
		} else {
			effective = policy.AutoOutputTokens
		}
		if policy.MaxOutputTokens > 0 && effective > policy.MaxOutputTokens {
			effective = policy.MaxOutputTokens
		}
		if policy.WindowMode == provider.ContextWindowShared && physical > 0 && effective > physical {
			effective = physical
			adm.Clipped = true
		}
	}
	switch policy.LimitMode {
	case provider.OutputLimitAlways, provider.OutputLimitRequired:
		if effective > 0 {
			adm.ApplyMaxTokens = true
			adm.EffectiveOutputTokens = effective
		}
	case provider.OutputLimitOmitWhenSafe:
		if userMax > 0 || adm.Clipped {
			adm.ApplyMaxTokens = true
			adm.EffectiveOutputTokens = effective
		}
	}
}

// outputBudgetReserveForWindow returns the reserve tokens for a given context window.
func outputBudgetReserveForWindow(window int) int {
	if window <= 0 {
		return outputBudgetReserve
	}
	return min(outputBudgetReserve, max(minOutputBudgetReserve, window/128))
}
