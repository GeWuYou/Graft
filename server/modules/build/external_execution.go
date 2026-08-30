package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

const (
	buildExecutionMaterialProtocol = "build-execution-material/v1"
	buildExecutionResultProtocol   = "build-execution-result/v1"
	buildCredentialSessionTTL      = 5 * time.Minute
	ociImageManifestMediaType      = "application/vnd.oci.image.manifest.v1+json"
)

type buildExternalExecutionDependencies struct {
	repository           buildstore.Repository
	service              *Service
	registry             moduleapi.RegistryPublicationResolver
	artifactCopyRegistry moduleapi.RegistryArtifactCopyResolver
	credentials          moduleapi.CredentialProvider
	credentialMaterials  moduleapi.EphemeralCredentialMaterialProvider
}

type buildExternalExecutionHandler struct {
	executorType moduleapi.StageExecutorType
	dependencies buildExternalExecutionDependencies
}

type buildRetryReservationRepository interface {
	ReserveBuilderAttemptWithCapacity(context.Context, moduleapi.BuilderReservation, int) (moduleapi.BuilderReservation, error)
}

type buildObservedRetryReservationRepository interface {
	ReserveBuilderAttemptWithCapacityAfterObservation(context.Context, moduleapi.BuilderReservation, int, time.Time) (moduleapi.BuilderReservation, error)
}

type buildExecutionMaterial struct {
	Context           *buildExecutionContextMaterial   `json:"context,omitempty"`
	Destination       *buildExecutionRegistryMaterial  `json:"destination,omitempty"`
	Source            *buildExecutionSourceMaterial    `json:"source,omitempty"`
	PlatformArtifacts []buildExecutionPlatformArtifact `json:"platform_artifacts,omitempty"`
}

type buildExecutionContextMaterial struct {
	Root           string                   `json:"root"`
	ContextPath    string                   `json:"context_path"`
	DockerfilePath string                   `json:"dockerfile_path"`
	Repository     string                   `json:"repository"`
	Reference      string                   `json:"reference"`
	Platform       string                   `json:"platform,omitempty"`
	BuildArgs      []buildExecutionBuildArg `json:"build_args,omitempty"`
}

type buildExecutionBuildArg struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type buildExecutionRegistryMaterial struct {
	Endpoint   string `json:"endpoint"`
	Repository string `json:"repository"`
	Reference  string `json:"reference"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

type buildExecutionSourceMaterial struct {
	Endpoint   string `json:"endpoint"`
	Repository string `json:"repository"`
	Digest     string `json:"digest"`
	MediaType  string `json:"media_type"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

type buildExecutionPlatformArtifact struct {
	Platform  string `json:"platform"`
	Digest    string `json:"digest"`
	MediaType string `json:"media_type"`
	SizeBytes int64  `json:"size_bytes"`
}

type buildExecutionResult struct {
	ImageID      string `json:"image_id,omitempty"`
	Digest       string `json:"digest,omitempty"`
	Repository   string `json:"repository,omitempty"`
	Reference    string `json:"reference,omitempty"`
	MediaType    string `json:"media_type,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	OS           string `json:"os,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Variant      string `json:"variant,omitempty"`
}

func registerBuildExternalExecution(registrar moduleapi.TaskRuntimeRegistrar, dependencies buildExternalExecutionDependencies) error {
	if registrar == nil || dependencies.repository == nil || dependencies.service == nil {
		return errors.New("build external execution dependencies are unavailable")
	}
	for _, executorType := range []moduleapi.StageExecutorType{buildStageExecutor, v2BuildStageExecutor, artifactPromotionStageExecutor} {
		handler := &buildExternalExecutionHandler{executorType: executorType, dependencies: dependencies}
		if err := registrar.RegisterExternalExecutionMaterialResolver(handler); err != nil {
			return err
		}
		if err := registrar.RegisterExternalExecutionResultRecorder(handler); err != nil {
			return err
		}
	}
	return nil
}

func (h *buildExternalExecutionHandler) Type() moduleapi.StageExecutorType {
	if h == nil {
		return ""
	}
	return h.executorType
}

func (h *buildExternalExecutionHandler) ResolveExternalExecutionMaterial(ctx context.Context, request moduleapi.ExternalExecutionMaterialRequest) (moduleapi.ExternalExecutionMaterial, error) {
	if err := h.validateRequest(request.ExecutorType); err != nil {
		return moduleapi.ExternalExecutionMaterial{}, err
	}
	var material buildExecutionMaterial
	var err error
	switch h.executorType {
	case buildStageExecutor:
		material, err = h.resolveLegacyMaterial(ctx, request)
	case v2BuildStageExecutor:
		material, err = h.resolveV2Material(ctx, request)
	case artifactPromotionStageExecutor:
		material, err = h.resolvePromotionMaterial(ctx, request)
	default:
		err = errors.New("build external executor is unsupported")
	}
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, err
	}
	payload, err := json.Marshal(material)
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, errors.New("encode build execution material")
	}
	return moduleapi.ExternalExecutionMaterial{Protocol: buildExecutionMaterialProtocol, Payload: payload}, nil
}

