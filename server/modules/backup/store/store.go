// Package store 持久化 backup 模块拥有的工件元数据，不读取或返回工件内容。
package store

import (
	"context"

	"graft/server/internal/moduleapi"
)

// Repository 是 backup 模块的窄持久化边界。
type Repository interface {
	Create(context.Context, moduleapi.CreateBackupInput) (moduleapi.Backup, error)
	PrepareRunnerHandoff(context.Context, moduleapi.BackupRunnerHandoffPlan) (moduleapi.BackupRunnerHandoffPlan, error)
	GetRunnerHandoff(context.Context, string, uint64) (moduleapi.BackupRunnerHandoffPlan, uint64, error)
	CompleteRunnerHandoff(context.Context, moduleapi.CompleteBackupRunnerHandoffInput) (moduleapi.BackupRunnerHandoffCompletion, error)
	Get(context.Context, uint64) (moduleapi.Backup, error)
	RecordRestoreEvidence(context.Context, moduleapi.RecordBackupRestoreInput) (moduleapi.Backup, error)
}
