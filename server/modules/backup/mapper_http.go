package backup

import (
	"time"

	"graft/server/internal/moduleapi"
)

type backupTaskReceiptResponse struct {
	TaskID uint64               `json:"task_id"`
	Status moduleapi.TaskStatus `json:"status"`
}

type backupSummaryResponse struct {
	ID          uint64                 `json:"id"`
	Purpose     string                 `json:"purpose"`
	Status      moduleapi.BackupStatus `json:"status"`
	RetainUntil time.Time              `json:"retain_until"`
	CreatedAt   time.Time              `json:"created_at"`
}

type backupArtifactMetadataResponse struct {
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

type backupRestoreEvidenceResponse struct {
	Status     string     `json:"status"`
	ResultCode *string    `json:"result_code"`
	RecordedAt *time.Time `json:"recorded_at"`
}

// backupDetailResponse 是 HTTP 专用的备份资产投影。它有意不包含内部存储引用、
// 创建者或工件正文，避免详情读取面成为工件访问通道。
type backupDetailResponse struct {
	ID              uint64                         `json:"id"`
	TaskID          *uint64                        `json:"task_id"`
	Purpose         string                         `json:"purpose"`
	Status          moduleapi.BackupStatus         `json:"status"`
	RetainUntil     time.Time                      `json:"retain_until"`
	CreatedAt       time.Time                      `json:"created_at"`
	ConfigSnapshot  backupArtifactMetadataResponse `json:"config_snapshot"`
	DatabaseDump    backupArtifactMetadataResponse `json:"database_dump"`
	RestoreEvidence backupRestoreEvidenceResponse  `json:"restore_evidence"`
}

type backupListResponse struct {
	Items  []backupSummaryResponse `json:"items"`
	Total  int64                   `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

func toBackupSummaryResponse(item moduleapi.BackupSummary) backupSummaryResponse {
	return backupSummaryResponse{ID: item.ID, Purpose: item.Purpose, Status: item.Status, RetainUntil: item.RetainUntil, CreatedAt: item.CreatedAt}
}

func toBackupDetailResponse(item moduleapi.Backup) backupDetailResponse {
	evidence := backupRestoreEvidenceResponse{Status: "NOT_VERIFIED"}
	if item.RestoreAt != nil {
		evidence.Status = "RECORDED"
		evidence.RecordedAt = item.RestoreAt
		if item.RestoreCode != "" {
			resultCode := item.RestoreCode
			evidence.ResultCode = &resultCode
		}
	}
	return backupDetailResponse{
		ID: item.ID, TaskID: item.TaskID, Purpose: item.Purpose, Status: item.Status,
		RetainUntil: item.RetainUntil, CreatedAt: item.CreatedAt,
		ConfigSnapshot: backupArtifactMetadataResponse{
			SizeBytes: item.ConfigSnapshot.SizeBytes,
			SHA256:    item.ConfigSnapshot.SHA256,
		},
		DatabaseDump: backupArtifactMetadataResponse{
			SizeBytes: item.DatabaseDump.SizeBytes,
			SHA256:    item.DatabaseDump.SHA256,
		},
		RestoreEvidence: evidence,
	}
}

func backupSummaryResponses(items []moduleapi.BackupSummary) []backupSummaryResponse {
	responses := make([]backupSummaryResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, toBackupSummaryResponse(item))
	}
	return responses
}
