package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	rsaKeyBits         = 2048
	requestTimeout     = 30 * time.Second
	deliveryWaitPeriod = 2 * time.Second
	maxResponseBytes   = 1 << 20
	bootstrapPath      = "/bootstrap/v1/certificate"
	ledgerSnapshotPath = "/agent/v1/ledger-snapshot"
	telemetryPath      = "/agent/v1/telemetry-reports"
)

type bootstrapResponse struct {
	CertificateChainDER [][]byte `json:"certificate_chain_der"`
	TrustBundle         struct {
		Reference string    `json:"reference"`
		Version   string    `json:"version"`
		ExpiresAt time.Time `json:"expires_at"`
	} `json:"trust_bundle"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ledgerSnapshot struct {
	Generation     int64     `json:"generation"`
	SnapshotID     string    `json:"snapshot_id"`
	SnapshotDigest string    `json:"snapshot_digest"`
	Available      bool      `json:"available"`
	ObservedAt     time.Time `json:"observed_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type certificateRequest struct {
	key    *rsa.PrivateKey
	csrDER []byte
	keyPEM []byte
}

// RunCLI 运行 Docker Runtime Agent 的进程入口。
// 该入口只装配 Agent 私有配置与生命周期，不承担 Runtime Target 的控制面授权。
//
//nolint:nestif // CLI 模式互斥且需要在建立信号生命周期后显式装配。
func RunCLI() {
	configPath := flag.String("config", defaultConfigFile(), "agent configuration path")
	showVersion := flag.Bool("version", false, "print agent version")
	once := flag.Bool("once", false, "run one lifecycle and exit")
	health := flag.Bool("healthcheck", false, "check local readiness without opening a listener")
	waitForDelivery := flag.Bool("wait-for-delivery", false, "wait for Backend-owned development delivery material")
	flag.Parse()
	if *showVersion {
		fmt.Println("graft-docker-runtime-agent " + version)
		return
	}
	if *health {
		if err := healthcheck(*configPath); err != nil {
			fatal(err)
		}
		return
	}
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	var err error
	if *waitForDelivery {
		err = runUntilDelivery(sigCtx, *configPath)
	} else {
		c, loadErr := loadConfig(*configPath)
		if loadErr != nil {
			fatal(loadErr)
		}
		if *once {
			err = runOnce(sigCtx, c)
		} else {
			err = run(sigCtx, c)
		}
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	if err != nil {
		fatal(err)
	}
	<-sigCtx.Done()
}

func runUntilDelivery(ctx context.Context, configPath string) error {
	lastError := ""
	for {
		c, err := loadConfig(configPath)
		if err == nil {
			err = run(ctx, c)
		}
		if err == nil {
			return nil
		}
		if message := safeAgentError(err); message != lastError {
			fmt.Fprintf(os.Stderr, "graft-docker-runtime-agent: waiting for Backend-owned delivery: %s\n", message)
			lastError = message
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(deliveryWaitPeriod):
		}
	}
}

func run(ctx context.Context, c config) error {
	state, err := runEnrollmentAndReconnect(ctx, c)
	if err != nil {
		return err
	}
	return runExecutionLoop(ctx, c, state)
}

func runOnce(ctx context.Context, c config) error {
	_, err := runEnrollmentAndReconnect(ctx, c)
	return err
}

func runEnrollmentAndReconnect(ctx context.Context, c config) (persistedState, error) {
	state, err := loadState(c.StateDir)
	if err != nil {
		return persistedState{}, err
	}
	// 本地 Vault 重建后 CA 可能变化；旧证书和旧信任束必须整体视为未 enrollment，
	// 否则 Agent 会持续用已不受后端信任的客户端证书重试 mTLS。
	if state.CertificatePEM != "" && !stateTrustBundleMatches(c.TrustBundle, c.StateDir) {
		state = persistedState{}
	}
	if state.CertificatePEM == "" {
		state, err = enroll(ctx, c)
		if err != nil {
			return persistedState{}, err
		}
	}
	return reconnectEnrolledState(ctx, c, state)
}

func enrollAndReconnect(ctx context.Context, c config) (persistedState, error) {
	state, err := enroll(ctx, c)
	if err != nil {
		return persistedState{}, err
	}
	return reconnectEnrolledState(ctx, c, state)
}

func reconnectEnrolledState(ctx context.Context, c config, state persistedState) (persistedState, error) {
	snapshot, err := reconnect(ctx, c, state)
	if err != nil {
		return persistedState{}, err
	}
	if err := persistGeneration(c.StateDir, state, snapshot.Generation); err != nil {
		return persistedState{}, err
	}
	state.Generation = snapshot.Generation
	return state, nil
}

func stateTrustBundleMatches(configuredPath, stateDir string) bool {
	// #nosec G304 -- 路径来自 Backend-owned Agent 配置与其私有状态目录。
	configured, err := os.ReadFile(configuredPath)
	if err != nil {
		return false
	}
	// #nosec G304 -- stateDir 是 Agent 私有的持久化状态边界。
	persisted, err := os.ReadFile(filepath.Join(stateDir, "trust-bundle.pem"))
	if err != nil {
		return false
	}
	return bytes.Equal(configured, persisted)
}

func enroll(ctx context.Context, c config) (persistedState, error) {
	token, err := c.token()
	if err != nil {
		return persistedState{}, err
	}
	identity := stableSPIFFE(c.TargetID, c.AgentID)
	request, err := createCSR(identity)
	if err != nil {
		return persistedState{}, err
	}
	response, err := bootstrap(ctx, c, token, request.csrDER)
	if err != nil {
		return persistedState{}, err
	}
	if len(response.CertificateChainDER) == 0 {
		return persistedState{}, errors.New("bootstrap returned no certificate")
	}
	if err := validateCertificate(response.CertificateChainDER, request.key, identity); err != nil {
		return persistedState{}, fmt.Errorf("validate bootstrap certificate: %w", err)
	}
	trustPEM, err := os.ReadFile(c.TrustBundle)
	if err != nil {
		return persistedState{}, fmt.Errorf("read trust bundle: %w", err)
	}
	state := persistedState{TargetID: c.TargetID, AgentID: c.AgentID, Identity: identity, CertificatePEM: string(encodeChain(response.CertificateChainDER)), TrustBundleRef: response.TrustBundle.Reference, TrustVersion: response.TrustBundle.Version, ExpiresAt: response.ExpiresAt}
	if err := saveState(c.StateDir, state, request.keyPEM, trustPEM); err != nil {
		return persistedState{}, err
	}
	return state, nil
}

func persistGeneration(stateDir string, state persistedState, generation int64) error {
	if state.Generation > 0 {
		return nil
	}
	// #nosec G304 -- stateDir 是首次持久化密钥时使用的 Agent 所有目录。
	keyPEM, err := os.ReadFile(filepath.Join(stateDir, "key.pem"))
	if err != nil {
		return err
	}
	// #nosec G304 -- stateDir 是首次持久化信任包时使用的 Agent 所有目录。
	trustPEM, err := os.ReadFile(filepath.Join(stateDir, "trust-bundle.pem"))
	if err != nil {
		return err
	}
	state.Generation = generation
	return saveState(stateDir, state, keyPEM, trustPEM)
}

func createCSR(identity string) (certificateRequest, error) {
	key, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return certificateRequest{}, err
	}
	uri, err := url.Parse(identity)
	if err != nil {
		return certificateRequest{}, err
	}
	tpl := &x509.CertificateRequest{Subject: pkix.Name{CommonName: identity}, URIs: []*url.URL{uri}}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, tpl, key)
	if err != nil {
		return certificateRequest{}, err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return certificateRequest{}, err
	}
	return certificateRequest{key: key, csrDER: csrDER, keyPEM: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})}, nil
}

