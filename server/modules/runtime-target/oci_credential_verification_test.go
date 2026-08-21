package runtimetarget

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestProbeOCIRegistryV2DoesNotFollowRedirects(t *testing.T) {
	redirectTargetCalls := 0
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirectTargetCalls++
	}))
	defer redirectTarget.Close()
	redirectSource := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, redirectTarget.URL, http.StatusFound)
	}))
	defer redirectSource.Close()
	previous := ociRegistryVerificationHTTPClient
	ociRegistryVerificationHTTPClient = &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	t.Cleanup(func() { ociRegistryVerificationHTTPClient = previous })
	endpoint, err := url.Parse(redirectSource.URL + "/v2/")
	if err != nil {
		t.Fatal(err)
	}

	result := probeOCIRegistryV2(context.Background(), endpoint, "Basic dXNlcjpwYXNz")
	if !result.Reachable || result.ProtocolCompatible || result.AuthenticationSucceeded {
		t.Fatalf("verification result = %#v", result)
	}
	if redirectTargetCalls != 0 {
		t.Fatalf("redirect target calls = %d, want 0", redirectTargetCalls)
	}
}
