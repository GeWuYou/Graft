package app

import (
	"context"
	"errors"
	"slices"
	"testing"

	"graft/server/internal/config"
	productmcp "graft/server/internal/mcp"
	"graft/server/internal/moduleapi"
)

func TestRegisterMCPRuntimeSkipsDisabledConfiguration(t *testing.T) {
	runtime := &Runtime{config: &config.Config{MCP: config.MCPConfig{Enabled: false}}}
	if err := runtime.registerMCPRuntime(testMCPAuthorizer{}); err != nil {
		t.Fatalf("disabled MCP runtime registration: %v", err)
	}
}

func TestOpenAPIDocsBundleCompilesCanonicalMCPReadTools(t *testing.T) {
	tools, err := productmcp.CompileReadTools(OpenAPIDocsBundle())
	if err != nil {
		t.Fatalf("compile embedded canonical OpenAPI bundle: %v", err)
	}
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name())
	}
	for _, expected := range []string{"get_application", "get_container", "get_runtime_target"} {
		if !slices.Contains(names, expected) {
			t.Fatalf("compiled tool names %v do not contain %q", names, expected)
		}
	}
}

func TestShutdownRuntimeClosesMCPAdapterBeforeHTTPServer(t *testing.T) {
	closer := &mcpRuntimeCloser{}
	runtime := &Runtime{mcpRuntime: closer}
	if err := runtime.shutdownRuntime(nil, nil); err != nil {
		t.Fatalf("shutdown runtime: %v", err)
	}
	if closer.calls != 1 || runtime.mcpRuntime != nil {
		t.Fatalf("MCP adapter lifecycle was not closed: calls=%d runtime=%#v", closer.calls, runtime.mcpRuntime)
	}
}

type mcpRuntimeCloser struct {
	calls int
}

func (c *mcpRuntimeCloser) Close() error {
	c.calls++
	return nil
}

var _ interface{ Close() error } = (*mcpRuntimeCloser)(nil)

func TestShutdownRuntimeReturnsMCPAdapterCloseError(t *testing.T) {
	runtime := &Runtime{mcpRuntime: mcpRuntimeCloseError{}}
	err := runtime.shutdownRuntime(nil, nil)
	if !errors.Is(err, errMCPRuntimeClose) {
		t.Fatalf("shutdown error = %v, want MCP close error", err)
	}
}

var errMCPRuntimeClose = errors.New("mcp runtime close")

type mcpRuntimeCloseError struct{}

func (mcpRuntimeCloseError) Close() error { return errMCPRuntimeClose }

type testMCPAuthorizer struct{}

func (testMCPAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, _ string) error {
	return nil
}

var _ moduleapi.Authorizer = testMCPAuthorizer{}