func (h *buildExternalExecutionHandler) RecordExternalExecutionResult(ctx context.Context, request moduleapi.ExternalExecutionResultRequest) error {
	if err := h.validateRequest(request.ExecutorType); err != nil {
		return err
	}
	if request.Protocol != buildExecutionResultProtocol {
		return errors.New("build execution result protocol is invalid")
	}
	var result buildExecutionResult
	if err := strictDecodeJSON(request.Result, &result); err != nil {
		return errors.New("build execution result is invalid")
	}
	if result.SizeBytes < 0 {
		return errors.New("build execution result is invalid")
	}
	switch h.executorType {
	case buildStageExecutor:
		return h.recordLegacyResult(ctx, request, result)
	case v2BuildStageExecutor:
		return h.recordV2Result(ctx, request, result)
	case artifactPromotionStageExecutor:
		return h.recordPromotionResult(ctx, request, result)
	default:
		return errors.New("build external executor is unsupported")
	}
}

func (h *buildExternalExecutionHandler) validateRequest(executorType moduleapi.StageExecutorType) error {
	if h == nil || h.dependencies.repository == nil || h.dependencies.service == nil || executorType != h.executorType {
		return errors.New("build external execution request is invalid")
	}
	return nil
}

// COMPAT(owner=Build Task Runtime canonical executor registry, cleanup=all pre-v2 build tasks settled)
// 旧版作业在全部结算后移除；当前仅基于冻结作业身份重新授权工作区并重建材料。
func (h *buildExternalExecutionHandler) resolveLegacyMaterial(ctx context.Context, request moduleapi.ExternalExecutionMaterialRequest) (buildExecutionMaterial, error) {
	input, err := decodeLegacyTaskInput(request)
	if err != nil {
		return buildExecutionMaterial{}, err
	}
	job, err := h.dependencies.repository.GetJobByTaskID(ctx, request.TaskID)
	if err != nil {
		return buildExecutionMaterial{}, err
	}
	if err := validateLegacyJobInput(input, job, request.RuntimeTargetID); err != nil {
		return buildExecutionMaterial{}, err
	}
	live, err := h.resolveLegacyBuildContext(ctx, job)
	if err != nil {
		return buildExecutionMaterial{}, err
	}
	return buildExecutionMaterial{Context: legacyExecutionContext(live, job)}, nil
}

func decodeLegacyTaskInput(request moduleapi.ExternalExecutionMaterialRequest) (moduleapi.BuildTaskInput, error) {
	if request.OperationID != buildImageLocalOperation {
		return moduleapi.BuildTaskInput{}, errors.New("build execution operation is invalid")
	}
	var input moduleapi.BuildTaskInput
	if err := strictDecodeJSON(request.Input, &input); err != nil || strings.TrimSpace(input.BuildID) == "" {
		return moduleapi.BuildTaskInput{}, errors.New("build task input is invalid")
	}
	return input, nil
}

func validateLegacyJobInput(input moduleapi.BuildTaskInput, job buildstore.JobSnapshot, runtimeTargetID int64) error {
	if job.BuildID != input.BuildID || job.RuntimeTargetID > uint64(^uint64(0)>>1) || int64(job.RuntimeTargetID) != runtimeTargetID {
		return errors.New("build task input does not match frozen job")
	}
	return nil
}

