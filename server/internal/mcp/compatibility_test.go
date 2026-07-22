package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestRESTMCPCompatibilityMatrix keeps the Phase 2.5 gate executable: both
// transports reach one Gin route, so canonical payloads, denials and audit facts
// cannot silently diverge as more OpenAPI operations opt into MCP.
func TestRESTMCPCompatibilityMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	audits := 0
	engine.GET("/api/items/:id", func(ginCtx *gin.Context) {
		switch ginCtx.Param("id") {
		case "denied":
			ginCtx.JSON(http.StatusForbidden, gin.H{"code": "permission_denied", "message": "item access denied"})
		case "missing":
			ginCtx.JSON(http.StatusNotFound, gin.H{"code": "not_found", "message": "item not found"})
		default:
			audits++
			ginCtx.JSON(http.StatusOK, gin.H{"id": ginCtx.Param("id"), "result": "ok"})
		}
	})
	dispatcher, err := newDispatcher(engine)
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}
	definition := toolDefinition{name: "get_item", method: http.MethodGet, path: "/api/items/{id}", inputs: []inputBinding{{name: "id", location: "path", required: true}}, metadata: mcpMetadata{resourceURIParameterBindings: map[string]string{"id": "id"}}}

	for _, id := range []string{"item-7", "denied", "missing"} {
		t.Run(id, func(t *testing.T) { assertRESTToolParity(t, engine, dispatcher, definition, id) })
	}
	if audits != 2 {
		t.Fatalf("audit parity drift: successful REST and MCP reads must each emit one audit, got %d", audits)
	}

	assertResourceErrorParity(t, dispatcher, definition)
	assertResourceSuccessParity(t, dispatcher, definition)
}

func assertRESTToolParity(t *testing.T, engine *gin.Engine, dispatcher *dispatcher, definition toolDefinition, id string) {
	t.Helper()
	rest := httptest.NewRecorder()
	engine.ServeHTTP(rest, httptest.NewRequest(http.MethodGet, "/api/items/"+id, nil))
	mcpResult, err := dispatcher.dispatch(context.Background(), definition, map[string]any{"id": id})
	if err != nil {
		t.Fatalf("dispatch MCP: %v", err)
	}
	if got := toolResultText(mcpResult); got != rest.Body.String() {
		t.Fatalf("canonical JSON drift: MCP %s, REST %s", got, rest.Body.String())
	}
	if mcpResult.IsError != (rest.Code >= http.StatusMultipleChoices) {
		t.Fatalf("error mapping drift: MCP=%t REST=%d", mcpResult.IsError, rest.Code)
	}
}

func assertResourceErrorParity(t *testing.T, dispatcher *dispatcher, definition toolDefinition) {
	t.Helper()
	resource := resourceDefinition{uriTemplate: "graft://docker/containers/{id}", tool: definition}
	_, err := dispatcher.resourceHandler(resource)(context.Background(), &mcp.ReadResourceRequest{Params: &mcp.ReadResourceParams{URI: "graft://docker/containers/missing"}})
	var rpcError *jsonrpc.Error
	if !errors.As(err, &rpcError) || string(rpcError.Data) != `{"code":"not_found","message":"item not found"}` {
		t.Fatalf("resource error must retain canonical REST error: %#v", err)
	}
}

func assertResourceSuccessParity(t *testing.T, dispatcher *dispatcher, definition toolDefinition) {
	t.Helper()
	resource := resourceDefinition{uriTemplate: "graft://docker/containers/{id}", tool: definition}
	// JSON parsing is intentional evidence that transport framing, not payload shape,
	// is the only difference for a successful resource read.
	result, err := dispatcher.resourceHandler(resource)(context.Background(), &mcp.ReadResourceRequest{Params: &mcp.ReadResourceParams{URI: "graft://docker/containers/item-8"}})
	if err != nil {
		t.Fatalf("read projected resource: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil || payload["id"] != "item-8" {
		t.Fatalf("resource canonical payload = %q, %v", result.Contents[0].Text, err)
	}
}
