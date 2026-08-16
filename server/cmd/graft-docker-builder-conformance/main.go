//go:build conformance

// Package main 提供仅供 docker-builder-agent fixture 使用的 conformance 进程入口。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"graft/server/internal/config"
	"graft/server/internal/database"
	"graft/server/internal/moduleapi"
	credentialvault "graft/server/modules/credential-vault"
	runtimetarget "graft/server/modules/runtime-target"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "docker-builder-agent conformance failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output *os.File) error {
	phase, scenario, baseline, err := parseArguments(args)
	if err != nil {
		return err
	}
	if phase != "prepare" && baseline.targetID == 0 {
		requestedAgentID, agentIDExplicit := baseline.requestedAgentID, baseline.agentIDExplicit
		baseline, err = readVerificationBaseline(envOr("GRAFT_CONFORMANCE_EVIDENCE_FILE", "/conformance/agent-config/conformance-evidence.json"))
		if err != nil {
			return err
		}
		baseline.requestedAgentID, baseline.agentIDExplicit = requestedAgentID, agentIDExplicit
	}
	if phase != "prepare" && baseline.agentID != "" {
		if baseline.agentIDExplicit && baseline.requestedAgentID != baseline.agentID {
			return errors.New("agent-id does not match conformance baseline")
		}
		scenario.AgentID = baseline.agentID
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load conformance config: %w", err)
	}
	resources, err := database.Open(cfg.Database)
	if err != nil {
		return fmt.Errorf("open conformance database: %w", err)
	}
	defer func() { _ = database.Close(resources) }()
	driver, err := newDriver(resources.SQL, cfg)
	if err != nil {
		return err
	}
	var evidence runtimetarget.DockerBuilderAgentConformanceEvidence
	switch phase {
	case "prepare":
		evidence, err = driver.Prepare(ctx, scenario)
	case "verify-bootstrap":
		evidence, err = driver.VerifyBootstrap(ctx, baseline.targetID, scenario.AgentID)
	case "verify-restart":
		evidence, err = driver.VerifyRestart(ctx, baseline.targetID, scenario.AgentID, baseline.identityID, baseline.generation, baseline.receiptCount)
	default:
		return errors.New("unsupported conformance phase")
	}
	if err != nil {
		return err
	}
	if phase == "verify-restart" {
		if err := validateRestartEvidence(baseline, evidence); err != nil {
			return err
		}
	}
	if err := writeVerificationBaseline(envOr("GRAFT_CONFORMANCE_EVIDENCE_FILE", "/conformance/agent-config/conformance-evidence.json"), evidence); err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(evidence)
}

func newDriver(db *sql.DB, cfg *config.Config) (*runtimetarget.DockerBuilderAgentConformanceDriver, error) {
	if cfg == nil {
		return nil, errors.New("conformance configuration is unavailable")
	}
	pepper, err := config.NewEnrollmentPepperProvider(cfg.EnrollmentSecurity)
	if err != nil {
		return nil, fmt.Errorf("load enrollment pepper: %w", err)
	}
	issuanceStore, err := credentialvault.NewSQLIssuanceStateStore(db)
	if err != nil {
		return nil, fmt.Errorf("create Vault issuance state store: %w", err)
	}
	issuer, err := credentialvault.NewVaultPKIClient(cfg.CredentialVault, issuanceStore)
	if err != nil {
		return nil, fmt.Errorf("create Vault PKI adapter: %w", err)
	}
	return runtimetarget.NewDockerBuilderAgentConformanceDriver(db, pepper, moduleapi.AgentCertificateIssuer(issuer))
}

type verificationBaseline struct {
	targetID          int64
	identityID        string
	agentID           string
	generation        int64
	receiptCount      int64
	builderScope      string
	capabilityProfile string
	capabilityVersion string
	requestedAgentID  string
	agentIDExplicit   bool
}

