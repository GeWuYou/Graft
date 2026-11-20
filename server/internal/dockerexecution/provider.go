package dockerexecution

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	composecli "github.com/compose-spec/compose-go/v2/cli"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	containerderrdefs "github.com/containerd/errdefs"
	dockercommand "github.com/docker/cli/cli/command"
	dockerflags "github.com/docker/cli/cli/flags"
	composeapi "github.com/docker/compose/v5/pkg/api"
	compose "github.com/docker/compose/v5/pkg/compose"
	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"

	"graft/server/internal/moduleapi"
)

const (
	applicationComposeProtocol         = "application-compose/v1"
	applicationComposeMaterialProtocol = "application-compose-material/v1"
	containerExecutionProtocol         = "container-execution/v1"
	updateControllerProtocol           = "platform-update-controller/v1"
	updateControllerMaterialProtocol   = "platform-update-controller-material/v1"
	updateControllerReceiptProtocol    = "compose-runner/v2"
	updateControllerCapability         = "update_controller"
	updateControllerOperation          = "platform.update.controller.launch.v1"
	maxProviderReferenceLength         = 512
	// composeLoadFailureSummaryMaxLines 限制摘要最多合并两条结构化错误行，防止单条摘要过长。
	composeLoadFailureSummaryMaxLines = 2
)

const (
	failureInvalidIntent      = "invalid_execution_intent"
	failureUnsupportedAction  = "unsupported_execution_operation"
	failureRuntimeUnavailable = "docker_runtime_unavailable"
	failureResourceNotFound   = "docker_resource_not_found"
	failureResourceConflict   = "docker_resource_conflict"
	failureProviderOperation  = "provider_operation_failed"
	failureInterrupted        = "agent_execution_interrupted"
)

type applicationComposeIntent struct {
	ApplicationID string                   `json:"application_id"`
	Policy        applicationComposePolicy `json:"policy"`
}

type applicationComposePolicy struct {
	SnapshotDigest     string `json:"snapshot_digest"`
	BuildBeforeUp      bool   `json:"build_before_up"`
	ForceRecreate      bool   `json:"force_recreate"`
	RemoveOrphans      bool   `json:"remove_orphans"`
	WaitAfterUp        bool   `json:"wait_after_up"`
	WaitTimeoutSeconds int    `json:"wait_timeout_seconds"`
	RenewAnonVolumes   bool   `json:"renew_anon_volumes"`
	RemoveNamedVolumes bool   `json:"remove_named_volumes"`
}

type applicationComposeMaterial struct {
	WorkspacePath       string   `json:"workspace_path"`
	ProjectName         string   `json:"project_name"`
	ComposeFiles        []string `json:"compose_files"`
	EnvFiles            []string `json:"env_files"`
	Profiles            []string `json:"profiles"`
	ManagedServiceNames []string `json:"managed_service_names"`
}

type containerExecutionIntent struct {
	ContainerRef string                      `json:"container_ref,omitempty"`
	ImageRef     string                      `json:"image_ref,omitempty"`
	TargetRef    string                      `json:"target_ref,omitempty"`
	TagRef       string                      `json:"tag_ref,omitempty"`
	NetworkRef   string                      `json:"network_ref,omitempty"`
	VolumeRef    string                      `json:"volume_ref,omitempty"`
	Force        bool                        `json:"force,omitempty"`
	Name         string                      `json:"name,omitempty"`
	Driver       string                      `json:"driver,omitempty"`
	Internal     bool                        `json:"internal,omitempty"`
	Attachable   bool                        `json:"attachable,omitempty"`
	IPAM         *containerNetworkIPAMIntent `json:"ipam,omitempty"`
}

type containerNetworkIPAMIntent struct {
	Subnet  string `json:"subnet"`
	Gateway string `json:"gateway"`
}

func executeProviderOperation(ctx context.Context, c Environment, lease Request) Result {
	switch lease.Capability {
	case buildExecutionCapability:
		return executeBuildOperation(ctx, c, lease)
	case "container_execution":
		return executeContainerOperation(ctx, c, lease)
	case "compose_execution":
		return executeApplicationComposeOperation(ctx, c, lease)
	case updateControllerCapability:
		return executeUpdateControllerOperation(ctx, c, lease)
	default:
		return failedExecution(failureUnsupportedAction)
	}
}

type updateControllerIntent struct {
	OperationID   string `json:"operation_id"`
	SourceVersion string `json:"source_version"`
	TargetVersion string `json:"target_version"`
}

type updateControllerMaterial struct {
	ControllerReference string `json:"controller_reference"`
	InputBase64         string `json:"input_b64"`
	RecoveryStateBase64 string `json:"recovery_state_b64,omitempty"`
	RecoveryClaimID     string `json:"recovery_claim_id,omitempty"`
	Recovery            bool   `json:"recovery,omitempty"`
	ComposeRoot         string `json:"compose_root"`
	DockerSocket        string `json:"docker_socket"`
	StateVolume         string `json:"state_volume"`
	OperationID         string `json:"operation_id"`
}

