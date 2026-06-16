// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MessageKey identifies a stable container management message key.
type MessageKey string

// String returns the canonical message key value.
func (k MessageKey) String() string {
	return string(k)
}

const (
	// OperationsMenuTitle identifies the operations menu title.
	OperationsMenuTitle MessageKey = "menu.ops.title"
	// ContainerMenuTitle identifies the container management menu title.
	ContainerMenuTitle MessageKey = "menu.ops.container.title"
	// ContainerRuntimeDisabled identifies disabled runtime errors.
	ContainerRuntimeDisabled MessageKey = "ops.container.error.runtimeDisabled"
	// ContainerRuntimeSocketMissing identifies missing runtime socket errors.
	ContainerRuntimeSocketMissing MessageKey = "ops.container.error.runtimeSocketMissing"
	// ContainerRuntimePermissionDenied identifies runtime socket permission errors.
	ContainerRuntimePermissionDenied MessageKey = "ops.container.error.runtimePermissionDenied"
	// ContainerRuntimeUnavailable identifies unavailable runtime connection errors.
	ContainerRuntimeUnavailable MessageKey = "ops.container.error.runtimeUnavailable"
	// ContainerNotFound identifies missing container errors.
	ContainerNotFound MessageKey = "ops.container.error.containerNotFound"
	// ContainerInvalidRef identifies invalid container reference errors.
	ContainerInvalidRef MessageKey = "ops.container.error.invalidContainerRef"
	// ContainerInvalidListQuery identifies invalid list query parameter errors.
	ContainerInvalidListQuery MessageKey = "ops.container.error.invalidListQuery"
	// ContainerInvalidBatchAction identifies invalid batch action request errors.
	ContainerInvalidBatchAction MessageKey = "ops.container.error.invalidBatchAction"
	// ContainerInvalidState identifies invalid action state errors.
	ContainerInvalidState MessageKey = "ops.container.error.invalidState"
	// ContainerLogsTooLarge identifies log limit errors.
	ContainerLogsTooLarge MessageKey = "ops.container.error.logsTooLarge"
	// ContainerInvalidLogQuery identifies invalid log query parameter errors.
	ContainerInvalidLogQuery MessageKey = "ops.container.error.invalidLogQuery"
	// ContainerTimeout identifies runtime timeout errors.
	ContainerTimeout MessageKey = "ops.container.error.timeout"
	// ContainerDangerousActionsDisabled identifies disabled action errors.
	ContainerDangerousActionsDisabled MessageKey = "ops.container.error.dangerousActionsDisabled"
	// ContainerActionStartCompleted identifies successful start action responses.
	ContainerActionStartCompleted MessageKey = "ops.container.action.start.completed"
	// ContainerActionStopCompleted identifies successful stop action responses.
	ContainerActionStopCompleted MessageKey = "ops.container.action.stop.completed"
	// ContainerActionRestartCompleted identifies successful restart action responses.
	ContainerActionRestartCompleted MessageKey = "ops.container.action.restart.completed"
	// ContainerActionRemoveCompleted identifies successful remove action responses.
	ContainerActionRemoveCompleted MessageKey = "ops.container.action.remove.completed"
	// ContainerBatchActionCompleted identifies fully successful batch action responses.
	ContainerBatchActionCompleted MessageKey = "ops.container.action.batch.completed"
	// ContainerBatchActionPartial identifies partially successful batch action responses.
	ContainerBatchActionPartial MessageKey = "ops.container.action.batch.partial"
	// ContainerBatchActionFailed identifies fully failed batch action responses.
	ContainerBatchActionFailed MessageKey = "ops.container.action.batch.failed"
)
