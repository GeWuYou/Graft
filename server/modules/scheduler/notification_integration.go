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
	schedulercontract "graft/server/modules/scheduler/contract"
)

const (
	schedulerNotificationSeverityInfo     moduleapi.NotificationSeverity       = "info"
	schedulerNotificationSeverityError    moduleapi.NotificationSeverity       = "error"
	schedulerNotificationCategoryTask     moduleapi.NotificationCategory       = "TASK"
	schedulerNotificationNavigationRun    moduleapi.NotificationNavigationKind = "SCHEDULER_RUN"
	schedulerNotificationTargetUser       moduleapi.NotificationTargetType     = "USER"
	schedulerNotificationTargetPermission moduleapi.NotificationTargetType     = "PERMISSION"
	schedulerTaskSucceededTitleKey                                             = "notification.title.scheduler.runSucceeded"
	schedulerTaskSucceededMessageKey                                           = "notification.message.scheduler.runSucceeded"
	schedulerTaskSucceededActionLabelKey                                       = "notification.action.openRunRecord"
	schedulerTaskCategoryKey                                                   = "notification.category.task"
	schedulerTaskSourceKey                                                     = "notification.source.scheduler"
	schedulerInfoLevelKey                                                      = "notification.level.info"
	schedulerTaskSucceededEventTypeKey                                         = "notification.event.taskSucceeded"
	schedulerTaskRunResourceTypeKey                                            = "notification.resourceType.scheduledTaskRun"
)

type schedulerRunFailureNotifier struct {
	publisher moduleapi.NotificationPublisher
	logger    *zap.Logger
}

