package project

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

func (s *Service) runtimeImportCandidateByKey(
	ctx context.Context,
	candidateKey string,
) (moduleapi.ContainerProjectRuntimeCandidate, error) {
	if s == nil || s.runtimeReader == nil {
		return moduleapi.ContainerProjectRuntimeCandidate{}, errProjectServiceUnavailable
	}
	candidateKey = strings.TrimSpace(candidateKey)
	if candidateKey == "" {
		return moduleapi.ContainerProjectRuntimeCandidate{}, errProjectInvalidArgument
	}
	candidates, err := s.runtimeReader.ListImportCandidates(ctx, projectcontract.HostScopeLocal.String())
	if err != nil {
		return moduleapi.ContainerProjectRuntimeCandidate{}, err
	}
	for _, candidate := range candidates {
		normalizedCandidateKey := strings.TrimSpace(candidate.CandidateKey)
		if normalizedCandidateKey == candidateKey {
			candidate.CandidateKey = normalizedCandidateKey
			return candidate, nil
		}
	}
	return moduleapi.ContainerProjectRuntimeCandidate{}, errProjectInvalidArgument
}

func runtimeImportCandidateFromModuleAPI(candidate moduleapi.ContainerProjectRuntimeCandidate) RuntimeImportCandidate {
	result := RuntimeImportCandidate{
		CandidateKey:           strings.TrimSpace(candidate.CandidateKey),
		CanonicalProjectName:   candidate.CanonicalProjectName,
		Status:                 candidate.Status,
		StatusReasonCodes:      append([]string(nil), candidate.StatusReasonCodes...),
		Importable:             candidate.Importable,
		RuntimeType:            candidate.RuntimeType,
		WorkingDirectory:       candidate.WorkingDirectory,
		WorkingDirectorySource: candidate.WorkingDirectorySource,
		ConfigFiles:            append([]string(nil), candidate.ConfigFiles...),
		ServiceNames:           append([]string(nil), candidate.ServiceNames...),
		ContainerCounts: RuntimeImportContainerCounts{
			Running: candidate.ContainerCounts.Running,
			Stopped: candidate.ContainerCounts.Stopped,
			Total:   candidate.ContainerCounts.Total,
		},
		Warnings: append([]string(nil), candidate.Warnings...),
	}
	if strings.TrimSpace(candidate.RuntimeVersion) != "" {
		result.RuntimeVersion = stringPointer(candidate.RuntimeVersion)
	}
	return result
}

func candidateFromValidatedRuntimeImportCandidate(
	rawCandidate moduleapi.ContainerProjectRuntimeCandidate,
	validated RuntimeImportCandidate,
) moduleapi.ContainerProjectRuntimeCandidate {
	rawCandidate.CandidateKey = validated.CandidateKey
	rawCandidate.Status = validated.Status
	rawCandidate.StatusReasonCodes = append([]string(nil), validated.StatusReasonCodes...)
	rawCandidate.Importable = validated.Importable
	rawCandidate.Warnings = append([]string(nil), validated.Warnings...)
	return rawCandidate
}

func sortRuntimeImportCandidates(items []RuntimeImportCandidate) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Importable != items[j].Importable {
			return items[i].Importable
		}
		left := strings.ToLower(strings.TrimSpace(items[i].CanonicalProjectName))
		right := strings.ToLower(strings.TrimSpace(items[j].CanonicalProjectName))
		if left != right {
			return left < right
		}
		return items[i].CandidateKey < items[j].CandidateKey
	})
}

