package build

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/distribution/reference"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

const (
	buildTaskType              = moduleapi.TaskType("build.dockerfile.v1")
	buildStageExecutor         = moduleapi.StageExecutorType("build.dockerfile.v1")
	buildTaskOwnerType         = "build_job"
	buildSubmissionLeaseTTL    = 2 * time.Minute
	buildSubmissionDeadline    = 10 * time.Minute
	buildSubmissionRenewBefore = 30 * time.Second
)

var (
	errInvalidBuildID      = errors.New("invalid build id")
	errInvalidBuildRequest = errors.New("invalid build submission")
)

// SubmitRequest 是 Dockerfile 构建提交的内部传输无关输入。
type SubmitRequest struct {
	ApplicationID   string
	ContextPath     string
	DockerfilePath  string
	ImageRepository string
	ImageTag        string
	BuildArgs       []moduleapi.DockerImageBuildArg
	RequestedBy     uint64
	IdempotencyKey  string
}

type preparedSubmission struct {
	request      SubmitRequest
	buildContext moduleapi.ApplicationBuildContext
	buildID      string
	input        json.RawMessage
}

// Service 拥有 Build 提交编排，并将工作区、任务状态和 Docker 执行委托给各自权威模块。
type Service struct {
	contexts         moduleapi.ApplicationBuildContextResolver
	submissions      moduleapi.TaskSubmissionService
	taskBatch        moduleapi.TaskBatchQueryService
	docker           moduleapi.DockerImageBuildCapability
	repository       buildstore.Repository
	snapshots        moduleapi.ApplicationWorkspaceSnapshotResolver
	buildTargets     moduleapi.BuildRuntimeTargetReader
	buildAssignments moduleapi.RuntimeTargetBuildAssignmentReader
	registry         moduleapi.RegistryDestinationResolver
	intents          IntentResolver
	workspaces       buildstore.WorkspaceRepository
	builderResources moduleapi.BuilderResourceRepository
}

// ListJobs 返回一个按 Build 快照和 Task 执行状态过滤的受限作业页。
func (s *Service) ListJobs(ctx context.Context, query buildstore.ListQuery) (buildstore.ListResult, error) {
	if s == nil || s.repository == nil {
		return buildstore.ListResult{}, errors.New("build service is unavailable")
	}
	result, err := s.repository.ListJobs(ctx, query)
	if err != nil {
		return buildstore.ListResult{}, err
	}
	return s.enrichJobs(ctx, result)
}

// ListArtifacts 返回与 Build Job 分离的不可变 v2 Artifact 投影。
func (s *Service) ListArtifacts(ctx context.Context, limit, offset int) (buildstore.V2ArtifactListResult, error) {
	if s == nil || s.repository == nil {
		return buildstore.V2ArtifactListResult{}, errors.New("build service is unavailable")
	}
	reader, ok := s.repository.(buildstore.V2ArtifactReader)
	if !ok {
		return buildstore.V2ArtifactListResult{}, errors.New("build artifact reading is unavailable")
	}
	return reader.ListV2Artifacts(ctx, limit, offset)
}

// ListArtifactPublicationSources 为 Promotion planning 提供 Build-owned Publication identity。
// Registry 仍负责 source/destination access authorization 和 private execution binding。
func (s *Service) ListArtifactPublicationSources(ctx context.Context, artifactID string) ([]moduleapi.ArtifactPublicationSource, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("build service is unavailable")
	}
	reader, ok := s.repository.(buildstore.ArtifactPublicationReader)
	if !ok {
		return nil, errors.New("build artifact publication reading is unavailable")
	}
	return reader.ListArtifactPublicationSources(ctx, artifactID)
}

func (s *Service) enrichJobs(ctx context.Context, result buildstore.ListResult) (buildstore.ListResult, error) {
	if s.taskBatch == nil || len(result.Items) == 0 {
		return result, nil
	}
	taskIDs := make([]uint64, 0, len(result.Items))
	for _, item := range result.Items {
		taskIDs = append(taskIDs, item.TaskID)
	}
	tasks, err := s.taskBatch.GetTasksByIDs(ctx, taskIDs)
	if err != nil {
		return buildstore.ListResult{}, fmt.Errorf("load build task execution projection: %w", err)
	}
	byID := make(map[uint64]moduleapi.TaskView, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}
	filtered := result.Items[:0]
	for index := range result.Items {
		item := &result.Items[index]
		task, ok := byID[item.TaskID]
		if !ok {
			continue
		}
		item.Execution = buildTaskExecution(task)
		filtered = append(filtered, *item)
	}
	result.Items = filtered
	return result, nil
}

