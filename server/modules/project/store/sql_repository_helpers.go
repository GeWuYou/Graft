package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	projectcontract "graft/server/modules/project/contract"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// normalizeSourceMetadata trims metadata keys and values and validates their content.
// It returns a normalized copy of the metadata or ErrInvalidInput when a key or value
// is empty or contains prohibited characters.
func normalizeSourceMetadata(metadata map[string]string) (map[string]string, error) {
	if len(metadata) == 0 {
		return map[string]string{}, nil
	}
	result := make(map[string]string, len(metadata))
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			return nil, ErrInvalidInput
		}
		if strings.ContainsAny(key, "\x00\r\n") || strings.ContainsAny(value, "\x00\r\n") {
			return nil, ErrInvalidInput
		}
		result[key] = value
	}
	return result, nil
}

// encodeSourceMetadataJSON normalizes source metadata and encodes it as JSON.
// It returns an error when the metadata is invalid or cannot be encoded.
func encodeSourceMetadataJSON(metadata map[string]string) ([]byte, error) {
	normalized, err := normalizeSourceMetadata(metadata)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("encode source metadata: %w", err)
	}
	return encoded, nil
}

// decodeSourceMetadataJSON decodes and normalizes source metadata from JSON.
// Empty input produces an empty metadata map. It returns an error when the JSON
// is invalid or the decoded metadata fails validation.
func decodeSourceMetadataJSON(raw []byte) (map[string]string, error) {
	if len(raw) == 0 {
		return map[string]string{}, nil
	}
	var metadata map[string]string
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, fmt.Errorf("decode source metadata: %w", err)
	}
	return normalizeSourceMetadata(metadata)
}

func (r *SQLRepository) ensureReady() error {
	if r == nil || r.db == nil {
		return errors.New("project repository is unavailable")
	}
	return nil
}

// normalizeListQuery 规范化项目列表查询条件，并将分页参数限制在允许范围内。
// 排序值只允许稳定的 created_at + id 组合；无效筛选、关键字或运行目标 ID 返回 ErrInvalidInput。
func normalizeListQuery(query ListQuery) (ListQuery, error) {
	normalizedSort, err := normalizeProjectListSort(query.Sort)
	if err != nil {
		return ListQuery{}, ErrInvalidInput
	}
	query.Sort = normalizedSort

	var normalizeErr error
	query.SourceKind, normalizeErr = normalizeOptionalContractValue(query.SourceKind, isValidSourceKind)
	if normalizeErr != nil {
		return ListQuery{}, normalizeErr
	}
	query.DriftStatus, normalizeErr = normalizeOptionalContractValue(query.DriftStatus, isValidDriftStatus)
	if normalizeErr != nil {
		return ListQuery{}, normalizeErr
	}
	query.Keyword = strings.TrimSpace(query.Keyword)
	if len(query.Keyword) > 128 || (query.RuntimeTargetID != nil && *query.RuntimeTargetID < 1) {
		return ListQuery{}, ErrInvalidInput
	}
	if query.Limit <= 0 {
		query.Limit = defaultListLimit
	}
	if query.Limit > maxListLimit {
		query.Limit = maxListLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query, nil
}

// normalizeProjectListSort 校验项目列表排序表达式，并返回可安全映射到固定 SQL 子句的值。
// 不支持的表达式返回 ErrInvalidInput，避免请求值进入 ORDER BY。
func normalizeProjectListSort(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case "", ProjectListSortCreatedAtDesc:
		return ProjectListSortCreatedAtDesc, nil
	case ProjectListSortCreatedAtAsc:
		return ProjectListSortCreatedAtAsc, nil
	default:
		return "", ErrInvalidInput
	}
}

// buildListOrderBy 将已校验的项目列表排序表达式映射为固定 SQL 子句。
// created_at 相同时始终使用 ID 降序作为并列排序，保证分页结果稳定。
func buildListOrderBy(sortExpression string) string {
	if sortExpression == ProjectListSortCreatedAtAsc {
		return "created_at ASC, id DESC"
	}
	return "created_at DESC, id DESC"
}