type updateControllerRecoveryInput struct {
	OperationID   string          `json:"operation_id"`
	RunnerID      string          `json:"runner_id"`
	SourceVersion string          `json:"source_version"`
	TargetVersion string          `json:"target_version"`
	Strategy      string          `json:"deployment_strategy"`
	ExecutionMode string          `json:"execution_mode,omitempty"`
	LaunchStageID uint64          `json:"launch_stage_id,omitempty"`
	LaunchAttempt int             `json:"launch_attempt,omitempty"`
	State         json.RawMessage `json:"state,omitempty"`
	Corrupt       bool            `json:"corrupt,omitempty"`
}

// executeUpdateControllerOperation 是调用方授权后的有界 Controller 启动边界。
// 材料由调用方按当前 attempt 解析，SDK 仅返回受限操作结果。
//
//nolint:cyclop,gocognit,gocyclo // 每个固定资源边界都以稳定代码 fail closed；控制器拉取、创建、启动和不确定启动分支属于同一提供方事务。
func executeUpdateControllerOperation(ctx context.Context, c Environment, lease Request) Result {
	if lease.Protocol != updateControllerProtocol || lease.OperationID != updateControllerOperation {
		return failedExecution(failureInvalidIntent)
	}
	materialResponse, err := c.ResolveMaterial(ctx, lease)
	if err != nil || materialResponse.Protocol != updateControllerMaterialProtocol {
		return failedExecution(failureInvalidIntent)
	}
	var material updateControllerMaterial
	if strictDecode(materialResponse.Payload, &material) != nil || !validUpdateControllerMaterial(material, lease) {
		return failedExecution(failureInvalidIntent)
	}
	docker, err := mobyclient.New(mobyclient.WithHost(c.DockerSocket))
	if err != nil {
		return failedExecution(failureRuntimeUnavailable)
	}
	defer func() { _ = docker.Close() }()
	pull, err := docker.ImagePull(ctx, material.ControllerReference, mobyclient.ImagePullOptions{})
	if err != nil {
		return failedExecution(failureRuntimeUnavailable)
	}
	if err := consumeDockerProgress(pull); err != nil {
		return failedExecutionAt(failureRuntimeUnavailable, "controller_pull")
	}
	configuration, host := updateControllerContainerConfig(material)
	name := materialContainerName(material.OperationID)
	if material.Recovery {
		name = "graft-update-recovery-" + material.OperationID + "-" + material.RecoveryClaimID
	}
	created, err := docker.ContainerCreate(ctx, mobyclient.ContainerCreateOptions{
		Name: name, Config: &configuration, HostConfig: &host,
	})
	if err != nil {
		return failedExecution(failureProviderOperation)
	}
	if _, err := docker.ContainerStart(ctx, created.ID, mobyclient.ContainerStartOptions{}); err != nil {
		// 启动响应丢失时 Controller 可能已执行替换；后续删除也无法撤销它的副作用。
		if !containerderrdefs.IsConflict(err) && !containerderrdefs.IsInvalidArgument(err) && !containerderrdefs.IsPermissionDenied(err) {
			return Result{Outcome: "needs_attention", FailureCode: failureInterrupted}
		}
		if cleanupErr := removeUnstartedController(ctx, docker, created.ID); cleanupErr != nil {
			return Result{Outcome: "needs_attention", FailureCode: failureInterrupted}
		}
		return failedExecution(failureProviderOperation)
	}
	return Result{Outcome: "success"}
}

//nolint:gocyclo,cyclop // 严格材料校验必须显式保留每个信任边界拒绝分支。
func validUpdateControllerMaterial(material updateControllerMaterial, lease Request) bool {
	var intent updateControllerIntent
	if strictDecode(lease.Input, &intent) != nil || strings.TrimSpace(intent.SourceVersion) == "" || strings.TrimSpace(intent.TargetVersion) == "" || strings.TrimSpace(intent.SourceVersion) == strings.TrimSpace(intent.TargetVersion) {
		return false
	}
	if !validDigestPinnedReference(material.ControllerReference) || !validRuntimeStateVolume(material.StateVolume) || material.OperationID != intent.OperationID || !validUpdateOperationID(material.OperationID) {
		return false
	}
	if material.Recovery {
		if strings.TrimSpace(material.InputBase64) != "" || strings.TrimSpace(material.ComposeRoot) != "" || strings.TrimSpace(material.DockerSocket) != "" || !validRecoveryClaimID(material.RecoveryClaimID) {
			return false
		}
		return validRecoveryControllerInput(material.RecoveryStateBase64, intent)
	}
	return strings.TrimSpace(material.InputBase64) != "" && filepath.IsAbs(material.ComposeRoot) && material.ComposeRoot != string(filepath.Separator) &&
		material.DockerSocket == "/var/run/docker.sock" &&
		material.OperationID == intent.OperationID
}

