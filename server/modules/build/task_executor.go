package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

const artifactSettlementTimeout = 5 * time.Second

type dockerfileBuildExecutor struct {
	repository   buildstore.Repository
	docker       moduleapi.DockerImageBuildCapability
	targetDocker moduleapi.TargetBoundDockerImageBuildCapability
	mu           sync.Mutex
	cancels      map[uint64]context.CancelFunc
}

func (e *dockerfileBuildExecutor) Type() moduleapi.StageExecutorType { return buildStageExecutor }

//nolint:cyclop // Task execution 在同一 audited boundary 内维护 validation、cancellation、build 与 settlement。
func (e *dockerfileBuildExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	if e == nil || e.repository == nil || e.docker == nil {
		return errors.New("build executor is unavailable")
	}
	var input moduleapi.BuildTaskInput
	if err := json.Unmarshal(run.Input(), &input); err != nil {
		return fmt.Errorf("decode build task input: %w", err)
	}
	if input.BuildID == "" {
		return errors.New("build task snapshot identity is missing")
	}
	contextInfo, err := e.repository.GetJobByTaskID(ctx, run.TaskID())
	if err != nil {
		return err
	}
	if contextInfo.BuildID != input.BuildID {
		return errors.New("build task snapshot identity does not match task input")
	}
	if contextInfo.RuntimeTargetID > uint64(^uint64(0)>>1) {
		return errors.New("build runtime target identity is invalid")
	}
	commandCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancels[run.StageID()] = cancel
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.cancels, run.StageID()); e.mu.Unlock(); cancel() }()
	buildInput := moduleapi.DockerImageBuildInput{WorkspaceRoot: contextInfo.WorkspaceRoot, ContextPath: contextInfo.ContextPath, DockerfilePath: contextInfo.DockerfilePath, ImageRepository: contextInfo.ImageRepository, ImageTag: contextInfo.ImageTag, BuildArgs: contextInfo.BuildArgs}
	var result moduleapi.DockerImageBuildResult
	if e.targetDocker != nil {
		result, err = e.targetDocker.BuildImageOnTarget(commandCtx, int64(contextInfo.RuntimeTargetID), buildInput, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) }) //nolint:gosec // RuntimeTargetID is range-checked immediately above.
	} else {
		result, err = e.docker.BuildImage(commandCtx, buildInput, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) })
	}
	if err != nil {
		return err
	}
	// Docker 已成功后仍需保留短暂的结算预算，避免调用方取消丢失 Build 产物事实。
	settlementCtx, settlementCancel := context.WithTimeout(context.WithoutCancel(ctx), artifactSettlementTimeout)
	defer settlementCancel()
	return e.repository.SettleDockerArtifact(settlementCtx, run.TaskID(), result)
}

