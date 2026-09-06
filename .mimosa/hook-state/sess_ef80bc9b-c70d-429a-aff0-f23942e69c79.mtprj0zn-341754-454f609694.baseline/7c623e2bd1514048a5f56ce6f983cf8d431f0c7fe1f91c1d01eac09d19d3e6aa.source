package responses

import (
	"strings"
)

// DistillEntry turns one web_search response text into a structured
// KnowledgeEntry: JSON extraction first (knowledge_extract schema), then
// markdown fallback for the common DeepSeek shape ("## 来源" + "- 标题：URL"
// lines, "**N. 事实**：..." lines). Shared by the retrieve_info tool and the
// websearch-smoke entry so both persist the same distilled shape.
func DistillEntry(query, text string, tokens int, tier string) *KnowledgeEntry {
	entry := &KnowledgeEntry{
		Query:         query,
		AnswerSummary: text,
		TotalTokens:   tokens,
		Tier:          tier,
	}
	if v, ok := ExtractJSONFromOutput(text); ok {
		if obj, ok := v.(map[string]any); ok {
			if s, ok := obj["answer_summary"].(string); ok && s != "" {
				entry.AnswerSummary = s
			}
			if facts, ok := obj["key_facts"].([]any); ok {
				for _, f := range facts {
					if fs, ok := f.(string); ok {
						entry.KeyFacts = append(entry.KeyFacts, fs)
					}
				}
			}
			if srcs, ok := obj["sources"].([]any); ok {
				for _, s := range srcs {
					if sm, ok := s.(map[string]any); ok {
						src := Source{}
						if t, ok := sm["title"].(string); ok {
							src.Title = t
						}
						if u, ok := sm["url"].(string); ok {
							src.URL = u
						}
						if sn, ok := sm["snippet"].(string); ok {
							src.Snippet = sn
						}
						entry.Sources = append(entry.Sources, src)
					}
				}
			}
		}
	}
	// DeepSeek json_schema is advisory: the reply is often markdown
	// ("## 关键事实 / ## 来源") instead of JSON. Fall back to markdown
	// extraction so sources/facts still land in the cache.
	if len(entry.Sources) == 0 {
		entry.Sources = extractMarkdownSources(text)
	}
	if len(entry.KeyFacts) == 0 {
		entry.KeyFacts = extractMarkdownFacts(text)
	}
	return entry
}

// extractMarkdownSources parses the common DeepSeek web_search markdown
// shape: a "## 来源" (or "Sources") section with "- 标题：URL" lines.
func extractMarkdownSources(text string) []Source {
	var out []Source
	lines := strings.Split(text, "\n")
	inSources := false
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		// 标题形式：## 来源 / ## Sources / ## References / 数据来源
		if strings.HasPrefix(trimmed, "## ") {
			inSources = strings.Contains(trimmed, "来源") || strings.Contains(trimmed, "Sources") ||
				strings.Contains(trimmed, "References") || strings.Contains(trimmed, "参考")
			continue
		}
		// 冒号行形式：数据来源：... / 来源：...
		if strings.HasPrefix(trimmed, "数据来源") || strings.HasPrefix(trimmed, "来源：") ||
			strings.HasPrefix(trimmed, "数据来源：") {
			body := strings.TrimPrefix(trimmed, "数据来源")
			body = strings.TrimPrefix(body, "：")
			body = strings.TrimPrefix(body, ":")
			body = strings.TrimPrefix(body, "来源")
			body = strings.TrimPrefix(body, "：")
			body = strings.TrimPrefix(body, ":")
			body = strings.TrimSpace(body)
			if body != "" && !strings.Contains(body, "http") {
				// 无 URL 的来源描述（如 "新加坡海事及港务管理局 MPA 2026年1月发布"）
				out = append(out, Source{Title: body, URL: ""})
			}
			continue
		}
		if !inSources {
			continue
		}
		// - 标题：https://...
		if after, ok := strings.CutPrefix(trimmed, "- "); ok {
			body := after
			// find first http(s):// URL
			urlStart := -1
			for i := 0; i+7 <= len(body); i++ {
				if (body[i] == 'h' && len(body) > i+7 && body[i:i+8] == "https://") ||
					(body[i] == 'h' && len(body) > i+6 && body[i:i+7] == "http://") {
					urlStart = i
					break
				}
			}
			if urlStart < 0 {
				continue
			}
			src := Source{
				Title: strings.TrimSpace(strings.TrimSuffix(body[:urlStart], "：")),
				URL:   strings.Fields(body[urlStart:])[0],
			}
			out = append(out, src)
		}
	}
	return out
}

// extractMarkdownFacts parses "**N. 事实**：..." or numbered list items.
func extractMarkdownFacts(text string) []string {
	var out []string
	for ln := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(ln)
		// **N. 标题**：内容  or  N. 内容
		if strings.HasPrefix(trimmed, "**") && strings.Contains(trimmed, "**：") {
			parts := strings.SplitN(trimmed, "**：", 2)
			title := strings.TrimSpace(strings.TrimPrefix(parts[0], "**"))
			body := strings.TrimSpace(parts[1])
			if body != "" {
				if title != "" {
					out = append(out, title+"："+body)
				} else {
					out = append(out, body)
				}
			}
			continue
		}
		// 数字列表：1. ... / N. ...
		if len(trimmed) > 2 && (trimmed[1] == '.' || trimmed[2] == '.') && isDigit(trimmed[0]) {
			body := strings.TrimSpace(strings.SplitN(trimmed, ".", 2)[1])
			if body != "" && !strings.HasPrefix(body, "http") {
				out = append(out, body)
			}
		}
	}
	return out
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }
