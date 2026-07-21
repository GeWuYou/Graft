package app

import (
	"context"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
)

func TestRegisterMCPRuntimeSkipsDisabledConfiguration(t *testing.T) {
	runtime := &Runtime{config: &config.Config{MCP: config.MCPConfig{Enabled: false}}}
	if err := runtime.registerMCPRuntime(testMCPAuthorizer{}); err != nil {
		t.Fatalf("disabled MCP runtime registration: %v", err)
	}
}

type testMCPAuthorizer struct{}

func (testMCPAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, _ string) error {
	return nil
}

var _ moduleapi.Authorizer = testMCPAuthorizer{}
