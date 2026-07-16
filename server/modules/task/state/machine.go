// Package state 拥有 Task Runtime 的状态机校验，阻止持久化历史被非法迁移覆盖。
package state

import (
	"errors"
	"fmt"

	"graft/server/internal/moduleapi"
)

// ErrInvalidTransition 表示 Task 或 Stage 的生命周期迁移会违反状态机并改写历史。
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

// CanTransitionTask 判断持久化 Task 是否允许从当前状态迁移到目标状态。
func CanTransitionTask(from moduleapi.TaskStatus, to moduleapi.TaskStatus) bool {
	allowed, exists := taskTransitions[from]
	if !exists {
		return false
	}
	_, exists = allowed[to]
	return exists
}

// ValidateTaskTransition 拒绝会改写 Task 历史的非法迁移，并返回包含前后状态的错误。
func ValidateTaskTransition(from moduleapi.TaskStatus, to moduleapi.TaskStatus) error {
	if CanTransitionTask(from, to) {
		return nil
	}
	return fmt.Errorf("%w: task %q -> %q", ErrInvalidTransition, from, to)
}

// CanTransitionStage 判断持久化 Stage 是否允许当前生命周期迁移；failed 和 unknown 只能回到 pending 以触发受控重试。
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