// validateImportInput 规范化并校验导入项目输入，返回可直接使用的输入值。
// validateImportInput 修剪并验证项目导入输入，规范化文件、快照、源元数据以及时间字段。
// 返回规范化后的导入输入；输入无效时返回错误。
func validateImportInput(input ImportProjectInput) (ImportProjectInput, error) {
	input = trimImportInput(input)
	if err := validateRequiredImportFields(input); err != nil {
		return ImportProjectInput{}, fmt.Errorf("validate required project fields: %w", err)
	}
	if err := validateImportContracts(input); err != nil {
		return ImportProjectInput{}, fmt.Errorf("validate project contracts: %w", err)
	}
	files, err := normalizeFiles(input.Files)
	if err != nil {
		return ImportProjectInput{}, fmt.Errorf("validate project files: %w", err)
	}
	input.Files = files
	snapshot, err := normalizeSnapshot(input.Snapshot)
	if err != nil {
		return ImportProjectInput{}, fmt.Errorf("validate project snapshot: %w", err)
	}
	input.Snapshot = snapshot
	metadata, err := normalizeSourceMetadata(input.SourceMetadata)
	if err != nil {
		return ImportProjectInput{}, fmt.Errorf("validate project source metadata: %w", err)
	}
	input.SourceMetadata = metadata
	normalizeTemporalPointers(&input.LastDriftCheckedAt)
	return input, nil
}

// trimImportInput 去除导入项目输入中字符串字段及来源元数据值的首尾空白。
func trimImportInput(input ImportProjectInput) ImportProjectInput {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.CanonicalProjectName = strings.TrimSpace(input.CanonicalProjectName)
	input.CanonicalProjectNameSource = strings.TrimSpace(input.CanonicalProjectNameSource)
	input.SourceKind = strings.TrimSpace(input.SourceKind)
	input.HostScope = strings.TrimSpace(input.HostScope)
	input.WorkingDirectory = strings.TrimSpace(input.WorkingDirectory)
	input.OwnershipMode = strings.TrimSpace(input.OwnershipMode)
	for key, value := range input.SourceMetadata {
		input.SourceMetadata[key] = strings.TrimSpace(value)
	}
	input.LifecycleStrategyKind = strings.TrimSpace(input.LifecycleStrategyKind)
	input.LifecycleReviewStatus = strings.TrimSpace(input.LifecycleReviewStatus)
	input.LastObservedConfigHash = strings.TrimSpace(input.LastObservedConfigHash)
	input.DriftStatus = strings.TrimSpace(input.DriftStatus)
	return input
}

// validateRequiredImportFields 检查导入项目所需的必填字段是否已提供。
// 若任一必填字段为空，返回 ErrInvalidInput。
func validateRequiredImportFields(input ImportProjectInput) error {
	required := []string{
		input.DisplayName,
		input.CanonicalProjectName,
		input.CanonicalProjectNameSource,
		input.SourceKind,
		input.HostScope,
		input.WorkingDirectory,
		input.OwnershipMode,
		input.LifecycleStrategyKind,
		input.LifecycleReviewStatus,
		input.DriftStatus,
	}
	for _, value := range required {
		if value == "" {
			return ErrInvalidInput
		}
	}
	return nil
}

// validateRefreshInput 规范化并校验项目刷新输入。
// 它会校验项目 ID，清理漂移状态字段，规范化文件和快照信息，并将时间字段转换为 UTC。
// @param input 待校验的刷新输入。
// @returns 规范化后的刷新输入，或在输入无效时返回 ErrInvalidInput。
func validateRefreshInput(input RefreshProjectInput) (RefreshProjectInput, error) {
	if input.ProjectID == 0 {
		return RefreshProjectInput{}, ErrInvalidInput
	}
	input.LastObservedConfigHash = strings.TrimSpace(input.LastObservedConfigHash)
	input.DriftStatus = strings.TrimSpace(input.DriftStatus)
	if input.DriftStatus == "" {
		return RefreshProjectInput{}, ErrInvalidInput
	}
	if err := validateRefreshContracts(input); err != nil {
		return RefreshProjectInput{}, err
	}
	files, err := normalizeFiles(input.Files)
	if err != nil {
		return RefreshProjectInput{}, err
	}
	input.Files = files
	snapshot, err := normalizeSnapshot(input.Snapshot)
	if err != nil {
		return RefreshProjectInput{}, err
	}
	input.Snapshot = snapshot
	normalizeTemporalPointers(&input.LastDriftCheckedAt)
	return input, nil
}

// validateUnregisterInput 校验注销项目请求输入。
// ProjectID 为空时返回 ErrInvalidInput。
func validateUnregisterInput(input UnregisterProjectInput) (UnregisterProjectInput, error) {
	if input.ProjectID == 0 {
		return UnregisterProjectInput{}, ErrInvalidInput
	}
	return input, nil
}

