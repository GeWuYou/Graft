//go:build conformance

package runtimetarget

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

const conformanceDeliveryProtocol = "graft.delivery-receipt.v1"

// DockerRuntimeAgentConformanceDriver 仅为 compose fixture 协调既有 Runtime Target 生命周期 authority。
// 它不属于生产服务注册面，也不决定 Runtime Target 的 provider 或连接事实。
type DockerRuntimeAgentConformanceDriver struct {
	repository *store.SQLRepository
	targets    dockerRuntimeAgentFixtureTargetAuthority
	enrollment moduleapi.AgentEnrollmentAuthority
	delivery   moduleapi.AgentDeliveryAuthority
	bindings   moduleapi.RuntimeTargetAgentBindingReader
	issuer     moduleapi.AgentCertificateIssuer
}

// dockerRuntimeAgentFixtureTargetAuthority 保持 fixture 场景与 Runtime Target 发现规则分离。
// 它复用模块既有的本地 Docker discovery，而不是让 CLI 或 fixture 决定 provider、端点或目标身份。
type dockerRuntimeAgentFixtureTargetAuthority struct{ repository *store.SQLRepository }

func (a dockerRuntimeAgentFixtureTargetAuthority) Resolve(ctx context.Context) (store.Target, error) {
	if a.repository == nil {
		return store.Target{}, errors.New("fixture runtime target authority is unavailable")
	}
	if err := discoverLocalDocker(ctx, a.repository); err != nil {
		return store.Target{}, fmt.Errorf("discover fixture runtime target: %w", err)
	}
	target, err := a.repository.FindSystemLocalDocker(ctx)
	if err != nil {
		return store.Target{}, fmt.Errorf("read fixture runtime target: %w", err)
	}
	if !target.Availability {
		return store.Target{}, errors.New("fixture runtime target is unavailable")
	}
	return target, nil
}

// DockerRuntimeAgentFixtureScenario 是 fixture 交给 Runtime Target 的非秘密场景输入。
type DockerRuntimeAgentFixtureScenario struct {
	AgentID               string
	ImageDigest           string
	AgentVersion          string
	EnrollmentRef         string
	ExpectedAutomationID  string
	DockerInstallationRef string
	DockerSecretRef       string
	BootstrapMaterialFile string
	AgentConfigFile       string
	BootstrapURL          string
	AgentURL              string
	BootstrapCAFile       string
	TrustBundleFile       string
	AgentSecretFile       string
}

// DockerRuntimeAgentConformanceMaterial 是只写入 fixture secret mount 的一次性 bootstrap 材料。
type DockerRuntimeAgentConformanceMaterial struct {
	TargetID       int64  `json:"target_id"`
	AgentID        string `json:"agent_id"`
	BootstrapToken string `json:"bootstrap_token"`
}

// DockerRuntimeAgentConformanceEvidence 是可安全输出到 fixture 日志的非秘密生命周期证据。
type DockerRuntimeAgentConformanceEvidence struct {
	IdentityID           string `json:"identity_id"`
	TargetID             int64  `json:"target_id"`
	AgentID              string `json:"agent_id"`
	Generation           int64  `json:"generation"`
	CertificateSerial    string `json:"certificate_serial"`
	PublicKeyFingerprint string `json:"public_key_fingerprint"`
	ProviderID           string `json:"provider_id"`
	BuilderScope         string `json:"builder_scope"`
	CapabilityProfile    string `json:"capability_profile"`
	CapabilityVersion    string `json:"capability_version"`
	LedgerProvenance     string `json:"ledger_provenance"`
	LedgerReceiptCount   int64  `json:"ledger_receipt_count"`
}

