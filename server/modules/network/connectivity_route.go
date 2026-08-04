package network

import (
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/http/httpproxy"

	"graft/server/internal/moduleapi"
)

// withRouteExplanation 添加管理员可读的策略决策，不暴露代理端点或规则。
func withRouteExplanation(report moduleapi.ConnectivityReport, policy moduleapi.OutboundNetworkPolicy, host string) moduleapi.ConnectivityReport {
	if report.Route != nil {
		return report.Snapshot()
	}
	route := resolveRouteExplanation(policy, host)
	return moduleapi.NewConnectivityReport(report.TargetID, report.Status, report.CheckedAt, report.TotalLatency, report.Probes, &route, report.ExitIP)
}

func resolveRouteExplanation(policy moduleapi.OutboundNetworkPolicy, host string) moduleapi.RouteExplanation {
	if !policy.Enabled || (policy.HTTPProxy == "" && policy.HTTPSProxy == "") {
		return moduleapi.RouteExplanation{MatchedStrategy: "Platform Default", Decision: "Direct", Reason: "Outbound proxy is disabled"}
	}
	host = strings.TrimSpace(host)
	requestURL := &url.URL{Scheme: "https", Host: host}
	config := httpproxy.Config{HTTPProxy: policy.HTTPProxy, HTTPSProxy: policy.HTTPSProxy, NoProxy: joinNoProxy(policy.NoProxy)}
	proxy, err := config.ProxyFunc()(requestURL)
	if err == nil && proxy != nil {
		return moduleapi.RouteExplanation{MatchedStrategy: "Platform Default", Decision: "HTTP Proxy", Reason: "Host is not matched by NO_PROXY"}
	}
	reason := "Matched NO_PROXY"
	if match := matchedNoProxyPattern(host, policy.NoProxy); match != "" {
		reason += " " + match
	} else if policy.HTTPSProxy == "" {
		reason = "No HTTPS proxy is configured"
	}
	return moduleapi.RouteExplanation{MatchedStrategy: "Direct", Decision: "Direct", Reason: reason}
}

func matchedNoProxyPattern(host string, patterns []string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(strings.ToLower(pattern))
		if pattern == "" {
			continue
		}
		if _, cidr, err := net.ParseCIDR(pattern); err == nil {
			if address := net.ParseIP(host); address != nil && cidr.Contains(address) {
				return pattern
			}
			continue
		}
		if pattern == "*" || host == strings.TrimPrefix(pattern, ".") || strings.HasSuffix(host, "."+strings.TrimPrefix(pattern, ".")) {
			return pattern
		}
	}
	return ""
}
