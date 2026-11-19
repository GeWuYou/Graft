package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaimExecutionLeaseClassifiesAdmission(t *testing.T) {
	tests := []struct {
		status     int
		wantClass  claimAdmissionClass
		wantErr    error
		wantStatus int
	}{
		{status: http.StatusNoContent, wantClass: claimAdmissionNoWork, wantErr: errExecutionNoWork, wantStatus: http.StatusNoContent},
		{status: http.StatusBadRequest, wantClass: claimAdmissionIdentityRejected, wantErr: errAgentIdentityRejected, wantStatus: http.StatusBadRequest},
		{status: http.StatusUnauthorized, wantClass: claimAdmissionIdentityRejected, wantErr: errAgentIdentityRejected, wantStatus: http.StatusUnauthorized},
		{status: http.StatusForbidden, wantClass: claimAdmissionIdentityRejected, wantErr: errAgentIdentityRejected, wantStatus: http.StatusForbidden},
		{status: http.StatusInternalServerError, wantClass: claimAdmissionTransportFailed, wantStatus: http.StatusInternalServerError},
	}
	for _, test := range tests {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(test.status)
		}))
		_, err := claimExecutionLease(context.Background(), server.Client(), server.URL, "docker", "compose_execution", "docker/v1")
		server.Close()
		var admission *claimAdmissionError
		if !errors.As(err, &admission) {
			t.Fatalf("claim status %d error = %v", test.status, err)
		}
		if admission.class != test.wantClass {
			t.Fatalf("claim status %d class = %s, want %s", test.status, admission.class, test.wantClass)
		}
		if admission.status != test.wantStatus {
			t.Fatalf("claim status %d recorded status = %d", test.status, admission.status)
		}
		if test.wantErr != nil && !errors.Is(err, test.wantErr) {
			t.Fatalf("claim status %d error chain missing %v", test.status, test.wantErr)
		}
	}
}

func TestClaimExecutionLeaseTransportFailureIsClassified(t *testing.T) {
	client := &http.Client{Timeout: time.Second}
	_, err := claimExecutionLease(context.Background(), client, "http://127.0.0.1:1", "docker", "compose_execution", "docker/v1")
	var admission *claimAdmissionError
	if !errors.As(err, &admission) || admission.class != claimAdmissionTransportFailed {
		t.Fatalf("transport error = %v", err)
	}
}

func TestAdmissionTrackerDedupesAndRateLimitsReports(t *testing.T) {
	tracker := newAdmissionTracker(t.TempDir())
	now := time.Now()
	tracker.validNow = func() time.Time { return now }

	tracker.observe(claimAdmissionNoWork, http.StatusNoContent)
	admission, report := tracker.shouldReport()
	if !report || admission.Class != claimAdmissionNoWork {
		t.Fatalf("first observation report = %#v %v", admission, report)
	}
	tracker.commitReport(admission)
	if _, report := tracker.shouldReport(); report {
		t.Fatal("unchanged admission was reported again")
	}
	now = now.Add(10 * time.Second)
	tracker.observe(claimAdmissionIdentityRejected, http.StatusForbidden)
	if _, report := tracker.shouldReport(); report {
		t.Fatal("changed admission bypassed the minimum reporting spacing")
	}
	now = now.Add(21 * time.Second)
	admission, report = tracker.shouldReport()
	if !report || admission.Class != claimAdmissionIdentityRejected || admission.HTTPStatus != http.StatusForbidden {
		t.Fatalf("released admission report = %#v %v", admission, report)
	}
	tracker.commitReport(admission)
	now = now.Add(5 * time.Second)
	tracker.observe(claimAdmissionTransportFailed, http.StatusInternalServerError)
	if _, report := tracker.shouldReport(); report {
		t.Fatal("rapid admission change bypassed the rate limit")
	}
	now = now.Add(30 * time.Second)
	if _, report := tracker.shouldReport(); !report {
		t.Fatal("rate limit window did not release the next report")
	}
	tracker.clear()
	admission, report = tracker.shouldReport()
	if !report || admission.Class != "" {
		t.Fatalf("recovery report = %#v %v", admission, report)
	}
	tracker.commitReport(admission)
	if _, report := tracker.shouldReport(); report {
		t.Fatal("recovered admission was reported again")
	}
	if class := tracker.currentAdmissionClass(); class != "" {
		t.Fatalf("cleared admission class = %s", class)
	}
}

