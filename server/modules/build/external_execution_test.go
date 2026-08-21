package build

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

const testExternalDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type externalExecutionRepository struct {
	job                  buildstore.JobSnapshot
	plan                 moduleapi.BuildExecutionPlan
	legacyResult         moduleapi.BuildArtifactResult
	v2Result             moduleapi.BuildArtifactResult
	platformArtifact     moduleapi.PlatformArtifact
	manifestInput        moduleapi.OCIManifestPublicationInput
	manifestResult       moduleapi.OCIManifestPublicationResult
	promotionInput       moduleapi.OCIArtifactCopyInput
	promotionResult      moduleapi.OCIArtifactCopyResult
	legacySettlements    int
	v2Settlements        int
	platformSettlements  int
	manifestSettlements  int
	promotionSettlements int
	reservationRunning   bool
}

func (*externalExecutionRepository) CreateJob(context.Context, buildstore.JobSnapshot) error {
	return nil
}
func (*externalExecutionRepository) MaterializeSubmissionSnapshot(context.Context, *sql.Tx, moduleapi.TaskSubmission, buildstore.JobSnapshot) (string, error) {
	return "", nil
}
func (r *externalExecutionRepository) GetJobByTaskID(context.Context, uint64) (buildstore.JobSnapshot, error) {
	return r.job, nil
}
func (r *externalExecutionRepository) SettleBuildArtifact(_ context.Context, _ uint64, result moduleapi.BuildArtifactResult) error {
	r.legacyResult = result
	r.legacySettlements++
	return nil
}
func (*externalExecutionRepository) ListJobs(context.Context, buildstore.ListQuery) (buildstore.ListResult, error) {
	return buildstore.ListResult{}, nil
}
func (*externalExecutionRepository) GetJobByBuildID(context.Context, string) (buildstore.JobProjection, error) {
	return buildstore.JobProjection{}, buildstore.ErrNotFound
}
func (r *externalExecutionRepository) GetExecutionPlanByTaskID(context.Context, uint64) (moduleapi.BuildExecutionPlan, error) {
	return r.plan, nil
}
func (r *externalExecutionRepository) SettleV2Artifact(_ context.Context, _ uint64, _ moduleapi.BuildExecutionPlan, result moduleapi.BuildArtifactResult, auth moduleapi.RegistryAuthExecution) error {
	if auth.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		return errors.New("unexpected auth mode")
	}
	r.v2Result = result
	r.v2Settlements++
	return nil
}
func (r *externalExecutionRepository) RecordPlatformArtifact(_ context.Context, _ uint64, _ moduleapi.BuildExecutionPlan, artifact moduleapi.PlatformArtifact) error {
	r.platformArtifact = artifact
	r.platformSettlements++
	return nil
}
func (*externalExecutionRepository) ListPlatformArtifacts(context.Context, uint64, moduleapi.BuildExecutionPlan) ([]moduleapi.PlatformArtifact, error) {
	return nil, nil
}
func (r *externalExecutionRepository) PrepareOCIManifestPublication(context.Context, uint64, moduleapi.BuildExecutionPlan) (moduleapi.OCIManifestPublicationInput, error) {
	return r.manifestInput, nil
}
func (r *externalExecutionRepository) SettleOCIManifestPublication(_ context.Context, _ uint64, _ moduleapi.BuildExecutionPlan, result moduleapi.OCIManifestPublicationResult, auth moduleapi.RegistryAuthExecution) error {
	if auth.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		return errors.New("unexpected auth mode")
	}
	r.manifestResult = result
	r.manifestSettlements++
	return nil
}
func (r *externalExecutionRepository) SettleArtifactPromotion(_ context.Context, input moduleapi.OCIArtifactCopyInput, result moduleapi.OCIArtifactCopyResult, auth moduleapi.RegistryAuthExecution) error {
	if auth.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		return errors.New("unexpected auth mode")
	}
	r.promotionInput = input
	r.promotionResult = result
	r.promotionSettlements++
	return nil
}
func (*externalExecutionRepository) ReserveBuilder(context.Context, *sql.Tx, moduleapi.BuilderReservation) (moduleapi.BuilderReservation, error) {
	return moduleapi.BuilderReservation{}, nil
}
func (*externalExecutionRepository) ReserveBuilderAttempt(context.Context, moduleapi.BuilderReservation) (moduleapi.BuilderReservation, error) {
	return moduleapi.BuilderReservation{}, nil
}
func (r *externalExecutionRepository) MarkBuilderReservationRunning(context.Context, uint64, string, string) error {
	if r.reservationRunning {
		return buildstore.ErrConflict
	}
	r.reservationRunning = true
	return nil
}
func (r *externalExecutionRepository) RenewBuilderReservation(context.Context, uint64, string, string, time.Time) error {
	if !r.reservationRunning {
		return buildstore.ErrConflict
	}
	return nil
}
func (r *externalExecutionRepository) ReleaseBuilderReservation(context.Context, uint64, string, string, string) error {
	if !r.reservationRunning {
		return buildstore.ErrConflict
	}
	r.reservationRunning = false
	return nil
}