func (h *buildExternalExecutionHandler) resolveLegacyBuildContext(ctx context.Context, job buildstore.JobSnapshot) (moduleapi.ApplicationBuildContext, error) {
	if h.dependencies.service.contexts == nil {
		return moduleapi.ApplicationBuildContext{}, errors.New("application build context resolver is unavailable")
	}
	live, err := h.dependencies.service.contexts.ResolveApplicationBuildContext(ctx, job.ApplicationID)
	if err != nil {
		return moduleapi.ApplicationBuildContext{}, errors.New("application build context resolution failed")
	}
	if live.ApplicationID != job.ApplicationID || live.RuntimeTargetID != job.RuntimeTargetID || !live.CanBuild || strings.TrimSpace(live.WorkspaceRoot) == "" {
		return moduleapi.ApplicationBuildContext{}, errors.New("application build context no longer authorizes execution")
	}
	return live, nil
}

func legacyExecutionContext(live moduleapi.ApplicationBuildContext, job buildstore.JobSnapshot) *buildExecutionContextMaterial {
	buildArgs := make([]buildExecutionBuildArg, 0, len(job.BuildArgs))
	for _, argument := range job.BuildArgs {
		buildArgs = append(buildArgs, buildExecutionBuildArg{Name: argument.Name, Value: argument.Value})
	}
	return &buildExecutionContextMaterial{Root: live.WorkspaceRoot, ContextPath: job.ContextPath, DockerfilePath: job.DockerfilePath, Repository: job.ImageRepository, Reference: job.ImageTag, BuildArgs: buildArgs}
}

//nolint:cyclop // V2 material joins plan placement, snapshot and ephemeral registry credentials in one fenced window.
func (h *buildExternalExecutionHandler) resolveV2Material(ctx context.Context, request moduleapi.ExternalExecutionMaterialRequest) (buildExecutionMaterial, error) {
	var input moduleapi.BuildPlanTaskInput
	if err := strictDecodeJSON(request.Input, &input); err != nil || input.ExecutionPlanID == "" {
		return buildExecutionMaterial{}, errors.New("build execution plan input is invalid")
	}
	reader, ok := h.dependencies.repository.(buildstore.ExecutionPlanReader)
	if !ok {
		return buildExecutionMaterial{}, errors.New("build execution plan reader is unavailable")
	}
	plan, err := reader.GetExecutionPlanByTaskID(ctx, request.TaskID)
	if err != nil {
		return buildExecutionMaterial{}, err
	}
	if plan.ID != input.ExecutionPlanID || input.BuildID != plan.ID {
		return buildExecutionMaterial{}, errors.New("build execution plan identity does not match task input")
	}
	if request.OperationID == buildManifestOperation {
		return h.resolveManifestMaterial(ctx, request, input, plan)
	}
	if request.OperationID != buildImagePublishOperation {
		return buildExecutionMaterial{}, errors.New("build execution operation is invalid")
	}
	platform, reference, targetID, err := buildExecutionLeg(plan, input)
	if err != nil || targetID != request.RuntimeTargetID {
		return buildExecutionMaterial{}, errors.New("build execution placement does not match task input")
	}
	if err := h.beginBuilderReservation(ctx, request, plan, platform); err != nil {
		return buildExecutionMaterial{}, err
	}
	root, err := resolveMaterializationReference(plan.Workspace.MaterializationRef)
	if err != nil {
		return buildExecutionMaterial{}, errors.New("build workspace materialization is invalid")
	}
	destination := moduleapi.AuthorizedArtifactDestination(plan.Destination)
	destination.Reference = reference
	registry, err := h.resolvePublicationMaterial(ctx, destination, "push")
	if err != nil {
		return buildExecutionMaterial{}, err
	}
	return buildExecutionMaterial{Context: &buildExecutionContextMaterial{Root: root, ContextPath: ".", DockerfilePath: "Dockerfile", Repository: plan.Destination.RepositoryRef, Reference: reference, Platform: platform}, Destination: &registry}, nil
}

