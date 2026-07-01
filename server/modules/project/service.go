package project

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type managedRootFS struct {
	root    *os.Root
	rootDir string
}

var (
	errProjectServiceUnavailable   = errors.New("project service is unavailable")
	errProjectInvalidArgument      = errors.New("project invalid argument")
	errProjectNotFound             = errors.New("project not found")
	errProjectConflict             = errors.New("project conflict")
	errProjectImportValidation     = errors.New("project import validation failed")
	errProjectUnsupportedLifecycle = errors.New("project lifecycle is unsupported")
	errProjectFileNotFound         = errors.New("project file not found")
	errProjectDestroyBlocked       = errors.New("project destroy blocked by ownership guard")
	errProjectManagedFlow          = errors.New("project managed flow is unsupported")
)

const (
	defaultProjectListLimit  = 20
	maxProjectListLimit      = 100
	projectConflictScanSize  = 100
	minLifecycleArgCount     = 2
	maxCommandOutputSummary  = 120
	managedCreateWarningsCap = 2
	draftWarningsCap         = 2
	managedCreateDirMode     = 0o750
	managedCreateFileMode    = 0o600
)

// ListQuery describes project list filters.
type ListQuery struct {
	Limit             int
	Offset            int
	SourceKind        string
	DriftStatus       string
	LastRefreshStatus string
}

// ImportRequest describes batch-2 import validate and import payloads.
type ImportRequest struct {
	WorkingDirectory             string
	DisplayName                  *string
	ComposeFiles                 []string
	EnvFiles                     []string
	CanonicalProjectNameOverride *string
	ActorID                      *uint64
}

// ListResult returns a paginated project list.
type ListResult struct {
	Items  []generated.ProjectListItem
	Total  int
	Limit  int
	Offset int
}

// ImportValidationResult returns the static import validation result.
type ImportValidationResult struct {
	CanonicalProjectName       string
	CanonicalProjectNameSource string
	WorkingDirectory           string
	ComposeFiles               []generated.ProjectFileItem
	EnvFiles                   []generated.ProjectFileItem
	ServiceCount               int
	Warnings                   []string
	Conflicts                  []string
	ConfigHash                 string
	DeclaredServiceNames       []string
}

// ConfigurationMetadataResult returns readonly configuration metadata.
type ConfigurationMetadataResult struct {
	ProjectID          uint64
	ComposeFiles       []generated.ProjectFileItem
	EnvFiles           []generated.ProjectFileItem
	OwnershipMode      string
	DriftStatus        string
	LastRefreshStatus  string
	LastRefreshAt      *time.Time
	DiagnosticsSummary []string
}

// ConfigurationPreviewResult returns readonly normalized compose preview.
type ConfigurationPreviewResult struct {
	ProjectID             uint64
	CanonicalProjectName  string
	ConfigHash            string
	NormalizedComposeYAML string
	RefreshedAt           *time.Time
}

// ConfigurationFileResult returns readonly raw file content.
type ConfigurationFileResult struct {
	FileID       uint64
	Kind         string
	Path         string
	Content      string
	DownloadName string
}

// ConfigurationDraft describes one Phase 2.4 managed configuration draft.
type ConfigurationDraft struct {
	ComposeFileContent string
	EnvFileContent     *string
}

// ConfigurationDiffFile describes one file-level diff projection.
type ConfigurationDiffFile struct {
	Kind            string
	Path            string
	Changed         bool
	CurrentHash     string
	ProposedHash    string
	CurrentContent  string
	ProposedContent string
}

// ConfigurationDiffResult returns bounded managed draft diff output.
type ConfigurationDiffResult struct {
	ProjectID            uint64
	CanonicalProjectName string
	OwnershipMode        string
	CurrentConfigHash    string
	ProposedConfigHash   string
	HasChanges           bool
	Files                []ConfigurationDiffFile
	Warnings             []string
}

// ConfigurationValidateResult returns bounded managed draft validation output.
type ConfigurationValidateResult struct {
	ProjectID             uint64
	CanonicalProjectName  string
	OwnershipMode         string
	ProposedConfigHash    string
	NormalizedComposeYAML string
	DeclaredServiceNames  []string
	Warnings              []string
}

// DeployResult returns bounded managed deploy output.
type DeployResult struct {
	ProjectID            uint64
	Action               string
	Result               string
	CanonicalProjectName string
	OwnershipMode        string
	ConfigHash           string
	RefreshedAt          time.Time
	DeclaredServiceCount int
	MessageKey           *string
	Message              *string
	GuardResults         []GuardResult
}

type preparedConfigurationDraft struct {
	Proposal    managedDraftProposal
	ParseResult projectcompose.Result
	Warnings    []string
}

// ActionResult returns bounded phase-1 action status.
type ActionResult struct {
	ProjectID    uint64
	Action       generated.ProjectActionResponseAction
	Result       generated.ProjectActionResponseResult
	MessageKey   *string
	Message      *string
	GuardResults []GuardResult
}

// GuardResult is the stable structured contract for blocked/guarded project actions.
type GuardResult struct {
	Code       string
	MessageKey *string
	Detail     *string
}

// DestroyRequest describes guarded destroy options.
type DestroyRequest struct {
	RemoveNamedVolumes          bool
	DeleteWorkingDirectory      bool
	ConfirmCanonicalProjectName string
	ActorID                     *uint64
}

// ManagedRootInfo returns bounded managed-root contract metadata.
type ManagedRootInfo struct {
	Status                  string
	ConfigKey               string
	ConfiguredRootDirectory *string
	OwnershipMode           string
	CreatePermission        string
	SupportsManagedCreate   bool
	StatusReason            *string
}

// ManagedProjectCreateRequest describes Phase 2 managed-create contract payloads.
type ManagedProjectCreateRequest struct {
	DisplayName              string
	CanonicalProjectName     string
	RelativeProjectDirectory string
	ComposeFileName          string
	ComposeFileContent       string
	EnvFileName              *string
	EnvFileContent           *string
}

// ManagedProjectCreateValidationResult returns create-contract validation metadata without writing files.
type ManagedProjectCreateValidationResult struct {
	ManagedRoot             ManagedRootInfo
	DisplayName             string
	CanonicalProjectName    string
	OwnershipMode           string
	WorkingDirectory        string
	ComposeFileName         string
	EnvFileName             *string
	ComposeFileAbsolutePath string
	EnvFileAbsolutePath     *string
	Warnings                []string
}

// ManagedProjectCreateResult returns the created managed project bootstrap after write + persist.
type ManagedProjectCreateResult struct {
	Validation           ManagedProjectCreateValidationResult
	ProjectID            uint64
	ConfigHash           string
	DeclaredServiceCount int
	RefreshedAt          time.Time
}

// Service owns project registry, import, and readonly refresh/configuration use cases.
type Service struct {
	repository     projectstore.Repository
	runtimeReader  moduleapi.ContainerProjectRuntimeReader
	configResolver moduleapi.SystemConfigResolver
}

// NewService 创建项目服务边界并应用可选配置。
// 当 repository 为空时返回错误。
func NewService(repository projectstore.Repository, options ...ServiceOption) (*Service, error) {
	if repository == nil {
		return nil, errors.New("project repository is unavailable")
	}
	service := &Service{
		repository: repository,
	}
	for _, option := range options {
		if option != nil {
			option.apply(service)
		}
	}
	return service, nil
}

// ServiceOption customizes project service dependencies.
type ServiceOption interface{ apply(*Service) }

type serviceOptionFunc func(*Service)

func (f serviceOptionFunc) apply(s *Service) { f(s) }

// WithRuntimeReader 设置容器运行时聚合读取器。
// 用于提供项目成员运行态汇总所需的运行时边界。
func WithRuntimeReader(reader moduleapi.ContainerProjectRuntimeReader) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.runtimeReader = reader
	})
}

// WithSystemConfigResolver 注入用于 managed-create 权限校验的系统配置读取边界。
func WithSystemConfigResolver(resolver moduleapi.SystemConfigResolver) ServiceOption {
	return serviceOptionFunc(func(s *Service) {
		s.configResolver = resolver
	})
}

// SetRuntimeReader injects the runtime reader after module registration resolves cross-module services.
func (s *Service) SetRuntimeReader(reader moduleapi.ContainerProjectRuntimeReader) {
	if s == nil {
		return
	}
	s.runtimeReader = reader
}

// SetSystemConfigResolver injects the system-config resolver after module registration.
func (s *Service) SetSystemConfigResolver(resolver moduleapi.SystemConfigResolver) {
	if s == nil {
		return
	}
	s.configResolver = resolver
}

// List returns one page of registered projects.
func (s *Service) List(ctx context.Context, query ListQuery) (ListResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ListResult{}, err
	}
	storeResult, err := repository.List(ctx, projectstore.ListQuery{
		Limit:             query.Limit,
		Offset:            query.Offset,
		SourceKind:        strings.TrimSpace(query.SourceKind),
		DriftStatus:       strings.TrimSpace(query.DriftStatus),
		LastRefreshStatus: strings.TrimSpace(query.LastRefreshStatus),
	})
	if err != nil {
		return ListResult{}, mapStoreError(err)
	}
	items := make([]generated.ProjectListItem, 0, len(storeResult.Items))
	for _, item := range storeResult.Items {
		runtimeSummary, _ := s.runtimeSummary(ctx, item)
		items = append(items, toProjectListItem(item, runtimeSummary))
	}
	return ListResult{Items: items, Total: storeResult.Total, Limit: normalizeListLimit(query.Limit), Offset: maxInt(query.Offset, 0)}, nil
}

// Get returns one project detail payload.
func (s *Service) Get(ctx context.Context, projectID uint64) (generated.ProjectDetailResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ProjectDetailResponse{}, err
	}
	runtimeSummary, _ := s.runtimeSummary(ctx, aggregate)
	return toProjectDetailResponse(aggregate, runtimeSummary), nil
}

