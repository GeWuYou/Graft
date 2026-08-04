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
