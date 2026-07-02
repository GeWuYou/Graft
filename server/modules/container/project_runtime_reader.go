package container

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"graft/server/internal/moduleapi"
)

const (
	projectRuntimeCandidateStatusReady              = "ready"
	projectRuntimeCandidateStatusIncompleteMetadata = "incomplete_metadata"
	projectRuntimeCandidateStatusUnsupportedRuntime = "unsupported_runtime"

	projectRuntimeReasonMissingProjectName       = "missing_project_name"
	projectRuntimeReasonMissingConfigFiles       = "missing_config_files"
	projectRuntimeReasonInvalidConfigFiles       = "invalid_config_files"
	projectRuntimeReasonConflictingMetadata      = "conflicting_runtime_metadata"
	projectRuntimeReasonUnsupportedRuntimeType   = "unsupported_runtime_type"
	projectRuntimeReasonConfigFilesNotAccessible = "config_files_not_accessible"
	projectRuntimeWarningWorkingDirDerived       = "working_directory_derived_from_config_files"
	projectRuntimeWorkingDirSourceRuntimeLabel   = "runtime_label"
	projectRuntimeWorkingDirSourceDerivedConfig  = "derived_from_config_files"
	projectRuntimeHostScopeLocal                 = "local"
)

type containerProjectRuntimeReader struct {
	service *service
}

type runtimeCandidateAccumulator struct {
	canonicalProjectName   string
	runtimeType            string
	runtimeVersion         string
	workingDirectory       string
	workingDirectorySource string
	configFiles            []string
	serviceNames           map[string]struct{}
	runningCount           int
	stoppedCount           int
	warnings               []string
	reasonCodes            []string
	metadataConflict       bool
	unsupportedRuntime     bool
}

func (r containerProjectRuntimeReader) ListProjectMembers(
	ctx context.Context,
	hostScope string,
	canonicalProjectName string,
) (moduleapi.ContainerProjectRuntimeSummary, error) {
	if r.service == nil {
		return moduleapi.ContainerProjectRuntimeSummary{}, errRuntimeDisabled
	}
	summary := moduleapi.ContainerProjectRuntimeSummary{
		CanonicalProjectName: strings.TrimSpace(canonicalProjectName),
		Members:              []moduleapi.ContainerProjectMember{},
	}
	if strings.TrimSpace(hostScope) == "" || strings.TrimSpace(canonicalProjectName) == "" {
		return summary, nil
	}
	runtime, err := r.service.runtimeForRequest()
	if err != nil {
		return summary, err
	}
	offset := 0
	for {
		items, listErr := runtime.List(ctx, ListQuery{
			Limit:           maxContainerListLimit,
			Offset:          offset,
			Orchestrator:    containerOrchestratorCompose,
			SourceScopeKind: composeProjectScopeKind,
			SourceScope:     canonicalProjectName,
		})
		if listErr != nil {
			return summary, listErr
		}
		appendProjectMembers(&summary, items, canonicalProjectName)
		if len(items) < maxContainerListLimit {
			break
		}
		offset += len(items)
	}
	sort.Slice(summary.Members, func(i, j int) bool {
		if summary.Members[i].ServiceName == summary.Members[j].ServiceName {
			if summary.Members[i].ContainerName == summary.Members[j].ContainerName {
				return summary.Members[i].ContainerID < summary.Members[j].ContainerID
			}
			return summary.Members[i].ContainerName < summary.Members[j].ContainerName
		}
		return summary.Members[i].ServiceName < summary.Members[j].ServiceName
	})
	return summary, nil
}