func parseArguments(args []string) (string, runtimetarget.DockerBuilderAgentFixtureScenario, verificationBaseline, error) {
	if len(args) > 0 && args[0] != "" && args[0][0] != '-' {
		args = append([]string{"--phase", args[0]}, args[1:]...)
	}
	flags := flag.NewFlagSet("graft-docker-builder-conformance", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	phase := flags.String("phase", "", "prepare, verify-bootstrap, or verify-restart")
	agentID := flags.String("agent-id", envOr("GRAFT_CONFORMANCE_AGENT_ID", "docker-builder-agent"), "fixture agent identity")
	imageDigest := flags.String("image-digest", envOr("GRAFT_CONFORMANCE_IMAGE_DIGEST", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "fixture OCI image digest")
	agentVersion := flags.String("agent-version", envOr("GRAFT_CONFORMANCE_AGENT_VERSION", "conformance"), "fixture agent version")
	enrollmentRef := flags.String("enrollment-ref", envOr("GRAFT_CONFORMANCE_ENROLLMENT_REF", "docker-builder-agent-conformance"), "fixture enrollment reference")
	automationID := flags.String("automation-id", envOr("GRAFT_CONFORMANCE_AUTOMATION_ID", "docker-builder-agent-conformance"), "fixture delivery actor")
	dockerInstallationRef := flags.String("docker-installation-ref", envOr("GRAFT_CONFORMANCE_DOCKER_INSTALLATION_REF", "docker-builder-agent-conformance"), "fixture Docker installation reference")
	dockerSecretRef := flags.String("docker-secret-ref", envOr("GRAFT_CONFORMANCE_DOCKER_SECRET_REF", "docker-builder-agent-bootstrap"), "fixture Docker secret reference")
	bootstrapMaterialFile := flags.String("bootstrap-material-file", envOr("GRAFT_CONFORMANCE_BOOTSTRAP_MATERIAL_FILE", "/conformance/agent-bootstrap/bootstrap-token"), "fixture bootstrap material path")
	agentConfigFile := flags.String("agent-config-file", envOr("GRAFT_CONFORMANCE_AGENT_CONFIG_FILE", "/conformance/agent-config/agent.json"), "fixture agent configuration path")
	bootstrapURL := flags.String("bootstrap-url", envOr("GRAFT_CONFORMANCE_BOOTSTRAP_URL", "https://backend:8443"), "fixture bootstrap listener URL")
	agentURL := flags.String("agent-url", envOr("GRAFT_CONFORMANCE_AGENT_URL", "https://backend:8444"), "fixture agent mTLS listener URL")
	bootstrapCAFile := flags.String("bootstrap-ca-file", envOr("GRAFT_CONFORMANCE_BOOTSTRAP_CA_FILE", "/run/graft-agent-trust/ca.pem"), "Agent-visible bootstrap CA path")
	trustBundleFile := flags.String("trust-bundle-file", envOr("GRAFT_CONFORMANCE_TRUST_BUNDLE_FILE", "/run/graft-agent-trust/ca.pem"), "Agent-visible trust bundle path")
	agentSecretFile := flags.String("agent-secret-file", envOr("GRAFT_CONFORMANCE_AGENT_SECRET_FILE", "/run/graft-bootstrap/bootstrap-token"), "Agent-visible bootstrap secret path")
	targetID := flags.Int64("target-id", 0, "previous target ID")
	identityID := flags.String("identity-id", "", "previous identity ID")
	generation := flags.Int64("generation", 0, "previous generation")
	receiptCount := flags.Int64("receipt-count", 0, "previous receipt count")
	if err := flags.Parse(args); err != nil {
		return "", runtimetarget.DockerBuilderAgentFixtureScenario{}, verificationBaseline{}, err
	}
	agentIDExplicit := false
	flags.Visit(func(flag *flag.Flag) {
		agentIDExplicit = agentIDExplicit || flag.Name == "agent-id"
	})
	if *phase == "verify-first" {
		*phase = "verify-bootstrap"
	}
	scenario := runtimetarget.DockerBuilderAgentFixtureScenario{AgentID: *agentID, ImageDigest: *imageDigest, AgentVersion: *agentVersion, EnrollmentRef: *enrollmentRef, ExpectedAutomationID: *automationID, DockerInstallationRef: *dockerInstallationRef, DockerSecretRef: *dockerSecretRef, BootstrapMaterialFile: *bootstrapMaterialFile, AgentConfigFile: *agentConfigFile, BootstrapURL: *bootstrapURL, AgentURL: *agentURL, BootstrapCAFile: *bootstrapCAFile, TrustBundleFile: *trustBundleFile, AgentSecretFile: *agentSecretFile}
	baseline := verificationBaseline{targetID: *targetID, identityID: *identityID, generation: *generation, receiptCount: *receiptCount, requestedAgentID: strings.TrimSpace(*agentID), agentIDExplicit: agentIDExplicit}
	if *phase != "prepare" && *phase != "verify-bootstrap" && *phase != "verify-restart" {
		return "", runtimetarget.DockerBuilderAgentFixtureScenario{}, verificationBaseline{}, errors.New("phase must be prepare, verify-bootstrap, or verify-restart")
	}
	if scenario.AgentID == "" {
		return "", runtimetarget.DockerBuilderAgentFixtureScenario{}, verificationBaseline{}, errors.New("agent-id is required")
	}
	if *phase == "prepare" && (scenario.ImageDigest == "" || scenario.AgentVersion == "" || scenario.EnrollmentRef == "" || scenario.ExpectedAutomationID == "" || scenario.DockerInstallationRef == "" || scenario.DockerSecretRef == "" || scenario.BootstrapMaterialFile == "" || scenario.AgentConfigFile == "" || scenario.BootstrapURL == "" || scenario.AgentURL == "" || scenario.BootstrapCAFile == "" || scenario.TrustBundleFile == "" || scenario.AgentSecretFile == "") {
		return "", runtimetarget.DockerBuilderAgentFixtureScenario{}, verificationBaseline{}, errors.New("prepare requires fixture scenario fields")
	}
	if *phase == "verify-bootstrap" && baseline.targetID != 0 && baseline.targetID < 1 {
		return "", runtimetarget.DockerBuilderAgentFixtureScenario{}, verificationBaseline{}, errors.New("verify-bootstrap requires target-id")
	}
	if *phase == "verify-restart" && baseline.targetID != 0 && (baseline.identityID == "" || baseline.generation < 1 || baseline.receiptCount < 1) {
		return "", runtimetarget.DockerBuilderAgentFixtureScenario{}, verificationBaseline{}, errors.New("verify-restart requires bootstrap evidence")
	}
	return *phase, scenario, baseline, nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func readVerificationBaseline(path string) (verificationBaseline, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return verificationBaseline{}, fmt.Errorf("read conformance baseline: %w", err)
	}
	var evidence runtimetarget.DockerBuilderAgentConformanceEvidence
	if err := json.Unmarshal(contents, &evidence); err != nil {
		return verificationBaseline{}, errors.New("decode conformance baseline")
	}
	if evidence.TargetID < 1 || strings.TrimSpace(evidence.IdentityID) == "" || strings.TrimSpace(evidence.AgentID) == "" || evidence.Generation < 1 || strings.TrimSpace(evidence.ProviderID) != "docker" || strings.TrimSpace(evidence.BuilderScope) == "" || strings.TrimSpace(evidence.CapabilityProfile) == "" || strings.TrimSpace(evidence.CapabilityVersion) == "" || evidence.LedgerProvenance != "runtime-target-controlled-execution-ledger" || evidence.LedgerReceiptCount < 0 {
		return verificationBaseline{}, errors.New("conformance baseline is invalid")
	}
	return verificationBaseline{targetID: evidence.TargetID, identityID: evidence.IdentityID, agentID: strings.TrimSpace(evidence.AgentID), generation: evidence.Generation, receiptCount: evidence.LedgerReceiptCount, builderScope: strings.TrimSpace(evidence.BuilderScope), capabilityProfile: strings.TrimSpace(evidence.CapabilityProfile), capabilityVersion: strings.TrimSpace(evidence.CapabilityVersion)}, nil
}

func validateRestartEvidence(baseline verificationBaseline, evidence runtimetarget.DockerBuilderAgentConformanceEvidence) error {
	if strings.TrimSpace(evidence.BuilderScope) == "" || strings.TrimSpace(evidence.CapabilityProfile) == "" || strings.TrimSpace(evidence.CapabilityVersion) == "" {
		return errors.New("restart conformance evidence is missing builder capability metadata")
	}
	if evidence.BuilderScope != baseline.builderScope || evidence.CapabilityProfile != baseline.capabilityProfile || evidence.CapabilityVersion != baseline.capabilityVersion {
		return errors.New("restart conformance evidence does not match bootstrap builder capability metadata")
	}
	return nil
}

func writeVerificationBaseline(path string, evidence runtimetarget.DockerBuilderAgentConformanceEvidence) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create conformance evidence directory: %w", err)
	}
	contents, err := json.Marshal(evidence)
	if err != nil {
		return errors.New("encode conformance evidence")
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return fmt.Errorf("write conformance evidence: %w", err)
	}
	return nil
}