func validRecoveryControllerInput(encoded string, intent updateControllerIntent) bool {
	decoded, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) == 0 {
		return false
	}
	var recovery updateControllerRecoveryInput
	if strictDecode(decoded, &recovery) != nil || !recovery.matchesIntent(intent) {
		return false
	}
	if !validRecoveryExecutionMode(recovery.ExecutionMode) || recovery.LaunchAttempt < 0 || (recovery.LaunchStageID == 0) != (recovery.LaunchAttempt == 0) {
		return false
	}
	return validRecoveryControllerState(recovery)
}

func (recovery updateControllerRecoveryInput) matchesIntent(intent updateControllerIntent) bool {
	return recovery.OperationID == intent.OperationID && validRecoveryRunnerID(recovery.RunnerID) &&
		recovery.SourceVersion == intent.SourceVersion && recovery.TargetVersion == intent.TargetVersion &&
		strings.TrimSpace(recovery.Strategy) != ""
}

func validRecoveryControllerState(recovery updateControllerRecoveryInput) bool {
	if len(recovery.State) == 0 || bytes.Equal(bytes.TrimSpace(recovery.State), []byte("null")) {
		return true
	}
	// 完整状态及其摘要由 Controller 校验，SDK 只复核外层恢复授权绑定。
	var state struct {
		OperationID   string `json:"operation_id"`
		RunnerID      string `json:"runner_id"`
		ExecutionMode string `json:"execution_mode"`
	}
	if json.Unmarshal(recovery.State, &state) != nil || state.OperationID != recovery.OperationID || state.RunnerID != recovery.RunnerID || !validRecoveryExecutionMode(state.ExecutionMode) {
		return false
	}
	stateMode := state.ExecutionMode
	if stateMode == "" {
		stateMode = "agent"
	}
	return recovery.ExecutionMode == "" || recovery.ExecutionMode == stateMode
}

func validRecoveryExecutionMode(mode string) bool {
	return mode == "" || mode == "agent" || mode == "local"
}

func validRecoveryClaimID(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "recovery-") && len(value) <= 128 && !strings.ContainsAny(value, "/\\ \t\r\n")
}

func validRecoveryRunnerID(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "runner-") && len(value) <= 128 && !strings.ContainsAny(value, "/\\ \t\r\n")
}

func validUpdateOperationID(value string) bool {
	value = strings.TrimSpace(value)
	return strings.HasPrefix(value, "update-") && len(value) <= 128 && !strings.ContainsAny(value, "/\\ \t\r\n")
}

func validDigestPinnedReference(value string) bool {
	value = strings.TrimSpace(value)
	marker := strings.LastIndex(value, "@sha256:")
	if marker <= 0 || len(value)-marker-8 != 64 {
		return false
	}
	_, err := hex.DecodeString(value[marker+8:])
	return err == nil
}

func validRuntimeStateVolume(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 128 && !strings.ContainsAny(value, "/\\ ")
}

func updateControllerContainerConfig(material updateControllerMaterial) (containertypes.Config, containertypes.HostConfig) {
	env := "GRAFT_UPDATE_RUNNER_INPUT_B64=" + material.InputBase64
	binds := []string{material.ComposeRoot + ":" + material.ComposeRoot + ":rw", "/var/run/docker.sock:/var/run/docker.sock:rw", material.StateVolume + ":/var/lib/graft/update-state:rw"}
	if material.Recovery {
		env = "GRAFT_UPDATE_RUNNER_RECOVERY_STATE_B64=" + material.RecoveryStateBase64
		binds = []string{material.StateVolume + ":/var/lib/graft/update-state:rw"}
	}
	labels := map[string]string{
		"io.graft.update.protocol":  updateControllerReceiptProtocol,
		"io.graft.update.operation": material.OperationID,
	}
	if material.Recovery {
		labels["io.graft.update.recovery"] = "true"
	}
	return containertypes.Config{
			Image:  material.ControllerReference,
			User:   "0:0",
			Env:    []string{env},
			Labels: labels,
		}, containertypes.HostConfig{
			Binds:          binds,
			NetworkMode:    "none",
			ReadonlyRootfs: true,
			CapDrop:        []string{"ALL"},
			// 状态和安装材料由 server 用户持有；短生命周期 Controller 跨 UID 读写需要绕过属主权限。
			CapAdd:      []string{"CHOWN", "DAC_OVERRIDE"},
			SecurityOpt: []string{"no-new-privileges:true"},
		}
}

func materialContainerName(operationID string) string { return "graft-update-" + operationID }

func removeUnstartedController(ctx context.Context, docker *mobyclient.Client, id string) error {
	inspected, err := docker.ContainerInspect(ctx, id, mobyclient.ContainerInspectOptions{})
	if err != nil {
		return err
	}
	state := inspected.Container.State
	if state == nil || state.Running || state.Status != containertypes.StateCreated {
		return errors.New("controller has no unstarted state confirmation")
	}
	startedAt, err := time.Parse(time.RFC3339Nano, state.StartedAt)
	if err != nil || !startedAt.IsZero() {
		return errors.New("controller has no unstarted timestamp confirmation")
	}
	_, err = docker.ContainerRemove(ctx, id, mobyclient.ContainerRemoveOptions{})
	return err
}