func (r containerProjectRuntimeReader) ListImportCandidates(
	ctx context.Context,
	hostScope string,
) ([]moduleapi.ContainerProjectRuntimeCandidate, error) {
	if r.service == nil {
		return nil, errRuntimeDisabled
	}
	hostScope = strings.TrimSpace(hostScope)
	if hostScope == "" || hostScope != projectRuntimeHostScopeLocal {
		return []moduleapi.ContainerProjectRuntimeCandidate{}, nil
	}
	runtime, err := r.service.runtimeForRequest()
	if err != nil {
		return nil, err
	}
	summaries, err := listComposeRuntimeSummaries(ctx, runtime)
	if err != nil {
		return nil, err
	}
	info, err := runtime.Info(ctx)
	if err != nil {
		return nil, err
	}
	candidates := runtimeImportCandidatesFromSummaries(hostScope, summaries, info)
	sortRuntimeCandidates(candidates)
	return candidates, nil
}

func (r containerProjectRuntimeReader) ListImportCandidateMembers(
	ctx context.Context,
	hostScope string,
	candidate moduleapi.ContainerProjectRuntimeCandidate,
) ([]moduleapi.ContainerProjectMember, error) {
	if r.service == nil {
		return nil, errRuntimeDisabled
	}
	hostScope = strings.TrimSpace(hostScope)
	if !supportsImportCandidateHostScope(hostScope) {
		return []moduleapi.ContainerProjectMember{}, nil
	}
	if !hasImportCandidateKey(candidate) {
		return []moduleapi.ContainerProjectMember{}, nil
	}
	runtime, err := r.service.runtimeForRequest()
	if err != nil {
		return nil, err
	}
	summaries, err := listComposeRuntimeSummaries(ctx, runtime)
	if err != nil {
		return nil, err
	}
	return importCandidateMembersFromSummaries(hostScope, summaries, candidate), nil
}

func listComposeRuntimeSummaries(ctx context.Context, runtime Runtime) ([]Summary, error) {
	summaries := make([]Summary, 0)
	offset := 0
	for {
		items, err := runtime.List(ctx, ListQuery{
			Limit:        maxContainerListLimit,
			Offset:       offset,
			Orchestrator: containerOrchestratorCompose,
		})
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, items...)
		if len(items) < maxContainerListLimit {
			return summaries, nil
		}
		offset += len(items)
	}
}

func supportsImportCandidateHostScope(hostScope string) bool {
	return hostScope == projectRuntimeHostScopeLocal
}

func hasImportCandidateKey(candidate moduleapi.ContainerProjectRuntimeCandidate) bool {
	return strings.TrimSpace(candidate.CandidateKey) != ""
}

func importCandidateMembersFromSummaries(
	hostScope string,
	summaries []Summary,
	candidate moduleapi.ContainerProjectRuntimeCandidate,
) []moduleapi.ContainerProjectMember {
	members := make([]moduleapi.ContainerProjectMember, 0)
	for _, item := range summaries {
		if !matchesRuntimeCandidate(hostScope, item, candidate) {
			continue
		}
		members = append(members, moduleapi.ContainerProjectMember{
			ContainerID:    strings.TrimSpace(item.ID),
			ContainerName:  strings.TrimSpace(item.Name),
			ServiceName:    composeServiceName(item),
			CanonicalState: normalizeContainerState(item.State),
		})
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].ServiceName == members[j].ServiceName {
			if members[i].ContainerName == members[j].ContainerName {
				return members[i].ContainerID < members[j].ContainerID
			}
			return members[i].ContainerName < members[j].ContainerName
		}
		return members[i].ServiceName < members[j].ServiceName
	})
	return members
}

func runtimeImportCandidatesFromSummaries(
	hostScope string,
	summaries []Summary,
	info RuntimeInfo,
) []moduleapi.ContainerProjectRuntimeCandidate {
	accumulators := make(map[string]*runtimeCandidateAccumulator)
	order := make([]string, 0)
	for _, item := range summaries {
		groupKey := runtimeCandidateGroupKey(hostScope, item)
		accumulator, ok := accumulators[groupKey]
		if !ok {
			accumulator = &runtimeCandidateAccumulator{
				serviceNames: make(map[string]struct{}),
			}
			accumulators[groupKey] = accumulator
			order = append(order, groupKey)
		}
		accumulator.absorb(item, info.Runtime, info.ServerVersion)
	}
	candidates := make([]moduleapi.ContainerProjectRuntimeCandidate, 0, len(order))
	for _, key := range order {
		candidates = append(candidates, accumulators[key].candidate(hostScope))
	}
	return candidates
}

