package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

func TestExecutionTransportRenewsLogsAndSettlesWithoutSecrets(t *testing.T) {
	var mu sync.Mutex
	paths := make([]string, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/agent/v1/execution-leases/lease-1/renew":
			_ = json.NewEncoder(w).Encode(executionLease{ID: "lease-1", FenceToken: "fence-1", LeaseTTLMS: 1000, LeaseExpiresAt: time.Now().Add(time.Second), AbsoluteDeadlineAt: time.Now().Add(time.Minute)})
		case "/agent/v1/execution-leases/lease-1/logs":
			w.WriteHeader(http.StatusNoContent)
		case "/agent/v1/execution-leases/lease-1/receipts":
			_ = json.NewEncoder(w).Encode(executionSettlement{TaskID: 1, StageID: 2})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	lease := executionLease{ID: "lease-1", TaskID: 1, StageID: 2, FenceToken: "fence-1", OperationID: "operation-1", PayloadSHA256: "payload", Protocol: "runtime-agent/v1", LeaseExpiresAt: time.Now().Add(time.Minute), AbsoluteDeadlineAt: time.Now().Add(time.Hour)}
	if err := executeLease(context.Background(), server.Client(), config{AgentURL: server.URL}, lease); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 3 || paths[0] != "/agent/v1/execution-leases/lease-1/renew" || paths[1] != "/agent/v1/execution-leases/lease-1/logs" || paths[2] != "/agent/v1/execution-leases/lease-1/receipts" {
		t.Fatalf("paths=%v", paths)
	}
}

func TestExecutionRenewsUntilCancellationAndSubmitsBoundedReceipt(t *testing.T) {
	var mu sync.Mutex
	renewals := 0
	logLines := make([]string, 0, 2)
	receiptOutcome := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/agent/v1/execution-leases/lease-2/renew":
			mu.Lock()
			renewals++
			cancelRequested := renewals > 1
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(executionLease{ID: "lease-2", FenceToken: "fence-2", LeaseTTLMS: 20, CancellationRequested: cancelRequested})
		case "/agent/v1/execution-leases/lease-2/logs":
			var request executionLogsRequest
			_ = json.NewDecoder(r.Body).Decode(&request)
			mu.Lock()
			for _, entry := range request.Entries {
				logLines = append(logLines, entry.Line)
			}
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		case "/agent/v1/execution-leases/lease-2/receipts":
			var request executionReceipt
			_ = json.NewDecoder(r.Body).Decode(&request)
			mu.Lock()
			receiptOutcome = request.Outcome
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(executionSettlement{TaskID: 2, StageID: 3})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	lease := executionLease{ID: "lease-2", FenceToken: "fence-2", OperationID: "operation-2", PayloadSHA256: "payload", Protocol: "runtime-agent/v1", LeaseTTLMS: 20}
	err := executeLeaseWithOperation(context.Background(), server.Client(), config{AgentURL: server.URL}, lease, func(ctx context.Context, _ executionLease) executionResult {
		<-ctx.Done()
		return executionResult{Outcome: "needs_attention", FailureCode: "interrupted"}
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if renewals < 2 || receiptOutcome != "success" {
		t.Fatalf("renewals=%d receipt=%q", renewals, receiptOutcome)
	}
	if len(logLines) != 2 || logLines[0] != "execution lease accepted" || logLines[1] != "cancellation observed" {
		t.Fatalf("logs=%v", logLines)
	}
}

func TestHealthcheckRejectsStaleTrustBundle(t *testing.T) {
	dir := t.TempDir()
	configPath := dir + "/agent.json"
	stateDir := dir + "/state"
	trustPath := dir + "/trust.pem"
	if err := os.WriteFile(configPath, []byte(`{"bootstrap_url":"https://localhost:8443","agent_url":"https://localhost:8444","target_id":1,"agent_id":"agent-1","bootstrap_ca_file":"`+trustPath+`","trust_bundle_file":"`+trustPath+`","state_dir":"`+stateDir+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trustPath, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveState(stateDir, persistedState{TargetID: 1, AgentID: "agent-1", Identity: stableSPIFFE(1, "agent-1"), Generation: 1, CertificatePEM: "cert", ExpiresAt: time.Now().Add(time.Hour)}, []byte("key"), []byte("stale")); err != nil {
		t.Fatal(err)
	}
	if err := healthcheck(configPath); err == nil {
		t.Fatal("stale trust bundle passed readiness")
	}
}

func TestHealthcheckAcceptsCurrentIdentityState(t *testing.T) {
	dir := t.TempDir()
	configPath := dir + "/agent.json"
	stateDir := dir + "/state"
	trustPath := dir + "/trust.pem"
	if err := os.WriteFile(configPath, []byte(`{"bootstrap_url":"https://localhost:8443","agent_url":"https://localhost:8444","target_id":1,"agent_id":"agent-1","bootstrap_ca_file":"`+trustPath+`","trust_bundle_file":"`+trustPath+`","state_dir":"`+stateDir+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trustPath, []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := persistedState{TargetID: 1, AgentID: "agent-1", Identity: stableSPIFFE(1, "agent-1"), Generation: 2, CertificatePEM: "cert", ExpiresAt: time.Now().Add(time.Hour)}
	if err := saveState(stateDir, state, []byte("key"), []byte("current")); err != nil {
		t.Fatal(err)
	}
	if err := healthcheck(configPath); err != nil {
		t.Fatal(err)
	}
}

func TestSafeAgentErrorRedactsTransportAndCredentialDetails(t *testing.T) {
	detail := "https://secret.example:8444 /run/secrets/agent.pem token-value"
	message := safeAgentError(errors.New(detail))
	if message == detail || message != "runtime agent operation failed" {
		t.Fatalf("safe error=%q", message)
	}
}

func TestExecutionLoopRetriesTransientClientFailureUntilCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	state := persistedState{CertificatePEM: "certificate", ExpiresAt: time.Now().Add(time.Hour)}
	err := runExecutionLoop(ctx, config{StateDir: t.TempDir(), PollInterval: time.Millisecond, RenewBefore: time.Minute}, state)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("execution loop error=%v", err)
	}
}