func (n schedulerRunFailureNotifier) NotifyRunFailed(ctx context.Context, run schedulercore.TaskRun) {
	if n.publisher == nil || run.ID == 0 {
		return
	}
	payload := schedulerRunNavigationPayload(run, n.logger)
	input := moduleapi.PublishNotificationInput{
		TitleKey:     schedulercontract.ScheduledTaskRunFailedNotificationTitle.String(),
		Title:        "Scheduled task failed",
		MessageKey:   schedulercontract.ScheduledTaskRunFailedNotificationMessage.String(),
		Message:      "Scheduled task " + firstNonEmptyTrimmed(run.TaskName, run.TaskKey) + " failed.",
		Severity:     schedulerNotificationSeverityError,
		Category:     schedulerNotificationCategoryTask,
		SourceModule: moduleID,
		EventType:    "task_failed",
		ResourceType: "scheduled_task_run",
		ResourceID:   strconv.FormatUint(run.ID, 10),
		ResourceName: firstNonEmptyTrimmed(run.TaskName, run.TaskKey),
		Navigation: moduleapi.NotificationNavigation{
			Kind:    schedulerNotificationNavigationRun,
			Payload: payload,
		},
		Metadata:   schedulerRunFailureMetadata(run, n.logger),
		DedupeKey:  "scheduler:run_failed:" + strconv.FormatUint(run.ID, 10),
		OccurredAt: firstNonZeroTime(run.FinishedAt, run.CreatedAt),
		Target: moduleapi.NotificationTarget{
			Type: schedulerNotificationTargetPermission,
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

type schedulerRunSuccessNotifier struct {
	publisher moduleapi.NotificationPublisher
	logger    *zap.Logger
}

func (n schedulerRunSuccessNotifier) NotifyRunSucceeded(ctx context.Context, run schedulercore.TaskRun, trigger schedulercore.RunTrigger) {
	if n.publisher == nil || run.ID == 0 {
		if n.logger != nil {
			n.logger.Debug("skip scheduler success notification without publisher or run id",
				zap.String("module", moduleID),
				zap.String("taskKey", run.TaskKey),
				zap.Uint64("runID", run.ID),
			)
		}
		return
	}
	if trigger.TriggerUserID == 0 {
		if n.logger != nil {
			n.logger.Debug("skip scheduler success notification without trigger user",
				zap.String("module", moduleID),
				zap.String("taskKey", run.TaskKey),
				zap.Uint64("runID", run.ID),
			)
		}
		return
	}
	payload := schedulerRunNavigationPayload(run, n.logger)
	metadata := schedulerRunSuccessMetadata(run, trigger, n.logger)
	taskName := firstNonEmptyTrimmed(run.TaskName, run.TaskKey)
	input := moduleapi.PublishNotificationInput{
		TitleKey:        schedulerTaskSucceededTitleKey,
		Title:           taskName,
		MessageKey:      schedulerTaskSucceededMessageKey,
		Message:         "Completed successfully.",
		CategoryKey:     schedulerTaskCategoryKey,
		SourceKey:       schedulerTaskSourceKey,
		LevelKey:        schedulerInfoLevelKey,
		EventTypeKey:    schedulerTaskSucceededEventTypeKey,
		ResourceTypeKey: schedulerTaskRunResourceTypeKey,
		ActionLabelKey:  schedulerTaskSucceededActionLabelKey,
		ActionLabel:     "Open scheduled task run",
		Severity:        schedulerNotificationSeverityInfo,
		Category:        schedulerNotificationCategoryTask,
		SourceModule:    moduleID,
		EventType:       "task_succeeded",
		ResourceType:    "scheduled_task_run",
		ResourceID:      strconv.FormatUint(run.ID, 10),
		ResourceName:    taskName,
		Navigation: moduleapi.NotificationNavigation{
			Kind:    schedulerNotificationNavigationRun,
			Payload: payload,
		},
		Metadata:   metadata,
		DedupeKey:  "scheduler:run_succeeded:" + strconv.FormatUint(run.ID, 10),
		OccurredAt: firstNonZeroTime(run.FinishedAt, run.CreatedAt),
		Target: moduleapi.NotificationTarget{
			Type: schedulerNotificationTargetUser,
			Ref:  strconv.FormatUint(trigger.TriggerUserID, 10),
		},
	}
	result, err := n.publisher.Publish(ctx, input)
	if err != nil && n.logger != nil {
		n.logger.Warn("publish scheduler success notification failed",
			zap.String("module", moduleID),
			zap.String("taskKey", run.TaskKey),
			zap.Uint64("runID", run.ID),
			zap.Uint64("triggerUserID", trigger.TriggerUserID),
			zap.Error(err),
		)
		return
	}
	if n.logger != nil {
		n.logger.Debug("publish scheduler success notification completed",
			zap.String("module", moduleID),
			zap.String("taskKey", run.TaskKey),
			zap.Uint64("runID", run.ID),
			zap.Uint64("triggerUserID", trigger.TriggerUserID),
			zap.String("dedupeKey", input.DedupeKey),
			zap.Uint64("notificationEventID", result.EventID),
			zap.Int("recipientCount", result.RecipientCount),
			zap.Bool("skipped", result.Skipped),
			zap.Bool("deduplicated", result.Deduplicated),
		)
	}
}

func schedulerRunSuccessMetadata(run schedulercore.TaskRun, trigger schedulercore.RunTrigger, logger *zap.Logger) json.RawMessage {
	payload, err := json.Marshal(map[string]any{
		"taskNameKey":   strings.TrimSpace(run.TaskNameKey),
		"taskName":      firstNonEmptyTrimmed(run.TaskName, run.TaskKey),
		"taskKey":       run.TaskKey,
		"jobKey":        run.JobKey,
		"runId":         run.ID,
		"triggerType":   string(trigger.Type),
		"resultSummary": firstNonEmptyTrimmed(run.Result, "success"),
	})
	if err != nil {
		if logger != nil {
			logger.Warn("marshal scheduler success notification metadata failed",
				zap.String("module", moduleID),
				zap.String("taskKey", run.TaskKey),
				zap.Uint64("runID", run.ID),
				zap.Error(err),
			)
		}
		return json.RawMessage(`{"serialization_error":true}`)
	}
	return payload
}

func schedulerRunFailureMetadata(run schedulercore.TaskRun, logger *zap.Logger) json.RawMessage {
	payload, err := json.Marshal(map[string]any{
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
	if err != nil {
		if logger != nil {
			logger.Warn("marshal scheduler notification metadata failed",
				zap.String("module", moduleID),
				zap.String("taskKey", run.TaskKey),
				zap.Uint64("runID", run.ID),
				zap.Error(err),
			)
		}
		return json.RawMessage(`{"serialization_error":true}`)
	}
	return payload
}

func schedulerRunNavigationPayload(run schedulercore.TaskRun, logger *zap.Logger) json.RawMessage {
	payload, err := json.Marshal(map[string]any{
		"run_id":   run.ID,
		"task_key": run.TaskKey,
		"job_key":  run.JobKey,
	})
	if err != nil {
		if logger != nil {
			logger.Warn("marshal scheduler notification navigation payload failed",
				zap.String("module", moduleID),
				zap.String("taskKey", run.TaskKey),
				zap.Uint64("runID", run.ID),
				zap.Error(err),
			)
		}
		return json.RawMessage(`{"serialization_error":true}`)
	}
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