func sortRuntimeCandidates(candidates []moduleapi.ContainerProjectRuntimeCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Importable != candidates[j].Importable {
			return candidates[i].Importable
		}
		leftName := strings.ToLower(strings.TrimSpace(candidates[i].CanonicalProjectName))
		rightName := strings.ToLower(strings.TrimSpace(candidates[j].CanonicalProjectName))
		if leftName != rightName {
			return leftName < rightName
		}
		return candidates[i].CandidateKey < candidates[j].CandidateKey
	})
}

func (a *runtimeCandidateAccumulator) absorb(item Summary, runtimeType string, runtimeVersion string) {
	projectName := composeProjectName(item)
	serviceName := composeServiceName(item)
	workingDirectory, workingDirectorySource, configFiles := a.resolvedWorkingDirectoryMetadata(item)
	a.mergeCanonicalProjectName(projectName)
	a.mergeRuntimeInfo(runtimeType, runtimeVersion)
	a.mergeWorkingDirectory(workingDirectory, workingDirectorySource)
	a.mergeConfigFiles(configFiles)
	if serviceName != "" {
		a.serviceNames[serviceName] = struct{}{}
	}
	a.addContainerState(item.State)
	a.markRuntimeSupport(item, runtimeType)
	a.warnings = appendUniqueString(a.warnings, item.Orchestrator.Warnings...)
}