func (e *dockerfileBuildExecutor) Cancel(_ context.Context, run moduleapi.StageRun) error {
	e.mu.Lock()
	cancel := e.cancels[run.StageID()]
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

type buildTaskExecutorDependencies struct {
	provider             moduleapi.TargetBoundDockerBuildProvider
	snapshotDelivery     moduleapi.TargetBoundWorkspaceSnapshotDeliveryCapability
	conformance          moduleapi.TargetBoundProviderExecutionConformanceCapability
	builderTelemetry     moduleapi.RuntimeTargetBuilderTelemetryReader
	registry             moduleapi.RegistryPublicationResolver
	executionAdapter     moduleapi.RuntimeExecutionAdapter
	targets              moduleapi.BuildRuntimeTargetReader
	intent               IntentResolver
	artifactCopyRegistry moduleapi.RegistryArtifactCopyResolver
}

func registerBuildTaskExecutor(registrar moduleapi.TaskRuntimeRegistrar, repository buildstore.Repository, docker moduleapi.DockerImageBuildCapability, promotions *Service, capabilities ...any) error {
	if registrar == nil {
		return errors.New("build task registrar is unavailable")
	}
	targetDocker := targetBoundDockerFromCapabilities(docker, capabilities)
	dependencies := resolveBuildTaskExecutorDependencies(capabilities)
	if err := registrar.RegisterStageExecutor(&dockerfileBuildExecutor{repository: repository, docker: docker, targetDocker: targetDocker, cancels: make(map[uint64]context.CancelFunc)}); err != nil {
		return err
	}
	// V2 plan 在执行前已被接受并持久化。可选 capability 使 legacy-focused test
	// 仍可构造，而 production wiring 提供完整的 target/publish boundary。
	if err := registrar.RegisterStageExecutor(v2ExecutionPlanExecutor{repository: repository, provider: dependencies.provider, targetDocker: targetDocker, executionAdapter: dependencies.executionAdapter, snapshotDelivery: dependencies.snapshotDelivery, conformance: dependencies.conformance, builderTelemetry: dependencies.builderTelemetry, registry: dependencies.registry, targets: dependencies.targets, intents: dependencies.intent}); err != nil {
		return err
	}
	return registrar.RegisterStageExecutor(&artifactPromotionExecutor{service: promotions, adapter: dependencies.executionAdapter, registry: dependencies.artifactCopyRegistry, cancels: make(map[uint64]context.CancelFunc)})
}

func targetBoundDockerFromCapabilities(legacy moduleapi.DockerImageBuildCapability, capabilities []any) moduleapi.TargetBoundDockerImageBuildCapability {
	for _, capability := range capabilities {
		if value, ok := capability.(moduleapi.TargetBoundDockerImageBuildCapability); ok && value != nil {
			return value
		}
	}
	value, _ := legacy.(moduleapi.TargetBoundDockerImageBuildCapability)
	return value
}

//nolint:cyclop // 注册边界集中解析完整 provider 与其窄能力，避免调用方自行拼装第二套执行 graph。
func resolveBuildTaskExecutorDependencies(capabilities []any) buildTaskExecutorDependencies {
	dependencies := buildTaskExecutorDependencies{intent: newBuiltinBuildIntentRegistry()}
	for _, capability := range capabilities {
		switch value := capability.(type) {
		case moduleapi.TargetBoundDockerBuildProvider:
			dependencies.provider = value
		case moduleapi.TargetBoundWorkspaceSnapshotDeliveryCapability:
			dependencies.snapshotDelivery = value
		case moduleapi.TargetBoundProviderExecutionConformanceCapability:
			dependencies.conformance = value
		case moduleapi.RuntimeTargetBuilderTelemetryReader:
			dependencies.builderTelemetry = value
		case moduleapi.RegistryPublicationResolver:
			dependencies.registry = value
		case moduleapi.RuntimeExecutionAdapter:
			dependencies.executionAdapter = value
		case moduleapi.BuildRuntimeTargetReader:
			dependencies.targets = value
		case IntentResolver:
			if value != nil {
				dependencies.intent = value
			}
		case moduleapi.RegistryArtifactCopyResolver:
			dependencies.artifactCopyRegistry = value
		}
	}
	return dependencies
}

type artifactPromotionExecutor struct {
	service  *Service
	adapter  moduleapi.RuntimeExecutionAdapter
	registry moduleapi.RegistryArtifactCopyResolver
	mu       sync.Mutex
	cancels  map[uint64]context.CancelFunc
}

func (*artifactPromotionExecutor) Type() moduleapi.StageExecutorType {
	return artifactPromotionStageExecutor
}

//nolint:cyclop // 复制、取消和结算必须保持在同一个 Task Runtime Stage 信任边界内。
func (e *artifactPromotionExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	if e == nil || e.service == nil || e.adapter == nil || e.registry == nil {
		return errors.New("artifact promotion executor is unavailable")
	}
	var input moduleapi.ArtifactPromotionTaskInput
	if err := json.Unmarshal(run.Input(), &input); err != nil {
		return fmt.Errorf("decode artifact promotion task input: %w", err)
	}
	if input.RuntimeTargetID < 1 || input.Source.ArtifactID == "" || input.Source.PublicationID == "" || input.Destination.Kind == "" {
		return errors.New("artifact promotion task input is incomplete")
	}
	copy := moduleapi.AuthorizedArtifactCopy{Source: input.Source, Destination: input.Destination}
	binding, err := e.registry.ResolveArtifactCopyBinding(ctx, copy)
	if err != nil {
		return fmt.Errorf("resolve artifact promotion binding: %w", err)
	}
	commandCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancels[run.StageID()] = cancel
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.cancels, run.StageID()); e.mu.Unlock(); cancel() }()
	result, err := e.adapter.CopyOCIArtifact(commandCtx, input.RuntimeTargetID, moduleapi.OCIArtifactCopyInput{Source: input.Source, Destination: input.Destination}, binding, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) })
	if err != nil {
		return err
	}
	settlementCtx, settlementCancel := context.WithTimeout(context.WithoutCancel(ctx), artifactSettlementTimeout)
	defer settlementCancel()
	return e.service.SettleArtifactPromotion(settlementCtx, moduleapi.OCIArtifactCopyInput{Source: input.Source, Destination: input.Destination}, result, binding.Destination.AuthExecution)
}

