package responses

import (
	"encoding/json"
	"fmt"
	"strings"
)

// decompose.go：大模型驱动语义拆解命题（2026-08-03 用户要求）。
// 复杂命题 → LLM 拆解为 2-5 个子命题（每子命题 = 独立因果线 + 检索任务），
// 多因果线分析后按子命题分配并行检索任务 → 信息流 → 信息帧拼图。

// Proposition is one decomposed sub-proposition of a complex query.
type Proposition struct {
	Title   string   `json:"title"`   // 子命题标题（因果线主题）
	Query   string   `json:"query"`   // 独立检索查询
	Aspects []string `json:"aspects"` // 关注维度（信号/原因/影响/后果）
}

// DecomposePrompt builds the LLM prompt that splits a complex proposition
// into independent sub-propositions. Output must be JSON (json_schema guided).
func DecomposePrompt(topic string) string {
	return fmt.Sprintf(
		"请把以下复杂研究命题拆解为 3-5 个相互独立、可分别检索的子命题。"+
			"每个子命题应覆盖该命题的一个独立因果维度（如：直接冲击、传导机制、区域差异、时间演化、应对与缓冲）。\n\n"+
			"命题：%s\n\n"+
			"只输出 JSON 数组（不要多余文字）：\n"+
			`[{"title":"子命题标题","query":"该子命题的检索查询词","aspects":["信号","影响","后果"]}]`,
		topic)
}

// DecomposeRequest is the provider request for LLM decomposition.
type DecomposeRequest struct {
	Topic string
	// MaxOutputTokens bounds the decomposition reply.
	MaxOutputTokens int
}

// ParsePropositions extracts propositions from an LLM reply that may wrap
// JSON in prose or fences. Returns an error when nothing parses.
func ParsePropositions(reply string) ([]Proposition, error) {
	text := strings.TrimSpace(reply)
	if i := strings.Index(text, "["); i >= 0 {
		if j := strings.LastIndex(text, "]"); j > i {
			text = text[i : j+1]
		}
	}
	var props []Proposition
	if err := json.Unmarshal([]byte(text), &props); err != nil {
		return nil, fmt.Errorf("parse propositions: %w", err)
	}
	// 过滤空/无效子命题
	out := make([]Proposition, 0, len(props))
	for _, p := range props {
		if strings.TrimSpace(p.Title) != "" && strings.TrimSpace(p.Query) != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid propositions in reply")
	}
	return out, nil
}

// DecomposeFunc runs the LLM decomposition. Implementers call the provider
// with DecomposePrompt + json_schema and return the raw reply text.
type DecomposeFunc func(topic string, maxTokens int) (string, error)

// DecomposeProposition orchestrates the LLM split via DecomposeFunc.
func DecomposeProposition(topic string, fn DecomposeFunc) ([]Proposition, error) {
	if fn == nil {
		return nil, fmt.Errorf("decompose func required")
	}
	reply, err := fn(topic, 4096)
	if err != nil {
		return nil, err
	}
	return ParsePropositions(reply)
}
