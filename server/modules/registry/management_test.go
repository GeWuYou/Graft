package registry

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"

	registrystore "graft/server/modules/registry/store"
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
	if _, err := resolvePublicRegistryEndpoint(context.Background(), registryResolverStub{err: errors.New("lookup unavailable")}, "registry.example"); err == nil || err.Error() != "dns_failed" {
		t.Fatalf("DNS lookup error = %v, want dns_failed", err)
	}
}

//nolint:gocognit // 表驱动用例同时保护连接、Repository 和错误码归一化边界。
func TestRegistryManagementNormalization(t *testing.T) {
	credentialRef := "credential:primary" // #nosec G101 -- 测试引用标识，不包含凭据。
	for _, testCase := range []struct {
		name      string
		input     registrystore.ConnectionInput
		wantError bool
		wantMode  string
	}{
		{name: "anonymous endpoint is normalized", input: registrystore.ConnectionInput{ConnectionRef: "primary", DisplayName: "Primary", Provider: registryProviderGenericOCI, Endpoint: "https://REGISTRY.example/v2/"}, wantMode: registrystore.AuthModeAnonymous},
		{name: "credential mode requires reference", input: registrystore.ConnectionInput{ConnectionRef: "primary", DisplayName: "Primary", Provider: registryProviderGenericOCI, Endpoint: "https://registry.example", AuthMode: registrystore.AuthModeCredentialRef}, wantError: true},
		{name: "credential mode is inferred", input: registrystore.ConnectionInput{ConnectionRef: "primary", DisplayName: "Primary", Provider: registryProviderGenericOCI, Endpoint: "https://registry.example", CredentialRef: credentialRef}, wantMode: registrystore.AuthModeCredentialRef},
		{name: "rune sized description is rejected", input: registrystore.ConnectionInput{ConnectionRef: "primary", DisplayName: "Primary", Provider: registryProviderGenericOCI, Endpoint: "https://registry.example", Description: strings.Repeat("界", 501)}, wantError: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := normalizeConnectionInput(testCase.input)
			if (err != nil) != testCase.wantError {
				t.Fatalf("normalizeConnectionInput() error = %v, wantError %v", err, testCase.wantError)
			}
			if err == nil && actual.AuthMode != testCase.wantMode {
				t.Fatalf("auth mode = %q, want %q", actual.AuthMode, testCase.wantMode)
			}
			if err == nil && actual.Endpoint != "https://registry.example/v2" && testCase.name == "anonymous endpoint is normalized" {
				t.Fatalf("endpoint = %q", actual.Endpoint)
			}
		})
	}

	for _, repositoryRef := range []string{"team/api", "team/api-v2", "team/api__v2"} {
		if _, err := normalizeRepositoryInput(registrystore.RepositoryInput{RepositoryRef: repositoryRef, DisplayName: "Repository"}); err != nil {
			t.Fatalf("normalizeRepositoryInput(%q): %v", repositoryRef, err)
		}
	}
	for _, repositoryRef := range []string{"Team/api", "team/api-", ".hidden", "team/my repo"} {
		if _, err := normalizeRepositoryInput(registrystore.RepositoryInput{RepositoryRef: repositoryRef, DisplayName: "Repository"}); err == nil {
			t.Fatalf("normalizeRepositoryInput(%q) unexpectedly succeeded", repositoryRef)
		}
	}

	if got := verificationErrorCode(errors.New("endpoint detail must not escape")); got != "verification_failed" {
		t.Fatalf("verification error code = %q", got)
	}
}

func TestRegistryManagementListErrorsIncludeOperationContext(t *testing.T) {
	var service *Service
	testCases := []struct {
		name string
		call func() error
		want string
	}{
		{name: "repositories", call: func() error { _, _, err := service.ListRepositories(context.Background(), "primary", 1, 0); return err }, want: "list registry artifact repositories"},
		{name: "assignments", call: func() error {
			_, _, err := service.ListAssignments(context.Background(), "primary", "team/api", 1, 0)
			return err
		}, want: "list registry artifact repository assignments"},
		{name: "destinations", call: func() error {
			_, _, err := service.ListAvailableDestinations(context.Background(), 1, 1, 0)
			return err
		}, want: "list available registry destinations"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.call()
			if err == nil || !strings.Contains(err.Error(), testCase.want) || !strings.Contains(err.Error(), "registry management service is unavailable") {
				t.Fatalf("list error = %v, want operation context %q with preserved cause", err, testCase.want)
			}
		})
	}
}

type registryResolverStub struct {
	addresses []netip.Addr
	err       error
}

func (s registryResolverStub) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return append([]netip.Addr(nil), s.addresses...), s.err
}