type externalExecutionContexts struct {
	context moduleapi.ApplicationBuildContext
	calls   int
}

func (r *externalExecutionContexts) ResolveApplicationBuildContext(context.Context, string) (moduleapi.ApplicationBuildContext, error) {
	r.calls++
	return r.context, nil
}

type externalExecutionCredentials struct {
	prepared []moduleapi.CredentialRequest
	revoked  int
	material moduleapi.EphemeralCredentialMaterial
}

func (*externalExecutionCredentials) Assess(context.Context, moduleapi.CredentialEligibilityRequest) (moduleapi.CredentialEligibility, error) {
	return moduleapi.CredentialEligibility{Status: moduleapi.CredentialEligibilityEligible}, nil
}
func (c *externalExecutionCredentials) Prepare(_ context.Context, request moduleapi.CredentialRequest) (moduleapi.EphemeralCredentialSession, error) {
	c.prepared = append(c.prepared, request)
	return moduleapi.EphemeralCredentialSession{ID: "session", ExpiresAt: request.ExpiresAt}, nil
}
func (*externalExecutionCredentials) Inject(context.Context, moduleapi.EphemeralCredentialSession, moduleapi.CredentialInjectionTarget) error {
	return nil
}
func (c *externalExecutionCredentials) Revoke(context.Context, moduleapi.EphemeralCredentialSession) error {
	c.revoked++
	return nil
}
func (c *externalExecutionCredentials) ResolveCredentialMaterial(context.Context, moduleapi.EphemeralCredentialSession, moduleapi.CredentialInjectionTarget) (moduleapi.EphemeralCredentialMaterial, error) {
	return c.material, nil
}

type externalExecutionRegistry struct {
	binding moduleapi.RegistryPublicationBinding
}

func (r externalExecutionRegistry) ResolvePublicationBinding(context.Context, moduleapi.AuthorizedArtifactDestination) (moduleapi.RegistryPublicationBinding, error) {
	return r.binding, nil
}

