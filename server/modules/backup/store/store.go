// Package store 持久化 backup 模块拥有的工件元数据，不读取或返回工件内容。
package store

import (
	"context"

	"graft/server/internal/moduleapi"
)

// Repository 是 backup 模块的窄持久化边界。
type Repository interface {
	Create(ctx context.Context, input moduleapi.CreateBackupInput) (moduleapi.Backup, error)
	PrepareRunnerHandoff(ctx context.Context, plan moduleapi.BackupRunnerHandoffPlan) (moduleapi.BackupRunnerHandoffPlan, error)
	CancelRunnerHandoff(ctx context.Context, operationID string, taskID uint64) error
	GetRunnerHandoff(ctx context.Context, operationID string, taskID uint64) (moduleapi.BackupRunnerHandoffPlan, uint64, error)
	CompleteRunnerHandoff(ctx context.Context, input moduleapi.CompleteBackupRunnerHandoffInput) (moduleapi.BackupRunnerHandoffCompletion, error)
	Get(ctx context.Context, id uint64) (moduleapi.Backup, error)
	GetSummary(ctx context.Context, id uint64) (moduleapi.BackupSummary, error)
	ListSummaries(ctx context.Context, limit int, offset int) ([]moduleapi.BackupSummary, int64, error)
	RecordRestoreEvidence(ctx context.Context, input moduleapi.RecordBackupRestoreInput) (moduleapi.Backup, error)
}
