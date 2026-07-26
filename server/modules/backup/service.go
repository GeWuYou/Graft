package backup

import (
	"context"

	"graft/server/internal/moduleapi"
	"graft/server/modules/backup/store"
)

// Service 收敛 backup 模块拥有的工件事实，并避免消费者直接访问持久化实现。
type Service struct {
	repository store.Repository
	tasks      moduleapi.TaskService
	writer     backupArtifactWriter
}

// NewService 创建备份服务。
func NewService(repository store.Repository) *Service { return &Service{repository: repository} }

func (s *Service) setTaskService(tasks moduleapi.TaskService) {
	if s != nil {
		s.tasks = tasks
	}
}

func (s *Service) setArtifactWriter(writer backupArtifactWriter) {
	if s != nil {
		s.writer = writer
	}
}

// Create 写入已经创建并校验过的配置快照和数据库 dump 元数据。
func (s *Service) Create(ctx context.Context, input moduleapi.CreateBackupInput) (moduleapi.Backup, error) {
	if s == nil || s.repository == nil {
		return moduleapi.Backup{}, moduleapi.ErrBackupInvalidInput
	}
	return s.repository.Create(ctx, input)
}

// Get 返回给受信任模块协作使用的完整内部备份事实。
func (s *Service) Get(ctx context.Context, id uint64) (moduleapi.Backup, error) {
	if s == nil || s.repository == nil || id == 0 {
		return moduleapi.Backup{}, moduleapi.ErrBackupInvalidInput
	}
	return s.repository.Get(ctx, id)
}

// RecordRestoreEvidence 写入恢复结论的稳定代码，不接收或持久化配置和 dump 内容。
func (s *Service) RecordRestoreEvidence(ctx context.Context, input moduleapi.RecordBackupRestoreInput) (moduleapi.Backup, error) {
	if s == nil || s.repository == nil {
		return moduleapi.Backup{}, moduleapi.ErrBackupInvalidInput
	}
	return s.repository.RecordRestoreEvidence(ctx, input)
}

// ToSummary 将内部工件事实投影为可用于未来 HTTP 读取面的安全摘要。
func ToSummary(item moduleapi.Backup) moduleapi.BackupSummary {
	return moduleapi.BackupSummary{ID: item.ID, TaskID: item.TaskID, Purpose: item.Purpose, Status: item.Status, RetainUntil: item.RetainUntil, CreatedAt: item.CreatedAt, RestoreCode: item.RestoreCode, RestoreAt: item.RestoreAt}
}