func (e *artifactPromotionExecutor) Cancel(_ context.Context, run moduleapi.StageRun) error {
	e.mu.Lock()
	cancel := e.cancels[run.StageID()]
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

type v2ExecutionPlanExecutor struct {
	repository       buildstore.Repository
	provider         moduleapi.TargetBoundDockerBuildProvider
	targetDocker     moduleapi.TargetBoundDockerImageBuildCapability
	snapshotDelivery moduleapi.TargetBoundWorkspaceSnapshotDeliveryCapability
	conformance      moduleapi.TargetBoundProviderExecutionConformanceCapability
	builderTelemetry moduleapi.RuntimeTargetBuilderTelemetryReader
	registry         moduleapi.RegistryPublicationResolver
	executionAdapter moduleapi.RuntimeExecutionAdapter
	targets          moduleapi.BuildRuntimeTargetReader
	intents          IntentResolver
}

func (v2ExecutionPlanExecutor) Type() moduleapi.StageExecutorType { return v2BuildStageExecutor }

//nolint:cyclop,gocyclo,gocognit,funlen // v2 executor 是 frozen-plan validation、reservation lifecycle 与 target-bound publication 的唯一 audited boundary。
func (e v2ExecutionPlanExecutor) Execute(ctx context.Context, run moduleapi.StageRun) (err error) {
	reader, ok := e.repository.(buildstore.ExecutionPlanReader)
	if !ok || e.provider == nil || e.targetDocker == nil || e.executionAdapter == nil || e.registry == nil || e.snapshotDelivery == nil || e.conformance == nil {
		return errors.New("build execution plan requires target-bound build and registry publication capability")
	}
	var input moduleapi.BuildPlanTaskInput
	if err := json.Unmarshal(run.Input(), &input); err != nil {
		return fmt.Errorf("decode execution plan task input: %w", err)
	}
	plan, err := reader.GetExecutionPlanByTaskID(ctx, run.TaskID())
	if err != nil || plan.ID != input.ExecutionPlanID {
		if err != nil {
			return err
		}
		return errors.New("execution plan identity does not match task input")
	}
	if len(plan.Platforms) > 1 && input.Platform == "" && input.LegID == "" {
		return e.publishPlatformManifest(ctx, run, plan)
	}
	reservationRepository, reservationOK := e.repository.(moduleapi.BuilderReservationRepository)
	if !reservationOK {
		return errors.New("builder reservation repository is unavailable")
	}
	reservationLegID, reservationInstanceID, err := e.reservationIdentityForRun(ctx, plan, input, run.Attempt())
	if err != nil {
		return err
	}
	slotBudget, observedAt, err := e.reservationSlotBudgetForRun(plan, input)
	if err != nil {
		return err
	}
	cleanupReservation, err := beginBuilderReservation(ctx, reservationRepository, builderReservationStart{planID: plan.ID, taskID: run.TaskID(), instanceID: reservationInstanceID, legID: reservationLegID, attempt: run.Attempt(), slotBudget: slotBudget, observedAt: observedAt})
	if err != nil {
		return err
	}
	defer cleanupReservation(&err)
	if plan.RuntimeTargetID < 1 || e.intents == nil || !e.compatibleIntent(plan) {
		return errors.New("execution plan is not supported by the selected build driver")
	}
	placement, found := plan.PlacementForPlatform(plan.Platforms[0])
	if !found {
		return errors.New("execution plan placement is missing")
	}
	deliveryMode, err := e.deliveryMode(placement)
	if err != nil {
		return err
	}
	if len(plan.Platforms) > 1 {
		return e.executePlatformLeg(ctx, run, input, plan)
	}
	request := moduleapi.ProviderExecutionConformanceRequest{TargetID: plan.RuntimeTargetID, DriverRef: plan.Driver, Platform: plan.Platforms[0], SnapshotID: plan.Workspace.ID, ContentDigest: plan.Workspace.ContentDigest, DeliveryMode: deliveryMode}
	conformance, err := verifyProviderConformance(ctx, e.conformance, request)
	if err != nil {
		return err
	}
	if err := verifySnapshotDelivery(ctx, e.snapshotDelivery, plan.RuntimeTargetID, plan.Workspace, deliveryMode); err != nil {
		return err
	}
	if err := recordProviderExecutionEvidence(ctx, e.repository, plan, moduleapi.ProviderExecutionEvidence{TaskID: run.TaskID(), StageID: run.StageID(), TargetID: plan.RuntimeTargetID, Platform: plan.Platforms[0], Conformance: conformance}); err != nil {
		return err
	}
	commandCtx, cancel := context.WithCancel(ctx)
	result, err := e.targetDocker.BuildImageOnTarget(commandCtx, plan.RuntimeTargetID, moduleapi.DockerImageBuildInput{MaterializationRef: plan.Workspace.MaterializationRef, ContextPath: ".", DockerfilePath: "Dockerfile", ImageRepository: plan.Destination.RepositoryRef, ImageTag: plan.Destination.Reference, Platform: plan.Platforms[0]}, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) })
	if err != nil {
		cancel()
		return err
	}
	binding, err := e.registry.ResolvePublicationBinding(commandCtx, moduleapi.AuthorizedArtifactDestination(plan.Destination))
	if err != nil {
		cancel()
		return err
	}
	result, err = e.executionAdapter.PublishImage(commandCtx, plan.RuntimeTargetID, result, binding, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) })
	if err != nil {
		cancel()
		return err
	}
	cancel()
	return settleV2Artifact(ctx, e.repository, run.TaskID(), plan, result, binding.AuthExecution)
}

