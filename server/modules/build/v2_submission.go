package build

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

const (
	v2BuildTaskType                = moduleapi.TaskType("build.execution-plan.v2")
	v2BuildStageExecutor           = moduleapi.StageExecutorType("build.execution-plan.v2")
	artifactPromotionTaskType      = moduleapi.TaskType("build.artifact-promotion.v1")
	artifactPromotionStageExecutor = moduleapi.StageExecutorType("build.artifact-promotion.v1")
	artifactPromotionTaskOwnerType = "build_artifact_promotion"
	v2DockerfileTemplate           = "oci-dockerfile/default@v1"
	v2DockerEngineDriver           = "docker-engine@v1"
	v2OCIDestination               = "oci_registry"
)

// ExecutionPlanRequest is the v2 public-write input after HTTP binding. It
// contains stable references only; source paths, daemon endpoints and
// credentials are resolved within their authority boundaries.
type ExecutionPlanRequest struct {
	WorkspaceID       string
	BuilderPoolID     string
	RuntimeTargetID   int64
	BuilderPlacements []moduleapi.BuilderPlacement
	TemplateRef       string
	Driver            string
	Platforms         []string
	Destination       moduleapi.BuildDestination
	RequestedBy       uint64
	IdempotencyKey    string
}

// ConfigureV2Submission wires the Build authority dependencies needed by v2
// writes without changing legacy read projections.
func (s *Service) ConfigureV2Submission(snapshots moduleapi.ApplicationWorkspaceSnapshotResolver, targets moduleapi.BuildRuntimeTargetReader, assignments moduleapi.RuntimeTargetBuildAssignmentReader, registries ...moduleapi.RegistryDestinationResolver) {
	if s == nil {
		return
	}
	s.snapshots, s.buildTargets, s.buildAssignments = snapshots, targets, assignments
	if len(registries) > 0 {
		s.registry = registries[0]
	}
	if repository, ok := s.repository.(buildstore.WorkspaceRepository); ok {
		s.workspaces = repository
	}
	if repository, ok := s.repository.(moduleapi.BuilderResourceRepository); ok {
		s.builderResources = repository
	}
}

// SelectBuilderFromPool 通过 Build-owned Pool 选择一个 ready Builder，并在返回前
// 重新校验操作者、Runtime Target capability 与冻结的 Driver 意图。它不提交 Task；
// 后续 Pool write contract 必须将结果冻结到 Execution Plan。
//
//nolint:gocyclo,cyclop // 选择后必须在同一边界复核身份、授权、能力与平台兼容性。
func (s *Service) SelectBuilderFromPool(ctx context.Context, poolID, driverRef string, platforms []string) (moduleapi.BuilderInstance, error) {
	if len(platforms) == 0 {
		return moduleapi.BuilderInstance{}, errors.New("builder pool platform selection is empty")
	}
	pool, err := s.builderPool(ctx, poolID)
	if err != nil {
		return moduleapi.BuilderInstance{}, err
	}
	return s.selectBuilderFromPool(ctx, pool, driverRef, platforms)
}

// SelectBuilderPlacementsFromPool 为每个冻结平台独立选择 Builder。选择顺序和策略也会
// 一并冻结到 Execution Plan，后续 Task Runtime 绝不能重新从 Pool 取值。
func (s *Service) SelectBuilderPlacementsFromPool(ctx context.Context, poolID, driverRef string, platforms []string) ([]moduleapi.BuilderPlacement, error) {
	if s == nil || s.builderResources == nil {
		return nil, errors.New("builder pool resource authority is unavailable")
	}
	pool, err := s.builderResources.GetBuilderPool(ctx, strings.TrimSpace(poolID))
	if err != nil {
		return nil, fmt.Errorf("read builder pool: %w", err)
	}
	placements := make([]moduleapi.BuilderPlacement, 0, len(platforms))
	for _, platform := range platforms {
		instance, selectionErr := s.selectBuilderFromPool(ctx, pool, driverRef, []string{platform})
		if selectionErr != nil {
			return nil, fmt.Errorf("select builder for %s: %w", platform, selectionErr)
		}
		placements = append(placements, moduleapi.BuilderPlacement{Platform: platform, BuilderInstanceID: instance.ID, RuntimeTargetID: instance.RuntimeTargetID, SchedulingPolicy: pool.SchedulingPolicy, SchedulingEvidence: schedulingEvidence(pool)})
	}
	return placements, nil
}