func executeContainerOperation(ctx context.Context, c Environment, lease Request) Result {
	var intent containerExecutionIntent
	if lease.Protocol != containerExecutionProtocol || strictDecode(lease.Input, &intent) != nil || !validContainerIntent(lease.OperationID, intent) {
		return failedExecution(failureInvalidIntent)
	}
	docker, err := mobyclient.New(mobyclient.WithHost(c.DockerSocket))
	if err != nil {
		return failedExecution(failureRuntimeUnavailable)
	}
	defer func() { _ = docker.Close() }()
	if err := dispatchContainerOperation(ctx, docker, lease.OperationID, intent); err != nil {
		return failedExecutionAt(mapProviderFailure(err), "container_apply")
	}
	return Result{Outcome: "success"}
}

func dispatchContainerOperation(ctx context.Context, docker *mobyclient.Client, operation string, intent containerExecutionIntent) error {
	switch {
	case strings.HasPrefix(operation, "container.lifecycle."):
		return dispatchContainerLifecycle(ctx, docker, operation, intent)
	case strings.HasPrefix(operation, "container.image."):
		return dispatchContainerImage(ctx, docker, operation, intent)
	case strings.HasPrefix(operation, "container.network."):
		return dispatchContainerNetwork(ctx, docker, operation, intent)
	case operation == "container.volume.remove.v1":
		_, err := docker.VolumeRemove(ctx, intent.VolumeRef, mobyclient.VolumeRemoveOptions{Force: intent.Force})
		return err
	default:
		return errUnsupportedProviderOperation
	}
}

func dispatchContainerLifecycle(ctx context.Context, docker *mobyclient.Client, operation string, intent containerExecutionIntent) error {
	switch operation {
	case "container.lifecycle.start.v1":
		_, err := docker.ContainerStart(ctx, intent.ContainerRef, mobyclient.ContainerStartOptions{})
		return err
	case "container.lifecycle.stop.v1":
		_, err := docker.ContainerStop(ctx, intent.ContainerRef, mobyclient.ContainerStopOptions{})
		return err
	case "container.lifecycle.restart.v1":
		_, err := docker.ContainerRestart(ctx, intent.ContainerRef, mobyclient.ContainerRestartOptions{})
		return err
	case "container.lifecycle.remove.v1":
		_, err := docker.ContainerRemove(ctx, intent.ContainerRef, mobyclient.ContainerRemoveOptions{Force: intent.Force})
		return err
	default:
		return errUnsupportedProviderOperation
	}
}

func dispatchContainerImage(ctx context.Context, docker *mobyclient.Client, operation string, intent containerExecutionIntent) error {
	switch operation {
	case "container.image.pull.v1":
		response, err := docker.ImagePull(ctx, intent.ImageRef, mobyclient.ImagePullOptions{})
		if err != nil {
			return err
		}
		return consumeDockerProgress(response)
	case "container.image.tag.v1":
		_, err := docker.ImageTag(ctx, mobyclient.ImageTagOptions{Source: intent.ImageRef, Target: intent.TargetRef})
		return err
	case "container.image.untag.v1":
		return removeBoundImageTag(ctx, docker, intent)

	case "container.image.remove.v1":
		_, err := docker.ImageRemove(ctx, intent.ImageRef, mobyclient.ImageRemoveOptions{Force: intent.Force})
		return err
	default:
		return errUnsupportedProviderOperation
	}
}

// removeBoundImageTag 同时核对来源和待移除标签的真实镜像身份，保留已有别名调用语义。
func removeBoundImageTag(ctx context.Context, docker *mobyclient.Client, intent containerExecutionIntent) error {
	expected, err := docker.ImageInspect(ctx, intent.ImageRef)
	if err != nil {
		return err
	}
	tagged, err := docker.ImageInspect(ctx, intent.TagRef)
	if err != nil {
		return err
	}
	if tagged.ID != expected.ID {
		return errContainerReferenceChanged
	}
	_, err = docker.ImageRemove(ctx, intent.TagRef, mobyclient.ImageRemoveOptions{})
	return err
}

func dispatchContainerNetwork(ctx context.Context, docker *mobyclient.Client, operation string, intent containerExecutionIntent) error {
	switch operation {
	case "container.network.create.v1":
		_, err := docker.NetworkCreate(ctx, intent.Name, networkCreateOptions(intent))
		return err
	case "container.network.remove.v1":
		_, err := docker.NetworkRemove(ctx, intent.NetworkRef, mobyclient.NetworkRemoveOptions{})
		return err
	default:
		return errUnsupportedProviderOperation
	}
}

var errUnsupportedProviderOperation = errors.New("unsupported provider operation")
var errContainerReferenceChanged = errors.New("container image reference changed")

