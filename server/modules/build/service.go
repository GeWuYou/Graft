package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
)

const (
	buildTaskType      = moduleapi.TaskType("build.dockerfile.v1")
	buildStageExecutor = moduleapi.StageExecutorType("build.dockerfile.v1")
	buildTaskOwnerType = "build_job"
)

// SubmitRequest 是 Dockerfile 构建提交的内部传输无关输入。
type SubmitRequest struct {
	ApplicationID   uint64
	ContextPath     string
	DockerfilePath  string
	ImageRepository string
	ImageTag        string
	BuildArgs       []moduleapi.DockerImageBuildArg
	RequestedBy     uint64
	IdempotencyKey  string
}

// Service 拥有 Build 提交编排，并将工作区、任务状态和 Docker 执行委托给各自权威模块。
type Service struct {
	contexts moduleapi.ApplicationBuildContextResolver
	tasks    moduleapi.TaskService
	docker   moduleapi.DockerImageBuildCapability
}

// NewService 以 Project、Task 和 Container 的窄能力创建 Build 提交服务。
func NewService(contexts moduleapi.ApplicationBuildContextResolver, tasks moduleapi.TaskService, docker moduleapi.DockerImageBuildCapability) (*Service, error) {
	if contexts == nil || tasks == nil || docker == nil {
		return nil, errors.New("build service dependencies are unavailable")
	}
	return &Service{contexts: contexts, tasks: tasks, docker: docker}, nil
}

// Submit 解析服务端授权工作区，并为 Build 请求创建单阶段的 Task 计划。
func (s *Service) Submit(ctx context.Context, request SubmitRequest) (moduleapi.TaskReceipt, error) {
	if s == nil || s.contexts == nil || s.tasks == nil {
		return moduleapi.TaskReceipt{}, errors.New("build service is unavailable")
	}
	request, err := normalizeSubmitRequest(request)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	buildContext, err := s.contexts.ResolveApplicationBuildContext(ctx, request.ApplicationID)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("resolve application build context: %w", err)
	}
	if !buildContext.CanBuild || buildContext.RuntimeProvider != "docker" {
		return moduleapi.TaskReceipt{}, errors.New("application does not support Docker builds")
	}
	input, err := json.Marshal(moduleapi.BuildTaskInput{ApplicationID: request.ApplicationID, ContextPath: request.ContextPath, DockerfilePath: request.DockerfilePath, ImageRepository: request.ImageRepository, ImageTag: request.ImageTag, BuildArgs: mustMarshalBuildArgs(request.BuildArgs)})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal build task input: %w", err)
	}
	return s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{Type: buildTaskType, Owner: moduleapi.TaskOwner{Type: buildTaskOwnerType, ID: fmt.Sprintf("application:%d", request.ApplicationID)}, RequestedBy: request.RequestedBy, IdempotencyKey: request.IdempotencyKey, Input: input, Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{Key: "dockerfile-build", ExecutorType: buildStageExecutor, Input: input, RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile}}}})
}

func normalizeSubmitRequest(request SubmitRequest) (SubmitRequest, error) {
	if request.ApplicationID == 0 || strings.TrimSpace(request.ImageRepository) == "" || strings.TrimSpace(request.ImageTag) == "" {
		return SubmitRequest{}, errors.New("invalid build submission")
	}
	var err error
	if request.ContextPath, err = normalizeBuildRelativePath(request.ContextPath); err != nil {
		return SubmitRequest{}, err
	}
	if request.DockerfilePath, err = normalizeBuildRelativePath(request.DockerfilePath); err != nil {
		return SubmitRequest{}, err
	}
	seen := make(map[string]struct{}, len(request.BuildArgs))
	for index := range request.BuildArgs {
		item := &request.BuildArgs[index]
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" || strings.ContainsAny(item.Name, "=\x00\r\n") {
			return SubmitRequest{}, errors.New("invalid build argument")
		}
		if _, ok := seen[item.Name]; ok {
			return SubmitRequest{}, errors.New("duplicate build argument")
		}
		seen[item.Name] = struct{}{}
	}
	return request, nil
}

func normalizeBuildRelativePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) || strings.ContainsAny(value, "\x00\r\n") {
		return "", errors.New("invalid build path")
	}
	value = filepath.Clean(value)
	if value == "." || value == ".." || strings.HasPrefix(value, ".."+string(filepath.Separator)) {
		return "", errors.New("build path escapes workspace")
	}
	return value, nil
}

func mustMarshalBuildArgs(args []moduleapi.DockerImageBuildArg) json.RawMessage {
	value, _ := json.Marshal(args)
	return value
}
