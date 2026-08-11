// Package store 持久化 Registry 拥有的外部连接与产物仓库事实。
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrNotFound 表示不存在匹配的存活 Registry 资源。
	ErrNotFound = errors.New("registry resource not found")
	// ErrUnauthorized 表示调用方没有有效的发布使用授权。
	ErrUnauthorized = errors.New("registry repository is not assigned to actor")
	// ErrConflict 表示 Registry 资源的存活唯一约束或授权状态发生冲突。
	ErrConflict = errors.New("registry resource conflict")
	// ErrSystemManaged 表示系统托管连接不可由 Registry 管理 API 修改或删除。
	ErrSystemManaged = errors.New("registry connection is system managed")
)

// AuthorizedRepository 是消费者可以保留的非秘密解析结果。
type AuthorizedRepository struct {
	ConnectionRef       string
	RepositoryRef       string
	ConnectionAvailable bool
	AllowPull           bool
	AllowPush           bool
	Endpoint            string
	CredentialRef       string
}

// AuthModeAnonymous 与 AuthModeCredentialRef 是连接持久化的认证模式；后者是引用而不是秘密值。
const (
	AuthModeAnonymous     = "anonymous"
	AuthModeCredentialRef = "credential_ref" // #nosec G101 -- 稳定字段名，不是认证材料。
)

// Connection 是 Registry Connection 的管理面非秘密快照。
type Connection struct {
	ConnectionRef             string
	DisplayName               string
	Provider                  string
	Endpoint                  string
	CredentialRef             string
	Enabled                   bool
	Insecure                  bool
	Description               string
	AuthMode                  string
	Availability              bool
	VerificationStatus        string
	LastVerifiedAt            *time.Time
	LastVerificationErrorCode string
	SystemManaged             bool
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// Repository 是 Connection 下受平台策略约束的 Artifact Repository。
type Repository struct {
	ConnectionRef string
	RepositoryRef string
	DisplayName   string
	AllowPull     bool
	AllowPush     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UserAssignment 记录用户对一个 Repository 的有效使用授权。
type UserAssignment struct {
	ConnectionRef string
	RepositoryRef string
	UserID        uint64
	CreatedAt     time.Time
	CreatedBy     uint64
}

// Destination 是 Build 创建页可安全消费的授权目的地。
type Destination struct {
	ConnectionRef  string
	ConnectionName string
	RepositoryRef  string
	RepositoryName string
	AllowPull      bool
	AllowPush      bool
}

// ConnectionInput 是连接创建与更新的非秘密输入。
type ConnectionInput struct {
	ConnectionRef string
	DisplayName   string
	Provider      string
	Endpoint      string
	CredentialRef string
	Enabled       bool
	Insecure      bool
	Description   string
	AuthMode      string
}

// RepositoryInput 是 Repository 策略写入输入。
type RepositoryInput struct {
	RepositoryRef string
	DisplayName   string
	AllowPull     bool
	AllowPush     bool
}

// ManagementRepository owns Registry management persistence without widening Build's resolver boundary.
type ManagementRepository interface {
	ListConnections(ctx context.Context, search string, limit, offset int) ([]Connection, int, error)
	GetConnection(ctx context.Context, connectionRef string) (Connection, error)
	CreateConnection(ctx context.Context, input ConnectionInput, actorID uint64) (Connection, error)
	UpdateConnection(ctx context.Context, connectionRef string, input ConnectionInput, actorID uint64) (Connection, error)
	DeleteConnection(ctx context.Context, connectionRef string, actorID uint64) error
	SetVerification(ctx context.Context, connectionRef string, available bool, status, errorCode string) (Connection, error)
	ListRepositories(ctx context.Context, connectionRef string) ([]Repository, error)
	CreateRepository(ctx context.Context, connectionRef string, input RepositoryInput, actorID uint64) (Repository, error)
	UpdateRepository(ctx context.Context, connectionRef, repositoryRef string, input RepositoryInput, actorID uint64) (Repository, error)
	DeleteRepository(ctx context.Context, connectionRef, repositoryRef string, actorID uint64) error
	ListAssignments(ctx context.Context, connectionRef, repositoryRef string) ([]UserAssignment, error)
	GrantAssignment(ctx context.Context, connectionRef, repositoryRef string, userID, actorID uint64) (UserAssignment, error)
	RevokeAssignment(ctx context.Context, connectionRef, repositoryRef string, userID, actorID uint64) error
	ReplaceAssignments(ctx context.Context, connectionRef, repositoryRef string, userIDs []uint64, actorID uint64) ([]UserAssignment, error)
	ListAvailableDestinations(ctx context.Context, actorID uint64) ([]Destination, error)
}

// DestinationRepository 是 Registry 服务使用的窄持久化契约，阻止 Build 直接查询基础设施表。
type DestinationRepository interface {
	ResolveAuthorizedRepository(context.Context, uint64, string, string) (AuthorizedRepository, error)
	ResolveAuthorizedCopySource(context.Context, uint64, string, string) (AuthorizedRepository, error)
	ResolveRepositoryBinding(context.Context, string, string) (AuthorizedRepository, error)
}

// SQLRepository 拥有 Registry Connection 与 Artifact Repository 的持久化访问。
type SQLRepository struct{ db *sql.DB }

// NewSQLRepository 创建由 Registry 模块拥有的 SQL 仓储。
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("registry repository requires a non-nil sql db")
	}
	return &SQLRepository{db: db}, nil
}

