package openapi

import (
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPI31RetainsNullableResponseProperties(t *testing.T) {
	t.Parallel()

	root, err := findRepositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	document, err := loader.LoadFromFile(filepath.Join(root, "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	properties := []struct {
		schema   string
		property string
	}{
		{"ServerStatusDependency", "latency_ms"},
		{"ServerStatusDependency", "pool"},
		{"ServerStatusDependency", "postgresql_metrics"},
		{"ServerStatusDependency", "redis_metrics"},
		{"ServerStatusDependency", "history"},
		{"ApplicationListItem", "runtime_target"},
		{"ApplicationListItem", "runtime_status"},
		{"ApiEnvelope", "data"},
	}
	for _, property := range properties {
		t.Run(property.schema+"/"+property.property, func(t *testing.T) {
			schema := document.Components.Schemas[property.schema].Value.Properties[property.property].Value
			if err := schema.VisitJSON(nil, openapi3.EnableJSONSchema2020()); err != nil {
				t.Fatalf("nullable response property rejects null: %v", err)
			}
		})
	}
	latency := document.Components.Schemas["ServerStatusDependency"].Value.Properties["latency_ms"].Value
	for _, invalid := range []any{"unknown", float64(-1)} {
		if err := latency.VisitJSON(invalid, openapi3.EnableJSONSchema2020()); err == nil {
			t.Fatalf("latency accepts invalid value %#v", invalid)
		}
	}
}