func validateUpdateLifecycleConfigInput(input UpdateLifecycleConfigInput) (UpdateLifecycleConfigInput, error) {
	if input.ProjectID == 0 {
		return UpdateLifecycleConfigInput{}, ErrInvalidInput
	}
	input.LifecycleStrategyKind = strings.TrimSpace(input.LifecycleStrategyKind)
	input.LifecycleReviewStatus = strings.TrimSpace(input.LifecycleReviewStatus)
	config, err := normalizeLifecycleConfig(input.LifecycleConfig)
	if err != nil {
		return UpdateLifecycleConfigInput{}, err
	}
	if !isValidLifecycleStrategyKind(input.LifecycleStrategyKind) || !isValidLifecycleReviewStatus(input.LifecycleReviewStatus) {
		return UpdateLifecycleConfigInput{}, ErrInvalidInput
	}
	input.LifecycleConfig = config
	return input, nil
}

func validateUpdateWorkspaceAnnotationInput(input UpdateWorkspaceAnnotationInput) (UpdateWorkspaceAnnotationInput, error) {
	if input.ProjectID == 0 {
		return UpdateWorkspaceAnnotationInput{}, ErrInvalidInput
	}
	input.RelativePath = normalizeWorkspaceAnnotationPath(input.RelativePath)
	if input.RelativePath == "" {
		return UpdateWorkspaceAnnotationInput{}, ErrInvalidInput
	}
	if input.Annotation == nil {
		return input, nil
	}
	annotation := strings.TrimSpace(*input.Annotation)
	if annotation == "" {
		input.Annotation = nil
		return input, nil
	}
	if len(annotation) > projectcontract.ProjectWorkspaceAnnotationMaxLength {
		return UpdateWorkspaceAnnotationInput{}, ErrInvalidInput
	}
	input.Annotation = &annotation
	return input, nil
}

// normalizeFiles 规范化项目文件列表并校验路径唯一性。
// 返回规范化后的文件切片；当输入为空、任一文件无效或存在重复的绝对路径时返回 `ErrInvalidInput`。
func normalizeFiles(files []ProjectFile) ([]ProjectFile, error) {
	if len(files) == 0 {
		return nil, ErrInvalidInput
	}
	normalized := make([]ProjectFile, 0, len(files))
	seenPaths := make(map[string]struct{}, len(files))
	for index, item := range files {
		normalizedItem, err := normalizeProjectFile(item, index)
		if err != nil {
			return nil, err
		}
		if _, exists := seenPaths[normalizedItem.AbsolutePath]; exists {
			return nil, ErrInvalidInput
		}
		seenPaths[normalizedItem.AbsolutePath] = struct{}{}
		normalized = append(normalized, normalizedItem)
	}
	return normalized, nil
}

// normalizeProjectFile 规范化并校验单个项目文件。
// 它会裁剪关键字段的空白，检查必填字段是否为空，并确保顺序索引不小于 0。
// 当文件顺序索引为 0 且位于后续位置时，会使用其在输入中的位置作为顺序索引。
// @param item 要规范化的项目文件。
// @param index 文件在输入列表中的位置。
// @returns 规范化后的项目文件，或 ErrInvalidInput。
func normalizeProjectFile(item ProjectFile, index int) (ProjectFile, error) {
	item.Kind = strings.TrimSpace(item.Kind)
	item.Role = strings.TrimSpace(item.Role)
	item.AbsolutePath = strings.TrimSpace(item.AbsolutePath)
	item.DisplayPath = strings.TrimSpace(item.DisplayPath)
	item.LastObservedHash = strings.TrimSpace(item.LastObservedHash)
	if item.Kind == "" || item.Role == "" || item.AbsolutePath == "" || item.DisplayPath == "" {
		return ProjectFile{}, ErrInvalidInput
	}
	if !isValidFileKind(item.Kind) || !isValidFileRole(item.Role) {
		return ProjectFile{}, ErrInvalidInput
	}
	if item.OrderIndex < 0 {
		return ProjectFile{}, ErrInvalidInput
	}
	if item.OrderIndex == 0 && index > 0 {
		item.OrderIndex = index
	}
	return item, nil
}

// normalizeSnapshot 规范化并校验快照信息。
// @param snapshot 待规范化的快照。
// @returns 规范化后的快照；当 snapshot 为空时返回 nil, nil。若 ConfigHash 为空或 RefreshedAt 为空时间，则返回 ErrInvalidInput。
func normalizeSnapshot(snapshot *Snapshot) (*Snapshot, error) {
	if snapshot == nil {
		return nil, nil
	}
	snapshot.ConfigHash = strings.TrimSpace(snapshot.ConfigHash)
	snapshot.DeclaredServicesDigest = strings.TrimSpace(snapshot.DeclaredServicesDigest)
	if snapshot.ConfigHash == "" || snapshot.RefreshedAt.IsZero() {
		return nil, ErrInvalidInput
	}
	snapshot.RefreshedAt = snapshot.RefreshedAt.UTC()
	return snapshot, nil
}

