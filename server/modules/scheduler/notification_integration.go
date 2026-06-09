// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
	schedulercore "graft/server/internal/scheduler"
	notificationcontract "graft/server/modules/notification/contract"
	schedulercontract "graft/server/modules/scheduler/contract"
)

type schedulerRunFailureNotifier struct {
	publisher moduleapi.NotificationPublisher
	logger    *zap.Logger
}

func (n schedulerRunFailureNotifier) NotifyRunFailed(ctx context.Context, run schedulercore.TaskRun) {
	if n.publisher == nil || run.ID == 0 {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"run_id":   run.ID,
		"task_key": run.TaskKey,
		"job_key":  run.JobKey,
	})
	input := moduleapi.PublishNotificationInput{
		Title:        "Scheduled task failed",
		Message:      "Scheduled task " + firstNonEmptyTrimmed(run.TaskName, run.TaskKey) + " failed.",
		Severity:     moduleapi.NotificationSeverity(notificationcontract.SeverityError),
		Category:     moduleapi.NotificationCategory(notificationcontract.CategoryTask),
		SourceModule: moduleID,
		EventType:    "task_failed",
		ResourceType: "scheduled_task_run",
		ResourceID:   strconv.FormatUint(run.ID, 10),
		ResourceName: firstNonEmptyTrimmed(run.TaskName, run.TaskKey),
		Navigation: moduleapi.NotificationNavigation{
			Kind:    moduleapi.NotificationNavigationKind(notificationcontract.NavigationSchedulerRun),
			Payload: payload,
		},
		Metadata:   schedulerRunFailureMetadata(run),
		DedupeKey:  "scheduler:run_failed:" + strconv.FormatUint(run.ID, 10),
		OccurredAt: firstNonZeroTime(run.FinishedAt, run.CreatedAt),
		Target: moduleapi.NotificationTarget{
			Type: moduleapi.NotificationTargetType(notificationcontract.TargetPermission),
			Ref:  schedulercontract.ScheduledTaskReadPermission.String(),
		},
	}
	if _, err := n.publisher.Publish(ctx, input); err != nil && n.logger != nil {
		n.logger.Warn("publish scheduler failure notification failed",
			zap.String("module", moduleID),
			zap.String("taskKey", run.TaskKey),
			zap.Uint64("runID", run.ID),
			zap.Error(err),
		)
	}
}

func schedulerRunFailureMetadata(run schedulercore.TaskRun) json.RawMessage {
	payload, _ := json.Marshal(map[string]any{
		"task_key":     run.TaskKey,
		"job_key":      run.JobKey,
		"trigger_type": string(run.TriggerType),
		"error":        run.Error,
		"result":       run.Result,
		"result_json":  run.ResultJSON,
		"duration_ms":  run.DurationMS,
		"started_at":   run.StartedAt,
		"finished_at":  run.FinishedAt,
	})
	return payload
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonZeroTime(value *time.Time, fallback time.Time) time.Time {
	if value != nil && !value.IsZero() {
		return value.UTC()
	}
	return fallback.UTC()
}