// NewDockerRuntimeAgentConformanceDriver 构造仅用于 conformance 的 Runtime Target 生命周期协调器。
func NewDockerRuntimeAgentConformanceDriver(db *sql.DB, pepper *config.EnrollmentPepperProvider, issuer moduleapi.AgentCertificateIssuer) (*DockerRuntimeAgentConformanceDriver, error) {
	if db == nil || issuer == nil {
		return nil, errors.New("docker runtime agent conformance dependencies are unavailable")
	}
	repository := store.NewSQLRepository(db)
	return &DockerRuntimeAgentConformanceDriver{
		repository: repository,
		targets:    dockerRuntimeAgentFixtureTargetAuthority{repository: repository},
		enrollment: newRuntimeTargetAgentEnrollmentAuthority(repository, nil),
		delivery:   newRuntimeTargetAgentDeliveryAuthority(repository, pepper),
		bindings:   runtimeTargetAgentBindingReader{repository: repository},
		issuer:     issuer,
	}, nil
}

// Prepare 创建 fixture 所需的 pending generation 和已确认交付回执；回执绝不激活 generation。
func (d *DockerRuntimeAgentConformanceDriver) Prepare(ctx context.Context, scenario DockerRuntimeAgentFixtureScenario) (DockerRuntimeAgentConformanceEvidence, error) {
	if d == nil || d.repository == nil || d.targets.repository == nil || d.enrollment == nil || d.delivery == nil || d.bindings == nil || d.issuer == nil || !validDockerRuntimeAgentFixtureScenario(scenario) {
		return DockerRuntimeAgentConformanceEvidence{}, errors.New("docker runtime agent conformance scenario is invalid")
	}
	target, err := d.targets.Resolve(ctx)
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, err
	}
	if target.ID > uint64(^uint64(0)>>1) {
		return DockerRuntimeAgentConformanceEvidence{}, errors.New("fixture runtime target ID overflows signed range")
	}
	targetID := int64(target.ID)
	if target.Provider != "docker" {
		return DockerRuntimeAgentConformanceEvidence{}, errors.New("fixture runtime target does not support Docker runtime enrollment")
	}
	now := time.Now().UTC()
	trustBundle, err := d.issuer.ReadTrustBundle(ctx, moduleapi.TrustBundleRequest{TargetID: targetID, ProviderID: target.Provider, Generation: 1})
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, fmt.Errorf("read fixture trust bundle: %w", err)
	}
	enrollment, err := d.enrollment.CreateEnrollment(ctx, moduleapi.AgentEnrollmentRequest{
		TargetID: targetID, AgentID: scenario.AgentID, ProviderID: target.Provider, BuilderScope: "docker-runtime-agent-conformance",
		CapabilityProfile: "oci-build", CapabilityVersion: "docker/v1", Capabilities: []string{"oci-build"}, RuntimeProtocol: "runtime/v1", ImageDigest: scenario.ImageDigest,
		AgentVersion: scenario.AgentVersion, EnrollmentRef: scenario.EnrollmentRef, TrustBundle: trustBundle, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, fmt.Errorf("create fixture enrollment: %w", err)
	}
	grant, err := d.delivery.CreateDeliveryGrant(ctx, moduleapi.AgentDeliveryGrantRequest{TargetID: targetID, AgentID: scenario.AgentID, Generation: enrollment.Generation, ExpectedAutomationID: scenario.ExpectedAutomationID, DockerInstallationRef: scenario.DockerInstallationRef, ExpiresAt: now.Add(15 * time.Minute)})
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, fmt.Errorf("create fixture delivery grant: %w", err)
	}
	actor := moduleapi.DeliveryActor{ID: scenario.ExpectedAutomationID, Type: "fixture"}
	handoff, err := d.delivery.HandoffDeliveryGrant(ctx, actor, moduleapi.AgentDeliveryHandoffRequest{GrantID: grant.GrantID})
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, fmt.Errorf("handoff fixture delivery grant: %w", err)
	}
	if err := writeDockerRuntimeAgentConformanceMaterial(scenario.BootstrapMaterialFile, DockerRuntimeAgentConformanceMaterial{TargetID: targetID, AgentID: scenario.AgentID, BootstrapToken: handoff.BootstrapToken}); err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, err
	}
	if err := writeDockerRuntimeAgentConformanceConfig(scenario, targetID); err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, err
	}
	if _, err := d.delivery.RecordDeliveryReceipt(ctx, actor, moduleapi.AgentDeliveryReceiptRequest{GrantID: grant.GrantID, ReceiptID: "fixture-" + grant.GrantID, ProtocolVersion: conformanceDeliveryProtocol, HandoffID: handoff.HandoffID, AssertedDeliveredAt: now, DockerInstallationRef: scenario.DockerInstallationRef, DockerSecretRef: scenario.DockerSecretRef, PayloadFingerprint: conformancePayloadFingerprint(scenario, targetID)}); err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, fmt.Errorf("record fixture delivery receipt: %w", err)
	}
	binding, err := d.bindings.ReadAgentBinding(ctx, targetID, scenario.AgentID)
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, fmt.Errorf("read pending fixture binding: %w", err)
	}
	if binding.Status != moduleapi.RuntimeTargetAgentStatusPending || binding.CertificateSerial != "" {
		return DockerRuntimeAgentConformanceEvidence{}, errors.New("delivery receipt activated fixture generation")
	}
	return conformanceEvidence(binding, 0), nil
}