// normalizeLifecycleConfig trims and deduplicates profiles, applies the default wait timeout, and validates the resulting configuration.
// normalizeLifecycleConfig trims and deduplicates profiles, normalizes additional arguments, and applies lifecycle timeout defaults and bounds.
// It returns ErrInvalidInput if a profile or additional argument is invalid, or if the wait timeout is outside the permitted range.
func normalizeLifecycleConfig(config LifecycleConfig) (LifecycleConfig, error) {
	normalizedProfiles := make([]string, 0, len(config.Profiles))
	seen := make(map[string]struct{}, len(config.Profiles))
	for _, item := range config.Profiles {
		profile := strings.TrimSpace(item)
		if profile == "" {
			return LifecycleConfig{}, ErrInvalidInput
		}
		if _, exists := seen[profile]; exists {
			continue
		}
		seen[profile] = struct{}{}
		normalizedProfiles = append(normalizedProfiles, profile)
	}
	config.Profiles = normalizedProfiles
	additionalArgs, err := normalizeLifecycleAdditionalArgs(config.AdditionalArgs)
	if err != nil {
		return LifecycleConfig{}, err
	}
	config.AdditionalArgs = additionalArgs
	if config.WaitTimeoutSeconds == 0 {
		config.WaitTimeoutSeconds = defaultLifecycleWaitTimeoutSeconds
	}
	if config.WaitTimeoutSeconds < minLifecycleWaitTimeoutSeconds || config.WaitTimeoutSeconds > maxLifecycleWaitTimeoutSeconds {
		return LifecycleConfig{}, ErrInvalidInput
	}
	return config, nil
}

// normalizeLifecycleAdditionalArgs trims and validates lifecycle configuration arguments.
// It returns the normalized arguments, or ErrInvalidInput if the count or any argument
// exceeds the allowed constraints or contains invalid content.
func normalizeLifecycleAdditionalArgs(values []string) ([]string, error) {
	normalized, valid := projectcontract.NormalizeLifecycleAdditionalArgs(values)
	if !valid {
		return nil, ErrInvalidInput
	}
	return normalized, nil
}

// normalizeTemporalPointers 将提供的时间指针统一转换为 UTC。
//
// 对每个非 nil 的时间指针，都会将其指向的时间值转换为 UTC 并写回。
func normalizeTemporalPointers(values ...**time.Time) {
	for _, value := range values {
		if value == nil || *value == nil {
			continue
		}
		utc := (**value).UTC()
		*value = &utc
	}
}

func validateImportContracts(input ImportProjectInput) error {
	switch {
	case !isValidCanonicalProjectNameSource(input.CanonicalProjectNameSource):
		return fmt.Errorf("unsupported canonical project name source %q: %w", input.CanonicalProjectNameSource, ErrInvalidInput)
	case !isValidSourceKind(input.SourceKind):
		return fmt.Errorf("unsupported project source kind %q: %w", input.SourceKind, ErrInvalidInput)
	case !isValidHostScope(input.HostScope):
		return fmt.Errorf("unsupported project host scope %q: %w", input.HostScope, ErrInvalidInput)
	case !isValidOwnershipMode(input.OwnershipMode):
		return fmt.Errorf("unsupported project ownership mode %q: %w", input.OwnershipMode, ErrInvalidInput)
	case !isValidLifecycleStrategyKind(input.LifecycleStrategyKind):
		return fmt.Errorf("unsupported project lifecycle strategy %q: %w", input.LifecycleStrategyKind, ErrInvalidInput)
	case !isValidLifecycleReviewStatus(input.LifecycleReviewStatus):
		return fmt.Errorf("unsupported project lifecycle review status %q: %w", input.LifecycleReviewStatus, ErrInvalidInput)
	case !isValidDriftStatus(input.DriftStatus):
		return fmt.Errorf("unsupported project drift status %q: %w", input.DriftStatus, ErrInvalidInput)
	default:
		return nil
	}
}

func validateRefreshContracts(input RefreshProjectInput) error {
	if !isValidDriftStatus(input.DriftStatus) {
		return ErrInvalidInput
	}
	return nil
}