func (e v2ExecutionPlanExecutor) reservationSlotBudgetForRun(plan moduleapi.BuildExecutionPlan, input moduleapi.BuildPlanTaskInput) (int, time.Time, error) {
	platform := input.Platform
	if platform == "" && len(plan.Platforms) == 1 {
		platform = plan.Platforms[0]
	}
	placement, found := plan.PlacementForPlatform(platform)
	if !found {
		return 0, time.Time{}, errors.New("execution plan reservation placement is missing")
	}
	return reservationSlotBudget(placement)
}

// reservationIdentityForRun derives retry capacity inputs from the frozen plan and never selects a new Builder.
func (e v2ExecutionPlanExecutor) reservationIdentityForRun(ctx context.Context, plan moduleapi.BuildExecutionPlan, input moduleapi.BuildPlanTaskInput, attempt int) (string, string, error) {
	platform := input.Platform
	if platform == "" && len(plan.Platforms) == 1 {
		platform = plan.Platforms[0]
	}
	placement, found := plan.PlacementForPlatform(platform)
	if !found || strings.TrimSpace(platform) == "" {
		return "", "", errors.New("execution plan reservation leg is missing")
	}
	if attempt > 1 {
		if err := e.reconfirmFrozenDynamicPlacement(ctx, plan, placement); err != nil {
			return "", "", err
		}
	}
	return platform, placement.BuilderInstanceID, nil
}

// reconfirmFrozenDynamicPlacement 只验证已冻结 Placement 对应的 Runtime Target；重试不得重新读取 Pool 或选择其他目标。
//
//nolint:cyclop,gocyclo // Retry admission intentionally keeps all frozen identity, capability and telemetry fences together.
func (e v2ExecutionPlanExecutor) reconfirmFrozenDynamicPlacement(ctx context.Context, plan moduleapi.BuildExecutionPlan, placement moduleapi.BuilderPlacement) error {
	if !isDynamicSchedulingPolicy(placement.SchedulingPolicy) {
		return nil
	}
	if e.builderTelemetry == nil || e.targets == nil || placement.RuntimeTargetID < 1 {
		return errors.New("dynamic retry telemetry authority is unavailable")
	}
	frozen, err := frozenPlacementCapability(placement)
	if err != nil {
		return err
	}
	target, err := e.targets.ReadBuildTarget(ctx, placement.RuntimeTargetID)
	if err != nil {
		return fmt.Errorf("read frozen builder capability: %w", err)
	}
	if !target.Available || target.ProviderCapabilityProfile != frozen.profile || target.ProviderCapabilityVersion != frozen.version {
		return errors.New("frozen builder capability profile is no longer available")
	}
	requirement := buildCapabilityRequirementForResolvedPolicy(frozen.negotiation.DriverRef, plan.CachePolicy, plan.SecurityPolicy, placement.Platform)
	if fingerprintBuildCapabilityRequirement(requirement) != frozen.requirementFingerprint {
		return errors.New("frozen builder capability requirement is invalid")
	}
	negotiation, err := (staticCapabilityMatcher{}).MatchBuildCapability(requirement, capabilityForBuildTarget(target))
	if err != nil || !sameNegotiatedCapability(negotiation, frozen.negotiation) {
		return errors.New("frozen builder capability negotiation is no longer conformant")
	}
	conformant, err := e.builderTelemetry.ConformBuilderTelemetry(ctx, []int64{placement.RuntimeTargetID})
	if err != nil {
		return fmt.Errorf("reconfirm frozen builder telemetry: %w", err)
	}
	if !conformant {
		return errors.New("frozen builder placement is no longer dynamically conformant")
	}
	snapshots, err := e.builderTelemetry.ListBuilderTelemetry(ctx, []int64{placement.RuntimeTargetID})
	if err != nil {
		return fmt.Errorf("read frozen builder telemetry: %w", err)
	}
	if len(snapshots) != 1 || snapshots[0].TargetID != placement.RuntimeTargetID || snapshots[0].CapabilityProfile != frozen.profile || snapshots[0].CapabilityVersion != frozen.version || !snapshots[0].DynamicPlacementConformantAt(time.Now().UTC()) {
		return errors.New("frozen builder placement telemetry is stale or invalid")
	}
	return nil
}

type frozenPlacementCapabilityEvidence struct {
	requirementFingerprint string
	profile, version       string
	negotiation            moduleapi.NegotiatedCapability
}