//nolint:cyclop // Manifest material validates all platform artifacts and destination credentials before publication.
func (h *buildExternalExecutionHandler) resolveManifestMaterial(ctx context.Context, request moduleapi.ExternalExecutionMaterialRequest, input moduleapi.BuildPlanTaskInput, plan moduleapi.BuildExecutionPlan) (buildExecutionMaterial, error) {
	if len(plan.Platforms) < 2 || input.Platform != "" || input.LegID != "" || manifestRuntimeTargetID(plan) != request.RuntimeTargetID {
		return buildExecutionMaterial{}, errors.New("build manifest input is invalid")
	}
	repository, ok := h.dependencies.repository.(buildstore.PlatformArtifactRepository)
	if !ok {
		return buildExecutionMaterial{}, errors.New("platform artifact repository is unavailable")
	}
	manifest, err := repository.PrepareOCIManifestPublication(ctx, request.TaskID, plan)
	if err != nil {
		return buildExecutionMaterial{}, err
	}
	destination, err := h.resolvePublicationMaterial(ctx, manifest.Destination, "manifest-push")
	if err != nil {
		return buildExecutionMaterial{}, err
	}
	artifacts := make([]buildExecutionPlatformArtifact, 0, len(manifest.PlatformArtifacts))
	for _, artifact := range manifest.PlatformArtifacts {
		digest, valid := normalizePlatformDigest(artifact.Digest)
		if !valid || !containsString(plan.Platforms, artifact.Platform) || strings.TrimSpace(artifact.MediaType) == "" || artifact.SizeBytes < 0 {
			return buildExecutionMaterial{}, errors.New("platform artifact material is invalid")
		}
		artifacts = append(artifacts, buildExecutionPlatformArtifact{Platform: artifact.Platform, Digest: digest, MediaType: artifact.MediaType, SizeBytes: artifact.SizeBytes})
	}
	return buildExecutionMaterial{Destination: &destination, PlatformArtifacts: artifacts}, nil
}

//nolint:cyclop // Promotion material binds source identity and destination credentials within the same fence.
func (h *buildExternalExecutionHandler) resolvePromotionMaterial(ctx context.Context, request moduleapi.ExternalExecutionMaterialRequest) (buildExecutionMaterial, error) {
	if request.OperationID != buildArtifactCopyOperation || h.dependencies.artifactCopyRegistry == nil {
		return buildExecutionMaterial{}, errors.New("build artifact copy operation is invalid")
	}
	var input moduleapi.ArtifactPromotionTaskInput
	if err := strictDecodeJSON(request.Input, &input); err != nil || input.RuntimeTargetID != request.RuntimeTargetID || input.RuntimeTargetID < 1 {
		return buildExecutionMaterial{}, errors.New("build artifact promotion input is invalid")
	}
	digest, valid := normalizePlatformDigest(input.Source.Digest)
	if !valid || strings.TrimSpace(input.Source.MediaType) == "" {
		return buildExecutionMaterial{}, errors.New("build artifact promotion source is invalid")
	}
	binding, err := h.dependencies.artifactCopyRegistry.ResolveArtifactCopyBinding(ctx, moduleapi.AuthorizedArtifactCopy{Source: input.Source, Destination: input.Destination})
	if err != nil {
		return buildExecutionMaterial{}, errors.New("artifact copy binding resolution failed")
	}
	if binding.Destination.Destination != input.Destination || binding.SourceAuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral || binding.Destination.AuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		return buildExecutionMaterial{}, errors.New("build artifact copy binding is invalid")
	}
	sourceCredential, err := h.resolveCredentialMaterial(ctx, binding.SourceCredentialRef, binding.SourceEndpoint, input.Source.RepositoryRef, "pull")
	if err != nil {
		return buildExecutionMaterial{}, err
	}
	destinationCredential, err := h.resolveCredentialMaterial(ctx, binding.Destination.CredentialRef, binding.Destination.Endpoint, input.Destination.RepositoryRef, "push")
	if err != nil {
		return buildExecutionMaterial{}, err
	}
	return buildExecutionMaterial{Source: &buildExecutionSourceMaterial{Endpoint: binding.SourceEndpoint, Repository: input.Source.RepositoryRef, Digest: digest, MediaType: input.Source.MediaType, Username: sourceCredential.Username, Password: sourceCredential.Secret}, Destination: &buildExecutionRegistryMaterial{Endpoint: binding.Destination.Endpoint, Repository: input.Destination.RepositoryRef, Reference: input.Destination.Reference, Username: destinationCredential.Username, Password: destinationCredential.Secret}}, nil
}

