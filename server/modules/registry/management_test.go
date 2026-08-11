package registry

import (
	"context"
	"net/netip"
	"testing"
)

func TestRegistryVerificationEndpointRejectsSchemeMismatch(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		endpoint string
		insecure bool
	}{
		{name: "secure connection cannot use HTTP", endpoint: "http://registry.example", insecure: false},
		{name: "insecure connection cannot use HTTPS", endpoint: "https://registry.example", insecure: true},
		{name: "embedded credentials are rejected", endpoint: "https://user:password@registry.example", insecure: false}, // #nosec G101 -- 测试断言 URL userinfo 会被拒绝；不是真实凭据。
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := registryVerificationEndpoint(testCase.endpoint, testCase.insecure); err == nil {
				t.Fatalf("registryVerificationEndpoint(%q, %t) unexpectedly succeeded", testCase.endpoint, testCase.insecure)
			}
		})
	}
}

func TestResolvePublicRegistryEndpointRejectsPrivateAndMixedDNSResults(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		addresses []netip.Addr
		wantError string
	}{
		{name: "private address", addresses: []netip.Addr{netip.MustParseAddr("10.0.0.10")}, wantError: "network_denied"},
		{name: "mixed addresses", addresses: []netip.Addr{netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("127.0.0.1")}, wantError: "network_denied"},
		{name: "public address", addresses: []netip.Addr{netip.MustParseAddr("93.184.216.34")}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := resolvePublicRegistryEndpoint(context.Background(), registryResolverStub{addresses: testCase.addresses}, "registry.example")
			if testCase.wantError == "" && err != nil {
				t.Fatalf("resolve public endpoint: %v", err)
			}
			if testCase.wantError != "" && (err == nil || err.Error() != testCase.wantError) {
				t.Fatalf("error = %v, want %q", err, testCase.wantError)
			}
		})
	}
}

type registryResolverStub struct{ addresses []netip.Addr }

func (s registryResolverStub) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return append([]netip.Addr(nil), s.addresses...), nil
}