func validContainerIntent(operation string, intent containerExecutionIntent) bool {
	switch {
	case strings.HasPrefix(operation, "container.lifecycle."):
		return validContainerLifecycleIntent(operation, intent)
	case strings.HasPrefix(operation, "container.image."):
		return validContainerImageIntent(operation, intent)
	case strings.HasPrefix(operation, "container.network."):
		return validContainerNetworkIntent(operation, intent)
	case operation == "container.volume.remove.v1":
		return validProviderReference(intent.VolumeRef) && containerNonVolumeFieldsEmpty(intent)
	default:
		return false
	}
}

func validContainerLifecycleIntent(operation string, intent containerExecutionIntent) bool {
	switch operation {
	case "container.lifecycle.start.v1", "container.lifecycle.stop.v1", "container.lifecycle.restart.v1", "container.lifecycle.remove.v1":
		if !validProviderReference(intent.ContainerRef) {
			return false
		}
		intent.ContainerRef = ""
		intent.Force = false
		return intent == (containerExecutionIntent{})
	default:
		return false
	}
}

func validContainerImageIntent(operation string, intent containerExecutionIntent) bool {
	switch operation {
	case "container.image.pull.v1":
		return validIntentReferences(intent, false, false, false)
	case "container.image.tag.v1":
		return validIntentReferences(intent, true, false, false)
	case "container.image.untag.v1":
		return validIntentReferences(intent, false, true, false)
	case "container.image.remove.v1":
		return validIntentReferences(intent, false, false, true)
	default:
		return false
	}
}

func validContainerNetworkIntent(operation string, intent containerExecutionIntent) bool {
	switch operation {
	case "container.network.create.v1":
		if !validNetworkIntent(intent) {
			return false
		}
		intent.Name, intent.Driver, intent.Internal, intent.Attachable, intent.IPAM = "", "", false, false, nil
		return intent == (containerExecutionIntent{})
	case "container.network.remove.v1":
		if !validProviderReference(intent.NetworkRef) {
			return false
		}
		intent.NetworkRef = ""
		return intent == (containerExecutionIntent{})
	default:
		return false
	}
}

func containerNonVolumeFieldsEmpty(intent containerExecutionIntent) bool {
	intent.VolumeRef = ""
	intent.Force = false
	return intent == (containerExecutionIntent{})
}

func validIntentReferences(intent containerExecutionIntent, allowTarget, allowTag, allowForce bool) bool {
	if !validProviderReference(intent.ImageRef) {
		return false
	}
	intent.ImageRef = ""
	if allowTarget {
		if !validProviderReference(intent.TargetRef) {
			return false
		}
		intent.TargetRef = ""
	}
	if allowTag {
		if !validProviderReference(intent.TagRef) {
			return false
		}
		intent.TagRef = ""
	}
	if allowForce {
		intent.Force = false
	}
	return intent == (containerExecutionIntent{})
}

func validNetworkIntent(intent containerExecutionIntent) bool {
	if !validProviderReference(intent.Name) || !validNetworkDriver(intent.Driver) {
		return false
	}
	return intent.IPAM == nil || validNetworkIPAM(*intent.IPAM)
}

func validNetworkDriver(driver string) bool {
	switch driver {
	case "bridge", "overlay", "macvlan", "ipvlan", "none":
		return true
	default:
		return false
	}
}

func validNetworkIPAM(item containerNetworkIPAMIntent) bool {
	subnet, err := netip.ParsePrefix(strings.TrimSpace(item.Subnet))
	if err != nil || !subnet.Addr().Is4() {
		return false
	}
	if strings.TrimSpace(item.Gateway) == "" {
		return true
	}
	gateway, err := netip.ParseAddr(strings.TrimSpace(item.Gateway))
	return err == nil && gateway.Is4() && subnet.Contains(gateway)
}

func networkCreateOptions(intent containerExecutionIntent) mobyclient.NetworkCreateOptions {
	options := mobyclient.NetworkCreateOptions{Driver: intent.Driver, Internal: intent.Internal, Attachable: intent.Attachable}
	if intent.IPAM == nil {
		return options
	}
	item := *intent.IPAM
	subnet, _ := netip.ParsePrefix(strings.TrimSpace(item.Subnet))
	entry := network.IPAMConfig{Subnet: subnet}
	if strings.TrimSpace(item.Gateway) != "" {
		entry.Gateway, _ = netip.ParseAddr(strings.TrimSpace(item.Gateway))
	}
	options.IPAM = &network.IPAM{Config: []network.IPAMConfig{entry}}
	return options
}

func executeApplicationComposeOperation(ctx context.Context, c Environment, lease Request) Result {
	intent, action, valid := decodeApplicationComposeIntent(lease)
	if !valid {
		return failedExecution(failureInvalidIntent)
	}
	resolved, failureCode := resolveApplicationComposeMaterial(ctx, c, lease)
	if failureCode != "" {
		return failedExecutionAt(failureCode, "material_resolve")
	}
	runtime, err := loadComposeProject(ctx, c.DockerSocket, resolved)
	if err != nil {
		return failedExecutionSummary(mapProviderFailure(err), "compose_load", composeLoadFailureSummary(err))
	}
	defer func() { _ = runtime.dockerCLI.Client().Close() }()
	request := composeDispatchRequest{runtime: runtime, managedServices: resolved.ManagedServiceNames, action: action, policy: intent.Policy}
	if err := dispatchComposeOperation(ctx, request); err != nil {
		return failedExecutionAt(mapProviderFailure(err), "compose_apply")
	}
	return Result{Outcome: "success"}
}

