package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

const (
	customConnectivityTargetPrefix = "custom-"
	customConnectivityTimeout      = 15 * time.Second
)

var errCustomConnectivityTargetNotFound = errors.New("custom connectivity target not found")

// CustomConnectivityTarget 是持久化的自定义 HTTP(S) 健康检查目标。Endpoint 仅能经由本文件的
// 校验和拨号器使用，避免普通 HTTP client 将管理员输入变成 SSRF 能力。
type CustomConnectivityTarget struct {
	ID          moduleapi.ConnectivityTargetID
	DisplayName string
	Endpoint    string
	Enabled     bool
	CreatedAt   time.Time
}

// CustomConnectivityTargetInput 是创建自定义目标的受限输入模型。
type CustomConnectivityTargetInput struct {
	TargetID    moduleapi.ConnectivityTargetID
	DisplayName string
	Endpoint    string
}

// CustomConnectivityTargetStore 是自定义目标持久化边界；检查报告仍由 ConnectivityStore 持有。
type CustomConnectivityTargetStore interface {
	CreateCustomTarget(context.Context, CustomConnectivityTarget, uint64) (CustomConnectivityTarget, error)
	ListCustomTargets(context.Context) ([]CustomConnectivityTarget, error)
	CustomTarget(context.Context, moduleapi.ConnectivityTargetID) (CustomConnectivityTarget, error)
	DeleteCustomTarget(context.Context, moduleapi.ConnectivityTargetID, uint64) error
}

// ConnectivityTargetDescriptor 返回 capability 驱动的自定义 HTTP 目标描述。
func (t CustomConnectivityTarget) ConnectivityTargetDescriptor() moduleapi.ConnectivityTargetDescriptor {
	return moduleapi.ConnectivityTargetDescriptor{
		ID:       t.ID,
		ModuleID: moduleID,
		Category: "general",
		TitleKey: t.DisplayName,
		Capabilities: moduleapi.ConnectivityTargetCapabilities{
			ProbeKinds: []moduleapi.ConnectivityProbeKind{moduleapi.ConnectivityProbeDNS, moduleapi.ConnectivityProbeTCP, moduleapi.ConnectivityProbeTLS, moduleapi.ConnectivityProbeHTTP},
			Features:   []moduleapi.ConnectivityTargetFeature{moduleapi.ConnectivityFeatureHistory, moduleapi.ConnectivityFeatureExport, moduleapi.ConnectivityFeatureProxyRoute},
		},
	}
}

// RunConnectivityProbes 满足 registry target 契约。运行时仍应经由 Service 调用，确保策略可用性先被检查。
func (t CustomConnectivityTarget) RunConnectivityProbes(ctx context.Context) (moduleapi.ConnectivityReport, error) {
	return t.RunCustomConnectivityProbes(ctx)
}

// RunCustomConnectivityProbes 在每次连接前重新解析并只使用校验后的 IP 拨号，同时禁用重定向，防止 DNS
// rebinding 和重定向进入私有网络。
func (t CustomConnectivityTarget) RunCustomConnectivityProbes(ctx context.Context) (moduleapi.ConnectivityReport, error) {
	started := time.Now()
	endpoint, err := validateCustomConnectivityEndpoint(t.Endpoint)
	if err != nil {
		return moduleapi.ConnectivityReport{}, err
	}
	resolver := net.DefaultResolver
	addresses, err := resolvePublicEndpoint(ctx, resolver, endpoint.Hostname())
	if err != nil {
		return moduleapi.ConnectivityReport{}, err
	}
	dnsDuration := time.Since(started)
	client := newCustomConnectivityHTTPClient(endpoint, resolver)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return moduleapi.ConnectivityReport{}, fmt.Errorf("build custom connectivity request: %w", err)
	}
	response, requestErr := client.Do(request)
	total := time.Since(started)
	probes := []moduleapi.ProbeResult{{Kind: moduleapi.ConnectivityProbeDNS, Status: moduleapi.ProbeStatusSucceeded, Duration: dnsDuration, Summary: fmt.Sprintf("%d public address(es) validated", len(addresses)), OccurredAt: started.Add(dnsDuration)}}
	status := moduleapi.ConnectivityReportStatusHealthy
	if requestErr != nil {
		status = moduleapi.ConnectivityReportStatusFailed
		probes = append(probes, moduleapi.ProbeResult{Kind: moduleapi.ConnectivityProbeHTTP, Status: moduleapi.ProbeStatusFailed, Duration: total - dnsDuration, ErrorCode: "request_failed", Summary: "HTTP request could not complete", OccurredAt: time.Now()})
	} else {
		defer func() { _ = response.Body.Close() }()
		probeStatus := moduleapi.ProbeStatusSucceeded
		if response.StatusCode >= http.StatusBadRequest {
			status, probeStatus = moduleapi.ConnectivityReportStatusDegraded, moduleapi.ProbeStatusFailed
		}
		probes = append(probes, moduleapi.ProbeResult{Kind: moduleapi.ConnectivityProbeHTTP, Status: probeStatus, Duration: total - dnsDuration, HTTPStatus: &response.StatusCode, Summary: "HTTP " + strconv.Itoa(response.StatusCode), OccurredAt: time.Now()})
	}
	route := moduleapi.RouteExplanation{MatchedStrategy: "Direct", Decision: "Direct", Reason: "Custom targets use validated direct dialing; proxy execution is intentionally disabled"}
	return moduleapi.NewConnectivityReport(t.ID, status, time.Now(), total, probes, &route, nil), nil
}

