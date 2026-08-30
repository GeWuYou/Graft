package build

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

const (
	buildTaskOwnerType         = "build_job"
	buildSubmissionLeaseTTL    = 2 * time.Minute
	buildSubmissionDeadline    = 10 * time.Minute
	buildSubmissionRenewBefore = 30 * time.Second
	buildExternalLeaseTTL      = time.Minute
	buildExternalDeadline      = 2 * time.Hour
	buildExecutionProvider     = "docker"
	buildExecutionCapability   = "oci-build"
	buildExecutionVersion      = "docker/v1"
	buildExecutionProtocol     = "build-execution/v1"
	buildImagePublishOperation = "build.image.publish.v1"
	buildManifestOperation     = "build.manifest.publish.v1"
	buildArtifactCopyOperation = "build.artifact.copy.v1"
)

var (
	errInvalidBuildID      = errors.New("invalid build id")
	errInvalidBuildRequest = errors.New("invalid build submission")
)

// Service 拥有 Build 提交编排，并将工作区、任务状态和 Docker 执行委托给各自权威模块。
type Service struct {
	contexts          moduleapi.ApplicationBuildContextResolver
	inputSnapshots    moduleapi.BuildInputSnapshotReader
	submissions       moduleapi.TaskSubmissionService
	taskBatch         moduleapi.TaskBatchQueryService
	repository        buildstore.Repository
	snapshots         moduleapi.ApplicationWorkspaceSnapshotResolver
	buildTargets      moduleapi.BuildRuntimeTargetReader
	buildAssignments  moduleapi.RuntimeTargetBuildAssignmentReader
	builderTelemetry  moduleapi.RuntimeTargetBuilderTelemetryReader
	registry          moduleapi.RegistryDestinationResolver
	intents           IntentResolver
	workspaces        buildstore.WorkspaceRepository
	builderResources  moduleapi.BuilderResourceRepository
	promotionTasks    moduleapi.TaskService
	promotionRegistry moduleapi.RegistryArtifactCopyResolver
	promotionTargets  moduleapi.RuntimeTargetBuildAssignmentReader
}

