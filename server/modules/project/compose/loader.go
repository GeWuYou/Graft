// Package compose provides bounded static compose parsing for project import and refresh.
package compose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	ContentOverrides map[string][]byte
}

// WithContentOverrides clones the input and applies absolute-path content overrides for bounded draft parsing.
func (in Input) WithContentOverrides(overrides map[string][]byte) Input {
	if len(overrides) == 0 {
		return in
	}
	cloned := make(map[string][]byte, len(overrides))
	for path, content := range overrides {
		if strings.TrimSpace(path) == "" || content == nil {
			continue
		}
		cloned[path] = append([]byte(nil), content...)
	}
	in.ContentOverrides = cloned
	return in
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
	NetworkNames          []string
	VolumeNames           []string
	Warnings              []string
}

// Load 解析项目文件并返回归一化的 Compose 服务投影结果与输入摘要。
// 它会解析工作目录、Compose 文件和 Env 文件，汇总服务定义，生成配置哈希，并输出归一化的 Compose 快照以及解析后的文件和服务列表。
func Load(input Input) (Result, error) {
	workingDirectory, err := resolveWorkingDirectory(input.WorkingDirectory)
	if err != nil {
		return Result{}, err
	}
	composeFiles, err := resolveComposeFiles(workingDirectory, input.ComposeFiles, input.ContentOverrides)
	if err != nil {
		return Result{}, err
	}
	envFiles, err := resolveEnvFiles(workingDirectory, input.EnvFiles, input.ContentOverrides)
	if err != nil {
		return Result{}, err
	}

	configHasher := sha256.New()
	collected, err := collectServices(composeFiles, configHasher)
	if err != nil {
		return Result{}, err
	}
	for _, file := range envFiles {
		if _, err := configHasher.Write(file.Content); err != nil {
			return Result{}, fmt.Errorf("hash env file %s: %w", file.AbsolutePath, err)
		}
	}

	root := map[string]any{
		"services": renderServicesMap(collected.serviceOrder, collected.serviceMap),
	}
	normalizedYAML, normalizedJSON, err := marshalNormalized(root)
	if err != nil {
		return Result{}, err
	}

	services, serviceNames := buildServiceProjections(collected.serviceOrder, collected.serviceMap)

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
		NetworkNames:          collected.networkNames,
		VolumeNames:           collected.volumeNames,
	}, nil
}

type collectedServices struct {
	serviceOrder []string
	serviceMap   map[string]ServiceProjection
	networkNames []string
	volumeNames  []string
}

// collectServices 处理已解析的 Compose 文件，汇总服务投影并计算配置哈希。
//
// 它按输入顺序处理每个文件，合并同名服务的静态信息，并返回服务首次出现的顺序和最终的服务映射。
// 当文件内容写入哈希器失败或 Compose 文档解析失败时返回错误。
//
// @param composeFiles 已解析的 Compose 文件。
// @param configHasher 用于累计配置内容哈希的写入器。
// @returns 服务首次出现的顺序、按名称合并后的服务映射，以及错误。
func collectServices(
	composeFiles []FileProjection,
	configHasher hashWriter,
) (collectedServices, error) {
	serviceOrder := make([]string, 0)
	serviceSet := make(map[string]struct{})
	serviceMap := make(map[string]ServiceProjection)
	networkSet := make(map[string]struct{})
	volumeSet := make(map[string]struct{})

	for _, file := range composeFiles {
		if _, err := configHasher.Write(file.Content); err != nil {
			return collectedServices{}, fmt.Errorf("hash compose file %s: %w", file.AbsolutePath, err)
		}
		doc, err := parseComposeDocument(file)
		if err != nil {
			return collectedServices{}, err
		}
		serviceOrder = collectServicesFromDocument(doc, serviceOrder, serviceSet, serviceMap)
		collectTopLevelKeys(doc, "networks", networkSet)
		collectTopLevelKeys(doc, "volumes", volumeSet)
	}

	return collectedServices{
		serviceOrder: serviceOrder,
		serviceMap:   serviceMap,
		networkNames: sortedKeys(networkSet),
		volumeNames:  sortedKeys(volumeSet),
	}, nil
}

type hashWriter interface {
	Write(p []byte) (n int, err error)
}

// parseComposeDocument 将 Compose 文件内容解析为通用文档。
// 解析失败时返回包含文件绝对路径的错误。
func parseComposeDocument(file FileProjection) (map[string]any, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(file.Content, &doc); err != nil {
		return nil, fmt.Errorf("parse compose file %s: %w", file.AbsolutePath, err)
	}
	return doc, nil
}