// ResolveAuthorizedRepository 仅在连接存活、仓库允许发布且操作者拥有明确授权时解析产物仓库。
func (r *SQLRepository) ResolveAuthorizedRepository(ctx context.Context, actorID uint64, connectionRef, repositoryRef string) (AuthorizedRepository, error) {
	return r.resolveAuthorizedRepository(ctx, actorID, connectionRef, repositoryRef, "destination")
}

// ResolveAuthorizedCopySource 校验操作者是否可以将仓库用作 OCI copy source；Pull
// authority 刻意与 publication 分离。
func (r *SQLRepository) ResolveAuthorizedCopySource(ctx context.Context, actorID uint64, connectionRef, repositoryRef string) (AuthorizedRepository, error) {
	return r.resolveAuthorizedRepository(ctx, actorID, connectionRef, repositoryRef, "copy source")
}

func (r *SQLRepository) resolveAuthorizedRepository(ctx context.Context, actorID uint64, connectionRef, repositoryRef, purpose string) (AuthorizedRepository, error) {
	if r == nil || r.db == nil || actorID == 0 {
		return AuthorizedRepository{}, ErrNotFound
	}
	connectionRef, repositoryRef = strings.TrimSpace(connectionRef), strings.TrimSpace(repositoryRef)
	if connectionRef == "" || repositoryRef == "" {
		return AuthorizedRepository{}, ErrNotFound
	}
	const query = `SELECT c.connection_ref, r.repository_ref, c.availability, r.allow_pull, r.allow_push
FROM artifact_repositories r
JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0
JOIN artifact_repository_user_assignments a ON a.repository_id = r.id AND a.user_id = $1 AND a.deleted_at = 0
WHERE c.connection_ref = $2 AND r.repository_ref = $3 AND r.deleted_at = 0 AND c.enabled = true`
	var result AuthorizedRepository
	if err := r.db.QueryRowContext(ctx, query, actorID, connectionRef, repositoryRef).Scan(&result.ConnectionRef, &result.RepositoryRef, &result.ConnectionAvailable, &result.AllowPull, &result.AllowPush); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthorizedRepository{}, r.resolveAuthorizationMiss(ctx, actorID, connectionRef, repositoryRef)
		}
		return AuthorizedRepository{}, fmt.Errorf("query registry %s: %w", purpose, err)
	}
	return result, nil
}

// ResolveRepositoryBinding reads provider-private details only at execution time.
func (r *SQLRepository) ResolveRepositoryBinding(ctx context.Context, connectionRef, repositoryRef string) (AuthorizedRepository, error) {
	if r == nil || r.db == nil || strings.TrimSpace(connectionRef) == "" || strings.TrimSpace(repositoryRef) == "" {
		return AuthorizedRepository{}, ErrNotFound
	}
	const query = `SELECT c.connection_ref, r.repository_ref, c.availability, r.allow_pull, r.allow_push, c.endpoint, COALESCE(c.credential_ref, '')
FROM artifact_repositories r
JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0
WHERE c.connection_ref = $1 AND r.repository_ref = $2 AND r.deleted_at = 0 AND c.enabled = true`
	var result AuthorizedRepository
	if err := r.db.QueryRowContext(ctx, query, strings.TrimSpace(connectionRef), strings.TrimSpace(repositoryRef)).Scan(&result.ConnectionRef, &result.RepositoryRef, &result.ConnectionAvailable, &result.AllowPull, &result.AllowPush, &result.Endpoint, &result.CredentialRef); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuthorizedRepository{}, ErrNotFound
		}
		return AuthorizedRepository{}, fmt.Errorf("query registry publication binding: %w", err)
	}
	return result, nil
}