// ValidateImport resolves static compose inputs and reports bounded import validation results.
func (s *Service) ValidateImport(ctx context.Context, request ImportRequest) (ImportValidationResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ImportValidationResult{}, err
	}
	parseResult, validation, err := s.parseImportRequest(request)
	if err != nil {
		return ImportValidationResult{}, err
	}
	conflicts, err := s.computeConflicts(ctx, repository, request, validation)
	if err != nil {
		return ImportValidationResult{}, err
	}
	return ImportValidationResult{
		CanonicalProjectName:       validation.CanonicalProjectName,
		CanonicalProjectNameSource: validation.CanonicalProjectNameSource,
		WorkingDirectory:           parseResult.WorkingDirectory,
		ComposeFiles:               validation.ComposeFiles,
		EnvFiles:                   validation.EnvFiles,
		ServiceCount:               len(parseResult.ServiceNames),
		Warnings:                   append([]string(nil), parseResult.Warnings...),
		Conflicts:                  conflicts,
		ConfigHash:                 parseResult.ConfigHash,
		DeclaredServiceNames:       append([]string(nil), parseResult.ServiceNames...),
	}, nil
}

// Import validates and registers one project.
func (s *Service) Import(ctx context.Context, request ImportRequest) (generated.ProjectImportResponse, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	parseResult, validation, err := s.parseImportRequest(request)
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	conflicts, err := s.computeConflicts(ctx, repository, request, validation)
	if err != nil {
		return generated.ProjectImportResponse{}, err
	}
	if len(conflicts) > 0 {
		return generated.ProjectImportResponse{}, fmt.Errorf("%w: %s", errProjectConflict, strings.Join(conflicts, ", "))
	}

	now := time.Now().UTC()
	aggregate, err := repository.ImportProject(ctx, projectstore.ImportProjectInput{
		DisplayName:                displayNameOrCanonical(request.DisplayName, validation.CanonicalProjectName),
		CanonicalProjectName:       validation.CanonicalProjectName,
		CanonicalProjectNameSource: validation.CanonicalProjectNameSource,
		SourceKind:                 projectcontract.SourceKindImported.String(),
		HostScope:                  projectcontract.HostScopeLocal.String(),
		WorkingDirectory:           parseResult.WorkingDirectory,
		OwnershipMode:              projectcontract.OwnershipModeExternal.String(),
		LastRefreshStatus:          projectcontract.RefreshStatusSuccess.String(),
		LastRefreshAt:              &now,
		LastRefreshConfigHash:      parseResult.ConfigHash,
		LastObservedConfigHash:     parseResult.ConfigHash,
		LastDriftCheckedAt:         &now,
		DriftStatus:                projectcontract.DriftStatusClean.String(),
		Files:                      toStoreFiles(parseResult.ComposeFiles, parseResult.EnvFiles),
		Snapshot: &projectstore.Snapshot{
			ConfigHash:             parseResult.ConfigHash,
			NormalizedComposeJSON:  normalizeSnapshotJSON(parseResult.NormalizedComposeJSON),
			DeclaredServiceCount:   len(parseResult.ServiceNames),
			DeclaredServicesDigest: digestServiceNames(parseResult.ServiceNames),
			RefreshedAt:            now,
		},
		ActorID: request.ActorID,
	})
	if err != nil {
		return generated.ProjectImportResponse{}, mapStoreError(err)
	}

	response := generated.ProjectImportResponse{
		Project: toProjectDetailResponse(aggregate),
	}
	response.SnapshotSummary.ConfigHash = parseResult.ConfigHash
	response.SnapshotSummary.RefreshedAt = now
	serviceCount := len(parseResult.ServiceNames)
	response.SnapshotSummary.DeclaredServiceCount = &serviceCount
	return response, nil
}

// Refresh reparses and persists the latest static compose snapshot.
func (s *Service) Refresh(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	aggregate, err := repository.Get(ctx, projectID)
	if err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	request := ImportRequest{
		WorkingDirectory: aggregate.Project.WorkingDirectory,
		DisplayName:      stringPointer(aggregate.Project.DisplayName),
		ComposeFiles:     collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		EnvFiles:         collectFilesByKind(aggregate.Files, projectcontract.FileKindEnv.String()),
		ActorID:          actorID,
	}
	parseResult, _, err := s.parseImportRequest(request)
	if err != nil {
		return ActionResult{}, err
	}
	now := time.Now().UTC()
	updated, err := repository.RefreshProject(ctx, buildRefreshProjectInput(projectID, parseResult, now, actorID))
	if err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	_ = updated
	return ActionResult{
		ProjectID:  projectID,
		Action:     generated.ProjectActionRefresh,
		Result:     generated.ProjectActionResultCompleted,
		MessageKey: stringPointer(projectcontract.ProjectRefreshCompleted.String()),
		Message:    stringPointer(projectcontract.ProjectRefreshCompleted.String()),
	}, nil
}

// Services returns static service projections plus empty runtime members for batch 2.
func (s *Service) Services(ctx context.Context, projectID uint64) (generated.ProjectServicesResponse, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return generated.ProjectServicesResponse{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return generated.ProjectServicesResponse{}, err
	}
	runtimeSummary, _ := s.runtimeSummary(ctx, aggregate)
	serviceMembers := membersByService(runtimeSummary.Members)
	items := make([]generated.ProjectServiceItem, 0, len(parseResult.Services))
	for _, item := range parseResult.Services {
		members := serviceMembers[item.ServiceName]
		generatedItem := generated.ProjectServiceItem{
			ServiceName: item.ServiceName,
		}
		applyGeneratedServiceMembers(&generatedItem, members)
		if item.Image != nil {
			generatedItem.Image = item.Image
		}
		if item.BuildContext != nil {
			generatedItem.BuildContext = item.BuildContext
		}
		if len(item.DeclaredPorts) > 0 {
			ports := append([]string(nil), item.DeclaredPorts...)
			generatedItem.DeclaredPorts = &ports
		}
		if len(item.DeclaredVolumes) > 0 {
			volumes := append([]string(nil), item.DeclaredVolumes...)
			generatedItem.DeclaredVolumes = &volumes
		}
		if len(item.DeclaredNetworks) > 0 {
			networks := append([]string(nil), item.DeclaredNetworks...)
			generatedItem.DeclaredNetworks = &networks
		}
		items = append(items, generatedItem)
	}
	return generated.ProjectServicesResponse{
		CanonicalProjectName: aggregate.Project.CanonicalProjectName,
		Items:                items,
		ProjectId:            mustGeneratedID(projectID),
	}, nil
}

// ManagedRoot reports the canonical managed-root authority for future managed-create flows.
func (s *Service) ManagedRoot(ctx context.Context) (ManagedRootInfo, error) {
	definitionKey := projectcontract.ProjectManagedRootConfig.String()
	info := ManagedRootInfo{
		Status:                projectcontract.ManagedRootStatusUnconfigured.String(),
		ConfigKey:             definitionKey,
		OwnershipMode:         projectcontract.OwnershipModeManagedRootDedicated.String(),
		CreatePermission:      projectcontract.ProjectCreatePermission.String(),
		SupportsManagedCreate: false,
	}
	if s.configResolver == nil {
		reason := "system config resolver unavailable"
		info.Status = projectcontract.ManagedRootStatusInvalid.String()
		info.StatusReason = &reason
		return info, nil
	}

	raw, err := s.configResolver.ResolveDefaultConfig(ctx, definitionKey)
	if err != nil {
		reason := "managed root config definition is unavailable"
		info.Status = projectcontract.ManagedRootStatusInvalid.String()
		info.StatusReason = &reason
		return info, nil
	}

	var root string
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		reason := "managed root config default value is not a string"
		info.Status = projectcontract.ManagedRootStatusInvalid.String()
		info.StatusReason = &reason
		return info, nil
	}
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" || root == "." {
		reason := "managed root directory is not configured"
		info.StatusReason = &reason
		return info, nil
	}
	if !filepath.IsAbs(root) {
		reason := "managed root directory must be absolute"
		info.Status = projectcontract.ManagedRootStatusInvalid.String()
		info.ConfiguredRootDirectory = stringPointer(root)
		info.StatusReason = &reason
		return info, nil
	}
	info.Status = projectcontract.ManagedRootStatusReady.String()
	info.SupportsManagedCreate = true
	info.ConfiguredRootDirectory = stringPointer(root)
	return info, nil
}

// ValidateManagedCreate resolves bounded managed-create paths and naming rules without writing files.
func (s *Service) ValidateManagedCreate(ctx context.Context, request ManagedProjectCreateRequest) (ManagedProjectCreateValidationResult, error) {
	rootInfo, err := s.ManagedRoot(ctx)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	if rootInfo.Status != projectcontract.ManagedRootStatusReady.String() || rootInfo.ConfiguredRootDirectory == nil {
		return ManagedProjectCreateValidationResult{}, fmt.Errorf("%w: %s", errProjectInvalidArgument, projectcontract.ProjectManagedRootUnconfigured.String())
	}

	normalized, err := normalizeManagedCreateRequest(request)
	if err != nil {
		return ManagedProjectCreateValidationResult{}, err
	}
	workingDirectory := filepath.Join(*rootInfo.ConfiguredRootDirectory, normalized.RelativeProjectDirectory)
	composeFileAbsolutePath := filepath.Join(workingDirectory, normalized.ComposeFileName)
	envFileAbsolutePath := managedCreateEnvAbsolutePath(workingDirectory, normalized.EnvFileName)
	warnings := make([]string, 0, managedCreateWarningsCap)
	warnings = append(warnings, "Managed create validation checks authority, normalized names, and target paths before any file-write execution.")
	if normalized.EnvFileName == nil {
		warnings = append(warnings, "No env file is declared; create execution will only materialize the compose file.")
	}

	return ManagedProjectCreateValidationResult{
		ManagedRoot:             rootInfo,
		DisplayName:             normalized.DisplayName,
		CanonicalProjectName:    normalized.CanonicalProjectName,
		OwnershipMode:           projectcontract.OwnershipModeManagedRootDedicated.String(),
		WorkingDirectory:        workingDirectory,
		ComposeFileName:         normalized.ComposeFileName,
		EnvFileName:             normalized.EnvFileName,
		ComposeFileAbsolutePath: composeFileAbsolutePath,
		EnvFileAbsolutePath:     envFileAbsolutePath,
		Warnings:                warnings,
	}, nil
}

