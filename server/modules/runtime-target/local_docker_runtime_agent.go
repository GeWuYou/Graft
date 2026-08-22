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

// LocalDockerRuntimeAgentDelivery 是仅供开发编排调用的 Backend-owned 交付输入。
// 它的文件路径必须位于调用方隔离的本地开发目录，不能指向部署路径。
type LocalDockerRuntimeAgentDelivery struct {
	AgentID, ImageDigest, AgentVersion, EnrollmentRef, AutomationID  string
	BootstrapURL, AgentURL, BootstrapTokenFile, ConfigFile, StateDir string
	BootstrapCAFile, TrustBundleFile                                 string
}

const localDeliverySecretDirPerm = 0o700
const localDeliverySecretFilePerm = 0o600
const localDeliveryGrantTTL = 15 * time.Minute

var localDockerRuntimeAgentCapabilities = []string{"oci-build", "compose_execution", "container_execution", "update_controller"}

// PrepareLocalDockerRuntimeAgent 使用 Runtime Target 的既有 authority 生成一次本地 Agent 交付。
// 该函数只适用于本地开发编排；调用方负责限制环境、路径和目录权限。
//
//nolint:cyclop,gocognit,gocyclo // enrollment、delivery 与受限文件交付必须保持在同一 authority 协调边界。
func PrepareLocalDockerRuntimeAgent(ctx context.Context, db *sql.DB, pepper *config.EnrollmentPepperProvider, issuer moduleapi.AgentCertificateIssuer, input LocalDockerRuntimeAgentDelivery) error {
	if db == nil || pepper == nil || issuer == nil || !validLocalDockerRuntimeAgentDelivery(input) {
		return errors.New("local Docker Runtime Agent delivery is invalid")
	}
	repository := store.NewSQLRepository(db)
	if err := discoverLocalDocker(ctx, repository); err != nil {
		return fmt.Errorf("discover local Docker runtime target: %w", err)
	}
	target, err := repository.FindSystemLocalDocker(ctx)
	if err != nil || !target.Availability || target.Provider != "docker" || target.ID > uint64(^uint64(0)>>1) {
		return errors.New("local Docker runtime target is unavailable")
	}
	targetID := int64(target.ID)
	bindings := runtimeTargetAgentBindingReader{repository: repository}
	binding, bindingErr := bindings.ReadAgentBinding(ctx, targetID, input.AgentID)
	now := time.Now().UTC()
	trustBundle, err := issuer.ReadTrustBundle(ctx, moduleapi.TrustBundleRequest{TargetID: targetID, ProviderID: target.Provider, Generation: binding.Generation + 1})
	if err != nil {
		return fmt.Errorf("read local agent trust bundle: %w", err)
	}
	generation := int64(0)
	//nolint:nestif // 本地交付必须在同一 authority 边界内区分活动、待激活和漂移恢复。
	if bindingErr == nil && binding.Status == moduleapi.RuntimeTargetAgentStatusActive {
		if binding.TrustBundleVersion == trustBundle.Version && sameRuntimeAgentCapabilities(binding.Capabilities, localDockerRuntimeAgentCapabilities) && binding.CapabilityVersion == "docker/v1" {
			if _, err := os.Stat(input.ConfigFile); err == nil {
				return nil
			} else if errors.Is(err, os.ErrNotExist) {
				return errors.New("local Docker Runtime Agent binding is active but delivery config is missing; reset the Agent binding before delivering again")
			}
			return fmt.Errorf("read local Docker Runtime Agent delivery config: %w", err)
		}
		// 信任束或能力集合变化都必须创建新世代，避免旧证书在原地获得扩展能力。
		rotated, err := newRuntimeTargetAgentEnrollmentAuthority(repository, nil).RotateGeneration(ctx, moduleapi.AgentEnrollmentRotationRequest{
			IdentityID: binding.IdentityID, TargetID: targetID, AgentID: input.AgentID, ProviderID: target.Provider,
			BuilderScope: "docker-runtime-agent-local", CapabilityProfile: "oci-build", CapabilityVersion: "docker/v1",
			Capabilities: append([]string(nil), localDockerRuntimeAgentCapabilities...), RuntimeProtocol: "runtime/v1",
			EnrollmentRef: input.EnrollmentRef, TrustBundle: trustBundle, ExpiresAt: now.Add(time.Hour), Reason: "local_agent_contract_rotated",
		})
		if err != nil {
			return fmt.Errorf("rotate local agent enrollment after trust bundle change: %w", err)
		}
		generation = rotated.Generation
	} else if bindingErr != nil && !errors.Is(bindingErr, store.ErrAgentTrustNotFound) {
		return fmt.Errorf("read local Docker Runtime Agent binding: %w", bindingErr)
	}
	if generation == 0 && shouldReusePendingLocalDockerRuntimeGeneration(binding, bindingErr, trustBundle) {
		generation = binding.Generation
	}
	if generation == 0 {
		enrollment, err := newRuntimeTargetAgentEnrollmentAuthority(repository, nil).CreateEnrollment(ctx, moduleapi.AgentEnrollmentRequest{
			TargetID: targetID, AgentID: input.AgentID, ProviderID: target.Provider, BuilderScope: "docker-runtime-agent-local",
			CapabilityProfile: "oci-build", CapabilityVersion: "docker/v1", Capabilities: append([]string(nil), localDockerRuntimeAgentCapabilities...), RuntimeProtocol: "runtime/v1", ImageDigest: input.ImageDigest,
			AgentVersion: input.AgentVersion, EnrollmentRef: input.EnrollmentRef, TrustBundle: trustBundle, ExpiresAt: now.Add(time.Hour),
		})
		if err != nil {
			return fmt.Errorf("create local agent enrollment: %w", err)
		}
		generation = enrollment.Generation
	}
	delivery := newRuntimeTargetAgentDeliveryAuthority(repository, pepper)
	if existing, err := repository.ReadLiveAgentDeliveryGrant(ctx, targetID, input.AgentID, generation, now); err == nil && existing.GenerationID > 0 {
		if existing.Status != "delivered" {
			return errors.New("local Docker Runtime Agent delivery is pending; reset the Agent binding before delivering again")
		}
		if existing.ExpectedAutomationID != input.AutomationID || existing.DockerInstallationRef != "docker:local" || existing.HandoffID == "" {
			return errors.New("local Docker Runtime Agent delivery grant does not match the local automation binding; reset the Agent binding before delivering again")
		}
		return writeLocalDockerRuntimeAgentFiles(input, targetID, deriveBootstrapToken(pepper.Pepper(), existing.GrantID))
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		return fmt.Errorf("read local agent delivery grant: %w", err)
	}
	grant, err := delivery.CreateDeliveryGrant(ctx, moduleapi.AgentDeliveryGrantRequest{TargetID: targetID, AgentID: input.AgentID, Generation: generation, ExpectedAutomationID: input.AutomationID, DockerInstallationRef: "docker:local", ExpiresAt: now.Add(localDeliveryGrantTTL)})
	if err != nil {
		return fmt.Errorf("create local agent delivery grant: %w", err)
	}
	actor := moduleapi.DeliveryActor{ID: input.AutomationID, Type: "development"}
	handoff, err := delivery.HandoffDeliveryGrant(ctx, actor, moduleapi.AgentDeliveryHandoffRequest{GrantID: grant.GrantID})
	if err != nil {
		return fmt.Errorf("handoff local agent delivery grant: %w", err)
	}
	fingerprint := localDockerRuntimeAgentFingerprint(input, targetID)
	if _, err := delivery.RecordDeliveryReceipt(ctx, actor, moduleapi.AgentDeliveryReceiptRequest{GrantID: grant.GrantID, ReceiptID: "local-" + grant.GrantID, ProtocolVersion: "graft.delivery-receipt.v1", HandoffID: handoff.HandoffID, AssertedDeliveredAt: now, DockerInstallationRef: "docker:local", DockerSecretRef: "local:delivery", PayloadFingerprint: fingerprint}); err != nil {
		return fmt.Errorf("record local agent delivery receipt: %w", err)
	}
	return writeLocalDockerRuntimeAgentFiles(input, targetID, handoff.BootstrapToken)
}

