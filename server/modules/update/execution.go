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
	DeclaredMode        string
	DetectedMode        string
	ComposeRoot         string
	Platform            string
	DockerSocket        string
	ComposeFiles        []string
	ServerReference     string
	WebReference        string
	ServerDigest        string
	WebDigest           string
	RunnerReference     string
	RunnerDigest        string
	BundledPostgres     bool
	OfficialServerImage string
	OfficialWebImage    string
	OfficialRunnerImage string
}

// RunnerInput 是 server 写入 runner 输入目录的版本化、无秘密协议。runner 只读取该文件，
// 不提供 HTTP API，也不得把 .env、数据库连接串或备份正文写入 receipt。
type RunnerInput struct {
	ProtocolVersion int
	OperationID     string
	TaskID          uint64
	Preflight       ComposePreflight
}

// RunnerReceipt 是 runner 写入受限状态目录的无秘密结算证据。
type RunnerReceipt struct {
	ProtocolVersion   int
	OperationID       string
	MigrationStarted  bool
	Succeeded         bool
	FailureCode       string
	RecoveryCompleted bool
	BackupCompletion  *moduleapi.CompleteBackupRunnerHandoffInput
}

// ExecutionOutcome 将 receipt 转换为 Update 业务状态。迁移已开始后不自动回退数据库。
type ExecutionOutcome string

const (
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
	if len(value.ComposeFiles) != 1 || filepath.Base(value.ComposeFiles[0]) != "compose.yml" {
		return errors.New("compose overrides and non-official compose files are not supported")
	}
	if !value.BundledPostgres {
		return errors.New("external database deployments are not supported")
	}
	return nil
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