func bootstrap(ctx context.Context, c config, token string, csr []byte) (bootstrapResponse, error) {
	body, err := json.Marshal(struct {
		BootstrapToken string `json:"bootstrap_token"`
		CSRDER         []byte `json:"csr_der"`
	}{token, csr})
	if err != nil {
		return bootstrapResponse{}, err
	}
	ca, err := os.ReadFile(c.BootstrapCA)
	if err != nil {
		return bootstrapResponse{}, fmt.Errorf("read bootstrap CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return bootstrapResponse{}, errors.New("bootstrap CA has no certificates")
	}
	client := newTLSClient(pool, nil, c.BootstrapURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BootstrapURL, "/")+bootstrapPath, strings.NewReader(string(body)))
	if err != nil {
		return bootstrapResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return bootstrapResponse{}, fmt.Errorf("bootstrap request: %w", err)
	}
	defer closeResponse(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return bootstrapResponse{}, fmt.Errorf("bootstrap returned HTTP %d", resp.StatusCode)
	}
	var result bootstrapResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&result); err != nil {
		return bootstrapResponse{}, fmt.Errorf("decode bootstrap response: %w", err)
	}
	return result, nil
}

func reconnect(ctx context.Context, c config, state persistedState) (ledgerSnapshot, error) {
	client, err := newMTLSClient(c, state)
	if err != nil {
		return ledgerSnapshot{}, err
	}
	snapshot, err := fetchLedgerSnapshot(ctx, client, c.AgentURL)
	if err != nil {
		return ledgerSnapshot{}, err
	}
	if state.Generation > 0 && snapshot.Generation != state.Generation {
		return ledgerSnapshot{}, fmt.Errorf("ledger generation %d does not match state generation %d", snapshot.Generation, state.Generation)
	}
	if err := submitTelemetry(ctx, client, c.AgentURL, snapshot); err != nil {
		return ledgerSnapshot{}, err
	}
	return snapshot, nil
}