// VerifyBootstrap 断言首次 bootstrap 已通过 Vault PKI 和 mTLS 激活，但不把重启行为混入该断言。
func (d *DockerRuntimeAgentConformanceDriver) VerifyBootstrap(ctx context.Context, targetID int64, agentID string) (DockerRuntimeAgentConformanceEvidence, error) {
	evidence, err := d.readActiveEvidence(ctx, targetID, agentID)
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, err
	}
	if evidence.LedgerReceiptCount < 1 {
		return DockerRuntimeAgentConformanceEvidence{}, errors.New("bootstrap has no accepted ledger receipt")
	}
	return evidence, nil
}

// VerifyRestart 断言持久化身份被复用、没有新 generation，并且重连产生新的 ledger receipt。
func (d *DockerRuntimeAgentConformanceDriver) VerifyRestart(ctx context.Context, targetID int64, agentID, identityID string, generation, previousReceiptCount int64) (DockerRuntimeAgentConformanceEvidence, error) {
	evidence, err := d.readActiveEvidence(ctx, targetID, agentID)
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, err
	}
	if evidence.IdentityID != strings.TrimSpace(identityID) || evidence.Generation != generation {
		return DockerRuntimeAgentConformanceEvidence{}, errors.New("agent restart created a new enrollment identity")
	}
	if evidence.LedgerReceiptCount <= previousReceiptCount {
		return DockerRuntimeAgentConformanceEvidence{}, errors.New("agent restart has no new ledger receipt")
	}
	return evidence, nil
}

func (d *DockerRuntimeAgentConformanceDriver) readActiveEvidence(ctx context.Context, targetID int64, agentID string) (DockerRuntimeAgentConformanceEvidence, error) {
	binding, err := d.bindings.ReadAgentBinding(ctx, targetID, agentID)
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, fmt.Errorf("read fixture binding: %w", err)
	}
	if binding.Status != moduleapi.RuntimeTargetAgentStatusActive || binding.CertificateSerial == "" || binding.PublicKeyFingerprint == "" {
		return DockerRuntimeAgentConformanceEvidence{}, errors.New("fixture generation is not active with certificate evidence")
	}
	count, err := d.repository.CountAgentLedgerReceipts(ctx, targetID, agentID, binding.Generation)
	if err != nil {
		return DockerRuntimeAgentConformanceEvidence{}, fmt.Errorf("read fixture ledger receipts: %w", err)
	}
	return conformanceEvidence(binding, count), nil
}

