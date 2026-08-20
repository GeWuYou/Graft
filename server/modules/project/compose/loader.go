// Package compose 提供项目导入和刷新使用的有界静态 Compose 解析能力。
package compose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var canonicalProjectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// IsValidComposeProjectName 判断 value 是否已经满足 Compose 规范项目名契约。
func IsValidComposeProjectName(value string) bool {
	return canonicalProjectNamePattern.MatchString(value)
}

// Input 定义导入和刷新的第一阶段静态解析输入。
// ContentOverrides 只用于草稿解析，键必须是工作区内的绝对路径，解析器不会执行 Compose 或读取外部服务状态。
type Input struct {
	WorkspacePath    string
	ComposeFiles     []string
	EnvFiles         []string
	ContentOverrides map[string][]byte
}

// WithContentOverrides 复制输入并应用绝对路径内容覆盖，用于不落盘的受控草稿解析。
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

// FileProjection 保存一个已解析项目文件的路径、内容摘要及请求顺序。
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

// ServiceProjection 保存从 Compose YAML 静态提取的服务摘要，不代表运行时服务状态。
type ServiceProjection struct {
	ServiceName      string
	DependsOn        []string
	Image            *string
	BuildContext     *string
	DeclaredPorts    []string
	DeclaredVolumes  []string
	DeclaredNetworks []string
}

// NetworkProjection 保存从 Compose 顶层 networks 静态提取的网络摘要。
type NetworkProjection struct {
	Name     string
	Driver   *string
	Scope    *string
	Internal *bool
}

// VolumeProjection 保存从 Compose 顶层 volumes 静态提取的卷摘要。
type VolumeProjection struct {
	Name   string
	Driver *string
}

// Result 保存第一阶段受控静态 Compose 解析结果及归一化摘要。
type Result struct {
	WorkspacePath         string
	ComposeProjectName    string
	CanonicalNameSource   string
	ConfigHash            string
	NormalizedComposeYAML string
	NormalizedComposeJSON []byte
	ComposeFiles          []FileProjection
	EnvFiles              []FileProjection
	ServiceNames          []string
	Services              []ServiceProjection
	Networks              []NetworkProjection
	Volumes               []VolumeProjection
	NetworkNames          []string
	VolumeNames           []string
	Warnings              []string
}

// Load 解析工作目录、Compose 文件和 Env 文件，返回静态服务投影、归一化快照及配置哈希。
// Compose 文件按输入顺序合并；服务、网络和卷的最终列表会稳定排序。解析阶段只读取文件和 YAML，不执行外部命令。
func Load(input Input) (Result, error) {
	workingDirectory, err := resolveWorkspacePath(input.WorkspacePath)
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

	canonicalProjectName := resolvedComposeProjectName(collected.projectName, workingDirectory)
	if !IsValidComposeProjectName(canonicalProjectName) {
		return Result{}, fmt.Errorf("computed canonical project name is invalid")
	}

	return Result{
		WorkspacePath:         workingDirectory,
		ComposeProjectName:    canonicalProjectName,
		CanonicalNameSource:   "computed",
		ConfigHash:            hex.EncodeToString(configHasher.Sum(nil)),
		NormalizedComposeYAML: normalizedYAML,
		NormalizedComposeJSON: normalizedJSON,
		ComposeFiles:          composeFiles,
		EnvFiles:              envFiles,
		ServiceNames:          serviceNames,
		Services:              services,
		Networks:              collected.networks,
		Volumes:               collected.volumes,
		NetworkNames:          collected.networkNames,
		VolumeNames:           collected.volumeNames,
	}, nil
}

type collectedServices struct {
	serviceOrder []string
	serviceMap   map[string]ServiceProjection
	projectName  string
	networks     []NetworkProjection
	volumes      []VolumeProjection
	networkNames []string
	volumeNames  []string
}

