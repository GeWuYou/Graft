package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

const (
	applicationTaskOwnerType          = "application"
	composeStagePrefix                = "application.compose."
	destroyCleanupStageType           = "application.cleanup.destroy"
	composeMaterialProtocol           = "application-compose-material/v1"
	composeExecutionProtocol          = "application-compose/v1"
	composeExecutionCapability        = "compose_execution"
	composeExecutionCapabilityVersion = "docker/v1"
)

// composeExecutionPolicy 冻结 Application 拥有的执行语义，不包含 provider 命令或运行时连接材料。
type composeExecutionPolicy struct {
	SnapshotDigest      string `json:"snapshot_digest"`
	BuildBeforeUp       bool   `json:"build_before_up,omitempty"`
	ForceRecreate       bool   `json:"force_recreate,omitempty"`
	RemoveOrphans       bool   `json:"remove_orphans,omitempty"`
	WaitAfterUp         bool   `json:"wait_after_up,omitempty"`
	WaitTimeoutSeconds  int    `json:"wait_timeout_seconds,omitempty"`
	RenewAnonVolumes    bool   `json:"renew_anon_volumes,omitempty"`
	RemoveNamedVolumes  bool   `json:"remove_named_volumes,omitempty"`
	DeleteWorkspacePath bool   `json:"delete_workspace_path,omitempty"`
	AutoUnregister      bool   `json:"auto_unregister,omitempty"`
	ActorID             uint64 `json:"actor_id,omitempty"`
}

// composeStageInput 是 Task 持久化的 provider-neutral Application 意图。
type composeStageInput struct {
	ApplicationID string                 `json:"application_id"`
	Policy        composeExecutionPolicy `json:"policy"`
}

// composeExecutionMaterial 仅在已围栏 Agent 请求期间解析，不进入 Task、日志或回执。
type composeExecutionMaterial struct {
	WorkspacePath       string   `json:"workspace_path"`
	ProjectName         string   `json:"project_name"`
	ComposeFiles        []string `json:"compose_files"`
	EnvFiles            []string `json:"env_files"`
	Profiles            []string `json:"profiles"`
	ManagedServiceNames []string `json:"managed_service_names"`
}

type composeExecutionMaterialResolver struct {
	typeName moduleapi.StageExecutorType
	service  *Service
}

func (r composeExecutionMaterialResolver) Type() moduleapi.StageExecutorType { return r.typeName }

func (r composeExecutionMaterialResolver) ResolveExternalExecutionMaterial(
	ctx context.Context,
	request moduleapi.ExternalExecutionMaterialRequest,
) (moduleapi.ExternalExecutionMaterial, error) {
	_, aggregate, err := r.resolveAggregate(ctx, request)
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, err
	}
	config := lifecycleConfigurationFromAggregate(aggregate)
	if strings.TrimSpace(config.WorkingDir) == "" || strings.TrimSpace(config.ApplicationName) == "" || len(config.ComposeFiles) == 0 {
		return moduleapi.ExternalExecutionMaterial{}, errProjectInvalidArgument
	}
	envFiles := collectFilesByKind(aggregate.Files, projectcontract.FileKindEnv.String())
	if !validMaterialFilePaths(config.WorkingDir, config.ComposeFiles) || !validMaterialFilePaths(config.WorkingDir, envFiles) {
		return moduleapi.ExternalExecutionMaterial{}, errProjectInvalidArgument
	}
	material := composeExecutionMaterial{
		WorkspacePath:       config.WorkingDir,
		ProjectName:         config.ApplicationName,
		ComposeFiles:        append([]string{}, config.ComposeFiles...),
		EnvFiles:            append([]string{}, envFiles...),
		Profiles:            append([]string{}, config.Standard.Profiles...),
		ManagedServiceNames: append([]string{}, config.Standard.ManagedServiceNames...),
	}
	payload, err := json.Marshal(material)
	if err != nil {
		return moduleapi.ExternalExecutionMaterial{}, fmt.Errorf("encode application compose material: %w", err)
	}
	return moduleapi.ExternalExecutionMaterial{Protocol: composeMaterialProtocol, Payload: payload}, nil
}