func normalizeOptionalContractValue(value string, valid func(string) bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !valid(value) {
		return "", ErrInvalidInput
	}
	return value, nil
}

// isValidSourceKind reports whether value is a supported project source kind.
func isValidSourceKind(value string) bool {
	switch value {
	case projectcontract.SourceKindImported.String(),
		projectcontract.SourceKindManaged.String(),
		projectcontract.SourceKindTemplate.String():
		return true
	default:
		return false
	}
}

func isValidHostScope(value string) bool {
	return value == projectcontract.HostScopeLocal.String()
}

func isValidOwnershipMode(value string) bool {
	switch value {
	case projectcontract.OwnershipModeExternal.String(),
		projectcontract.OwnershipModeManagedRootDedicated.String():
		return true
	default:
		return false
	}
}

func isValidDriftStatus(value string) bool {
	switch value {
	case projectcontract.DriftStatusUnknown.String(),
		projectcontract.DriftStatusClean.String(),
		projectcontract.DriftStatusChanged.String(),
		projectcontract.DriftStatusMissing.String():
		return true
	default:
		return false
	}
}

func isValidCanonicalProjectNameSource(value string) bool {
	switch value {
	case projectcontract.CanonicalProjectNameSourceComputed.String(),
		projectcontract.CanonicalProjectNameSourceOverride.String():
		return true
	default:
		return false
	}
}

func isValidFileKind(value string) bool {
	switch value {
	case projectcontract.FileKindCompose.String(),
		projectcontract.FileKindEnv.String():
		return true
	default:
		return false
	}
}

func isValidFileRole(value string) bool {
	switch value {
	case projectcontract.FileRolePrimary.String(),
		projectcontract.FileRoleOverride.String(),
		projectcontract.FileRoleEnv.String():
		return true
	default:
		return false
	}
}

func isValidLifecycleStrategyKind(value string) bool {
	return value == projectcontract.LifecycleStrategyKindStandard.String()
}

func isValidLifecycleReviewStatus(value string) bool {
	switch value {
	case projectcontract.LifecycleReviewStatusReviewRequired.String(),
		projectcontract.LifecycleReviewStatusConfirmed.String():
		return true
	default:
		return false
	}
}

// closeRows 关闭 rows 并忽略关闭过程中返回的错误。
func closeRows(rows *sql.Rows) {
	if rows != nil {
		_ = rows.Close()
	}
}

func listPageCapacity(total int, offset int, limit int) int {
	if total <= offset {
		return 0
	}
	remaining := total - offset
	if remaining > limit {
		return limit
	}
	return remaining
}

func toDBArgs(values []uint64) ([]any, error) {
	args := make([]any, 0, len(values))
	for _, value := range values {
		dbID, err := toDBID(value)
		if err != nil {
			return nil, err
		}
		args = append(args, dbID)
	}
	return args, nil
}

func placeholderList(count int) string {
	if count <= 0 {
		return ""
	}
	var builder strings.Builder
	builder.Grow(count*2 - 1)
	for i := 0; i < count; i++ {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteByte('?')
	}
	return builder.String()
}

// rollbackTx 回滚事务并忽略回滚过程中出现的错误。
func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

// scanProject 读取并组装项目记录。
//
// 将查询结果中的可空时间和可空用户 ID 转换为对应的指针字段。
// scanProject 扫描并组装项目记录，解析可空字段及工作区注释、源元数据和生命周期配置。
// scanProject 扫描数据库行并解码项目字段，构造完整的 Project。
// 扫描或数据解码失败时返回错误。
func scanProject(scanner interface{ Scan(dest ...any) error }) (Project, error) {
	var item Project
	var lastDriftCheckedAt sql.NullTime
	var runtimeTargetID sql.NullInt64
	var applicationName sql.NullString
	var createdBy sql.NullInt64
	var updatedBy sql.NullInt64
	var deletedBy sql.NullInt64
	var lifecycleConfigJSON []byte
	var sourceMetadataJSON []byte
	var workspaceAnnotationsJSON []byte
	if err := scanner.Scan(
		&item.ID,
		&item.ApplicationID,
		&applicationName,
		&item.WorkspacePath,
		&item.ComposeProjectName,
		&item.ComposeProjectNameSource,
		&runtimeTargetID,
		&item.DisplayName,
		&item.CanonicalProjectName,
		&item.CanonicalProjectNameSource,
		&item.SourceKind,
		&item.HostScope,
		&item.WorkingDirectory,
		&item.OwnershipMode,
		&sourceMetadataJSON,
		&item.LifecycleStrategyKind,
		&item.LifecycleReviewStatus,
		&lifecycleConfigJSON,
		&item.LastObservedConfigHash,
		&workspaceAnnotationsJSON,
		&lastDriftCheckedAt,
		&item.DriftStatus,
		&createdBy,
		&updatedBy,
		&deletedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return Project{}, err
	}
	item.LastDriftCheckedAt = nullableTime(lastDriftCheckedAt)
	item.RuntimeTargetID = nullableUint64(runtimeTargetID)
	if applicationName.Valid {
		value := applicationName.String
		item.ApplicationName = &value
	}
	item.CreatedBy = nullableUint64(createdBy)
	item.UpdatedBy = nullableUint64(updatedBy)
	item.DeletedBy = nullableUint64(deletedBy)
	annotations, err := decodeWorkspaceAnnotationsJSON(workspaceAnnotationsJSON)
	if err != nil {
		return Project{}, err
	}
	item.WorkspaceAnnotations = annotations
	metadata, err := decodeSourceMetadataJSON(sourceMetadataJSON)
	if err != nil {
		return Project{}, err
	}
	item.SourceMetadata = metadata
	config, err := decodeLifecycleConfigJSON(lifecycleConfigJSON)
	if err != nil {
		return Project{}, err
	}
	item.LifecycleConfig = config
	return item, nil
}

