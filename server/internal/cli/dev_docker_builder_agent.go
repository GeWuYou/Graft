package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/database"
	"graft/server/internal/moduleapi"
	credentialvault "graft/server/modules/credential-vault"
	runtimetarget "graft/server/modules/runtime-target"
)

const localDockerBuilderAgentName = "docker-builder-agent-local"
const localDockerBuilderSecretDirPerm = 0o700
const (
	localDockerBuilderSecretFilePerm = 0o600
	localDockerBuilderHealthTimeout  = 5 * time.Second
)
const localDockerBuilderServerWait = 90 * time.Second

func newDevDockerBuilderAgentCommand() *cobra.Command {
	command := &cobra.Command{Use: "docker-builder-agent", Short: "Prepare the local Docker Builder Agent development topology"}
	command.AddCommand(&cobra.Command{Use: "prepare", Short: "Start isolated local dependencies and write Server environment", Args: cobra.NoArgs, RunE: runDevDockerBuilderAgentPrepare})
	command.AddCommand(&cobra.Command{Use: "deliver", Short: "Wait for the local Server and deliver the local Agent identity", Args: cobra.NoArgs, RunE: runDevDockerBuilderAgentDeliver})
	command.AddCommand(&cobra.Command{Use: "reset", Short: "Archive and rebuild the isolated local Docker Builder Agent development environment", Args: cobra.NoArgs, RunE: runDevDockerBuilderAgentReset})
	return command
}

func runDevDockerBuilderAgentPrepare(cmd *cobra.Command, _ []string) error {
	root, err := localDockerBuilderAgentRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, localDockerBuilderSecretDirPerm); err != nil {
		return err
	}
	composeFile := filepath.Join(filepath.Dir(filepath.Dir(root)), "deployments", "docker-builder-agent-dev", "compose.yml")
	if err := runLocalDockerBuilderCompose(cmd.Context(), root, composeFile, "up", "-d", "postgres", "redis", "vault"); err != nil {
		return err
	}
	if err := runLocalDockerBuilderCompose(cmd.Context(), root, composeFile, "run", "--rm", "vault-init"); err != nil {
		return err
	}
	if err := exportLocalVaultCA(cmd.Context(), root, composeFile); err != nil {
		return err
	}
	return writeLocalDockerBuilderServerEnv(root)
}