func (s *Service) builderPool(ctx context.Context, poolID string) (moduleapi.BuilderPool, error) {
	if s == nil || s.builderResources == nil {
		return moduleapi.BuilderPool{}, errors.New("builder pool resource authority is unavailable")
	}
	pool, err := s.builderResources.GetBuilderPool(ctx, strings.TrimSpace(poolID))
	if err != nil {
		return moduleapi.BuilderPool{}, fmt.Errorf("read builder pool: %w", err)
	}
	return pool, nil
}

func (s *Service) selectBuilderFromPool(ctx context.Context, pool moduleapi.BuilderPool, driverRef string, platforms []string) (moduleapi.BuilderInstance, error) {
	if s == nil || s.builderResources == nil || s.buildTargets == nil || s.buildAssignments == nil || s.intents == nil {
		return moduleapi.BuilderInstance{}, errors.New("builder pool selection dependencies are unavailable")
	}
	_, driver, err := s.intents.ResolveBuildIntent(v2DockerfileTemplate, driverRef)
	if err != nil {
		return moduleapi.BuilderInstance{}, fmt.Errorf("resolve builder pool driver: %w", err)
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.User == nil {
		return moduleapi.BuilderInstance{}, moduleapi.ErrUnauthenticated
	}
	members, err := s.builderResources.ListBuilderPoolMembers(ctx, strings.TrimSpace(pool.ID))
	if err != nil {
		return moduleapi.BuilderInstance{}, fmt.Errorf("list builder pool members: %w", err)
	}
	return s.selectCompatibleBuilderFromPool(ctx, pool, members, driver.Ref, platforms, auth.User.ID)
}

func (s *Service) selectCompatibleBuilderFromPool(ctx context.Context, pool moduleapi.BuilderPool, members []moduleapi.BuilderInstance, driverRef string, platforms []string, actorID uint64) (moduleapi.BuilderInstance, error) {
	if pool.SchedulingPolicy == "labels" {
		return s.selectLabeledBuilder(ctx, pool.Selector, members, driverRef, platforms, actorID)
	}
	if pool.SchedulingPolicy != "round_robin" {
		return moduleapi.BuilderInstance{}, fmt.Errorf("builder pool scheduling policy %q is not supported", pool.SchedulingPolicy)
	}
	return s.selectRoundRobinBuilder(ctx, pool.ID, len(members), driverRef, platforms, actorID)
}

func (s *Service) selectLabeledBuilder(ctx context.Context, rawSelector json.RawMessage, members []moduleapi.BuilderInstance, driverRef string, platforms []string, actorID uint64) (moduleapi.BuilderInstance, error) {
	selector, err := decodeBuilderLabelSelector(rawSelector)
	if err != nil {
		return moduleapi.BuilderInstance{}, err
	}
	for _, instance := range members {
		if instance.Status == "ready" && matchesBuilderLabels(instance.Labels, selector) && s.builderInstanceSupportsPlan(ctx, instance, driverRef, platforms, actorID) {
			return instance, nil
		}
	}
	return moduleapi.BuilderInstance{}, errors.New("builder pool has no compatible labeled instance")
}

func (s *Service) selectRoundRobinBuilder(ctx context.Context, poolID string, memberCount int, driverRef string, platforms []string, actorID uint64) (moduleapi.BuilderInstance, error) {
	for range memberCount {
		instance, err := s.builderResources.SelectRoundRobinBuilderInstance(ctx, poolID)
		if err != nil {
			return moduleapi.BuilderInstance{}, fmt.Errorf("select builder pool instance: %w", err)
		}
		if s.builderInstanceSupportsPlan(ctx, instance, driverRef, platforms, actorID) {
			return instance, nil
		}
	}
	return moduleapi.BuilderInstance{}, errors.New("builder pool has no compatible assigned instance")
}

type builderLabelSelector struct {
	Labels map[string]string `json:"labels"`
}

func decodeBuilderLabelSelector(raw json.RawMessage) (map[string]string, error) {
	selector := builderLabelSelector{Labels: map[string]string{}}
	if len(raw) == 0 {
		return nil, errors.New("builder labels selector is empty")
	}
	if err := json.Unmarshal(raw, &selector); err != nil || selector.Labels == nil {
		return nil, errors.New("invalid builder labels selector")
	}
	if len(selector.Labels) == 0 {
		return nil, errors.New("builder labels selector is empty")
	}
	for key, value := range selector.Labels {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return nil, errors.New("invalid builder labels selector")
		}
	}
	return selector.Labels, nil
}

