package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
)

// PrepareRunnerHandoff 冻结一次性 runner 的更新前备份范围；runner 不获得 Backup store 或任意工件路径写入权限。
func (s *Service) PrepareRunnerHandoff(ctx context.Context, plan moduleapi.BackupRunnerHandoffPlan) (moduleapi.BackupRunnerHandoffPlan, error) {
	if s == nil || s.repository == nil || !runnerArtifactPathsAllowed(plan) {
		return moduleapi.BackupRunnerHandoffPlan{}, moduleapi.ErrBackupInvalidInput
	}
	return s.repository.PrepareRunnerHandoff(ctx, plan)
}

// CancelRunnerHandoff 仅撤销未开始的 runner handoff，供调用方在 runner 启动前失败时清理冻结计划。
func (s *Service) CancelRunnerHandoff(ctx context.Context, operationID string, taskID uint64) error {
	if s == nil || s.repository == nil {
		return moduleapi.ErrBackupInvalidInput
	}
	return s.repository.CancelRunnerHandoff(ctx, operationID, taskID)
}

// CompleteRunnerHandoff 重新读取冻结引用的工件并验证大小和 SHA-256，避免 runner 用任意引用或伪造元数据创建备份事实。
func (s *Service) CompleteRunnerHandoff(ctx context.Context, input moduleapi.CompleteBackupRunnerHandoffInput) (moduleapi.BackupRunnerHandoffCompletion, error) {
	if s == nil || s.repository == nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, moduleapi.ErrBackupInvalidInput
	}
	plan, backupID, err := s.repository.GetRunnerHandoff(ctx, input.OperationID, input.TaskID)
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, err
	}
	if backupID != 0 {
		// 已结算回执只和持久化的 Backup 完整性元数据比较，避免工件过期后破坏幂等重放。
		return s.repository.CompleteRunnerHandoff(ctx, input)
	}
	if !runnerArtifactPathsAllowed(plan) {
		return moduleapi.BackupRunnerHandoffCompletion{}, moduleapi.ErrBackupInvalidInput
	}
	config, err := verifyRunnerArtifact(plan.ArtifactRoot, plan.ConfigSnapshotRef)
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, err
	}
	dump, err := verifyRunnerArtifact(plan.ArtifactRoot, plan.DatabaseDumpRef)
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, err
	}
	if !matchesRunnerArtifact(config, input.ConfigSnapshotSHA256, input.ConfigSnapshotBytes) || !matchesRunnerArtifact(dump, input.DatabaseDumpSHA256, input.DatabaseDumpBytes) {
		return moduleapi.BackupRunnerHandoffCompletion{}, moduleapi.ErrBackupInvalidInput
	}
	input.ConfigSnapshotSHA256 = config.SHA256
	input.ConfigSnapshotBytes = config.SizeBytes
	input.DatabaseDumpSHA256 = dump.SHA256
	input.DatabaseDumpBytes = dump.SizeBytes
	return s.repository.CompleteRunnerHandoff(ctx, input)
}

func runnerArtifactPathsAllowed(plan moduleapi.BackupRunnerHandoffPlan) bool {
	return filepath.IsAbs(plan.ArtifactRoot) && pathWithinRoot(plan.ArtifactRoot, plan.ConfigSnapshotRef) && pathWithinRoot(plan.ArtifactRoot, plan.DatabaseDumpRef) && plan.ConfigSnapshotRef != plan.DatabaseDumpRef
}

func pathWithinRoot(root string, artifact string) bool {
	if !filepath.IsAbs(artifact) {
		return false
	}
	relative, err := filepath.Rel(root, artifact)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func verifyRunnerArtifact(root string, ref string) (moduleapi.BackupArtifact, error) {
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return moduleapi.BackupArtifact{}, moduleapi.ErrBackupInvalidInput
	}
	canonicalRef, err := filepath.EvalSymlinks(ref)
	if err != nil || !pathWithinRoot(canonicalRoot, canonicalRef) {
		return moduleapi.BackupArtifact{}, moduleapi.ErrBackupInvalidInput
	}
	file, err := os.Open(canonicalRef)
	if err != nil {
		return moduleapi.BackupArtifact{}, moduleapi.ErrBackupInvalidInput
	}
	defer func() { _ = file.Close() }()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return moduleapi.BackupArtifact{}, moduleapi.ErrBackupInvalidInput
	}
	return moduleapi.BackupArtifact{StorageRef: ref, SHA256: hex.EncodeToString(hash.Sum(nil)), SizeBytes: size}, nil
}

func matchesRunnerArtifact(actual moduleapi.BackupArtifact, claimedSHA256 string, claimedBytes int64) bool {
	return actual.SHA256 == strings.ToLower(strings.TrimSpace(claimedSHA256)) && actual.SizeBytes == claimedBytes
}
