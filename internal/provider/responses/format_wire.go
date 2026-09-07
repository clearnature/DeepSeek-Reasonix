package responses

import (
	"encoding/json"

	"reasonix/internal/provider"
)

// encodeResponsesTextFormat builds the Responses text object's format for the
// wire. json_schema (the local retrieval extension) must carry name + schema —
// DeepSeek rejects a nameless json_schema with 400 ("missing field `name`"),
// which retrieve_info hit when only {"type":"json_schema"} was serialized.
// Lives in its own file so buildRequestBody does not drift further past its
// repolint budget.
func encodeResponsesTextFormat(rf *provider.ResponseFormat) map[string]any {
	format := map[string]any{"type": rf.Type}
	if rf.Name != "" {
		format["name"] = rf.Name
	}
	if len(rf.Schema) > 0 {
		format["schema"] = json.RawMessage(rf.Schema)
	}
	return map[string]any{"format": format}
}
