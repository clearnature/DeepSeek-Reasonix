package responses

import "reasonix/internal/provider"

// OutputBudget reports the default total output budget sent by this client.
func (c *client) OutputBudget() int { return c.maxOutputTokens }

// CompactionOutputTokens reports the vendor's dedicated summary digest budget
// (deepseek 16K, dashscope 8192, mimo 4096); the agent's summary output budget
// prefers it over the default.
func (c *client) CompactionOutputTokens() int { return c.caps.compactionOutputTokens }

// SharesContextWindow is true only for the official DeepSeek vendor mode.
func (c *client) SharesContextWindow() bool { return c.vendor == "deepseek" }

func (c *client) ContextBudgetPolicy() provider.ContextBudgetPolicy {
	if lim, ok := provider.LookupOfficialOpenCodeGo("responses", c.baseURL, c.model); ok {
		return provider.ContextBudgetPolicy{
			WindowMode:       provider.ContextWindowShared,
			AutoOutputTokens: lim.MaxOutput,
			MaxOutputTokens:  lim.MaxOutput,
			LimitMode:        provider.OutputLimitAlways,
		}
	}
	if c.vendor == "deepseek" {
		return provider.ContextBudgetPolicy{
			WindowMode:       provider.ContextWindowShared,
			AutoOutputTokens: provider.DeepSeekMaxOutputTokens,
			MaxOutputTokens:  provider.DeepSeekMaxOutputTokens,
			LimitMode:        provider.OutputLimitOmitWhenSafe,
		}
	}
	return provider.ContextBudgetPolicy{WindowMode: provider.ContextWindowUnknown, LimitMode: provider.OutputLimitOmitWhenSafe}
}

// SharedWindowInputPolicy reports the Responses-only history items that the
// official DeepSeek adapter replays into later requests.
func (c *client) SharedWindowInputPolicy() provider.SharedWindowInputPolicy {
	replay := c.vendor == "deepseek"
	return provider.SharedWindowInputPolicy{
		ReplaysOrdinaryReasoning: replay,
		ReplaysResponsesItems:    replay,
	}
}
