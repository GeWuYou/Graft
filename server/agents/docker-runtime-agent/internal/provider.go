package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	composecli "github.com/compose-spec/compose-go/v2/cli"
	composetypes "github.com/compose-spec/compose-go/v2/types"
	containerderrdefs "github.com/containerd/errdefs"
	dockercommand "github.com/docker/cli/cli/command"
	dockerflags "github.com/docker/cli/cli/flags"
	composeapi "github.com/docker/compose/v2/pkg/api"
	compose "github.com/docker/compose/v2/pkg/compose"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/moby/moby/api/types/network"
	mobyclient "github.com/moby/moby/client"
)

const (
	applicationComposeProtocol         = "application-compose/v1"
	applicationComposeMaterialProtocol = "application-compose-material/v1"
	containerExecutionProtocol         = "container-execution/v1"
	maxProviderReferenceLength         = 512
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

func executeProviderOperation(ctx context.Context, client *http.Client, c config, lease executionLease) executionResult {
	switch lease.Capability {
	case buildExecutionCapability:
		return executeBuildOperation(ctx, client, c, lease)
	case "container_execution":
		return executeContainerOperation(ctx, c, lease)
	case "compose_execution":
		return executeApplicationComposeOperation(ctx, client, c, lease)
	default:
		return failedExecution(failureUnsupportedAction)
	}
}

func executeContainerOperation(ctx context.Context, c config, lease executionLease) executionResult {
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
		return failedExecution(mapProviderFailure(err))
	}
	return executionResult{Outcome: "success"}
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
		_, copyErr := io.Copy(io.Discard, response)
		return errors.Join(copyErr, response.Close())
	case "container.image.tag.v1":
		_, err := docker.ImageTag(ctx, mobyclient.ImageTagOptions{Source: intent.ImageRef, Target: intent.TargetRef})
		return err
	case "container.image.untag.v1":
		_, err := docker.ImageRemove(ctx, intent.TagRef, mobyclient.ImageRemoveOptions{})
		return err
	case "container.image.remove.v1":
		_, err := docker.ImageRemove(ctx, intent.ImageRef, mobyclient.ImageRemoveOptions{Force: intent.Force})
		return err
	default:
		return errUnsupportedProviderOperation
	}
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

func executeApplicationComposeOperation(ctx context.Context, transport *http.Client, c config, lease executionLease) executionResult {
	intent, action, valid := decodeApplicationComposeIntent(lease)
	if !valid {
		return failedExecution(failureInvalidIntent)
	}
	resolved, failureCode := resolveApplicationComposeMaterial(ctx, transport, c, lease)
	if failureCode != "" {
		return failedExecution(failureCode)
	}
	runtime, err := loadComposeProject(ctx, c.DockerSocket, resolved)
	if err != nil {
		return failedExecution(mapProviderFailure(err))
	}
	defer func() { _ = runtime.dockerCLI.Client().Close() }()
	request := composeDispatchRequest{runtime: runtime, managedServices: resolved.ManagedServiceNames, action: action, policy: intent.Policy}
	if err := dispatchComposeOperation(ctx, request); err != nil {
		return failedExecution(mapProviderFailure(err))
	}
	return executionResult{Outcome: "success"}
}

func decodeApplicationComposeIntent(lease executionLease) (applicationComposeIntent, string, bool) {
	var intent applicationComposeIntent
	action, supported := applicationComposeAction(lease.OperationID)
	if lease.Protocol != applicationComposeProtocol || !supported || strictDecode(lease.Input, &intent) != nil || strings.TrimSpace(intent.ApplicationID) == "" || !validApplicationComposePolicy(action, intent.Policy) {
		return applicationComposeIntent{}, "", false
	}
	return intent, action, true
}

func resolveApplicationComposeMaterial(ctx context.Context, transport *http.Client, c config, lease executionLease) (applicationComposeMaterial, string) {
	material, err := resolveExecutionMaterial(ctx, transport, c.AgentURL, lease)
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
	return &composeRuntime{project: project, service: compose.NewComposeService(dockerCLI), dockerCLI: dockerCLI, pruner: dockerCLI.Client()}, nil
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
	return projectOptions.LoadProject(ctx)
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
	_, err := pruner.ImagesPrune(ctx, dockerfilters.NewArgs(dockerfilters.Arg("dangling", "true")))
	return err
}

type dockerImagePruner interface {
	ImagesPrune(context.Context, dockerfilters.Args) (dockerimage.PruneReport, error)
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

func failedExecution(code string) executionResult {
	return executionResult{Outcome: "failed", FailureCode: code}
}

func mapProviderFailure(err error) string {
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