//nolint:cyclop // 开发交付需要在同一 CLI 边界显式装配依赖、配置与 authority。
func runDevDockerBuilderAgentDeliver(cmd *cobra.Command, _ []string) error {
	if err := runDevDockerBuilderAgentPrepare(cmd, nil); err != nil {
		return err
	}
	root, err := localDockerBuilderAgentRoot()
	if err != nil {
		return err
	}
	httpAddress := localDevAddress("GRAFT_DOCKER_BUILDER_DEV_HTTP_ADDR", "127.0.0.1:8080")
	bootstrapAddress := localDevAddress("GRAFT_DOCKER_BUILDER_DEV_BOOTSTRAP_ADDR", "127.0.0.1:8443")
	agentAddress := localDevAddress("GRAFT_DOCKER_BUILDER_DEV_AGENT_ADDR", "127.0.0.1:8444")
	if err := waitForLocalDockerBuilderServer(cmd.Context(), httpAddress); err != nil {
		return err
	}
	oldEnv, hadEnv := os.LookupEnv("GRAFT_ENV_FILE")
	if err := os.Setenv("GRAFT_ENV_FILE", filepath.Join(root, "server.env")); err != nil {
		return err
	}
	defer func() {
		if hadEnv {
			_ = os.Setenv("GRAFT_ENV_FILE", oldEnv)
		} else {
			_ = os.Unsetenv("GRAFT_ENV_FILE")
		}
	}()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	resources, err := database.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(resources) }()
	pepper, err := config.NewEnrollmentPepperProvider(cfg.EnrollmentSecurity)
	if err != nil {
		return err
	}
	issuanceStore, err := credentialvault.NewSQLIssuanceStateStore(resources.SQL)
	if err != nil {
		return err
	}
	issuer, err := credentialvault.NewVaultPKIClient(cfg.CredentialVault, issuanceStore)
	if err != nil {
		return err
	}
	repositoryRoot := filepath.Dir(filepath.Dir(root))
	return runtimetarget.PrepareLocalDockerBuilderAgent(cmd.Context(), resources.SQL, pepper, moduleapi.AgentCertificateIssuer(issuer), runtimetarget.LocalDockerBuilderAgentDelivery{AgentID: localDockerBuilderAgentName, ImageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", AgentVersion: "local", EnrollmentRef: localDockerBuilderAgentName, AutomationID: localDockerBuilderAgentName, BootstrapURL: "https://" + bootstrapAddress, AgentURL: "https://" + agentAddress, BootstrapTokenFile: filepath.Join(root, "agent", "bootstrap", "bootstrap-token"), ConfigFile: filepath.Join(repositoryRoot, "server", "agents", "docker-builder-agent", "config", "agent.local.json"), StateDir: filepath.Join(root, "agent", "state"), BootstrapCAFile: filepath.Join(root, "agent", "trust", "ca.pem"), TrustBundleFile: filepath.Join(root, "agent", "trust", "ca.pem")})
}

// runDevDockerBuilderAgentReset 归档本开发拓扑的受忽略状态后重建依赖与环境文件。
// 数据库中的 Agent binding 不在该命令的清理范围内；缺少交付配置时 deliver 会明确要求先重置该 binding。
func runDevDockerBuilderAgentReset(cmd *cobra.Command, _ []string) error {
	root, err := localDockerBuilderAgentRoot()
	if err != nil {
		return err
	}
	composeFile := filepath.Join(filepath.Dir(filepath.Dir(root)), "deployments", "docker-builder-agent-dev", "compose.yml")
	if err := runLocalDockerBuilderCompose(cmd.Context(), root, composeFile, "down", "--remove-orphans"); err != nil {
		return err
	}
	if _, err := os.Stat(root); err == nil {
		archive := root + ".reset-" + time.Now().UTC().Format("20060102T150405Z")
		if err := os.Rename(root, archive); err != nil {
			return fmt.Errorf("archive local Docker Builder development state: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	configFile := filepath.Join(filepath.Dir(filepath.Dir(root)), "server", "agents", "docker-builder-agent", "config", "agent.local.json")
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove generated local Agent config: %w", err)
	}
	return runDevDockerBuilderAgentPrepare(cmd, nil)
}

func localDockerBuilderAgentRoot() (string, error) {
	moduleRoot, err := resolveBackendModuleRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(moduleRoot), ".data", "docker-builder-agent-dev"), nil
}

func runLocalDockerBuilderCompose(ctx context.Context, root, composeFile string, args ...string) error {
	//nolint:gosec // compose 文件和子命令均来自本开发 CLI 的固定调用点。
	command := exec.CommandContext(ctx, "docker", append([]string{"compose", "-p", "graft-docker-builder-agent-dev", "-f", composeFile}, args...)...)
	command.Env = append(os.Environ(), "GRAFT_DOCKER_BUILDER_DEV_ROOT="+root)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run local Docker Builder dependency topology: %w", err)
	}
	return nil
}

func exportLocalVaultCA(ctx context.Context, root, composeFile string) error {
	//nolint:gosec // compose 文件与 vault service 名均由本开发 CLI 固定。
	pathCommand := exec.CommandContext(ctx, "docker", "compose", "-p", "graft-docker-builder-agent-dev", "-f", composeFile, "exec", "-T", "vault", "sh", "-ec", "find /tmp -name vault-ca.pem -print -quit")
	pathCommand.Env = append(os.Environ(), "GRAFT_DOCKER_BUILDER_DEV_ROOT="+root)
	path, err := pathCommand.Output()
	if err != nil {
		return fmt.Errorf("locate local Vault CA: %w", err)
	}
	if len(path) == 0 {
		return errors.New("locate local Vault CA: Vault returned an empty CA path")
	}
	//nolint:gosec // compose 文件与 vault service 名均由本开发 CLI 固定。
	containerCommand := exec.CommandContext(ctx, "docker", "compose", "-p", "graft-docker-builder-agent-dev", "-f", composeFile, "ps", "-q", "vault")
	containerCommand.Env = append(os.Environ(), "GRAFT_DOCKER_BUILDER_DEV_ROOT="+root)
	container, err := containerCommand.Output()
	if err != nil {
		return fmt.Errorf("resolve local Vault container: %w", err)
	}
	//nolint:gosec // 源路径只来自当前 compose vault 容器的公开 CA 文件。
	copyCommand := exec.CommandContext(ctx, "docker", "cp", strings.TrimSpace(string(container))+":"+strings.TrimSpace(string(path)), filepath.Join(root, "secrets", "vault-ca.pem"))
	if err := copyCommand.Run(); err != nil {
		return fmt.Errorf("export local Vault CA: %w", err)
	}
	return os.Chmod(filepath.Join(root, "secrets", "vault-ca.pem"), localDockerBuilderSecretFilePerm)
}

func writeLocalDockerBuilderServerEnv(root string) error {
	secrets := filepath.Join(root, "secrets")
	entries := []struct{ key, value string }{
		{"GRAFT_APP_ENV", "local"},
		{"GRAFT_CONFIG_SCHEMA_VERSION", "1"},
		{"GRAFT_HTTP_ADDR", localDevAddress("GRAFT_DOCKER_BUILDER_DEV_HTTP_ADDR", "127.0.0.1:8080")},
		{"GRAFT_DATABASE_URL", "postgres://graft:graft@127.0.0.1:15432/graft?sslmode=disable"},
		{"GRAFT_REDIS_ADDR", "127.0.0.1:16379"},
		{"GRAFT_AUTH_JWT_SECRET", "local-docker-builder-agent-development"},
		{"GRAFT_CREDENTIAL_VAULT_ENABLED", "true"},
		{"GRAFT_CREDENTIAL_VAULT_BACKEND", "vault-pki"},
		{"GRAFT_CREDENTIAL_VAULT_ADDRESS", "https://127.0.0.1:18200"},
		{"GRAFT_CREDENTIAL_VAULT_CA_FILE", filepath.Join(secrets, "vault-ca.pem")},
		{"GRAFT_CREDENTIAL_VAULT_AUTH_MOUNT", "approle"},
		{"GRAFT_CREDENTIAL_VAULT_AUTH_ROLE", "graft-docker-builder-agent"},
		{"GRAFT_CREDENTIAL_VAULT_AUTH_ROLE_ID_FILE", filepath.Join(secrets, "role_id")},
		{"GRAFT_CREDENTIAL_VAULT_AUTH_SECRET_ID_FILE", filepath.Join(secrets, "secret_id")},
		{"GRAFT_CREDENTIAL_VAULT_PKI_MOUNT", "pki"},
		{"GRAFT_CREDENTIAL_VAULT_PKI_ROLE", "graft-docker-builder-agent"},
		{"GRAFT_CREDENTIAL_VAULT_TRUST_BUNDLE_REF", "vault://pki/cert/ca"},
		{"GRAFT_ENROLLMENT_PEPPER_FILE", filepath.Join(secrets, "enrollment-pepper")},
		{"GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_ENABLED", "true"},
		{"GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_ADDR", localDevAddress("GRAFT_DOCKER_BUILDER_DEV_BOOTSTRAP_ADDR", "127.0.0.1:8443")},
		{"GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_CERTIFICATE_FILE", filepath.Join(secrets, "server-cert.pem")},
		{"GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_KEY_FILE", filepath.Join(secrets, "server-key.pem")},
		{"GRAFT_HTTPX_AGENT_TLS_ENABLED", "true"},
		{"GRAFT_HTTPX_AGENT_TLS_ADDR", localDevAddress("GRAFT_DOCKER_BUILDER_DEV_AGENT_ADDR", "127.0.0.1:8444")},
		{"GRAFT_HTTPX_AGENT_TLS_CERTIFICATE_FILE", filepath.Join(secrets, "server-cert.pem")},
		{"GRAFT_HTTPX_AGENT_TLS_KEY_FILE", filepath.Join(secrets, "server-key.pem")},
		{"GRAFT_HTTPX_AGENT_TLS_CLIENT_CA_FILE", filepath.Join(secrets, "ca.pem")},
	}
	var contents strings.Builder
	for _, entry := range entries {
		contents.WriteString(entry.key)
		contents.WriteByte('=')
		contents.WriteString(entry.value)
		contents.WriteByte('\n')
	}
	return os.WriteFile(filepath.Join(root, "server.env"), []byte(contents.String()), localDockerBuilderSecretFilePerm)
}

func localDevAddress(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func waitForLocalDockerBuilderServer(ctx context.Context, address string) error {
	client := &http.Client{Timeout: localDockerBuilderHealthTimeout}
	deadline := time.Now().Add(localDockerBuilderServerWait)
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/healthz", nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err == nil && response.StatusCode == http.StatusOK {
			_ = response.Body.Close()
			return nil
		}
		if response != nil {
			_ = response.Body.Close()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("local Server did not become healthy within %s", localDockerBuilderServerWait)
}
