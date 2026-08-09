package credentialvault

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/event"
	"graft/server/internal/moduleapi"
	runtimetargetcontract "graft/server/modules/runtime-target/contract"
)

//nolint:gocognit,gocyclo,cyclop // 单个场景固定验证 AppRole、PKI 签发和秘密不落盘这条不可拆分的安全边界。
func TestVaultPKIClientUsesDockerSecretsForAppRoleAndPersistsOnlySerial(t *testing.T) {
	certificatePEM, certificateSerial, csrDER := newVaultConformanceCertificate(t)
	roleIDPath := filepath.Join(t.TempDir(), "role-id")
	secretIDPath := filepath.Join(t.TempDir(), "secret-id")
	if err := os.WriteFile(roleIDPath, []byte("role-from-docker-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretIDPath, []byte("secret-from-docker-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := &conformanceIssuanceStore{}
	var (
		requestMu    sync.Mutex
		loginBody    map[string]string
		issueBody    map[string]string
		requestCount int
	)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestMu.Lock()
		defer requestMu.Unlock()
		requestCount++
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/auth/approle/login":
			if request.Header.Get("X-Vault-Token") != "" {
				t.Errorf("login request unexpectedly has Vault token")
			}
			if err := json.NewDecoder(request.Body).Decode(&loginBody); err != nil {
				t.Errorf("decode login body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"auth":{"client_token":"session-token"}}`))
		case "/v1/pki/issue/agent":
			if got := request.Header.Get("X-Vault-Token"); got != "session-token" {
				t.Errorf("issue token = %q, want session-token", got)
			}
			if err := json.NewDecoder(request.Body).Decode(&issueBody); err != nil {
				t.Errorf("decode issue body: %v", err)
			}
			_, _ = writer.Write([]byte(`{"data":{"certificate":` + strconvQuote(certificatePEM) + `,"serial_number":"` + certificateSerial + `","expiration":4102444800}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewVaultPKIClient(config.CredentialVaultConfig{
		Address: server.URL, AuthMount: "approle", AuthRole: "agent", AuthRoleIDFile: roleIDPath,
		AuthSecretIDFile: secretIDPath, PKIMount: "pki", PKIRole: "agent", TrustBundleRef: "vault://pki/ca",
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	if client.http.Timeout != vaultRequestTimeout {
		t.Fatalf("Vault HTTP timeout = %s, want %s", client.http.Timeout, vaultRequestTimeout)
	}
	issued, err := client.IssueCSR(context.Background(), moduleapi.AgentCertificateIssuanceRequest{IssuanceKey: "issue-1", SPIFFEURI: "spiffe://graft/runtime-target/7/builder-agent/agent-7/generation/1", CSRDER: csrDER})
	if err != nil {
		t.Fatalf("issue certificate: %v", err)
	}
	if issued.CertificateSerial != certificateSerial {
		t.Fatalf("certificate serial = %q, want %q", issued.CertificateSerial, certificateSerial)
	}
	requestMu.Lock()
	defer requestMu.Unlock()
	if loginBody["role_id"] != "role-from-docker-secret" || loginBody["secret_id"] != "secret-from-docker-secret" {
		t.Fatalf("AppRole login body = %#v", loginBody)
	}
	if !strings.HasPrefix(issueBody["csr"], "-----BEGIN CERTIFICATE REQUEST-----") {
		t.Fatalf("issue CSR is not PEM encoded: %q", issueBody["csr"])
	}
	if issueBody["uri_sans"] != "spiffe://graft/runtime-target/7/builder-agent/agent-7/generation/1" {
		t.Fatalf("issue URI SAN = %q", issueBody["uri_sans"])
	}
	if store.state != (IssuanceState{IssuanceKey: "issue-1", Serial: certificateSerial}) {
		t.Fatalf("persisted state = %#v", store.state)
	}
	stateJSON, _ := json.Marshal(store.state)
	for _, secret := range []string{"role-from-docker-secret", "secret-from-docker-secret", "session-token", certificatePEM} {
		if strings.Contains(string(stateJSON), secret) {
			t.Fatalf("persisted state contains secret material %q: %s", secret, stateJSON)
		}
	}
	if requestCount != 2 {
		t.Fatalf("Vault request count = %d, want login plus issue", requestCount)
	}
}

func TestVaultPKIClientReconcileRehydratesPersistedSerialAfterRestart(t *testing.T) {
	certificatePEM, certificateSerial, _ := newVaultConformanceCertificate(t)
	store := &conformanceIssuanceStore{state: IssuanceState{IssuanceKey: "issue-1", Serial: certificateSerial}}
	roleIDPath, secretIDPath := writeConformanceSecrets(t)
	var (
		requestMu sync.Mutex
		paths     []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestMu.Lock()
		defer requestMu.Unlock()
		paths = append(paths, request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/v1/auth/approle/login":
			_, _ = writer.Write([]byte(`{"auth":{"client_token":"restart-token"}}`))
		case "/v1/pki/cert/serial-42":
			if request.Header.Get("X-Vault-Token") != "restart-token" {
				t.Errorf("certificate read token = %q", request.Header.Get("X-Vault-Token"))
			}
			_, _ = writer.Write([]byte(`{"data":{"certificate":` + strconvQuote(certificatePEM) + `,"serial_number":"incorrect-vault-serial","expiration":4102444800}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client, err := NewVaultPKIClient(config.CredentialVaultConfig{Address: server.URL, AuthMount: "approle", AuthRole: "agent", AuthRoleIDFile: roleIDPath, AuthSecretIDFile: secretIDPath, PKIMount: "pki", PKIRole: "agent"}, store)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := client.ReconcileCSR(context.Background(), "issue-1")
	if err != nil {
		t.Fatalf("reconcile certificate: %v", err)
	}
	if issued.CertificateSerial != certificateSerial || len(issued.CertificateChainDER) != 1 {
		t.Fatalf("reconciled certificate = %#v", issued)
	}
	requestMu.Lock()
	defer requestMu.Unlock()
	if strings.Join(paths, ",") != "/v1/auth/approle/login,/v1/pki/cert/serial-42" {
		t.Fatalf("restart reconciliation paths = %v", paths)
	}
}

func TestVaultPKIClientRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"auth":{"client_token":"` + strings.Repeat("x", vaultResponseMaxBytes) + `"}}`))
	}))
	defer server.Close()

	client := &VaultPKIClient{config: config.CredentialVaultConfig{Address: server.URL}, http: server.Client()}
	var response vaultLoginResponse
	err := client.call(context.Background(), "", http.MethodGet, "/", nil, &response)
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("oversized Vault response error = %v", err)
	}
}

func TestAgentCertificateRevocationHandlerConformance(t *testing.T) {
	tests := []struct {
		name        string
		version     uint16
		id          string
		idempotency string
		payload     string
		wantErr     bool
		wantKey     string
		wantCalls   int
	}{
		{name: "explicit stable key", version: 1, id: "event-id", idempotency: "stable-key", payload: `{"certificate_serial":"serial-1"}`, wantKey: "stable-key", wantCalls: 1},
		{name: "event id fallback", version: 1, id: "event-id", payload: `{"certificate_serial":"serial-1"}`, wantKey: "event-id", wantCalls: 1},
		{name: "invalid version", version: 2, id: "event-id", payload: `{"certificate_serial":"serial-1"}`, wantErr: true},
		{name: "invalid envelope payload", version: 1, id: "event-id", payload: `{`, wantErr: true},
		{name: "empty serial no-op", version: 1, id: "event-id", payload: `{"certificate_serial":" "}`, wantCalls: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issuer := &conformanceRevocationIssuer{}
			handler := agentCertificateRevocationHandler{issuer: issuer}
			err := handler.Handle(context.Background(), event.Event{ID: test.id, Type: runtimetargetcontract.AgentCertificateRevocationEventType, Version: test.version, Payload: []byte(test.payload), IdempotencyKey: test.idempotency})
			if (err != nil) != test.wantErr {
				t.Fatalf("handle error = %v, wantErr %v", err, test.wantErr)
			}
			if issuer.calls != test.wantCalls {
				t.Fatalf("issuer calls = %d, want %d", issuer.calls, test.wantCalls)
			}
			if test.wantKey != "" && issuer.revocation.IdempotencyKey != test.wantKey {
				t.Fatalf("idempotency key = %q, want %q", issuer.revocation.IdempotencyKey, test.wantKey)
			}
		})
	}
}

//nolint:gocognit,gocyclo // 同时断言 Vault 短暂不可用的 retry 与耗尽后的 terminal 状态，避免两条 conformance 路径分离。
func TestAgentCertificateRevocationUsesDurableOutboxRetryAndTerminalFailure(t *testing.T) {
	payload, err := json.Marshal(runtimetargetcontract.AgentCertificateRevocationEvent{
		IdentityID: "identity-1", TargetID: 7, AgentID: "agent-1", Generation: 3,
		CertificateSerial: "serial-1", Reason: "generation revoked",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("eventual Vault reconciliation", func(t *testing.T) {
		store := newConformanceOutboxStore()
		issuer := &durableRevocationIssuer{}
		issuer.failFor.Store(1)
		dispatcher, err := event.NewDurableDispatcher(nil, event.Options{
			WorkerCount: 1, OutboxPoll: time.Millisecond, RetryDelay: time.Millisecond, MaxAttempts: 3,
		}, store)
		if err != nil {
			t.Fatal(err)
		}
		if err := dispatcher.Register(agentCertificateRevocationHandler{issuer: issuer}); err != nil {
			t.Fatal(err)
		}
		if err := dispatcher.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		defer shutdownConformanceDispatcher(t, dispatcher)

		receipt, err := dispatcher.Publish(context.Background(), event.Event{
			ID: "revocation-event-retry", Type: runtimetargetcontract.AgentCertificateRevocationEventType,
			Version: 1, Source: runtimetargetcontract.ModuleID, Payload: payload,
			IdempotencyKey: "revocation-idempotency-1",
		}, event.PublishOptions{Delivery: event.DeliveryDurable})
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Delivery != event.DeliveryDurable {
			t.Fatalf("delivery mode = %q, want durable", receipt.Delivery)
		}

		waitForConformanceDelivery(t, store, "revocation-event-retry", agentCertificateRevocationHandlerID, conformanceDeliveryDelivered)
		if got := issuer.revokeCalls.Load(); got != 2 {
			t.Fatalf("Vault revoke calls = %d, want transient failure plus retry", got)
		}
		issuer.mu.Lock()
		defer issuer.mu.Unlock()
		if issuer.revocation.IdempotencyKey != "revocation-idempotency-1" || issuer.revocation.CertificateSerial != "serial-1" {
			t.Fatalf("revocation binding = %#v", issuer.revocation)
		}
	})

	t.Run("terminal Vault failure", func(t *testing.T) {
		store := newConformanceOutboxStore()
		issuer := &durableRevocationIssuer{}
		issuer.failFor.Store(100)
		dispatcher, err := event.NewDurableDispatcher(nil, event.Options{
			WorkerCount: 1, OutboxPoll: time.Millisecond, RetryDelay: time.Millisecond, MaxAttempts: 2,
		}, store)
		if err != nil {
			t.Fatal(err)
		}
		if err := dispatcher.Register(agentCertificateRevocationHandler{issuer: issuer}); err != nil {
			t.Fatal(err)
		}
		if err := dispatcher.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		defer shutdownConformanceDispatcher(t, dispatcher)

		if _, err := dispatcher.Publish(context.Background(), event.Event{
			ID: "revocation-event-terminal", Type: runtimetargetcontract.AgentCertificateRevocationEventType,
			Version: 1, Source: runtimetargetcontract.ModuleID, Payload: payload,
			IdempotencyKey: "revocation-idempotency-terminal",
		}, event.PublishOptions{Delivery: event.DeliveryDurable}); err != nil {
			t.Fatal(err)
		}
		waitForConformanceDelivery(t, store, "revocation-event-terminal", agentCertificateRevocationHandlerID, conformanceDeliveryFailed)
		if got := issuer.revokeCalls.Load(); got != 2 {
			t.Fatalf("Vault revoke calls = %d, want max attempts 2", got)
		}
		time.Sleep(10 * time.Millisecond)
		if got := issuer.revokeCalls.Load(); got != 2 {
			t.Fatalf("terminal delivery was retried after failure, calls = %d", got)
		}
	})
}

const (
	conformanceDeliveryPending    = "pending"
	conformanceDeliveryProcessing = "processing"
	conformanceDeliveryDelivered  = "delivered"
	conformanceDeliveryFailed     = "failed"
)

type conformanceOutboxStore struct {
	mu         sync.Mutex
	deliveries map[string]*conformanceOutboxDelivery
}

type conformanceOutboxDelivery struct {
	delivery    event.ClaimedDelivery
	status      string
	availableAt time.Time
	leaseUntil  time.Time
}

func newConformanceOutboxStore() *conformanceOutboxStore {
	return &conformanceOutboxStore{deliveries: make(map[string]*conformanceOutboxDelivery)}
}

func (s *conformanceOutboxStore) Append(_ context.Context, incoming event.Event, consumers []string) (event.Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, consumer := range consumers {
		key := incoming.ID + "\x00" + consumer
		if _, exists := s.deliveries[key]; !exists {
			s.deliveries[key] = &conformanceOutboxDelivery{delivery: event.ClaimedDelivery{Event: incoming, ConsumerID: consumer}, status: conformanceDeliveryPending, availableAt: incoming.CreatedAt}
		}
	}
	return event.Receipt{EventID: incoming.ID, Delivery: event.DeliveryDurable}, nil
}

func (s *conformanceOutboxStore) Claim(_ context.Context, _ string, now time.Time, lease time.Duration, limit int) ([]event.ClaimedDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	claimed := make([]event.ClaimedDelivery, 0, limit)
	for _, item := range s.deliveries {
		pending := item.status == conformanceDeliveryPending
		reclaimable := item.status == conformanceDeliveryProcessing && !item.leaseUntil.After(now)
		if len(claimed) >= limit || (!pending && !reclaimable) || item.availableAt.After(now) {
			continue
		}
		item.status = conformanceDeliveryProcessing
		item.delivery.Attempt++
		item.leaseUntil = now.Add(lease)
		claimed = append(claimed, item.delivery)
	}
	return claimed, nil
}

func (s *conformanceOutboxStore) Renew(context.Context, event.ClaimedDelivery, time.Time, time.Duration) error {
	return nil
}

func (s *conformanceOutboxStore) Complete(_ context.Context, delivery event.ClaimedDelivery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.deliveries[delivery.Event.ID+"\x00"+delivery.ConsumerID]
	if item == nil {
		return errors.New("delivery not found")
	}
	item.status, item.leaseUntil = conformanceDeliveryDelivered, time.Time{}
	return nil
}

func (s *conformanceOutboxStore) Retry(_ context.Context, delivery event.ClaimedDelivery, availableAt time.Time, _ error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.deliveries[delivery.Event.ID+"\x00"+delivery.ConsumerID]
	if item == nil {
		return errors.New("delivery not found")
	}
	item.status, item.availableAt, item.leaseUntil = conformanceDeliveryPending, availableAt, time.Time{}
	return nil
}

func (s *conformanceOutboxStore) Fail(_ context.Context, delivery event.ClaimedDelivery, _ error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.deliveries[delivery.Event.ID+"\x00"+delivery.ConsumerID]
	if item == nil {
		return errors.New("delivery not found")
	}
	item.status, item.leaseUntil = conformanceDeliveryFailed, time.Time{}
	return nil
}

func (s *conformanceOutboxStore) status(eventID, consumerID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item := s.deliveries[eventID+"\x00"+consumerID]; item != nil {
		return item.status
	}
	return ""
}

func waitForConformanceDelivery(t *testing.T, store *conformanceOutboxStore, eventID, consumerID, want string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if store.status(eventID, consumerID) == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("delivery %s/%s did not reach %s", eventID, consumerID, want)
}

func shutdownConformanceDispatcher(t *testing.T, dispatcher *event.Dispatcher) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := dispatcher.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown dispatcher: %v", err)
	}
}

type durableRevocationIssuer struct {
	conformanceRevocationIssuer
	failFor     atomic.Int32
	revokeCalls atomic.Int32
	mu          sync.Mutex
}

func (i *durableRevocationIssuer) RevokeCertificate(_ context.Context, revocation moduleapi.AgentCertificateRevocation) error {
	i.revokeCalls.Add(1)
	i.mu.Lock()
	i.revocation = revocation
	i.mu.Unlock()
	if i.revokeCalls.Load() <= i.failFor.Load() {
		return errors.New("vault unavailable")
	}
	return nil
}

func TestAgentCertificateRevocationHandlerRetriesTransientVaultFailure(t *testing.T) {
	issuer := &conformanceRevocationIssuer{failures: 1}
	handler := agentCertificateRevocationHandler{issuer: issuer}
	incoming := event.Event{ID: "event-retry", Type: runtimetargetcontract.AgentCertificateRevocationEventType, Version: 1, IdempotencyKey: "stable-revocation", Payload: []byte(`{"certificate_serial":"serial-retry"}`)}
	if err := handler.Handle(context.Background(), incoming); err == nil {
		t.Fatal("transient Vault failure unexpectedly succeeded")
	}
	if err := handler.Handle(context.Background(), incoming); err != nil {
		t.Fatalf("retry after transient Vault failure: %v", err)
	}
	if issuer.calls != 2 || issuer.revocation.IdempotencyKey != "stable-revocation" {
		t.Fatalf("retry evidence = calls=%d revocation=%#v", issuer.calls, issuer.revocation)
	}
}

type conformanceIssuanceStore struct{ state IssuanceState }

func (s *conformanceIssuanceStore) Load(_ context.Context, issuanceKey string) (IssuanceState, error) {
	if s.state.IssuanceKey != issuanceKey {
		return IssuanceState{}, moduleapi.ErrAgentCertificateIssuanceNotFound
	}
	return s.state, nil
}

func (s *conformanceIssuanceStore) Save(_ context.Context, state IssuanceState) error {
	s.state = state
	return nil
}

type conformanceRevocationIssuer struct {
	calls      int
	failures   int
	revocation moduleapi.AgentCertificateRevocation
}

func (i *conformanceRevocationIssuer) IssueCSR(context.Context, moduleapi.AgentCertificateIssuanceRequest) (moduleapi.IssuedAgentCertificate, error) {
	return moduleapi.IssuedAgentCertificate{}, nil
}

func (i *conformanceRevocationIssuer) ReconcileCSR(context.Context, string) (moduleapi.IssuedAgentCertificate, error) {
	return moduleapi.IssuedAgentCertificate{}, nil
}

func (i *conformanceRevocationIssuer) ReadTrustBundle(context.Context, moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	return moduleapi.TrustBundleReference{}, nil
}

func (i *conformanceRevocationIssuer) RevokeCertificate(_ context.Context, revocation moduleapi.AgentCertificateRevocation) error {
	i.calls++
	if i.failures > 0 {
		i.failures--
		return errors.New("vault unavailable")
	}
	i.revocation = revocation
	return nil
}

func writeConformanceSecrets(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	roleIDPath, secretIDPath := filepath.Join(dir, "role-id"), filepath.Join(dir, "secret-id")
	if err := os.WriteFile(roleIDPath, []byte("role\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretIDPath, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return roleIDPath, secretIDPath
}

func newVaultConformanceCertificate(t *testing.T) (string, string, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{SerialNumber: new(big.Int).SetInt64(42), Subject: pkix.Name{CommonName: "agent"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: "agent"}}, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), "serial-42", csrDER
}

func strconvQuote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