// ListJobs 返回一个按 Build 快照和 Task 执行状态过滤的受限作业页。
func (s *Service) ListJobs(ctx context.Context, requestedBy uint64, query buildstore.ListQuery) (buildstore.ListResult, error) {
	if s == nil || s.repository == nil {
		return buildstore.ListResult{}, errors.New("build service is unavailable")
	}
	reader, hasV2 := s.repository.(buildstore.V2JobReader)
	var result buildstore.ListResult
	var err error
	if hasV2 {
		result, err = reader.ListV2Jobs(ctx, requestedBy, query)
	} else {
		result, err = s.repository.ListJobs(ctx, query)
	}
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

// ListInputSnapshots 返回当前用户可复用的 Build 输入快照分页。
func (s *Service) ListInputSnapshots(ctx context.Context, userID uint64, limit, offset int) (buildstore.InputSnapshotListResult, error) {
	if s == nil || s.repository == nil {
		return buildstore.InputSnapshotListResult{}, errors.New("build service is unavailable")
	}
	reader, ok := s.repository.(buildstore.InputSnapshotReader)
	if !ok {
		return buildstore.InputSnapshotListResult{}, errors.New("build input snapshot reader is unavailable")
	}
	return reader.ListBuildInputSnapshots(ctx, userID, limit, offset)
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

// ListArtifactPublications 返回指定 Artifact 的 Build-owned Publication 历史投影。
func (s *Service) ListArtifactPublications(ctx context.Context, artifactID string) ([]buildstore.ArtifactPublicationProjection, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("build service is unavailable")
	}
	reader, ok := s.repository.(buildstore.ArtifactPublicationProjectionReader)
	if !ok {
		return nil, errors.New("build artifact publication projection reading is unavailable")
	}
	return reader.ListArtifactPublications(ctx, artifactID)
}

// SettleArtifactPromotion 记录 provider 已证明的不可变 promotion 结果；Build 不解析 Registry 连接细节。
func (s *Service) SettleArtifactPromotion(ctx context.Context, input moduleapi.OCIArtifactCopyInput, result moduleapi.OCIArtifactCopyResult, authExecution moduleapi.RegistryAuthExecution) error {
	if s == nil || s.repository == nil {
		return errors.New("build promotion settlement is unavailable")
	}
	settler, ok := s.repository.(buildstore.ArtifactPromotionSettlementRepository)
	if !ok {
		return errors.New("build promotion settlement is unavailable")
	}
	return settler.SettleArtifactPromotion(ctx, input, result, authExecution)
}

// ArtifactPromotionRequest 只接受 Build 与 Registry 的稳定身份。来源 Publication
// 必须由 Build 读取，不能信任调用方提供的复制来源。
type ArtifactPromotionRequest struct {
	ArtifactID      string
	PublicationID   string
	Destination     moduleapi.BuildDestination
	RuntimeTargetID int64
	RequestedBy     uint64
	IdempotencyKey  string
}

// ConfigureArtifactPromotion 装配 Promotion 需要的 Task Runtime 提交和授权能力，
// 不向 Build service 暴露基础设施私有 binding。
func (s *Service) ConfigureArtifactPromotion(tasks moduleapi.TaskService, registry moduleapi.RegistryArtifactCopyResolver, targets moduleapi.RuntimeTargetBuildAssignmentReader) {
	if s == nil {
		return
	}
	s.promotionTasks, s.promotionRegistry, s.promotionTargets = tasks, registry, targets
}

// SubmitArtifactPromotion 在 Registry 授权两端后冻结 Build-owned digest 来源，
// 再委托既有 Task Runtime 执行。
//
//nolint:gocyclo,cyclop // 提交边界必须集中验证调用者、Build 来源、Registry 授权和 Runtime Target 授权。
func (s *Service) SubmitArtifactPromotion(ctx context.Context, request ArtifactPromotionRequest) (moduleapi.TaskReceipt, error) {
	if s == nil || s.promotionTasks == nil || s.promotionRegistry == nil || s.promotionTargets == nil {
		return moduleapi.TaskReceipt{}, errors.New("build artifact promotion dependencies are unavailable")
	}
	request.ArtifactID, request.PublicationID = strings.TrimSpace(request.ArtifactID), strings.TrimSpace(request.PublicationID)
	if request.ArtifactID == "" || request.PublicationID == "" || request.RuntimeTargetID < 1 {
		return moduleapi.TaskReceipt{}, errors.New("invalid artifact promotion request")
	}
	actorID := request.RequestedBy
	if actorID == 0 {
		if auth, ok := moduleapi.RequestAuthContextFromContext(ctx); ok && auth.User != nil {
			actorID = auth.User.ID
		}
	}
	if actorID == 0 {
		return moduleapi.TaskReceipt{}, moduleapi.ErrUnauthenticated
	}
	sources, err := s.ListArtifactPublicationSources(ctx, request.ArtifactID)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("list artifact publication sources: %w", err)
	}
	var source moduleapi.ArtifactPublicationSource
	for _, candidate := range sources {
		if candidate.PublicationID == request.PublicationID {
			source = candidate
			break
		}
	}
	if source.PublicationID == "" {
		return moduleapi.TaskReceipt{}, errors.New("artifact publication source is not found")
	}
	authorized, err := s.promotionRegistry.AuthorizeArtifactCopy(ctx, actorID, source, request.Destination)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("authorize artifact promotion: %w", err)
	}
	allowed, err := s.promotionTargets.CanUseBuildTarget(ctx, actorID, request.RuntimeTargetID)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("authorize promotion runtime target: %w", err)
	}
	if !allowed {
		return moduleapi.TaskReceipt{}, errors.New("promotion runtime target is not assigned to actor")
	}
	input, err := json.Marshal(moduleapi.ArtifactPromotionTaskInput{Source: authorized.Source, Destination: authorized.Destination, RuntimeTargetID: request.RuntimeTargetID})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal artifact promotion task input: %w", err)
	}
	task := moduleapi.SubmitTaskInput{
		Type:           artifactPromotionTaskType,
		Owner:          moduleapi.TaskOwner{Type: artifactPromotionTaskOwnerType, ID: source.PublicationID},
		RequestedBy:    actorID,
		IdempotencyKey: request.IdempotencyKey,
		Input:          input,
		Plan:           moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{Key: "copy-artifact", ExecutorType: artifactPromotionStageExecutor, Input: input, RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile, ExternalExecution: buildExternalExecution(request.RuntimeTargetID, buildArtifactCopyOperation, input)}}},
	}
	return s.promotionTasks.Submit(ctx, task)
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
func (s *Service) GetJob(ctx context.Context, requestedBy uint64, buildID string) (buildstore.JobProjection, error) {
	if s == nil || s.repository == nil {
		return buildstore.JobProjection{}, errors.New("build service is unavailable")
	}
	if strings.TrimSpace(buildID) == "" {
		return buildstore.JobProjection{}, errInvalidBuildID
	}
	var job buildstore.JobProjection
	var err error
	if reader, ok := s.repository.(buildstore.V2JobReader); ok {
		job, err = reader.GetV2JobByBuildID(ctx, requestedBy, buildID)
	} else {
		job, err = s.repository.GetJobByBuildID(ctx, buildID)
	}
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
	ListWorkspaces(context.Context, uint64, buildstore.WorkspaceListQuery) (buildstore.WorkspaceListResult, error)
}

// ListWorkspaces 返回当前调用者可选择的 Build-owned Workspace 投影。
func (s *Service) ListWorkspaces(ctx context.Context, requestedBy uint64, query buildstore.WorkspaceListQuery) (buildstore.WorkspaceListResult, error) {
	lister, ok := s.repository.(workspaceLister)
	if !ok {
		return buildstore.WorkspaceListResult{}, errors.New("build workspace listing is unavailable")
	}
	return lister.ListWorkspaces(ctx, requestedBy, query)
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
	return policy == "manual" || policy == "round_robin" || policy == "random"
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
func NewService(contexts moduleapi.ApplicationBuildContextResolver, submissions moduleapi.TaskSubmissionService, taskBatch moduleapi.TaskBatchQueryService, repository buildstore.Repository) (*Service, error) {
	if submissions == nil || taskBatch == nil || repository == nil {
		return nil, errors.New("build service dependencies are unavailable")
	}
	return &Service{contexts: contexts, submissions: submissions, taskBatch: taskBatch, repository: repository, intents: newBuiltinBuildIntentRegistry()}, nil
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

func buildExternalExecution(targetID int64, operation string, input json.RawMessage) *moduleapi.ExternalExecutionExpectation {
	digest := sha256.Sum256(input)
	return &moduleapi.ExternalExecutionExpectation{RuntimeTargetID: targetID, ProviderID: buildExecutionProvider, Capability: buildExecutionCapability, CapabilityVersion: buildExecutionVersion, Protocol: buildExecutionProtocol, OperationID: operation, PayloadSHA256: hex.EncodeToString(digest[:]), LeaseTTL: buildExternalLeaseTTL, AbsoluteDeadline: buildExternalDeadline}
}

func newBuildID() (string, error) {
	var value [13]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate build id: %w", err)
	}
	return fmt.Sprintf("build_%x", value), nil
}
