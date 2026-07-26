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
	TaskID      *uint64                `json:"task_id"`
	Purpose     string                 `json:"purpose"`
	Status      moduleapi.BackupStatus `json:"status"`
	RetainUntil time.Time              `json:"retain_until"`
	CreatedAt   time.Time              `json:"created_at"`
}

type backupListResponse struct {
	Items  []backupSummaryResponse `json:"items"`
	Total  int64                   `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

func toBackupSummaryResponse(item moduleapi.BackupSummary) backupSummaryResponse {
	return backupSummaryResponse{ID: item.ID, TaskID: item.TaskID, Purpose: item.Purpose, Status: item.Status, RetainUntil: item.RetainUntil, CreatedAt: item.CreatedAt}
}

func backupSummaryResponses(items []moduleapi.BackupSummary) []backupSummaryResponse {
	responses := make([]backupSummaryResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, toBackupSummaryResponse(item))
	}
	return responses
}
