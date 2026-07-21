package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/moduleapi"
)

func TestCompileReadToolsUsesOptedInReadOperationsOnly(t *testing.T) {
	bundle := compilerTestBundle(map[string]any{
		"/api/items/{id}": map[string]any{
			"get": compilerTestOperation("getItem", compilerTestMetadata("low", false), true),
		},
		"/api/items/{id}/restart": map[string]any{
			"post": compilerTestOperation("postItemRestart", compilerTestMetadata("high", true), false),
		},
	})

	tools, err := CompileReadTools(bundle)
	if err != nil {
		t.Fatalf("compile read tools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("compiled tool count = %d, want 1", len(tools))
	}
	tool := tools[0]
	if tool.name != "get_item" || tool.method != http.MethodGet || tool.path != "/api/items/{id}" {
		t.Fatalf("unexpected compiled tool: %#v", tool)
	}
	properties, ok := tool.inputSchema["properties"].(map[string]any)
	if !ok || properties["id"] == nil || properties["filter"] == nil || properties[inputBodyName] == nil {
		t.Fatalf("input schema must derive path, query, and body inputs: %#v", tool.inputSchema)
	}
	required, ok := tool.inputSchema["required"].([]string)
	if !ok || len(required) != 2 || required[0] != inputBodyName || required[1] != "id" {
		t.Fatalf("unexpected required input schema fields: %#v", tool.inputSchema["required"])
	}
}

func TestCompileReadToolsRejectsUnsafeOrAmbiguousMetadata(t *testing.T) {
	tests := []struct {
		name   string
		paths  map[string]any
		needle string
	}{
		{
			name: "missing operation ID",
			paths: map[string]any{
				"/api/items/{id}": map[string]any{"get": compilerTestOperation("", compilerTestMetadata("low", false), false)},
			},
			needle: "without operationId",
		},
		{
			name: "name collision",
			paths: map[string]any{
				"/api/items/{id}":      map[string]any{"get": compilerTestOperation("getItem", compilerTestMetadata("low", false), false)},
				"/api/items/{id}/copy": map[string]any{"get": compilerTestOperation("get_item", compilerTestMetadata("low", false), false)},
			},
			needle: "collides",
		},
		{
			name: "unbound resource placeholder",
			paths: map[string]any{
				"/api/items/{id}": map[string]any{"get": compilerTestOperation("getItem", compilerTestMetadataWithBinding("missing"), false)},
			},
			needle: "declared path parameter",
		},
		{
			name: "unsafe read risk",
			paths: map[string]any{
				"/api/items/{id}": map[string]any{"get": compilerTestOperation("getItem", compilerTestMetadata("high", true), false)},
			},
			needle: "not safe to compile",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := CompileReadTools(compilerTestBundle(testCase.paths))
			if err == nil || !strings.Contains(err.Error(), testCase.needle) {
				t.Fatalf("compile error = %v, want %q", err, testCase.needle)
			}
		})
	}
}

func TestPositiveISODurationRejectsIncompleteAndOutOfOrderValues(t *testing.T) {
	for _, value := range []string{"P1D", "PT5M", "P1DT2H"} {
		if !positiveISODuration(value) {
			t.Fatalf("positiveISODuration(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"P", "PT", "P1DT", "P1H", "PT1Y", "P1M2Y", "PT1M2H", "P1W1D", "PT0S"} {
		if positiveISODuration(value) {
			t.Fatalf("positiveISODuration(%q) = true, want false", value)
		}
	}
}

func TestDispatcherInvokesExistingGinRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/api/items/:id", func(ginCtx *gin.Context) {
		caller, ok := moduleapi.PersonalAccessTokenCallerFromContext(ginCtx.Request.Context())
		if !ok || caller.User.ID != 7 {
			t.Fatalf("expected personal token caller to reach REST route, got %#v", caller)
		}
		requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
		if !ok || requestAuth.User == nil || requestAuth.User.ID != 7 {
			t.Fatalf("expected request auth context to reach REST route, got %#v", requestAuth)
		}
		ginCtx.JSON(http.StatusOK, gin.H{"id": ginCtx.Param("id"), "filter": ginCtx.Query("filter")})
	})
	dispatcher, err := newDispatcher(engine)
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	caller := moduleapi.PersonalAccessTokenCaller{
		TokenID:   11,
		User:      moduleapi.CurrentUser{ID: 7},
		Scopes:    []string{"container.read"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	ctx := moduleapi.WithPersonalAccessTokenCaller(context.Background(), caller)
	ctx = moduleapi.WithRequestAuthContext(ctx, moduleapi.RequestAuthContext{User: &caller.User})
	result, err := dispatcher.dispatch(ctx, toolDefinition{
		name:   "get_item",
		method: http.MethodGet,
		path:   "/api/items/{id}",
		inputs: []inputBinding{
			{name: "id", location: "path", required: true},
			{name: "filter", location: "query"},
		},
	}, map[string]any{"id": "item-7", "filter": "ready"})
	if err != nil {
		t.Fatalf("dispatch tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("dispatch result is error: %#v", result)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok || structured["id"] != "item-7" || structured["filter"] != "ready" {
		t.Fatalf("unexpected REST response projection: %#v", result.StructuredContent)
	}
}

func compilerTestBundle(paths map[string]any) []byte {
	bundle := map[string]any{
		"openapi": "3.1.0",
		"info":    map[string]any{"title": "MCP compiler test", "version": "1.0.0"},
		"x-graft-mcp-schema": map[string]any{
			"type":       "object",
			"required":   []string{"risk", "confirmation"},
			"properties": map[string]any{"risk": map[string]any{}, "confirmation": map[string]any{}},
		},
		"paths": paths,
	}
	encoded, err := json.Marshal(bundle)
	if err != nil {
		panic(err)
	}
	return encoded
}

func compilerTestOperation(operationID string, metadata map[string]any, includeBody bool) map[string]any {
	operation := map[string]any{
		"operationId": operationID,
		"description": "Read one item.",
		"x-graft-mcp": metadata,
		"parameters": []any{
			map[string]any{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}},
			map[string]any{"name": "filter", "in": "query", "schema": map[string]any{"type": "string"}},
		},
		"responses": map[string]any{"200": map[string]any{"description": "ok"}},
	}
	if includeBody {
		operation["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"type": "object", "properties": map[string]any{"locale": map[string]any{"type": "string"}}},
				},
			},
		}
	}
	return operation
}

func compilerTestMetadata(risk string, confirmationRequired bool) map[string]any {
	strategy, ttl := "none", "PT0S"
	if confirmationRequired {
		strategy, ttl = "two_phase", "PT5M"
	}
	return map[string]any{
		"resource_uri_templates":          []string{"graft://docker/containers/{id}"},
		"resource_uri_parameter_bindings": map[string]any{"id": "id"},
		"risk":                            map[string]any{"level": risk, "reason": "test", "reversible": risk == "low", "impact": "test"},
		"confirmation":                    map[string]any{"required": confirmationRequired, "strategy": strategy, "ttl": ttl},
	}
}

func compilerTestMetadataWithBinding(binding string) map[string]any {
	metadata := compilerTestMetadata("low", false)
	metadata["resource_uri_parameter_bindings"] = map[string]any{"id": binding}
	return metadata
}