func (r *SQLRepository) resolveAuthorizationMiss(ctx context.Context, actorID uint64, connectionRef, repositoryRef string) error {
	const existsQuery = `SELECT EXISTS (
	SELECT 1 FROM artifact_repositories r
	JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0
	WHERE c.connection_ref = $1 AND r.repository_ref = $2 AND r.deleted_at = 0
)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, existsQuery, connectionRef, repositoryRef).Scan(&exists); err != nil {
		return fmt.Errorf("query registry destination existence: %w", err)
	}
	if !exists || actorID == 0 {
		return ErrNotFound
	}
	return ErrUnauthorized
}

// ListConnections 按管理页搜索和分页语义读取存活连接。
//
//nolint:cyclop // 查询窗口归一化、count 与行扫描必须在同一分页边界内保持一致。
func (r *SQLRepository) ListConnections(ctx context.Context, search string, limit, offset int) ([]Connection, int, error) {
	if r == nil || r.db == nil {
		return nil, 0, ErrNotFound
	}
	search = strings.TrimSpace(search)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	pattern := "%" + strings.ToLower(search) + "%"
	const countQuery = `SELECT COUNT(*) FROM registry_connections
WHERE deleted_at = 0 AND ($1 = '%%' OR LOWER(display_name) LIKE $1 OR LOWER(endpoint) LIKE $1 OR LOWER(connection_ref) LIKE $1)`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, pattern).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count registry connections: %w", err)
	}
	const query = `SELECT connection_ref, display_name, provider, endpoint, COALESCE(credential_ref, ''), enabled, insecure, description, auth_mode, availability, verification_status, last_verified_at, last_verification_error_code, system_managed, created_at, updated_at
FROM registry_connections
WHERE deleted_at = 0 AND ($1 = '%%' OR LOWER(display_name) LIKE $1 OR LOWER(endpoint) LIKE $1 OR LOWER(connection_ref) LIKE $1)
ORDER BY display_name, connection_ref LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, pattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list registry connections: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]Connection, 0)
	for rows.Next() {
		item, scanErr := scanConnection(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate registry connections: %w", err)
	}
	return items, total, nil
}

// GetConnection 返回一个存活连接的管理面快照。
func (r *SQLRepository) GetConnection(ctx context.Context, connectionRef string) (Connection, error) {
	const query = `SELECT connection_ref, display_name, provider, endpoint, COALESCE(credential_ref, ''), enabled, insecure, description, auth_mode, availability, verification_status, last_verified_at, last_verification_error_code, system_managed, created_at, updated_at
FROM registry_connections WHERE connection_ref = $1 AND deleted_at = 0`
	item, err := scanConnection(r.db.QueryRowContext(ctx, query, strings.TrimSpace(connectionRef)))
	if errors.Is(err, sql.ErrNoRows) {
		return Connection{}, ErrNotFound
	}
	if err != nil {
		return Connection{}, fmt.Errorf("read registry connection: %w", err)
	}
	return item, nil
}

// CreateConnection 持久化待验证的连接，默认不可用于 Build。
func (r *SQLRepository) CreateConnection(ctx context.Context, input ConnectionInput, actorID uint64) (Connection, error) {
	const query = `INSERT INTO registry_connections (connection_ref, display_name, provider, endpoint, credential_ref, enabled, insecure, description, auth_mode, availability, verification_status, last_verification_error_code, created_by, updated_by)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, false, 'unknown', '', $10, $10)
RETURNING connection_ref, display_name, provider, endpoint, COALESCE(credential_ref, ''), enabled, insecure, description, auth_mode, availability, verification_status, last_verified_at, last_verification_error_code, system_managed, created_at, updated_at`
	item, err := scanConnection(r.db.QueryRowContext(ctx, query, input.ConnectionRef, input.DisplayName, input.Provider, input.Endpoint, input.CredentialRef, input.Enabled, input.Insecure, input.Description, input.AuthMode, actorID))
	if err != nil {
		return Connection{}, fmt.Errorf("create registry connection: %w", registryWriteError(err))
	}
	return item, nil
}

