package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
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
	localDockerBuilderSecretFilePerm      = 0o600
	localDockerBuilderServerEnvFileName   = ".env.docker-builder-agent"
	localDockerBuilderAgentConfigFileName = "agent.json"
	localDockerBuilderHealthTimeout       = 5 * time.Second
	localDockerBuilderIsolatedDatabaseURL = "postgres://graft:graft@127.0.0.1:15432/graft?sslmode=disable" //nolint:gosec // 仅用于显式 isolated 调试模式的可丢弃 Compose PostgreSQL。
)
const localDockerBuilderServerWait = 90 * time.Second

type localDockerBuilderDatabaseMode string

const (
	localDockerBuilderDatabaseModeShared   localDockerBuilderDatabaseMode = "shared"
	localDockerBuilderDatabaseModeIsolated localDockerBuilderDatabaseMode = "isolated"
)

func newDevDockerBuilderAgentCommand() *cobra.Command {
	command := &cobra.Command{Use: "docker-builder-agent", Short: "Prepare the local Docker Builder Agent development topology"}

	prepareMode := string(localDockerBuilderDatabaseModeShared)
	prepare := &cobra.Command{Use: "prepare", Short: "Start local dependencies and write the local Server integration environment", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return runDevDockerBuilderAgentPrepare(cmd, localDockerBuilderDatabaseMode(prepareMode))
	}}
	prepare.Flags().StringVar(&prepareMode, "database-mode", prepareMode, "Database mode: shared or isolated")

	deliverMode := string(localDockerBuilderDatabaseModeShared)
	deliverPrepared := false
	deliver := &cobra.Command{Use: "deliver", Short: "Wait for the local Server and deliver the local Agent identity", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return runDevDockerBuilderAgentDeliver(cmd, localDockerBuilderDatabaseMode(deliverMode), deliverPrepared)
	}}
	deliver.Flags().StringVar(&deliverMode, "database-mode", deliverMode, "Database mode: shared or isolated")
	deliver.Flags().BoolVar(&deliverPrepared, "prepared", deliverPrepared, "Reuse an already prepared local Docker Builder development environment")

	resetMode := string(localDockerBuilderDatabaseModeShared)
	reset := &cobra.Command{Use: "reset", Short: "Archive and rebuild the local Docker Builder Agent development environment", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return runDevDockerBuilderAgentReset(cmd, localDockerBuilderDatabaseMode(resetMode))
	}}
	reset.Flags().StringVar(&resetMode, "database-mode", resetMode, "Database mode: shared or isolated")

	command.AddCommand(prepare, deliver, reset)
	return command
}

func runDevDockerBuilderAgentPrepare(cmd *cobra.Command, databaseMode localDockerBuilderDatabaseMode) error {
	databaseMode, err := normalizeLocalDockerBuilderDatabaseMode(databaseMode)
	if err != nil {
		return err
	}

	root, err := localDockerBuilderAgentRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, localDockerBuilderSecretDirPerm); err != nil {
		return err
	}
	composeFile := filepath.Join(filepath.Dir(filepath.Dir(root)), "deployments", "docker-builder-agent-dev", "compose.yml")
	services := localDockerBuilderComposeDependencyServices(databaseMode)
	if err := runLocalDockerBuilderCompose(cmd.Context(), root, composeFile, services...); err != nil {
		return err
	}
	if err := runLocalDockerBuilderCompose(cmd.Context(), root, composeFile, "run", "--rm", "vault-init"); err != nil {
		return err
	}
	if err := exportLocalVaultCA(cmd.Context(), root, composeFile); err != nil {
		return err
	}
	return writeLocalDockerBuilderServerEnv(root, databaseMode)
}

// localDockerBuilderComposeDependencyServices 仅选择当前数据库模式必须启动的开发依赖，避免 shared 模式意外创建独立 PostgreSQL。
func localDockerBuilderComposeDependencyServices(mode localDockerBuilderDatabaseMode) []string {
	services := []string{"up", "-d", "redis", "vault"}
	if mode == localDockerBuilderDatabaseModeIsolated {
		return append(services, "postgres")
	}
	return services
}