func TestBuildExternalExecutionLegacyUsesLiveApplicationContext(t *testing.T) {
	repository := &externalExecutionRepository{job: buildstore.JobSnapshot{BuildID: "build-1", ApplicationID: "app-1", WorkspaceRoot: "/stale", ContextPath: "src", DockerfilePath: "Dockerfile", RuntimeTargetID: 9, ImageRepository: "team/api", ImageTag: "latest"}}
	contexts := &externalExecutionContexts{context: moduleapi.ApplicationBuildContext{ApplicationID: "app-1", WorkspaceRoot: "/live", RuntimeTargetID: 9, CanBuild: true}}
	service := &Service{contexts: contexts, repository: repository}
	handler := &buildExternalExecutionHandler{executorType: buildStageExecutor, dependencies: buildExternalExecutionDependencies{repository: repository, service: service}}
	input := mustExternalJSON(t, moduleapi.BuildTaskInput{BuildID: "build-1", ApplicationID: "app-1"})

	material, err := handler.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{TaskID: 41, ExecutorType: buildStageExecutor, RuntimeTargetID: 9, OperationID: buildImageLocalOperation, Input: input})
	if err != nil {
		t.Fatalf("resolve legacy material: %v", err)
	}
	var decoded buildExecutionMaterial
	if err := json.Unmarshal(material.Payload, &decoded); err != nil {
		t.Fatalf("decode legacy material: %v", err)
	}
	if material.Protocol != buildExecutionMaterialProtocol || decoded.Context == nil || decoded.Context.Root != "/live" || decoded.Context.Root == repository.job.WorkspaceRoot || contexts.calls != 1 {
		t.Fatalf("unexpected legacy material: protocol=%q material=%#v calls=%d", material.Protocol, decoded, contexts.calls)
	}
	if _, err := handler.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{TaskID: 41, ExecutorType: buildStageExecutor, RuntimeTargetID: 9, OperationID: buildImagePublishOperation, Input: input}); err == nil {
		t.Fatal("expected operation mismatch to fail")
	}
}

func TestBuildExternalExecutionPublicationMaterialRevokesCredential(t *testing.T) {
	repository := &externalExecutionRepository{plan: singlePlatformExecutionPlan()}
	credentials := &externalExecutionCredentials{material: moduleapi.EphemeralCredentialMaterial{Username: "builder", Secret: "secret"}}
	destination := moduleapi.AuthorizedArtifactDestination(repository.plan.Destination)
	registry := externalExecutionRegistry{binding: moduleapi.RegistryPublicationBinding{Destination: destination, Endpoint: "https://registry.example", CredentialRef: "registry:build", AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}}}
	service := &Service{repository: repository}
	handler := &buildExternalExecutionHandler{executorType: v2BuildStageExecutor, dependencies: buildExternalExecutionDependencies{repository: repository, service: service, registry: registry, credentials: credentials, credentialMaterials: credentials}}
	input := mustExternalJSON(t, moduleapi.BuildPlanTaskInput{BuildID: "plan-1", ExecutionPlanID: "plan-1"})

	material, err := handler.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{TaskID: 42, Attempt: 1, ExecutorType: v2BuildStageExecutor, RuntimeTargetID: 9, OperationID: buildImagePublishOperation, Input: input})
	if err != nil {
		t.Fatalf("resolve v2 material: %v", err)
	}
	var decoded buildExecutionMaterial
	if err := json.Unmarshal(material.Payload, &decoded); err != nil {
		t.Fatalf("decode v2 material: %v", err)
	}
	if decoded.Destination == nil || decoded.Destination.Username != "builder" || decoded.Destination.Password != "secret" || credentials.revoked != 1 || len(credentials.prepared) != 1 || credentials.prepared[0].Operation != "push" {
		t.Fatalf("unexpected credential lifecycle: material=%#v credentials=%#v", decoded, credentials)
	}
	if _, err := handler.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{TaskID: 42, Attempt: 1, ExecutorType: v2BuildStageExecutor, RuntimeTargetID: 9, OperationID: buildImagePublishOperation, Input: input}); err != nil {
		t.Fatalf("replay v2 material resolution: %v", err)
	}
	if credentials.revoked != 2 {
		t.Fatalf("expected replay credential to be revoked, got %d revocations", credentials.revoked)
	}
}