// scanProjectFile 扫描并构造项目文件记录。
//
// 返回从数据库行中读取的 ProjectFile；扫描失败时返回错误。
func scanProjectFile(scanner interface{ Scan(dest ...any) error }) (ProjectFile, error) {
	var item ProjectFile
	if err := scanner.Scan(
		&item.ID,
		&item.ProjectID,
		&item.Kind,
		&item.Role,
		&item.AbsolutePath,
		&item.DisplayPath,
		&item.OrderIndex,
		&item.LastObservedHash,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return ProjectFile{}, err
	}
	return item, nil
}

func scanProjectFileSummary(scanner interface{ Scan(dest ...any) error }) (ProjectFile, error) {
	var item ProjectFile
	if err := scanner.Scan(
		&item.ID,
		&item.ProjectID,
		&item.Kind,
		&item.Role,
		&item.AbsolutePath,
		&item.DisplayPath,
		&item.OrderIndex,
		&item.LastObservedHash,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return ProjectFile{}, err
	}
	return item, nil
}

// scanSnapshot 扫描并返回快照记录。
//
// 扫描项目 ID、规范化 Compose JSON、配置哈希、声明的服务数量、声明的服务摘要和刷新时间。
//
// @returns 成功时返回扫描得到的 Snapshot；扫描失败时返回错误。
func scanSnapshot(scanner interface{ Scan(dest ...any) error }) (Snapshot, error) {
	var item Snapshot
	if err := scanner.Scan(
		&item.ProjectID,
		&item.NormalizedComposeJSON,
		&item.ConfigHash,
		&item.DeclaredServiceCount,
		&item.DeclaredServicesDigest,
		&item.RefreshedAt,
	); err != nil {
		return Snapshot{}, err
	}
	return item, nil
}

// nullableTime 将有效的数据库时间转换为 UTC 时间指针。
//
// @return 有效时返回指向 UTC 时间的指针；无效时返回 nil。
func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

// nullableUint64 将可空整数转换为 uint64 指针。
// 当值无效或小于等于 0 时，返回 nil。
// @returns 有效且大于 0 时对应的 uint64 指针，否则为 nil。
func nullableUint64(value sql.NullInt64) *uint64 {
	if !value.Valid || value.Int64 <= 0 {
		return nil
	}
	v := uint64(value.Int64)
	return &v
}

func encodeLifecycleConfigJSON(config LifecycleConfig) ([]byte, error) {
	normalized, err := normalizeLifecycleConfig(config)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, ErrInvalidInput
	}
	return encoded, nil
}

// 如果数据为空、格式无效、缺少必需字段或配置值无效，则返回 ErrInvalidInput。
func decodeLifecycleConfigJSON(raw []byte) (LifecycleConfig, error) {
	if len(raw) == 0 {
		return LifecycleConfig{}, ErrInvalidInput
	}
	payload, err := unmarshalLifecycleConfigPayload(raw)
	if err != nil {
		return LifecycleConfig{}, err
	}
	config, err := payload.lifecycleConfig()
	if err != nil {
		return LifecycleConfig{}, err
	}
	return normalizeLifecycleConfig(config)
}

