// Package store 持久化 Registry 拥有的外部连接与产物仓库事实。
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNotFound 表示不存在匹配的存活 Registry 资源。
	ErrNotFound = errors.New("registry resource not found")
	// ErrUnauthorized 表示调用方没有有效的发布使用授权。
	ErrUnauthorized = errors.New("registry repository is not assigned to actor")
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
WHERE c.connection_ref = $2 AND r.repository_ref = $3 AND r.deleted_at = 0`
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
WHERE c.connection_ref = $1 AND r.repository_ref = $2 AND r.deleted_at = 0`
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