func decodeApplicationComposeIntent(lease Request) (applicationComposeIntent, string, bool) {
	var intent applicationComposeIntent
	action, supported := applicationComposeAction(lease.OperationID)
	if lease.Protocol != applicationComposeProtocol || !supported || strictDecode(lease.Input, &intent) != nil || strings.TrimSpace(intent.ApplicationID) == "" || !validApplicationComposePolicy(action, intent.Policy) {
		return applicationComposeIntent{}, "", false
	}
	return intent, action, true
}

func resolveApplicationComposeMaterial(ctx context.Context, c Environment, lease Request) (applicationComposeMaterial, string) {
	material, err := c.ResolveMaterial(ctx, lease)
	if err != nil || material.Protocol != applicationComposeMaterialProtocol {
		return applicationComposeMaterial{}, failureProviderOperation
	}
	var resolved applicationComposeMaterial
	if strictDecode(material.Payload, &resolved) != nil || !validComposeMaterial(resolved) {
		return applicationComposeMaterial{}, failureInvalidIntent
	}
	return resolved, ""
}

type composeRuntime struct {
	project   *composetypes.Project
	service   composeapi.Compose
	dockerCLI *dockercommand.DockerCli
	pruner    dockerImagePruner
}

func loadComposeProject(ctx context.Context, dockerSocket string, material applicationComposeMaterial) (*composeRuntime, error) {
	dockerCLI, err := dockercommand.NewDockerCli(
		dockercommand.WithInputStream(io.NopCloser(bytes.NewReader(nil))),
		dockercommand.WithOutputStream(io.Discard),
		dockercommand.WithErrorStream(io.Discard),
	)
	if err != nil {
		return nil, err
	}
	clientOptions := dockerflags.NewClientOptions()
	clientOptions.Hosts = []string{dockerSocket}
	if err := dockerCLI.Initialize(clientOptions); err != nil {
		return nil, err
	}
	project, err := loadComposeDefinition(ctx, material)
	if err != nil {
		_ = dockerCLI.Client().Close()
		return nil, err
	}
	service, err := compose.NewComposeService(dockerCLI)
	if err != nil {
		_ = dockerCLI.Client().Close()
		return nil, err
	}
	return &composeRuntime{project: project, service: service, dockerCLI: dockerCLI, pruner: dockerCLI.Client()}, nil
}

func loadComposeDefinition(ctx context.Context, material applicationComposeMaterial) (*composetypes.Project, error) {
	projectOptions, err := composecli.NewProjectOptions(material.ComposeFiles,
		composecli.WithWorkingDirectory(material.WorkspacePath),
		composecli.WithEnv(nil),
		composecli.WithEnvFiles(material.EnvFiles...),
		composecli.WithDotEnv,
		composecli.WithName(material.ProjectName),
		composecli.WithProfiles(material.Profiles),
	)
	if err != nil {
		return nil, err
	}
	project, err := projectOptions.LoadProject(ctx)
	if err != nil {
		return nil, err
	}
	PrepareComposeProjectLabels(project, material.EnvFiles)
	return project, nil
}

type composeDispatchRequest struct {
	runtime         *composeRuntime
	managedServices []string
	action          string
	policy          applicationComposePolicy
}

func dispatchComposeOperation(ctx context.Context, request composeDispatchRequest) error {
	runtime := request.runtime
	project := runtime.project
	timeout := composeTimeout(request.policy.WaitTimeoutSeconds)
	services := append([]string(nil), request.managedServices...)
	switch request.action {
	case "up":
		return dispatchComposeUp(ctx, request, services, timeout)
	case "stop":
		return runtime.service.Stop(ctx, project.Name, composeapi.StopOptions{Project: project, Services: services, Timeout: timeout})
	case "restart":
		return runtime.service.Restart(ctx, project.Name, composeapi.RestartOptions{Project: project, Services: services, Timeout: timeout})
	case "down":
		return runtime.service.Down(ctx, project.Name, composeapi.DownOptions{Project: project, RemoveOrphans: request.policy.RemoveOrphans, Volumes: request.policy.RemoveNamedVolumes, Timeout: timeout})
	case "pull":
		return dispatchComposePull(ctx, runtime.service, project, services)
	case "image-prune":
		return dispatchComposeImagePrune(ctx, runtime.pruner)
	default:
		return errUnsupportedProviderOperation
	}
}

