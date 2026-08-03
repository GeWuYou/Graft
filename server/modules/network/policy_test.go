package network

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"graft/server/internal/configregistry"
	"graft/server/internal/moduleapi"
)

func TestRegisterOutboundConfigDefinesModuleManagedRuntimeHotPolicy(t *testing.T) {
	registry := configregistry.NewRegistry()
	if err := registerOutboundConfig(registry); err != nil {
		t.Fatalf("register outbound config: %v", err)
	}
	definition, ok := registry.Get(outboundConfigKey)
	if !ok || !definition.ModuleManaged || definition.RuntimeApplyMode != configregistry.RuntimeApplyModeRuntimeHot {
		t.Fatalf("unexpected outbound config definition: %#v", definition)
	}
	if _, err := decodeOutboundPolicy(definition.DefaultValue); err != nil {
		t.Fatalf("default outbound policy must be decodable: %v", err)
	}
}

func TestDecodeOutboundPolicyRejectsUnsupportedProxyForms(t *testing.T) {
	for _, raw := range []string{
		`{"enabled":true,"http_proxy":"socks5://proxy.example:1080","https_proxy":"","no_proxy":[]}`,
		`{"enabled":true,"http_proxy":"http://user:secret@proxy.example:8080","https_proxy":"","no_proxy":[]}`,
		`{"enabled":true,"http_proxy":"http://proxy.example:8080/path","https_proxy":"","no_proxy":[]}`,
		`{"enabled":true,"http_proxy":"http://proxy.example:8080","https_proxy":"","no_proxy":[],"authentication":{}}`,
	} {
		if _, err := decodeOutboundPolicy(json.RawMessage(raw)); !errors.Is(err, errInvalidOutboundPolicy) {
			t.Fatalf("expected invalid policy for %s, got %v", raw, err)
		}
	}
}

func TestProxyFuncUsesGoNoProxyMatching(t *testing.T) {
	policy := moduleapi.OutboundNetworkPolicy{Enabled: true, HTTPProxy: "http://proxy.example:8080", HTTPSProxy: "http://proxy.example:8080", NoProxy: []string{"localhost", ".internal", "10.0.0.0/8", "example.com:8443"}}
	proxy := proxyFunc(policy)
	for _, rawURL := range []string{"http://localhost/service", "https://api.internal/v1", "http://10.1.2.3/health", "https://example.com:8443/health"} {
		request, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		got, err := proxy(request)
		if err != nil || got != nil {
			t.Fatalf("expected no proxy for %s, got %v, %v", rawURL, got, err)
		}
	}
	request, _ := http.NewRequest(http.MethodGet, "https://api.example.com/v1", nil)
	got, err := proxy(request)
	if err != nil || got == nil || got.String() != "http://proxy.example:8080" {
		t.Fatalf("expected configured proxy, got %v, %v", got, err)
	}
}

func TestDecodeOutboundPolicyRejectsMalformedNoProxyEntries(t *testing.T) {
	for _, entry := range []string{"example.com:invalid", "10.0.0.0/99", "http://internal", "*.bad*example"} {
		raw := json.RawMessage(`{"enabled":true,"http_proxy":"http://proxy.example:8080","https_proxy":"","no_proxy":["` + entry + `"]}`)
		if _, err := decodeOutboundPolicy(raw); !errors.Is(err, errInvalidOutboundPolicy) {
			t.Fatalf("expected malformed no_proxy entry %q to fail, got %v", entry, err)
		}
	}
}

func TestDecodeOutboundPolicyRejectsInvalidProxyPort(t *testing.T) {
	if _, err := decodeOutboundPolicy(json.RawMessage(`{"enabled":true,"http_proxy":"http://proxy.example:70000","https_proxy":"","no_proxy":[]}`)); !errors.Is(err, errInvalidOutboundPolicy) {
		t.Fatalf("expected invalid proxy port to fail, got %v", err)
	}
}

func TestHTTPClientFactoryDoesNotUseEnvironmentProxy(t *testing.T) {
	provider := outboundNetworkProviderStub{policy: moduleapi.OutboundNetworkPolicy{Enabled: false, HTTPProxy: "http://proxy.example:8080"}}
	factory, err := NewHTTPClientFactory(provider)
	if err != nil {
		t.Fatalf("new factory: %v", err)
	}
	client, err := factory.NewOutboundHTTPClient(context.Background(), moduleapi.WithTimeout(time.Second))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil || client.Timeout != time.Second {
		t.Fatalf("expected direct cloned transport and selected timeout, got %#v", client)
	}
}

type outboundNetworkProviderStub struct {
	policy moduleapi.OutboundNetworkPolicy
}

func (s outboundNetworkProviderStub) CurrentOutboundNetworkPolicy(context.Context) (moduleapi.OutboundNetworkPolicy, error) {
	return s.policy, nil
}