// GetJob returns the Build-owned detail projection for one public build ID.
func (s *Service) GetJob(ctx context.Context, buildID string) (buildstore.JobProjection, error) {
	if s == nil || s.repository == nil {
		return buildstore.JobProjection{}, errors.New("build service is unavailable")
	}
	if strings.TrimSpace(buildID) == "" {
		return buildstore.JobProjection{}, errInvalidBuildID
	}
	job, err := s.repository.GetJobByBuildID(ctx, buildID)
	if err != nil {
		return buildstore.JobProjection{}, err
	}
	tasks, err := s.taskBatch.GetTasksByIDs(ctx, []uint64{job.TaskID})
	if err != nil {
		return buildstore.JobProjection{}, fmt.Errorf("load build task execution projection: %w", err)
	}
	if len(tasks) != 1 || tasks[0].ID != job.TaskID {
		return buildstore.JobProjection{}, buildstore.ErrNotFound
	}
	job.Execution = buildTaskExecution(tasks[0])
	return job, nil
}

// CreateWorkspace 创建 Build-owned 来源定义；Application source 的访问权仍由 Project resolver 校验。
func (s *Service) CreateWorkspace(ctx context.Context, name, sourceKind, sourceReference string, requestedBy uint64) (moduleapi.BuildWorkspace, error) {
	if s == nil || s.workspaces == nil || s.contexts == nil {
		return moduleapi.BuildWorkspace{}, errors.New("build workspace service is unavailable")
	}
	if sourceKind != moduleapi.WorkspaceSourceApplication || strings.TrimSpace(name) == "" || strings.TrimSpace(sourceReference) == "" {
		return moduleapi.BuildWorkspace{}, errors.New("unsupported build workspace source")
	}
	if _, err := s.contexts.ResolveApplicationBuildContext(ctx, strings.TrimSpace(sourceReference)); err != nil {
		return moduleapi.BuildWorkspace{}, fmt.Errorf("authorize application workspace source: %w", err)
	}
	buildID, err := newBuildID()
	if err != nil {
		return moduleapi.BuildWorkspace{}, err
	}
	workspace := moduleapi.BuildWorkspace{ID: "workspace_" + strings.TrimPrefix(buildID, "build_"), Name: strings.TrimSpace(name), SourceKind: sourceKind, SourceReference: strings.TrimSpace(sourceReference), RetentionPolicy: "workspace", CreatedBy: requestedBy, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := s.workspaces.CreateWorkspace(ctx, workspace); err != nil {
		return moduleapi.BuildWorkspace{}, err
	}
	return workspace, nil
}

type workspaceLister interface {
	ListWorkspaces(context.Context, uint64) ([]moduleapi.BuildWorkspace, error)
}

// ListWorkspaces 返回当前调用者可选择的 Build-owned Workspace 投影。
func (s *Service) ListWorkspaces(ctx context.Context, requestedBy uint64) ([]moduleapi.BuildWorkspace, error) {
	lister, ok := s.repository.(workspaceLister)
	if !ok {
		return nil, errors.New("build workspace listing is unavailable")
	}
	return lister.ListWorkspaces(ctx, requestedBy)
}

// ListBuildTargets 返回授权且具备 Build 能力的 Runtime Target 投影。
func (s *Service) ListBuildTargets(ctx context.Context, requestedBy uint64) ([]moduleapi.BuildRuntimeTargetSummary, error) {
	if s == nil || s.buildTargets == nil || s.buildAssignments == nil {
		return nil, errors.New("build target listing is unavailable")
	}
	return s.buildAssignments.ListAssignedBuildTargets(ctx, requestedBy)
}

type builderPoolLister interface {
	ListBuilderPools(context.Context) ([]moduleapi.BuilderPool, error)
}

// ListBuilderPools 返回可供执行计划选择的 Pool 身份投影。
func (s *Service) ListBuilderPools(ctx context.Context, requestedBy uint64) ([]moduleapi.BuilderPool, error) {
	lister, ok := s.repository.(builderPoolLister)
	if !ok || s.builderResources == nil || s.buildAssignments == nil {
		return nil, errors.New("builder pool listing is unavailable")
	}
	pools, err := lister.ListBuilderPools(ctx)
	if err != nil {
		return nil, err
	}
	pools = filterSupportedBuilderPools(pools)
	visible := make([]moduleapi.BuilderPool, 0, len(pools))
	for _, pool := range pools {
		members, memberErr := s.builderResources.ListBuilderPoolMembers(ctx, pool.ID)
		if memberErr != nil {
			return nil, memberErr
		}
		for _, member := range members {
			allowed, assignmentErr := s.buildAssignments.CanUseBuildTarget(ctx, requestedBy, member.RuntimeTargetID)
			if assignmentErr != nil {
				return nil, assignmentErr
			}
			if allowed {
				visible = append(visible, pool)
				break
			}
		}
	}
	return visible, nil
}

func supportedBuilderPoolPolicy(policy string) bool {
	return policy == "round_robin" || policy == "labels"
}

func filterSupportedBuilderPools(pools []moduleapi.BuilderPool) []moduleapi.BuilderPool {
	filtered := make([]moduleapi.BuilderPool, 0, len(pools))
	for _, pool := range pools {
		if supportedBuilderPoolPolicy(pool.SchedulingPolicy) {
			filtered = append(filtered, pool)
		}
	}
	return filtered
}

// NewService 以 Project、Task 和 Container 的窄能力创建 Build 提交服务。
func NewService(contexts moduleapi.ApplicationBuildContextResolver, submissions moduleapi.TaskSubmissionService, taskBatch moduleapi.TaskBatchQueryService, docker moduleapi.DockerImageBuildCapability, repository buildstore.Repository) (*Service, error) {
	if contexts == nil || submissions == nil || taskBatch == nil || docker == nil || repository == nil {
		return nil, errors.New("build service dependencies are unavailable")
	}
	return &Service{contexts: contexts, submissions: submissions, taskBatch: taskBatch, docker: docker, repository: repository, intents: newBuiltinBuildIntentRegistry()}, nil
}

// Submit 解析服务端授权工作区，并为 Build 请求创建单阶段的 Task 计划。
func (s *Service) Submit(ctx context.Context, request SubmitRequest) (moduleapi.TaskReceipt, error) {
	if s == nil || s.contexts == nil || s.submissions == nil {
		return moduleapi.TaskReceipt{}, errors.New("build service is unavailable")
	}
	prepared, err := s.prepareSubmission(ctx, request)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	taskInput := s.taskInput(prepared.request, prepared.input)
	handle, err := s.submissions.BeginSubmission(ctx, moduleapi.BeginTaskSubmissionInput{Task: taskInput, Policy: moduleapi.TaskSubmissionPolicy{LeaseTTL: buildSubmissionLeaseTTL, AbsoluteDeadline: buildSubmissionDeadline, RenewBefore: buildSubmissionRenewBefore, AllowRenew: true, PrerequisiteKind: "build.snapshot.v1"}})
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if handle.Submission.State == moduleapi.TaskSubmissionStateActivated && handle.Submission.TaskID != nil {
		return s.activatedSubmissionReceipt(ctx, *handle.Submission.TaskID)
	}
	snapshot := buildstore.JobSnapshot{BuildID: prepared.buildID, ApplicationID: prepared.buildContext.ApplicationID, ApplicationRecordID: prepared.buildContext.ApplicationRecordID, ApplicationName: prepared.buildContext.DisplayName, WorkspaceRoot: prepared.buildContext.WorkspaceRoot, ContextPath: prepared.request.ContextPath, DockerfilePath: prepared.request.DockerfilePath, RuntimeTargetID: prepared.buildContext.RuntimeTargetID, RuntimeTargetName: prepared.buildContext.RuntimeTargetName, RuntimeProvider: prepared.buildContext.RuntimeProvider, ImageRepository: prepared.request.ImageRepository, ImageTag: prepared.request.ImageTag, BuildArgs: prepared.request.BuildArgs, RequestedBy: prepared.request.RequestedBy}
	receipt, err := s.submissions.MaterializeSubmission(ctx, handle, taskInput, buildSubmissionWriter{repository: s.repository, snapshot: snapshot})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("materialize build submission: %w", err)
	}
	return receipt, nil
}