func dispatchComposeUp(ctx context.Context, request composeDispatchRequest, services []string, timeout *time.Duration) error {
	create := composeapi.CreateOptions{Services: services, RemoveOrphans: request.policy.RemoveOrphans, Inherit: !request.policy.RenewAnonVolumes, Timeout: timeout}
	if request.policy.ForceRecreate {
		create.Recreate = composeapi.RecreateForce
	}
	if request.policy.BuildBeforeUp {
		create.Build = &composeapi.BuildOptions{Progress: "quiet", Quiet: true}
	}
	start := composeapi.StartOptions{Project: request.runtime.project, Services: services, Wait: request.policy.WaitAfterUp}
	if request.policy.WaitAfterUp && timeout != nil {
		start.WaitTimeout = *timeout
	}
	return request.runtime.service.Up(ctx, request.runtime.project, composeapi.UpOptions{Create: create, Start: start})
}

func dispatchComposePull(ctx context.Context, service composeapi.Compose, project *composetypes.Project, services []string) error {
	selected := project
	if len(services) > 0 {
		var err error
		selected, err = project.WithSelectedServices(services, composetypes.IgnoreDependencies)
		if err != nil {
			return err
		}
	}
	return service.Pull(ctx, selected, composeapi.PullOptions{Quiet: true})
}

func dispatchComposeImagePrune(ctx context.Context, pruner dockerImagePruner) error {
	if pruner == nil {
		return errUnsupportedProviderOperation
	}
	_, err := pruner.ImagePrune(ctx, mobyclient.ImagePruneOptions{Filters: make(mobyclient.Filters).Add("dangling", "true")})
	return err
}

type dockerImagePruner interface {
	ImagePrune(context.Context, mobyclient.ImagePruneOptions) (mobyclient.ImagePruneResult, error)
}

func composeTimeout(seconds int) *time.Duration {
	if seconds <= 0 {
		return nil
	}
	value := time.Duration(seconds) * time.Second
	return &value
}

func applicationComposeAction(operationID string) (string, bool) {
	switch operationID {
	case "application.compose.up.v1":
		return "up", true
	case "application.compose.stop.v1":
		return "stop", true
	case "application.compose.restart.v1":
		return "restart", true
	case "application.compose.down.v1":
		return "down", true
	case "application.compose.pull.v1":
		return "pull", true
	case "application.compose.image-prune.v1":
		return "image-prune", true
	default:
		return "", false
	}
}

func validApplicationComposePolicy(action string, policy applicationComposePolicy) bool {
	if strings.TrimSpace(policy.SnapshotDigest) == "" {
		return false
	}
	switch action {
	case "up":
		return validApplicationUpPolicy(policy)
	case "down":
		return applicationUpFieldsEmpty(policy)
	case "stop", "restart", "pull", "image-prune":
		return applicationUpFieldsEmpty(policy) && !policy.RemoveNamedVolumes
	default:
		return false
	}
}

func validApplicationUpPolicy(policy applicationComposePolicy) bool {
	if policy.RemoveNamedVolumes {
		return false
	}
	if policy.WaitAfterUp {
		return policy.WaitTimeoutSeconds >= 1 && policy.WaitTimeoutSeconds <= 3600
	}
	return policy.WaitTimeoutSeconds == 0
}

func applicationUpFieldsEmpty(policy applicationComposePolicy) bool {
	return !policy.BuildBeforeUp && !policy.ForceRecreate && !policy.RemoveOrphans && !policy.WaitAfterUp && policy.WaitTimeoutSeconds == 0 && !policy.RenewAnonVolumes
}

func validComposeMaterial(material applicationComposeMaterial) bool {
	if strings.TrimSpace(material.WorkspacePath) == "" || !validProviderReference(material.ProjectName) || len(material.ComposeFiles) == 0 {
		return false
	}
	workspace, err := filepath.EvalSymlinks(filepath.Clean(material.WorkspacePath))
	if err != nil || !filepath.IsAbs(workspace) {
		return false
	}
	for _, name := range append(append([]string(nil), material.ComposeFiles...), material.EnvFiles...) {
		if !pathWithinWorkspace(workspace, name) {
			return false
		}
	}
	return true
}

func pathWithinWorkspace(workspace, name string) bool {
	resolved, err := filepath.EvalSymlinks(filepath.Clean(name))
	if err != nil || !filepath.IsAbs(resolved) {
		return false
	}
	relative, err := filepath.Rel(workspace, resolved)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func validProviderReference(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= maxProviderReferenceLength && strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) < 0
}

