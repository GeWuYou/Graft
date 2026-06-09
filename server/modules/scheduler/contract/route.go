// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

const (
	// ScheduledTasksGroup identifies the scheduled task API route group.
	ScheduledTasksGroup = "/scheduled-tasks"
	// ScheduledTaskCollectionRoute identifies the scheduled task collection route fragment.
	ScheduledTaskCollectionRoute = ""
	// ScheduledTaskJobDefinitionsRoute identifies the creatable job definition collection route fragment.
	ScheduledTaskJobDefinitionsRoute = "/job-definitions"
	// ScheduledTaskJobDefinitionDetailRoute identifies one job definition route fragment.
	ScheduledTaskJobDefinitionDetailRoute = "/job-definitions/:jobKey"
	// ScheduledTaskDetailRoute identifies the scheduled task detail route fragment.
	ScheduledTaskDetailRoute = "/:taskKey"
	// ScheduledTaskEnableRoute identifies the scheduled task enable route fragment.
	ScheduledTaskEnableRoute = "/:taskKey/enable"
	// ScheduledTaskDisableRoute identifies the scheduled task disable route fragment.
	ScheduledTaskDisableRoute = "/:taskKey/disable"
	// ScheduledTaskRunRoute identifies the manual run route fragment.
	ScheduledTaskRunRoute = "/:taskKey/run"
	// ScheduledTaskActionRoute identifies one backend-defined task action route fragment.
	ScheduledTaskActionRoute = "/:taskKey/actions/:actionKey"
	// ScheduledTaskRunsRoute identifies the scheduled task run history route fragment.
	ScheduledTaskRunsRoute = "/:taskKey/runs"
	// ScheduledTaskRunDetailRoute identifies one run-history detail route fragment.
	ScheduledTaskRunDetailRoute = "/runs/:runID"
	// ScheduledTaskMenuPath identifies the canonical scheduled task menu path.
	ScheduledTaskMenuPath = "/server/scheduled-tasks"
)
