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

func normalizeListQuery(query ListQuery) ListQuery {
	query.SourceKind = strings.TrimSpace(query.SourceKind)
	query.DriftStatus = strings.TrimSpace(query.DriftStatus)
	query.LastRefreshStatus = strings.TrimSpace(query.LastRefreshStatus)
	if query.Limit <= 0 {
		query.Limit = defaultListLimit
	}
	if query.Limit > maxListLimit {
		query.Limit = maxListLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query
}

func validateImportInput(input ImportProjectInput) (ImportProjectInput, error) {
	input = trimImportInput(input)
	if err := validateRequiredImportFields(input); err != nil {
		return ImportProjectInput{}, ErrInvalidInput
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

func validateUnregisterInput(input UnregisterProjectInput) (UnregisterProjectInput, error) {
	if input.ProjectID == 0 {
		return UnregisterProjectInput{}, ErrInvalidInput
	}
	return input, nil
}

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

func normalizeProjectFile(item ProjectFile, index int) (ProjectFile, error) {
	item.Kind = strings.TrimSpace(item.Kind)
	item.Role = strings.TrimSpace(item.Role)
	item.AbsolutePath = strings.TrimSpace(item.AbsolutePath)
	item.DisplayPath = strings.TrimSpace(item.DisplayPath)
	item.LastObservedHash = strings.TrimSpace(item.LastObservedHash)
	if item.Kind == "" || item.Role == "" || item.AbsolutePath == "" || item.DisplayPath == "" {
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

func normalizeTemporalPointers(values ...**time.Time) {
	for _, value := range values {
		if value == nil || *value == nil {
			continue
		}
		utc := (**value).UTC()
		*value = &utc
	}
}

func closeRows(rows *sql.Rows) {
	if rows != nil {
		_ = rows.Close()
	}
}

func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

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

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

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

func toDBID(value uint64) (int64, error) {
	if value == 0 || value > math.MaxInt64 {
		return 0, ErrInvalidInput
	}
	return int64(value), nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func mapWriteErr(action string, err error) error {
	if err == nil {
		return nil
	}
	if isUniqueViolation(err) {
		return fmt.Errorf("%s: %w", action, ErrProjectConflict)
	}
	return fmt.Errorf("%s: %w", action, err)
}