func (s *Service) activatedSubmissionReceipt(ctx context.Context, taskID uint64) (moduleapi.TaskReceipt, error) {
	tasks, err := s.taskBatch.GetTasksByIDs(ctx, []uint64{taskID})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("load idempotent build task receipt: %w", err)
	}
	if len(tasks) != 1 || tasks[0].ID != taskID {
		return moduleapi.TaskReceipt{}, buildstore.ErrNotFound
	}
	return moduleapi.TaskReceipt{TaskID: tasks[0].ID, Status: tasks[0].Status}, nil
}

func buildTaskExecution(task moduleapi.TaskView) buildstore.TaskExecution {
	completed := 0
	if task.Status == moduleapi.TaskStatusSuccess || task.Status == moduleapi.TaskStatusFailed || task.Status == moduleapi.TaskStatusCancelled {
		completed = 1
	}
	capabilities := moduleapi.TaskCapabilities{
		Cancel:      task.Status == moduleapi.TaskStatusPending || task.Status == moduleapi.TaskStatusReady || task.Status == moduleapi.TaskStatusScheduled || task.Status == moduleapi.TaskStatusRunning || task.Status == moduleapi.TaskStatusNeedsAttention,
		Retry:       task.Status == moduleapi.TaskStatusFailed || task.Status == moduleapi.TaskStatusNeedsAttention,
		DownloadLog: true,
	}
	var recoveryReason *string
	if task.Status == moduleapi.TaskStatusNeedsAttention {
		recoveryReason = task.FailureMessage
	}
	return buildstore.TaskExecution{Status: task.Status, CurrentStageKey: task.CurrentStageKey, StageCount: 1, CompletedStageCount: completed, DurationMS: task.DurationMS, FailureCode: task.FailureCode, FailureMessage: task.FailureMessage, RecoveryReason: recoveryReason, Capabilities: capabilities}
}

