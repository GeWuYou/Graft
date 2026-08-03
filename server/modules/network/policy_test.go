package network

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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
	userinfoProxy := "http://" + "placeholder-user" + ":" + "placeholder-value" + "@proxy.example:8080"
	for _, raw := range []string{
		`{"enabled":true,"http_proxy":"socks5://proxy.example:1080","https_proxy":"","no_proxy":[]}`,
		`{"enabled":true,"http_proxy":"` + userinfoProxy + `","https_proxy":"","no_proxy":[]}`,
		`{"enabled":true,"http_proxy":"http://proxy.example:8080/path","https_proxy":"","no_proxy":[]}`,
		`{"enabled":true,"http_proxy":"http://proxy.example:8080","https_proxy":"","no_proxy":[],"authentication":{}}`,
	} {
		if _, err := decodeOutboundPolicy(json.RawMessage(raw)); !errors.Is(err, errInvalidOutboundPolicy) {
			t.Fatalf("expected invalid policy for %s, got %v", raw, err)
		}
	}
}

func TestDecodeOutboundPolicyNormalizesEmptyNoProxyAsJSONArray(t *testing.T) {
	policy, err := decodeOutboundPolicy(json.RawMessage(`{"enabled":false,"http_proxy":"","https_proxy":"","no_proxy":[]}`))
	if err != nil {
		t.Fatalf("decode default outbound policy: %v", err)
	}
	if policy.NoProxy == nil || len(policy.NoProxy) != 0 {
		t.Fatalf("expected an empty no_proxy array, got %#v", policy.NoProxy)
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

func TestDecodeOutboundPolicyAcceptsCIDRNoProxyEntry(t *testing.T) {
	policy, err := decodeOutboundPolicy(json.RawMessage(`{"enabled":true,"http_proxy":"http://proxy.example:8080","https_proxy":"","no_proxy":["10.0.0.0/8"]}`))
	if err != nil {
		t.Fatalf("decode policy with CIDR no_proxy entry: %v", err)
	}
	if len(policy.NoProxy) != 1 || policy.NoProxy[0] != "10.0.0.0/8" {
		t.Fatalf("unexpected normalized no_proxy values: %#v", policy.NoProxy)
	}
}

func TestDecodeOutboundPolicyRejectsMalformedNoProxyEntries(t *testing.T) {
	for _, testCase := range []struct {
		entry    string
		expected string
	}{
		{entry: "example.com:invalid", expected: "no_proxy host-port is invalid"},
		{entry: "10.0.0.0/99", expected: "no_proxy CIDR is invalid"},
		{entry: "http://internal", expected: "no_proxy CIDR is invalid"},
		{entry: "*.bad*example", expected: "no_proxy wildcard is invalid"},
	} {
		raw := json.RawMessage(`{"enabled":true,"http_proxy":"http://proxy.example:8080","https_proxy":"","no_proxy":["` + testCase.entry + `"]}`)
		if _, err := decodeOutboundPolicy(raw); !errors.Is(err, errInvalidOutboundPolicy) || !strings.Contains(err.Error(), testCase.expected) {
			t.Fatalf("expected malformed no_proxy entry %q to fail with %q, got %v", testCase.entry, testCase.expected, err)
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

func TestHTTPClientFactoryDoesNotDependOnDefaultTransportType(t *testing.T) {
	original := http.DefaultTransport
	http.DefaultTransport = roundTripperStub{}
	t.Cleanup(func() { http.DefaultTransport = original })

	factory, err := NewHTTPClientFactory(outboundNetworkProviderStub{})
	if err != nil {
		t.Fatalf("new factory: %v", err)
	}
	client, err := factory.NewOutboundHTTPClient(context.Background())
	if err != nil {
		t.Fatalf("new client with custom default transport: %v", err)
	}
	if _, ok := client.Transport.(*http.Transport); !ok {
		t.Fatalf("expected factory-owned HTTP transport, got %T", client.Transport)
	}
}

func TestHTTPClientFactoryReusesTransportUntilPolicyChanges(t *testing.T) {
	provider := &mutableOutboundNetworkProvider{policy: moduleapi.OutboundNetworkPolicy{Enabled: true, HTTPProxy: "http://proxy.example:8080"}}
	factory, err := NewHTTPClientFactory(provider)
	if err != nil {
		t.Fatalf("new factory: %v", err)
	}
	first, err := factory.NewOutboundHTTPClient(context.Background())
	if err != nil {
		t.Fatalf("new first client: %v", err)
	}
	second, err := factory.NewOutboundHTTPClient(context.Background())
	if err != nil {
		t.Fatalf("new second client: %v", err)
	}
	if first.Transport != second.Transport {
		t.Fatal("expected unchanged policy to reuse its transport")
	}
	provider.policy.NoProxy = []string{"localhost"}
	replaced, err := factory.NewOutboundHTTPClient(context.Background())
	if err != nil {
		t.Fatalf("new replacement client: %v", err)
	}
	if first.Transport == replaced.Transport {
		t.Fatal("expected changed policy to replace its transport")
	}
}

type outboundNetworkProviderStub struct {
	policy moduleapi.OutboundNetworkPolicy
}

type mutableOutboundNetworkProvider struct {
	policy moduleapi.OutboundNetworkPolicy
}

type roundTripperStub struct{}

func (roundTripperStub) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("round trip is not expected in this test")
}

func (s *mutableOutboundNetworkProvider) CurrentOutboundNetworkPolicy(context.Context) (moduleapi.OutboundNetworkPolicy, error) {
	return s.policy, nil
}

func (s outboundNetworkProviderStub) CurrentOutboundNetworkPolicy(context.Context) (moduleapi.OutboundNetworkPolicy, error) {
	return s.policy, nil
}