func frozenPlacementCapability(placement moduleapi.BuilderPlacement) (frozenPlacementCapabilityEvidence, error) {
	var evidence struct {
		RequirementFingerprint string                         `json:"capability_requirement_fingerprint"`
		Profile                string                         `json:"capability_profile"`
		Version                string                         `json:"capability_version"`
		Negotiation            moduleapi.NegotiatedCapability `json:"capability_negotiation"`
	}
	if len(placement.SchedulingEvidence) == 0 || json.Unmarshal(placement.SchedulingEvidence, &evidence) != nil || evidence.RequirementFingerprint == "" || evidence.Profile == "" || evidence.Version == "" || evidence.Negotiation.ProviderCapabilityProfile != evidence.Profile || evidence.Negotiation.ProviderCapabilityVersion != evidence.Version || evidence.Negotiation.DriverRef == "" || evidence.Negotiation.SnapshotDeliveryMode == "" {
		return frozenPlacementCapabilityEvidence{}, errors.New("frozen builder capability evidence is invalid")
	}
	return frozenPlacementCapabilityEvidence{requirementFingerprint: evidence.RequirementFingerprint, profile: evidence.Profile, version: evidence.Version, negotiation: evidence.Negotiation}, nil
}

func capabilityForBuildTarget(target moduleapi.BuildRuntimeTargetSummary) moduleapi.BuildExecutionCapability {
	return moduleapi.BuildExecutionCapability{ProviderCapabilityProfile: target.ProviderCapabilityProfile, ProviderCapabilityVersion: target.ProviderCapabilityVersion, SupportedDrivers: append([]string(nil), target.SupportedDrivers...), SupportedPlatforms: append([]string(nil), target.SupportedPlatforms...), SnapshotDeliveryModes: append([]string(nil), target.SnapshotDeliveryModes...), Features: append([]string(nil), target.BuildFeatures...)}
}

func sameNegotiatedCapability(left, right moduleapi.NegotiatedCapability) bool {
	return left.ProviderCapabilityProfile == right.ProviderCapabilityProfile && left.ProviderCapabilityVersion == right.ProviderCapabilityVersion && left.DriverRef == right.DriverRef && left.SnapshotDeliveryMode == right.SnapshotDeliveryMode && slices.Equal(left.SatisfiedFeatures, right.SatisfiedFeatures) && slices.Equal(left.UnsatisfiedFeatures, right.UnsatisfiedFeatures) && maps.Equal(left.PreferredMissReasons, right.PreferredMissReasons) && maps.Equal(left.OptionalOmissionReasons, right.OptionalOmissionReasons)
}

type builderReservationStart struct {
	planID, instanceID, legID string
	taskID                    uint64
	attempt                   int
	slotBudget                int
	observedAt                time.Time
}

type slotAwareBuilderReservationRepository interface {
	ReserveBuilderAttemptWithCapacity(context.Context, moduleapi.BuilderReservation, int) (moduleapi.BuilderReservation, error)
}

type observationAwareBuilderReservationRepository interface {
	ReserveBuilderAttemptWithCapacityAfterObservation(context.Context, moduleapi.BuilderReservation, int, time.Time) (moduleapi.BuilderReservation, error)
}

//nolint:cyclop,gocyclo,nestif // 容量租约获取、续租与有界释放必须保留在同一 fencing 审计边界。
func beginBuilderReservation(ctx context.Context, repository moduleapi.BuilderReservationRepository, start builderReservationStart) (func(*error), error) {
	if repository == nil || strings.TrimSpace(start.planID) == "" || start.taskID == 0 || strings.TrimSpace(start.instanceID) == "" || strings.TrimSpace(start.legID) == "" || start.attempt < 1 || start.slotBudget < 1 {
		return nil, errors.New("builder reservation is invalid")
	}
	fence := buildstore.BuilderReservationFence(start.planID, start.taskID, start.legID, start.attempt)
	if start.attempt == 1 {
		if err := repository.MarkBuilderReservationRunning(ctx, start.taskID, start.legID, fence); err != nil {
			return nil, fmt.Errorf("start builder reservation: %w", err)
		}
	} else {
		now := time.Now().UTC()
		slotAwareRepository, ok := repository.(slotAwareBuilderReservationRepository)
		if !ok {
			return nil, errors.New("slot-aware builder reservation persistence is unavailable")
		}
		reservation := moduleapi.BuilderReservation{ID: fmt.Sprintf("reservation_%s_%s_%d", start.planID, start.legID, start.attempt), InstanceID: start.instanceID, PlanID: start.planID, TaskID: start.taskID, Attempt: start.attempt, LegID: start.legID, FenceToken: fence, State: moduleapi.BuilderReservationRunning, LeaseExpiresAt: now.Add(buildstore.BuilderReservationLeaseTTL), CreatedAt: now, UpdatedAt: now}
		if observationAware, ok := repository.(observationAwareBuilderReservationRepository); ok {
			if _, err := observationAware.ReserveBuilderAttemptWithCapacityAfterObservation(ctx, reservation, start.slotBudget, start.observedAt); err != nil {
				return nil, fmt.Errorf("reserve builder retry capacity: %w", err)
			}
		} else if _, err := slotAwareRepository.ReserveBuilderAttemptWithCapacity(ctx, reservation, start.slotBudget); err != nil {
			return nil, fmt.Errorf("reserve builder retry capacity: %w", err)
		}
	}
	cleanup := func(executionErr *error) {
		state := moduleapi.BuilderReservationReleased
		if executionErr != nil && *executionErr != nil {
			state = moduleapi.BuilderReservationAbandoned
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), artifactSettlementTimeout)
		defer cleanupCancel()
		if releaseErr := repository.ReleaseBuilderReservation(cleanupCtx, start.taskID, start.legID, fence, state); releaseErr != nil && executionErr != nil {
			*executionErr = errors.Join(*executionErr, fmt.Errorf("release builder reservation: %w", releaseErr))
		}
	}
	if err := repository.RenewBuilderReservation(ctx, start.taskID, start.legID, fence, time.Now().UTC().Add(buildstore.BuilderReservationLeaseTTL)); err != nil {
		renewErr := fmt.Errorf("renew builder reservation: %w", err)
		cleanup(&renewErr)
		return nil, renewErr
	}
	return cleanup, nil
}

