package project

import (
	"encoding/json"
	"testing"
)

func TestImportExecuteRequestOmitsAbsentLifecycleConfiguration(t *testing.T) {
	payload, err := json.Marshal(ImportExecuteRequest{InspectionID: "inspection-1"})
	if err != nil {
		t.Fatalf("marshal import execute request: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatalf("unmarshal import execute request: %v", err)
	}
	if _, exists := fields["lifecycle_configuration"]; exists {
		t.Fatalf("expected absent lifecycle configuration to be omitted, got %s", payload)
	}
}
