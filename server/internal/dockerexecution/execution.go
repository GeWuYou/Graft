// Package dockerexecution 实现 Docker/Compose 与 OCI 的有界 SDK 操作，调度、身份和持久化由调用方拥有。
package dockerexecution

import (
	"context"
	"encoding/json"
)

// Environment 只持有当前执行所需的连接和临时材料解析能力。
type Environment struct {
	DockerSocket    string
	ResolveMaterial func(context.Context, Request) (Material, error)
}

// Request 是冻结操作的身份与输入，不授予调度或持久化权限。
type Request struct {
	Capability  string
	Protocol    string
	OperationID string
	Input       json.RawMessage
}

// Material 由领域 owner 按当前 attempt 临时解析，调用方不得记录其载荷。
type Material struct {
	Protocol string          `json:"protocol"`
	Payload  json.RawMessage `json:"payload"`
}

// Result 只返回受限终态、稳定失败码和经领域 owner 校验的临时结果。
type Result struct {
	Outcome      string
	FailureCode  string
	FailureStage string
	Protocol     string
	Payload      json.RawMessage
}

// Execute 执行固定协议中的单次 SDK 操作，错误以稳定代码返回。
func Execute(ctx context.Context, environment Environment, lease Request) Result {
	if environment.ResolveMaterial == nil && lease.Capability != "container_execution" {
		return failedExecution(failureInvalidIntent)
	}
	return executeProviderOperation(ctx, environment, lease)
}

// Decode 对 journal 和操作输入采用相同的严格 JSON 规则。
func Decode(payload []byte, destination any) error { return strictDecode(payload, destination) }