//nolint:cyclop // 开发交付需要在同一 CLI 边界显式装配依赖、配置与 authority。
func runDevDockerBuilderAgentDeliver(cmd *cobra.Command, databaseMode localDockerBuilderDatabaseMode, prepared bool) error {
	if !prepared {
		if err := runDevDockerBuilderAgentPrepare(cmd, databaseMode); err != nil {
			return err
		}
	}
	root, err := localDockerBuilderAgentRoot()
	if err != nil {
		return err
	}
	if err := migrateLocalDockerBuilderAgentConfig(root); err != nil {
		return err
	}
	httpAddress := localDevAddress("GRAFT_DOCKER_BUILDER_DEV_HTTP_ADDR", "127.0.0.1:8080")
	bootstrapAddress := localDevAddress("GRAFT_DOCKER_BUILDER_DEV_BOOTSTRAP_ADDR", "127.0.0.1:8443")
	agentAddress := localDevAddress("GRAFT_DOCKER_BUILDER_DEV_AGENT_ADDR", "127.0.0.1:8444")
	if err := waitForLocalDockerBuilderServer(cmd.Context(), httpAddress); err != nil {
		return err
	}
	oldEnv, hadEnv := os.LookupEnv("GRAFT_ENV_FILE")
	if err := os.Setenv("GRAFT_ENV_FILE", localDockerBuilderServerEnvFile(root)); err != nil {
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
	return runtimetarget.PrepareLocalDockerBuilderAgent(cmd.Context(), resources.SQL, pepper, moduleapi.AgentCertificateIssuer(issuer), runtimetarget.LocalDockerBuilderAgentDelivery{AgentID: localDockerBuilderAgentName, ImageDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", AgentVersion: "local", EnrollmentRef: localDockerBuilderAgentName, AutomationID: localDockerBuilderAgentName, BootstrapURL: "https://" + bootstrapAddress, AgentURL: "https://" + agentAddress, BootstrapTokenFile: filepath.Join(root, "agent", "bootstrap", "bootstrap-token"), ConfigFile: localDockerBuilderAgentConfigFile(root), StateDir: filepath.Join(root, "agent", "state"), BootstrapCAFile: filepath.Join(root, "agent", "trust", "ca.pem"), TrustBundleFile: filepath.Join(root, "agent", "trust", "ca.pem")})
}

// runDevDockerBuilderAgentReset 归档本开发拓扑的受忽略状态后重建依赖与环境文件。
// 数据库中的 Agent binding 不在该命令的清理范围内；缺少交付配置时 deliver 会明确要求先重置该 binding。
func runDevDockerBuilderAgentReset(cmd *cobra.Command, databaseMode localDockerBuilderDatabaseMode) error {
	databaseMode, err := normalizeLocalDockerBuilderDatabaseMode(databaseMode)
	if err != nil {
		return err
	}

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
	configFile := localDockerBuilderAgentConfigFile(root)
	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove generated local Agent config: %w", err)
	}
	return runDevDockerBuilderAgentPrepare(cmd, databaseMode)
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
	command.Env = localDockerBuilderComposeEnv(root)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run local Docker Builder dependency topology: %w", err)
	}
	return nil
}

func exportLocalVaultCA(ctx context.Context, root, composeFile string) error {
	//nolint:gosec // compose 文件与 vault service 名均由本开发 CLI 固定。
	// Vault dev 在重启后会保留多个 /tmp/vault-tls* 目录；必须选择当前实例最新生成的 CA。
	pathCommand := exec.CommandContext(ctx, "docker", "compose", "-p", "graft-docker-builder-agent-dev", "-f", composeFile, "exec", "-T", "vault", "sh", "-ec", "ls -t /tmp/vault-tls*/vault-ca.pem 2>/dev/null | head -n 1")
	pathCommand.Env = localDockerBuilderComposeEnv(root)
	path, err := pathCommand.Output()
	if err != nil {
		return fmt.Errorf("locate local Vault CA: %w", err)
	}
	if len(path) == 0 {
		return errors.New("locate local Vault CA: Vault returned an empty CA path")
	}
	//nolint:gosec // compose 文件与 vault service 名均由本开发 CLI 固定。
	containerCommand := exec.CommandContext(ctx, "docker", "compose", "-p", "graft-docker-builder-agent-dev", "-f", composeFile, "ps", "-q", "vault")
	containerCommand.Env = localDockerBuilderComposeEnv(root)
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

func writeLocalDockerBuilderServerEnv(root string, databaseMode localDockerBuilderDatabaseMode) error {
	values, err := godotenv.Read(localDockerBuilderServerEnvSource(root))
	if err != nil {
		return fmt.Errorf("read local Server environment: %w", err)
	}
	if err := validateLocalDockerBuilderServerEnvironment(values, localDockerBuilderServerEnvSource(root)); err != nil {
		return err
	}
	databaseURL, err := resolveLocalDockerBuilderDatabaseURL(root, databaseMode)
	if err != nil {
		return err
	}

	secrets := filepath.Join(root, "secrets")
	// #nosec G101 -- 本地开发数据库地址由显式 database-mode 解析，不是凭据。
	overrides := map[string]string{
		"GRAFT_DATABASE_URL":                               databaseURL,
		"GRAFT_REDIS_ADDR":                                 "127.0.0.1:16379",
		"GRAFT_CREDENTIAL_VAULT_ENABLED":                   "true",
		"GRAFT_CREDENTIAL_VAULT_BACKEND":                   "vault-pki",
		"GRAFT_CREDENTIAL_VAULT_ADDRESS":                   "https://127.0.0.1:18200",
		"GRAFT_CREDENTIAL_VAULT_CA_FILE":                   filepath.Join(secrets, "vault-ca.pem"),
		"GRAFT_CREDENTIAL_VAULT_AUTH_MOUNT":                "approle",
		"GRAFT_CREDENTIAL_VAULT_AUTH_ROLE":                 "graft-docker-builder-agent",
		"GRAFT_CREDENTIAL_VAULT_AUTH_ROLE_ID_FILE":         filepath.Join(secrets, "role_id"),
		"GRAFT_CREDENTIAL_VAULT_AUTH_SECRET_ID_FILE":       filepath.Join(secrets, "secret_id"),
		"GRAFT_CREDENTIAL_VAULT_PKI_MOUNT":                 "pki",
		"GRAFT_CREDENTIAL_VAULT_PKI_ROLE":                  "graft-docker-builder-agent",
		"GRAFT_CREDENTIAL_VAULT_TRUST_BUNDLE_REF":          "vault://pki/cert/ca",
		"GRAFT_ENROLLMENT_PEPPER_FILE":                     filepath.Join(secrets, "enrollment-pepper"),
		"GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_ENABLED":          "true",
		"GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_ADDR":             localDevAddress("GRAFT_DOCKER_BUILDER_DEV_BOOTSTRAP_ADDR", "127.0.0.1:8443"),
		"GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_CERTIFICATE_FILE": filepath.Join(secrets, "server-cert.pem"),
		"GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_KEY_FILE":         filepath.Join(secrets, "server-key.pem"),
		"GRAFT_HTTPX_AGENT_TLS_ENABLED":                    "true",
		"GRAFT_HTTPX_AGENT_TLS_ADDR":                       localDevAddress("GRAFT_DOCKER_BUILDER_DEV_AGENT_ADDR", "127.0.0.1:8444"),
		"GRAFT_HTTPX_AGENT_TLS_CERTIFICATE_FILE":           filepath.Join(secrets, "server-cert.pem"),
		"GRAFT_HTTPX_AGENT_TLS_KEY_FILE":                   filepath.Join(secrets, "server-key.pem"),
		"GRAFT_HTTPX_AGENT_TLS_CLIENT_CA_FILE":             filepath.Join(secrets, "ca.pem"),
	}
	for key, value := range overrides {
		values[key] = value
	}
	contents, err := godotenv.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal local Server integration environment: %w", err)
	}
	return os.WriteFile(localDockerBuilderServerEnvFile(root), []byte(contents+"\n"), localDockerBuilderSecretFilePerm)
}

func localDockerBuilderServerEnvSource(root string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(root)), "server", ".env")
}

