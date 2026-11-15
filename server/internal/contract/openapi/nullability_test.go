package openapi

import (
	"net/url"
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

func TestOpenAPI31NullableEnumsRetainNullAndEnumConstraints(t *testing.T) {
	t.Parallel()

	root, err := findRepositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		file     string
		property string
		allowed  string
	}{
		{"container-summary.yaml", "health", "healthy"},
		{"container-dashboard-top-item.yaml", "health", "healthy"},
		{"container-dashboard-anomaly-item.yaml", "health", "healthy"},
		{"container-dashboard-anomaly-item.yaml", "reason_code", "health.unhealthy"},
		{"application-file-tree-item.yaml", "tooltip_source", "application-note"},
		{"container-orchestrator-info.yaml", "group_scope_kind", "compose_project"},
		{"container-orchestrator-info.yaml", "member_scope_kind", "compose_service"},
		{"server-status-dependency-history.yaml", "unavailable_reason", "redis_not_configured"},
		{"server-status-redis-current-metrics.yaml", "replication_role", "master"},
		{"server-status-redis-current-metrics.yaml", "master_link_status", "up"},
	}
	document := &openapi3.T{
		OpenAPI:    "3.1.0",
		Components: &openapi3.Components{Schemas: make(openapi3.Schemas)},
	}
	for _, test := range cases {
		document.Components.Schemas[test.file] = &openapi3.SchemaRef{Ref: "./components/schemas/" + test.file}
	}
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	if err := loader.ResolveRefsIn(document, &url.URL{Path: filepath.Join(root, "openapi", "openapi.yaml")}); err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.file+"/"+test.property, func(t *testing.T) {
			schema := document.Components.Schemas[test.file].Value.Properties[test.property].Value
			for _, allowed := range []any{nil, test.allowed} {
				if err := schema.VisitJSON(allowed, openapi3.EnableJSONSchema2020()); err != nil {
					t.Errorf("nullable enum rejects allowed value %#v: %v", allowed, err)
				}
			}
			for _, invalid := range []any{"invalid-enum-value", true} {
				if err := schema.VisitJSON(invalid, openapi3.EnableJSONSchema2020()); err == nil {
					t.Errorf("nullable enum accepts invalid value %#v", invalid)
				}
			}
		})
	}
}