// collectServices 按请求顺序解析并合并 Compose 文件，累计原始文件内容哈希。
// 同名服务按后出现的文件覆盖可静态提取字段；网络和卷在返回前按名称排序，解析或哈希失败时返回错误。
func collectServices(
	composeFiles []FileProjection,
	configHasher hashWriter,
) (collectedServices, error) {
	serviceOrder := make([]string, 0)
	serviceSet := make(map[string]struct{})
	serviceMap := make(map[string]ServiceProjection)
	projectName := ""
	networkSet := make(map[string]struct{})
	volumeSet := make(map[string]struct{})
	networkMap := make(map[string]NetworkProjection)
	volumeMap := make(map[string]VolumeProjection)

	for _, file := range composeFiles {
		if _, err := configHasher.Write(file.Content); err != nil {
			return collectedServices{}, fmt.Errorf("hash compose file %s: %w", file.AbsolutePath, err)
		}
		doc, err := parseComposeDocument(file)
		if err != nil {
			return collectedServices{}, err
		}
		serviceOrder = collectServicesFromDocument(doc, serviceOrder, serviceSet, serviceMap)
		projectName = collectProjectNameFromDocument(doc, projectName)
		collectTopLevelNetworks(doc, networkSet, networkMap)
		collectTopLevelVolumes(doc, volumeSet, volumeMap)
	}

	return collectedServices{
		serviceOrder: serviceOrder,
		serviceMap:   serviceMap,
		projectName:  projectName,
		networks:     sortedNetworkProjections(networkMap),
		volumes:      sortedVolumeProjections(volumeMap),
		networkNames: sortedKeys(networkSet),
		volumeNames:  sortedKeys(volumeSet),
	}, nil
}

type hashWriter interface {
	Write(p []byte) (n int, err error)
}

// parseComposeDocument 将 Compose 文件内容解析为文档映射；失败错误包含文件绝对路径，便于定位外部文件输入。
func parseComposeDocument(file FileProjection) (map[string]any, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(file.Content, &doc); err != nil {
		return nil, fmt.Errorf("parse compose file %s: %w", file.AbsolutePath, err)
	}
	return doc, nil
}

// collectProjectNameFromDocument 从 Compose 文档读取非空顶层 name；没有可用值时保留 current。
func collectProjectNameFromDocument(doc map[string]any, current string) string {
	raw, ok := doc["name"]
	if !ok {
		return current
	}
	name, ok := raw.(string)
	if !ok {
		return current
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return current
	}
	return name
}

// resolvedComposeProjectName 优先使用 Compose 顶层 name，否则使用工作目录名，并统一归一化结果。
func resolvedComposeProjectName(projectName string, workingDirectory string) string {
	candidate := filepath.Base(workingDirectory)
	if strings.TrimSpace(projectName) != "" {
		candidate = projectName
	}
	return normalizeComputedProjectName(candidate)
}

// normalizeComputedProjectName 将项目名称规范化为符合 Compose 项目名格式的值。
// 对空白、大小写和分隔符进行处理；如果规范化结果无效，则返回空字符串。
func normalizeComputedProjectName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return value
	}

	var builder strings.Builder
	builder.Grow(len(normalized))
	lastSeparator := false
	for _, r := range normalized {
		if isAlphaNumericProjectNameRune(r) {
			builder.WriteRune(r)
			lastSeparator = false
			continue
		}
		if shouldSkipProjectNameSeparator(builder.Len(), lastSeparator) {
			continue
		}
		if r == '-' || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteByte('-')
		}
		lastSeparator = true
	}

	result := strings.Trim(builder.String(), "-_")
	if !IsValidComposeProjectName(result) {
		return ""
	}
	return result
}

// isAlphaNumericProjectNameRune 判断字符是否为规范项目名允许的小写 ASCII 字母或数字。
func isAlphaNumericProjectNameRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// shouldSkipProjectNameSeparator 判断当前规范化结果是否应跳过重复或开头的分隔符。
func shouldSkipProjectNameSeparator(builderLen int, lastSeparator bool) bool {
	return builderLen == 0 || lastSeparator
}

// collectServicesFromDocument 记录服务首次出现的名称，并合并同名服务的静态投影。
// YAML 映射本身无顺序保证，因此该顺序只用于去重，最终展示顺序由 buildServiceProjections 稳定排序。
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

// buildServiceProjections 按名称排序生成服务投影列表和服务名列表，消除 YAML 映射遍历顺序带来的结果抖动。
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

// buildServiceProjection 设置服务名，并按字典序排序端口、卷和网络，保证快照序列化稳定。
func buildServiceProjection(name string, projection ServiceProjection) ServiceProjection {
	projection.ServiceName = name
	sort.Strings(projection.DeclaredPorts)
	sort.Strings(projection.DeclaredVolumes)
	sort.Strings(projection.DeclaredNetworks)
	sort.Strings(projection.DependsOn)
	return projection
}

