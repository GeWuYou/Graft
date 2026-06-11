// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MessageKey identifies a stable scheduled task module message key.
type MessageKey string

// String returns the canonical message key value.
func (k MessageKey) String() string {
	return string(k)
}

const (
	// ScheduledTaskMenuTitle identifies the localized title for the scheduled task menu.
	ScheduledTaskMenuTitle MessageKey = "menu.server.scheduled_tasks.title"
	// ScheduledTaskNotFound identifies missing scheduled task failures.
	ScheduledTaskNotFound MessageKey = "scheduled_task.not_found"
	// ScheduledTaskAlreadyRunning identifies duplicate manual run failures.
	ScheduledTaskAlreadyRunning MessageKey = "scheduled_task.already_running"
	// ScheduledTaskInvalidRequest identifies invalid scheduler management input.
	ScheduledTaskInvalidRequest MessageKey = "scheduled_task.invalid_request"
	// ScheduledTaskRunFailedNotificationTitle identifies scheduler failure notification titles.
	ScheduledTaskRunFailedNotificationTitle MessageKey = "scheduledTask.notification.runFailed.title"
	// ScheduledTaskRunFailedNotificationMessage identifies scheduler failure notification messages.
	ScheduledTaskRunFailedNotificationMessage MessageKey = "scheduledTask.notification.runFailed.message"
	// ScheduledTaskRunSucceededNotificationTitle identifies scheduler manual success notification titles.
	ScheduledTaskRunSucceededNotificationTitle MessageKey = "scheduledTask.notification.runSucceeded.title"
	// ScheduledTaskRunSucceededNotificationMessage identifies scheduler manual success notification messages.
	ScheduledTaskRunSucceededNotificationMessage MessageKey = "scheduledTask.notification.runSucceeded.message"
)