func (h *buildExternalExecutionHandler) resolvePublicationMaterial(ctx context.Context, destination moduleapi.AuthorizedArtifactDestination, operation string) (buildExecutionRegistryMaterial, error) {
	if h.dependencies.registry == nil {
		return buildExecutionRegistryMaterial{}, errors.New("registry publication resolver is unavailable")
	}
	binding, err := h.dependencies.registry.ResolvePublicationBinding(ctx, destination)
	if err != nil {
		return buildExecutionRegistryMaterial{}, errors.New("registry publication binding resolution failed")
	}
	if binding.Destination != destination || binding.AuthExecution.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		return buildExecutionRegistryMaterial{}, errors.New("registry publication binding is invalid")
	}
	credential, err := h.resolveCredentialMaterial(ctx, binding.CredentialRef, binding.Endpoint, destination.RepositoryRef, operation)
	if err != nil {
		return buildExecutionRegistryMaterial{}, err
	}
	return buildExecutionRegistryMaterial{Endpoint: binding.Endpoint, Repository: destination.RepositoryRef, Reference: destination.Reference, Username: credential.Username, Password: credential.Secret}, nil
}

func (h *buildExternalExecutionHandler) resolveCredentialMaterial(ctx context.Context, credentialRef, endpoint, repository, operation string) (material moduleapi.EphemeralCredentialMaterial, err error) {
	if h.dependencies.credentials == nil || h.dependencies.credentialMaterials == nil || strings.TrimSpace(credentialRef) == "" || strings.TrimSpace(endpoint) == "" || strings.TrimSpace(repository) == "" {
		return moduleapi.EphemeralCredentialMaterial{}, errors.New("registry credential material dependencies are unavailable")
	}
	session, err := h.dependencies.credentials.Prepare(ctx, moduleapi.CredentialRequest{CredentialRef: credentialRef, Endpoint: endpoint, RepositoryRef: repository, Operation: operation, ExpiresAt: time.Now().UTC().Add(buildCredentialSessionTTL)})
	if err != nil {
		return moduleapi.EphemeralCredentialMaterial{}, errors.New("registry credential preparation failed")
	}
	defer func() {
		if revokeErr := h.dependencies.credentials.Revoke(context.WithoutCancel(ctx), session); revokeErr != nil {
			material = moduleapi.EphemeralCredentialMaterial{}
			err = errors.New("registry credential cleanup could not be verified")
		}
	}()
	material, err = h.dependencies.credentialMaterials.ResolveCredentialMaterial(ctx, session, moduleapi.CredentialInjectionTarget{Endpoint: endpoint, RepositoryRef: repository})
	if err != nil {
		return moduleapi.EphemeralCredentialMaterial{}, errors.New("registry credential material resolution failed")
	}
	return material, nil
}