// CreateManagedProject writes managed project files under the configured managed root and persists the registry bootstrap.
func (s *Service) CreateManagedProject(ctx context.Context, request ManagedProjectCreateRequest, actorID *uint64) (ManagedProjectCreateResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	validation, err := s.ValidateManagedCreate(ctx, request)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	normalized, err := normalizeManagedCreateRequest(request)
	if err != nil {
		return ManagedProjectCreateResult{}, err
	}
	if err := ensureManagedCreatePathsUnderRoot(validation); err != nil {
		return ManagedProjectCreateResult{}, err
	}

	createdDir, createdFiles, err := writeManagedProjectFiles(validation, normalized)
	if err != nil {
		return ManagedProjectCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	shouldCleanup := true
	defer func() {
		if shouldCleanup {
			cleanupManagedCreate(createdDir, createdFiles)
		}
	}()

	parseResult, err := projectcompose.Load(projectcompose.Input{
		WorkingDirectory: validation.WorkingDirectory,
		ComposeFiles:     []string{validation.ComposeFileAbsolutePath},
		EnvFiles:         managedCreateEnvFileList(validation.EnvFileAbsolutePath),
	})
	if err != nil {
		return ManagedProjectCreateResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}

	now := time.Now().UTC()
	aggregate, err := repository.ImportProject(ctx, projectstore.ImportProjectInput{
		DisplayName:                normalized.DisplayName,
		CanonicalProjectName:       normalized.CanonicalProjectName,
		CanonicalProjectNameSource: projectcontract.CanonicalProjectNameSourceOverride.String(),
		SourceKind:                 projectcontract.SourceKindManaged.String(),
		HostScope:                  projectcontract.HostScopeLocal.String(),
		WorkingDirectory:           validation.WorkingDirectory,
		OwnershipMode:              projectcontract.OwnershipModeManagedRootDedicated.String(),
		LastRefreshStatus:          projectcontract.RefreshStatusSuccess.String(),
		LastRefreshAt:              &now,
		LastRefreshConfigHash:      parseResult.ConfigHash,
		LastObservedConfigHash:     parseResult.ConfigHash,
		LastDriftCheckedAt:         &now,
		DriftStatus:                projectcontract.DriftStatusClean.String(),
		Files:                      toStoreFiles(parseResult.ComposeFiles, parseResult.EnvFiles),
		Snapshot: &projectstore.Snapshot{
			ConfigHash:             parseResult.ConfigHash,
			NormalizedComposeJSON:  normalizeSnapshotJSON(parseResult.NormalizedComposeJSON),
			DeclaredServiceCount:   len(parseResult.ServiceNames),
			DeclaredServicesDigest: digestServiceNames(parseResult.ServiceNames),
			RefreshedAt:            now,
		},
		ActorID: actorID,
	})
	if err != nil {
		return ManagedProjectCreateResult{}, mapStoreError(err)
	}
	shouldCleanup = false

	return ManagedProjectCreateResult{
		Validation:           validation,
		ProjectID:            aggregate.Project.ID,
		ConfigHash:           parseResult.ConfigHash,
		DeclaredServiceCount: len(parseResult.ServiceNames),
		RefreshedAt:          now,
	}, nil
}

type normalizedManagedCreateRequest struct {
	DisplayName              string
	CanonicalProjectName     string
	RelativeProjectDirectory string
	ComposeFileName          string
	ComposeFileContent       string
	EnvFileName              *string
	EnvFileContent           *string
}

// normalizeManagedCreateRequest 规范化受控创建请求并校验必填字段。
// 它会修剪显示名、规范名、compose 内容和文件名，校验相对目录、compose 文件名以及可选 env 文件信息，并在必填项缺失时返回错误。
func normalizeManagedCreateRequest(request ManagedProjectCreateRequest) (normalizedManagedCreateRequest, error) {
	displayName := strings.TrimSpace(request.DisplayName)
	canonicalName := strings.TrimSpace(request.CanonicalProjectName)
	composeFileContent := strings.TrimSpace(request.ComposeFileContent)
	relativeDir, err := normalizeManagedRelativeDirectory(request.RelativeProjectDirectory)
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	composeFileName, err := normalizeManagedFileName(request.ComposeFileName, "compose")
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	if displayName == "" || canonicalName == "" || composeFileContent == "" {
		return normalizedManagedCreateRequest{}, fmt.Errorf("%w: missing required managed-create fields", errProjectInvalidArgument)
	}
	envFileName, err := normalizeManagedOptionalFileName(request.EnvFileName, "env")
	if err != nil {
		return normalizedManagedCreateRequest{}, err
	}
	envFileContent := normalizeManagedOptionalContent(request.EnvFileContent)
	if envFileName == nil {
		envFileContent = nil
	}
	return normalizedManagedCreateRequest{
		DisplayName:              displayName,
		CanonicalProjectName:     canonicalName,
		RelativeProjectDirectory: relativeDir,
		ComposeFileName:          composeFileName,
		ComposeFileContent:       composeFileContent,
		EnvFileName:              envFileName,
		EnvFileContent:           envFileContent,
	}, nil
}

// normalizeManagedRelativeDirectory 规范化并校验 managed-create 的相对项目目录。
// 它会去除首尾空白、统一路径分隔符并清理路径，同时确保目录保持在 managed root 下。
// @param value 待规范化的相对目录。
// @returns 规范化后的相对目录；当目录为空、为绝对路径或会逃逸 managed root 时返回错误。
func normalizeManagedRelativeDirectory(value string) (string, error) {
	relativeDir := strings.TrimSpace(value)
	if relativeDir == "" {
		return "", fmt.Errorf("%w: missing required managed-create fields", errProjectInvalidArgument)
	}
	if filepath.IsAbs(relativeDir) || relativeDir == "." || relativeDir == ".." {
		return "", fmt.Errorf("%w: relative project directory must stay under managed root", errProjectInvalidArgument)
	}
	if strings.Contains(relativeDir, `\`) {
		relativeDir = strings.ReplaceAll(relativeDir, `\`, "/")
	}
	relativeDir = filepath.Clean(relativeDir)
	if relativeDir == "." || strings.HasPrefix(relativeDir, "..") {
		return "", fmt.Errorf("%w: relative project directory escapes managed root", errProjectInvalidArgument)
	}
	return relativeDir, nil
}

// 它会去除首尾空白并提取最后一个路径段；当结果为空、`.` 或路径分隔符时返回错误。
func normalizeManagedFileName(value string, label string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "." || trimmed == string(filepath.Separator) {
		return "", fmt.Errorf("%w: invalid %s file name", errProjectInvalidArgument, label)
	}
	if filepath.IsAbs(trimmed) || strings.Contains(trimmed, "/") || strings.Contains(trimmed, `\`) {
		return "", fmt.Errorf("%w: invalid %s file name", errProjectInvalidArgument, label)
	}
	fileName := filepath.Base(trimmed)
	if fileName == "" || fileName == "." || fileName != trimmed {
		return "", fmt.Errorf("%w: invalid %s file name", errProjectInvalidArgument, label)
	}
	return fileName, nil
}

// normalizeManagedOptionalFileName 规范化可选文件名，空值或纯空白时返回 nil。
//
// 返回规范化后的文件名指针；当输入为空、仅包含空白字符或校验失败时分别返回 nil 或错误。
func normalizeManagedOptionalFileName(value *string, label string) (*string, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	fileName, err := normalizeManagedFileName(*value, label)
	if err != nil {
		return nil, err
	}
	return &fileName, nil
}

// managedCreateEnvAbsolutePath 生成受控项目环境文件的绝对路径。
//
// 当未提供环境文件名时返回 nil；否则返回工作目录下对应环境文件的绝对路径。
func managedCreateEnvAbsolutePath(workingDirectory string, envFileName *string) *string {
	if envFileName == nil {
		return nil
	}
	envAbs := filepath.Join(workingDirectory, *envFileName)
	return &envAbs
}

// normalizeManagedOptionalContent 去除可选内容两端的空白字符并返回结果。
//
// @param value 待规范化的内容指针。
// @returns 规范化后的内容指针；当输入为 nil 时返回 nil。
func normalizeManagedOptionalContent(value *string) *string {
	if value == nil {
		return nil
	}
	content := strings.TrimSpace(*value)
	return &content
}

// ensureManagedCreatePathsUnderRoot 验证托管项目工作目录位于已配置的 managed root 下。
// 当工作目录缺少有效根目录关系或超出 managed root 时，返回 errProjectInvalidArgument。
func ensureManagedCreatePathsUnderRoot(validation ManagedProjectCreateValidationResult) error {
	if validation.ManagedRoot.ConfiguredRootDirectory == nil {
		return errProjectInvalidArgument
	}
	root := filepath.Clean(*validation.ManagedRoot.ConfiguredRootDirectory)
	workingDirectory := filepath.Clean(validation.WorkingDirectory)
	relative, err := filepath.Rel(root, workingDirectory)
	if err != nil {
		return fmt.Errorf("%w: invalid managed root relationship", errProjectInvalidArgument)
	}
	if relative == "." || relative == "" || strings.HasPrefix(relative, "..") {
		return fmt.Errorf("%w: managed project directory must stay under managed root", errProjectInvalidArgument)
	}
	return nil
}

// writeManagedProjectFiles 在受控工作目录中写入 compose 文件和可选的 env 文件，并返回创建的目录与文件路径。
// 成功时返回清理所需的工作目录和已写入文件列表。
func writeManagedProjectFiles(validation ManagedProjectCreateValidationResult, normalized normalizedManagedCreateRequest) (string, []string, error) {
	workingDirectory := filepath.Clean(validation.WorkingDirectory)
	if err := os.MkdirAll(workingDirectory, managedCreateDirMode); err != nil {
		return "", nil, fmt.Errorf("create working directory: %w", err)
	}
	createdFiles := []string{}
	if err := os.WriteFile(validation.ComposeFileAbsolutePath, []byte(normalized.ComposeFileContent), managedCreateFileMode); err != nil {
		return workingDirectory, createdFiles, fmt.Errorf("write compose file: %w", err)
	}
	createdFiles = append(createdFiles, validation.ComposeFileAbsolutePath)
	if validation.EnvFileAbsolutePath != nil && normalized.EnvFileContent != nil {
		if err := os.WriteFile(*validation.EnvFileAbsolutePath, []byte(*normalized.EnvFileContent), managedCreateFileMode); err != nil {
			return workingDirectory, createdFiles, fmt.Errorf("write env file: %w", err)
		}
		createdFiles = append(createdFiles, *validation.EnvFileAbsolutePath)
	}
	return workingDirectory, createdFiles, nil
}

// cleanupManagedCreate 依次删除已创建的文件，并在目录路径非空时删除创建的目录。
func cleanupManagedCreate(createdDir string, createdFiles []string) {
	if len(createdFiles) == 0 && createdDir == "" {
		return
	}
	fsRoot, err := openManagedRootFSForPaths(createdDir, createdFiles...)
	if err != nil {
		return
	}
	defer fsRoot.close()
	for i := len(createdFiles) - 1; i >= 0; i-- {
		relative, relErr := fsRoot.relative(createdFiles[i])
		if relErr != nil {
			continue
		}
		_ = fsRoot.root.Remove(relative)
	}
	if createdDir != "" {
		relative, relErr := fsRoot.relative(createdDir)
		if relErr == nil {
			_ = fsRoot.root.Remove(relative)
		}
	}
}

// managedCreateEnvFileList 返回受管创建流程中的 env 文件路径列表。
//
// 当提供 env 文件绝对路径时，返回仅包含该路径的切片；否则返回 nil。
//
// @param envFileAbsolutePath env 文件的绝对路径。
// @returns env 文件路径列表。
func managedCreateEnvFileList(envFileAbsolutePath *string) []string {
	if envFileAbsolutePath == nil {
		return nil
	}
	return []string{*envFileAbsolutePath}
}

func guardCode(code string) GuardResult {
	return GuardResult{Code: code}
}

func guardDetail(code string, detail string) GuardResult {
	detail = strings.TrimSpace(detail)
	return GuardResult{Code: code, Detail: &detail}
}

// Up executes docker compose up -d within the project's registered working directory.
func (s *Service) Up(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionUp, []string{"compose", "up", "-d"})
}

// Down executes docker compose down for the registered project without removing volumes by default.
func (s *Service) Down(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionDown, []string{"compose", "down"})
}

// Restart executes docker compose restart for the registered project.
func (s *Service) Restart(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionRestart, []string{"compose", "restart"})
}

// Unregister removes the project registry record without touching host files.
func (s *Service) Unregister(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	if _, err := s.getAggregate(ctx, projectID); err != nil {
		return ActionResult{}, err
	}
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	if err := repository.UnregisterProject(ctx, projectstore.UnregisterProjectInput{
		ProjectID: projectID,
		ActorID:   actorID,
	}); err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	messageKey := projectcontract.ProjectUnregisterCompleted.String()
	return ActionResult{
		ProjectID:  projectID,
		Action:     generated.ProjectActionUnregister,
		Result:     generated.ProjectActionResultCompleted,
		MessageKey: &messageKey,
		Message:    &messageKey,
		GuardResults: []GuardResult{
			guardCode("registry_deleted"),
			guardCode("working_directory_preserved"),
			guardCode("runtime_state_not_persisted"),
		},
	}, nil
}

// Destroy executes guarded teardown steps and then unregisters the project record.
func (s *Service) Destroy(ctx context.Context, projectID uint64, request DestroyRequest) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	if result, blockErr := validateDestroyRequest(projectID, aggregate, request); blockErr != nil {
		return result, blockErr
	}
	return s.destroyAfterGuard(ctx, aggregate, request)
}

// validateDestroyRequest 校验销毁请求是否允许继续执行，并在违反保护条件时返回阻断结果。
// 当确认名称不匹配、请求删除命名卷，或请求删除工作目录但项目并非受控根专属所有权时，
// 返回带有相应守卫结果的阻断动作结果和销毁阻断错误。
// @param projectID 项目标识。
// @param aggregate 项目聚合数据。
// @param request 销毁请求。
// @returns 允许继续销毁时返回空动作结果和 nil；否则返回阻断动作结果和销毁阻断错误。
func validateDestroyRequest(
	projectID uint64,
	aggregate projectstore.ProjectAggregate,
	request DestroyRequest,
) (ActionResult, error) {
	guardResults := []GuardResult{}
	if strings.TrimSpace(request.ConfirmCanonicalProjectName) != aggregate.Project.CanonicalProjectName {
		return blockedActionResult(projectID, generated.ProjectActionDestroy, append(guardResults, guardCode("confirm_canonical_project_name_mismatch"))), errProjectDestroyBlocked
	}
	guardResults = append(guardResults, guardCode("confirm_canonical_project_name_matched"))

	if request.RemoveNamedVolumes {
		guardResults = append(guardResults, guardDetail("remove_named_volumes_blocked", "phase1_volume_exclusivity_not_proven"))
		return blockedActionResult(projectID, generated.ProjectActionDestroy, guardResults), errProjectDestroyBlocked
	}

	if request.DeleteWorkingDirectory && aggregate.Project.OwnershipMode != projectcontract.OwnershipModeManagedRootDedicated.String() {
		guardResults = append(guardResults, guardDetail("delete_working_directory_blocked", "ownership_mode_external"))
		return blockedActionResult(projectID, generated.ProjectActionDestroy, guardResults), errProjectDestroyBlocked
	}
	return ActionResult{}, nil
}

func (s *Service) destroyAfterGuard(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	request DestroyRequest,
) (ActionResult, error) {
	projectID := aggregate.Project.ID
	guardResults := []GuardResult{guardCode("confirm_canonical_project_name_matched")}
	if _, err := s.runLifecycleActionWithAggregate(ctx, aggregate, request.ActorID, generated.ProjectActionDown, []string{"compose", "down"}); err != nil {
		return ActionResult{}, err
	}
	guardResults = append(guardResults, guardCode("compose_down_completed"))

	if request.DeleteWorkingDirectory {
		fsRoot, err := openManagedRootFS(filepath.Clean(aggregate.Project.WorkingDirectory))
		if err != nil {
			return ActionResult{}, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
		}
		if err := fsRoot.root.RemoveAll("."); err != nil {
			fsRoot.close()
			return ActionResult{}, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
		}
		if err := fsRoot.close(); err != nil {
			return ActionResult{}, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
		}
		guardResults = append(guardResults, guardCode("working_directory_deleted"))
	} else {
		guardResults = append(guardResults, guardCode("working_directory_preserved"))
	}

	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	if err := repository.UnregisterProject(ctx, projectstore.UnregisterProjectInput{
		ProjectID: projectID,
		ActorID:   request.ActorID,
	}); err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	guardResults = append(guardResults, guardCode("registry_deleted"))
	messageKey := projectcontract.ProjectDestroyCompleted.String()
	return ActionResult{
		ProjectID:    projectID,
		Action:       generated.ProjectActionDestroy,
		Result:       generated.ProjectActionResultCompleted,
		MessageKey:   &messageKey,
		Message:      &messageKey,
		GuardResults: guardResults,
	}, nil
}

// ConfigurationMetadata returns readonly configuration metadata.
func (s *Service) ConfigurationMetadata(ctx context.Context, projectID uint64) (ConfigurationMetadataResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationMetadataResult{}, err
	}
	return ConfigurationMetadataResult{
		ProjectID:          projectID,
		ComposeFiles:       toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		EnvFiles:           toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())),
		OwnershipMode:      aggregate.Project.OwnershipMode,
		DriftStatus:        aggregate.Project.DriftStatus,
		LastRefreshStatus:  aggregate.Project.LastRefreshStatus,
		LastRefreshAt:      aggregate.Project.LastRefreshAt,
		DiagnosticsSummary: nil,
	}, nil
}

// ConfigurationPreview returns the latest static normalized compose preview.
func (s *Service) ConfigurationPreview(ctx context.Context, projectID uint64) (ConfigurationPreviewResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationPreviewResult{}, err
	}
	parseResult, err := s.loadFromAggregate(aggregate)
	if err != nil {
		return ConfigurationPreviewResult{}, err
	}
	return ConfigurationPreviewResult{
		ProjectID:             projectID,
		CanonicalProjectName:  aggregate.Project.CanonicalProjectName,
		ConfigHash:            parseResult.ConfigHash,
		NormalizedComposeYAML: parseResult.NormalizedComposeYAML,
		RefreshedAt:           aggregate.Project.LastRefreshAt,
	}, nil
}

// ConfigurationFile returns one readonly configuration file content payload.
func (s *Service) ConfigurationFile(ctx context.Context, projectID uint64, fileID uint64) (ConfigurationFileResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ConfigurationFileResult{}, err
	}
	file, err := repository.GetFile(ctx, projectID, fileID)
	if err != nil {
		return ConfigurationFileResult{}, mapStoreError(err)
	}
	content, err := os.ReadFile(file.AbsolutePath)
	if err != nil {
		return ConfigurationFileResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	return ConfigurationFileResult{
		FileID:       file.ID,
		Kind:         file.Kind,
		Path:         file.AbsolutePath,
		Content:      string(content),
		DownloadName: fileName(file.AbsolutePath),
	}, nil
}

// DiffConfiguration compares a managed draft against current tracked project files without writing persistent changes.
func (s *Service) DiffConfiguration(ctx context.Context, projectID uint64, draft ConfigurationDraft) (ConfigurationDiffResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	if err := ensureManagedProjectAggregate(aggregate); err != nil {
		return ConfigurationDiffResult{}, err
	}
	current, err := loadManagedDraftContent(aggregate)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	prepared, err := s.prepareConfigurationDraft(aggregate, draft)
	if err != nil {
		return ConfigurationDiffResult{}, err
	}
	files := []ConfigurationDiffFile{
		buildConfigurationDiffFile(projectcontract.FileKindCompose.String(), current.ComposePath, current.ComposeContent, prepared.Proposal.ComposeContent),
	}
	if current.EnvPath != "" || prepared.Proposal.EnvPath != "" {
		files = append(files, buildConfigurationDiffFile(projectcontract.FileKindEnv.String(), nonEmptyString(current.EnvPath, prepared.Proposal.EnvPath), current.EnvContent, derefString(prepared.Proposal.EnvContent)))
	}
	hasChanges := false
	for _, item := range files {
		if item.Changed {
			hasChanges = true
			break
		}
	}
	return ConfigurationDiffResult{
		ProjectID:            projectID,
		CanonicalProjectName: aggregate.Project.CanonicalProjectName,
		OwnershipMode:        aggregate.Project.OwnershipMode,
		CurrentConfigHash:    nonEmptyString(aggregate.Project.LastRefreshConfigHash, current.CurrentConfigHash),
		ProposedConfigHash:   prepared.ParseResult.ConfigHash,
		HasChanges:           hasChanges,
		Files:                files,
		Warnings:             prepared.Warnings,
	}, nil
}

// ValidateConfiguration validates a managed draft without persisting any file changes.
func (s *Service) ValidateConfiguration(ctx context.Context, projectID uint64, draft ConfigurationDraft) (ConfigurationValidateResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ConfigurationValidateResult{}, err
	}
	if err := ensureManagedProjectAggregate(aggregate); err != nil {
		return ConfigurationValidateResult{}, err
	}
	prepared, err := s.prepareConfigurationDraft(aggregate, draft)
	if err != nil {
		return ConfigurationValidateResult{}, err
	}
	return ConfigurationValidateResult{
		ProjectID:             projectID,
		CanonicalProjectName:  aggregate.Project.CanonicalProjectName,
		OwnershipMode:         aggregate.Project.OwnershipMode,
		ProposedConfigHash:    prepared.ParseResult.ConfigHash,
		NormalizedComposeYAML: prepared.ParseResult.NormalizedComposeYAML,
		DeclaredServiceNames:  append([]string(nil), prepared.ParseResult.ServiceNames...),
		Warnings:              prepared.Warnings,
	}, nil
}

// DeployConfiguration writes one managed draft, refreshes the snapshot, and runs docker compose up -d.
func (s *Service) DeployConfiguration(ctx context.Context, projectID uint64, draft ConfigurationDraft, actorID *uint64) (DeployResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return DeployResult{}, err
	}
	if err := ensureManagedProjectAggregate(aggregate); err != nil {
		return DeployResult{}, err
	}
	prepared, err := s.prepareConfigurationDraft(aggregate, draft)
	if err != nil {
		return DeployResult{}, err
	}
	repository, err := s.repositoryOrErr()
	if err != nil {
		return DeployResult{}, err
	}
	restoreItems, err := writeManagedDraft(aggregate.Project.WorkingDirectory, prepared.Proposal)
	if err != nil {
		return DeployResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	defer restoreManagedDraft(aggregate.Project.WorkingDirectory, restoreItems)

	now := time.Now().UTC()
	if _, err := s.runLifecycleActionWithAggregate(ctx, aggregate, actorID, generated.ProjectActionDeploy, []string{"compose", "up", "-d"}); err != nil {
		return DeployResult{}, err
	}
	updated, err := repository.RefreshProject(ctx, buildRefreshProjectInput(projectID, prepared.ParseResult, now, actorID))
	if err != nil {
		return DeployResult{}, mapStoreError(err)
	}
	messageKey := projectcontract.ProjectDeployCompleted.String()
	guardResults := []GuardResult{
		guardCode("managed_project"),
		guardCode("draft_written"),
		guardDetail("command", "docker compose up -d"),
		guardCode("snapshot_refreshed"),
	}
	if len(prepared.Warnings) > 0 {
		guardResults = append(guardResults, guardDetail("warnings", strings.Join(prepared.Warnings, "|")))
	}
	return DeployResult{
		ProjectID:            projectID,
		Action:               "deploy",
		Result:               "completed",
		CanonicalProjectName: updated.Project.CanonicalProjectName,
		OwnershipMode:        updated.Project.OwnershipMode,
		ConfigHash:           prepared.ParseResult.ConfigHash,
		RefreshedAt:          now,
		DeclaredServiceCount: len(prepared.ParseResult.ServiceNames),
		MessageKey:           &messageKey,
		Message:              &messageKey,
		GuardResults:         guardResults,
	}, nil
}

// UnsupportedLifecycleAction returns an explicit batch-2 blocked action result.
func (s *Service) UnsupportedLifecycleAction(projectID uint64, action generated.ProjectActionResponseAction) (ActionResult, error) {
	return ActionResult{
		ProjectID:    projectID,
		Action:       action,
		Result:       generated.ProjectActionResultBlocked,
		MessageKey:   stringPointer(projectcontract.ProjectLifecycleAccepted.String()),
		Message:      stringPointer(projectcontract.ProjectLifecycleAccepted.String()),
		GuardResults: []GuardResult{guardDetail("batch-2-scope", "lifecycle execution is deferred to phase-1-batch-3")},
	}, errProjectUnsupportedLifecycle
}

func (s *Service) runLifecycleAction(
	ctx context.Context,
	projectID uint64,
	actorID *uint64,
	action generated.ProjectActionResponseAction,
	args []string,
) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	return s.runLifecycleActionWithAggregate(ctx, aggregate, actorID, action, args)
}

func (s *Service) runLifecycleActionWithAggregate(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	_ *uint64,
	action generated.ProjectActionResponseAction,
	args []string,
) (ActionResult, error) {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		return blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_blocked", "refresh_required")}), err
	}
	if err := ensureLifecycleCommandArgs(args); err != nil {
		return blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_blocked", "invalid_command")}), err
	}
	commandOutput, err := s.runComposeCommand(ctx, aggregate, args)
	if err != nil {
		result := blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_failed", summarizeCommandOutput(commandOutput))})
		return result, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
	}
	messageKey := lifecycleMessageKey(action).String()
	return ActionResult{
		ProjectID:  aggregate.Project.ID,
		Action:     action,
		Result:     generated.ProjectActionResultCompleted,
		MessageKey: &messageKey,
		Message:    &messageKey,
		GuardResults: []GuardResult{
			guardDetail("command", strings.Join(args, " ")),
			guardDetail("host_scope", aggregate.Project.HostScope),
		},
	}, nil
}

func (s *Service) runComposeCommand(ctx context.Context, aggregate projectstore.ProjectAggregate, args []string) (string, error) {
	// #nosec G204 -- binary is fixed to docker and args are validated command fragments, not shell-expanded input.
	command := exec.CommandContext(ctx, "docker", args...)
	command.Dir = aggregate.Project.WorkingDirectory
	command.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return strings.TrimSpace(stdout.String() + "\n" + stderr.String()), err
}

// ensureLifecycleCommandArgs 校验生命周期命令参数。
// 只有当参数数量满足要求且每个参数都包含非空白内容时才通过。
func ensureLifecycleCommandArgs(args []string) error {
	if len(args) < minLifecycleArgCount {
		return errProjectInvalidArgument
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			return errProjectInvalidArgument
		}
	}
	return nil
}

// ensureProjectLifecycleReady 检查项目是否满足执行生命周期操作的条件。
func ensureProjectLifecycleReady(aggregate projectstore.ProjectAggregate) error {
	if strings.TrimSpace(aggregate.Project.HostScope) != projectcontract.HostScopeLocal.String() {
		return errProjectUnsupportedLifecycle
	}
	if aggregate.Project.LastRefreshStatus != projectcontract.RefreshStatusSuccess.String() {
		return errProjectUnsupportedLifecycle
	}
	return nil
}

// blockedActionResult 返回一个标记为阻止的项目操作结果，并保留给定的守卫结果。
//
// @param projectID 项目 ID。
// @param action 操作类型。
// @param guardResults 守卫结果列表。
// @returns 标记为 blocked 的 ActionResult，包含项目 ID、操作类型、阻止消息以及守卫结果副本。
func blockedActionResult(projectID uint64, action generated.ProjectActionResponseAction, guardResults []GuardResult) ActionResult {
	messageKey := projectcontract.ProjectLifecycleBlocked.String()
	return ActionResult{
		ProjectID:    projectID,
		Action:       action,
		Result:       generated.ProjectActionResultBlocked,
		MessageKey:   &messageKey,
		Message:      &messageKey,
		GuardResults: append([]GuardResult(nil), guardResults...),
	}
}

// lifecycleMessageKey 返回指定生命周期动作对应的完成消息键。
func lifecycleMessageKey(action generated.ProjectActionResponseAction) projectcontract.MessageKey {
	switch action {
	case generated.ProjectActionUp:
		return projectcontract.ProjectUpCompleted
	case generated.ProjectActionDown:
		return projectcontract.ProjectDownCompleted
	case generated.ProjectActionRestart:
		return projectcontract.ProjectRestartCompleted
	case generated.ProjectActionDestroy:
		return projectcontract.ProjectDestroyCompleted
	case generated.ProjectActionDeploy:
		return projectcontract.ProjectDeployCompleted
	case generated.ProjectActionUnregister:
		return projectcontract.ProjectUnregisterCompleted
	default:
		return projectcontract.ProjectLifecycleAccepted
	}
}

// summarizeCommandOutput 归一化并截断命令输出摘要。
// 它会去除首尾空白，空输出返回 "command_failed"，并将过长内容截断到最大摘要长度。
// @param output 原始命令输出。
// @returns 处理后的输出摘要。
func summarizeCommandOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "command_failed"
	}
	if len(trimmed) > maxCommandOutputSummary {
		return trimmed[:maxCommandOutputSummary]
	}
	return trimmed
}

func (s *Service) runtimeSummary(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
) (moduleapi.ContainerProjectRuntimeSummary, error) {
	if s == nil || s.runtimeReader == nil {
		return moduleapi.ContainerProjectRuntimeSummary{
			CanonicalProjectName: aggregate.Project.CanonicalProjectName,
			Members:              []moduleapi.ContainerProjectMember{},
		}, nil
	}
	return s.runtimeReader.ListProjectMembers(ctx, aggregate.Project.HostScope, aggregate.Project.CanonicalProjectName)
}

// membersByService 按服务名称对容器运行时成员进行分组。
// @return 按服务名称索引的成员列表。
func membersByService(items []moduleapi.ContainerProjectMember) map[string][]moduleapi.ContainerProjectMember {
	result := make(map[string][]moduleapi.ContainerProjectMember)
	for _, item := range items {
		result[item.ServiceName] = append(result[item.ServiceName], item)
	}
	return result
}

// applyGeneratedServiceMembers 将运行时成员填充到服务项中，并统计运行与停止数量。
// 当 target 为空时不执行任何操作。
func applyGeneratedServiceMembers(target *generated.ProjectServiceItem, items []moduleapi.ContainerProjectMember) {
	if target == nil {
		return
	}
	//nolint:revive // OpenAPI generated anonymous member field is ContainerId.
	type generatedProjectServiceMember = struct {
		ContainerId   string `json:"container_id"`
		ContainerName string `json:"container_name"`
		State         string `json:"state"`
	}
	members := make([]generatedProjectServiceMember, 0, len(items))
	for _, item := range items {
		members = append(members, generatedProjectServiceMember{
			ContainerId:   item.ContainerID,
			ContainerName: item.ContainerName,
			State:         item.CanonicalState,
		})
		if item.CanonicalState == "running" {
			target.RunningCount++
		} else {
			target.StoppedCount++
		}
	}
	target.ContainerMembers = members
}

func (s *Service) repositoryOrErr() (projectstore.Repository, error) {
	if s == nil || s.repository == nil {
		return nil, errProjectServiceUnavailable
	}
	return s.repository, nil
}

func (s *Service) getAggregate(ctx context.Context, projectID uint64) (projectstore.ProjectAggregate, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return projectstore.ProjectAggregate{}, err
	}
	if projectID == 0 {
		return projectstore.ProjectAggregate{}, errProjectInvalidArgument
	}
	aggregate, err := repository.Get(ctx, projectID)
	if err != nil {
		return projectstore.ProjectAggregate{}, mapStoreError(err)
	}
	return aggregate, nil
}

func (s *Service) parseImportRequest(
	request ImportRequest,
) (projectcompose.Result, ImportValidationResult, error) {
	parseResult, err := projectcompose.Load(projectcompose.Input{
		WorkingDirectory: request.WorkingDirectory,
		ComposeFiles:     request.ComposeFiles,
		EnvFiles:         request.EnvFiles,
	})
	if err != nil {
		return projectcompose.Result{}, ImportValidationResult{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	canonicalName := parseResult.CanonicalProjectName
	canonicalNameSource := parseResult.CanonicalNameSource
	if request.CanonicalProjectNameOverride != nil {
		override := strings.TrimSpace(*request.CanonicalProjectNameOverride)
		if override == "" {
			return projectcompose.Result{}, ImportValidationResult{}, errProjectInvalidArgument
		}
		canonicalName = override
		canonicalNameSource = projectcontract.CanonicalProjectNameSourceOverride.String()
	}
	validation := ImportValidationResult{
		CanonicalProjectName:       canonicalName,
		CanonicalProjectNameSource: canonicalNameSource,
		WorkingDirectory:           parseResult.WorkingDirectory,
		ComposeFiles:               toGeneratedFilesFromCompose(parseResult.ComposeFiles),
		EnvFiles:                   toGeneratedFilesFromCompose(parseResult.EnvFiles),
	}
	return parseResult, validation, nil
}

func (s *Service) computeConflicts(
	ctx context.Context,
	repository projectstore.Repository,
	request ImportRequest,
	validation ImportValidationResult,
) ([]string, error) {
	existing, err := repository.List(ctx, projectstore.ListQuery{Limit: projectConflictScanSize, Offset: 0})
	if err != nil {
		return nil, mapStoreError(err)
	}
	conflicts := make([]string, 0)
	targetWD := strings.TrimSpace(validation.WorkingDirectory)
	targetCanonical := strings.TrimSpace(validation.CanonicalProjectName)
	for _, item := range existing.Items {
		if strings.EqualFold(item.Project.WorkingDirectory, targetWD) && !sameDisplayName(request.DisplayName, item.Project.DisplayName) {
			conflicts = append(conflicts, "working_directory")
		}
		if strings.EqualFold(item.Project.CanonicalProjectName, targetCanonical) && !sameWorkingDirectory(targetWD, item.Project.WorkingDirectory) {
			conflicts = append(conflicts, "canonical_project_name")
		}
	}
	sort.Strings(conflicts)
	return uniqueStrings(conflicts), nil
}

// sameDisplayName 判断给定名称与已有名称在去除首尾空白后是否一致。
//
// @param value 待比较的名称。
// @param existing 已存在的名称。
// @returns 当两者去除首尾空白后相等时返回 true，否则返回 false。
func sameDisplayName(value *string, existing string) bool {
	if value == nil {
		return false
	}
	return strings.TrimSpace(*value) == strings.TrimSpace(existing)
}

// sameWorkingDirectory 判断两个工作目录在去除首尾空白后是否相同。
// @returns 去除首尾空白并忽略大小写后路径相同则为 `true`，否则为 `false`。
func sameWorkingDirectory(left string, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

// toProjectListItem 将项目聚合转换为列表项，并在提供运行时摘要时补充容器运行统计。
// 它包含项目标识、名称、来源、工作目录、声明服务数，以及最近刷新和漂移状态。
func toProjectListItem(
	aggregate projectstore.ProjectAggregate,
	runtimeSummary ...moduleapi.ContainerProjectRuntimeSummary,
) generated.ProjectListItem {
	serviceCount := 0
	if aggregate.Snapshot != nil {
		serviceCount = aggregate.Snapshot.DeclaredServiceCount
	}
	counts := generated.ProjectContainerCounts{}
	if len(runtimeSummary) > 0 {
		counts.Running = runtimeSummary[0].RunningCount
		counts.Stopped = runtimeSummary[0].StoppedCount
		counts.Total = runtimeSummary[0].RunningCount + runtimeSummary[0].StoppedCount
	}
	return generated.ProjectListItem{
		Id:                         mustGeneratedID(aggregate.Project.ID),
		DisplayName:                aggregate.Project.DisplayName,
		CanonicalProjectName:       aggregate.Project.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(aggregate.Project.CanonicalProjectNameSource),
		SourceKind:                 generated.ProjectSourceKind(aggregate.Project.SourceKind),
		HostScope:                  generated.ProjectHostScope(aggregate.Project.HostScope),
		OwnershipMode:              generated.ProjectOwnershipMode(aggregate.Project.OwnershipMode),
		WorkingDirectory:           aggregate.Project.WorkingDirectory,
		ServiceCount:               serviceCount,
		ContainerCounts:            counts,
		LastRefreshStatus:          generated.ProjectRefreshStatus(aggregate.Project.LastRefreshStatus),
		LastRefreshAt:              aggregate.Project.LastRefreshAt,
		DriftStatus:                generated.ProjectDriftStatus(aggregate.Project.DriftStatus),
	}
}

// toProjectDetailResponse 将项目聚合数据转换为详情响应。
//
// 当提供运行时汇总时，会填充容器运行与停止数量；当聚合包含快照时，会填充服务数。项目的错误信息和配置哈希仅在存在时写入响应。
func toProjectDetailResponse(
	aggregate projectstore.ProjectAggregate,
	runtimeSummary ...moduleapi.ContainerProjectRuntimeSummary,
) generated.ProjectDetailResponse {
	counts := generated.ProjectContainerCounts{}
	if len(runtimeSummary) > 0 {
		counts.Running = runtimeSummary[0].RunningCount
		counts.Stopped = runtimeSummary[0].StoppedCount
		counts.Total = runtimeSummary[0].RunningCount + runtimeSummary[0].StoppedCount
	}
	item := generated.ProjectDetailResponse{
		CanonicalProjectName:       aggregate.Project.CanonicalProjectName,
		CanonicalProjectNameSource: generated.ProjectCanonicalNameSource(aggregate.Project.CanonicalProjectNameSource),
		ComposeFiles:               toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())),
		ContainerCounts:            counts,
		DisplayName:                aggregate.Project.DisplayName,
		DriftStatus:                generated.ProjectDriftStatus(aggregate.Project.DriftStatus),
		EnvFiles:                   toGeneratedFiles(filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())),
		HostScope:                  generated.ProjectHostScope(aggregate.Project.HostScope),
		Id:                         mustGeneratedID(aggregate.Project.ID),
		LastDriftCheckedAt:         aggregate.Project.LastDriftCheckedAt,
		LastRefreshAt:              aggregate.Project.LastRefreshAt,
		LastRefreshStatus:          generated.ProjectRefreshStatus(aggregate.Project.LastRefreshStatus),
		OwnershipMode:              generated.ProjectOwnershipMode(aggregate.Project.OwnershipMode),
		SourceKind:                 generated.ProjectSourceKind(aggregate.Project.SourceKind),
		WorkingDirectory:           aggregate.Project.WorkingDirectory,
	}
	if aggregate.Project.LastRefreshErrorCode != "" {
		item.LastRefreshErrorCode = stringPointer(aggregate.Project.LastRefreshErrorCode)
	}
	if aggregate.Project.LastRefreshErrorMessage != "" {
		item.LastRefreshErrorMessage = stringPointer(aggregate.Project.LastRefreshErrorMessage)
	}
	if aggregate.Project.LastRefreshConfigHash != "" {
		item.LastRefreshConfigHash = stringPointer(aggregate.Project.LastRefreshConfigHash)
	}
	if aggregate.Project.LastObservedConfigHash != "" {
		item.LastObservedConfigHash = stringPointer(aggregate.Project.LastObservedConfigHash)
	}
	if aggregate.Snapshot != nil {
		item.ServiceCount = aggregate.Snapshot.DeclaredServiceCount
	}
	return item
}

// toGeneratedFiles 将存储的文件记录转换为生成的文件项列表。
func toGeneratedFiles(files []projectstore.ProjectFile) []generated.ProjectFileItem {
	items := make([]generated.ProjectFileItem, 0, len(files))
	for _, item := range files {
		items = append(items, generated.ProjectFileItem{
			Id:                  mustGeneratedID(item.ID),
			Kind:                generated.ProjectFileKind(item.Kind),
			Role:                generated.ProjectFileRole(item.Role),
			AbsolutePath:        item.AbsolutePath,
			DisplayPath:         item.DisplayPath,
			OrderIndex:          item.OrderIndex,
			ExistsOnLastRefresh: item.ExistsOnLastRefresh,
			LastObservedHash:    optionalString(item.LastObservedHash),
		})
	}
	return items
}

// 将 compose 投影文件转换为生成的项目文件项列表。
func toGeneratedFilesFromCompose(files []projectcompose.FileProjection) []generated.ProjectFileItem {
	items := make([]generated.ProjectFileItem, 0, len(files))
	for index, item := range files {
		hash := item.Hash
		items = append(items, generated.ProjectFileItem{
			Id:                  int64(index + 1),
			Kind:                generated.ProjectFileKind(item.Kind),
			Role:                generated.ProjectFileRole(item.Role),
			AbsolutePath:        item.AbsolutePath,
			DisplayPath:         item.DisplayPath,
			OrderIndex:          item.OrderIndex,
			ExistsOnLastRefresh: item.Exists,
			LastObservedHash:    &hash,
		})
	}
	return items
}

// toStoreFiles 将 compose 和 env 文件投影转换为存储层文件记录。
func toStoreFiles(composeFiles []projectcompose.FileProjection, envFiles []projectcompose.FileProjection) []projectstore.ProjectFile {
	items := make([]projectstore.ProjectFile, 0, len(composeFiles)+len(envFiles))
	for _, item := range append(append([]projectcompose.FileProjection(nil), composeFiles...), envFiles...) {
		items = append(items, projectstore.ProjectFile{
			Kind:                item.Kind,
			Role:                item.Role,
			AbsolutePath:        item.AbsolutePath,
			DisplayPath:         item.DisplayPath,
			OrderIndex:          item.OrderIndex,
			ExistsOnLastRefresh: item.Exists,
			LastObservedHash:    item.Hash,
		})
	}
	return items
}

// 返回的文件先按 OrderIndex 升序排列，OrderIndex 相同时按 ID 升序排列。
func filterFiles(files []projectstore.ProjectFile, kind string) []projectstore.ProjectFile {
	items := make([]projectstore.ProjectFile, 0)
	for _, item := range files {
		if item.Kind == kind {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(left, right int) bool {
		if items[left].OrderIndex == items[right].OrderIndex {
			return items[left].ID < items[right].ID
		}
		return items[left].OrderIndex < items[right].OrderIndex
	})
	return items
}

// collectFilesByKind 返回指定类型的文件绝对路径列表。
func collectFilesByKind(files []projectstore.ProjectFile, kind string) []string {
	filtered := filterFiles(files, kind)
	paths := make([]string, 0, len(filtered))
	for _, item := range filtered {
		paths = append(paths, item.AbsolutePath)
	}
	return paths
}

func (s *Service) loadFromAggregate(aggregate projectstore.ProjectAggregate) (projectcompose.Result, error) {
	return projectcompose.Load(projectcompose.Input{
		WorkingDirectory: aggregate.Project.WorkingDirectory,
		ComposeFiles:     collectFilesByKind(aggregate.Files, projectcontract.FileKindCompose.String()),
		EnvFiles:         collectFilesByKind(aggregate.Files, projectcontract.FileKindEnv.String()),
	})
}

// normalizeSnapshotJSON 将输入内容规范化为 JSON 表示。
// 输入为空时返回 "{}"；当解析或重新编码失败时，返回原始内容。
func normalizeSnapshotJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	var generic any
	if err := yamlJSONRoundTrip(raw, &generic); err != nil {
		return raw
	}
	encoded, err := json.Marshal(generic)
	if err != nil {
		return raw
	}
	return encoded
}

// yamlJSONRoundTrip 将 JSON 数据解析到目标值。
//
// @param raw 要解析的数据。
// @param target 接收解析结果的目标值。
// @returns 解析过程中返回的错误。
func yamlJSONRoundTrip(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}

// digestServiceNames 计算服务名称集合的稳定摘要。
// 它会按字典序规范化名称后生成 SHA-256 十六进制字符串。
func digestServiceNames(names []string) string {
	normalized := append([]string(nil), names...)
	sort.Strings(normalized)
	hasher := sha256.New()
	for _, item := range normalized {
		hasher.Write([]byte(item))
		hasher.Write([]byte{0})
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// buildRefreshProjectInput 组装项目刷新持久化输入，包含刷新状态、快照、文件与操作者信息。
// 该输入将刷新时间、配置哈希、归一化后的 compose 快照和声明服务摘要写入存储层。
func buildRefreshProjectInput(
	projectID uint64,
	parseResult projectcompose.Result,
	now time.Time,
	actorID *uint64,
) projectstore.RefreshProjectInput {
	return projectstore.RefreshProjectInput{
		ProjectID:              projectID,
		LastRefreshStatus:      projectcontract.RefreshStatusSuccess.String(),
		LastRefreshAt:          &now,
		LastRefreshConfigHash:  parseResult.ConfigHash,
		LastObservedConfigHash: parseResult.ConfigHash,
		LastDriftCheckedAt:     &now,
		DriftStatus:            projectcontract.DriftStatusClean.String(),
		Files:                  toStoreFiles(parseResult.ComposeFiles, parseResult.EnvFiles),
		Snapshot: &projectstore.Snapshot{
			ProjectID:              projectID,
			ConfigHash:             parseResult.ConfigHash,
			NormalizedComposeJSON:  normalizeSnapshotJSON(parseResult.NormalizedComposeJSON),
			DeclaredServiceCount:   len(parseResult.ServiceNames),
			DeclaredServicesDigest: digestServiceNames(parseResult.ServiceNames),
			RefreshedAt:            now,
		},
		ActorID: actorID,
	}
}

// displayNameOrCanonical 返回修剪后的显示名称或规范名称。
//
// 当显示名称存在且非空时，返回其去除首尾空白后的值；否则返回规范名称。
func displayNameOrCanonical(displayName *string, canonical string) string {
	if displayName != nil && strings.TrimSpace(*displayName) != "" {
		return strings.TrimSpace(*displayName)
	}
	return canonical
}

// fileName 返回路径中的最后一个段。
func fileName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

type managedDraftContent struct {
	ComposePath       string
	ComposeContent    string
	EnvPath           string
	EnvContent        string
	CurrentConfigHash string
}

type managedDraftProposal struct {
	ComposePath    string
	ComposeContent string
	EnvPath        string
	EnvContent     *string
}

type managedDraftRestore struct {
	Path    string
	Content []byte
	Exists  bool
}

// ensureManagedProjectAggregate 仅允许受控根目录专用归属模式的项目进入受控草案流程。
// 当项目归属模式不是 managed-root-dedicated 时返回 errProjectManagedFlow。
func ensureManagedProjectAggregate(aggregate projectstore.ProjectAggregate) error {
	if aggregate.Project.OwnershipMode != projectcontract.OwnershipModeManagedRootDedicated.String() {
		return errProjectManagedFlow
	}
	return nil
}

// loadManagedDraftContent 读取托管草案所需的当前 compose 和 env 内容。
// 它要求至少存在一个 compose 文件，并返回首个 compose 文件的绝对路径、归一化后的文件内容、当前配置哈希，以及可选的 env 文件路径和内容。
func loadManagedDraftContent(aggregate projectstore.ProjectAggregate) (managedDraftContent, error) {
	composeFiles := filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())
	if len(composeFiles) == 0 {
		return managedDraftContent{}, fmt.Errorf("%w: missing compose file authority", errProjectImportValidation)
	}
	fsRoot, err := openManagedRootFS(filepath.Clean(aggregate.Project.WorkingDirectory))
	if err != nil {
		return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	defer fsRoot.close()
	composeRelativePath, err := fsRoot.relative(composeFiles[0].AbsolutePath)
	if err != nil {
		return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	composeContent, err := fsRoot.root.ReadFile(composeRelativePath)
	if err != nil {
		return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, err)
	}
	result := managedDraftContent{
		ComposePath:       composeFiles[0].AbsolutePath,
		ComposeContent:    normalizeTextBlock(string(composeContent)),
		CurrentConfigHash: aggregate.Project.LastRefreshConfigHash,
	}
	envFiles := filterFiles(aggregate.Files, projectcontract.FileKindEnv.String())
	if len(envFiles) > 0 {
		envRelativePath, relErr := fsRoot.relative(envFiles[0].AbsolutePath)
		if relErr != nil {
			return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, relErr)
		}
		envContent, readErr := fsRoot.root.ReadFile(envRelativePath)
		if readErr != nil {
			return managedDraftContent{}, fmt.Errorf("%w: %v", errProjectImportValidation, readErr)
		}
		result.EnvPath = envFiles[0].AbsolutePath
		result.EnvContent = normalizeTextBlock(string(envContent))
	}
	return result, nil
}

func (s *Service) prepareConfigurationDraft(
	aggregate projectstore.ProjectAggregate,
	draft ConfigurationDraft,
) (preparedConfigurationDraft, error) {
	current, err := loadManagedDraftContent(aggregate)
	if err != nil {
		return preparedConfigurationDraft{}, err
	}
	composeContent := normalizeTextBlock(draft.ComposeFileContent)
	if strings.TrimSpace(composeContent) == "" {
		return preparedConfigurationDraft{}, errProjectInvalidArgument
	}
	proposal := managedDraftProposal{
		ComposePath:    current.ComposePath,
		ComposeContent: composeContent,
		EnvPath:        current.EnvPath,
	}
	if draft.EnvFileContent != nil {
		content := normalizeTextBlock(*draft.EnvFileContent)
		proposal.EnvContent = &content
	} else if current.EnvPath != "" {
		content := current.EnvContent
		proposal.EnvContent = &content
	}
	draftInput, err := buildManagedDraftInput(aggregate.Project.WorkingDirectory, proposal)
	if err != nil {
		return preparedConfigurationDraft{}, err
	}
	parseResult, err := projectcompose.Load(draftInput)
	if err != nil {
		return preparedConfigurationDraft{}, err
	}
	warnings := make([]string, 0, draftWarningsCap)
	if strings.TrimSpace(current.ComposeContent) == strings.TrimSpace(proposal.ComposeContent) &&
		strings.TrimSpace(current.EnvContent) == strings.TrimSpace(derefString(proposal.EnvContent)) {
		warnings = append(warnings, "Draft matches the current tracked managed project files.")
	}
	return preparedConfigurationDraft{
		Proposal:    proposal,
		ParseResult: parseResult,
		Warnings:    warnings,
	}, nil
}

func buildManagedDraftInput(workingDirectory string, proposal managedDraftProposal) (projectcompose.Input, error) {
	input := projectcompose.Input{
		WorkingDirectory: workingDirectory,
		ComposeFiles:     []string{proposal.ComposePath},
	}
	if proposal.EnvPath != "" && proposal.EnvContent != nil {
		input.EnvFiles = []string{proposal.EnvPath}
	}
	return input.WithContentOverrides(map[string][]byte{
		proposal.ComposePath: []byte(proposal.ComposeContent),
		proposal.EnvPath:     optionalDraftBytes(proposal.EnvPath, proposal.EnvContent),
	}), nil
}

func optionalDraftBytes(path string, content *string) []byte {
	if path == "" || content == nil {
		return nil
	}
	return []byte(*content)
}

// writeManagedDraft 写入托管草案的 compose 文件，并在需要时写入 env 文件，同时返回用于恢复原状态的记录。
// 它会先保存目标文件的原始内容和是否存在，以便后续恢复。
func writeManagedDraft(workingDirectory string, proposal managedDraftProposal) ([]managedDraftRestore, error) {
	targets := []struct {
		path    string
		content string
	}{
		{path: proposal.ComposePath, content: proposal.ComposeContent},
	}
	if proposal.EnvPath != "" && proposal.EnvContent != nil {
		targets = append(targets, struct {
			path    string
			content string
		}{path: proposal.EnvPath, content: *proposal.EnvContent})
	}
	fsRoot, err := openManagedRootFS(filepath.Clean(workingDirectory))
	if err != nil {
		return nil, err
	}
	defer fsRoot.close()
	restoreItems := make([]managedDraftRestore, 0, len(targets))
	for _, target := range targets {
		relative, err := fsRoot.relative(target.path)
		if err != nil {
			return nil, err
		}
		original, err := fsRoot.root.ReadFile(relative)
		exists := err == nil
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		restoreItems = append(restoreItems, managedDraftRestore{
			Path:    target.path,
			Content: append([]byte(nil), original...),
			Exists:  exists,
		})
		if err := fsRoot.root.WriteFile(relative, []byte(target.content), managedCreateFileMode); err != nil {
			return nil, err
		}
	}
	return restoreItems, nil
}

// restoreManagedDraft 按相反顺序恢复受控草案写入前的文件状态。
func restoreManagedDraft(workingDirectory string, items []managedDraftRestore) {
	fsRoot, err := openManagedRootFS(filepath.Clean(workingDirectory))
	if err != nil {
		return
	}
	defer fsRoot.close()
	for index := len(items) - 1; index >= 0; index-- {
		item := items[index]
		relative, relErr := fsRoot.relative(item.Path)
		if relErr != nil {
			continue
		}
		if item.Exists {
			_ = fsRoot.root.WriteFile(relative, item.Content, managedCreateFileMode)
			continue
		}
		_ = fsRoot.root.Remove(relative)
	}
}

func openManagedRootFS(rootDir string) (*managedRootFS, error) {
	absolute := filepath.Clean(strings.TrimSpace(rootDir))
	if absolute == "" {
		return nil, fmt.Errorf("managed root directory is required")
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, err
	}
	return &managedRootFS{root: root, rootDir: absolute}, nil
}

func openManagedRootFSForPaths(rootDir string, paths ...string) (*managedRootFS, error) {
	if strings.TrimSpace(rootDir) != "" {
		return openManagedRootFS(rootDir)
	}
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		return openManagedRootFS(filepath.Dir(filepath.Clean(trimmed)))
	}
	return nil, fmt.Errorf("managed root directory is required")
}

func (fsRoot *managedRootFS) relative(path string) (string, error) {
	if fsRoot == nil || fsRoot.root == nil {
		return "", fmt.Errorf("managed root is unavailable")
	}
	relative, err := filepath.Rel(fsRoot.rootDir, filepath.Clean(strings.TrimSpace(path)))
	if err != nil {
		return "", err
	}
	if relative == "." || relative == "" {
		return ".", nil
	}
	if strings.HasPrefix(relative, "..") {
		return "", fmt.Errorf("path escapes managed root")
	}
	return relative, nil
}

func (fsRoot *managedRootFS) close() error {
	if fsRoot == nil || fsRoot.root == nil {
		return nil
	}
	return fsRoot.root.Close()
}

// buildConfigurationDiffFile 构建配置文件的差异结果，包含内容变更、哈希和统一 diff。
//
// 返回的结果会反映当前内容与提议内容的归一化比较，并保留对应的文件类型和路径。
func buildConfigurationDiffFile(kind string, path string, current string, proposed string) ConfigurationDiffFile {
	return ConfigurationDiffFile{
		Kind:            kind,
		Path:            path,
		Changed:         normalizeTextBlock(current) != normalizeTextBlock(proposed),
		CurrentHash:     hashString(current),
		ProposedHash:    hashString(proposed),
		CurrentContent:  normalizeTextBlock(current),
		ProposedContent: buildUnifiedDiff(current, proposed),
	}
}

// buildUnifiedDiff 生成当前内容与提议内容之间的统一差异文本。
// 当差异文本为空或仅包含空白字符时，返回规范化后的提议内容。
func buildUnifiedDiff(current string, proposed string) string {
	differ := diffmatchpatch.New()
	patches := differ.PatchMake(current, proposed)
	text := differ.PatchToText(patches)
	if strings.TrimSpace(text) == "" {
		return normalizeTextBlock(proposed)
	}
	return text
}

// hashString 返回归一化文本块的 SHA-256 十六进制摘要。
func hashString(value string) string {
	sum := sha256.Sum256([]byte(normalizeTextBlock(value)))
	return hex.EncodeToString(sum[:])
}

// normalizeTextBlock 规范化文本块的换行、行尾空白和整体边界，并在非空时补充结尾换行符。
func normalizeTextBlock(value string) string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	joined := strings.TrimSpace(strings.Join(lines, "\n"))
	if joined == "" {
		return ""
	}
	return joined + "\n"
}

// derefString 返回指针指向的字符串值。
// 如果指针为空，则返回空字符串。
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// nonEmptyString 在 primary 为空白时返回 fallback。
//
// @return primary 经修剪后非空时返回其原值；否则返回 fallback。
func nonEmptyString(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

// stringPointer 返回一个指向非空白字符串的指针。
//
// 如果输入经修剪后为空，则返回 nil。
func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// optionalString 将字符串包装为可选字符串指针。
func optionalString(value string) *string {
	return stringPointer(value)
}

// uniqueStrings 返回去重后的字符串切片，保留首次出现的顺序。
func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// normalizeListLimit 将列表限制值规范到默认值或最大值范围内。
//
// @returns 规范后的列表限制值：当输入小于等于 0 时返回默认值，超过最大值时返回最大值，否则返回原值。
func normalizeListLimit(value int) int {
	switch {
	case value <= 0:
		return defaultProjectListLimit
	case value > maxProjectListLimit:
		return maxProjectListLimit
	default:
		return value
	}
}

// mustGeneratedID 将生成的无符号 ID 转换为 int64。
// 当值为 0 或超出 int64 可表示范围时会 panic。
func mustGeneratedID(value uint64) int64 {
	if value == 0 || value > math.MaxInt64 {
		panic("project generated id out of range")
	}
	return int64(value)
}

// maxInt 返回 value 与 minimum 中较大的值。
func maxInt(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

// mapStoreError 将存储层错误映射为本包使用的错误。
// 已知的无效输入、未找到、冲突和文件不存在错误会转换为对应的本地哨兵错误；其他错误原样返回。
func mapStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, projectstore.ErrInvalidInput):
		return errProjectInvalidArgument
	case errors.Is(err, projectstore.ErrProjectNotFound):
		return errProjectNotFound
	case errors.Is(err, projectstore.ErrProjectConflict):
		return errProjectConflict
	case errors.Is(err, projectstore.ErrFileNotFound):
		return errProjectFileNotFound
	default:
		return err
	}
}
