package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
)

const runnerProtocolVersion = 1

// ComposePreflight 是 server 在启动一次性 runner 前冻结的无秘密部署证据。
// 它只描述官方 Compose 画像，不接受自定义覆盖层、外部数据库或可变镜像标签。
type ComposePreflight struct {
	DeclaredMode        string   `json:"declared_mode"`
	DetectedMode        string   `json:"detected_mode"`
	ComposeRoot         string   `json:"compose_root"`
	Platform            string   `json:"platform"`
	DockerSocket        string   `json:"docker_socket"`
	ComposeFiles        []string `json:"compose_files"`
	ServerReference     string   `json:"server_reference"`
	WebReference        string   `json:"web_reference"`
	ServerDigest        string   `json:"server_digest"`
	WebDigest           string   `json:"web_digest"`
	RunnerReference     string   `json:"runner_reference"`
	RunnerDigest        string   `json:"runner_digest"`
	BundledPostgres     bool     `json:"bundled_postgres"`
	OfficialServerImage string   `json:"official_server_image"`
	OfficialWebImage    string   `json:"official_web_image"`
	OfficialRunnerImage string   `json:"official_runner_image"`
}

// RunnerInput 是 server 写入 runner 输入目录的版本化、无秘密协议。runner 只读取该文件，
// 不提供 HTTP API，也不得把 .env、数据库连接串或备份正文写入 receipt。
type RunnerInput struct {
	ProtocolVersion int              `json:"protocol_version"`
	OperationID     string           `json:"operation_id"`
	TaskID          uint64           `json:"task_id"`
	Preflight       ComposePreflight `json:"preflight"`
}

// RunnerReceipt 是 runner 写入受限状态目录的无秘密结算证据。
type RunnerReceipt struct {
	ProtocolVersion   int                                         `json:"protocol_version"`
	OperationID       string                                      `json:"operation_id"`
	MigrationStarted  bool                                        `json:"migration_started"`
	Succeeded         bool                                        `json:"succeeded"`
	FailureCode       string                                      `json:"failure_code,omitempty"`
	RecoveryCompleted bool                                        `json:"recovery_completed"`
	BackupCompletion  *moduleapi.CompleteBackupRunnerHandoffInput `json:"backup_completion,omitempty"`
}

// ExecutionOutcome 将 receipt 转换为 Update 业务状态。迁移已开始后不自动回退数据库。
type ExecutionOutcome string

const (
	// ExecutionOutcomePlanning 表示操作已经持久化，尚未交给 runner。
	ExecutionOutcomePlanning ExecutionOutcome = "PLANNING"
	// ExecutionOutcomeBackingUp 为 runner 启动前备份交接的可见阶段。
	ExecutionOutcomeBackingUp ExecutionOutcome = "BACKING_UP"
	// ExecutionOutcomePulling 表示 runner 正在拉取经 manifest 固定的目标镜像。
	ExecutionOutcomePulling ExecutionOutcome = "PULLING"
	// ExecutionOutcomeMigrating 表示目标 bootstrap 已开始执行 forward-only Atlas migration。
	ExecutionOutcomeMigrating ExecutionOutcome = "MIGRATING"
	// ExecutionOutcomeRecreating 表示 runner 正在受控重建 server 和 web。
	ExecutionOutcomeRecreating ExecutionOutcome = "RECREATING"
	// ExecutionOutcomeVerifying 表示 runner 正在执行 Docker health 与 /healthz 验证。
	ExecutionOutcomeVerifying ExecutionOutcome = "VERIFYING"
	// ExecutionOutcomeSuccess 表示 runner 已完成目标版本健康验证。
	ExecutionOutcomeSuccess ExecutionOutcome = "SUCCESS"
	// ExecutionOutcomeFailed 表示 runner 未开始迁移且未提供可恢复失败证据。
	ExecutionOutcomeFailed ExecutionOutcome = "FAILED"
	// ExecutionOutcomeRecovered 表示迁移前失败已恢复配置快照和旧镜像引用。
	ExecutionOutcomeRecovered ExecutionOutcome = "RECOVERED"
	// ExecutionOutcomeNeedsAttention 表示 forward-only migration 开始后失败，必须由操作者恢复。
	ExecutionOutcomeNeedsAttention ExecutionOutcome = "NEEDS_ATTENTION"
)

// ValidateComposePreflight 拒绝不能证明为官方单节点 Compose 安装的输入。
func ValidateComposePreflight(value ComposePreflight) error {
	if normalizeMode(value.DeclaredMode) != "compose" || strings.TrimSpace(value.DetectedMode) != "compose" {
		return errors.New("official compose deployment was not declared and detected")
	}
	if err := validateComposeHost(value); err != nil {
		return err
	}
	if err := validateComposeTopology(value); err != nil {
		return err
	}
	return validateComposeImages(value)
}