func reservationSlotBudget(placement moduleapi.BuilderPlacement) (int, time.Time, error) {
	var evidence struct {
		ReservationSlotBudget int       `json:"reservation_slot_budget"`
		ReservationObservedAt time.Time `json:"reservation_observed_at"`
	}
	if len(placement.SchedulingEvidence) == 0 || json.Unmarshal(placement.SchedulingEvidence, &evidence) != nil || evidence.ReservationSlotBudget < 1 {
		return 0, time.Time{}, errors.New("execution plan placement has no frozen reservation slot budget")
	}
	if isDynamicSchedulingPolicy(placement.SchedulingPolicy) && evidence.ReservationObservedAt.IsZero() {
		return 0, time.Time{}, errors.New("dynamic execution plan placement has no frozen telemetry observation time")
	}
	return evidence.ReservationSlotBudget, evidence.ReservationObservedAt.UTC(), nil
}

func (e v2ExecutionPlanExecutor) executePlatformLeg(ctx context.Context, run moduleapi.StageRun, input moduleapi.BuildPlanTaskInput, plan moduleapi.BuildExecutionPlan) error {
	placement, err := validatePlatformLeg(plan, input, e.executionAdapter)
	if err != nil {
		return err
	}
	commandCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	result, err := e.buildAndPublishPlatformLeg(commandCtx, run, plan, input, placement)
	if err != nil {
		return err
	}
	return e.recordPlatformArtifact(ctx, run, plan, input, result)
}

func validatePlatformLeg(plan moduleapi.BuildExecutionPlan, input moduleapi.BuildPlanTaskInput, executionAdapter moduleapi.RuntimeExecutionAdapter) (moduleapi.BuilderPlacement, error) {
	if executionAdapter == nil || strings.TrimSpace(input.Platform) == "" || strings.TrimSpace(input.LegID) == "" || !containsString(plan.Platforms, input.Platform) {
		return moduleapi.BuilderPlacement{}, errors.New("multi-platform build leg is incomplete")
	}
	placement, found := plan.PlacementForPlatform(input.Platform)
	if !found {
		return moduleapi.BuilderPlacement{}, errors.New("multi-platform build placement is missing")
	}
	return placement, nil
}

func (e v2ExecutionPlanExecutor) buildAndPublishPlatformLeg(ctx context.Context, run moduleapi.StageRun, plan moduleapi.BuildExecutionPlan, input moduleapi.BuildPlanTaskInput, placement moduleapi.BuilderPlacement) (moduleapi.DockerImageBuildResult, error) {
	legTag := temporaryPlatformTag(plan.Destination.Reference, input.LegID)
	legDestination := moduleapi.AuthorizedArtifactDestination(plan.Destination)
	legDestination.Reference = legTag
	binding, err := e.registry.ResolvePublicationBinding(ctx, legDestination)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	deliveryMode, err := e.deliveryMode(placement)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	request := moduleapi.ProviderExecutionConformanceRequest{TargetID: placement.RuntimeTargetID, DriverRef: plan.Driver, Platform: input.Platform, SnapshotID: plan.Workspace.ID, ContentDigest: plan.Workspace.ContentDigest, DeliveryMode: deliveryMode}
	conformance, err := verifyProviderConformance(ctx, e.conformance, request)
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	if err := verifySnapshotDelivery(ctx, e.snapshotDelivery, placement.RuntimeTargetID, plan.Workspace, deliveryMode); err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	if err := recordProviderExecutionEvidence(ctx, e.repository, plan, moduleapi.ProviderExecutionEvidence{TaskID: run.TaskID(), StageID: run.StageID(), TargetID: placement.RuntimeTargetID, Platform: input.Platform, Conformance: conformance}); err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	result, err := e.targetDocker.BuildImageOnTarget(ctx, placement.RuntimeTargetID, moduleapi.DockerImageBuildInput{MaterializationRef: plan.Workspace.MaterializationRef, ContextPath: ".", DockerfilePath: "Dockerfile", ImageRepository: plan.Destination.RepositoryRef, ImageTag: legTag, Platform: input.Platform}, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) })
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	result, err = e.executionAdapter.PublishImage(ctx, placement.RuntimeTargetID, result, binding, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) })
	if err != nil {
		return moduleapi.DockerImageBuildResult{}, err
	}
	return result, nil
}