func conformanceEvidence(binding moduleapi.RuntimeTargetAgentBinding, receiptCount int64) DockerRuntimeAgentConformanceEvidence {
	return DockerRuntimeAgentConformanceEvidence{
		IdentityID: binding.IdentityID, TargetID: binding.TargetID, AgentID: binding.AgentID, Generation: binding.Generation,
		CertificateSerial: binding.CertificateSerial, PublicKeyFingerprint: binding.PublicKeyFingerprint,
		ProviderID: binding.ProviderID, BuilderScope: binding.BuilderScope, CapabilityProfile: binding.CapabilityProfile,
		CapabilityVersion: binding.CapabilityVersion, LedgerProvenance: "runtime-target-controlled-execution-ledger",
		LedgerReceiptCount: receiptCount,
	}
}

func validDockerRuntimeAgentFixtureScenario(s DockerRuntimeAgentFixtureScenario) bool {
	return strings.TrimSpace(s.AgentID) != "" && strings.TrimSpace(s.ImageDigest) != "" && strings.TrimSpace(s.AgentVersion) != "" && strings.TrimSpace(s.EnrollmentRef) != "" && strings.TrimSpace(s.ExpectedAutomationID) != "" && strings.TrimSpace(s.DockerInstallationRef) != "" && strings.TrimSpace(s.DockerSecretRef) != "" && strings.TrimSpace(s.BootstrapMaterialFile) != "" && strings.TrimSpace(s.AgentConfigFile) != "" && validDockerRuntimeAgentFixtureHTTPSURL(s.BootstrapURL) && validDockerRuntimeAgentFixtureHTTPSURL(s.AgentURL) && strings.TrimSpace(s.BootstrapCAFile) != "" && strings.TrimSpace(s.TrustBundleFile) != "" && strings.TrimSpace(s.AgentSecretFile) != ""
}

func validDockerRuntimeAgentFixtureHTTPSURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}

func conformancePayloadFingerprint(s DockerRuntimeAgentFixtureScenario, targetID int64) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{fmt.Sprint(targetID), s.AgentID, s.ImageDigest, s.AgentVersion, s.EnrollmentRef}, "\n")))
	return hex.EncodeToString(digest[:])
}

func writeDockerRuntimeAgentConformanceMaterial(path string, material DockerRuntimeAgentConformanceMaterial) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create fixture secret directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(material.BootstrapToken+"\n"), 0o600); err != nil {
		return fmt.Errorf("write fixture bootstrap material: %w", err)
	}
	return nil
}

func writeDockerRuntimeAgentConformanceConfig(s DockerRuntimeAgentFixtureScenario, targetID int64) error {
	if err := os.MkdirAll(filepath.Dir(s.AgentConfigFile), 0o755); err != nil {
		return fmt.Errorf("create fixture agent config directory: %w", err)
	}
	encoded, err := json.Marshal(struct {
		BootstrapURL      string   `json:"bootstrap_url"`
		AgentURL          string   `json:"agent_url"`
		TargetID          int64    `json:"target_id"`
		AgentID           string   `json:"agent_id"`
		SecretFile        string   `json:"secret_file"`
		BootstrapCA       string   `json:"bootstrap_ca_file"`
		TrustBundle       string   `json:"trust_bundle_file"`
		DockerSocket      string   `json:"docker_socket"`
		ProviderID        string   `json:"provider_id"`
		Capabilities      []string `json:"capabilities"`
		CapabilityVersion string   `json:"capability_version"`
	}{s.BootstrapURL, s.AgentURL, targetID, s.AgentID, s.AgentSecretFile, s.BootstrapCAFile, s.TrustBundleFile, "/var/run/docker.sock", "docker", []string{"oci-build"}, "runtime/v1"})
	if err != nil {
		return errors.New("encode fixture agent configuration")
	}
	if err := os.WriteFile(s.AgentConfigFile, encoded, 0o644); err != nil {
		return fmt.Errorf("write fixture agent configuration: %w", err)
	}
	return nil
}