//nolint:gocognit,gocyclo,cyclop // V2 settlement separates manifest, platform-leg and single-artifact interpretations.
func (h *buildExternalExecutionHandler) recordV2Result(ctx context.Context, request moduleapi.ExternalExecutionResultRequest, result buildExecutionResult) error {
	var input moduleapi.BuildPlanTaskInput
	if err := strictDecodeJSON(request.Input, &input); err != nil {
		return errors.New("build execution plan input is invalid")
	}
	reader, ok := h.dependencies.repository.(buildstore.ExecutionPlanReader)
	if !ok {
		return errors.New("build execution plan reader is unavailable")
	}
	plan, err := reader.GetExecutionPlanByTaskID(ctx, request.TaskID)
	if err != nil {
		return err
	}
	if plan.ID != input.ExecutionPlanID || input.BuildID != plan.ID {
		return errors.New("build execution plan identity does not match task input")
	}
	digest, valid := normalizePlatformDigest(result.Digest)
	if !valid {
		return errors.New("build execution result digest is invalid")
	}
	if request.OperationID == buildManifestOperation {
		if len(plan.Platforms) < 2 || input.Platform != "" || input.LegID != "" || manifestRuntimeTargetID(plan) != request.RuntimeTargetID || result.Repository != plan.Destination.RepositoryRef || result.Reference != plan.Destination.Reference || strings.TrimSpace(result.MediaType) == "" {
			return errors.New("build manifest result is invalid")
		}
		settler, ok := h.dependencies.repository.(buildstore.OCIManifestSettlementRepository)
		if !ok {
			return errors.New("OCI manifest settlement is unavailable")
		}
		return settler.SettleOCIManifestPublication(ctx, request.TaskID, plan, moduleapi.OCIManifestPublicationResult{Digest: digest, MediaType: result.MediaType, SizeBytes: result.SizeBytes}, moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral})
	}
	if request.OperationID != buildImagePublishOperation {
		return errors.New("build execution operation is invalid")
	}
	platform, reference, targetID, err := buildExecutionLeg(plan, input)
	if err != nil || targetID != request.RuntimeTargetID || result.Repository != plan.Destination.RepositoryRef || result.Reference != reference {
		return errors.New("build image result does not match execution plan")
	}
	if !resultMatchesPlatform(result, platform) {
		return errors.New("build image result platform does not match execution plan")
	}
	if input.LegID != "" {
		repository, ok := h.dependencies.repository.(buildstore.PlatformArtifactRepository)
		if !ok {
			return errors.New("platform artifact repository is unavailable")
		}
		mediaType := result.MediaType
		if mediaType == "" {
			mediaType = ociImageManifestMediaType
		}
		if err := repository.RecordPlatformArtifact(ctx, request.TaskID, plan, moduleapi.PlatformArtifact{LegID: input.LegID, Platform: platform, Digest: digest, MediaType: mediaType, SizeBytes: result.SizeBytes, ProducedAt: time.Now().UTC()}); err != nil {
			return err
		}
		return h.releaseBuilderReservation(ctx, request, plan, platform)
	}
	if err := settleV2Artifact(ctx, h.dependencies.repository, request.TaskID, plan, moduleapi.BuildArtifactResult{ImageID: result.ImageID, Digest: digest, Repository: result.Repository, Tag: result.Reference, SizeBytes: result.SizeBytes, OS: result.OS, Architecture: result.Architecture, Variant: result.Variant}, moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}); err != nil {
		return err
	}
	return h.releaseBuilderReservation(ctx, request, plan, platform)
}

// COMPAT(owner=Build Task Runtime canonical executor registry, cleanup=all pre-v2 build tasks settled)
// recordLegacyResult 保留存量 Task Runtime 结算路径，清理触发为全部 v2 前作业完成结算。
func (h *buildExternalExecutionHandler) recordLegacyResult(ctx context.Context, request moduleapi.ExternalExecutionResultRequest, result buildExecutionResult) error {
	if err := validateLegacyResultRequest(request, result); err != nil {
		return err
	}
	input, err := decodeLegacyResultInput(request)
	if err != nil {
		return err
	}
	job, err := h.dependencies.repository.GetJobByTaskID(ctx, request.TaskID)
	if err != nil {
		return err
	}
	if err := validateLegacyResultAgainstJob(input, job, request.RuntimeTargetID, result); err != nil {
		return err
	}
	digest, err := optionalNormalizedDigest(result.Digest)
	if err != nil {
		return err
	}
	return h.dependencies.repository.SettleBuildArtifact(ctx, request.TaskID, moduleapi.BuildArtifactResult{ImageID: result.ImageID, Digest: digest, Repository: result.Repository, Tag: result.Reference, SizeBytes: result.SizeBytes, OS: result.OS, Architecture: result.Architecture, Variant: result.Variant})
}

func validateLegacyResultRequest(request moduleapi.ExternalExecutionResultRequest, result buildExecutionResult) error {
	if request.OperationID != buildImageLocalOperation || strings.TrimSpace(result.ImageID) == "" {
		return errors.New("build image result is invalid")
	}
	return nil
}

func decodeLegacyResultInput(request moduleapi.ExternalExecutionResultRequest) (moduleapi.BuildTaskInput, error) {
	var input moduleapi.BuildTaskInput
	if err := strictDecodeJSON(request.Input, &input); err != nil {
		return moduleapi.BuildTaskInput{}, errors.New("build task input is invalid")
	}
	return input, nil
}

