package update

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestPlatformUpdateDiagnosticUsesDefaultReleaseRepository(t *testing.T) {
	transport := &recordingRoundTripper{response: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}}
	factory := &outboundHTTPClientFactoryStub{client: &http.Client{Transport: transport}}
	target := platformUpdateDiagnosticTarget{factory: factory}

	result, err := target.ExecuteOutboundDiagnostic(context.Background())
	if err != nil {
		t.Fatalf("execute outbound diagnostic: %v", err)
	}
	if !result.Connected {
		t.Fatalf("expected successful diagnostic result, got %#v", result)
	}
	if factory.timeout != releaseHTTPTimeout {
		t.Fatalf("expected release timeout %s, got %s", releaseHTTPTimeout, factory.timeout)
	}
	wantURL := "https://api.github.com/repos/" + defaultReleaseRepository + "/releases"
	if transport.request == nil || transport.request.URL.String() != wantURL {
		t.Fatalf("expected diagnostic request URL %q, got %#v", wantURL, transport.request)
	}
}

type outboundHTTPClientFactoryStub struct {
	client  *http.Client
	timeout time.Duration
}

func (s *outboundHTTPClientFactoryStub) NewOutboundHTTPClient(_ context.Context, options ...moduleapi.OutboundHTTPClientOption) (*http.Client, error) {
	configured := moduleapi.OutboundHTTPClientOptions{}
	for _, option := range options {
		if err := option(&configured); err != nil {
			return nil, err
		}
	}
	s.timeout = configured.Timeout
	return s.client, nil
}

type recordingRoundTripper struct {
	request  *http.Request
	response *http.Response
}

func (t *recordingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.request = request
	return t.response, nil
}