func (e v2ExecutionPlanExecutor) recordPlatformArtifact(ctx context.Context, run moduleapi.StageRun, plan moduleapi.BuildExecutionPlan, input moduleapi.BuildPlanTaskInput, result moduleapi.DockerImageBuildResult) error {
	digest, ok := normalizePlatformDigest(result.Digest)
	if !ok {
		return errors.New("platform build did not return a valid digest")
	}
	repository, ok := e.repository.(buildstore.PlatformArtifactRepository)
	if !ok {
		return errors.New("platform artifact repository is unavailable")
	}
	artifact := moduleapi.PlatformArtifact{LegID: input.LegID, Platform: input.Platform, Digest: digest, MediaType: "application/vnd.oci.image.manifest.v1+json", SizeBytes: result.SizeBytes, ProducedAt: time.Now().UTC()}
	settlementCtx, settlementCancel := context.WithTimeout(context.WithoutCancel(ctx), artifactSettlementTimeout)
	defer settlementCancel()
	if err := repository.RecordPlatformArtifact(settlementCtx, run.TaskID(), plan, artifact); err != nil {
		return err
	}
	return nil
}

// publishPlatformManifest 只由 Task Runtime 领取的聚合阶段调用。
// Build 保持 Artifact/Publication 结算 owner，但单个平台 leg 不能自行判断协调组是否完成。
func (e v2ExecutionPlanExecutor) publishPlatformManifest(ctx context.Context, run moduleapi.StageRun, plan moduleapi.BuildExecutionPlan) error {
	repository, ok := e.repository.(buildstore.PlatformArtifactRepository)
	if !ok {
		return errors.New("platform artifact repository is unavailable")
	}
	settler, ok := e.repository.(buildstore.OCIManifestSettlementRepository)
	if !ok {
		return errors.New("OCI manifest settlement is unavailable")
	}
	manifestInput, err := repository.PrepareOCIManifestPublication(ctx, run.TaskID(), plan)
	if err != nil {
		return err
	}
	finalBinding, err := e.registry.ResolvePublicationBinding(ctx, manifestInput.Destination)
	if err != nil {
		return err
	}
	manifest, err := e.executionAdapter.PublishManifest(ctx, plan.RuntimeTargetID, manifestInput, finalBinding, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) })
	if err != nil {
		return err
	}
	manifest.Digest, ok = normalizePlatformDigest(manifest.Digest)
	if !ok {
		return errors.New("OCI manifest publication did not return a valid digest")
	}
	settlementCtx, settlementCancel := context.WithTimeout(context.WithoutCancel(ctx), artifactSettlementTimeout)
	defer settlementCancel()
	return settler.SettleOCIManifestPublication(settlementCtx, run.TaskID(), plan, manifest, finalBinding.AuthExecution)
}

func (e v2ExecutionPlanExecutor) compatibleIntent(plan moduleapi.BuildExecutionPlan) bool {
	if e.intents == nil {
		return false
	}
	template, driver, err := e.intents.ResolveBuildIntent(plan.TemplateRef, plan.Driver)
	if err != nil || template.Ref != plan.TemplateRef || driver.Ref != plan.Driver || len(plan.Platforms) == 0 {
		return false
	}
	for _, platform := range plan.Platforms {
		if !containsString(driver.Platforms, platform) {
			return false
		}
	}
	return len(plan.Platforms) == 1 || plan.Driver == "docker-buildx@v1"
}

func (e v2ExecutionPlanExecutor) deliveryMode(placement moduleapi.BuilderPlacement) (string, error) {
	frozen, err := frozenPlacementCapability(placement)
	if err != nil {
		return "", err
	}
	return frozen.negotiation.SnapshotDeliveryMode, nil
}