func TestBuildExternalExecutionResultMappingsAndReplay(t *testing.T) {
	repository := &externalExecutionRepository{job: buildstore.JobSnapshot{BuildID: "build-1", RuntimeTargetID: 9, ImageRepository: "team/api", ImageTag: "latest"}, plan: singlePlatformExecutionPlan()}
	service := &Service{repository: repository}
	legacy := &buildExternalExecutionHandler{executorType: buildStageExecutor, dependencies: buildExternalExecutionDependencies{repository: repository, service: service}}
	legacyRequest := moduleapi.ExternalExecutionResultRequest{TaskID: 41, ExecutorType: buildStageExecutor, RuntimeTargetID: 9, OperationID: buildImageLocalOperation, Input: mustExternalJSON(t, moduleapi.BuildTaskInput{BuildID: "build-1"}), Protocol: buildExecutionResultProtocol, Result: mustExternalJSON(t, buildExecutionResult{ImageID: "image-1", Digest: "team/api@" + testExternalDigest, Repository: "team/api", Reference: "latest"})}
	if err := legacy.RecordExternalExecutionResult(context.Background(), legacyRequest); err != nil {
		t.Fatalf("record legacy result: %v", err)
	}
	if err := legacy.RecordExternalExecutionResult(context.Background(), legacyRequest); err != nil {
		t.Fatalf("replay legacy result: %v", err)
	}
	if repository.legacySettlements != 2 || repository.legacyResult.Digest != testExternalDigest {
		t.Fatalf("unexpected legacy settlement: %#v count=%d", repository.legacyResult, repository.legacySettlements)
	}

	v2 := &buildExternalExecutionHandler{executorType: v2BuildStageExecutor, dependencies: buildExternalExecutionDependencies{repository: repository, service: service}}
	repository.reservationRunning = true
	v2Request := moduleapi.ExternalExecutionResultRequest{TaskID: 42, Attempt: 1, ExecutorType: v2BuildStageExecutor, RuntimeTargetID: 9, OperationID: buildImagePublishOperation, Input: mustExternalJSON(t, moduleapi.BuildPlanTaskInput{BuildID: "plan-1", ExecutionPlanID: "plan-1"}), Protocol: buildExecutionResultProtocol, Result: mustExternalJSON(t, buildExecutionResult{ImageID: "image-2", Digest: testExternalDigest, Repository: "team/api", Reference: "latest", OS: "linux", Architecture: "amd64"})}
	if err := v2.RecordExternalExecutionResult(context.Background(), v2Request); err != nil {
		t.Fatalf("record v2 result: %v", err)
	}
	if err := v2.RecordExternalExecutionResult(context.Background(), v2Request); err != nil {
		t.Fatalf("replay v2 result: %v", err)
	}
	if repository.v2Settlements != 2 || repository.v2Result.Tag != "latest" {
		t.Fatalf("unexpected v2 settlement: %#v", repository.v2Result)
	}
}