func schedulingEvidence(pool moduleapi.BuilderPool) json.RawMessage {
	if len(pool.Selector) == 0 {
		return json.RawMessage(`{}`)
	}
	return append(json.RawMessage(nil), pool.Selector...)
}

func matchesBuilderLabels(labels, selector map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

//nolint:cyclop // Builder compatibility must retain driver, assignment, target and snapshot-delivery gates together.
func (s *Service) builderInstanceSupportsPlan(ctx context.Context, instance moduleapi.BuilderInstance, driverRef string, platforms []string, actorID uint64) bool {
	selectedDriverRef := instance.DriverRef
	if !strings.Contains(selectedDriverRef, "@") && instance.DriverVersion != "" {
		selectedDriverRef += "@" + instance.DriverVersion
	}
	if !containsBuildRef([]string{selectedDriverRef}, driverRef) {
		return false
	}
	allowed, err := s.buildAssignments.CanUseBuildTarget(ctx, actorID, instance.RuntimeTargetID)
	if err != nil || !allowed {
		return false
	}
	target, err := s.buildTargets.ReadBuildTarget(ctx, instance.RuntimeTargetID)
	if err != nil {
		return false
	}
	return target.Available &&
		containsBuildRef(target.SupportedDrivers, driverRef) &&
		slices.Contains(target.WorkspaceLocalities, "build-snapshot") &&
		hasSnapshotDeliveryMode(target.SnapshotDeliveryModes) &&
		containsAll(target.SupportedPlatforms, platforms)
}

// SubmitExecutionPlan freezes an Application Workspace Snapshot and a
// provider-neutral Execution Plan before Task Runtime materializes execution.
//
//nolint:gocognit,gocyclo,cyclop,funlen // Submission keeps authorization, freezing and Task reservation in one auditable boundary.
func (s *Service) SubmitExecutionPlan(ctx context.Context, request ExecutionPlanRequest) (moduleapi.TaskReceipt, error) {
	if s == nil || s.snapshots == nil || s.buildTargets == nil || s.buildAssignments == nil || s.registry == nil || s.workspaces == nil {
		return moduleapi.TaskReceipt{}, errors.New("build v2 submission dependencies are unavailable")
	}
	request, err := normalizeExecutionPlanRequest(request)
	if err != nil {
		return moduleapi.TaskReceipt{}, errInvalidBuildRequest
	}
	if len(request.Platforms) > 1 && strings.TrimSpace(request.BuilderPoolID) == "" {
		return moduleapi.TaskReceipt{}, errors.New("multi-platform build requires a Builder Pool")
	}
	if s.intents == nil {
		return moduleapi.TaskReceipt{}, errors.New("build template and driver authority is unavailable")
	}
	template, driver, err := s.intents.ResolveBuildIntent(request.TemplateRef, request.Driver)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("resolve build intent: %w", err)
	}
	request.TemplateRef, request.Driver = template.Ref, driver.Ref
	actorID := request.RequestedBy
	if actorID == 0 {
		if auth, ok := moduleapi.RequestAuthContextFromContext(ctx); ok && auth.User != nil {
			actorID = auth.User.ID
		}
	}
	authorizedDestination, err := s.registry.ResolveArtifactDestination(ctx, actorID, request.Destination)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("resolve artifact destination: %w", err)
	}
	request.Destination = moduleapi.BuildDestination(authorizedDestination)
	workspace, err := s.workspaces.GetWorkspace(ctx, request.WorkspaceID)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("resolve build workspace: %w", err)
	}
	if workspace.SourceKind != moduleapi.WorkspaceSourceApplication {
		return moduleapi.TaskReceipt{}, errors.New("workspace source is not supported by the selected builder")
	}
	snapshot, _, err := s.snapshots.FreezeApplicationWorkspaceSnapshot(ctx, workspace.SourceReference)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("freeze application workspace snapshot: %w", err)
	}
	snapshot.WorkspaceID = workspace.ID
	if snapshot.ID == "" || snapshot.ContentDigest == "" || snapshot.MaterializedRoot == "" {
		return moduleapi.TaskReceipt{}, errors.New("application workspace snapshot is incomplete")
	}
	selectedBuilderID := ""
	if workspaceSelection := strings.TrimSpace(request.BuilderPoolID); workspaceSelection != "" {
		placements, selectionErr := s.SelectBuilderPlacementsFromPool(ctx, workspaceSelection, request.Driver, request.Platforms)
		if selectionErr != nil {
			return moduleapi.TaskReceipt{}, selectionErr
		}
		request.BuilderPlacements = placements
		request.RuntimeTargetID = placements[0].RuntimeTargetID
		selectedBuilderID = placements[0].BuilderInstanceID
	} else {
		request.BuilderPlacements = make([]moduleapi.BuilderPlacement, 0, len(request.Platforms))
		selectedBuilderID = fmt.Sprintf("runtime-target:%d", request.RuntimeTargetID)
		for _, platform := range request.Platforms {
			request.BuilderPlacements = append(request.BuilderPlacements, moduleapi.BuilderPlacement{Platform: platform, BuilderInstanceID: selectedBuilderID, RuntimeTargetID: request.RuntimeTargetID, SchedulingPolicy: "manual"})
		}
	}
	for _, placement := range request.BuilderPlacements {
		if err := s.authorizeBuildPlacement(ctx, placement.RuntimeTargetID, request.Driver, placement.Platform); err != nil {
			return moduleapi.TaskReceipt{}, err
		}
	}
	// Project 仅负责来源授权和初次捕获；提交后由 Build 保留执行物化内容，
	// 后续保留策略不会改变 Snapshot identity。
	managedSnapshot, adoptErr := adoptSnapshotMaterialization(snapshot)
	_ = os.RemoveAll(snapshot.MaterializedRoot)
	if adoptErr != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("adopt workspace snapshot materialization: %w", adoptErr)
	}
	snapshot = managedSnapshot
	plan, err := freezeExecutionPlan(snapshot, request, selectedBuilderID)
	if err != nil {
		_ = os.RemoveAll(snapshot.MaterializedRoot)
		return moduleapi.TaskReceipt{}, err
	}
	input, err := json.Marshal(moduleapi.BuildPlanTaskInput{BuildID: plan.ID, ExecutionPlanID: plan.ID})
	if err != nil {
		_ = os.RemoveAll(snapshot.MaterializedRoot)
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal execution plan task input: %w", err)
	}
	stageTemplate := moduleapi.StagePlan{Key: "execution-plan", ExecutorType: v2BuildStageExecutor, Input: input, RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile}
	taskPlan := moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{stageTemplate}}
	if len(plan.Platforms) > 1 {
		legs := make([]moduleapi.CoordinatedLegPlan, 0, len(plan.Platforms))
		for index, platform := range plan.Platforms {
			legID := fmt.Sprintf("platform-%d", index+1)
			legInput, marshalErr := json.Marshal(moduleapi.BuildPlanTaskInput{BuildID: plan.ID, ExecutionPlanID: plan.ID, Platform: platform, LegID: legID})
			if marshalErr != nil {
				return moduleapi.TaskReceipt{}, fmt.Errorf("marshal coordinated build leg input: %w", marshalErr)
			}
			placement, found := plan.PlacementForPlatform(platform)
			if !found || placement.BuilderInstanceID == "" {
				return moduleapi.TaskReceipt{}, errors.New("coordinated build placement is incomplete")
			}
			legs = append(legs, moduleapi.CoordinatedLegPlan{ID: legID, Platform: platform, BuilderInstanceID: placement.BuilderInstanceID, RuntimeTargetID: placement.RuntimeTargetID, Input: legInput})
		}
		taskPlan.Coordination = &moduleapi.CoordinatedTaskPlan{Version: "build-legs/v1", AggregateStageKey: "build-platforms", Legs: legs}
	}
	task := moduleapi.SubmitTaskInput{
		Type:           v2BuildTaskType,
		Owner:          moduleapi.TaskOwner{Type: buildTaskOwnerType, ID: "workspace:" + request.WorkspaceID},
		RequestedBy:    request.RequestedBy,
		IdempotencyKey: request.IdempotencyKey,
		Input:          input,
		Plan:           taskPlan,
	}
	handle, err := s.submissions.BeginSubmission(ctx, moduleapi.BeginTaskSubmissionInput{Task: task, Policy: moduleapi.TaskSubmissionPolicy{LeaseTTL: buildSubmissionLeaseTTL, AbsoluteDeadline: buildSubmissionDeadline, RenewBefore: buildSubmissionRenewBefore, AllowRenew: true, PrerequisiteKind: "build.execution_plan.v2"}})
	if err != nil {
		_ = os.RemoveAll(snapshot.MaterializedRoot)
		return moduleapi.TaskReceipt{}, err
	}
	if handle.Submission.State == moduleapi.TaskSubmissionStateActivated && handle.Submission.TaskID != nil {
		_ = os.RemoveAll(snapshot.MaterializedRoot)
		return s.activatedSubmissionReceipt(ctx, *handle.Submission.TaskID)
	}
	repository, ok := s.repository.(buildstore.ExecutionPlanRepository)
	if !ok {
		_ = os.RemoveAll(snapshot.MaterializedRoot)
		return moduleapi.TaskReceipt{}, errors.New("build execution plan persistence is unavailable")
	}
	receipt, err := s.submissions.MaterializeSubmission(ctx, handle, task, executionPlanSubmissionWriter{repository: repository, plan: plan, requestedBy: request.RequestedBy})
	if err != nil {
		_ = os.RemoveAll(snapshot.MaterializedRoot)
	}
	return receipt, err
}