func (a *runtimeCandidateAccumulator) candidate(hostScope string) moduleapi.ContainerProjectRuntimeCandidate {
	status := a.finalizeStatus()
	serviceNames := make([]string, 0, len(a.serviceNames))
	for name := range a.serviceNames {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	normalizedProjectName := strings.TrimSpace(a.canonicalProjectName)
	candidateKey := runtimeCandidateKey(hostScope, normalizedProjectName, a.runtimeType, a.workingDirectory, a.configFiles)
	reasonCodes := nonNilStringSlice(a.reasonCodes)
	sort.Strings(reasonCodes)
	configFiles := nonNilStringSlice(a.configFiles)
	warnings := nonNilStringSlice(a.warnings)
	sort.Strings(warnings)

	return moduleapi.ContainerProjectRuntimeCandidate{
		CandidateKey:           candidateKey,
		CanonicalProjectName:   normalizedProjectName,
		Status:                 status,
		StatusReasonCodes:      reasonCodes,
		Importable:             status == projectRuntimeCandidateStatusReady,
		RuntimeType:            strings.TrimSpace(a.runtimeType),
		RuntimeVersion:         strings.TrimSpace(a.runtimeVersion),
		WorkingDirectory:       strings.TrimSpace(a.workingDirectory),
		WorkingDirectorySource: strings.TrimSpace(a.workingDirectorySource),
		ConfigFiles:            configFiles,
		ServiceNames:           serviceNames,
		ContainerCounts: moduleapi.ContainerProjectRuntimeContainerCounts{
			Running: a.runningCount,
			Stopped: a.stoppedCount,
			Total:   a.runningCount + a.stoppedCount,
		},
		Warnings: warnings,
	}
}

func (a *runtimeCandidateAccumulator) resolvedWorkingDirectoryMetadata(item Summary) (string, string, []string) {
	workingDirectory, workingDirectorySource, configFiles := resolveRuntimeCandidateMetadata(item)
	if workingDirectorySource == projectRuntimeWorkingDirSourceDerivedConfig {
		a.addWarning(projectRuntimeWarningWorkingDirDerived)
	}
	return workingDirectory, workingDirectorySource, configFiles
}

func resolveRuntimeCandidateMetadata(item Summary) (string, string, []string) {
	workingDirectory := strings.TrimSpace(item.Orchestrator.WorkingDir)
	workingDirectorySource := ""
	if workingDirectory != "" {
		workingDirectorySource = projectRuntimeWorkingDirSourceRuntimeLabel
	}
	configFiles := normalizedStringSlice(item.Orchestrator.ConfigFiles)
	if workingDirectory == "" && len(configFiles) > 0 {
		if derivedWorkingDirectory, ok := derivedWorkingDirectoryFromConfigFile(configFiles[0]); ok {
			workingDirectory = derivedWorkingDirectory
			workingDirectorySource = projectRuntimeWorkingDirSourceDerivedConfig
		}
	}
	return workingDirectory, workingDirectorySource, configFiles
}

func (a *runtimeCandidateAccumulator) mergeCanonicalProjectName(projectName string) {
	if a.canonicalProjectName == "" {
		a.canonicalProjectName = projectName
		return
	}
	if projectName != "" && !strings.EqualFold(a.canonicalProjectName, projectName) {
		a.metadataConflict = true
	}
}

func (a *runtimeCandidateAccumulator) mergeRuntimeInfo(runtimeType string, runtimeVersion string) {
	if a.runtimeType == "" {
		a.runtimeType = strings.TrimSpace(runtimeType)
	}
	if a.runtimeVersion == "" {
		a.runtimeVersion = strings.TrimSpace(runtimeVersion)
	}
}

func (a *runtimeCandidateAccumulator) mergeWorkingDirectory(workingDirectory string, source string) {
	if a.workingDirectory == "" {
		a.workingDirectory = workingDirectory
		a.workingDirectorySource = source
		return
	}
	if workingDirectory != "" && filepath.Clean(a.workingDirectory) != filepath.Clean(workingDirectory) {
		a.metadataConflict = true
	}
}

func (a *runtimeCandidateAccumulator) mergeConfigFiles(configFiles []string) {
	if len(a.configFiles) == 0 {
		a.configFiles = append([]string(nil), configFiles...)
		return
	}
	if len(configFiles) > 0 && !sameStringSlice(a.configFiles, configFiles) {
		a.metadataConflict = true
	}
}

func (a *runtimeCandidateAccumulator) addContainerState(state string) {
	if normalizeContainerState(state) == "running" {
		a.runningCount++
		return
	}
	a.stoppedCount++
}

func (a *runtimeCandidateAccumulator) markRuntimeSupport(item Summary, runtimeType string) {
	if effectiveOrchestratorType(item) != containerOrchestratorCompose || !strings.EqualFold(strings.TrimSpace(runtimeType), runtimeNameDocker) {
		a.unsupportedRuntime = true
	}
}

func (a *runtimeCandidateAccumulator) finalizeStatus() string {
	a.recordFinalReasons()
	if a.unsupportedRuntime {
		return projectRuntimeCandidateStatusUnsupportedRuntime
	}
	if len(a.reasonCodes) > 0 {
		return projectRuntimeCandidateStatusIncompleteMetadata
	}
	return projectRuntimeCandidateStatusReady
}

func (a *runtimeCandidateAccumulator) recordFinalReasons() {
	if strings.TrimSpace(a.canonicalProjectName) == "" {
		a.addReason(projectRuntimeReasonMissingProjectName)
	}
	if len(a.configFiles) == 0 {
		a.addReason(projectRuntimeReasonMissingConfigFiles)
	}
	if a.metadataConflict {
		a.addReason(projectRuntimeReasonConflictingMetadata)
	}
	if a.unsupportedRuntime {
		a.addReason(projectRuntimeReasonUnsupportedRuntimeType)
	}
	if !a.hasConfigFiles() {
		return
	}
	if !configFilesWithinWorkingDirectory(a.workingDirectory, a.configFiles) {
		a.addReason(projectRuntimeReasonInvalidConfigFiles)
	}
	if !configFilesAccessible(a.workingDirectory, a.configFiles) {
		a.addReason(projectRuntimeReasonConfigFilesNotAccessible)
	}
}

func (a *runtimeCandidateAccumulator) hasConfigFiles() bool {
	return len(a.configFiles) > 0
}

func (a *runtimeCandidateAccumulator) addReason(code string) {
	a.reasonCodes = appendUniqueString(a.reasonCodes, code)
}

func (a *runtimeCandidateAccumulator) addWarning(code string) {
	a.warnings = appendUniqueString(a.warnings, code)
}

// appendProjectMembers 将匹配指定项目的运行时摘要转换为成员列表，并更新运行与停止数量。
// 仅会追加与 canonicalProjectName 对应的条目；状态为 running 的成员计入运行数，其余成员计入停止数。
func appendProjectMembers(
	summary *moduleapi.ContainerProjectRuntimeSummary,
	items []Summary,
	canonicalProjectName string,
) {
	for _, item := range items {
		member, ok := toProjectMember(item, canonicalProjectName)
		if !ok {
			continue
		}
		summary.Members = append(summary.Members, member)
		if member.CanonicalState == "running" {
			summary.RunningCount++
			continue
		}
		summary.StoppedCount++
	}
}

// toProjectMember 将运行时摘要项转换为指定项目的成员信息。
// 仅当 item 的 ComposeProject 去除空格后与 canonicalProjectName 忽略大小写相等时，才返回转换后的成员信息。
func toProjectMember(item Summary, canonicalProjectName string) (moduleapi.ContainerProjectMember, bool) {
	if !strings.EqualFold(composeProjectName(item), canonicalProjectName) {
		return moduleapi.ContainerProjectMember{}, false
	}
	return moduleapi.ContainerProjectMember{
		ContainerID:    strings.TrimSpace(item.ID),
		ContainerName:  strings.TrimSpace(item.Name),
		ServiceName:    composeServiceName(item),
		CanonicalState: normalizeContainerState(item.State),
	}, true
}

func composeProjectName(item Summary) string {
	return firstNonEmpty(strings.TrimSpace(item.Orchestrator.Project), strings.TrimSpace(item.ComposeProject))
}

func composeServiceName(item Summary) string {
	return firstNonEmpty(strings.TrimSpace(item.Orchestrator.Service), strings.TrimSpace(item.ComposeService))
}

func runtimeCandidateGroupKey(hostScope string, item Summary) string {
	workingDirectory, _, configFiles := resolveRuntimeCandidateMetadata(item)
	if len(configFiles) > 0 {
		return hostScope + "|config|" + configFilesDigest(runtimeCandidateIdentityConfigFiles(workingDirectory, configFiles))
	}
	projectName := strings.ToLower(composeProjectName(item))
	switch {
	case projectName != "" && workingDirectory != "":
		return hostScope + "|project|" + projectName + "|" + filepath.Clean(workingDirectory)
	case projectName != "":
		return hostScope + "|project|" + projectName
	case workingDirectory != "":
		return hostScope + "|dir|" + filepath.Clean(workingDirectory)
	default:
		return hostScope + "|member|" + strings.TrimSpace(item.ID)
	}
}

func runtimeCandidateGroupKeyFromCandidate(hostScope string, candidate moduleapi.ContainerProjectRuntimeCandidate) string {
	workingDirectory := strings.TrimSpace(candidate.WorkingDirectory)
	configFiles := normalizedStringSlice(candidate.ConfigFiles)
	if len(configFiles) > 0 {
		return hostScope + "|config|" + configFilesDigest(runtimeCandidateIdentityConfigFiles(workingDirectory, configFiles))
	}
	projectName := strings.ToLower(strings.TrimSpace(candidate.CanonicalProjectName))
	switch {
	case projectName != "" && workingDirectory != "":
		return hostScope + "|project|" + projectName + "|" + filepath.Clean(workingDirectory)
	case projectName != "":
		return hostScope + "|project|" + projectName
	case workingDirectory != "":
		return hostScope + "|dir|" + filepath.Clean(workingDirectory)
	default:
		return hostScope + "|candidate|" + strings.TrimSpace(candidate.CandidateKey)
	}
}

func matchesRuntimeCandidate(
	hostScope string,
	item Summary,
	candidate moduleapi.ContainerProjectRuntimeCandidate,
) bool {
	return runtimeCandidateGroupKey(hostScope, item) == runtimeCandidateGroupKeyFromCandidate(hostScope, candidate)
}

func runtimeCandidateKey(hostScope string, projectName string, runtimeType string, workingDirectory string, configFiles []string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(hostScope),
		strings.ToLower(strings.TrimSpace(projectName)),
		strings.TrimSpace(runtimeType),
		filepath.Clean(strings.TrimSpace(workingDirectory)),
		configFilesDigest(runtimeCandidateIdentityConfigFiles(workingDirectory, configFiles)),
	}, "|")))
	return "runtime_" + hex.EncodeToString(sum[:12])
}

