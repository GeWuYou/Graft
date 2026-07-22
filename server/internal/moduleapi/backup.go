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
// 只供服务端模块协作使用，任何 HTTP 读取面都必须投影为 BackupSummary。
type BackupArtifact struct {
	StorageRef string
	SHA256     string
	SizeBytes  int64
}

// Backup 是 backup 模块拥有的完整内部备份事实。
type Backup struct {
	ID             uint64
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
	RestoreCode string
	RestoreAt   *time.Time
}

// CreateBackupInput 描述完成配置快照和数据库 dump 后要记录的元数据。
type CreateBackupInput struct {
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

// BackupService 是 Update 等消费者可依赖的窄备份 capability。
//
// 它只记录由 backup owner 创建或验证过的工件事实；不会下载工件、返回配置内容，
// 也不会执行数据库恢复或 migration。
type BackupService interface {
	Create(ctx context.Context, input CreateBackupInput) (Backup, error)
	Get(ctx context.Context, id uint64) (Backup, error)
	RecordRestoreEvidence(ctx context.Context, input RecordBackupRestoreInput) (Backup, error)
}