// UpdateConnection 覆盖连接配置并使先前验证结果失效。
func (r *SQLRepository) UpdateConnection(ctx context.Context, connectionRef string, input ConnectionInput, actorID uint64) (Connection, error) {
	const query = `UPDATE registry_connections SET display_name = $2, provider = $3, endpoint = $4, credential_ref = NULLIF($5, ''), enabled = $6, insecure = $7, description = $8, auth_mode = $9, availability = false, verification_status = 'unknown', last_verified_at = NULL, last_verification_error_code = '', updated_at = NOW(), updated_by = $10
WHERE connection_ref = $1 AND deleted_at = 0 AND system_managed = false
RETURNING connection_ref, display_name, provider, endpoint, COALESCE(credential_ref, ''), enabled, insecure, description, auth_mode, availability, verification_status, last_verified_at, last_verification_error_code, system_managed, created_at, updated_at`
	item, err := scanConnection(r.db.QueryRowContext(ctx, query, strings.TrimSpace(connectionRef), input.DisplayName, input.Provider, input.Endpoint, input.CredentialRef, input.Enabled, input.Insecure, input.Description, input.AuthMode, actorID))
	if errors.Is(err, sql.ErrNoRows) {
		return Connection{}, ErrNotFound
	}
	if err != nil {
		return Connection{}, fmt.Errorf("update registry connection: %w", registryWriteError(err))
	}
	return item, nil
}

// DeleteConnection 只在不存在活跃 Repository 时软删除连接。
func (r *SQLRepository) DeleteConnection(ctx context.Context, connectionRef string, actorID uint64) error {
	const query = `UPDATE registry_connections SET deleted_at = EXTRACT(EPOCH FROM NOW())::BIGINT, deleted_by = $2, updated_at = NOW(), updated_by = $2
WHERE connection_ref = $1 AND deleted_at = 0 AND system_managed = false AND NOT EXISTS (
 SELECT 1 FROM artifact_repositories r WHERE r.connection_id = registry_connections.id AND r.deleted_at = 0
)`
	result, err := r.db.ExecContext(ctx, query, strings.TrimSpace(connectionRef), actorID)
	if err != nil {
		return fmt.Errorf("delete registry connection: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check registry connection delete: %w", err)
	}
	if changed == 0 {
		return ErrNotFound
	}
	return nil
}

// SetVerification 写入验证结果及其脱敏错误码。
func (r *SQLRepository) SetVerification(ctx context.Context, connectionRef string, available bool, status, errorCode string) (Connection, error) {
	const query = `UPDATE registry_connections SET availability = $2, verification_status = $3, last_verified_at = NOW(), last_verification_error_code = $4, updated_at = NOW()
WHERE connection_ref = $1 AND deleted_at = 0
RETURNING connection_ref, display_name, provider, endpoint, COALESCE(credential_ref, ''), enabled, insecure, description, auth_mode, availability, verification_status, last_verified_at, last_verification_error_code, system_managed, created_at, updated_at`
	item, err := scanConnection(r.db.QueryRowContext(ctx, query, strings.TrimSpace(connectionRef), available, status, errorCode))
	if errors.Is(err, sql.ErrNoRows) {
		return Connection{}, ErrNotFound
	}
	if err != nil {
		return Connection{}, fmt.Errorf("update registry verification: %w", err)
	}
	return item, nil
}