func validateLegacyResultAgainstJob(input moduleapi.BuildTaskInput, job buildstore.JobSnapshot, runtimeTargetID int64, result buildExecutionResult) error {
	if job.BuildID != input.BuildID || job.RuntimeTargetID > uint64(^uint64(0)>>1) || int64(job.RuntimeTargetID) != runtimeTargetID || result.Repository != job.ImageRepository || result.Reference != job.ImageTag {
		return errors.New("build image result does not match frozen job")
	}
	return nil
}

func manifestRuntimeTargetID(plan moduleapi.BuildExecutionPlan) int64 {
	if plan.RuntimeTargetID > 0 {
		return plan.RuntimeTargetID
	}
	if len(plan.Platforms) == 0 {
		return 0
	}
	placement, ok := plan.PlacementForPlatform(plan.Platforms[0])
	if !ok {
		return 0
	}
	return placement.RuntimeTargetID
}

func (h *buildExternalExecutionHandler) recordPromotionResult(ctx context.Context, request moduleapi.ExternalExecutionResultRequest, result buildExecutionResult) error {
	if request.OperationID != buildArtifactCopyOperation {
		return errors.New("build artifact copy operation is invalid")
	}
	var input moduleapi.ArtifactPromotionTaskInput
	if err := strictDecodeJSON(request.Input, &input); err != nil || input.RuntimeTargetID != request.RuntimeTargetID {
		return errors.New("build artifact promotion input is invalid")
	}
	digest, valid := normalizePlatformDigest(result.Digest)
	if !valid || digest != input.Source.Digest || result.MediaType != input.Source.MediaType || result.Repository != input.Destination.RepositoryRef || result.Reference != input.Destination.Reference {
		return errors.New("build artifact promotion result does not match source")
	}
	return h.dependencies.service.SettleArtifactPromotion(ctx, moduleapi.OCIArtifactCopyInput{Source: input.Source, Destination: input.Destination}, moduleapi.OCIArtifactCopyResult{Digest: digest, MediaType: result.MediaType, SizeBytes: result.SizeBytes}, moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral})
}

//nolint:revive // Four return values make the platform/reference/target/error tuple explicit at this narrow internal seam.
func buildExecutionLeg(plan moduleapi.BuildExecutionPlan, input moduleapi.BuildPlanTaskInput) (platform, reference string, targetID int64, err error) {
	if len(plan.Platforms) == 0 {
		return "", "", 0, errors.New("build execution plan has no platform")
	}
	platform = input.Platform
	if len(plan.Platforms) == 1 {
		if input.LegID != "" || (platform != "" && platform != plan.Platforms[0]) {
			return "", "", 0, errors.New("single-platform build input is invalid")
		}
		platform = plan.Platforms[0]
		reference = plan.Destination.Reference
	} else {
		if strings.TrimSpace(platform) == "" || strings.TrimSpace(input.LegID) == "" || !containsString(plan.Platforms, platform) {
			return "", "", 0, errors.New("multi-platform build leg is invalid")
		}
		reference = temporaryPlatformTag(plan.Destination.Reference, input.LegID)
	}
	placement, found := plan.PlacementForPlatform(platform)
	if !found {
		return "", "", 0, errors.New("build execution placement is missing")
	}
	return platform, reference, placement.RuntimeTargetID, nil
}

