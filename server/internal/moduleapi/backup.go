package moduleapi

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrBackupNotFound 表示请求的备份事实不存在或已被软删除。
	ErrBackupNotFound = errors.New("backup not found")
	// ErrBackupInvalidInput 表示备份元数据不满足跨模块能力的写入约束。
	ErrBackupInvalidInput = errors.New("backup invalid input")
)

// BackupStatus 标识备份在保留和恢复生命周期中的稳定状态。
type BackupStatus string

const (
	// BackupStatusAvailable 表示备份工件已记录且仍在保留期内。
	BackupStatusAvailable BackupStatus = "AVAILABLE"
	// BackupStatusExpired 表示备份已超过保留期，不能再作为恢复依据。
	BackupStatusExpired BackupStatus = "EXPIRED"
	// BackupStatusRestored 表示操作者已记录一次恢复结果；数据库恢复本身仍由后续受控流程拥有。
	BackupStatusRestored BackupStatus = "RESTORED"
)

// BackupArtifact 是备份工件的内部引用和完整性元数据。
//
// StorageRef 只能是受控存储位置或不透明标识，不承载 .env 或 dump 正文。该 DTO
// 只供服务端模块协作使用；HTTP 读取面只能映射为安全摘要或详情元数据，绝不返回
// StorageRef 或工件内容。
type BackupArtifact struct {
	StorageRef string
	SHA256     string
	SizeBytes  int64
}

// Backup 是 backup 模块拥有的完整内部备份事实。
type Backup struct {
	ID uint64
	// TaskID 为空表示历史或非 Task Runtime 创建的备份事实。
	TaskID         *uint64
	Purpose        string
	Status         BackupStatus
	ConfigSnapshot BackupArtifact
	DatabaseDump   BackupArtifact
	RetainUntil    time.Time
	CreatedBy      *uint64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	RestoreCode    string
	RestoreAt      *time.Time
}

// BackupSummary 是未来 HTTP 查询可安全映射的只读投影；它明确不含存储位置和工件内容。
type BackupSummary struct {
	ID          uint64
	Purpose     string
	Status      BackupStatus
	RetainUntil time.Time
	CreatedAt   time.Time
}

// CreateBackupInput 描述完成配置快照和数据库 dump 后要记录的元数据。
type CreateBackupInput struct {
	// TaskID 为零时创建不关联 Task 的备份；非零值要求持久化层按 Task 幂等。
	TaskID         uint64
	Purpose        string
	ConfigSnapshot BackupArtifact
	DatabaseDump   BackupArtifact
	RetainUntil    time.Time
	CreatedBy      *uint64
}

// RecordBackupRestoreInput 描述不含秘密内容的恢复结果证据。
type RecordBackupRestoreInput struct {
	ID          uint64
	Status      BackupStatus
	RestoreCode string
	RecordedAt  time.Time
}

// BackupRunnerHandoffPlan 冻结一次性 runner 可写入的更新前备份范围，不包含配置或 dump 内容。
type BackupRunnerHandoffPlan struct {
	OperationID       string
	TaskID            uint64
	Purpose           string
	RetainUntil       time.Time
	CreatedBy         *uint64
	ArtifactRoot      string
	ConfigSnapshotRef string
	DatabaseDumpRef   string
}

// CompleteBackupRunnerHandoffInput 只携带 runner 计算出的无秘密完整性元数据；工件引用必须复用冻结计划。
type CompleteBackupRunnerHandoffInput struct {
	OperationID          string
	TaskID               uint64
	ConfigSnapshotSHA256 string
	ConfigSnapshotBytes  int64
	DatabaseDumpSHA256   string
	DatabaseDumpBytes    int64
}

// BackupRunnerHandoffCompletion 是可写入 runner receipt 的安全备份证据，不暴露工件引用或内容。
type BackupRunnerHandoffCompletion struct {
	BackupID             uint64
	OperationID          string
	TaskID               uint64
	ConfigSnapshotSHA256 string
	ConfigSnapshotBytes  int64
	DatabaseDumpSHA256   string
	DatabaseDumpBytes    int64
	Idempotent           bool
}

// BackupService 是 Update 等消费者可依赖的窄备份 capability。
//
// 它只记录由 backup owner 创建或验证过的工件事实；不会下载工件、返回配置内容，
// 也不会执行数据库恢复或 migration。
type BackupService interface {
	Create(ctx context.Context, input CreateBackupInput) (Backup, error)
	PrepareRunnerHandoff(ctx context.Context, plan BackupRunnerHandoffPlan) (BackupRunnerHandoffPlan, error)
	CancelRunnerHandoff(ctx context.Context, operationID string, taskID uint64) error
	CompleteRunnerHandoff(ctx context.Context, input CompleteBackupRunnerHandoffInput) (BackupRunnerHandoffCompletion, error)
	Get(ctx context.Context, id uint64) (Backup, error)
	RecordRestoreEvidence(ctx context.Context, input RecordBackupRestoreInput) (Backup, error)
}
