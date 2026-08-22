package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	containerdplatforms "github.com/containerd/platforms"
	distributionref "github.com/distribution/reference"
	dockerregistry "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/go-archive"
	mobyclient "github.com/moby/moby/client"
	"github.com/moby/patternmatcher/ignorefile"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	buildExecutionCapability       = "oci-build"
	buildExecutionProtocol         = "build-execution/v1"
	buildExecutionMaterialProtocol = "build-execution-material/v1"
	buildExecutionResultProtocol   = "build-execution-result/v1"

	buildImageLocalOperation   = "build.image.local.v1"
	buildImagePublishOperation = "build.image.publish.v1"
	buildManifestOperation     = "build.manifest.publish.v1"
	buildArtifactCopyOperation = "build.artifact.copy.v1"
)

type buildExecutionMaterial struct {
	Context           *buildContextMaterial           `json:"context,omitempty"`
	Destination       *buildRegistryMaterial          `json:"destination,omitempty"`
	Source            *buildRegistrySourceMaterial    `json:"source,omitempty"`
	PlatformArtifacts []buildPlatformArtifactMaterial `json:"platform_artifacts,omitempty"`
}

type buildContextMaterial struct {
	Root           string          `json:"root"`
	ContextPath    string          `json:"context_path"`
	DockerfilePath string          `json:"dockerfile_path"`
	Repository     string          `json:"repository"`
	Reference      string          `json:"reference"`
	Platform       string          `json:"platform,omitempty"`
	BuildArgs      []buildArgument `json:"build_args,omitempty"`
}

type buildArgument struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type buildRegistryMaterial struct {
	Endpoint   string `json:"endpoint"`
	Repository string `json:"repository"`
	Reference  string `json:"reference"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

type buildRegistrySourceMaterial struct {
	Endpoint   string `json:"endpoint"`
	Repository string `json:"repository"`
	Digest     string `json:"digest"`
	MediaType  string `json:"media_type"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
}

type buildPlatformArtifactMaterial struct {
	Platform  string `json:"platform"`
	Digest    string `json:"digest"`
	MediaType string `json:"media_type"`
	SizeBytes int64  `json:"size_bytes"`
}

