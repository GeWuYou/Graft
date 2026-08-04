package build

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
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
	contexts   moduleapi.ApplicationBuildContextResolver
	tasks      moduleapi.TaskService
	docker     moduleapi.DockerImageBuildCapability
	repository buildstore.Repository
}

// ListJobs returns a bounded Build-owned job page.
func (s *Service) ListJobs(ctx context.Context, limit, offset int) (buildstore.ListResult, error) {
	if s == nil || s.repository == nil {
		return buildstore.ListResult{}, errors.New("build service is unavailable")
	}
	return s.repository.ListJobs(ctx, limit, offset)
}

// GetJob returns the Build-owned detail projection for one public build ID.
func (s *Service) GetJob(ctx context.Context, buildID string) (buildstore.JobProjection, error) {
	if s == nil || s.repository == nil {
		return buildstore.JobProjection{}, errors.New("build service is unavailable")
	}
	if strings.TrimSpace(buildID) == "" {
		return buildstore.JobProjection{}, errors.New("build id is required")
	}
	return s.repository.GetJobByBuildID(ctx, buildID)
}

// NewService 以 Project、Task 和 Container 的窄能力创建 Build 提交服务。
func NewService(contexts moduleapi.ApplicationBuildContextResolver, tasks moduleapi.TaskService, docker moduleapi.DockerImageBuildCapability, repository buildstore.Repository) (*Service, error) {
	if contexts == nil || tasks == nil || docker == nil || repository == nil {
		return nil, errors.New("build service dependencies are unavailable")
	}
	return &Service{contexts: contexts, tasks: tasks, docker: docker, repository: repository}, nil
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
	buildContext, err := s.resolveBuildContext(ctx, request.ApplicationID)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	buildID, err := newBuildID()
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	input, err := buildTaskInput(buildID)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	receipt, err := s.submitTask(ctx, request, input)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if err := s.freezeSnapshot(ctx, buildID, receipt.TaskID, buildContext, request); err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("freeze build job snapshot: %w", err)
	}
	return receipt, nil
}

func (s *Service) resolveBuildContext(ctx context.Context, applicationID uint64) (moduleapi.ApplicationBuildContext, error) {
	buildContext, err := s.contexts.ResolveApplicationBuildContext(ctx, applicationID)
	if err != nil {
		return moduleapi.ApplicationBuildContext{}, fmt.Errorf("resolve application build context: %w", err)
	}
	if !buildContext.CanBuild || buildContext.RuntimeProvider != "docker" {
		return moduleapi.ApplicationBuildContext{}, errors.New("application does not support Docker builds")
	}
	return buildContext, nil
}

func buildTaskInput(buildID string) (json.RawMessage, error) {
	input, err := json.Marshal(moduleapi.BuildTaskInput{BuildID: buildID})
	if err != nil {
		return nil, fmt.Errorf("marshal build task input: %w", err)
	}
	return input, nil
}

func (s *Service) submitTask(ctx context.Context, request SubmitRequest, input json.RawMessage) (moduleapi.TaskReceipt, error) {
	return s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{Type: buildTaskType, Owner: moduleapi.TaskOwner{Type: buildTaskOwnerType, ID: fmt.Sprintf("application:%d", request.ApplicationID)}, RequestedBy: request.RequestedBy, IdempotencyKey: request.IdempotencyKey, Input: input, Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{Key: "dockerfile-build", ExecutorType: buildStageExecutor, Input: input, RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile}}}})
}

func (s *Service) freezeSnapshot(ctx context.Context, buildID string, taskID uint64, buildContext moduleapi.ApplicationBuildContext, request SubmitRequest) error {
	return s.repository.CreateJob(ctx, buildstore.JobSnapshot{BuildID: buildID, TaskID: taskID, ApplicationID: buildContext.ApplicationID, ApplicationName: buildContext.DisplayName, WorkspaceRoot: buildContext.WorkspaceRoot, ContextPath: request.ContextPath, DockerfilePath: request.DockerfilePath, RuntimeTargetID: buildContext.RuntimeTargetID, RuntimeProvider: buildContext.RuntimeProvider, ImageRepository: request.ImageRepository, ImageTag: request.ImageTag, BuildArgs: request.BuildArgs, RequestedBy: request.RequestedBy})
}

func newBuildID() (string, error) {
	var value [13]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate build id: %w", err)
	}
	return fmt.Sprintf("build_%x", value), nil
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