func shouldReusePendingLocalDockerRuntimeGeneration(binding moduleapi.RuntimeTargetAgentBinding, bindingErr error, trustBundle moduleapi.TrustBundleReference) bool {
	return bindingErr == nil && binding.Status == moduleapi.RuntimeTargetAgentStatusPending && binding.TrustBundleVersion == trustBundle.Version && binding.CapabilityVersion == "docker/v1" && sameRuntimeAgentCapabilities(binding.Capabilities, localDockerRuntimeAgentCapabilities)
}

func sameRuntimeAgentCapabilities(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func writeLocalDockerRuntimeAgentFiles(input LocalDockerRuntimeAgentDelivery, targetID int64, bootstrapToken string) error {
	if err := os.MkdirAll(filepath.Dir(input.BootstrapTokenFile), localDeliverySecretDirPerm); err != nil {
		return fmt.Errorf("create local agent bootstrap token directory: %w", err)
	}
	if err := os.WriteFile(input.BootstrapTokenFile, []byte(bootstrapToken+"\n"), localDeliverySecretFilePerm); err != nil {
		return fmt.Errorf("write local agent bootstrap token: %w", err)
	}
	configBytes, err := json.Marshal(struct {
		BootstrapURL      string   `json:"bootstrap_url"`
		AgentURL          string   `json:"agent_url"`
		TargetID          int64    `json:"target_id"`
		AgentID           string   `json:"agent_id"`
		SecretFile        string   `json:"secret_file"`
		BootstrapCA       string   `json:"bootstrap_ca_file"`
		TrustBundle       string   `json:"trust_bundle_file"`
		StateDir          string   `json:"state_dir"`
		DockerSocket      string   `json:"docker_socket"`
		ProviderID        string   `json:"provider_id"`
		Capabilities      []string `json:"capabilities"`
		CapabilityVersion string   `json:"capability_version"`
	}{input.BootstrapURL, input.AgentURL, targetID, input.AgentID, input.BootstrapTokenFile, input.BootstrapCAFile, input.TrustBundleFile, input.StateDir, "unix:///var/run/docker.sock", "docker", append([]string(nil), localDockerRuntimeAgentCapabilities...), "docker/v1"})
	if err != nil {
		return fmt.Errorf("marshal local agent delivery config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(input.ConfigFile), localDeliverySecretDirPerm); err != nil {
		return fmt.Errorf("create local agent delivery config directory: %w", err)
	}
	if err := os.WriteFile(input.ConfigFile, configBytes, localDeliverySecretFilePerm); err != nil {
		return fmt.Errorf("write local agent delivery config: %w", err)
	}
	return nil
}

func validLocalDockerRuntimeAgentDelivery(input LocalDockerRuntimeAgentDelivery) bool {
	for _, value := range []string{input.AgentID, input.ImageDigest, input.AgentVersion, input.EnrollmentRef, input.AutomationID, input.BootstrapTokenFile, input.ConfigFile, input.StateDir, input.BootstrapCAFile, input.TrustBundleFile} {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return validLocalDockerRuntimeAgentHTTPSURL(input.BootstrapURL) && validLocalDockerRuntimeAgentHTTPSURL(input.AgentURL)
}

func validLocalDockerRuntimeAgentHTTPSURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}

func localDockerRuntimeAgentFingerprint(input LocalDockerRuntimeAgentDelivery, targetID int64) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{fmt.Sprint(targetID), input.AgentID, input.ImageDigest, input.AgentVersion, input.EnrollmentRef}, "\n")))
	return hex.EncodeToString(digest[:])
}