func dedupeRuntimeImportCandidates(items []RuntimeImportCandidate) []RuntimeImportCandidate {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]RuntimeImportCandidate, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.CandidateKey)
		if key == "" {
			result = append(result, item)
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func buildRuntimeImportCandidatesResult(
	items []RuntimeImportCandidate,
	query RuntimeImportCandidateListQuery,
) RuntimeImportCandidatesResult {
	normalizedQuery := normalizeRuntimeImportCandidateListQuery(query)
	keywordFiltered := filterRuntimeImportCandidatesByKeyword(items, normalizedQuery.Keyword)
	filterCounts := RuntimeImportCandidateFilterCounts{
		All:         len(keywordFiltered),
		Ready:       countRuntimeImportCandidatesByAvailability(keywordFiltered, runtimeImportCandidateAvailabilityReady),
		Imported:    countRuntimeImportCandidatesByAvailability(keywordFiltered, runtimeImportCandidateAvailabilityImported),
		Unavailable: countRuntimeImportCandidatesByAvailability(keywordFiltered, runtimeImportCandidateAvailabilityUnavailable),
	}
	availabilityFiltered := filterRuntimeImportCandidatesByAvailability(keywordFiltered, normalizedQuery.Availability)
	total := len(availabilityFiltered)
	page := paginateRuntimeImportCandidates(availabilityFiltered, normalizedQuery.Offset, normalizedQuery.Limit)
	return RuntimeImportCandidatesResult{
		Items:        page,
		Total:        total,
		Limit:        normalizedQuery.Limit,
		Offset:       normalizedQuery.Offset,
		FilterCounts: filterCounts,
	}
}

func normalizeRuntimeImportCandidateListQuery(query RuntimeImportCandidateListQuery) RuntimeImportCandidateListQuery {
	normalized := query
	normalized.Keyword = strings.TrimSpace(query.Keyword)
	if normalized.Limit <= 0 {
		normalized.Limit = runtimeImportCandidatesDefaultLimit
	}
	if normalized.Limit > maxProjectListLimit {
		normalized.Limit = maxProjectListLimit
	}
	if normalized.Offset < 0 {
		normalized.Offset = 0
	}
	if normalized.Availability != nil {
		value := strings.TrimSpace(string(*normalized.Availability))
		if value == "" {
			normalized.Availability = nil
		} else {
			availability := RuntimeImportCandidateAvailability(value)
			normalized.Availability = &availability
		}
	}
	return normalized
}

func filterRuntimeImportCandidatesByKeyword(items []RuntimeImportCandidate, keyword string) []RuntimeImportCandidate {
	if keyword == "" {
		return append([]RuntimeImportCandidate(nil), items...)
	}
	normalized := strings.ToLower(keyword)
	filtered := make([]RuntimeImportCandidate, 0, len(items))
	for _, candidate := range items {
		haystack := []string{
			candidate.CanonicalProjectName,
			candidate.WorkingDirectory,
			candidate.RuntimeType,
			strings.TrimSpace(stringValue(candidate.RuntimeVersion)),
			strings.Join(candidate.ConfigFiles, " "),
			strings.Join(candidate.ServiceNames, " "),
			strings.Join(candidate.StatusReasonCodes, " "),
			strings.Join(candidate.Warnings, " "),
		}
		if strings.Contains(strings.ToLower(strings.Join(haystack, " ")), normalized) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func filterRuntimeImportCandidatesByAvailability(
	items []RuntimeImportCandidate,
	availability *RuntimeImportCandidateAvailability,
) []RuntimeImportCandidate {
	if availability == nil {
		return append([]RuntimeImportCandidate(nil), items...)
	}
	filtered := make([]RuntimeImportCandidate, 0, len(items))
	for _, candidate := range items {
		if runtimeImportCandidateMatchesAvailability(candidate, *availability) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func countRuntimeImportCandidatesByAvailability(
	items []RuntimeImportCandidate,
	availability RuntimeImportCandidateAvailability,
) int {
	count := 0
	for _, candidate := range items {
		if runtimeImportCandidateMatchesAvailability(candidate, availability) {
			count++
		}
	}
	return count
}

func runtimeImportCandidateMatchesAvailability(
	candidate RuntimeImportCandidate,
	availability RuntimeImportCandidateAvailability,
) bool {
	ready := candidate.Importable && candidate.Status == importRuntimeCandidateStatusReady
	if availability == runtimeImportCandidateAvailabilityReady {
		return ready
	}
	if availability == runtimeImportCandidateAvailabilityImported {
		return candidate.Status == importRuntimeCandidateStatusAlreadyImported
	}
	if availability == runtimeImportCandidateAvailabilityUnavailable {
		return !ready && candidate.Status != importRuntimeCandidateStatusAlreadyImported
	}
	return true
}

func paginateRuntimeImportCandidates(items []RuntimeImportCandidate, offset int, limit int) []RuntimeImportCandidate {
	if offset >= len(items) {
		return []RuntimeImportCandidate{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]RuntimeImportCandidate(nil), items[offset:end]...)
}

func (s *Service) validatedRuntimeImportCandidate(
	rawCandidate moduleapi.ContainerProjectRuntimeCandidate,
) (RuntimeImportCandidate, error) {
	candidate := runtimeImportCandidateFromModuleAPI(rawCandidate)
	if !rawCandidate.Importable || rawCandidate.Status != importRuntimeCandidateStatusReady {
		return candidate, nil
	}
	validation, err := s.runtimeImportCandidateParseValidation(rawCandidate)
	if err != nil {
		return RuntimeImportCandidate{}, err
	}
	if validation.reason != "" {
		return markBrokenRuntimeImportCandidate(candidate, validation.reason), nil
	}
	if len(validation.warnings) > 0 {
		candidate.Warnings = uniqueStrings(append(candidate.Warnings, validation.warnings...))
	}
	return candidate, nil
}

type runtimeCandidateValidation struct {
	reason   string
	warnings []string
}

type importInspectionPreview struct {
	inspectionID               string
	resolvedWorkingDirectory   string
	canonicalProjectName       string
	canonicalProjectNameSource string
	displayNameSuggested       string
	composeFiles               []FileView
	envFiles                   []FileView
	serviceNames               []string
	networkNames               []string
	volumeNames                []string
	configHash                 string
	warnings                   []string
	conflicts                  []string
	validationStatus           string
}

func (s *Service) runtimeImportCandidateParseValidation(
	rawCandidate moduleapi.ContainerProjectRuntimeCandidate,
) (runtimeCandidateValidation, error) {
	envFiles, err := discoverEnvFilesForWorkingDirectory(rawCandidate.WorkingDirectory)
	if err != nil {
		return runtimeCandidateValidation{reason: importRuntimeReasonConfigFilesNotAccessible}, nil
	}
	_, validation, parseErr := s.parseImportRequest(ImportRequest{
		WorkingDirectory: rawCandidate.WorkingDirectory,
		ComposeFiles:     append([]string(nil), rawCandidate.ConfigFiles...),
		EnvFiles:         envFiles,
	})
	if parseErr != nil {
		return runtimeCandidateValidation{reason: runtimeCandidateParseReason(parseErr)}, nil
	}
	return runtimeCandidateValidation{warnings: append([]string(nil), validation.Warnings...)}, nil
}

func (s *Service) inspectRuntimeCandidateSession(
	ctx context.Context,
	repository projectstore.Repository,
	candidate moduleapi.ContainerProjectRuntimeCandidate,
	request RuntimeImportInspectRequest,
) (importInspectionSession, error) {
	envFiles, err := discoverEnvFilesForWorkingDirectory(candidate.WorkingDirectory)
	if err != nil {
		return importInspectionSession{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	session, err := s.inspectImportRequest(ctx, repository, ImportRequest{
		WorkingDirectory:             candidate.WorkingDirectory,
		ComposeFiles:                 append([]string(nil), candidate.ConfigFiles...),
		EnvFiles:                     envFiles,
		DisplayName:                  request.DisplayName,
		CanonicalProjectNameOverride: request.CanonicalProjectNameOverride,
	})
	if err != nil {
		return importInspectionSession{}, err
	}
	session.CandidateKey = candidate.CandidateKey
	session.CandidateKey = strings.TrimSpace(session.CandidateKey)
	session.Warnings = uniqueStrings(append(session.Warnings, candidate.Warnings...))
	if s.inspectCache != nil {
		s.inspectCache.storeSession(session)
	}
	return session, nil
}

func (s *Service) runtimeImportCandidateMembers(
	ctx context.Context,
	candidate moduleapi.ContainerProjectRuntimeCandidate,
) ([]RuntimeImportMember, error) {
	if s == nil || s.runtimeReader == nil {
		return nil, errProjectServiceUnavailable
	}
	items, err := s.runtimeReader.ListImportCandidateMembers(ctx, projectcontract.HostScopeLocal.String(), candidate)
	if err != nil {
		return nil, err
	}
	members := make([]RuntimeImportMember, 0, len(items))
	for _, item := range items {
		members = append(members, RuntimeImportMember{
			ContainerID:   item.ContainerID,
			ContainerName: item.ContainerName,
			ServiceName:   item.ServiceName,
			State:         item.CanonicalState,
		})
	}
	return members, nil
}

func importInspectResultFromSession(
	directoryRef ImportDirectoryReference,
	session importInspectionSession,
) ImportInspectResult {
	preview := inspectionPreviewFromSession(session)
	return ImportInspectResult{
		InspectionID:               preview.inspectionID,
		DirectoryRef:               directoryRef,
		ResolvedWorkingDirectory:   preview.resolvedWorkingDirectory,
		CanonicalProjectName:       preview.canonicalProjectName,
		CanonicalProjectNameSource: preview.canonicalProjectNameSource,
		DisplayNameSuggested:       preview.displayNameSuggested,
		ComposeFiles:               preview.composeFiles,
		EnvFiles:                   preview.envFiles,
		ServiceNames:               preview.serviceNames,
		NetworkNames:               preview.networkNames,
		VolumeNames:                preview.volumeNames,
		ConfigHash:                 preview.configHash,
		Warnings:                   preview.warnings,
		Conflicts:                  preview.conflicts,
		ValidationStatus:           preview.validationStatus,
	}
}

// runtimeImportInspectResultFromSession 根据检查会话和运行时成员构建运行时导入检查结果，并初始化默认生命周期配置。
func runtimeImportInspectResultFromSession(
	candidateKey string,
	session importInspectionSession,
	runtimeMembers []RuntimeImportMember,
) RuntimeImportInspectResult {
	preview := inspectionPreviewFromSession(session)
	members := append([]RuntimeImportMember(nil), runtimeMembers...)
	return RuntimeImportInspectResult{
		InspectionID:               preview.inspectionID,
		CandidateKey:               candidateKey,
		ResolvedWorkingDirectory:   preview.resolvedWorkingDirectory,
		CanonicalProjectName:       preview.canonicalProjectName,
		CanonicalProjectNameSource: preview.canonicalProjectNameSource,
		DisplayNameSuggested:       preview.displayNameSuggested,
		ComposeFiles:               preview.composeFiles,
		EnvFiles:                   preview.envFiles,
		ServiceNames:               preview.serviceNames,
		NetworkResources:           buildRuntimeImportNetworkResources(session.ParseResult, members),
		VolumeResources:            buildRuntimeImportVolumeResources(session.ParseResult, members),
		RuntimeMembers:             members,
		ConfigHash:                 preview.configHash,
		Warnings:                   preview.warnings,
		Conflicts:                  preview.conflicts,
		ValidationStatus:           preview.validationStatus,
		LifecycleConfiguration:     defaultLifecycleStandardConfig(),
	}
}

func inspectionValidationStatus(session importInspectionSession) string {
	if len(session.Conflicts) > 0 {
		return "conflict"
	}
	return "ready"
}

func inspectionPreviewFromSession(session importInspectionSession) importInspectionPreview {
	return importInspectionPreview{
		inspectionID:               session.ID,
		resolvedWorkingDirectory:   session.WorkingDir,
		canonicalProjectName:       session.CanonicalName,
		canonicalProjectNameSource: session.CanonicalSource,
		displayNameSuggested:       session.DisplayName,
		composeFiles:               toFileViews(session.ParseResult.ComposeFiles),
		envFiles:                   toFileViews(session.ParseResult.EnvFiles),
		serviceNames:               append([]string(nil), session.ParseResult.ServiceNames...),
		networkNames:               append([]string(nil), session.ParseResult.NetworkNames...),
		volumeNames:                append([]string(nil), session.ParseResult.VolumeNames...),
		configHash:                 session.ParseResult.ConfigHash,
		warnings:                   append([]string(nil), session.Warnings...),
		conflicts:                  append([]string(nil), session.Conflicts...),
		validationStatus:           inspectionValidationStatus(session),
	}
}

func markBrokenRuntimeImportCandidate(candidate RuntimeImportCandidate, reason string) RuntimeImportCandidate {
	candidate.Status = importRuntimeCandidateStatusBrokenCompose
	candidate.Importable = false
	candidate.StatusReasonCodes = uniqueStrings(append([]string(nil), reason))
	return candidate
}

func markAlreadyImportedRuntimeImportCandidate(candidate RuntimeImportCandidate, reason string) RuntimeImportCandidate {
	candidate.Status = importRuntimeCandidateStatusAlreadyImported
	candidate.Importable = false
	if strings.TrimSpace(reason) == "" {
		reason = importRuntimeReasonAlreadyImported
	}
	candidate.StatusReasonCodes = uniqueStrings(append([]string(nil), reason))
	return candidate
}

func runtimeImportCandidateExistingConflict(
	candidate RuntimeImportCandidate,
	existing []projectstore.ProjectAggregate,
) string {
	targetWD := strings.TrimSpace(candidate.WorkingDirectory)
	targetCanonical := strings.TrimSpace(candidate.CanonicalProjectName)
	for _, item := range existing {
		if sameWorkingDirectory(targetWD, item.Project.WorkingDirectory) {
			return importRuntimeReasonAlreadyImported
		}
		if targetCanonical != "" && strings.EqualFold(item.Project.CanonicalProjectName, targetCanonical) {
			return importRuntimeReasonAlreadyImported
		}
	}
	return ""
}

func runtimeCandidateParseReason(err error) string {
	if err == nil {
		return importRuntimeReasonComposeParseFailed
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "read project file"),
		strings.Contains(text, "permission denied"),
		strings.Contains(text, "no such file or directory"):
		return importRuntimeReasonConfigFilesNotAccessible
	default:
		return importRuntimeReasonComposeParseFailed
	}
}
