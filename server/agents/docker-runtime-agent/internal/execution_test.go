package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
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
	if len(paths) != 5 || paths[0] != "/agent/v1/execution-leases/lease-1/renew" || paths[1] != "/agent/v1/execution-leases/lease-1/logs" || paths[2] != "/agent/v1/execution-leases/lease-1/logs" || paths[3] != "/agent/v1/execution-leases/lease-1/logs" || paths[4] != "/agent/v1/execution-leases/lease-1/receipts" {
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
	if len(logLines) != 3 || logLines[0] != "execution lease accepted" || logLines[1] != "provider operation started" || logLines[2] != "cancellation observed" {
		t.Fatalf("logs=%v", logLines)
	}
}

//nolint:gocognit,gocyclo,cyclop // This test records the complete result-before-terminal ordering and recovery assertions.
func TestBuildExecutionSubmitsTransientResultBeforeTerminalJournalAndReceipt(t *testing.T) {
	stateDir := t.TempDir()
	lease := executionLease{ID: "lease-build", FenceToken: "fence-build", OperationID: buildImagePublishOperation, Protocol: buildExecutionProtocol, Capability: buildExecutionCapability, PayloadSHA256: "payload", LeaseTTLMS: 1000}
	resultPayload := json.RawMessage(`{"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","repository":"team/app","reference":"stable","media_type":"application/vnd.oci.image.manifest.v1+json","size_bytes":42}`)
	var mu sync.Mutex
	paths := make([]string, 0, 6)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/agent/v1/execution-leases/lease-build/renew":
			_ = json.NewEncoder(w).Encode(lease)
		case "/agent/v1/execution-leases/lease-build/logs":
			w.WriteHeader(http.StatusNoContent)
		case "/agent/v1/execution-leases/lease-build/results":
			var request executionResultRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.FenceToken != lease.FenceToken || request.Protocol != buildExecutionResultProtocol || string(request.Payload) != string(resultPayload) {
				t.Fatalf("result request=%#v", request)
			}
			journal, found, err := executionJournalForLease(stateDir, lease.ID)
			if err != nil || !found || journal.Phase != executionJournalRunning {
				t.Fatalf("journal before result settlement=%#v found=%v err=%v", journal, found, err)
			}
			journalJSON, _ := json.Marshal(journal)
			if strings.Contains(string(journalJSON), "team/app") || strings.Contains(string(journalJSON), "stable") {
				t.Fatalf("result leaked into running journal: %s", journalJSON)
			}
			w.WriteHeader(http.StatusNoContent)
		case "/agent/v1/execution-leases/lease-build/receipts":
			journal, found, err := executionJournalForLease(stateDir, lease.ID)
			if err != nil || !found || journal.Phase != executionJournalTerminal || journal.Outcome != "success" {
				t.Fatalf("terminal journal before receipt=%#v found=%v err=%v", journal, found, err)
			}
			_ = json.NewEncoder(w).Encode(executionSettlement{})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	err := executeLeaseWithOperation(context.Background(), server.Client(), config{AgentURL: server.URL, StateDir: stateDir}, lease, func(context.Context, executionLease) executionResult {
		return executionResult{Outcome: "success", Protocol: buildExecutionResultProtocol, Payload: resultPayload}
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{
		"/agent/v1/execution-leases/lease-build/renew",
		"/agent/v1/execution-leases/lease-build/logs",
		"/agent/v1/execution-leases/lease-build/logs",
		"/agent/v1/execution-leases/lease-build/results",
		"/agent/v1/execution-leases/lease-build/logs",
		"/agent/v1/execution-leases/lease-build/receipts",
	}
	if !slices.Equal(paths, want) {
		t.Fatalf("paths=%v want=%v", paths, want)
	}
}

func TestBuildResultTransportFailureRecoversAsNeedsAttentionWithoutProviderReplay(t *testing.T) {
	stateDir := t.TempDir()
	lease := executionLease{ID: "lease-build-recover", FenceToken: "fence-build-recover", OperationID: buildImageLocalOperation, Protocol: buildExecutionProtocol, Capability: buildExecutionCapability, PayloadSHA256: "payload", LeaseTTLMS: 1000} //nolint:gosec // Test-only opaque lease identifiers are not credentials.
	providerCalls := 0
	var receipt executionReceipt
	resultFailures := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/agent/v1/execution-leases/lease-build-recover/renew":
			_ = json.NewEncoder(w).Encode(lease)
		case "/agent/v1/execution-leases/lease-build-recover/logs":
			w.WriteHeader(http.StatusNoContent)
		case "/agent/v1/execution-leases/lease-build-recover/results":
			resultFailures++
			w.WriteHeader(http.StatusServiceUnavailable)
		case "/agent/v1/execution-leases/lease-build-recover/receipts":
			_ = json.NewDecoder(r.Body).Decode(&receipt)
			_ = json.NewEncoder(w).Encode(executionSettlement{})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	operation := func(context.Context, executionLease) executionResult {
		providerCalls++
		return executionResult{Outcome: "success", Protocol: buildExecutionResultProtocol, Payload: json.RawMessage(`{"repository":"local/app","reference":"latest","size_bytes":1}`)}
	}
	if err := executeLeaseWithOperation(context.Background(), server.Client(), config{AgentURL: server.URL, StateDir: stateDir}, lease, operation); err == nil {
		t.Fatal("result transport failure unexpectedly terminalized execution")
	}
	journal, found, err := executionJournalForLease(stateDir, lease.ID)
	if err != nil || !found || journal.Phase != executionJournalRunning {
		t.Fatalf("journal after result failure=%#v found=%v err=%v", journal, found, err)
	}
	if err := executeLeaseWithOperation(context.Background(), server.Client(), config{AgentURL: server.URL, StateDir: stateDir}, lease, operation); err != nil {
		t.Fatal(err)
	}
	if providerCalls != 1 || resultFailures != 1 || receipt.Outcome != "needs_attention" || receipt.FailureCode != failureInterrupted {
		t.Fatalf("providerCalls=%d resultFailures=%d receipt=%#v", providerCalls, resultFailures, receipt)
	}
	if _, err := os.Stat(executionJournalPath(stateDir, lease.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovered journal was not removed: %v", err)
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

func TestExecutionJournalRunningLeaseSettlesNeedsAttentionWithoutReplay(t *testing.T) {
	stateDir := t.TempDir()
	lease := executionLease{ID: "lease-recover", FenceToken: "fence-recover", OperationID: "container.lifecycle.start.v1", Protocol: containerExecutionProtocol, Capability: "container_execution", LeaseTTLMS: 1000}
	if err := saveExecutionJournal(stateDir, executionJournal{Lease: lease, Phase: executionJournalRunning}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(executionJournalPath(stateDir, lease.ID))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("journal mode=%v", info.Mode().Perm())
	}
	replayed := false
	var receipt executionReceipt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/agent/v1/execution-leases/lease-recover/renew":
			_ = json.NewEncoder(w).Encode(lease)
		case "/agent/v1/execution-leases/lease-recover/receipts":
			_ = json.NewDecoder(r.Body).Decode(&receipt)
			_ = json.NewEncoder(w).Encode(executionSettlement{})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	if err := executeLeaseWithOperation(context.Background(), server.Client(), config{AgentURL: server.URL, StateDir: stateDir}, lease, func(context.Context, executionLease) executionResult {
		replayed = true
		return executionResult{Outcome: "success"}
	}); err != nil {
		t.Fatal(err)
	}
	if replayed || receipt.Outcome != "needs_attention" || receipt.FailureCode != failureInterrupted {
		t.Fatalf("replayed=%v receipt=%#v", replayed, receipt)
	}
	if _, err := os.Stat(executionJournalPath(stateDir, lease.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("terminal receipt did not clear journal: %v", err)
	}
}

func TestPollExecutionCapabilitiesClaimsWithoutSequentialStarvation(t *testing.T) {
	var mu sync.Mutex
	claims := 0
	allClaims := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != executionClaimPath {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		mu.Lock()
		claims++
		if claims == 3 {
			close(allClaims)
		}
		mu.Unlock()
		select {
		case <-allClaims:
			w.WriteHeader(http.StatusNoContent)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	pollExecutionCapabilities(ctx, server.Client(), config{AgentURL: server.URL, ProviderID: "docker", CapabilityVersion: "docker/v1", Capabilities: []string{"oci-build", "compose_execution", "container_execution"}}, persistedState{})
	mu.Lock()
	defer mu.Unlock()
	if claims != 3 {
		t.Fatalf("claims=%d", claims)
	}
}

func TestContainerProviderRejectsUnknownAndCrossOperationFields(t *testing.T) {
	tests := []json.RawMessage{
		json.RawMessage(`{"container_ref":"container-1","endpoint":"secret"}`),
		json.RawMessage(`{"container_ref":"container-1","image_ref":"image-1"}`),
	}
	for _, payload := range tests {
		result := executeContainerOperation(context.Background(), config{DockerSocket: "unix:///missing"}, executionLease{Protocol: containerExecutionProtocol, OperationID: "container.lifecycle.start.v1", Input: payload})
		if result.Outcome != "failed" || result.FailureCode != failureInvalidIntent {
			t.Fatalf("result=%#v", result)
		}
	}
}
