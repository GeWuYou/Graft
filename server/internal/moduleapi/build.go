package moduleapi

import (
	"context"
	"database/sql"
	"encoding/json"
)

// ApplicationBuildContext 是 Project 模块为构建消费者冻结的、经授权的应用来源上下文。
// 它只包含构建所需的公开身份和受控 workspace 根目录，不暴露 Application entity 或仓储。
type ApplicationBuildContext struct {
	ApplicationID       string
	ApplicationRecordID uint64
	DisplayName         string
	WorkspaceRoot       string
	RuntimeTargetID     uint64
	RuntimeTargetName   string
	RuntimeProvider     string
	CanBuild            bool
}

// ApplicationBuildContextResolver 解析当前操作者可访问的 Application 构建上下文。
type ApplicationBuildContextResolver interface {
	ResolveApplicationBuildContext(context.Context, string) (ApplicationBuildContext, error)
}

// DockerImageBuildInput 是 Container 模块接受的受控 Docker 构建请求。
// 路径必须相对于已授权 workspace，调用方不能传入 daemon、host 或任意 CLI 参数。
type DockerImageBuildInput struct {
	WorkspaceRoot   string
	ContextPath     string
	DockerfilePath  string
	ImageRepository string
	ImageTag        string
	BuildArgs       []DockerImageBuildArg
}

// DockerImageBuildArg 表示非敏感 Docker 构建参数。
type DockerImageBuildArg struct {
	Name  string
	Value string
}

// DockerImageBuildResult 是 Docker executor 生成的规范化镜像事实。
type DockerImageBuildResult struct {
	ImageID      string
	Digest       string
	Repository   string
	Tag          string
	SizeBytes    int64
	OS           string
	Architecture string
	Variant      string
}

// DockerImageBuildLogSink 接收经过 executor 限长和脱敏的逐行构建输出。
type DockerImageBuildLogSink func(context.Context, TaskLogEntry) error

// DockerImageBuildCapability 由 Container 模块提供 Docker image build 执行能力。
type DockerImageBuildCapability interface {
	BuildImage(context.Context, DockerImageBuildInput, DockerImageBuildLogSink) (DockerImageBuildResult, error)
}

// TaskBatchQueryService 为列表型消费者提供批量 Task 读取能力，避免其自行逐行查询。
// 它独立于 TaskQueryService，便于现有实现渐进接入而不破坏旧消费者。
type TaskBatchQueryService interface {
	GetTasksByIDs(context.Context, []uint64) ([]TaskView, error)
}

// TaskTransactionalSubmissionAdapter 绑定调用方拥有的 SQL transaction，写入 Task 及其 stages。
// 实现不得提交、回滚或在调用完成后继续使用 transaction。
type TaskTransactionalSubmissionAdapter interface {
	SubmitTask(context.Context, SubmitTaskInput) (TaskReceipt, error)
}

// TaskTransactionalSubmissionFactory 将 caller-owned transaction 绑定为 Task 写参与者。
type TaskTransactionalSubmissionFactory interface {
	BindTaskTransaction(*sql.Tx) (TaskTransactionalSubmissionAdapter, error)
}

// BuildTaskInput 返回 Build executor 在 Task metadata 中使用的稳定、非敏感输入载荷。
// 该辅助类型刻意不包含 build arg 值、凭据或绝对路径。
type BuildTaskInput struct {
	BuildID         string          `json:"build_id"`
	ApplicationID   string          `json:"application_id"`
	ContextPath     string          `json:"context_path"`
	DockerfilePath  string          `json:"dockerfile_path"`
	ImageRepository string          `json:"image_repository"`
	ImageTag        string          `json:"image_tag"`
	BuildArgs       json.RawMessage `json:"build_args,omitempty"`
}