// collectTopLevelNetworks 从 compose 顶层 networks 节点收集网络名称和可静态识别的元数据。
func collectTopLevelNetworks(
	doc map[string]any,
	target map[string]struct{},
	projections map[string]NetworkProjection,
) {
	if target == nil || projections == nil {
		return
	}
	raw, ok := doc["networks"]
	if !ok {
		return
	}
	items, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for name, definition := range items {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		target[trimmed] = struct{}{}
		projections[trimmed] = networkProjectionFromDefinition(trimmed, projections[trimmed], definition)
	}
}

// collectTopLevelVolumes 从 compose 顶层 volumes 节点收集卷名称和可静态识别的元数据。
func collectTopLevelVolumes(
	doc map[string]any,
	target map[string]struct{},
	projections map[string]VolumeProjection,
) {
	if target == nil || projections == nil {
		return
	}
	raw, ok := doc["volumes"]
	if !ok {
		return
	}
	items, ok := raw.(map[string]any)
	if !ok {
		return
	}
	for name, definition := range items {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		target[trimmed] = struct{}{}
		projection := projections[trimmed]
		projection.Name = trimmed
		if node, ok := definition.(map[string]any); ok {
			if driver, ok := scalarString(node["driver"]); ok {
				projection.Driver = &driver
			}
		}
		projections[trimmed] = projection
	}
}

func sortedNetworkProjections(items map[string]NetworkProjection) []NetworkProjection {
	if len(items) == 0 {
		return nil
	}
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]NetworkProjection, 0, len(names))
	for _, name := range names {
		projection := items[name]
		if projection.Name == "" {
			projection.Name = name
		}
		result = append(result, projection)
	}
	return result
}

func networkProjectionFromDefinition(name string, current NetworkProjection, definition any) NetworkProjection {
	current.Name = name
	node, ok := definition.(map[string]any)
	if !ok {
		return current
	}
	if driver, ok := scalarString(node["driver"]); ok {
		current.Driver = &driver
	}
	if scope, ok := scalarString(node["scope"]); ok {
		current.Scope = &scope
	}
	if internal, ok := scalarBool(node["internal"]); ok {
		current.Internal = &internal
	}
	return current
}

func sortedVolumeProjections(items map[string]VolumeProjection) []VolumeProjection {
	if len(items) == 0 {
		return nil
	}
	names := make([]string, 0, len(items))
	for name := range items {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]VolumeProjection, 0, len(names))
	for _, name := range names {
		projection := items[name]
		if projection.Name == "" {
			projection.Name = name
		}
		result = append(result, projection)
	}
	return result
}

// sortedKeys 返回按字典序排序的键名列表。
// 当输入集合为空时返回 nil。
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

// resolveWorkspacePath 解析并校验工作目录路径，确认路径存在且为目录后返回绝对路径。
func resolveWorkspacePath(raw string) (string, error) {
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

// resolveComposeFiles 按请求顺序解析 Compose 文件投影；未指定文件时只使用默认的 compose.yaml。
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

// resolveEnvFiles 按请求顺序解析环境文件投影，不擅自添加默认 Env 文件。
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
		content, err = readFileWithinWorkspacePath(workingDirectory, absolute)
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

func readFileWithinWorkspacePath(workingDirectory string, absolutePath string) ([]byte, error) {
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

// composeRole 返回 Compose 文件在请求顺序中的角色：第一个为 primary，其余为 override。
func composeRole(index int) string {
	if index == 0 {
		return "primary"
	}
	return "override"
}

// mergeServiceProjection 合并服务投影中可静态提取的字段。
//
// 会从服务节点中提取并合并 image、build.context、depends_on、ports、volumes 和 networks 等信息，
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
	result.DependsOn = mergeStringList(result.DependsOn, dependencyValues(node["depends_on"]))
	return result
}

// dependencyValues 提取 Compose depends_on 的列表或映射形式，返回非空服务名。
func dependencyValues(raw any) []string {
	switch value := raw.(type) {
	case []any:
		return listValues(value)
	case map[string]any:
		result := make([]string, 0, len(value))
		for name := range value {
			if trimmed := strings.TrimSpace(name); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	default:
		return nil
	}
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

func scalarBool(raw any) (bool, bool) {
	value, ok := raw.(bool)
	return value, ok
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
