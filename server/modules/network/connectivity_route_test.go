package network

import (
	"testing"

	"graft/server/internal/moduleapi"
)

func TestResolveRouteExplanationUsesProxyAndNoProxySemantics(t *testing.T) {
	policy := moduleapi.OutboundNetworkPolicy{Enabled: true, HTTPSProxy: "http://proxy.company.test:8080", NoProxy: []string{".internal.test", "192.168.0.0/16"}}
	proxied := resolveRouteExplanation(policy, "github.com")
	if proxied.Decision != "HTTP Proxy" || proxied.Reason != "Host is not matched by NO_PROXY" {
		t.Fatalf("unexpected proxied route: %#v", proxied)
	}
	direct := resolveRouteExplanation(policy, "api.internal.test")
	if direct.Decision != "Direct" || direct.Reason != "Matched NO_PROXY .internal.test" {
		t.Fatalf("unexpected direct route: %#v", direct)
	}
}

func TestResolveRouteExplanationCoversDisabledHTTPOnlyAndNoProxyMatches(t *testing.T) {
	tests := []struct {
		name   string
		policy moduleapi.OutboundNetworkPolicy
		host   string
		reason string
	}{
		{name: "disabled", host: "github.com", reason: "Outbound proxy is disabled"},
		{name: "http only", policy: moduleapi.OutboundNetworkPolicy{Enabled: true, HTTPProxy: "http://proxy.test:8080"}, host: "github.com", reason: "No HTTPS proxy is configured"},
		{name: "host port", policy: moduleapi.OutboundNetworkPolicy{Enabled: true, HTTPSProxy: "http://proxy.test:8080", NoProxy: []string{".internal.test"}}, host: "api.internal.test:443", reason: "Matched NO_PROXY .internal.test"},
		{name: "cidr", policy: moduleapi.OutboundNetworkPolicy{Enabled: true, HTTPSProxy: "http://proxy.test:8080", NoProxy: []string{"192.168.0.0/16"}}, host: "192.168.10.4:443", reason: "Matched NO_PROXY 192.168.0.0/16"},
		{name: "wildcard", policy: moduleapi.OutboundNetworkPolicy{Enabled: true, HTTPSProxy: "http://proxy.test:8080", NoProxy: []string{"*"}}, host: "github.com", reason: "Matched NO_PROXY *"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			route := resolveRouteExplanation(test.policy, test.host)
			if route.Decision != "Direct" || route.Reason != test.reason {
				t.Fatalf("unexpected route: %#v", route)
			}
		})
	}
}
