package httpx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/moduleapi"
)

func TestAgentLedgerRoutesUseVerifiedMTLSIdentityAndNoStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reader := &recordingAgentLedgerReader{snapshot: moduleapi.RuntimeTargetLedgerSnapshot{SnapshotID: "snapshot", SnapshotDigest: "digest", ObservedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Minute)}}
	engine := gin.New()
	engine.Use(RequestIDMiddleware(), RequireAgentMTLSIdentity())
	server := &AgentServer{engine: engine}
	if err := server.ConfigureLedgerRoutes(reader); err != nil {
		t.Fatalf("configure ledger routes: %v", err)
	}

	request := agentLedgerMTLSRequest(t, http.MethodGet, agentLedgerSnapshotPath, "")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("snapshot response = %d %#v", response.Code, response.Header())
	}
	if reader.issued.TargetID != 7 || reader.issued.AgentID != "builder-7" || reader.issued.CertificateSerial != "42" {
		t.Fatalf("untrusted snapshot identity %#v", reader.issued)
	}
	for _, field := range []string{"snapshot_id", "snapshot_digest", "observed_at", "expires_at"} {
		if !strings.Contains(response.Body.String(), `"`+field+`"`) {
			t.Fatalf("snapshot response lacks private Agent wire field %q: %s", field, response.Body.String())
		}
	}

	reportBody := `{"snapshot_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","snapshot_digest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","observed_at":"2026-08-09T12:00:00Z","expires_at":"2026-08-09T12:01:00Z","available":true,"implementation_version":"v1"}`
	request = agentLedgerMTLSRequest(t, http.MethodPost, agentTelemetryReportsPath, reportBody)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("report response = %d %#v", response.Code, response.Header())
	}
	if reader.report.TargetID != 7 || reader.report.AgentID != "builder-7" || reader.report.CertificateSerial != "42" || reader.report.IdentityID != "" {
		t.Fatalf("untrusted report identity %#v", reader.report)
	}
}

func TestAgentLedgerRoutesRejectBearerAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestIDMiddleware(), RequireAgentMTLSIdentity())
	server := &AgentServer{engine: engine}
	if err := server.ConfigureLedgerRoutes(&recordingAgentLedgerReader{}); err != nil {
		t.Fatalf("configure ledger routes: %v", err)
	}
	request := agentLedgerMTLSRequest(t, http.MethodGet, agentLedgerSnapshotPath, "")
	request.Header.Set("Authorization", "Bearer forbidden")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("bearer rejection = %d %#v", response.Code, response.Header())
	}
}

func TestAgentLedgerRoutesRejectRepeatedConfigurationBeforeRouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &AgentServer{engine: gin.New()}
	if err := server.ConfigureLedgerRoutes(&recordingAgentLedgerReader{}); err != nil {
		t.Fatalf("configure ledger routes: %v", err)
	}
	if err := server.ConfigureLedgerRoutes(&recordingAgentLedgerReader{}); err == nil {
		t.Fatal("repeated ledger route configuration must return an error instead of re-registering Gin routes")
	}
}

func agentLedgerMTLSRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	certificate := testAgentCertificate(t, "spiffe://graft/runtime-target/7/builder-agent/builder-7")
	request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{certificate}}}
	return request
}

type recordingAgentLedgerReader struct {
	issued   moduleapi.AgentIdentity
	report   moduleapi.RuntimeTargetTelemetryReport
	snapshot moduleapi.RuntimeTargetLedgerSnapshot
}

func (r *recordingAgentLedgerReader) IssueLedgerSnapshot(_ context.Context, identity moduleapi.AgentIdentity) (moduleapi.RuntimeTargetLedgerSnapshot, error) {
	r.issued = identity
	return r.snapshot, nil
}
func (r *recordingAgentLedgerReader) SubmitTelemetryReport(_ context.Context, report moduleapi.RuntimeTargetTelemetryReport) error {
	r.report = report
	return nil
}

var _ moduleapi.RuntimeTargetAgentLedgerReader = (*recordingAgentLedgerReader)(nil)
