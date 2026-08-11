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
		config:             &config.Config{DotenvPath: "./.data/docker-builder-agent-dev/server.env"},
		canonicalAppLogger: logger.NewAppLogger(zap.New(core)),
	}

	runtime.logRealtimeGatewayConfiguration()

	entries := observed.FilterMessage("realtime gateway is using Docker Builder Agent environment").All()
	if len(entries) != 1 {
		t.Fatalf("Docker Builder Agent environment warnings = %d, want 1", len(entries))
	}
}
