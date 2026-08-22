package app

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/config"
	"graft/server/internal/logger"
)

func TestLogRealtimeGatewayConfigurationWarnsForRelativeDockerBuilderEnvFile(t *testing.T) {
	core, observed := observer.New(zap.WarnLevel)
	runtime := &Runtime{
		config:             &config.Config{DotenvPath: "./.data/docker-runtime-agent-dev/server.env"},
		canonicalAppLogger: logger.NewAppLogger(zap.New(core)),
	}

	runtime.logRealtimeGatewayConfiguration()

	entries := observed.FilterMessage("realtime gateway is using Docker Runtime Agent environment").All()
	if len(entries) != 1 {
		t.Fatalf("Docker Runtime Agent environment warnings = %d, want 1", len(entries))
	}
}

func TestLogRealtimeGatewayConfigurationDoesNotWarnForServerOwnedIntegrationEnvFile(t *testing.T) {
	core, observed := observer.New(zap.WarnLevel)
	runtime := &Runtime{
		config:             &config.Config{DotenvPath: "./server/.env.docker-runtime-agent"},
		canonicalAppLogger: logger.NewAppLogger(zap.New(core)),
	}

	runtime.logRealtimeGatewayConfiguration()

	entries := observed.FilterMessage("realtime gateway is using Docker Runtime Agent environment").All()
	if len(entries) != 0 {
		t.Fatalf("Docker Runtime Agent environment warnings = %d, want 0", len(entries))
	}
}