func runtimeCandidateIdentityConfigFiles(workingDirectory string, configFiles []string) []string {
	resolved, ok := resolveConfigFilesAgainstWorkingDirectory(workingDirectory, configFiles)
	if ok {
		return resolved
	}
	return normalizedStringSlice(configFiles)
}

func configFilesDigest(configFiles []string) string {
	normalized := append([]string(nil), normalizedStringSlice(configFiles)...)
	sort.Strings(normalized)
	sum := sha256.Sum256([]byte(strings.Join(normalized, "|")))
	return hex.EncodeToString(sum[:12])
}

func configFilesWithinWorkingDirectory(workingDirectory string, configFiles []string) bool {
	workingDirectory = strings.TrimSpace(workingDirectory)
	if workingDirectory == "" {
		return false
	}
	root := filepath.Clean(workingDirectory)
	resolvedConfigFiles, ok := resolveConfigFilesAgainstWorkingDirectory(workingDirectory, configFiles)
	if !ok {
		return false
	}
	for _, absolute := range resolvedConfigFiles {
		relative, err := filepath.Rel(root, absolute)
		if err != nil || relative == "." || strings.HasPrefix(relative, "..") {
			return false
		}
	}
	return true
}

func configFilesAccessible(workingDirectory string, configFiles []string) bool {
	resolvedConfigFiles, ok := resolveConfigFilesAgainstWorkingDirectory(workingDirectory, configFiles)
	if !ok {
		return false
	}
	for _, file := range resolvedConfigFiles {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

func sameStringSlice(left []string, right []string) bool {
	left = append([]string(nil), normalizedStringSlice(left)...)
	right = append([]string(nil), normalizedStringSlice(right)...)
	if len(left) != len(right) {
		return false
	}
	sort.Strings(left)
	sort.Strings(right)
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func derivedWorkingDirectoryFromConfigFile(configFile string) (string, bool) {
	configFile = strings.TrimSpace(configFile)
	if configFile == "" || !filepath.IsAbs(configFile) {
		return "", false
	}
	return filepath.Dir(configFile), true
}

func resolveConfigFilesAgainstWorkingDirectory(workingDirectory string, configFiles []string) ([]string, bool) {
	normalized := normalizedStringSlice(configFiles)
	if len(normalized) == 0 {
		return []string{}, true
	}
	workingDirectory = strings.TrimSpace(workingDirectory)
	resolved := make([]string, 0, len(normalized))
	for _, file := range normalized {
		switch {
		case filepath.IsAbs(file):
			resolved = append(resolved, filepath.Clean(file))
		case workingDirectory == "" || !filepath.IsAbs(workingDirectory):
			return nil, false
		default:
			resolved = append(resolved, filepath.Clean(filepath.Join(workingDirectory, file)))
		}
	}
	return resolved, true
}

func nonNilStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func appendUniqueString(items []string, values ...string) []string {
	if len(values) == 0 {
		return items
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		seen[item] = struct{}{}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

var _ moduleapi.ContainerProjectRuntimeReader = containerProjectRuntimeReader{}
