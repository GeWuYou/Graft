// Package compose provides bounded static compose parsing for project import and refresh.
package compose

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Input defines the bounded static project parse input for phase 1 import and refresh.
type Input struct {
	WorkingDirectory string
	ComposeFiles     []string
	EnvFiles         []string
}

// FileProjection stores one resolved project file input.
type FileProjection struct {
	AbsolutePath string
	DisplayPath  string
	Kind         string
	Role         string
	OrderIndex   int
	Content      []byte
	Hash         string
	Exists       bool
}

// ServiceProjection stores one static service summary parsed from compose YAML.
type ServiceProjection struct {
	ServiceName      string
	Image            *string
	BuildContext     *string
	DeclaredPorts    []string
	DeclaredVolumes  []string
	DeclaredNetworks []string
}

// Result stores the bounded phase-1 static compose parse result.
type Result struct {
	WorkingDirectory      string
	CanonicalProjectName  string
	CanonicalNameSource   string
	ConfigHash            string
	NormalizedComposeYAML string
	NormalizedComposeJSON []byte
	ComposeFiles          []FileProjection
	EnvFiles              []FileProjection
	ServiceNames          []string
	Services              []ServiceProjection
	Warnings              []string
}

// Load resolves project files, parses a bounded Compose service projection, and returns normalized artifacts.
func Load(input Input) (Result, error) {
	workingDirectory, err := resolveWorkingDirectory(input.WorkingDirectory)
	if err != nil {
		return Result{}, err
	}
	composeFiles, err := resolveComposeFiles(workingDirectory, input.ComposeFiles)
	if err != nil {
		return Result{}, err
	}
	envFiles, err := resolveEnvFiles(workingDirectory, input.EnvFiles)
	if err != nil {
		return Result{}, err
	}

	configHasher := sha256.New()
	serviceOrder, serviceMap, err := collectServices(composeFiles, configHasher)
	if err != nil {
		return Result{}, err
	}
	for _, file := range envFiles {
		if _, err := configHasher.Write(file.Content); err != nil {
			return Result{}, fmt.Errorf("hash env file %s: %w", file.AbsolutePath, err)
		}
	}

	root := map[string]any{
		"services": renderServicesMap(serviceOrder, serviceMap),
	}
	normalizedYAML, normalizedJSON, err := marshalNormalized(root)
	if err != nil {
		return Result{}, err
	}

	services, serviceNames := buildServiceProjections(serviceOrder, serviceMap)

	return Result{
		WorkingDirectory:      workingDirectory,
		CanonicalProjectName:  filepath.Base(workingDirectory),
		CanonicalNameSource:   "computed",
		ConfigHash:            hex.EncodeToString(configHasher.Sum(nil)),
		NormalizedComposeYAML: normalizedYAML,
		NormalizedComposeJSON: normalizedJSON,
		ComposeFiles:          composeFiles,
		EnvFiles:              envFiles,
		ServiceNames:          serviceNames,
		Services:              services,
	}, nil
}

func collectServices(
	composeFiles []FileProjection,
	configHasher hashWriter,
) ([]string, map[string]ServiceProjection, error) {
	serviceOrder := make([]string, 0)
	serviceSet := make(map[string]struct{})
	serviceMap := make(map[string]ServiceProjection)

	for _, file := range composeFiles {
		if _, err := configHasher.Write(file.Content); err != nil {
			return nil, nil, fmt.Errorf("hash compose file %s: %w", file.AbsolutePath, err)
		}
		doc, err := parseComposeDocument(file)
		if err != nil {
			return nil, nil, err
		}
		serviceOrder = collectServicesFromDocument(doc, serviceOrder, serviceSet, serviceMap)
	}

	return serviceOrder, serviceMap, nil
}

type hashWriter interface {
	Write(p []byte) (n int, err error)
}

func parseComposeDocument(file FileProjection) (map[string]any, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(file.Content, &doc); err != nil {
		return nil, fmt.Errorf("parse compose file %s: %w", file.AbsolutePath, err)
	}
	return doc, nil
}

func collectServicesFromDocument(
	doc map[string]any,
	serviceOrder []string,
	serviceSet map[string]struct{},
	serviceMap map[string]ServiceProjection,
) []string {
	for name, raw := range serviceNodesFromDocument(doc) {
		if _, exists := serviceSet[name]; !exists {
			serviceOrder = append(serviceOrder, name)
			serviceSet[name] = struct{}{}
		}
		serviceMap[name] = mergeServiceProjection(serviceMap[name], raw)
	}
	return serviceOrder
}

func serviceNodesFromDocument(doc map[string]any) map[string]any {
	servicesNode, ok := doc["services"].(map[string]any)
	if !ok {
		return nil
	}
	return servicesNode
}

func buildServiceProjections(
	serviceOrder []string,
	serviceMap map[string]ServiceProjection,
) ([]ServiceProjection, []string) {
	sortedNames := append([]string(nil), serviceOrder...)
	sort.Strings(sortedNames)

	services := make([]ServiceProjection, 0, len(sortedNames))
	for _, name := range sortedNames {
		services = append(services, buildServiceProjection(name, serviceMap[name]))
	}
	return services, sortedNames
}

func buildServiceProjection(name string, projection ServiceProjection) ServiceProjection {
	projection.ServiceName = name
	sort.Strings(projection.DeclaredPorts)
	sort.Strings(projection.DeclaredVolumes)
	sort.Strings(projection.DeclaredNetworks)
	return projection
}

func resolveWorkingDirectory(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("working directory is required")
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("stat working directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("working directory must be a directory")
	}
	return absolute, nil
}