func localDockerBuilderServerEnvFile(root string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(root)), "server", localDockerBuilderServerEnvFileName)
}

func localDockerBuilderAgentConfigFile(root string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(root)), "server", "agents", "docker-builder-agent", localDockerBuilderAgentConfigFileName)
}

func legacyLocalDockerBuilderAgentConfigFile(root string) string {
	return filepath.Join(filepath.Dir(filepath.Dir(root)), "server", "agents", "docker-builder-agent", "config", "agent.local.json")
}

func migrateLocalDockerBuilderAgentConfig(root string) error {
	configFile := localDockerBuilderAgentConfigFile(root)
	if _, err := os.Stat(configFile); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect local Agent config: %w", err)
	}

	legacyFile := legacyLocalDockerBuilderAgentConfigFile(root)
	err := os.Rename(legacyFile, configFile)
	if err == nil {
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("move legacy local Agent config to Agent root: %w", err)
}

func normalizeLocalDockerBuilderDatabaseMode(mode localDockerBuilderDatabaseMode) (localDockerBuilderDatabaseMode, error) {
	switch localDockerBuilderDatabaseMode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case localDockerBuilderDatabaseModeShared:
		return localDockerBuilderDatabaseModeShared, nil
	case localDockerBuilderDatabaseModeIsolated:
		return localDockerBuilderDatabaseModeIsolated, nil
	default:
		return "", fmt.Errorf("unsupported Docker Builder database mode %q; expected shared or isolated", mode)
	}
}

