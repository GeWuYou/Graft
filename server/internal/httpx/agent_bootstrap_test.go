package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
)

func TestNewAgentBootstrapServerIsAbsentWhenDisabled(t *testing.T) {
	server, err := NewAgentBootstrapServer(config.AgentBootstrapTLSConfig{}, nil)
	if err != nil {
		t.Fatalf("create disabled Agent bootstrap server: %v", err)
	}
	if server != nil {
		t.Fatal("disabled Agent bootstrap TLS must not create a listener")
	}
}

func TestAgentBootstrapCertificateHandlerReturnsOnlyIssuedMaterialWithoutCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authority := &recordingAgentBootstrapAuthority{result: moduleapi.AgentBootstrapResult{
		CertificateChainDER: [][]byte{[]byte("leaf"), []byte("issuer")},
		TrustBundle:         moduleapi.TrustBundleReference{Reference: "vault://agent-ca", Version: "v1", ExpiresAt: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)},
		ExpiresAt:           time.Date(2026, 8, 9, 1, 2, 3, 0, time.UTC),
	}}
	engine := gin.New()
	engine.Use(RequestIDMiddleware(), agentBootstrapNoStore())
	engine.POST(agentBootstrapCertificatePath, agentBootstrapCertificateHandler(authority, nil))

	request := httptest.NewRequest(http.MethodPost, agentBootstrapCertificatePath, strings.NewReader(`{"bootstrap_token":"one-time-token","csr_der":"AQI="}`))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected bootstrap success, got %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected no-store response, got %q", response.Header().Get("Cache-Control"))
	}
	if authority.request.BootstrapToken != "one-time-token" || string(authority.request.CSRDER) != "\x01\x02" {
		t.Fatalf("unexpected authority request %#v", authority.request)
	}
	var payload agentBootstrapCertificateResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode bootstrap response: %v", err)
	}
	if len(payload.CertificateChainDER) != 2 || string(payload.CertificateChainDER[0]) != "leaf" || payload.TrustBundle.Reference != "vault://agent-ca" {
		t.Fatalf("unexpected bootstrap response %#v", payload)
	}
	if strings.Contains(response.Body.String(), "one-time-token") || strings.Contains(response.Body.String(), "private") {
		t.Fatalf("bootstrap response leaked request secret: %s", response.Body.String())
	}
}

func TestAgentBootstrapCertificateHandlerRejectsUnexpectedAuthenticationAndPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authority := &recordingAgentBootstrapAuthority{}
	engine := gin.New()
	engine.Use(RequestIDMiddleware(), agentBootstrapNoStore())
	engine.POST(agentBootstrapCertificatePath, agentBootstrapCertificateHandler(authority, nil))

	for _, testCase := range []struct {
		name          string
		body          string
		authorization string
	}{
		{name: "bearer-header", body: `{"bootstrap_token":"token","csr_der":"AQI="}`, authorization: "Bearer forbidden"},
		{name: "unknown-field", body: `{"bootstrap_token":"token","csr_der":"AQI=","identity":"forged"}`},
		{name: "multiple-json-values", body: `{"bootstrap_token":"token","csr_der":"AQI="} {}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, agentBootstrapCertificatePath, strings.NewReader(testCase.body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", testCase.authorization)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("expected rejected request, got %d: %s", response.Code, response.Body.String())
			}
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("expected no-store rejection, got %q", response.Header().Get("Cache-Control"))
			}
		})
	}
	if authority.calls != 0 {
		t.Fatalf("invalid requests must not reach bootstrap authority, got %d calls", authority.calls)
	}
}

func TestAgentBootstrapCertificateHandlerMapsRejectionAndOperationalErrorsSeparately(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, testCase := range []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "rejected", err: moduleapi.ErrAgentBootstrapRejected, wantStatus: http.StatusUnauthorized},
		{name: "operational", err: errors.New("Vault temporarily unavailable"), wantStatus: http.StatusInternalServerError},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			authority := &recordingAgentBootstrapAuthority{err: testCase.err}
			engine := gin.New()
			engine.POST(agentBootstrapCertificatePath, agentBootstrapCertificateHandler(authority, nil))
			request := httptest.NewRequest(http.MethodPost, agentBootstrapCertificatePath, strings.NewReader(`{"bootstrap_token":"token","csr_der":"AQI="}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, testCase.wantStatus, response.Body.String())
			}
			if strings.Contains(response.Body.String(), testCase.err.Error()) {
				t.Fatalf("bootstrap response leaked authority error: %s", response.Body.String())
			}
		})
	}
}

type recordingAgentBootstrapAuthority struct {
	request moduleapi.AgentBootstrapRequest
	result  moduleapi.AgentBootstrapResult
	err     error
	calls   int
}

func (a *recordingAgentBootstrapAuthority) BootstrapAgent(_ context.Context, request moduleapi.AgentBootstrapRequest) (moduleapi.AgentBootstrapResult, error) {
	a.calls++
	a.request = request
	return a.result, a.err
}

var _ moduleapi.AgentBootstrapAuthority = (*recordingAgentBootstrapAuthority)(nil)