func validateComposeHost(value ComposePreflight) error {
	if !filepath.IsAbs(value.ComposeRoot) {
		return errors.New("compose root must be an absolute host path")
	}
	if strings.TrimSpace(value.Platform) != "linux/amd64" {
		return errors.New("only linux/amd64 compose deployment is supported")
	}
	if strings.TrimSpace(value.DockerSocket) != "/var/run/docker.sock" {
		return errors.New("only the official docker socket location is supported")
	}
	return nil
}

func validateComposeTopology(value ComposePreflight) error {
	if err := validateComposeFiles(value.ComposeRoot, value.ComposeFiles); err != nil {
		return err
	}
	if !value.BundledPostgres {
		return errors.New("external database deployments are not supported")
	}
	return nil
}

func validateComposeFiles(root string, composeFiles []string) error {
	if err := validateComposeFileList(composeFiles); err != nil {
		return err
	}
	firstFile := filepath.Clean(composeFiles[0])
	baseName, err := validateFirstComposeFileName(firstFile)
	if err != nil {
		return err
	}
	realRoot := resolveComposePath(root)
	for _, composeFile := range composeFiles {
		if err := validateComposeFilePath(realRoot, composeFile); err != nil {
			return err
		}
	}
	return validateFirstComposeFileLocation(realRoot, firstFile, baseName)
}

func validateComposeFileList(composeFiles []string) error {
	if len(composeFiles) == 0 {
		return errors.New("at least one compose file is required")
	}
	return nil
}

func validateFirstComposeFileName(firstFile string) (string, error) {
	baseName := filepath.Base(firstFile)
	if baseName != "compose.yml" && baseName != "compose.yaml" {
		return "", errors.New("the first compose file must be the official compose.yml or compose.yaml")
	}
	return baseName, nil
}

func validateComposeFilePath(realRoot string, composeFile string) error {
	if !filepath.IsAbs(composeFile) {
		return errors.New("compose files must be absolute host paths")
	}
	realFile := resolveComposePath(composeFile)
	rel, err := filepath.Rel(realRoot, realFile)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return errors.New("compose files must be under the compose root")
	}
	return nil
}

func validateFirstComposeFileLocation(realRoot string, firstFile string, baseName string) error {
	if resolveComposePath(firstFile) != filepath.Join(realRoot, baseName) {
		return errors.New("the first compose file must be directly under the compose root")
	}
	return nil
}

func resolveComposePath(path string) string {
	cleanPath := filepath.Clean(path)
	current := cleanPath
	var suffix []string
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for index := len(suffix) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, suffix[index])
			}
			return filepath.Clean(resolved)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return cleanPath
		}
		suffix = append(suffix, filepath.Base(current))
		current = parent
	}
}

func validateComposeImages(value ComposePreflight) error {
	if !validDigest(value.ServerDigest) || !validDigest(value.WebDigest) || !validDigest(value.RunnerDigest) {
		return errors.New("server, web, and runner image digests are required")
	}
	if value.ServerReference != value.OfficialServerImage+"@"+value.ServerDigest || value.WebReference != value.OfficialWebImage+"@"+value.WebDigest || value.RunnerReference != value.OfficialRunnerImage+"@"+value.RunnerDigest {
		return errors.New("server, web, and runner must use official immutable image references")
	}
	if value.OfficialRunnerImage != composeRunnerImage(value.OfficialServerImage) {
		return errors.New("runner image does not belong to the official release authority")
	}
	return nil
}

// ValidateRunnerInput verifies the server-to-runner contract before Docker is invoked.
func ValidateRunnerInput(value RunnerInput) error {
	if value.ProtocolVersion != runnerProtocolVersion || strings.TrimSpace(value.OperationID) == "" || value.TaskID == 0 {
		return errors.New("runner input has an unsupported protocol or missing operation identity")
	}
	return ValidateComposePreflight(value.Preflight)
}

// RunnerSequence is the only command order a runner may execute. The actual command adapter must not accept caller supplied commands.
func RunnerSequence() []string {
	return []string{"backup", "compose pull", "bootstrap migrate up", "compose recreate server web", "docker health", "healthz", "write receipt"}
}

// ClassifyRunnerReceipt preserves the forward-only schema boundary in the durable operation state.
func ClassifyRunnerReceipt(value RunnerReceipt) ExecutionOutcome {
	if value.Succeeded {
		return ExecutionOutcomeSuccess
	}
	if value.MigrationStarted {
		return ExecutionOutcomeNeedsAttention
	}
	if value.RecoveryCompleted {
		return ExecutionOutcomeRecovered
	}
	return ExecutionOutcomeFailed
}

// RunnerReceiptIntegrity 为 Task Runtime 提供可重复计算的 receipt 完整性摘要；它不把 receipt 内容直接写入 Task 事实。
func RunnerReceiptIntegrity(value RunnerReceipt) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", errors.New("encode runner receipt integrity")
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