func TestAdmissionTrackerPersistsStateForHealthcheck(t *testing.T) {
	dir := t.TempDir()
	tracker := newAdmissionTracker(dir)
	tracker.observe(claimAdmissionIdentityRejected, http.StatusForbidden)
	now := time.Now().UTC()
	if admission, ready, err := admissionReadiness(dir, now); err != nil || ready || admission.Class != claimAdmissionIdentityRejected {
		t.Fatalf("recent identity rejection readiness = %#v ready=%v err=%v", admission, ready, err)
	}
	if admission, ready, err := admissionReadiness(dir, now.Add(2*admissionHealthcheckWindow)); err != nil || !ready {
		t.Fatalf("stale identity rejection readiness = %#v ready=%v err=%v", admission, ready, err)
	}
	tracker.clear()
	if _, err := os.Stat(filepath.Join(dir, admissionStateFile)); !os.IsNotExist(err) {
		t.Fatalf("cleared admission state file err=%v", err)
	}
}

func TestPollExecutionCapabilitiesRecordsClaimAdmission(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	tracker := newAdmissionTracker(dir)
	pollExecutionCapabilities(ctx, server.Client(), config{AgentURL: server.URL, ProviderID: "docker", CapabilityVersion: "docker/v1", Capabilities: []string{"compose_execution"}}, persistedState{}, tracker)
	if class := tracker.currentAdmissionClass(); class != claimAdmissionIdentityRejected {
		t.Fatalf("observed admission class = %s", class)
	}
	persisted, err := loadAdmissionState(dir)
	if err != nil || persisted.Class != claimAdmissionIdentityRejected || persisted.HTTPStatus != http.StatusForbidden {
		t.Fatalf("persisted admission = %#v err=%v", persisted, err)
	}
}

func TestPollExecutionCapabilitiesClearsAdmissionAfterSuccessfulClaim(t *testing.T) {
	dir := t.TempDir()
	tracker := newAdmissionTracker(dir)
	tracker.observe(claimAdmissionIdentityRejected, http.StatusForbidden)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == executionClaimPath {
			_, _ = w.Write([]byte(`{"lease_id":"lease-admission-clear"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	pollExecutionCapabilities(context.Background(), server.Client(), config{AgentURL: server.URL, ProviderID: "docker", CapabilityVersion: "docker/v1", Capabilities: []string{"compose_execution"}}, persistedState{CertificatePEM: "certificate", ExpiresAt: time.Now().Add(time.Hour)}, tracker)
	if class := tracker.currentAdmissionClass(); class != "" {
		t.Fatalf("admission class after successful claim = %s", class)
	}
}

func TestHealthcheckSurfacesIdentityRejection(t *testing.T) {
	dir := t.TempDir()
	configPath := dir + "/agent.json"
	trustPath := dir + "/trust.pem"
	statePath := dir + "/state"
	if err := os.MkdirAll(statePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`{"bootstrap_url":"https://localhost:8443","agent_url":"https://localhost:8444","target_id":1,"agent_id":"agent-1","bootstrap_ca_file":"`+trustPath+`","trust_bundle_file":"`+trustPath+`","state_dir":"`+statePath+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trustPath, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveState(statePath, persistedState{TargetID: 1, AgentID: "agent-1", Identity: stableSPIFFE(1, "agent-1"), Generation: 1, CertificatePEM: "cert", ExpiresAt: time.Now().Add(time.Hour)}, []byte("key"), []byte("current")); err != nil {
		t.Fatal(err)
	}
	tracker := newAdmissionTracker(statePath)
	tracker.observe(claimAdmissionIdentityRejected, http.StatusForbidden)
	if err := healthcheck(configPath); err == nil || !strings.Contains(err.Error(), string(claimAdmissionIdentityRejected)) {
		t.Fatalf("healthcheck with recent identity rejection = %v", err)
	}
	if err := healthcheck(configPath); err == nil {
		t.Fatal("healthcheck should remain failing while the rejection window applies")
	}
	tracker.clear()
	if err := healthcheck(configPath); err != nil {
		t.Fatalf("healthcheck without admission record = %v", err)
	}
}
