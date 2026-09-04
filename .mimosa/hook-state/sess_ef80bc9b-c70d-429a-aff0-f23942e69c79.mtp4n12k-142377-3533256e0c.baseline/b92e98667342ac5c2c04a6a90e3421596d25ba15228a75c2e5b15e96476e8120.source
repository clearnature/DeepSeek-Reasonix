package provider

// OutputBudgetProvider reports the output budget used when a request leaves it unset.
type OutputBudgetProvider interface {
	OutputBudget() int
}

// CompactionOutputTokensProvider sizes summary/compaction requests separately
// from ordinary output (vendor-specific digest budgets: dashscope 8192,
// deepseek 16K, mimo 4096). Zero means "no dedicated budget; use the default".
type CompactionOutputTokensProvider interface {
	CompactionOutputTokens() int
}

// SharedWindowOutputProvider reports whether input and output share one window.
type SharedWindowOutputProvider interface {
	SharesContextWindow() bool
}

// SharedWindowInputPolicy describes adapter-specific text replayed into a
// shared context window in addition to the fields common to all transports.
type SharedWindowInputPolicy struct {
	ReplaysOrdinaryReasoning bool
	ReplaysResponsesItems    bool
}

// SharedWindowInputPolicyProvider reports adapter-specific replay behavior so
// admission estimates can follow the actual wire without changing its bytes.
type SharedWindowInputPolicyProvider interface {
	SharedWindowInputPolicy() SharedWindowInputPolicy
}