func validateCustomConnectivityTargetInput(input CustomConnectivityTargetInput) (CustomConnectivityTarget, error) {
	targetID := moduleapi.ConnectivityTargetID(strings.TrimSpace(string(input.TargetID)))
	displayName := strings.TrimSpace(input.DisplayName)
	if !strings.HasPrefix(string(targetID), customConnectivityTargetPrefix) || len(targetID) > 128 || displayName == "" || len(displayName) > 128 {
		return CustomConnectivityTarget{}, errors.New("custom connectivity target is invalid")
	}
	endpoint, err := validateCustomConnectivityEndpoint(input.Endpoint)
	if err != nil {
		return CustomConnectivityTarget{}, err
	}
	return CustomConnectivityTarget{ID: targetID, DisplayName: displayName, Endpoint: endpoint.String(), Enabled: true}, nil
}

//nolint:gocyclo,cyclop // 每个拒绝条件对应独立 SSRF 信任边界，合并会降低审计可读性。
func validateCustomConnectivityEndpoint(raw string) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || endpoint == nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, errors.New("custom connectivity endpoint is invalid")
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return nil, errors.New("custom connectivity endpoint must use HTTP or HTTPS")
	}
	if endpoint.User != nil || endpoint.Fragment != "" || endpoint.RawQuery != "" || endpoint.Hostname() == "" {
		return nil, errors.New("custom connectivity endpoint contains unsupported authority or URL parts")
	}
	port := endpoint.Port()
	if port != "" && port != "80" && port != "443" {
		return nil, errors.New("custom connectivity endpoint port is not allowed")
	}
	host := strings.TrimSuffix(strings.ToLower(endpoint.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.Contains(host, "%") {
		return nil, errors.New("custom connectivity endpoint host is not public")
	}
	if address, err := netip.ParseAddr(host); err == nil && !isPublicConnectivityAddress(address) {
		return nil, errors.New("custom connectivity endpoint address is not public")
	}
	endpoint.Host = endpoint.Hostname()
	if port != "" {
		endpoint.Host = net.JoinHostPort(endpoint.Hostname(), port)
	}
	return endpoint, nil
}

func resolvePublicEndpoint(ctx context.Context, resolver *net.Resolver, host string) ([]netip.Addr, error) {
	if address, err := netip.ParseAddr(host); err == nil {
		if !isPublicConnectivityAddress(address) {
			return nil, errors.New("custom connectivity endpoint address is not public")
		}
		return []netip.Addr{address}, nil
	}
	addresses, err := resolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return nil, errors.New("custom connectivity endpoint cannot resolve a public address")
	}
	for _, address := range addresses {
		if !isPublicConnectivityAddress(address) {
			return nil, errors.New("custom connectivity endpoint resolved a non-public address")
		}
	}
	return addresses, nil
}

func newCustomConnectivityHTTPClient(endpoint *url.URL, resolver *net.Resolver) *http.Client {
	expectedHost := endpoint.Hostname()
	transport := newDefaultTransport()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || !strings.EqualFold(strings.TrimSuffix(host, "."), strings.TrimSuffix(expectedHost, ".")) {
			return nil, errors.New("custom connectivity dial target is not allowed")
		}
		if port != "80" && port != "443" {
			return nil, errors.New("custom connectivity dial port is not allowed")
		}
		addresses, err := resolvePublicEndpoint(ctx, resolver, expectedHost)
		if err != nil {
			return nil, err
		}
		// 拨号器只接收已校验的字面 IP，绝不将可变化的 hostname 交给底层网络调用。
		return (&net.Dialer{Timeout: defaultDialTimeout, KeepAlive: defaultDialKeepAlive}).DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
	}
	return &http.Client{Transport: transport, Timeout: customConnectivityTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func isPublicConnectivityAddress(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	for _, blocked := range []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001:db8::/32"),
	} {
		if blocked.Contains(address) {
			return false
		}
	}
	return true
}
