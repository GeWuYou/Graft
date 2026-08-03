package update

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"graft/server/internal/moduleapi"
)

const platformUpdateDiagnosticTargetName = "platform-update-release"

type platformUpdateDiagnosticTarget struct {
	factory moduleapi.OutboundHTTPClientFactory
}

func (t platformUpdateDiagnosticTarget) Name() string { return platformUpdateDiagnosticTargetName }

func (t platformUpdateDiagnosticTarget) DisplayName() string { return "network.diagnosticTargets.platformUpdate" }

func (t platformUpdateDiagnosticTarget) ExecuteOutboundDiagnostic(ctx context.Context) (moduleapi.OutboundDiagnosticResult, error) {
	if t.factory == nil {
		return moduleapi.OutboundDiagnosticResult{}, fmt.Errorf("outbound diagnostic client factory is unavailable")
	}
	client, err := t.factory.NewOutboundHTTPClient(ctx, moduleapi.WithTimeout(releaseHTTPTimeout))
	if err != nil {
		return moduleapi.OutboundDiagnosticResult{}, fmt.Errorf("create outbound diagnostic client: %w", err)
	}
	startedAt := time.Now()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/GeWuYou/Graft/releases", nil)
	if err != nil {
		return moduleapi.OutboundDiagnosticResult{}, fmt.Errorf("create outbound diagnostic request: %w", err)
	}
	response, err := client.Do(request)
	latency := time.Since(startedAt)
	result := moduleapi.OutboundDiagnosticResult{Latency: latency, TestedAt: time.Now().UTC()}
	if err != nil {
		result.Message = "outbound request failed"
		return result, nil
	}
	defer func() { _ = response.Body.Close() }()
	result.HTTPStatus = response.StatusCode
	result.Connected = response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
	if !result.Connected {
		result.Message = "remote service returned an unexpected response"
	}
	return result, nil
}