func TestBuildExternalExecutionMapsLegManifestAndPromotionResults(t *testing.T) {
	plan := multiPlatformExecutionPlan()
	repository := &externalExecutionRepository{plan: plan}
	service := &Service{repository: repository}
	v2 := &buildExternalExecutionHandler{executorType: v2BuildStageExecutor, dependencies: buildExternalExecutionDependencies{repository: repository, service: service}}
	repository.reservationRunning = true
	leg := moduleapi.ExternalExecutionResultRequest{TaskID: 43, Attempt: 1, ExecutorType: v2BuildStageExecutor, RuntimeTargetID: 10, OperationID: buildImagePublishOperation, Input: mustExternalJSON(t, moduleapi.BuildPlanTaskInput{BuildID: "plan-2", ExecutionPlanID: "plan-2", Platform: "linux/arm64", LegID: "platform-2"}), Protocol: buildExecutionResultProtocol, Result: mustExternalJSON(t, buildExecutionResult{Digest: testExternalDigest, Repository: "team/api", Reference: temporaryPlatformTag("latest", "platform-2"), MediaType: ociImageManifestMediaType, OS: "linux", Architecture: "arm64"})}
	if err := v2.RecordExternalExecutionResult(context.Background(), leg); err != nil {
		t.Fatalf("record platform result: %v", err)
	}
	manifest := moduleapi.ExternalExecutionResultRequest{TaskID: 43, ExecutorType: v2BuildStageExecutor, RuntimeTargetID: 9, OperationID: buildManifestOperation, Input: mustExternalJSON(t, moduleapi.BuildPlanTaskInput{BuildID: "plan-2", ExecutionPlanID: "plan-2"}), Protocol: buildExecutionResultProtocol, Result: mustExternalJSON(t, buildExecutionResult{Digest: testExternalDigest, Repository: "team/api", Reference: "latest", MediaType: "application/vnd.oci.image.index.v1+json"})}
	if err := v2.RecordExternalExecutionResult(context.Background(), manifest); err != nil {
		t.Fatalf("record manifest result: %v", err)
	}
	if repository.platformArtifact.LegID != "platform-2" || repository.manifestResult.Digest != testExternalDigest {
		t.Fatalf("unexpected coordinated settlements: artifact=%#v manifest=%#v", repository.platformArtifact, repository.manifestResult)
	}

	source := moduleapi.ArtifactPublicationSource{ArtifactID: "artifact-1", PublicationID: "publication-1", Digest: testExternalDigest, MediaType: ociImageManifestMediaType, DestinationKind: "oci_registry", RepositoryRef: "team/api"}
	destination := moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry-2", RepositoryRef: "release/api", Reference: "stable"}
	promotion := &buildExternalExecutionHandler{executorType: artifactPromotionStageExecutor, dependencies: buildExternalExecutionDependencies{repository: repository, service: service}}
	promotionRequest := moduleapi.ExternalExecutionResultRequest{TaskID: 44, ExecutorType: artifactPromotionStageExecutor, RuntimeTargetID: 11, OperationID: buildArtifactCopyOperation, Input: mustExternalJSON(t, moduleapi.ArtifactPromotionTaskInput{Source: source, Destination: destination, RuntimeTargetID: 11}), Protocol: buildExecutionResultProtocol, Result: mustExternalJSON(t, buildExecutionResult{Digest: testExternalDigest, Repository: "release/api", Reference: "stable", MediaType: ociImageManifestMediaType})}
	if err := promotion.RecordExternalExecutionResult(context.Background(), promotionRequest); err != nil {
		t.Fatalf("record promotion result: %v", err)
	}
	if repository.promotionSettlements != 1 || repository.promotionInput.Destination != destination || repository.promotionResult.Digest != testExternalDigest {
		t.Fatalf("unexpected promotion settlement: input=%#v result=%#v", repository.promotionInput, repository.promotionResult)
	}
}

func singlePlatformExecutionPlan() moduleapi.BuildExecutionPlan {
	reference, _ := moduleapi.NewWorkspaceSnapshotMaterializationReference("snapshot-1", testExternalDigest, "/tmp/snapshot-test")
	evidence, _ := json.Marshal(map[string]any{"reservation_slot_budget": 1})
	return moduleapi.BuildExecutionPlan{ID: "plan-1", Workspace: moduleapi.WorkspaceSnapshot{MaterializationRef: reference}, RuntimeTargetID: 9, BuilderPlacements: []moduleapi.BuilderPlacement{{Platform: "linux/amd64", BuilderInstanceID: "builder-1", RuntimeTargetID: 9, SchedulingPolicy: "manual", SchedulingEvidence: evidence}}, Platforms: []string{"linux/amd64"}, Destination: moduleapi.BuildDestination{Kind: "oci_registry", ConnectionRef: "registry-1", RepositoryRef: "team/api", Reference: "latest"}}
}

func multiPlatformExecutionPlan() moduleapi.BuildExecutionPlan {
	evidence, _ := json.Marshal(map[string]any{"reservation_slot_budget": 1})
	return moduleapi.BuildExecutionPlan{ID: "plan-2", RuntimeTargetID: 9, BuilderPlacements: []moduleapi.BuilderPlacement{{Platform: "linux/amd64", BuilderInstanceID: "builder-1", RuntimeTargetID: 9, SchedulingPolicy: "manual", SchedulingEvidence: evidence}, {Platform: "linux/arm64", BuilderInstanceID: "builder-2", RuntimeTargetID: 10, SchedulingPolicy: "manual", SchedulingEvidence: evidence}}, Platforms: []string{"linux/amd64", "linux/arm64"}, Destination: moduleapi.BuildDestination{Kind: "oci_registry", ConnectionRef: "registry-1", RepositoryRef: "team/api", Reference: "latest"}}
}

func mustExternalJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test value: %v", err)
	}
	return raw
}
