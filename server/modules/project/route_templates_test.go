package project

import (
	"encoding/json"
	"testing"
	"time"

	projectstore "graft/server/modules/project/store"
)

func TestTemplateAggregateHTTPPreservesOpenAPIFields(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	response := templateAggregateHTTP(projectstore.ApplicationTemplateAggregate{
		Template: projectstore.ApplicationTemplate{ID: "tpl_1", DisplayName: "Demo", UpdatedAt: updatedAt},
		Version:  projectstore.ApplicationTemplateVersion{ID: "tplv_1", VersionNumber: 2, DefinitionJSON: []byte(`{"services":{}}`)},
	})
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal template response: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("decode template response: %v", err)
	}
	for _, field := range []string{"template_id", "display_name", "description", "category", "deployment_adapter_kind", "updated_at", "version"} {
		if _, ok := fields[field]; !ok {
			t.Fatalf("response field %q is missing: %s", field, raw)
		}
	}
	if _, ok := fields["archived_at"]; ok {
		t.Fatalf("archived_at = %s, want omitted", fields["archived_at"])
	}
	if string(fields["version"]) == "null" {
		t.Fatal("version must be an object")
	}
}
