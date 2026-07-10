// Package state owns Task Runtime state-machine validation.
package state

import (
	"errors"
	"fmt"

	"graft/server/internal/moduleapi"
)

// ErrInvalidTransition reports a Task or Stage lifecycle transition that would rewrite history.
var ErrInvalidTransition = errors.New("invalid task runtime state transition")

var taskTransitions = map[moduleapi.TaskStatus]map[moduleapi.TaskStatus]struct{}{
	moduleapi.TaskStatusPending: {
		moduleapi.TaskStatusScheduled: {}, moduleapi.TaskStatusRunning: {}, moduleapi.TaskStatusCancelled: {},
	},
	moduleapi.TaskStatusScheduled: {
		moduleapi.TaskStatusRunning: {}, moduleapi.TaskStatusCancelled: {},
	},
	moduleapi.TaskStatusRunning: {
		moduleapi.TaskStatusSuccess: {}, moduleapi.TaskStatusFailed: {}, moduleapi.TaskStatusCancelled: {}, moduleapi.TaskStatusNeedsAttention: {},
	},
	moduleapi.TaskStatusFailed: {
		moduleapi.TaskStatusRunning: {},
	},
	moduleapi.TaskStatusNeedsAttention: {
		moduleapi.TaskStatusRunning: {}, moduleapi.TaskStatusCancelled: {},
	},
}

// CanTransitionTask reports whether a persisted Task transition is legal.
func CanTransitionTask(from moduleapi.TaskStatus, to moduleapi.TaskStatus) bool {
	allowed, exists := taskTransitions[from]
	if !exists {
		return false
	}
	_, exists = allowed[to]
	return exists
}

// 迁移非法时返回包装 ErrInvalidTransition 的错误，并包含源状态和目标状态。
func ValidateTaskTransition(from moduleapi.TaskStatus, to moduleapi.TaskStatus) error {
	if CanTransitionTask(from, to) {
		return nil
	}
	return fmt.Errorf("%w: task %q -> %q", ErrInvalidTransition, from, to)
}

// CanTransitionStage reports whether a persisted Stage transition is legal.
func CanTransitionStage(from moduleapi.StageStatus, to moduleapi.StageStatus) bool {
	switch from {
	case moduleapi.StageStatusPending:
		return to == moduleapi.StageStatusRunning || to == moduleapi.StageStatusSkipped || to == moduleapi.StageStatusCancelled
	case moduleapi.StageStatusRunning:
		return to == moduleapi.StageStatusSuccess || to == moduleapi.StageStatusFailed || to == moduleapi.StageStatusCancelled || to == moduleapi.StageStatusUnknown
	case moduleapi.StageStatusFailed, moduleapi.StageStatusUnknown:
		return to == moduleapi.StageStatusPending
	default:
		return false
	}
}

// ValidateStageTransition 验证阶段状态迁移是否符合允许的生命周期规则；非法迁移返回包含原状态和目标状态的错误。
func ValidateStageTransition(from moduleapi.StageStatus, to moduleapi.StageStatus) error {
	if CanTransitionStage(from, to) {
		return nil
	}
	return fmt.Errorf("%w: stage %q -> %q", ErrInvalidTransition, from, to)
}