type buildExecutionResultPayload struct {
	ImageID      string `json:"image_id,omitempty"`
	Digest       string `json:"digest,omitempty"`
	Repository   string `json:"repository"`
	Reference    string `json:"reference"`
	MediaType    string `json:"media_type,omitempty"`
	SizeBytes    int64  `json:"size_bytes"`
	OS           string `json:"os,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Variant      string `json:"variant,omitempty"`
}

type buildPaths struct {
	contextRoot         string
	dockerfileInContext string
}

func executeBuildOperation(ctx context.Context, transport *http.Client, c config, lease executionLease) executionResult {
	if lease.Protocol != buildExecutionProtocol || !validBuildLeaseInput(lease.Input) {
		return failedExecution(failureInvalidIntent)
	}
	material, err := resolveExecutionMaterial(ctx, transport, c.AgentURL, lease)
	if err != nil || material.Protocol != buildExecutionMaterialProtocol {
		return failedExecution(failureProviderOperation)
	}
	var resolved buildExecutionMaterial
	if strictDecode(material.Payload, &resolved) != nil || !validBuildMaterial(lease.OperationID, resolved) {
		return failedExecution(failureInvalidIntent)
	}
	payload, failureCode := dispatchBuildOperation(ctx, c, lease.OperationID, resolved)
	if failureCode != "" {
		return failedExecution(failureCode)
	}
	if !validBuildResult(lease.OperationID, payload) {
		return failedExecution(failureProviderOperation)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return failedExecution(failureProviderOperation)
	}
	return executionResult{Outcome: "success", Protocol: buildExecutionResultProtocol, Payload: encoded}
}

func dispatchBuildOperation(ctx context.Context, c config, operation string, material buildExecutionMaterial) (buildExecutionResultPayload, string) {
	switch operation {
	case buildImageLocalOperation, buildImagePublishOperation:
		return executeBuildImage(ctx, c, operation, material)
	case buildManifestOperation:
		result, err := publishBuildManifest(ctx, *material.Destination, material.PlatformArtifacts)
		if err != nil {
			return buildExecutionResultPayload{}, mapProviderFailure(err)
		}
		return result, ""
	case buildArtifactCopyOperation:
		result, err := copyBuildArtifact(ctx, *material.Source, *material.Destination)
		if err != nil {
			return buildExecutionResultPayload{}, mapProviderFailure(err)
		}
		return result, ""
	default:
		return buildExecutionResultPayload{}, failureUnsupportedAction
	}
}

//nolint:gocognit,gocyclo,cyclop,funlen // Build SDK orchestration keeps build, inspect, publish and digest verification in one fenced operation.
func executeBuildImage(ctx context.Context, c config, operation string, material buildExecutionMaterial) (buildExecutionResultPayload, string) {
	paths, err := normalizeBuildPaths(*material.Context)
	if err != nil {
		return buildExecutionResultPayload{}, failureInvalidIntent
	}
	buildContext, err := buildContextArchive(paths)
	if err != nil {
		return buildExecutionResultPayload{}, failureProviderOperation
	}
	defer func() { _ = buildContext.Close() }()
	docker, err := mobyclient.New(mobyclient.WithHost(c.DockerSocket))
	if err != nil {
		return buildExecutionResultPayload{}, failureRuntimeUnavailable
	}
	defer func() { _ = docker.Close() }()
	localReference, err := taggedImageReference(material.Context.Repository, material.Context.Reference)
	if err != nil {
		return buildExecutionResultPayload{}, failureInvalidIntent
	}
	options := mobyclient.ImageBuildOptions{
		Tags:       []string{localReference},
		Dockerfile: filepath.ToSlash(paths.dockerfileInContext),
		BuildArgs:  buildArgumentMap(material.Context.BuildArgs),
		Remove:     true,
	}
	if strings.TrimSpace(material.Context.Platform) != "" {
		platform, parseErr := containerdplatforms.Parse(material.Context.Platform)
		if parseErr != nil {
			return buildExecutionResultPayload{}, failureInvalidIntent
		}
		options.Platforms = []ocispec.Platform{containerdplatforms.Normalize(platform)}
	}
	response, err := docker.ImageBuild(ctx, buildContext, options)
	if err != nil {
		return buildExecutionResultPayload{}, mapProviderFailure(err)
	}
	if err := consumeBuildResponse(response.Body); err != nil {
		return buildExecutionResultPayload{}, failureProviderOperation
	}
	inspect, err := docker.ImageInspect(ctx, localReference)
	if err != nil {
		return buildExecutionResultPayload{}, mapProviderFailure(err)
	}
	result := buildResultFromInspect(inspect, *material.Context)
	if operation == buildImageLocalOperation {
		return result, ""
	}
	remoteReference, _, err := registryReference(*material.Destination)
	if err != nil {
		return buildExecutionResultPayload{}, failureInvalidIntent
	}
	if _, err := docker.ImageTag(ctx, mobyclient.ImageTagOptions{Source: localReference, Target: remoteReference}); err != nil {
		return buildExecutionResultPayload{}, mapProviderFailure(err)
	}
	registryAuth, err := dockerregistry.EncodeAuthConfig(dockerregistry.AuthConfig{Username: material.Destination.Username, Password: material.Destination.Password, ServerAddress: material.Destination.Endpoint})
	if err != nil {
		return buildExecutionResultPayload{}, failureProviderOperation
	}
	push, err := docker.ImagePush(ctx, remoteReference, mobyclient.ImagePushOptions{RegistryAuth: registryAuth})
	if err != nil {
		return buildExecutionResultPayload{}, mapProviderFailure(err)
	}
	if err := push.Wait(ctx); err != nil {
		_ = push.Close()
		return buildExecutionResultPayload{}, failureProviderOperation
	}
	if err := push.Close(); err != nil {
		return buildExecutionResultPayload{}, failureProviderOperation
	}
	repository, err := newBuildRepository(material.Destination.Endpoint, material.Destination.Repository, material.Destination.Username, material.Destination.Password)
	if err != nil {
		return buildExecutionResultPayload{}, failureInvalidIntent
	}
	descriptor, err := repository.Resolve(ctx, material.Destination.Reference)
	if err != nil {
		return buildExecutionResultPayload{}, mapProviderFailure(err)
	}
	result.Repository = material.Destination.Repository
	result.Reference = material.Destination.Reference
	result.Digest = descriptor.Digest.String()
	result.MediaType = descriptor.MediaType
	result.SizeBytes = descriptor.Size
	return result, ""
}

func consumeBuildResponse(body io.ReadCloser) error {
	if body == nil {
		return errors.New("provider build response is unavailable")
	}
	defer func() { _ = body.Close() }()
	decoder := json.NewDecoder(body)
	for {
		var message jsonmessage.JSONMessage
		if err := decoder.Decode(&message); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return errors.New("decode provider build response")
		}
		if message.Error != nil || (message.Error != nil && message.Error.Message != "") {
			return errors.New("provider build failed")
		}
	}
}

func buildResultFromInspect(inspect mobyclient.ImageInspectResult, material buildContextMaterial) buildExecutionResultPayload {
	result := buildExecutionResultPayload{
		ImageID:      inspect.ID,
		Repository:   material.Repository,
		Reference:    material.Reference,
		SizeBytes:    inspect.Size,
		OS:           inspect.Os,
		Architecture: inspect.Architecture,
		Variant:      inspect.Variant,
	}
	if inspect.Descriptor != nil {
		result.Digest = inspect.Descriptor.Digest.String()
		result.MediaType = inspect.Descriptor.MediaType
	}
	return result
}

func publishBuildManifest(ctx context.Context, destination buildRegistryMaterial, artifacts []buildPlatformArtifactMaterial) (buildExecutionResultPayload, error) {
	repository, err := newBuildRepository(destination.Endpoint, destination.Repository, destination.Username, destination.Password)
	if err != nil {
		return buildExecutionResultPayload{}, err
	}
	payload, descriptor, err := buildManifestDocument(artifacts)
	if err != nil {
		return buildExecutionResultPayload{}, err
	}
	if err := repository.PushReference(ctx, descriptor, bytes.NewReader(payload), destination.Reference); err != nil {
		return buildExecutionResultPayload{}, err
	}
	return buildExecutionResultPayload{Digest: descriptor.Digest.String(), Repository: destination.Repository, Reference: destination.Reference, MediaType: descriptor.MediaType, SizeBytes: descriptor.Size}, nil
}

func buildManifestDocument(artifacts []buildPlatformArtifactMaterial) ([]byte, ocispec.Descriptor, error) {
	descriptors := make([]ocispec.Descriptor, 0, len(artifacts))
	for _, artifact := range artifacts {
		parsed, err := digest.Parse(artifact.Digest)
		if err != nil {
			return nil, ocispec.Descriptor{}, errors.New("invalid platform artifact digest")
		}
		platform, err := containerdplatforms.Parse(artifact.Platform)
		if err != nil {
			return nil, ocispec.Descriptor{}, errors.New("invalid platform artifact platform")
		}
		normalized := containerdplatforms.Normalize(platform)
		descriptors = append(descriptors, ocispec.Descriptor{Digest: parsed, MediaType: artifact.MediaType, Size: artifact.SizeBytes, Platform: &normalized})
	}
	const ociSchemaVersion = 2
	index := ocispec.Index{Versioned: specs.Versioned{SchemaVersion: ociSchemaVersion}, MediaType: ocispec.MediaTypeImageIndex, Manifests: descriptors}
	payload, err := json.Marshal(index)
	if err != nil {
		return nil, ocispec.Descriptor{}, errors.New("encode OCI image index")
	}
	descriptor := ocispec.Descriptor{Digest: digest.FromBytes(payload), MediaType: ocispec.MediaTypeImageIndex, Size: int64(len(payload))}
	return payload, descriptor, nil
}

func copyBuildArtifact(ctx context.Context, source buildRegistrySourceMaterial, destination buildRegistryMaterial) (buildExecutionResultPayload, error) {
	sourceRepository, err := newBuildRepository(source.Endpoint, source.Repository, source.Username, source.Password)
	if err != nil {
		return buildExecutionResultPayload{}, err
	}
	destinationRepository, err := newBuildRepository(destination.Endpoint, destination.Repository, destination.Username, destination.Password)
	if err != nil {
		return buildExecutionResultPayload{}, err
	}
	expected, err := digest.Parse(source.Digest)
	if err != nil {
		return buildExecutionResultPayload{}, errors.New("invalid source digest")
	}
	resolved, err := sourceRepository.Resolve(ctx, source.Digest)
	if err != nil || resolved.Digest != expected || resolved.MediaType != source.MediaType {
		return buildExecutionResultPayload{}, errors.New("source artifact identity mismatch")
	}
	descriptor, err := oras.Copy(ctx, sourceRepository, source.Digest, destinationRepository, destination.Reference, oras.DefaultCopyOptions)
	if err != nil {
		return buildExecutionResultPayload{}, err
	}
	if descriptor.Digest != expected {
		return buildExecutionResultPayload{}, errors.New("copied artifact digest mismatch")
	}
	return buildExecutionResultPayload{Digest: descriptor.Digest.String(), Repository: destination.Repository, Reference: destination.Reference, MediaType: descriptor.MediaType, SizeBytes: descriptor.Size}, nil
}

func newBuildRepository(endpoint, repository, username, password string) (*remote.Repository, error) {
	base, plainHTTP, err := registryBase(endpoint)
	if err != nil || !validRegistryRepository(repository) {
		return nil, errors.New("registry material is invalid")
	}
	remoteRepository, err := remote.NewRepository(base + "/" + strings.Trim(repository, "/"))
	if err != nil {
		return nil, errors.New("registry material is invalid")
	}
	remoteRepository.PlainHTTP = plainHTTP
	remoteRepository.Client = &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.NewSingleContextCache(),
		Credential: auth.StaticCredential(remoteRepository.Reference.Registry, auth.Credential{
			Username: username,
			Password: password,
		}),
	}
	return remoteRepository, nil
}

func registryBase(endpoint string) (string, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || strings.ContainsAny(endpoint, "\x00\r\n") {
		return "", false, errors.New("registry endpoint is invalid")
	}
	base := parsed.Host
	if path := strings.Trim(parsed.EscapedPath(), "/"); path != "" {
		base += "/" + path
	}
	return base, parsed.Scheme == "http", nil
}

func registryReference(material buildRegistryMaterial) (string, string, error) {
	base, _, err := registryBase(material.Endpoint)
	if err != nil || !validRegistryRepository(material.Repository) || !validImageReference(material.Reference) {
		return "", "", errors.New("registry reference is invalid")
	}
	repository := strings.Trim(material.Repository, "/")
	full := base + "/" + repository + ":" + material.Reference
	if _, err := distributionref.ParseNormalizedNamed(full); err != nil {
		return "", "", errors.New("registry reference is invalid")
	}
	return full, repository, nil
}

func taggedImageReference(repository, reference string) (string, error) {
	if !validRegistryRepository(repository) || !validImageReference(reference) {
		return "", errors.New("image reference is invalid")
	}
	value := strings.Trim(repository, "/") + ":" + reference
	if _, err := distributionref.ParseNormalizedNamed(value); err != nil {
		return "", errors.New("image reference is invalid")
	}
	return value, nil
}

func validBuildLeaseInput(input json.RawMessage) bool {
	var object map[string]json.RawMessage
	return strictDecode(input, &object) == nil && len(object) > 0
}

//nolint:gocyclo,cyclop // Operation-specific shape checks are intentionally strict and co-located at the provider boundary.
func validBuildMaterial(operation string, material buildExecutionMaterial) bool {
	switch operation {
	case buildImageLocalOperation:
		return validBuildContext(material.Context) && material.Destination == nil && material.Source == nil && material.PlatformArtifacts == nil
	case buildImagePublishOperation:
		return validBuildContext(material.Context) && validBuildDestination(material.Destination) && material.Source == nil && material.PlatformArtifacts == nil
	case buildManifestOperation:
		return material.Context == nil && validBuildDestination(material.Destination) && material.Source == nil && validPlatformArtifacts(material.PlatformArtifacts)
	case buildArtifactCopyOperation:
		return material.Context == nil && validBuildDestination(material.Destination) && validBuildSource(material.Source) && material.PlatformArtifacts == nil
	default:
		return false
	}
}

//nolint:cyclop // Fenced context validation must reject every unsafe path and argument shape before SDK access.
func validBuildContext(material *buildContextMaterial) bool {
	if material == nil || strings.TrimSpace(material.Root) == "" || !validRegistryRepository(material.Repository) || !validImageReference(material.Reference) {
		return false
	}
	if _, err := normalizeBuildPaths(*material); err != nil {
		return false
	}
	if strings.TrimSpace(material.Platform) != "" {
		if _, err := containerdplatforms.Parse(material.Platform); err != nil {
			return false
		}
	}
	seen := make(map[string]struct{}, len(material.BuildArgs))
	for _, argument := range material.BuildArgs {
		name := strings.TrimSpace(argument.Name)
		if name == "" || strings.ContainsAny(argument.Name+argument.Value, "\x00\r\n") {
			return false
		}
		if _, exists := seen[name]; exists {
			return false
		}
		seen[name] = struct{}{}
	}
	return true
}

func validBuildDestination(material *buildRegistryMaterial) bool {
	if material == nil || strings.ContainsAny(material.Username+material.Password, "\x00\r\n") {
		return false
	}
	_, _, err := registryReference(*material)
	return err == nil
}

func validBuildSource(material *buildRegistrySourceMaterial) bool {
	if material == nil || strings.TrimSpace(material.MediaType) == "" || strings.ContainsAny(material.Username+material.Password+material.MediaType, "\x00\r\n") || !validRegistryRepository(material.Repository) {
		return false
	}
	if _, _, err := registryBase(material.Endpoint); err != nil {
		return false
	}
	_, err := digest.Parse(material.Digest)
	return err == nil
}

func validPlatformArtifacts(artifacts []buildPlatformArtifactMaterial) bool {
	if len(artifacts) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		if artifact.SizeBytes <= 0 || strings.TrimSpace(artifact.MediaType) == "" || strings.ContainsAny(artifact.MediaType, "\x00\r\n") {
			return false
		}
		if _, err := containerdplatforms.Parse(artifact.Platform); err != nil {
			return false
		}
		parsed, err := digest.Parse(artifact.Digest)
		if err != nil {
			return false
		}
		if _, exists := seen[parsed.String()]; exists {
			return false
		}
		seen[parsed.String()] = struct{}{}
	}
	return true
}

//nolint:gocyclo,cyclop // Result validation is operation-specific to prevent provider facts leaking across stages.
func validBuildResult(operation string, result buildExecutionResultPayload) bool {
	if !validRegistryRepository(result.Repository) || !validImageReference(result.Reference) || result.SizeBytes < 0 || strings.ContainsAny(result.MediaType+result.OS+result.Architecture+result.Variant+result.ImageID, "\x00\r\n") {
		return false
	}
	digestValid := false
	if result.Digest != "" {
		_, err := digest.Parse(result.Digest)
		digestValid = err == nil
	}
	switch operation {
	case buildImageLocalOperation:
		return result.ImageID != "" && result.OS != "" && result.Architecture != "" && (result.Digest == "" || digestValid)
	case buildImagePublishOperation:
		return result.ImageID != "" && result.OS != "" && result.Architecture != "" && digestValid && result.MediaType != "" && result.SizeBytes > 0
	case buildManifestOperation, buildArtifactCopyOperation:
		return result.ImageID == "" && result.OS == "" && result.Architecture == "" && result.Variant == "" && digestValid && result.MediaType != "" && result.SizeBytes > 0
	default:
		return false
	}
}

func validRegistryRepository(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.ContainsAny(value, "\x00\r\n@") {
		return false
	}
	_, err := distributionref.ParseNormalizedNamed(value + ":validation")
	return err == nil
}

func validImageReference(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 128 && !strings.ContainsAny(value, "\x00\r\n/@:")
}

//nolint:cyclop // Symlink, root containment and Dockerfile-in-context checks form one security gate.
func normalizeBuildPaths(material buildContextMaterial) (buildPaths, error) {
	root, err := filepath.EvalSymlinks(filepath.Clean(strings.TrimSpace(material.Root)))
	if err != nil || !filepath.IsAbs(root) {
		return buildPaths{}, errors.New("build root is invalid")
	}
	contextPath, err := safeBuildRelativePath(material.ContextPath, true)
	if err != nil {
		return buildPaths{}, err
	}
	dockerfilePath, err := safeBuildRelativePath(material.DockerfilePath, false)
	if err != nil {
		return buildPaths{}, err
	}
	contextRoot, err := filepath.EvalSymlinks(filepath.Join(root, contextPath))
	if err != nil || !pathWithinWorkspace(root, contextRoot) {
		return buildPaths{}, errors.New("build context is invalid")
	}
	dockerfile, err := filepath.EvalSymlinks(filepath.Join(root, dockerfilePath))
	if err != nil || !pathWithinWorkspace(root, dockerfile) {
		return buildPaths{}, errors.New("build Dockerfile is invalid")
	}
	info, err := os.Stat(dockerfile)
	if err != nil || !info.Mode().IsRegular() {
		return buildPaths{}, errors.New("build Dockerfile is invalid")
	}
	relativeDockerfile, err := filepath.Rel(contextRoot, dockerfile)
	if err != nil || relativeDockerfile == ".." || strings.HasPrefix(relativeDockerfile, ".."+string(os.PathSeparator)) || filepath.IsAbs(relativeDockerfile) {
		return buildPaths{}, errors.New("build Dockerfile is outside context")
	}
	return buildPaths{contextRoot: contextRoot, dockerfileInContext: relativeDockerfile}, nil
}

func safeBuildRelativePath(value string, allowDot bool) (string, error) {
	value = filepath.Clean(strings.TrimSpace(value))
	if value == "" || (!allowDot && value == ".") || filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, ".."+string(os.PathSeparator)) || strings.ContainsAny(value, "\x00\r\n") {
		return "", errors.New("build path is invalid")
	}
	return value, nil
}

func buildContextArchive(paths buildPaths) (io.ReadCloser, error) {
	excludes, err := dockerIgnorePatterns(paths.contextRoot)
	if err != nil {
		return nil, err
	}
	dockerfile := filepath.ToSlash(paths.dockerfileInContext)
	excludes = append(excludes, "!"+dockerfile, "!.dockerignore")
	return archive.TarWithOptions(paths.contextRoot, &archive.TarOptions{ExcludePatterns: excludes})
}

func dockerIgnorePatterns(contextRoot string) ([]string, error) {
	file, err := os.Open(filepath.Join(contextRoot, ".dockerignore")) // #nosec G304 -- contextRoot 已通过 fence-bound material 的根目录约束。
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("read Docker ignore policy")
	}
	defer func() { _ = file.Close() }()
	patterns, err := ignorefile.ReadAll(file)
	if err != nil {
		return nil, errors.New("decode Docker ignore policy")
	}
	return patterns, nil
}

func buildArgumentMap(arguments []buildArgument) map[string]*string {
	result := make(map[string]*string, len(arguments))
	for _, argument := range arguments {
		value := argument.Value
		result[strings.TrimSpace(argument.Name)] = &value
	}
	return result
}