type executionPlanSubmissionWriter struct {
	repository  buildstore.ExecutionPlanRepository
	plan        moduleapi.BuildExecutionPlan
	requestedBy uint64
}

func (w executionPlanSubmissionWriter) MaterializeTaskSubmission(ctx context.Context, tx *sql.Tx, submission moduleapi.TaskSubmission) (string, error) {
	return w.repository.MaterializeExecutionPlan(ctx, tx, submission, w.plan, w.requestedBy)
}

//nolint:cyclop // Placement authorization intentionally rechecks identity, assignment and all target capability gates.
func (s *Service) authorizeBuildPlacement(ctx context.Context, runtimeTargetID int64, driverRef, platform string) error {
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.User == nil {
		return moduleapi.ErrUnauthenticated
	}
	allowed, err := s.buildAssignments.CanUseBuildTarget(ctx, auth.User.ID, runtimeTargetID)
	if err != nil {
		return fmt.Errorf("authorize build target: %w", err)
	}
	if !allowed {
		return errors.New("build target is not assigned to actor")
	}
	target, err := s.buildTargets.ReadBuildTarget(ctx, runtimeTargetID)
	if err != nil {
		return fmt.Errorf("read build target: %w", err)
	}
	if !target.Available || !containsBuildRef(target.SupportedDrivers, driverRef) || !slices.Contains(target.WorkspaceLocalities, "build-snapshot") || !hasSnapshotDeliveryMode(target.SnapshotDeliveryModes) || !containsString(target.SupportedPlatforms, platform) {
		return errors.New("build target is incompatible with execution plan")
	}
	return nil
}