func newMTLSClient(c config, state persistedState) (*http.Client, error) {
	keyPEM, err := os.ReadFile(filepath.Join(c.StateDir, "key.pem"))
	if err != nil {
		return nil, err
	}
	// Backend delivery 是当前信任束 authority；状态副本只用于审计与恢复，轮换后可能过期。
	trustPEM, err := os.ReadFile(c.TrustBundle)
	if err != nil {
		return nil, fmt.Errorf("read configured trust bundle: %w", err)
	}
	cert, err := tls.X509KeyPair([]byte(state.CertificatePEM), keyPEM)
	if err != nil {
		return nil, fmt.Errorf("load client certificate: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(trustPEM) {
		return nil, errors.New("configured trust bundle has no certificates")
	}
	return newTLSClient(pool, []tls.Certificate{cert}, c.AgentURL), nil
}

func fetchLedgerSnapshot(ctx context.Context, client *http.Client, agentURL string) (ledgerSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(agentURL, "/")+ledgerSnapshotPath, nil)
	if err != nil {
		return ledgerSnapshot{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return ledgerSnapshot{}, fmt.Errorf("ledger snapshot: %w", err)
	}
	defer closeResponse(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ledgerSnapshot{}, fmt.Errorf("ledger snapshot returned HTTP %d", resp.StatusCode)
	}
	var snapshot ledgerSnapshot
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&snapshot); err != nil {
		return ledgerSnapshot{}, err
	}
	return snapshot, nil
}

func submitTelemetry(ctx context.Context, client *http.Client, agentURL string, snapshot ledgerSnapshot) error {
	report := struct {
		SnapshotID            string `json:"snapshot_id"`
		SnapshotDigest        string `json:"snapshot_digest"`
		ObservedAt            string `json:"observed_at"`
		ExpiresAt             string `json:"expires_at"`
		Available             bool   `json:"available"`
		ImplementationVersion string `json:"implementation_version"`
		Diagnostic            string `json:"diagnostic"`
	}{snapshot.SnapshotID, snapshot.SnapshotDigest, snapshot.ObservedAt.UTC().Format(time.RFC3339), snapshot.ExpiresAt.UTC().Format(time.RFC3339), snapshot.Available, version, ""}
	body, err := json.Marshal(report)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(agentURL, "/")+telemetryPath, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("telemetry receipt: %w", err)
	}
	defer closeResponse(resp.Body)
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("telemetry receipt returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func newTLSClient(pool *x509.CertPool, certificates []tls.Certificate, rawURL string) *http.Client {
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, Certificates: certificates, RootCAs: pool, ServerName: serverName(rawURL)}}
	return &http.Client{Timeout: requestTimeout, Transport: transport}
}

func closeResponse(body io.Closer) { _ = body.Close() }

func validateCertificate(chain [][]byte, key *rsa.PrivateKey, identity string) error {
	leaf, err := x509.ParseCertificate(chain[0])
	if err != nil {
		return err
	}
	if len(leaf.URIs) != 1 || leaf.URIs[0].String() != identity {
		return errors.New("certificate SPIFFE URI does not match configured identity")
	}
	certificateKey, ok := leaf.PublicKey.(*rsa.PublicKey)
	if !ok || certificateKey.N.Cmp(key.N) != 0 || certificateKey.E != key.E {
		return errors.New("certificate public key does not match CSR key")
	}
	return nil
}

func encodeChain(chain [][]byte) []byte {
	var out []byte
	for _, der := range chain {
		out = append(out, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
	}
	return out
}

func serverName(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func safeAgentError(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "operation cancelled or timed out"
	}
	return "runtime agent operation failed"
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "graft-docker-runtime-agent:", safeAgentError(err))
	os.Exit(1)
}