func resolveComposeFiles(workingDirectory string, requested []string) ([]FileProjection, error) {
	if len(requested) == 0 {
		requested = []string{"compose.yaml"}
	}
	items := make([]FileProjection, 0, len(requested))
	for index, path := range requested {
		item, err := resolveFileProjection(workingDirectory, path, "compose", composeRole(index), index)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func resolveEnvFiles(workingDirectory string, requested []string) ([]FileProjection, error) {
	items := make([]FileProjection, 0, len(requested))
	for index, path := range requested {
		item, err := resolveFileProjection(workingDirectory, path, "env", "env", index)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func resolveFileProjection(
	workingDirectory string,
	path string,
	kind string,
	role string,
	orderIndex int,
) (FileProjection, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return FileProjection{}, fmt.Errorf("project file path is required")
	}
	absolute := trimmed
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(workingDirectory, trimmed)
	}
	absolute = filepath.Clean(absolute)
	content, err := os.ReadFile(absolute)
	if err != nil {
		return FileProjection{}, fmt.Errorf("read project file %s: %w", absolute, err)
	}
	hash := sha256.Sum256(content)
	return FileProjection{
		AbsolutePath: absolute,
		DisplayPath:  absolute,
		Kind:         kind,
		Role:         role,
		OrderIndex:   orderIndex,
		Content:      content,
		Hash:         hex.EncodeToString(hash[:]),
		Exists:       true,
	}, nil
}

func composeRole(index int) string {
	if index == 0 {
		return "primary"
	}
	return "override"
}

func mergeServiceProjection(existing ServiceProjection, raw any) ServiceProjection {
	result := existing
	node, ok := raw.(map[string]any)
	if !ok {
		return result
	}
	if image, ok := scalarString(node["image"]); ok {
		result.Image = &image
	}
	if buildContext, ok := buildContextValue(node["build"]); ok {
		result.BuildContext = &buildContext
	}
	result.DeclaredPorts = mergeStringList(result.DeclaredPorts, listValues(node["ports"]))
	result.DeclaredVolumes = mergeStringList(result.DeclaredVolumes, listValues(node["volumes"]))
	result.DeclaredNetworks = mergeStringList(result.DeclaredNetworks, networkValues(node["networks"]))
	return result
}

func buildContextValue(raw any) (string, bool) {
	switch value := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(value)
		return trimmed, trimmed != ""
	case map[string]any:
		return scalarString(value["context"])
	default:
		return "", false
	}
}

func scalarString(raw any) (string, bool) {
	value, ok := raw.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	return trimmed, trimmed != ""
}

func listValues(raw any) []string {
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, item := range values {
		switch typed := item.(type) {
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		case map[string]any:
			if target, ok := scalarString(typed["target"]); ok {
				result = append(result, target)
			}
		}
	}
	return result
}

func networkValues(raw any) []string {
	switch typed := raw.(type) {
	case []any:
		return listValues(typed)
	case map[string]any:
		result := make([]string, 0, len(typed))
		for key := range typed {
			trimmed := strings.TrimSpace(key)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	default:
		return nil
	}
}

func mergeStringList(existing []string, values []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(values))
	result := make([]string, 0, len(existing)+len(values))
	for _, item := range append(existing, values...) {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func renderServicesMap(order []string, services map[string]ServiceProjection) map[string]any {
	result := make(map[string]any, len(order))
	for _, name := range order {
		item := services[name]
		service := make(map[string]any)
		if item.Image != nil {
			service["image"] = *item.Image
		}
		if item.BuildContext != nil {
			service["build"] = map[string]any{"context": *item.BuildContext}
		}
		if len(item.DeclaredPorts) > 0 {
			service["ports"] = item.DeclaredPorts
		}
		if len(item.DeclaredVolumes) > 0 {
			service["volumes"] = item.DeclaredVolumes
		}
		if len(item.DeclaredNetworks) > 0 {
			service["networks"] = item.DeclaredNetworks
		}
		result[name] = service
	}
	return result
}

func marshalNormalized(root map[string]any) (string, []byte, error) {
	jsonBytes, err := yaml.Marshal(root)
	if err != nil {
		return "", nil, fmt.Errorf("marshal normalized compose yaml: %w", err)
	}
	var generic any
	if err := yaml.Unmarshal(jsonBytes, &generic); err != nil {
		return "", nil, fmt.Errorf("reparse normalized compose yaml: %w", err)
	}
	jsonCompat, err := toJSONCompatible(generic)
	if err != nil {
		return "", nil, err
	}
	normalizedJSON, err := yaml.Marshal(jsonCompat)
	if err != nil {
		return "", nil, fmt.Errorf("marshal normalized compose snapshot: %w", err)
	}
	return string(jsonBytes), normalizedJSON, nil
}

func toJSONCompatible(raw any) (any, error) {
	switch typed := raw.(type) {
	case map[string]any:
		return normalizeStringMap(typed)
	case map[any]any:
		return normalizeUntypedMap(typed)
	case []any:
		return normalizeArray(typed)
	default:
		return typed, nil
	}
}

func normalizeStringMap(raw map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(raw))
	for key, value := range raw {
		normalized, err := toJSONCompatible(value)
		if err != nil {
			return nil, err
		}
		result[key] = normalized
	}
	return result, nil
}

func normalizeUntypedMap(raw map[any]any) (map[string]any, error) {
	result := make(map[string]any, len(raw))
	for key, value := range raw {
		stringKey, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("compose snapshot contains non-string map key")
		}
		normalized, err := toJSONCompatible(value)
		if err != nil {
			return nil, err
		}
		result[stringKey] = normalized
	}
	return result, nil
}

func normalizeArray(raw []any) ([]any, error) {
	result := make([]any, 0, len(raw))
	for _, value := range raw {
		normalized, err := toJSONCompatible(value)
		if err != nil {
			return nil, err
		}
		result = append(result, normalized)
	}
	return result, nil
}
