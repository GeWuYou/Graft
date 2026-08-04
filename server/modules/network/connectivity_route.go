package network

import "graft/server/internal/moduleapi"

// withRouteExplanation adds an administrator-facing policy decision without exposing proxy endpoints or rules.
func withRouteExplanation(report moduleapi.ConnectivityReport, policy moduleapi.OutboundNetworkPolicy) moduleapi.ConnectivityReport {
	if report.Route != nil {
		return report.Snapshot()
	}
	route := moduleapi.RouteExplanation{MatchedStrategy: "Platform Default", Decision: "Direct", Reason: "Outbound proxy is disabled"}
	if policy.Enabled && (policy.HTTPProxy != "" || policy.HTTPSProxy != "") {
		route = moduleapi.RouteExplanation{MatchedStrategy: "Platform Default", Decision: "HTTP Proxy", Reason: "Target host is not matched by NO_PROXY"}
	}
	return moduleapi.NewConnectivityReport(report.TargetID, report.Status, report.CheckedAt, report.TotalLatency, report.Probes, &route, report.ExitIP)
}