// 它会按首次出现的顺序更新 serviceOrder，并将同名服务的静态投影合并到 serviceMap 中。
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

// serviceNodesFromDocument 提取文档中的 services 节点。
// 如果该节点不存在或类型不匹配，则返回 nil。
func serviceNodesFromDocument(doc map[string]any) map[string]any {
	servicesNode, ok := doc["services"].(map[string]any)
	if !ok {
		return nil
	}
	return servicesNode
}

// buildServiceProjections 按名称排序生成服务投影列表和服务名列表。
// 返回的服务切片与服务名切片均按字典序排列。
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

// buildServiceProjection 设置服务名并对声明的端口、卷和网络列表按字典序排序。
func buildServiceProjection(name string, projection ServiceProjection) ServiceProjection {
	projection.ServiceName = name
	sort.Strings(projection.DeclaredPorts)
	sort.Strings(projection.DeclaredVolumes)
	sort.Strings(projection.DeclaredNetworks)
	return projection
}

func collectTopLevelKeys(doc map[string]any, key string, target map[string]struct{}) {
	if target == nil {
		return
	}
	raw, ok := doc[key]
	if !ok {
		return
	}
	items, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for name := range items {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		target[trimmed] = struct{}{}
	}
}

func sortedKeys(items map[string]struct{}) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	for item := range items {
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

// resolveWorkingDirectory 解析并校验工作目录路径。
// 它会去除首尾空白，转换为绝对路径，并确认该路径存在且为目录。
// @param raw 原始工作目录路径。
// @returns 解析后的绝对目录路径，或返回错误。
func resolveWorkingDirectory(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("working directory is required")
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return "", fmt.Errorf("open working directory: %w", err)
	}
	defer func() {
		_ = root.Close()
	}()
	info, err := root.Stat(".")
	if err != nil {
		return "", fmt.Errorf("stat working directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("working directory must be a directory")
	}
	return absolute, nil
}

// resolveComposeFiles 解析并读取 Compose 文件投影。
// 当未指定文件时，默认使用 `compose.yaml`。
// @param workingDirectory 用于解析相对路径的工作目录。
// @param requested 请求的 Compose 文件路径列表。
// @returns 解析后的 Compose 文件投影列表。
func resolveComposeFiles(workingDirectory string, requested []string, overrides map[string][]byte) ([]FileProjection, error) {
	if len(requested) == 0 {
		requested = []string{"compose.yaml"}
	}
	resolvedOverrides, err := inputOverrides(workingDirectory, overrides)
	if err != nil {
		return nil, err
	}
	items := make([]FileProjection, 0, len(requested))
	for index, path := range requested {
		item, err := resolveFileProjection(workingDirectory, path, "compose", composeRole(index), index, resolvedOverrides)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// resolveEnvFiles 按顺序解析并读取环境文件投影。
//
// requested 为空时不添加默认文件；返回的切片保持请求顺序。
func resolveEnvFiles(workingDirectory string, requested []string, overrides map[string][]byte) ([]FileProjection, error) {
	resolvedOverrides, err := inputOverrides(workingDirectory, overrides)
	if err != nil {
		return nil, err
	}
	items := make([]FileProjection, 0, len(requested))
	for index, path := range requested {
		item, err := resolveFileProjection(workingDirectory, path, "env", "env", index, resolvedOverrides)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// resolveFileProjection 解析并读取项目文件，生成其路径、内容和哈希投影。
// path 为空时返回错误；相对路径会基于工作目录解析，绝对路径保持不变。
// 返回包含文件绝对路径、显示路径、类型、角色、顺序索引、内容、哈希以及存在标记的投影。
func resolveFileProjection(
	workingDirectory string,
	path string,
	kind string,
	role string,
	orderIndex int,
	overrides map[string][]byte,
) (FileProjection, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return FileProjection{}, fmt.Errorf("project file path is required")
	}
	absolute, err := resolveBoundedPath(workingDirectory, trimmed)
	if err != nil {
		return FileProjection{}, err
	}
	content, ok := overrides[absolute]
	if !ok {
		content, err = readFileWithinWorkingDirectory(workingDirectory, absolute)
		if err != nil {
			return FileProjection{}, fmt.Errorf("read project file %s: %w", absolute, err)
		}
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

func inputOverrides(workingDirectory string, overrides map[string][]byte) (map[string][]byte, error) {
	if len(overrides) == 0 {
		return nil, nil
	}
	result := make(map[string][]byte, len(overrides))
	for rawPath, content := range overrides {
		absolute, err := resolveBoundedPath(workingDirectory, rawPath)
		if err != nil {
			return nil, fmt.Errorf("resolve content override %q: %w", rawPath, err)
		}
		result[absolute] = append([]byte(nil), content...)
	}
	return result, nil
}

func resolveBoundedPath(workingDirectory string, rawPath string) (string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", fmt.Errorf("project file path is required")
	}
	if filepath.IsAbs(trimmed) {
		absolute := filepath.Clean(trimmed)
		relative, err := filepath.Rel(workingDirectory, absolute)
		if err != nil {
			return "", fmt.Errorf("resolve project file path %s: %w", trimmed, err)
		}
		if relative == "." || strings.HasPrefix(relative, "..") {
			return "", fmt.Errorf("project file path must stay under working directory")
		}
		return absolute, nil
	}
	absolute := filepath.Clean(filepath.Join(workingDirectory, trimmed))
	relative, err := filepath.Rel(workingDirectory, absolute)
	if err != nil {
		return "", fmt.Errorf("resolve project file path %s: %w", trimmed, err)
	}
	if relative == "." || strings.HasPrefix(relative, "..") {
		return "", fmt.Errorf("project file path must stay under working directory")
	}
	return absolute, nil
}

func readFileWithinWorkingDirectory(workingDirectory string, absolutePath string) ([]byte, error) {
	root, err := os.OpenRoot(workingDirectory)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = root.Close()
	}()
	relative, err := filepath.Rel(workingDirectory, absolutePath)
	if err != nil {
		return nil, err
	}
	if relative == "." || relative == "" || strings.HasPrefix(relative, "..") {
		return nil, fmt.Errorf("project file path must stay under working directory")
	}
	return root.ReadFile(relative)
}

// composeRole 返回 Compose 文件在请求顺序中的角色。
//
// @returns 第 0 个文件返回 "primary"，其余文件返回 "override"。
func composeRole(index int) string {
	if index == 0 {
		return "primary"
	}
	return "override"
}

// mergeServiceProjection 合并服务投影中可静态提取的字段。
//
// 会从服务节点中提取并合并 image、build.context、ports、volumes 和 networks 等信息，
// 并保留已有投影中已收集的值。
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

// buildContextValue 提取可静态识别的构建上下文路径。
// 当输入为字符串时，返回去除首尾空白后的值；当输入为包含 `context` 键的映射时，返回该键对应的字符串值。
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

// scalarString 提取并裁剪字符串值。
// 返回裁剪后的字符串，以及其是否非空。
func scalarString(raw any) (string, bool) {
	value, ok := raw.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	return trimmed, trimmed != ""
}

// listValues 提取列表中的字符串值。
// 它会返回字符串元素的去空白结果，并从包含 target 字段的映射元素中提取 target 值。
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

// networkValues 提取服务网络配置中的网络名称。
// 当输入为数组时，返回其中可识别的网络项；当输入为映射时，返回映射键。
// @returns 提取到的网络名称列表。
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

// mergeStringList 合并两个字符串列表，去重并裁剪空白后返回。
// 保持输入拼接后的相对顺序，忽略空字符串和重复项。
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

// renderServicesMap 按给定顺序构建服务配置映射，仅保留已解析出的服务字段。
//
// 返回的映射以服务名为键；每个服务只包含存在的 image、build.context、ports、volumes 和 networks 字段。
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

// marshalNormalized 生成 Compose 根结构的归一化 YAML 和 JSON 兼容快照。
//
// 返回归一化后的 YAML 文本和再次规范化后的快照字节。
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
	normalizedJSON, err := json.Marshal(jsonCompat)
	if err != nil {
		return "", nil, fmt.Errorf("marshal normalized compose snapshot: %w", err)
	}
	return string(jsonBytes), normalizedJSON, nil
}

// toJSONCompatible 递归将任意 YAML 解析结果转换为 JSON 兼容的结构。
// 它会把映射和数组中的嵌套值继续规范化，确保映射键为字符串。
// @returns 规范化后的值；当输入已是可直接使用的标量值时，原值原样返回。
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

// normalizeStringMap 递归规范化字符串键映射中的所有值。
// 返回规范化后的 map[string]any。
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

// normalizeUntypedMap 递归规范化键为任意类型的映射，生成以字符串为键的映射。
// 如果输入中包含无法转换为字符串的键，返回错误。
//
// @param raw 原始映射。
// @returns 规范化后的 `map[string]any`；当存在非字符串键或子值规范化失败时返回错误。
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

// normalizeArray 递归规范化数组中的每个元素。
// 返回规范化后的切片；如果任一元素无法转换为 JSON 兼容结构，则返回错误。
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
