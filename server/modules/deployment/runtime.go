package deployment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"graft/server/internal/moduleapi"
)

const (
	deploymentRuntimeEnv     = "GRAFT_DEPLOYMENT_RUNTIME"
	deploymentComposeRootEnv = "GRAFT_DEPLOYMENT_COMPOSE_ROOT"
	composeProjectLabel      = "com.docker.compose.project"
	composeWorkingDirLabel   = "com.docker.compose.project.working_dir"
	composeConfigFilesLabel  = "com.docker.compose.project.config_files"
)

type environmentLookup func(string) (string, bool)

type runtime struct {
	lookup   environmentLookup
	provider moduleapi.DockerFactsProvider
}

// NewRuntime 以显式 operator 声明和 Docker 原始事实依赖创建 Deployment Runtime。
func NewRuntime(lookup func(string) (string, bool), provider moduleapi.DockerFactsProvider) moduleapi.DeploymentRuntime {
	return runtime{lookup: lookup, provider: provider}
}

func (r runtime) Current(ctx context.Context) moduleapi.DeploymentContext {
	mode := deploymentRuntime(r.lookup)
	root, configured := environmentValue(r.lookup, deploymentComposeRootEnv)
	if configured {
		if mode != "compose" {
			return unavailableContext(mode, "deployment_runtime_unsupported", "deployment runtime must be compose when a Compose root is configured")
		}
		return r.explicitContext(mode, root)
	}
	if mode != "compose" {
		return moduleapi.NewDeploymentContext(mode, "unavailable", false, nil, []moduleapi.DeploymentDiagnostic{{Code: "deployment_mode_unsupported", Message: "deployment mode must be compose before Compose runtime discovery can run"}})
	}
	if r.provider == nil {
		return unavailableContext(mode, "docker_facts_unavailable", "Docker facts provider is unavailable")
	}
	facts, err := r.provider.CurrentContainer(ctx)
	if err != nil {
		return unavailableContext(mode, "docker_facts_unavailable", "current server Docker inspect facts are unavailable")
	}
	candidates := composeCandidates(facts)
	if len(candidates) == 0 {
		return unavailableContext(mode, "compose_candidate_unavailable", "no Compose root candidate was found in current Docker facts")
	}
	return moduleapi.NewDeploymentContext(mode, "docker_discovered", len(candidates) != 1 || candidates[0].Confidence() != "high", candidates, nil)
}

func (r runtime) Freeze(ctx context.Context, request moduleapi.DeploymentFreezeRequest) (moduleapi.DeploymentSnapshot, error) {
	current := r.Current(ctx)
	if !current.IsAvailable() {
		return moduleapi.DeploymentSnapshot{}, errors.New("deployment context is not available for a controlled operation")
	}
	candidates := current.ComposeCandidates()
	candidate, err := selectCandidate(candidates, request.CandidateKey, current.IsComposeConfirmationRequired())
	if err != nil {
		return moduleapi.DeploymentSnapshot{}, err
	}
	fingerprint := candidateFingerprint(current, candidate)
	return moduleapi.NewDeploymentSnapshot(current, candidate, fingerprint), nil
}

func (r runtime) explicitContext(mode, rawRoot string) moduleapi.DeploymentContext {
	root := filepath.Clean(strings.TrimSpace(rawRoot))
	if root == "." || !filepath.IsAbs(root) || root == string(filepath.Separator) {
		return unavailableContext(mode, "configured_compose_root_invalid", deploymentComposeRootEnv+" must be a non-root absolute host path")
	}
	candidate := newCandidate(root, nil, "", "explicit", nil)
	return moduleapi.NewDeploymentContext(mode, "explicit_config", false, []moduleapi.DeploymentComposeCandidate{candidate}, nil)
}

func unavailableContext(mode, code, message string) moduleapi.DeploymentContext {
	return moduleapi.NewDeploymentContext(mode, "unavailable", false, nil, []moduleapi.DeploymentDiagnostic{{Code: code, Message: message}})
}

func deploymentRuntime(lookup environmentLookup) string {
	value, configured := environmentValue(lookup, deploymentRuntimeEnv)
	if !configured || strings.TrimSpace(value) == "" {
		return "compose"
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "compose", "binary":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "unknown"
	}
}

func environmentValue(lookup environmentLookup, key string) (string, bool) {
	if lookup == nil {
		return "", false
	}
	return lookup(key)
}

func composeCandidates(facts moduleapi.DockerContainerFacts) []moduleapi.DeploymentComposeCandidate {
	labels := facts.Labels
	workingDir := cleanAbsolute(labels[composeWorkingDirLabel])
	files := composeConfigFiles(labels[composeConfigFilesLabel])
	if workingDir == "" && len(files) > 0 {
		workingDir = filepath.Dir(files[0])
	}
	values := map[string]moduleapi.DeploymentComposeCandidate{}
	if workingDir != "" {
		values[workingDir] = newCandidate(workingDir, filesWithin(workingDir, files), strings.TrimSpace(labels[composeProjectLabel]), "high", nil)
	}
	if len(values) == 0 {
		for _, mount := range facts.Mounts {
			if mount.Type != "bind" {
				continue
			}
			root := cleanAbsolute(mount.Source)
			if root == "" {
				continue
			}
			values[root] = newCandidate(root, nil, "", "medium", []string{"bind_mount_candidate_requires_administrator_confirmation"})
		}
	}
	result := make([]moduleapi.DeploymentComposeCandidate, 0, len(values))
	for _, candidate := range values {
		result = append(result, candidate)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Confidence() != result[j].Confidence() {
			return result[i].Confidence() == "high"
		}
		return result[i].Root() < result[j].Root()
	})
	return result
}

func newCandidate(root string, files []string, project, confidence string, warnings []string) moduleapi.DeploymentComposeCandidate {
	if confidence == "high" && len(files) == 0 {
		files = []string{filepath.Join(root, "compose.yml")}
	}
	digest := sha256.Sum256([]byte(root + "\x00" + strings.Join(files, "\x00")))
	return moduleapi.NewDeploymentComposeCandidate("compose-"+hex.EncodeToString(digest[:8]), root, files, project, confidence, warnings)
}

func composeConfigFiles(raw string) []string {
	values := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		if path := cleanAbsolute(value); path != "" {
			values = append(values, path)
		}
	}
	return values
}

func filesWithin(root string, files []string) []string {
	values := make([]string, 0, len(files))
	for _, file := range files {
		relative, err := filepath.Rel(root, file)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && relative != "." {
			values = append(values, file)
		}
	}
	return values
}

func cleanAbsolute(value string) string {
	path := filepath.Clean(strings.TrimSpace(value))
	if path == "." || path == string(filepath.Separator) || !filepath.IsAbs(path) {
		return ""
	}
	return path
}

func selectCandidate(candidates []moduleapi.DeploymentComposeCandidate, key string, confirmationRequired bool) (moduleapi.DeploymentComposeCandidate, error) {
	if key == "" && !confirmationRequired && len(candidates) == 1 {
		return candidates[0], nil
	}
	for _, candidate := range candidates {
		if candidate.Key() == key {
			return candidate, nil
		}
	}
	return moduleapi.DeploymentComposeCandidate{}, errors.New("a current Compose candidate confirmation is required")
}

func candidateFingerprint(context moduleapi.DeploymentContext, candidate moduleapi.DeploymentComposeCandidate) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{context.Mode(), context.ComposeRootSource(), candidate.Key(), candidate.Root(), strings.Join(candidate.ConfigFiles(), "\x00"), candidate.ProjectName()}, "\x00")))
	return hex.EncodeToString(digest[:])
}