// ListRepositories 返回一个连接的存活 Repository。
func (r *SQLRepository) ListRepositories(ctx context.Context, connectionRef string) ([]Repository, error) {
	const query = `SELECT c.connection_ref, r.repository_ref, r.display_name, r.allow_pull, r.allow_push, r.created_at, r.updated_at
FROM artifact_repositories r JOIN registry_connections c ON c.id = r.connection_id
WHERE c.connection_ref = $1 AND c.deleted_at = 0 AND r.deleted_at = 0
ORDER BY r.display_name, r.repository_ref`
	rows, err := r.db.QueryContext(ctx, query, strings.TrimSpace(connectionRef))
	if err != nil {
		return nil, fmt.Errorf("list artifact repositories: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]Repository, 0)
	for rows.Next() {
		var item Repository
		if err := rows.Scan(&item.ConnectionRef, &item.RepositoryRef, &item.DisplayName, &item.AllowPull, &item.AllowPush, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan artifact repository: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artifact repositories: %w", err)
	}
	return items, nil
}

// CreateRepository 在指定连接下创建可授权的 Repository。
func (r *SQLRepository) CreateRepository(ctx context.Context, connectionRef string, input RepositoryInput, actorID uint64) (Repository, error) {
	const query = `INSERT INTO artifact_repositories (connection_id, repository_ref, display_name, allow_pull, allow_push, created_by, updated_by)
SELECT id, $2, $3, $4, $5, $6, $6 FROM registry_connections WHERE connection_ref = $1 AND deleted_at = 0
RETURNING repository_ref, display_name, allow_pull, allow_push, created_at, updated_at`
	var item Repository
	item.ConnectionRef = strings.TrimSpace(connectionRef)
	err := r.db.QueryRowContext(ctx, query, item.ConnectionRef, input.RepositoryRef, input.DisplayName, input.AllowPull, input.AllowPush, actorID).Scan(&item.RepositoryRef, &item.DisplayName, &item.AllowPull, &item.AllowPush, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Repository{}, ErrNotFound
	}
	if err != nil {
		return Repository{}, fmt.Errorf("create artifact repository: %w", registryWriteError(err))
	}
	return item, nil
}

// UpdateRepository 更新 Repository 的显示和 pull/push 策略。
func (r *SQLRepository) UpdateRepository(ctx context.Context, connectionRef, repositoryRef string, input RepositoryInput, actorID uint64) (Repository, error) {
	const query = `UPDATE artifact_repositories r SET repository_ref = $3, display_name = $4, allow_pull = $5, allow_push = $6, updated_at = NOW(), updated_by = $7
FROM registry_connections c WHERE r.connection_id = c.id AND c.connection_ref = $1 AND r.repository_ref = $2 AND c.deleted_at = 0 AND r.deleted_at = 0
RETURNING c.connection_ref, r.repository_ref, r.display_name, r.allow_pull, r.allow_push, r.created_at, r.updated_at`
	var item Repository
	err := r.db.QueryRowContext(ctx, query, strings.TrimSpace(connectionRef), strings.TrimSpace(repositoryRef), input.RepositoryRef, input.DisplayName, input.AllowPull, input.AllowPush, actorID).Scan(&item.ConnectionRef, &item.RepositoryRef, &item.DisplayName, &item.AllowPull, &item.AllowPush, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Repository{}, ErrNotFound
	}
	if err != nil {
		return Repository{}, fmt.Errorf("update artifact repository: %w", registryWriteError(err))
	}
	return item, nil
}

// DeleteRepository 软删除 Repository，保留发布历史引用。
func (r *SQLRepository) DeleteRepository(ctx context.Context, connectionRef, repositoryRef string, actorID uint64) error {
	const query = `UPDATE artifact_repositories r SET deleted_at = EXTRACT(EPOCH FROM NOW())::BIGINT, deleted_by = $3, updated_at = NOW(), updated_by = $3
FROM registry_connections c WHERE r.connection_id = c.id AND c.connection_ref = $1 AND r.repository_ref = $2 AND c.deleted_at = 0 AND r.deleted_at = 0`
	result, err := r.db.ExecContext(ctx, query, strings.TrimSpace(connectionRef), strings.TrimSpace(repositoryRef), actorID)
	if err != nil {
		return fmt.Errorf("delete artifact repository: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check artifact repository delete: %w", err)
	}
	if changed == 0 {
		return ErrNotFound
	}
	return nil
}

// ListAssignments 返回 Repository 的有效用户授权。
func (r *SQLRepository) ListAssignments(ctx context.Context, connectionRef, repositoryRef string) ([]UserAssignment, error) {
	const query = `SELECT c.connection_ref, r.repository_ref, a.user_id, a.created_at, a.created_by
FROM artifact_repository_user_assignments a
JOIN artifact_repositories r ON r.id = a.repository_id AND r.deleted_at = 0
JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0
WHERE c.connection_ref = $1 AND r.repository_ref = $2 AND a.deleted_at = 0 ORDER BY a.user_id`
	rows, err := r.db.QueryContext(ctx, query, strings.TrimSpace(connectionRef), strings.TrimSpace(repositoryRef))
	if err != nil {
		return nil, fmt.Errorf("list artifact repository assignments: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]UserAssignment, 0)
	for rows.Next() {
		var item UserAssignment
		if err := rows.Scan(&item.ConnectionRef, &item.RepositoryRef, &item.UserID, &item.CreatedAt, &item.CreatedBy); err != nil {
			return nil, fmt.Errorf("scan artifact repository assignment: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate artifact repository assignments: %w", err)
	}
	return items, nil
}

// GrantAssignment 原子地验证目标用户与 Repository，并拒绝重复的有效授权。
func (r *SQLRepository) GrantAssignment(ctx context.Context, connectionRef, repositoryRef string, userID, actorID uint64) (UserAssignment, error) {
	if userID == 0 {
		return UserAssignment{}, ErrNotFound
	}
	const query = `INSERT INTO artifact_repository_user_assignments (repository_id, user_id, created_by, updated_by)
	SELECT r.id, $3, $4, $4 FROM artifact_repositories r
	JOIN registry_connections c ON c.id = r.connection_id
	JOIN users u ON u.id = $3
	WHERE c.connection_ref = $1 AND r.repository_ref = $2 AND c.deleted_at = 0 AND r.deleted_at = 0
ON CONFLICT DO NOTHING`
	item := UserAssignment{ConnectionRef: strings.TrimSpace(connectionRef), RepositoryRef: strings.TrimSpace(repositoryRef), UserID: userID}
	result, err := r.db.ExecContext(ctx, query, item.ConnectionRef, item.RepositoryRef, userID, actorID)
	if err != nil {
		return UserAssignment{}, fmt.Errorf("grant artifact repository assignment: %w", registryWriteError(err))
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return UserAssignment{}, fmt.Errorf("check artifact repository assignment grant: %w", err)
	}
	if changed == 0 {
		const selectAssignment = `SELECT 1
	FROM artifact_repository_user_assignments a
	JOIN artifact_repositories r ON r.id = a.repository_id AND r.deleted_at = 0
	JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0
	WHERE c.connection_ref = $1 AND r.repository_ref = $2 AND a.user_id = $3 AND a.deleted_at = 0`
		var exists int
		err := r.db.QueryRowContext(ctx, selectAssignment, item.ConnectionRef, item.RepositoryRef, userID).Scan(&exists)
		if err == nil {
			return UserAssignment{}, ErrConflict
		}
		if errors.Is(err, sql.ErrNoRows) {
			return UserAssignment{}, ErrNotFound
		}
		return UserAssignment{}, fmt.Errorf("check existing artifact repository assignment: %w", err)
	}
	const readAssignment = `SELECT a.created_at, a.created_by
	FROM artifact_repository_user_assignments a
	JOIN artifact_repositories r ON r.id = a.repository_id AND r.deleted_at = 0
	JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0
	WHERE c.connection_ref = $1 AND r.repository_ref = $2 AND a.user_id = $3 AND a.deleted_at = 0`
	if err := r.db.QueryRowContext(ctx, readAssignment, item.ConnectionRef, item.RepositoryRef, userID).Scan(&item.CreatedAt, &item.CreatedBy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserAssignment{}, ErrNotFound
		}
		return UserAssignment{}, fmt.Errorf("read granted artifact repository assignment: %w", err)
	}
	return item, nil
}

// RevokeAssignment 软删除单个用户授权，保留其余用户的并发变更。
func (r *SQLRepository) RevokeAssignment(ctx context.Context, connectionRef, repositoryRef string, userID, actorID uint64) error {
	if userID == 0 {
		return ErrNotFound
	}
	const query = `UPDATE artifact_repository_user_assignments a SET deleted_at = EXTRACT(EPOCH FROM NOW())::BIGINT, deleted_by = $4, updated_at = NOW(), updated_by = $4
FROM artifact_repositories r JOIN registry_connections c ON c.id = r.connection_id
WHERE a.repository_id = r.id AND c.connection_ref = $1 AND r.repository_ref = $2 AND a.user_id = $3
  AND c.deleted_at = 0 AND r.deleted_at = 0 AND a.deleted_at = 0`
	result, err := r.db.ExecContext(ctx, query, strings.TrimSpace(connectionRef), strings.TrimSpace(repositoryRef), userID, actorID)
	if err != nil {
		return fmt.Errorf("revoke artifact repository assignment: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check artifact repository assignment revoke: %w", err)
	}
	if changed == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceAssignments 在一个事务中用完整用户集合替换有效授权。
func (r *SQLRepository) ReplaceAssignments(ctx context.Context, connectionRef, repositoryRef string, userIDs []uint64, actorID uint64) ([]UserAssignment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin artifact repository assignments: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var repositoryID uint64
	const find = `SELECT r.id FROM artifact_repositories r JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0 WHERE c.connection_ref = $1 AND r.repository_ref = $2 AND r.deleted_at = 0`
	if err := tx.QueryRowContext(ctx, find, strings.TrimSpace(connectionRef), strings.TrimSpace(repositoryRef)).Scan(&repositoryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find artifact repository: %w", err)
	}
	seen, err := existingAssignmentUsers(ctx, tx, userIDs)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE artifact_repository_user_assignments SET deleted_at = EXTRACT(EPOCH FROM NOW())::BIGINT, deleted_by = $2, updated_at = NOW(), updated_by = $2 WHERE repository_id = $1 AND deleted_at = 0`, repositoryID, actorID); err != nil {
		return nil, fmt.Errorf("clear artifact repository assignments: %w", err)
	}
	for userID := range seen {
		if _, err := tx.ExecContext(ctx, `INSERT INTO artifact_repository_user_assignments (repository_id, user_id, created_by, updated_by) VALUES ($1, $2, $3, $3)`, repositoryID, userID, actorID); err != nil {
			return nil, fmt.Errorf("create artifact repository assignment: %w", registryWriteError(err))
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit artifact repository assignments: %w", err)
	}
	return r.ListAssignments(ctx, connectionRef, repositoryRef)
}

func existingAssignmentUsers(ctx context.Context, tx *sql.Tx, userIDs []uint64) (map[uint64]struct{}, error) {
	seen := make(map[uint64]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID == 0 {
			return nil, ErrNotFound
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		var userExists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&userExists); err != nil {
			return nil, fmt.Errorf("check replacement artifact repository assignment user: %w", err)
		}
		if !userExists {
			return nil, ErrNotFound
		}
	}
	return seen, nil
}

// ListAvailableDestinations 只返回 actor 已授权且可 push 的可用目的地。
func (r *SQLRepository) ListAvailableDestinations(ctx context.Context, actorID uint64) ([]Destination, error) {
	const query = `SELECT c.connection_ref, c.display_name, r.repository_ref, r.display_name, r.allow_pull, r.allow_push
FROM artifact_repositories r JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0
JOIN artifact_repository_user_assignments a ON a.repository_id = r.id AND a.deleted_at = 0
WHERE a.user_id = $1 AND r.deleted_at = 0 AND c.enabled = true AND c.availability = true AND r.allow_push = true
ORDER BY c.display_name, r.display_name`
	rows, err := r.db.QueryContext(ctx, query, actorID)
	if err != nil {
		return nil, fmt.Errorf("list available registry destinations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]Destination, 0)
	for rows.Next() {
		var item Destination
		if err := rows.Scan(&item.ConnectionRef, &item.ConnectionName, &item.RepositoryRef, &item.RepositoryName, &item.AllowPull, &item.AllowPush); err != nil {
			return nil, fmt.Errorf("scan available registry destination: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate available registry destinations: %w", err)
	}
	return items, nil
}

type rowScanner interface{ Scan(...any) error }

func registryWriteError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) && databaseError.Code == "23505" {
		return ErrConflict
	}
	return err
}

func scanConnection(row rowScanner) (Connection, error) {
	var item Connection
	err := row.Scan(&item.ConnectionRef, &item.DisplayName, &item.Provider, &item.Endpoint, &item.CredentialRef, &item.Enabled, &item.Insecure, &item.Description, &item.AuthMode, &item.Availability, &item.VerificationStatus, &item.LastVerifiedAt, &item.LastVerificationErrorCode, &item.SystemManaged, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return Connection{}, err
	}
	return item, nil
}
