package responses

import (
	"testing"

	"reasonix/internal/provider"
)

// TestEncodeResponsesTextFormatCarriesSchemaName locks in the json_schema
// wire shape: the local retrieval extension must serialize name + schema —
// DeepSeek rejects a nameless json_schema with 400 ("missing field `name`"),
// which retrieve_info hit when only {"type":"json_schema"} was sent.
func TestEncodeResponsesTextFormatCarriesSchemaName(t *testing.T) {
	rf := provider.JSONSchemaFormat("knowledge_extract", map[string]any{"type": "object"})
	text := encodeResponsesTextFormat(rf)
	format, _ := text["format"].(map[string]any)
	if format["name"] != "knowledge_extract" {
		t.Fatalf("format.name = %v, want knowledge_extract", format["name"])
	}
	if _, ok := format["schema"]; !ok {
		t.Fatal("format.schema must be serialized for json_schema")
	}

	jsonObj := encodeResponsesTextFormat(&provider.ResponseFormat{Type: "json_object"})
	f, _ := jsonObj["format"].(map[string]any)
	if _, hasName := f["name"]; hasName {
		t.Fatal("plain json_object must not carry a schema name")
	}
}