func (s *Service) prepareSubmission(ctx context.Context, request SubmitRequest) (preparedSubmission, error) {
	normalizedRequest, err := normalizeSubmitRequest(request)
	if err != nil {
		return preparedSubmission{}, err
	}
	buildContext, err := s.resolveBuildContext(ctx, normalizedRequest.ApplicationID)
	if err != nil {
		return preparedSubmission{}, err
	}
	buildID, err := newBuildID()
	if err != nil {
		return preparedSubmission{}, err
	}
	input, err := buildTaskInput(buildID)
	if err != nil {
		return preparedSubmission{}, err
	}
	return preparedSubmission{request: normalizedRequest, buildContext: buildContext, buildID: buildID, input: input}, nil
}

func (s *Service) resolveBuildContext(ctx context.Context, applicationID string) (moduleapi.ApplicationBuildContext, error) {
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

func (s *Service) taskInput(request SubmitRequest, input json.RawMessage) moduleapi.SubmitTaskInput {
	return moduleapi.SubmitTaskInput{Type: buildTaskType, Owner: moduleapi.TaskOwner{Type: buildTaskOwnerType, ID: "application:" + request.ApplicationID}, RequestedBy: request.RequestedBy, IdempotencyKey: request.IdempotencyKey, Input: input, Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{Key: "dockerfile-build", ExecutorType: buildStageExecutor, Input: input, RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile}}}}
}

type buildSubmissionWriter struct {
	repository buildstore.Repository
	snapshot   buildstore.JobSnapshot
}

func (w buildSubmissionWriter) MaterializeTaskSubmission(ctx context.Context, tx *sql.Tx, submission moduleapi.TaskSubmission) (string, error) {
	return w.repository.MaterializeSubmissionSnapshot(ctx, tx, submission, w.snapshot)
}

func newBuildID() (string, error) {
	var value [13]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate build id: %w", err)
	}
	return fmt.Sprintf("build_%x", value), nil
}

func normalizeSubmitRequest(request SubmitRequest) (SubmitRequest, error) {
	request.ImageRepository = strings.TrimSpace(request.ImageRepository)
	request.ImageTag = strings.TrimSpace(request.ImageTag)
	request.ApplicationID = strings.TrimSpace(request.ApplicationID)
	if request.ApplicationID == "" || !validDockerImageReference(request.ImageRepository, request.ImageTag) {
		return SubmitRequest{}, errInvalidBuildRequest
	}
	var err error
	if request.ContextPath, err = normalizeBuildRelativePath(request.ContextPath); err != nil {
		return SubmitRequest{}, fmt.Errorf("%w: %v", errInvalidBuildRequest, err)
	}
	if request.DockerfilePath, err = normalizeBuildRelativePath(request.DockerfilePath); err != nil {
		return SubmitRequest{}, fmt.Errorf("%w: %v", errInvalidBuildRequest, err)
	}
	seen := make(map[string]struct{}, len(request.BuildArgs))
	for index := range request.BuildArgs {
		item := &request.BuildArgs[index]
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" || strings.ContainsAny(item.Name, "=\x00\r\n") {
			return SubmitRequest{}, errInvalidBuildRequest
		}
		if _, ok := seen[item.Name]; ok {
			return SubmitRequest{}, errInvalidBuildRequest
		}
		seen[item.Name] = struct{}{}
	}
	return request, nil
}

func validDockerImageReference(repository, tag string) bool {
	named, err := reference.ParseNormalizedNamed(repository)
	if err != nil {
		return false
	}
	if _, hasTag := named.(reference.Tagged); hasTag {
		return false
	}
	if _, hasDigest := named.(reference.Canonical); hasDigest {
		return false
	}
	return reference.TagRegexp.FindString(tag) == tag
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