func strictDecode(payload []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	err := decoder.Decode(&struct{}{})
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func failedExecution(code string) Result {
	return Result{Outcome: "failed", FailureCode: code, FailureClass: providerFailureClass(code)}
}

// providerFailureClass 把稳定 Provider 失败码映射为跨模块失败分类；未知码保守归类为 unknown。
func providerFailureClass(code string) moduleapi.ExecutionFailureClass {
	switch code {
	case failureInvalidIntent, failureUnsupportedAction, failureResourceNotFound:
		return moduleapi.ExecutionFailureClassPermanent
	case failureRuntimeUnavailable, failureResourceConflict:
		return moduleapi.ExecutionFailureClassInfrastructure
	case failureProviderOperation:
		return moduleapi.ExecutionFailureClassProvider
	case failureInterrupted:
		return moduleapi.ExecutionFailureClassUnknown
	default:
		return moduleapi.ExecutionFailureClassUnknown
	}
}

// composeLoadFailureSummary 从 Compose 加载错误文本按允许列表提取单行脱敏摘要。
// loadComposeProject 的输出流保持 io.Discard；这里只消费返回的错误文本，
// 剔除路径、疑似凭据片段与控制字符，并把结果限制在稳定摘要上限内。
func composeLoadFailureSummary(err error) string {
	if err == nil {
		return ""
	}
	var selected []string
	for _, line := range strings.Split(err.Error(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !composeLoadFailureLineAllowed(line) {
			continue
		}
		if selectedLine, ok := redactComposeFailureLine(line); ok {
			selected = append(selected, selectedLine)
		}
		if len(selected) == composeLoadFailureSummaryMaxLines {
			break
		}
	}
	return moduleapi.SanitizeExecutionDiagnosticSummary(strings.Join(selected, "; "))
}

// composeLoadFailureLineAllowed 只放行可安全归因于结构化 Compose 校验错误的行；
// 不匹配允许列表的文本一律丢弃，避免把任意 Provider 输出带进 Task 持久化。
func composeLoadFailureLineAllowed(line string) bool {
	for _, pattern := range composeLoadFailureLineAllowlist {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

// redactComposeFailureLine 移除路径形态的片段并在疑似凭据文本出现时整行丢弃。
func redactComposeFailureLine(line string) (string, bool) {
	if sensitiveFragmentPattern.MatchString(line) {
		return "", false
	}
	redacted := absolutePathPattern.ReplaceAllString(line, "[path]")
	return redacted, strings.TrimSpace(redacted) != ""
}

var composeLoadFailureLineAllowlist = []*regexp.Regexp{
	regexp.MustCompile(`^yaml:\s`),
	regexp.MustCompile(`\bservices?\.[A-Za-z0-9_.-]+\b`),
	regexp.MustCompile(`\bvolumes?\.[A-Za-z0-9_.-]+\b`),
	regexp.MustCompile(`\bunknown\b`),
	regexp.MustCompile(`\binvalid\b`),
	regexp.MustCompile(`\bunsupported\b`),
	regexp.MustCompile(`\bmust be\b`),
	regexp.MustCompile(`\bnot allowed\b`),
	regexp.MustCompile(`\brequired\b`),
	regexp.MustCompile(`\bcannot\b`),
}

var absolutePathPattern = regexp.MustCompile(`(?:[A-Za-z]:)?(?:/|\\)[^\s"'']*`)
var sensitiveFragmentPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|credential|authorization|api[_-]?key)`)

func mapProviderFailure(err error) string {
	if errors.Is(err, errContainerReferenceChanged) {
		return failureResourceConflict
	}
	if errors.Is(err, errUnsupportedProviderOperation) {
		return failureUnsupportedAction
	}
	if containerderrdefs.IsNotFound(err) {
		return failureResourceNotFound
	}
	if mobyclient.IsErrConnectionFailed(err) || errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
		return failureRuntimeUnavailable
	}
	if containerderrdefs.IsConflict(err) || containerderrdefs.IsPermissionDenied(err) {
		return failureResourceConflict
	}
	return failureProviderOperation
}

// PrepareComposeProjectLabels 为直接调用 Compose SDK 的执行器准备受控资源归属标签。
func PrepareComposeProjectLabels(project *composetypes.Project, envFiles []string) {
	// Compose 的服务归属标签由 CLI 加载阶段准备，直接调用 SDK 时必须在受控加载边界补齐。
	// CustomLabels 在 SDK 合并标签时优先于用户标签，避免归属和 one-off 标识被应用配置覆盖。
	for name, service := range project.Services {
		service.CustomLabels = composetypes.Labels{
			composeapi.ProjectLabel:     project.Name,
			composeapi.ServiceLabel:     name,
			composeapi.VersionLabel:     composeapi.ComposeVersion,
			composeapi.WorkingDirLabel:  project.WorkingDir,
			composeapi.ConfigFilesLabel: strings.Join(project.ComposeFiles, ","),
			composeapi.OneoffLabel:      "False",
		}
		if len(envFiles) > 0 {
			service.CustomLabels[composeapi.EnvironmentFileLabel] = strings.Join(envFiles, ",")
		}
		project.Services[name] = service
	}
}

func failedExecutionAt(code, stage string) Result {
	result := failedExecution(code)
	result.FailureStage = stage
	return result
}

// failedExecutionSummary 在阶段归类之上携带一条经允许列表脱敏的诊断摘要；
// 摘要只服务于操作者观察，不参与失败码结算语义。
func failedExecutionSummary(code, stage, summary string) Result {
	result := failedExecutionAt(code, stage)
	result.FailureSummary = summary
	return result
}
