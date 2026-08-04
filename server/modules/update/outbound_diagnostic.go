package update

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"graft/server/internal/moduleapi"
)

const platformUpdateDiagnosticTargetName = "platform-update-release"

const platformUpdateConnectivityTargetID moduleapi.ConnectivityTargetID = "platform-update"

type platformUpdateOutboundNetworkConsumer struct{}

func (platformUpdateOutboundNetworkConsumer) Name() string { return "platform-update" }

func (platformUpdateOutboundNetworkConsumer) DisplayName() string {
	return "network.consumers.platformUpdate"
}

type platformUpdateDiagnosticTarget struct {
	factory moduleapi.OutboundHTTPClientFactory
}

func (t platformUpdateDiagnosticTarget) Name() string { return platformUpdateDiagnosticTargetName }

func (t platformUpdateDiagnosticTarget) DisplayName() string {
	return "network.diagnosticTargets.platformUpdate"
}

// ConnectivityRouteHost 返回用于出站策略解释的固定远端主机，不向报告暴露完整请求 URL。
func (platformUpdateDiagnosticTarget) ConnectivityRouteHost() string { return "api.github.com" }

// ConnectivityTargetDescriptor 声明平台更新的稳定连通性目标能力；后续持久化和 UI 只消费此声明，不解析更新模块内部 URL。
func (t platformUpdateDiagnosticTarget) ConnectivityTargetDescriptor() moduleapi.ConnectivityTargetDescriptor {
	return moduleapi.ConnectivityTargetDescriptor{
		ID:       platformUpdateConnectivityTargetID,
		ModuleID: "platform-update",
		Category: "platform",
		TitleKey: "network.diagnosticTargets.platformUpdate",
		Capabilities: moduleapi.ConnectivityTargetCapabilities{
			ProbeKinds: []moduleapi.ConnectivityProbeKind{
				moduleapi.ConnectivityProbeDNS,
				moduleapi.ConnectivityProbeTCP,
				moduleapi.ConnectivityProbeTLS,
				moduleapi.ConnectivityProbeCertificate,
				moduleapi.ConnectivityProbeHTTP,
			},
			Features: []moduleapi.ConnectivityTargetFeature{
				moduleapi.ConnectivityFeatureHistory,
				moduleapi.ConnectivityFeatureExport,
				moduleapi.ConnectivityFeatureProxyRoute,
			},
		},
	}
}

// RunConnectivityProbes 提供 Phase 1 的 HTTP 兼容报告；更细的 DNS/TCP/TLS 采集将在 probe pipeline 落地后替换该适配实现。
func (t platformUpdateDiagnosticTarget) RunConnectivityProbes(ctx context.Context) (moduleapi.ConnectivityReport, error) {
	result, err := t.ExecuteOutboundDiagnostic(ctx)
	if err != nil {
		return moduleapi.ConnectivityReport{}, err
	}
	status := moduleapi.ConnectivityReportStatusFailed
	probeStatus := moduleapi.ProbeStatusFailed
	if result.Connected {
		status = moduleapi.ConnectivityReportStatusHealthy
		probeStatus = moduleapi.ProbeStatusSucceeded
	}
	summary := "outbound request failed"
	if result.Connected {
		summary = "HTTP response succeeded"
	} else if result.Message != "" {
		summary = result.Message
	}
	var httpStatus *int
	if result.HTTPStatus >= http.StatusContinue && result.HTTPStatus <= 599 {
		httpStatus = &result.HTTPStatus
	}
	return moduleapi.NewConnectivityReport(platformUpdateConnectivityTargetID, status, result.TestedAt, result.Latency, []moduleapi.ProbeResult{{Kind: moduleapi.ConnectivityProbeHTTP, Status: probeStatus, Duration: result.Latency, HTTPStatus: httpStatus, Summary: summary}}, nil, nil), nil
}

func (t platformUpdateDiagnosticTarget) ExecuteOutboundDiagnostic(ctx context.Context) (moduleapi.OutboundDiagnosticResult, error) {
	if t.factory == nil {
		return moduleapi.OutboundDiagnosticResult{}, fmt.Errorf("outbound diagnostic client factory is unavailable")
	}
	client, err := t.factory.NewOutboundHTTPClient(ctx, moduleapi.WithTimeout(releaseHTTPTimeout))
	if err != nil {
		return moduleapi.OutboundDiagnosticResult{}, fmt.Errorf("create outbound diagnostic client: %w", err)
	}
	startedAt := time.Now()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+defaultReleaseRepository+"/releases", nil)
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
