package runtimetarget

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type sequentialVerificationTransport struct {
	responses []*http.Response
	requests  []*http.Request
}

func (t *sequentialVerificationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.requests = append(t.requests, request.Clone(request.Context()))
	response := t.responses[0]
	t.responses = t.responses[1:]
	return response, nil
}

func TestProbeOCIRegistryV2SeparatesChallengeFromAuthenticationSuccess(t *testing.T) {
	transport := &sequentialVerificationTransport{responses: []*http.Response{{StatusCode: http.StatusUnauthorized, Header: make(http.Header), Body: http.NoBody}, {StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody}}}
	previous := ociRegistryVerificationHTTPClient
	ociRegistryVerificationHTTPClient = &http.Client{Transport: transport}
	t.Cleanup(func() { ociRegistryVerificationHTTPClient = previous })
	endpoint, err := url.Parse("https://registry.example/v2/")
	if err != nil {
		t.Fatal(err)
	}

	result := probeOCIRegistryV2(context.Background(), endpoint, "Basic dXNlcjpwYXNz")
	if !result.Reachable || !result.ProtocolCompatible || !result.AuthenticationChallenged || !result.AuthenticationSucceeded || result.ProviderScopeConforms {
		t.Fatalf("verification result = %#v", result)
	}
	if len(transport.requests) != 2 || transport.requests[0].Header.Get("Authorization") != "" || transport.requests[1].Header.Get("Authorization") != "Basic dXNlcjpwYXNz" {
		t.Fatalf("verification requests = %#v", transport.requests)
	}
}

func TestIsolatedRegistryAuthorizationReadsOnlyTemporaryDockerConfig(t *testing.T) {
	directory := t.TempDir()
	encoded := base64.StdEncoding.EncodeToString([]byte("user:password"))
	if err := os.WriteFile(filepath.Join(directory, "config.json"), []byte(`{"auths":{"registry.example":{"auth":"`+encoded+`"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	authorization, err := isolatedRegistryAuthorization(context.WithValue(context.Background(), dockerCredentialConfigContextKey{}, directory), "https://registry.example")
	if err != nil || authorization != "Basic "+encoded {
		t.Fatalf("isolatedRegistryAuthorization() = %q, %v", authorization, err)
	}
}