func hasSnapshotDeliveryMode(modes []string) bool {
	for _, mode := range modes {
		if strings.TrimSpace(mode) != "" {
			return true
		}
	}
	return false
}

func containsBuildRef(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted || strings.TrimSuffix(value, "@v1") == strings.TrimSuffix(wanted, "@v1") {
			return true
		}
	}
	return false
}

//nolint:cyclop,gocyclo // 规范化必须在持久化前枚举所有调用方可控的计划引用。
func normalizeExecutionPlanRequest(request ExecutionPlanRequest) (ExecutionPlanRequest, error) {
	request.WorkspaceID, request.BuilderPoolID, request.TemplateRef, request.Driver = strings.TrimSpace(request.WorkspaceID), strings.TrimSpace(request.BuilderPoolID), strings.TrimSpace(request.TemplateRef), strings.TrimSpace(request.Driver)
	request.Destination.Kind = strings.TrimSpace(request.Destination.Kind)
	request.Destination.ConnectionRef = strings.TrimSpace(request.Destination.ConnectionRef)
	request.Destination.RepositoryRef = strings.TrimSpace(request.Destination.RepositoryRef)
	request.Destination.Reference = strings.TrimSpace(request.Destination.Reference)
	if request.WorkspaceID == "" || (request.RuntimeTargetID <= 0 && request.BuilderPoolID == "") || (request.RuntimeTargetID > 0 && request.BuilderPoolID != "") || request.TemplateRef == "" || request.Driver == "" || request.Destination.Kind != v2OCIDestination || request.Destination.ConnectionRef == "" || request.Destination.RepositoryRef == "" || request.Destination.Reference == "" || strings.ContainsAny(request.WorkspaceID+request.BuilderPoolID+request.Destination.RepositoryRef+request.Destination.Reference, "\x00\r\n") {
		return ExecutionPlanRequest{}, errors.New("invalid execution plan request")
	}
	if len(request.Platforms) == 0 {
		request.Platforms = []string{"linux/amd64"}
	}
	for index := range request.Platforms {
		request.Platforms[index] = strings.TrimSpace(request.Platforms[index])
		if request.Platforms[index] == "" {
			return ExecutionPlanRequest{}, errors.New("invalid execution plan platform")
		}
	}
	request.Platforms = slices.Compact(request.Platforms)
	return request, nil
}