type lifecycleConfigPayload struct {
	Profiles                 *[]string `json:"profiles"`
	DownBeforeRedeploy       *bool     `json:"down_before_redeploy"`
	PullBeforeRedeploy       *bool     `json:"pull_before_redeploy"`
	BuildBeforeUp            *bool     `json:"build_before_up"`
	ForceRecreate            *bool     `json:"force_recreate"`
	RemoveOrphans            *bool     `json:"remove_orphans"`
	WaitAfterUp              *bool     `json:"wait_after_up"`
	WaitTimeoutSeconds       *int      `json:"wait_timeout_seconds"`
	RenewAnonVolumes         *bool     `json:"renew_anon_volumes"`
	PruneImagesAfterRedeploy *bool     `json:"prune_images_after_redeploy"`
	AdditionalArgs           *[]string `json:"additional_args"`
}

// 如果 JSON 数据格式无效，则返回 ErrInvalidInput。
func unmarshalLifecycleConfigPayload(raw []byte) (lifecycleConfigPayload, error) {
	var payload lifecycleConfigPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return lifecycleConfigPayload{}, ErrInvalidInput
	}
	return payload, nil
}

func (payload lifecycleConfigPayload) lifecycleConfig() (LifecycleConfig, error) {
	payload.applyLegacyDefaults()
	if err := payload.validateRequiredFields(); err != nil {
		return LifecycleConfig{}, err
	}
	return LifecycleConfig{
		Profiles:                 append([]string(nil), (*payload.Profiles)...),
		DownBeforeRedeploy:       *payload.DownBeforeRedeploy,
		PullBeforeRedeploy:       *payload.PullBeforeRedeploy,
		BuildBeforeUp:            *payload.BuildBeforeUp,
		ForceRecreate:            *payload.ForceRecreate,
		RemoveOrphans:            *payload.RemoveOrphans,
		WaitAfterUp:              *payload.WaitAfterUp,
		WaitTimeoutSeconds:       *payload.WaitTimeoutSeconds,
		RenewAnonVolumes:         *payload.RenewAnonVolumes,
		PruneImagesAfterRedeploy: *payload.PruneImagesAfterRedeploy,
		AdditionalArgs:           append([]string(nil), (*payload.AdditionalArgs)...),
	}, nil
}

// applyLegacyDefaults keeps rows written before lifecycle configuration support readable.
func (payload *lifecycleConfigPayload) applyLegacyDefaults() {
	payload.Profiles = lifecycleSliceOrDefault(payload.Profiles, []string{})
	payload.DownBeforeRedeploy = lifecycleBoolOrDefault(payload.DownBeforeRedeploy, false)
	payload.PullBeforeRedeploy = lifecycleBoolOrDefault(payload.PullBeforeRedeploy, false)
	payload.BuildBeforeUp = lifecycleBoolOrDefault(payload.BuildBeforeUp, false)
	payload.ForceRecreate = lifecycleBoolOrDefault(payload.ForceRecreate, false)
	payload.RemoveOrphans = lifecycleBoolOrDefault(payload.RemoveOrphans, true)
	payload.WaitAfterUp = lifecycleBoolOrDefault(payload.WaitAfterUp, false)
	payload.WaitTimeoutSeconds = lifecycleIntOrDefault(payload.WaitTimeoutSeconds, defaultLifecycleWaitTimeoutSeconds)
	payload.RenewAnonVolumes = lifecycleBoolOrDefault(payload.RenewAnonVolumes, false)
	payload.PruneImagesAfterRedeploy = lifecycleBoolOrDefault(payload.PruneImagesAfterRedeploy, false)
	payload.AdditionalArgs = lifecycleSliceOrDefault(payload.AdditionalArgs, []string{})
}

// lifecycleSliceOrDefault 返回 value 指向的切片；当 value 为 nil 时返回 fallback 的地址。
func lifecycleSliceOrDefault(value *[]string, fallback []string) *[]string {
	if value != nil {
		return value
	}
	return &fallback
}

func lifecycleBoolOrDefault(value *bool, fallback bool) *bool {
	if value != nil {
		return value
	}
	return &fallback
}

func lifecycleIntOrDefault(value *int, fallback int) *int {
	if value != nil {
		return value
	}
	return &fallback
}

func (payload lifecycleConfigPayload) validateRequiredFields() error {
	required := []bool{
		payload.Profiles != nil,
		payload.DownBeforeRedeploy != nil,
		payload.PullBeforeRedeploy != nil,
		payload.BuildBeforeUp != nil,
		payload.ForceRecreate != nil,
		payload.RemoveOrphans != nil,
		payload.WaitAfterUp != nil,
		payload.WaitTimeoutSeconds != nil,
		payload.RenewAnonVolumes != nil,
		payload.PruneImagesAfterRedeploy != nil,
		payload.AdditionalArgs != nil,
	}
	for _, present := range required {
		if !present {
			return ErrInvalidInput
		}
	}
	return nil
}

