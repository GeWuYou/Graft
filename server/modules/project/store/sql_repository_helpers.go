package store

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
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

func (r *SQLRepository) ensureReady() error {
	if r == nil || r.db == nil {
		return errors.New("project repository is unavailable")
	}
	return nil
}

// normalizeListQuery 规范化列表查询参数。
// 它会去除筛选字段首尾空白、校验可选 typed-contract 过滤值，并将分页参数限制在允许范围内。
func normalizeListQuery(query ListQuery) (ListQuery, error) {
	var err error
	query.SourceKind, err = normalizeOptionalContractValue(query.SourceKind, isValidSourceKind)
	if err != nil {
		return ListQuery{}, err
	}
	query.DriftStatus, err = normalizeOptionalContractValue(query.DriftStatus, isValidDriftStatus)
	if err != nil {
		return ListQuery{}, err
	}
	query.LastRefreshStatus, err = normalizeOptionalContractValue(query.LastRefreshStatus, isValidRefreshStatus)
	if err != nil {
		return ListQuery{}, err
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

// validateImportInput 规范化并校验导入项目输入，返回可直接使用的输入值。
// 该函数会修剪字符串字段、校验必填字段、规范化文件和快照，并将时间指针统一转换为 UTC。
func validateImportInput(input ImportProjectInput) (ImportProjectInput, error) {
	input = trimImportInput(input)
	if err := validateRequiredImportFields(input); err != nil {
		return ImportProjectInput{}, ErrInvalidInput
	}
	if err := validateImportContracts(input); err != nil {
		return ImportProjectInput{}, err
	}
	files, err := normalizeFiles(input.Files)
	if err != nil {
		return ImportProjectInput{}, err
	}
	input.Files = files
	snapshot, err := normalizeSnapshot(input.Snapshot)
	if err != nil {
		return ImportProjectInput{}, err
	}
	input.Snapshot = snapshot
	normalizeTemporalPointers(&input.LastRefreshAt, &input.LastDriftCheckedAt)
	return input, nil
}

// trimImportInput 去除导入项目输入中字符串字段首尾空白。
func trimImportInput(input ImportProjectInput) ImportProjectInput {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.CanonicalProjectName = strings.TrimSpace(input.CanonicalProjectName)
	input.CanonicalProjectNameSource = strings.TrimSpace(input.CanonicalProjectNameSource)
	input.SourceKind = strings.TrimSpace(input.SourceKind)
	input.HostScope = strings.TrimSpace(input.HostScope)
	input.WorkingDirectory = strings.TrimSpace(input.WorkingDirectory)
	input.OwnershipMode = strings.TrimSpace(input.OwnershipMode)
	input.LastRefreshStatus = strings.TrimSpace(input.LastRefreshStatus)
	input.LastRefreshErrorCode = strings.TrimSpace(input.LastRefreshErrorCode)
	input.LastRefreshErrorMessage = strings.TrimSpace(input.LastRefreshErrorMessage)
	input.LastRefreshConfigHash = strings.TrimSpace(input.LastRefreshConfigHash)
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
		input.LastRefreshStatus,
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
// 它会校验项目 ID，清理刷新状态与漂移状态字段，规范化文件和快照信息，并将时间字段转换为 UTC。
// @param input 待校验的刷新输入。
// @returns 规范化后的刷新输入，或在输入无效时返回 ErrInvalidInput。
func validateRefreshInput(input RefreshProjectInput) (RefreshProjectInput, error) {
	if input.ProjectID == 0 {
		return RefreshProjectInput{}, ErrInvalidInput
	}
	input.LastRefreshStatus = strings.TrimSpace(input.LastRefreshStatus)
	input.LastRefreshErrorCode = strings.TrimSpace(input.LastRefreshErrorCode)
	input.LastRefreshErrorMessage = strings.TrimSpace(input.LastRefreshErrorMessage)
	input.LastRefreshConfigHash = strings.TrimSpace(input.LastRefreshConfigHash)
	input.LastObservedConfigHash = strings.TrimSpace(input.LastObservedConfigHash)
	input.DriftStatus = strings.TrimSpace(input.DriftStatus)
	if input.LastRefreshStatus == "" || input.DriftStatus == "" {
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
	normalizeTemporalPointers(&input.LastRefreshAt, &input.LastDriftCheckedAt)
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
		return ErrInvalidInput
	case !isValidSourceKind(input.SourceKind):
		return ErrInvalidInput
	case !isValidHostScope(input.HostScope):
		return ErrInvalidInput
	case !isValidOwnershipMode(input.OwnershipMode):
		return ErrInvalidInput
	case !isValidRefreshStatus(input.LastRefreshStatus):
		return ErrInvalidInput
	case !isValidDriftStatus(input.DriftStatus):
		return ErrInvalidInput
	default:
		return nil
	}
}

func validateRefreshContracts(input RefreshProjectInput) error {
	if !isValidRefreshStatus(input.LastRefreshStatus) || !isValidDriftStatus(input.DriftStatus) {
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

func isValidSourceKind(value string) bool {
	switch value {
	case projectcontract.SourceKindImported.String(),
		projectcontract.SourceKindManaged.String(),
		projectcontract.SourceKindGit.String(),
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

func isValidRefreshStatus(value string) bool {
	switch value {
	case projectcontract.RefreshStatusNever.String(),
		projectcontract.RefreshStatusSuccess.String(),
		projectcontract.RefreshStatusFailed.String():
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
// @returns 组装后的项目记录；扫描失败时返回错误。
func scanProject(scanner interface{ Scan(dest ...any) error }) (Project, error) {
	var item Project
	var lastRefreshAt sql.NullTime
	var lastDriftCheckedAt sql.NullTime
	var createdBy sql.NullInt64
	var updatedBy sql.NullInt64
	var deletedBy sql.NullInt64
	if err := scanner.Scan(
		&item.ID,
		&item.DisplayName,
		&item.CanonicalProjectName,
		&item.CanonicalProjectNameSource,
		&item.SourceKind,
		&item.HostScope,
		&item.WorkingDirectory,
		&item.OwnershipMode,
		&item.LastRefreshStatus,
		&lastRefreshAt,
		&item.LastRefreshErrorCode,
		&item.LastRefreshErrorMessage,
		&item.LastRefreshConfigHash,
		&item.LastObservedConfigHash,
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
	item.LastRefreshAt = nullableTime(lastRefreshAt)
	item.LastDriftCheckedAt = nullableTime(lastDriftCheckedAt)
	item.CreatedBy = nullableUint64(createdBy)
	item.UpdatedBy = nullableUint64(updatedBy)
	item.DeletedBy = nullableUint64(deletedBy)
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
		&item.ExistsOnLastRefresh,
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