//nolint:cyclop,nestif // Reservation retries must preserve capacity evidence and the same fencing token.
func (h *buildExternalExecutionHandler) beginBuilderReservation(ctx context.Context, request moduleapi.ExternalExecutionMaterialRequest, plan moduleapi.BuildExecutionPlan, platform string) error {
	repository, ok := h.dependencies.repository.(moduleapi.BuilderReservationRepository)
	if !ok {
		return errors.New("builder reservation repository is unavailable")
	}
	placement, found := plan.PlacementForPlatform(platform)
	if !found || request.Attempt < 1 {
		return errors.New("builder reservation input is invalid")
	}
	slotBudget, observedAt, err := externalReservationBudget(placement)
	if err != nil {
		return err
	}
	fence := buildstore.BuilderReservationFence(plan.ID, request.TaskID, platform, request.Attempt)
	if request.Attempt == 1 {
		err = repository.MarkBuilderReservationRunning(ctx, request.TaskID, platform, fence)
	} else {
		now := time.Now().UTC()
		reservation := moduleapi.BuilderReservation{ID: fmt.Sprintf("reservation_%s_%s_%d", plan.ID, platform, request.Attempt), InstanceID: placement.BuilderInstanceID, PlanID: plan.ID, TaskID: request.TaskID, Attempt: request.Attempt, LegID: platform, FenceToken: fence, State: moduleapi.BuilderReservationRunning, LeaseExpiresAt: now.Add(buildstore.BuilderReservationLeaseTTL), CreatedAt: now, UpdatedAt: now}
		if observed, ok := repository.(buildObservedRetryReservationRepository); ok {
			_, err = observed.ReserveBuilderAttemptWithCapacityAfterObservation(ctx, reservation, slotBudget, observedAt)
		} else if capacity, ok := repository.(buildRetryReservationRepository); ok {
			_, err = capacity.ReserveBuilderAttemptWithCapacity(ctx, reservation, slotBudget)
		} else {
			return errors.New("builder retry reservation repository is unavailable")
		}
	}
	if err != nil {
		// 首次尝试重放可能观察到已运行的 reservation；仅在这一窄场景续租匹配
		// fence，不能把 reservation 失败或重试容量冲突变成可执行租约。
		if request.Attempt != 1 || !errors.Is(err, buildstore.ErrConflict) {
			return fmt.Errorf("start builder reservation: %w", err)
		}
		if renewErr := repository.RenewBuilderReservation(ctx, request.TaskID, platform, fence, time.Now().UTC().Add(buildstore.BuilderReservationLeaseTTL)); renewErr != nil {
			return fmt.Errorf("start builder reservation: %w", err)
		}
		return nil
	}
	if err := repository.RenewBuilderReservation(ctx, request.TaskID, platform, fence, time.Now().UTC().Add(buildstore.BuilderReservationLeaseTTL)); err != nil {
		return fmt.Errorf("renew builder reservation: %w", err)
	}
	return nil
}

func externalReservationBudget(placement moduleapi.BuilderPlacement) (int, time.Time, error) {
	var evidence struct {
		ReservationSlotBudget int       `json:"reservation_slot_budget"`
		ReservationObservedAt time.Time `json:"reservation_observed_at"`
	}
	if len(placement.SchedulingEvidence) == 0 || json.Unmarshal(placement.SchedulingEvidence, &evidence) != nil || evidence.ReservationSlotBudget < 1 {
		return 0, time.Time{}, errors.New("build execution placement has no reservation budget")
	}
	if (placement.SchedulingPolicy == "least_load" || placement.SchedulingPolicy == "capacity" || placement.SchedulingPolicy == "affinity") && evidence.ReservationObservedAt.IsZero() {
		return 0, time.Time{}, errors.New("dynamic build execution placement has no observation time")
	}
	return evidence.ReservationSlotBudget, evidence.ReservationObservedAt.UTC(), nil
}

func (h *buildExternalExecutionHandler) releaseBuilderReservation(ctx context.Context, request moduleapi.ExternalExecutionResultRequest, plan moduleapi.BuildExecutionPlan, platform string) error {
	repository, ok := h.dependencies.repository.(moduleapi.BuilderReservationRepository)
	if !ok {
		return errors.New("builder reservation repository is unavailable")
	}
	fence := buildstore.BuilderReservationFence(plan.ID, request.TaskID, platform, request.Attempt)
	err := repository.ReleaseBuilderReservation(ctx, request.TaskID, platform, fence, moduleapi.BuilderReservationReleased)
	if errors.Is(err, buildstore.ErrConflict) {
		return nil
	}
	return err
}

func resultMatchesPlatform(result buildExecutionResult, platform string) bool {
	parts := strings.Split(platform, "/")
	if len(parts) < 2 || len(parts) > 3 || result.OS != parts[0] || result.Architecture != parts[1] {
		return false
	}
	expectedVariant := ""
	const platformPartsWithVariant = 3
	if len(parts) == platformPartsWithVariant {
		expectedVariant = parts[2]
	}
	return result.Variant == expectedVariant
}

func strictDecodeJSON(raw json.RawMessage, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("unexpected trailing JSON data")
	}
	return nil
}

func optionalNormalizedDigest(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	digest, valid := normalizePlatformDigest(value)
	if !valid {
		return "", errors.New("build execution result digest is invalid")
	}
	return digest, nil
}