// encodeWorkspaceAnnotationsJSON 将工作区注释规范化并编码为 JSON。
// 如果注释无效或编码失败，则返回 ErrInvalidInput。
func encodeWorkspaceAnnotationsJSON(annotations map[string]string) ([]byte, error) {
	normalized, err := normalizeWorkspaceAnnotations(annotations)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, ErrInvalidInput
	}
	return encoded, nil
}

func decodeWorkspaceAnnotationsJSON(raw []byte) (map[string]string, error) {
	if len(raw) == 0 {
		return map[string]string{}, nil
	}
	var annotations map[string]string
	if err := json.Unmarshal(raw, &annotations); err != nil {
		return nil, ErrInvalidInput
	}
	return normalizeWorkspaceAnnotations(annotations)
}

func normalizeWorkspaceAnnotations(annotations map[string]string) (map[string]string, error) {
	if len(annotations) == 0 {
		return map[string]string{}, nil
	}
	normalized := make(map[string]string, len(annotations))
	for key, value := range annotations {
		path := normalizeWorkspaceAnnotationPath(key)
		if path == "" {
			return nil, ErrInvalidInput
		}
		note := strings.TrimSpace(value)
		if note == "" {
			continue
		}
		normalized[path] = note
	}
	return normalized, nil
}

func normalizeWorkspaceAnnotationPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	normalized := filepath.ToSlash(filepath.Clean(trimmed))
	normalized = strings.TrimPrefix(normalized, "./")
	if normalized == "" || normalized == "." || strings.HasPrefix(normalized, "../") || filepath.IsAbs(normalized) {
		return ""
	}
	return normalized
}

type placeholderStyle int

const (
	placeholderDollar placeholderStyle = iota
	placeholderQuestion
)

// detectPlaceholderStyle 根据数据库驱动选择参数占位符风格。
// 当 db 为空，或驱动类型不匹配 PostgreSQL 驱动时，返回 `?` 风格；当驱动包路径包含 `pgx` 或 `pq` 时，返回 `$1` 风格。
func detectPlaceholderStyle(db *sql.DB) placeholderStyle {
	if db == nil {
		return placeholderQuestion
	}
	driverType := reflect.TypeOf(db.Driver())
	for driverType != nil {
		pkgPath := driverType.PkgPath()
		if strings.Contains(pkgPath, "pgx") || strings.Contains(pkgPath, "pq") {
			return placeholderDollar
		}
		if driverType.Kind() != reflect.Pointer {
			break
		}
		driverType = driverType.Elem()
	}
	return placeholderQuestion
}

func (s placeholderStyle) rebind(query string) string {
	if s != placeholderDollar {
		return query
	}
	var builder strings.Builder
	index := 1
	for _, r := range query {
		if r == '?' {
			builder.WriteByte('$')
			builder.WriteString(strconv.Itoa(index))
			index++
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func (s placeholderStyle) jsonParamExpr() string {
	if s == placeholderDollar {
		return "?::jsonb"
	}
	return "?"
}

// toDBID 将 uint64 主键值转换为数据库可用的 int64。
// 当值为 0 或超出 int64 可表示范围时，返回 ErrInvalidInput。
// @returns 转换后的 int64 值；当输入无效时返回 0 和 ErrInvalidInput。
func toDBID(value uint64) (int64, error) {
	if value == 0 || value > math.MaxInt64 {
		return 0, ErrInvalidInput
	}
	return int64(value), nil
}

// isUniqueViolation 判断错误是否为 PostgreSQL 唯一约束冲突。
// 当错误可解析为 `pgconn.PgError` 且错误码为 `23505` 时返回 `true`，否则返回 `false`。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// mapWriteErr 为写入操作错误添加动作前缀，并将唯一约束冲突映射为项目冲突错误。
// 当检测到 PostgreSQL 唯一约束冲突时，返回包装了 ErrProjectConflict 的错误；否则返回包装原始错误的结果。
func mapWriteErr(action string, err error) error {
	if err == nil {
		return nil
	}
	if isUniqueViolation(err) {
		return fmt.Errorf("%s: %w", action, ErrProjectConflict)
	}
	return fmt.Errorf("%s: %w", action, err)
}