func resolveLocalDockerBuilderDatabaseURL(root string, mode localDockerBuilderDatabaseMode) (string, error) {
	mode, err := normalizeLocalDockerBuilderDatabaseMode(mode)
	if err != nil {
		return "", err
	}
	if mode == localDockerBuilderDatabaseModeIsolated {
		return localDockerBuilderIsolatedDatabaseURL, nil
	}

	return readLocalDockerBuilderServerDatabaseURL(localDockerBuilderServerEnvSource(root))
}

func readLocalDockerBuilderServerDatabaseURL(envFile string) (string, error) {
	values, err := godotenv.Read(envFile)
	if err != nil {
		return "", fmt.Errorf("read local Server environment %s: %w", envFile, err)
	}
	if err := validateLocalDockerBuilderServerEnvironment(values, envFile); err != nil {
		return "", err
	}
	return strings.TrimSpace(values["GRAFT_DATABASE_URL"]), nil
}

func validateLocalDockerBuilderServerEnvironment(values map[string]string, envFile string) error {
	databaseURL := strings.TrimSpace(values["GRAFT_DATABASE_URL"])
	if databaseURL == "" {
		return fmt.Errorf("GRAFT_DATABASE_URL is required in local Server environment %s", envFile)
	}
	if appEnv := strings.TrimSpace(values["GRAFT_APP_ENV"]); !isDevelopmentAppEnv(appEnv) {
		return fmt.Errorf("local Docker Builder shared database requires local/test GRAFT_APP_ENV in %s, got %q", envFile, appEnv)
	}

	return nil
}

func localDevAddress(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func localDockerBuilderComposeEnv(root string) []string {
	return append(os.Environ(),
		"GRAFT_DOCKER_BUILDER_DEV_ROOT="+root,
		"GRAFT_DOCKER_BUILDER_DEV_UID="+strconv.Itoa(os.Getuid()),
		"GRAFT_DOCKER_BUILDER_DEV_GID="+strconv.Itoa(os.Getgid()),
	)
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