func freezeExecutionPlan(snapshot moduleapi.WorkspaceSnapshot, request ExecutionPlanRequest, builderInstanceID string) (moduleapi.BuildExecutionPlan, error) {
	canonical := struct {
		SnapshotDigest    string                       `json:"snapshot_digest"`
		BuilderPoolID     string                       `json:"builder_pool_id,omitempty"`
		BuilderInstanceID string                       `json:"builder_instance_id,omitempty"`
		RuntimeTarget     int64                        `json:"runtime_target_id"`
		BuilderPlacements []moduleapi.BuilderPlacement `json:"builder_placements"`
		Template          string                       `json:"template_ref"`
		Driver            string                       `json:"driver"`
		Platforms         []string                     `json:"platforms"`
		Destination       moduleapi.BuildDestination   `json:"destination"`
	}{snapshot.ContentDigest, request.BuilderPoolID, builderInstanceID, request.RuntimeTargetID, append([]moduleapi.BuilderPlacement(nil), request.BuilderPlacements...), request.TemplateRef, request.Driver, append([]string(nil), request.Platforms...), request.Destination}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return moduleapi.BuildExecutionPlan{}, err
	}
	digest := sha256.Sum256(payload)
	digestText := hex.EncodeToString(digest[:])
	return moduleapi.BuildExecutionPlan{ID: "plan_" + digestText[:26], Digest: "sha256:" + digestText, Workspace: snapshot, BuilderPoolID: request.BuilderPoolID, BuilderInstanceID: builderInstanceID, RuntimeTargetID: request.RuntimeTargetID, BuilderPlacements: append([]moduleapi.BuilderPlacement(nil), request.BuilderPlacements...), Driver: request.Driver, TemplateRef: request.TemplateRef, Platforms: append([]string(nil), request.Platforms...), Destination: request.Destination, CreatedAt: time.Now().UTC()}, nil
}

func containsAll(supported, requested []string) bool {
	for _, item := range requested {
		if !slices.Contains(supported, item) {
			return false
		}
	}
	return true
}
