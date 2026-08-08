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
	"sort"
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
	CachePolicy       string
	SecurityPolicy    string
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

// ConfigureBuilderTelemetry 注入 Runtime Target 的窄化 Builder 遥测读取器，用于 Phase 4 动态 Placement。
// 读取器缺失时静态 Pool 策略仍保持可用，动态策略则 fail-closed。
func (s *Service) ConfigureBuilderTelemetry(telemetry moduleapi.RuntimeTargetBuilderTelemetryReader) {
	if s != nil {
		s.builderTelemetry = telemetry
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
	selection, selectionErr := s.selectBuilderFromPool(ctx, pool, driverRef, platforms, poolID+":"+driverRef+":"+strings.Join(platforms, ","))
	return selection.instance, selectionErr
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
		selection, selectionErr := s.selectBuilderFromPool(ctx, pool, driverRef, []string{platform}, poolID+":"+driverRef+":"+platform)
		if selectionErr != nil {
			return nil, fmt.Errorf("select builder for %s: %w", platform, selectionErr)
		}
		evidence, evidenceErr := freezePlacementNegotiation(selection.evidence, selection.authorization)
		if evidenceErr != nil {
			return nil, evidenceErr
		}
		placements = append(placements, moduleapi.BuilderPlacement{Platform: platform, BuilderInstanceID: selection.instance.ID, RuntimeTargetID: selection.instance.RuntimeTargetID, SchedulingPolicy: pool.SchedulingPolicy, SchedulingEvidence: evidence})
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

type staticBuilderSelection struct {
	instance      moduleapi.BuilderInstance
	evidence      json.RawMessage
	authorization placementCapabilityAuthorization
}

func (s *Service) selectBuilderFromPool(ctx context.Context, pool moduleapi.BuilderPool, driverRef string, platforms []string, seedMaterial string) (staticBuilderSelection, error) {
	if s == nil || s.builderResources == nil || s.buildTargets == nil || s.buildAssignments == nil || s.intents == nil {
		return staticBuilderSelection{}, errors.New("builder pool selection dependencies are unavailable")
	}
	_, driver, err := s.intents.ResolveBuildIntent(v2DockerfileTemplate, driverRef)
	if err != nil {
		return staticBuilderSelection{}, fmt.Errorf("resolve builder pool driver: %w", err)
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.User == nil {
		return staticBuilderSelection{}, moduleapi.ErrUnauthenticated
	}
	members, err := s.builderResources.ListBuilderPoolMembers(ctx, strings.TrimSpace(pool.ID))
	if err != nil {
		return staticBuilderSelection{}, fmt.Errorf("list builder pool members: %w", err)
	}
	return s.selectCompatibleBuilderFromPool(ctx, pool, members, staticPoolSelectionInput{DriverRef: driver.Ref, Platforms: platforms, ActorID: auth.User.ID, SeedMaterial: seedMaterial, MemberCount: len(members)})
}

type builderLabelSelector struct {
	Labels      map[string]string `json:"labels"`
	InstanceID  string            `json:"instance_id"`
	AffinityKey string            `json:"affinity_key"`
}

func decodeBuilderSelector(raw json.RawMessage) (builderLabelSelector, error) {
	selector := builderLabelSelector{Labels: map[string]string{}}
	if len(raw) == 0 {
		return selector, nil
	}
	if err := json.Unmarshal(raw, &selector); err != nil || selector.Labels == nil {
		return builderLabelSelector{}, errors.New("invalid builder pool selector")
	}
	for key, value := range selector.Labels {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return builderLabelSelector{}, errors.New("invalid builder pool selector")
		}
	}
	selector.InstanceID = strings.TrimSpace(selector.InstanceID)
	selector.AffinityKey = strings.TrimSpace(selector.AffinityKey)
	return selector, nil
}

type staticPlacementEvidence struct {
	PolicyID                         string                         `json:"policy_id"`
	PolicyVersion                    string                         `json:"policy_version"`
	Labels                           map[string]string              `json:"labels,omitempty"`
	CandidateFingerprint             string                         `json:"candidate_fingerprint"`
	SelectedInstanceID               string                         `json:"selected_instance_id"`
	Seed                             string                         `json:"seed,omitempty"`
	Cursor                           *int64                         `json:"cursor,omitempty"`
	ReservationSlotBudget            int                            `json:"reservation_slot_budget"`
	CapabilityRequirementFingerprint string                         `json:"capability_requirement_fingerprint"`
	CapabilityProfile                string                         `json:"capability_profile"`
	CapabilityVersion                string                         `json:"capability_version"`
	CapabilityNegotiation            moduleapi.NegotiatedCapability `json:"capability_negotiation"`
}

type staticPoolSelectionInput struct {
	DriverRef    string
	Platforms    []string
	ActorID      uint64
	SeedMaterial string
	MemberCount  int
}

type frozenBuilderTelemetryEvidence struct {
	TargetID              int64     `json:"target_id"`
	BuilderScope          string    `json:"builder_scope"`
	ProviderID            string    `json:"provider_id"`
	CapabilityProfile     string    `json:"capability_profile"`
	CapabilityVersion     string    `json:"capability_version"`
	Available             bool      `json:"available"`
	Running               int       `json:"running"`
	Queued                int       `json:"queued"`
	AllocatableSlots      int       `json:"allocatable_slots"`
	ObservedAt            time.Time `json:"observed_at"`
	ExpiresAt             time.Time `json:"expires_at"`
	SourceRef             string    `json:"source_ref"`
	Provenance            string    `json:"provenance"`
	Integrity             string    `json:"integrity"`
	AffinityKey           string    `json:"affinity_key,omitempty"`
	UnsupportedDimensions []string  `json:"unsupported_dimensions"`
}

type dynamicPlacementEvidence struct {
	PolicyID                         string                         `json:"policy_id"`
	PolicyVersion                    string                         `json:"policy_version"`
	CandidateFingerprint             string                         `json:"candidate_fingerprint"`
	SelectedInstanceID               string                         `json:"selected_instance_id"`
	Telemetry                        frozenBuilderTelemetryEvidence `json:"telemetry"`
	ReservationSlotBudget            int                            `json:"reservation_slot_budget"`
	ReservationObservedAt            time.Time                      `json:"reservation_observed_at"`
	CapabilityRequirementFingerprint string                         `json:"capability_requirement_fingerprint"`
	CapabilityProfile                string                         `json:"capability_profile"`
	CapabilityVersion                string                         `json:"capability_version"`
	CapabilityNegotiation            moduleapi.NegotiatedCapability `json:"capability_negotiation"`
}

type dynamicBuilderCandidate struct {
	instance      moduleapi.BuilderInstance
	telemetry     moduleapi.BuilderTelemetrySnapshot
	authorization placementCapabilityAuthorization
}

//nolint:cyclop // Static placement must keep policy admission, eligibility and frozen evidence in one authority boundary.
func (s *Service) selectCompatibleBuilderFromPool(ctx context.Context, pool moduleapi.BuilderPool, members []moduleapi.BuilderInstance, input staticPoolSelectionInput) (staticBuilderSelection, error) {
	if isDynamicSchedulingPolicy(pool.SchedulingPolicy) {
		return s.selectDynamicPoolCandidate(ctx, pool, members, input)
	}
	if pool.SchedulingPolicy != "manual" && pool.SchedulingPolicy != "round_robin" && pool.SchedulingPolicy != "random" {
		return staticBuilderSelection{}, fmt.Errorf("builder pool scheduling policy %q is not supported", pool.SchedulingPolicy)
	}
	selector, err := decodeBuilderSelector(pool.Selector)
	if err != nil {
		return staticBuilderSelection{}, err
	}
	candidates := make([]moduleapi.BuilderInstance, 0, len(members))
	authorizations := make(map[string]placementCapabilityAuthorization, len(members))
	for _, instance := range members {
		authorization, eligible := s.builderInstanceSupportsPlan(ctx, instance, input.DriverRef, input.Platforms, input.ActorID)
		if instance.Status == "ready" && matchesBuilderLabels(instance.Labels, selector.Labels) && eligible {
			candidates = append(candidates, instance)
			authorizations[instance.ID] = authorization
		}
	}
	if len(candidates) == 0 {
		return staticBuilderSelection{}, errors.New("builder pool has no compatible eligible instance")
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	fingerprint := staticCandidateFingerprint(pool, selector, candidates)
	evidence := staticPlacementEvidence{PolicyID: "build.pool." + pool.SchedulingPolicy, PolicyVersion: "v1", Labels: selector.Labels, CandidateFingerprint: fingerprint, ReservationSlotBudget: 1}
	selected, selectionEvidence, err := s.selectStaticPoolCandidate(ctx, pool, selector, candidates, input, fingerprint)
	if err != nil {
		return staticBuilderSelection{}, err
	}
	evidence.Seed, evidence.Cursor = selectionEvidence.Seed, selectionEvidence.Cursor
	evidence.SelectedInstanceID = selected.ID
	raw, err := json.Marshal(evidence)
	if err != nil {
		return staticBuilderSelection{}, fmt.Errorf("marshal static placement evidence: %w", err)
	}
	return staticBuilderSelection{instance: selected, evidence: raw, authorization: authorizations[selected.ID]}, nil
}

func isDynamicSchedulingPolicy(policy string) bool {
	return policy == "least_load" || policy == "capacity" || policy == "affinity"
}

// selectDynamicPoolCandidate admits each candidate only from fresh, provider-conformant Runtime Target telemetry.
// It freezes the observation used to rank the Builder so a retry never re-reads mutable telemetry or reselects placement.
func (s *Service) selectDynamicPoolCandidate(ctx context.Context, pool moduleapi.BuilderPool, members []moduleapi.BuilderInstance, input staticPoolSelectionInput) (staticBuilderSelection, error) {
	if s == nil || s.builderTelemetry == nil {
		return staticBuilderSelection{}, errors.New("dynamic builder placement telemetry authority is unavailable")
	}
	selector, err := decodeBuilderSelector(pool.Selector)
	if err != nil {
		return staticBuilderSelection{}, err
	}
	candidates := make([]dynamicBuilderCandidate, 0, len(members))
	for _, instance := range members {
		candidate, eligible, candidateErr := s.dynamicEligibleCandidate(ctx, pool.SchedulingPolicy, selector, instance, input)
		if candidateErr != nil {
			return staticBuilderSelection{}, candidateErr
		}
		if eligible {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return staticBuilderSelection{}, errors.New("builder pool has no dynamically conformant eligible instance")
	}
	less := dynamicCandidateLess(pool.SchedulingPolicy)
	sort.Slice(candidates, func(i, j int) bool { return less(candidates[i], candidates[j]) })
	selected := candidates[0]
	evidence := dynamicPlacementEvidence{
		PolicyID: "build.pool." + pool.SchedulingPolicy, PolicyVersion: "v1",
		CandidateFingerprint: dynamicCandidateFingerprint(pool, selector, candidates), SelectedInstanceID: selected.instance.ID,
		Telemetry: freezeBuilderTelemetryEvidence(selected.telemetry), ReservationSlotBudget: selected.telemetry.AllocatableSlots, ReservationObservedAt: selected.telemetry.ObservedAt.UTC(),
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		return staticBuilderSelection{}, fmt.Errorf("marshal dynamic placement evidence: %w", err)
	}
	return staticBuilderSelection{instance: selected.instance, evidence: raw, authorization: selected.authorization}, nil
}

// dynamicEligibleCandidate validates one Builder using only Runtime Target's provider-conformant telemetry authority.
//
//nolint:cyclop,gocyclo // Fail-closed eligibility must keep provider admission, freshness and policy constraints together.
func (s *Service) dynamicEligibleCandidate(ctx context.Context, policy string, selector builderLabelSelector, instance moduleapi.BuilderInstance, input staticPoolSelectionInput) (dynamicBuilderCandidate, bool, error) {
	authorization, eligible := s.builderInstanceSupportsPlan(ctx, instance, input.DriverRef, input.Platforms, input.ActorID)
	if instance.Status != "ready" || !matchesBuilderLabels(instance.Labels, selector.Labels) || !eligible {
		return dynamicBuilderCandidate{}, false, nil
	}
	admitted, err := s.builderTelemetry.ConformBuilderTelemetry(ctx, []int64{instance.RuntimeTargetID})
	if err != nil {
		return dynamicBuilderCandidate{}, false, fmt.Errorf("conform builder telemetry: %w", err)
	}
	if !admitted {
		return dynamicBuilderCandidate{}, false, nil
	}
	snapshots, err := s.builderTelemetry.ListBuilderTelemetry(ctx, []int64{instance.RuntimeTargetID})
	if err != nil {
		return dynamicBuilderCandidate{}, false, fmt.Errorf("read builder telemetry: %w", err)
	}
	if len(snapshots) != 1 || snapshots[0].TargetID != instance.RuntimeTargetID || snapshots[0].AllocatableSlots < 1 || !snapshots[0].DynamicPlacementConformantAt(time.Now().UTC()) || (policy == "affinity" && (selector.AffinityKey == "" || snapshots[0].AffinityKey != selector.AffinityKey)) {
		return dynamicBuilderCandidate{}, false, nil
	}
	target, err := s.buildTargets.ReadBuildTarget(ctx, instance.RuntimeTargetID)
	if err != nil {
		return dynamicBuilderCandidate{}, false, fmt.Errorf("read dynamic builder capability: %w", err)
	}
	if snapshots[0].CapabilityProfile != target.ProviderCapabilityProfile || snapshots[0].CapabilityVersion != target.ProviderCapabilityVersion {
		return dynamicBuilderCandidate{}, false, nil
	}
	return dynamicBuilderCandidate{instance: instance, telemetry: snapshots[0], authorization: authorization}, true, nil
}

func dynamicCandidateLess(policy string) func(dynamicBuilderCandidate, dynamicBuilderCandidate) bool {
	return func(left, right dynamicBuilderCandidate) bool {
		if policy == "capacity" && left.telemetry.AllocatableSlots != right.telemetry.AllocatableSlots {
			return left.telemetry.AllocatableSlots > right.telemetry.AllocatableSlots
		}
		leftLoad, rightLoad := left.telemetry.Running+left.telemetry.Queued, right.telemetry.Running+right.telemetry.Queued
		if policy != "capacity" && leftLoad != rightLoad {
			return leftLoad < rightLoad
		}
		if left.telemetry.AllocatableSlots != right.telemetry.AllocatableSlots {
			return left.telemetry.AllocatableSlots > right.telemetry.AllocatableSlots
		}
		return left.instance.ID < right.instance.ID
	}
}

func freezeBuilderTelemetryEvidence(snapshot moduleapi.BuilderTelemetrySnapshot) frozenBuilderTelemetryEvidence {
	return frozenBuilderTelemetryEvidence{TargetID: snapshot.TargetID, BuilderScope: snapshot.BuilderScope, ProviderID: snapshot.ProviderID, CapabilityProfile: snapshot.CapabilityProfile, CapabilityVersion: snapshot.CapabilityVersion, Available: snapshot.Available, Running: snapshot.Running, Queued: snapshot.Queued, AllocatableSlots: snapshot.AllocatableSlots, ObservedAt: snapshot.ObservedAt.UTC(), ExpiresAt: snapshot.ExpiresAt.UTC(), SourceRef: snapshot.SourceRef, Provenance: snapshot.Provenance, Integrity: snapshot.Integrity, AffinityKey: snapshot.AffinityKey, UnsupportedDimensions: append([]string(nil), snapshot.UnsupportedDimensions...)}
}

func dynamicCandidateFingerprint(pool moduleapi.BuilderPool, selector builderLabelSelector, candidates []dynamicBuilderCandidate) string {
	type fingerprintCandidate struct {
		InstanceID string                         `json:"instance_id"`
		Telemetry  frozenBuilderTelemetryEvidence `json:"telemetry"`
	}
	fingerprintCandidates := make([]fingerprintCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		fingerprintCandidates = append(fingerprintCandidates, fingerprintCandidate{InstanceID: candidate.instance.ID, Telemetry: freezeBuilderTelemetryEvidence(candidate.telemetry)})
	}
	payload, _ := json.Marshal(struct {
		PoolID, Policy string
		Selector       builderLabelSelector
		Candidates     []fingerprintCandidate
	}{pool.ID, pool.SchedulingPolicy, selector, fingerprintCandidates})
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type staticPoolSelectionEvidence struct {
	Seed   string
	Cursor *int64
}

func (s *Service) selectStaticPoolCandidate(ctx context.Context, pool moduleapi.BuilderPool, selector builderLabelSelector, candidates []moduleapi.BuilderInstance, input staticPoolSelectionInput, fingerprint string) (moduleapi.BuilderInstance, staticPoolSelectionEvidence, error) {
	switch pool.SchedulingPolicy {
	case "manual":
		selected, err := selectManualPoolCandidate(selector.InstanceID, candidates)
		return selected, staticPoolSelectionEvidence{}, err
	case "random":
		sum := sha256.Sum256([]byte(input.SeedMaterial + ":" + fingerprint))
		return candidates[int(sum[0])%len(candidates)], staticPoolSelectionEvidence{Seed: hex.EncodeToString(sum[:])}, nil
	case "round_robin":
		return s.selectRoundRobinPoolCandidate(ctx, pool.ID, candidates, input.MemberCount)
	}
	return moduleapi.BuilderInstance{}, staticPoolSelectionEvidence{}, fmt.Errorf("builder pool scheduling policy %q is not supported", pool.SchedulingPolicy)
}

func selectManualPoolCandidate(instanceID string, candidates []moduleapi.BuilderInstance) (moduleapi.BuilderInstance, error) {
	if instanceID == "" {
		return moduleapi.BuilderInstance{}, errors.New("manual builder pool requires instance_id selector")
	}
	for _, candidate := range candidates {
		if candidate.ID == instanceID {
			return candidate, nil
		}
	}
	return moduleapi.BuilderInstance{}, errors.New("manual builder pool instance is not eligible")
}

func (s *Service) selectRoundRobinPoolCandidate(ctx context.Context, poolID string, candidates []moduleapi.BuilderInstance, memberCount int) (moduleapi.BuilderInstance, staticPoolSelectionEvidence, error) {
	for range memberCount {
		choice, err := s.builderResources.SelectRoundRobinBuilderInstance(ctx, poolID)
		if err != nil {
			return moduleapi.BuilderInstance{}, staticPoolSelectionEvidence{}, fmt.Errorf("select builder pool instance: %w", err)
		}
		for _, candidate := range candidates {
			if candidate.ID == choice.Instance.ID {
				return candidate, staticPoolSelectionEvidence{Cursor: choice.Cursor}, nil
			}
		}
	}
	return moduleapi.BuilderInstance{}, staticPoolSelectionEvidence{}, errors.New("builder pool has no compatible assigned instance")
}

func staticCandidateFingerprint(pool moduleapi.BuilderPool, selector builderLabelSelector, candidates []moduleapi.BuilderInstance) string {
	input := struct {
		PoolID, Policy string
		Selector       builderLabelSelector
		Candidates     []moduleapi.BuilderInstance
	}{pool.ID, pool.SchedulingPolicy, selector, candidates}
	payload, _ := json.Marshal(input)
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
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
func (s *Service) builderInstanceSupportsPlan(ctx context.Context, instance moduleapi.BuilderInstance, driverRef string, platforms []string, actorID uint64) (placementCapabilityAuthorization, bool) {
	selectedDriverRef := instance.DriverRef
	if !strings.Contains(selectedDriverRef, "@") && instance.DriverVersion != "" {
		selectedDriverRef += "@" + instance.DriverVersion
	}
	if !containsBuildRef([]string{selectedDriverRef}, driverRef) {
		return placementCapabilityAuthorization{}, false
	}
	authorization, err := s.authorizeBuildPlacementForPlatforms(ctx, actorID, instance.RuntimeTargetID, driverRef, platforms)
	if err != nil {
		return placementCapabilityAuthorization{}, false
	}
	return authorization, true
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
			evidence, marshalErr := json.Marshal(staticPlacementEvidence{PolicyID: "build.direct.manual", PolicyVersion: "v1", SelectedInstanceID: selectedBuilderID, ReservationSlotBudget: 1})
			if marshalErr != nil {
				return moduleapi.TaskReceipt{}, fmt.Errorf("marshal direct placement evidence: %w", marshalErr)
			}
			request.BuilderPlacements = append(request.BuilderPlacements, moduleapi.BuilderPlacement{Platform: platform, BuilderInstanceID: selectedBuilderID, RuntimeTargetID: request.RuntimeTargetID, SchedulingPolicy: "manual", SchedulingEvidence: evidence})
		}
	}
	if request.BuilderPoolID == "" {
		for index, placement := range request.BuilderPlacements {
			authorization, authorizeErr := s.authorizeBuildPlacementForPlatforms(ctx, actorID, placement.RuntimeTargetID, request.Driver, []string{placement.Platform})
			if authorizeErr != nil {
				return moduleapi.TaskReceipt{}, authorizeErr
			}
			evidence, evidenceErr := freezePlacementNegotiation(placement.SchedulingEvidence, authorization)
			if evidenceErr != nil {
				return moduleapi.TaskReceipt{}, evidenceErr
			}
			request.BuilderPlacements[index].SchedulingEvidence = evidence
		}
	}
	// Project 仅负责来源授权和初次捕获；提交后由 Build 保留执行物化内容，
	// 后续保留策略不会改变 Snapshot identity。
	materialization, materializeErr := (buildWorkspaceMaterializer{}).MaterializeSnapshot(ctx, snapshot, moduleapi.WorkspaceMaterializationRequest{ExecutionID: request.IdempotencyKey})
	_ = os.RemoveAll(snapshot.MaterializedRoot)
	if materializeErr != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("materialize workspace snapshot: %w", materializeErr)
	}
	snapshot.MaterializedRoot = ""
	snapshot.MaterializationRef = materialization.MaterializationRef
	plan, err := freezeExecutionPlan(snapshot, request, selectedBuilderID)
	if err != nil {
		_ = releaseMaterialization(snapshot.MaterializationRef)
		return moduleapi.TaskReceipt{}, err
	}
	input, err := json.Marshal(moduleapi.BuildPlanTaskInput{BuildID: plan.ID, ExecutionPlanID: plan.ID})
	if err != nil {
		_ = releaseMaterialization(snapshot.MaterializationRef)
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
		_ = releaseMaterialization(snapshot.MaterializationRef)
		return moduleapi.TaskReceipt{}, err
	}
	if handle.Submission.State == moduleapi.TaskSubmissionStateActivated && handle.Submission.TaskID != nil {
		_ = releaseMaterialization(snapshot.MaterializationRef)
		return s.activatedSubmissionReceipt(ctx, *handle.Submission.TaskID)
	}
	repository, ok := s.repository.(buildstore.ExecutionPlanRepository)
	if !ok {
		_ = releaseMaterialization(snapshot.MaterializationRef)
		return moduleapi.TaskReceipt{}, errors.New("build execution plan persistence is unavailable")
	}
	receipt, err := s.submissions.MaterializeSubmission(ctx, handle, task, executionPlanSubmissionWriter{repository: repository, plan: plan, requestedBy: request.RequestedBy})
	if err != nil {
		_ = releaseMaterialization(snapshot.MaterializationRef)
	}
	return receipt, err
}

func releaseMaterialization(reference string) error {
	return (buildWorkspaceMaterializer{}).ReleaseMaterialization(context.Background(), moduleapi.WorkspaceMaterialization{MaterializationRef: reference})
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
type placementCapabilityAuthorization struct {
	requirement            moduleapi.BuildCapabilityRequirement
	requirementFingerprint string
	capabilityProfile      string
	capabilityVersion      string
	negotiation            moduleapi.NegotiatedCapability
}

func (s *Service) authorizeBuildPlacementForPlatforms(ctx context.Context, actorID uint64, runtimeTargetID int64, driverRef string, platforms []string) (placementCapabilityAuthorization, error) {
	allowed, err := s.buildAssignments.CanUseBuildTarget(ctx, actorID, runtimeTargetID)
	if err != nil {
		return placementCapabilityAuthorization{}, fmt.Errorf("authorize build target: %w", err)
	}
	if !allowed {
		return placementCapabilityAuthorization{}, errors.New("build target is not assigned to actor")
	}
	target, err := s.buildTargets.ReadBuildTarget(ctx, runtimeTargetID)
	if err != nil {
		return placementCapabilityAuthorization{}, fmt.Errorf("read build target: %w", err)
	}
	if !target.Available || !containsBuildRef(target.SupportedDrivers, driverRef) || !slices.Contains(target.WorkspaceLocalities, "build-snapshot") || !hasSnapshotDeliveryMode(target.SnapshotDeliveryModes) || !containsAll(target.SupportedPlatforms, platforms) {
		return placementCapabilityAuthorization{}, errors.New("build target is incompatible with execution plan")
	}
	requirement := buildCapabilityRequirement(driverRef, platforms...)
	capability := moduleapi.BuildExecutionCapability{ProviderCapabilityProfile: target.ProviderCapabilityProfile, ProviderCapabilityVersion: target.ProviderCapabilityVersion, SupportedDrivers: append([]string(nil), target.SupportedDrivers...), SupportedPlatforms: append([]string(nil), target.SupportedPlatforms...), SnapshotDeliveryModes: append([]string(nil), target.SnapshotDeliveryModes...), Features: append([]string(nil), target.BuildFeatures...)}
	negotiation, err := (staticCapabilityMatcher{}).MatchBuildCapability(requirement, capability)
	if err != nil {
		return placementCapabilityAuthorization{}, fmt.Errorf("match build capability: %w", err)
	}
	return placementCapabilityAuthorization{requirement: requirement, requirementFingerprint: fingerprintBuildCapabilityRequirement(requirement), capabilityProfile: capability.ProviderCapabilityProfile, capabilityVersion: capability.ProviderCapabilityVersion, negotiation: negotiation}, nil
}

//nolint:cyclop // Frozen evidence must reject every incomplete capability fact at one persistence boundary.
func freezePlacementNegotiation(raw json.RawMessage, authorization placementCapabilityAuthorization) (json.RawMessage, error) {
	if len(raw) == 0 || authorization.requirementFingerprint == "" || authorization.capabilityProfile == "" || authorization.capabilityVersion == "" || authorization.negotiation.ProviderCapabilityProfile != authorization.capabilityProfile || authorization.negotiation.ProviderCapabilityVersion != authorization.capabilityVersion {
		return nil, errors.New("placement evidence is missing")
	}
	var evidence map[string]json.RawMessage
	if err := json.Unmarshal(raw, &evidence); err != nil {
		return nil, errors.New("placement evidence is invalid")
	}
	encodedNegotiation, err := json.Marshal(authorization.negotiation)
	if err != nil {
		return nil, fmt.Errorf("marshal capability negotiation: %w", err)
	}
	encodedFingerprint, err := json.Marshal(authorization.requirementFingerprint)
	if err != nil {
		return nil, fmt.Errorf("marshal capability requirement fingerprint: %w", err)
	}
	encodedProfile, err := json.Marshal(authorization.capabilityProfile)
	if err != nil {
		return nil, fmt.Errorf("marshal capability profile: %w", err)
	}
	encodedVersion, err := json.Marshal(authorization.capabilityVersion)
	if err != nil {
		return nil, fmt.Errorf("marshal capability version: %w", err)
	}
	evidence["capability_negotiation"] = encodedNegotiation
	evidence["capability_requirement_fingerprint"] = encodedFingerprint
	evidence["capability_profile"] = encodedProfile
	evidence["capability_version"] = encodedVersion
	frozen, err := json.Marshal(evidence)
	if err != nil {
		return nil, fmt.Errorf("freeze placement negotiation: %w", err)
	}
	return frozen, nil
}

func fingerprintBuildCapabilityRequirement(requirement moduleapi.BuildCapabilityRequirement) string {
	payload, _ := json.Marshal(struct {
		DriverRef             string                                        `json:"driver_ref"`
		TemplateRef           string                                        `json:"template_ref"`
		DestinationKind       string                                        `json:"destination_kind"`
		CachePolicy           string                                        `json:"cache_policy"`
		SecurityPolicy        string                                        `json:"security_policy"`
		Platforms             []string                                      `json:"platforms"`
		SnapshotDeliveryModes []string                                      `json:"snapshot_delivery_modes"`
		RequiredFeatures      []string                                      `json:"required_features"`
		FeatureRequirements   []moduleapi.BuildCapabilityFeatureRequirement `json:"feature_requirements"`
	}{DriverRef: strings.TrimSpace(requirement.DriverRef), TemplateRef: strings.TrimSpace(requirement.TemplateRef), DestinationKind: strings.TrimSpace(requirement.DestinationKind), CachePolicy: strings.TrimSpace(requirement.CachePolicy), SecurityPolicy: strings.TrimSpace(requirement.SecurityPolicy), Platforms: canonicalCapabilityValues(requirement.Platforms), SnapshotDeliveryModes: canonicalCapabilityValues(requirement.SnapshotDeliveryModes), RequiredFeatures: canonicalCapabilityValues(requirement.RequiredFeatures), FeatureRequirements: canonicalFeatureRequirements(requirement.FeatureRequirements)})
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func canonicalFeatureRequirements(values []moduleapi.BuildCapabilityFeatureRequirement) []moduleapi.BuildCapabilityFeatureRequirement {
	canonical := append([]moduleapi.BuildCapabilityFeatureRequirement(nil), values...)
	for index := range canonical {
		canonical[index].Feature = strings.TrimSpace(canonical[index].Feature)
		canonical[index].Mode = strings.TrimSpace(canonical[index].Mode)
	}
	sort.Slice(canonical, func(i, j int) bool {
		return canonical[i].Feature+":"+canonical[i].Mode < canonical[j].Feature+":"+canonical[j].Mode
	})
	return canonical
}

func buildCapabilityRequirement(driverRef string, platforms ...string) moduleapi.BuildCapabilityRequirement {
	return buildCapabilityRequirementForResolvedPolicy(driverRef, "disabled", "default", platforms...)
}

func buildCapabilityRequirementForResolvedPolicy(driverRef, cachePolicy, securityPolicy string, platforms ...string) moduleapi.BuildCapabilityRequirement {
	return moduleapi.BuildCapabilityRequirement{DriverRef: strings.TrimSpace(driverRef), TemplateRef: v2DockerfileTemplate, DestinationKind: v2OCIDestination, CachePolicy: strings.TrimSpace(cachePolicy), SecurityPolicy: strings.TrimSpace(securityPolicy), Platforms: canonicalCapabilityValues(platforms), SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal, moduleapi.SnapshotDeliveryModeProviderTransfer}, FeatureRequirements: []moduleapi.BuildCapabilityFeatureRequirement{{Feature: "registry-login", Mode: moduleapi.BuildCapabilityFeatureRequired}, {Feature: "provenance", Mode: moduleapi.BuildCapabilityFeaturePreferred}, {Feature: "sbom", Mode: moduleapi.BuildCapabilityFeatureOptional}}}
}

func canonicalCapabilityValues(values []string) []string {
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			unique[value] = struct{}{}
		}
	}
	canonical := make([]string, 0, len(unique))
	for value := range unique {
		canonical = append(canonical, value)
	}
	sort.Strings(canonical)
	return canonical
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
	request.CachePolicy = strings.TrimSpace(request.CachePolicy)
	request.SecurityPolicy = strings.TrimSpace(request.SecurityPolicy)
	if request.CachePolicy == "" {
		request.CachePolicy = "disabled"
	}
	if request.SecurityPolicy == "" {
		request.SecurityPolicy = "default"
	}
	if request.CachePolicy != "disabled" || request.SecurityPolicy != "default" {
		return ExecutionPlanRequest{}, errors.New("execution plan policy is unsupported")
	}
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
		CachePolicy       string                       `json:"cache_policy"`
		SecurityPolicy    string                       `json:"security_policy"`
		Destination       moduleapi.BuildDestination   `json:"destination"`
	}{SnapshotDigest: snapshot.ContentDigest, BuilderPoolID: request.BuilderPoolID, BuilderInstanceID: builderInstanceID, RuntimeTarget: request.RuntimeTargetID, BuilderPlacements: append([]moduleapi.BuilderPlacement(nil), request.BuilderPlacements...), Template: request.TemplateRef, Driver: request.Driver, CachePolicy: request.CachePolicy, SecurityPolicy: request.SecurityPolicy, Platforms: append([]string(nil), request.Platforms...), Destination: request.Destination}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return moduleapi.BuildExecutionPlan{}, err
	}
	digest := sha256.Sum256(payload)
	digestText := hex.EncodeToString(digest[:])
	return moduleapi.BuildExecutionPlan{ID: "plan_" + digestText[:26], Digest: "sha256:" + digestText, Workspace: snapshot, BuilderPoolID: request.BuilderPoolID, BuilderInstanceID: builderInstanceID, RuntimeTargetID: request.RuntimeTargetID, BuilderPlacements: append([]moduleapi.BuilderPlacement(nil), request.BuilderPlacements...), Driver: request.Driver, TemplateRef: request.TemplateRef, CachePolicy: request.CachePolicy, SecurityPolicy: request.SecurityPolicy, Platforms: append([]string(nil), request.Platforms...), Destination: request.Destination, CreatedAt: time.Now().UTC()}, nil
}

func containsAll(supported, requested []string) bool {
	for _, item := range requested {
		if !slices.Contains(supported, item) {
			return false
		}
	}
	return true
}