func verifyProviderConformance(ctx context.Context, capability moduleapi.TargetBoundProviderExecutionConformanceCapability, request moduleapi.ProviderExecutionConformanceRequest) (moduleapi.ProviderExecutionConformanceResult, error) {
	if capability == nil || request.TargetID < 1 || strings.TrimSpace(request.DriverRef) == "" || strings.TrimSpace(request.Platform) == "" || strings.TrimSpace(request.SnapshotID) == "" || strings.TrimSpace(request.ContentDigest) == "" || strings.TrimSpace(request.DeliveryMode) == "" {
		return moduleapi.ProviderExecutionConformanceResult{}, errors.New("provider execution conformance input is incomplete")
	}
	result, err := capability.ConformProviderExecution(ctx, request)
	if err != nil {
		return moduleapi.ProviderExecutionConformanceResult{}, fmt.Errorf("provider execution conformance: %w", err)
	}
	if !completeProviderConformanceEvidence(result) {
		return moduleapi.ProviderExecutionConformanceResult{}, errors.New("provider execution conformance is incomplete")
	}
	return result, nil
}

func completeProviderConformanceEvidence(result moduleapi.ProviderExecutionConformanceResult) bool {
	return result.Executable && strings.TrimSpace(result.ProviderID) != "" && strings.TrimSpace(result.ConformanceVersion) != "" && result.SnapshotDeliveryProof && result.DriverExecutionProof && result.PublicationProof && result.CancellationProof && result.CleanupProof
}

func verifySnapshotDelivery(ctx context.Context, capability moduleapi.TargetBoundWorkspaceSnapshotDeliveryCapability, targetID int64, snapshot moduleapi.WorkspaceSnapshot, deliveryMode string) error {
	if !validSnapshotDeliveryInput(capability, targetID, snapshot, deliveryMode) {
		return errors.New("execution plan snapshot delivery input is incomplete")
	}
	result, err := capability.DeliverWorkspaceSnapshot(ctx, moduleapi.WorkspaceSnapshotDeliveryRequest{
		TargetID:           targetID,
		SnapshotID:         snapshot.ID,
		ContentDigest:      snapshot.ContentDigest,
		MaterializationRef: snapshot.MaterializationRef,
		DeliveryMode:       deliveryMode,
	})
	if err != nil {
		return fmt.Errorf("deliver workspace snapshot: %w", err)
	}
	if !matchesSnapshotDeliveryProof(result, targetID, snapshot, deliveryMode) {
		return errors.New("workspace snapshot delivery proof does not match execution plan")
	}
	return nil
}

func validSnapshotDeliveryInput(capability moduleapi.TargetBoundWorkspaceSnapshotDeliveryCapability, targetID int64, snapshot moduleapi.WorkspaceSnapshot, deliveryMode string) bool {
	return capability != nil && targetID > 0 && snapshot.ID != "" && snapshot.ContentDigest != "" && snapshot.MaterializationRef != "" && strings.TrimSpace(deliveryMode) != ""
}

func matchesSnapshotDeliveryProof(result moduleapi.WorkspaceSnapshotDeliveryResult, targetID int64, snapshot moduleapi.WorkspaceSnapshot, deliveryMode string) bool {
	return result.TargetID == targetID && result.SnapshotID == snapshot.ID && result.ContentDigest == snapshot.ContentDigest && result.DeliveryMode == deliveryMode
}
func (v2ExecutionPlanExecutor) Cancel(context.Context, moduleapi.StageRun) error { return nil }

var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

func normalizePlatformDigest(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if digestPattern.MatchString(value) {
		return value, true
	}
	if index := strings.LastIndex(value, "@sha256:"); index >= 0 {
		candidate := value[index+1:]
		if digestPattern.MatchString(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func temporaryPlatformTag(reference, legID string) string {
	reference = strings.TrimSpace(reference)
	legID = strings.NewReplacer("/", "-", ":", "-", "@", "-", "_", "-").Replace(strings.TrimSpace(legID))
	return reference + "-graft-" + legID
}

func settleV2Artifact(ctx context.Context, repository buildstore.Repository, taskID uint64, plan moduleapi.BuildExecutionPlan, result moduleapi.DockerImageBuildResult, authExecution moduleapi.RegistryAuthExecution) error {
	settler, ok := repository.(buildstore.V2ArtifactSettlementRepository)
	if !ok {
		return errors.New("v2 artifact settlement is unavailable")
	}
	return settler.SettleV2Artifact(ctx, taskID, plan, result, authExecution)
}

func recordProviderExecutionEvidence(ctx context.Context, repository buildstore.Repository, plan moduleapi.BuildExecutionPlan, evidence moduleapi.ProviderExecutionEvidence) error {
	recorder, ok := repository.(buildstore.ProviderExecutionEvidenceRepository)
	if !ok {
		return errors.New("provider execution evidence repository is unavailable")
	}
	return recorder.RecordProviderExecutionEvidence(ctx, plan, evidence)
}