func validMaterialFilePaths(workspacePath string, paths []string) bool {
	workspacePath = filepath.Clean(strings.TrimSpace(workspacePath))
	if !filepath.IsAbs(workspacePath) {
		return false
	}
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if !filepath.IsAbs(path) {
			return false
		}
		relative, err := filepath.Rel(workspacePath, path)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return false
		}
	}
	return true
}

func (r composeExecutionMaterialResolver) resolveAggregate(
	ctx context.Context,
	request moduleapi.ExternalExecutionMaterialRequest,
) (composeStageInput, projectstore.ApplicationAggregate, error) {
	var input composeStageInput
	if request.ExecutorType != r.typeName || decodeComposeStageInput(request.Input, &input) != nil || !validComposeStageInput(input) {
		return composeStageInput{}, projectstore.ApplicationAggregate{}, errProjectInvalidArgument
	}
	if r.service == nil {
		return composeStageInput{}, projectstore.ApplicationAggregate{}, errors.New("project service is unavailable")
	}
	recordID, err := r.service.ResolveApplicationID(ctx, input.ApplicationID)
	if err != nil {
		return composeStageInput{}, projectstore.ApplicationAggregate{}, err
	}
	aggregate, err := r.service.getAggregate(ctx, recordID)
	if err != nil {
		return composeStageInput{}, projectstore.ApplicationAggregate{}, err
	}
	if aggregate.Snapshot == nil || aggregate.Snapshot.ConfigHash != input.Policy.SnapshotDigest {
		return composeStageInput{}, projectstore.ApplicationAggregate{}, errProjectLifecycleReview
	}
	return input, aggregate, nil
}

type destroyCleanupExecutor struct{ service *Service }

func (e destroyCleanupExecutor) Type() moduleapi.StageExecutorType {
	return moduleapi.StageExecutorType(destroyCleanupStageType)
}

func (e destroyCleanupExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	var input composeStageInput
	if decodeComposeStageInput(run.Input(), &input) != nil || strings.TrimSpace(input.ApplicationID) == "" || input.Policy.ActorID == 0 {
		return errProjectInvalidArgument
	}
	if e.service == nil {
		return errors.New("project service is unavailable")
	}
	recordID, err := e.service.ResolveApplicationID(ctx, input.ApplicationID)
	if errors.Is(err, errProjectNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	aggregate, err := e.service.getAggregate(ctx, recordID)
	if err != nil {
		return err
	}
	request := DestroyRequest{DeleteWorkspacePath: input.Policy.DeleteWorkspacePath, AutoUnregister: input.Policy.AutoUnregister}
	guards, autoUnregister, err := e.service.applyDestroyWorkspacePathStep(aggregate, request, nil)
	if err != nil {
		return err
	}
	_, err = e.service.applyDestroyUnregisterStep(ctx, recordID, actionActor{id: input.Policy.ActorID}, guards, autoUnregister)
	return err
}

func (destroyCleanupExecutor) Cancel(context.Context, moduleapi.StageRun) error { return nil }

func validComposeStageInput(input composeStageInput) bool {
	return strings.TrimSpace(input.ApplicationID) != "" && strings.TrimSpace(input.Policy.SnapshotDigest) != ""
}

func decodeComposeStageInput(raw json.RawMessage, destination *composeStageInput) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errProjectInvalidArgument
	}
	return nil
}

// registerProjectTaskExecution 注册外部 Compose 材料解析器、领域清理阶段与 Task owner 鉴权器。
func registerProjectTaskExecution(registrar moduleapi.TaskRuntimeRegistrar, service *Service) error {
	if registrar == nil {
		return errors.New("task runtime registrar is unavailable")
	}
	for _, action := range []string{"down", "pull", "up", "stop", "restart", "image-prune"} {
		resolver := composeExecutionMaterialResolver{typeName: moduleapi.StageExecutorType(composeStagePrefix + action), service: service}
		if err := registrar.RegisterExternalExecutionMaterialResolver(resolver); err != nil {
			return err
		}
	}
	if err := registrar.RegisterStageExecutor(destroyCleanupExecutor{service: service}); err != nil {
		return err
	}
	return registrar.RegisterTaskOwnerAuthorizer(projectTaskOwnerAuthorizer{service: service})
}
