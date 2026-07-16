package storeent

import (
	"context"
	"fmt"

	authent "graft/server/modules/auth/ent"
	authrefreshsessionent "graft/server/modules/auth/ent/authrefreshsession"
	"graft/server/modules/auth/store"
)

// NewSessionStore 创建基于 Ent 客户端的认证刷新会话存储。
func NewSessionStore(client *authent.Client) (store.SessionStore, error) {
	return newSessionStore(client)
}

type sessionStore struct {
	client *authent.Client
}

// newSessionStore 创建基于指定 Ent 客户端的会话存储；客户端为空时返回错误。
func newSessionStore(client *authent.Client) (*sessionStore, error) {
	if client == nil {
		return nil, fmt.Errorf("auth session store requires a non-nil ent client")
	}
	return &sessionStore{client: client}, nil
}

func (r *sessionStore) CreateRefreshSession(ctx context.Context, input store.CreateRefreshSessionInput) (store.RefreshSession, error) {
	record, err := r.client.AuthRefreshSession.Create().SetUserID(input.UserID).SetTokenID(input.TokenID).SetExpiresAt(input.ExpiresAt).Save(ctx)
	if err != nil {
		return store.RefreshSession{}, fmt.Errorf("create auth refresh session: %w", err)
	}
	return toStoreRefreshSession(record), nil
}

func (r *sessionStore) GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (store.RefreshSession, error) {
	record, err := r.client.AuthRefreshSession.Query().Where(authrefreshsessionent.TokenIDEQ(tokenID)).Only(ctx)
	if err != nil {
		if authent.IsNotFound(err) {
			return store.RefreshSession{}, store.ErrRefreshSessionNotFound
		}
		return store.RefreshSession{}, fmt.Errorf("query auth refresh session by token id: %w", err)
	}
	return toStoreRefreshSession(record), nil
}

func (r *sessionStore) RevokeRefreshSession(ctx context.Context, input store.RevokeRefreshSessionInput) error {
	updater := r.client.AuthRefreshSession.Update().Where(authrefreshsessionent.TokenIDEQ(input.TokenID)).SetRevokedAt(input.RevokedAt)
	if input.ReplacedByTokenID != nil {
		updater.SetReplacedByTokenID(*input.ReplacedByTokenID)
	}
	affected, err := updater.Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke auth refresh session: %w", err)
	}
	if affected == 0 {
		return store.ErrRefreshSessionNotFound
	}
	return nil
}

func (r *sessionStore) RevokeRefreshSessionsByUserID(ctx context.Context, input store.RevokeRefreshSessionsByUserIDInput) error {
	_, err := r.client.AuthRefreshSession.Update().Where(authrefreshsessionent.UserIDEQ(input.UserID), authrefreshsessionent.RevokedAtIsNil()).SetRevokedAt(input.RevokedAt).Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke auth refresh sessions by user id: %w", err)
	}
	return nil
}

func (r *sessionStore) RevokeOtherRefreshSessionsByUserID(ctx context.Context, input store.RevokeOtherRefreshSessionsInput) error {
	_, err := r.client.AuthRefreshSession.Update().Where(authrefreshsessionent.UserIDEQ(input.UserID), authrefreshsessionent.RevokedAtIsNil(), authrefreshsessionent.TokenIDNEQ(input.CurrentTokenID)).SetRevokedAt(input.RevokedAt).Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke other auth refresh sessions by user id: %w", err)
	}
	return nil
}

func (r *sessionStore) RevokeRefreshSessionByUserID(ctx context.Context, input store.RevokeRefreshSessionByUserIDInput) error {
	affected, err := r.client.AuthRefreshSession.Update().Where(
		authrefreshsessionent.UserIDEQ(input.UserID),
		authrefreshsessionent.TokenIDEQ(input.TokenID),
		authrefreshsessionent.RevokedAtIsNil(),
		authrefreshsessionent.ExpiresAtGT(input.RevokedAt),
	).SetRevokedAt(input.RevokedAt).Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke auth refresh session by user id: %w", err)
	}
	if affected == 0 {
		return store.ErrRefreshSessionNotFound
	}
	return nil
}

func (r *sessionStore) ListActiveRefreshSessionsByUserID(ctx context.Context, input store.ListActiveRefreshSessionsByUserIDInput) ([]store.RefreshSession, error) {
	records, err := r.client.AuthRefreshSession.Query().Where(
		authrefreshsessionent.UserIDEQ(input.UserID),
		authrefreshsessionent.RevokedAtIsNil(),
		authrefreshsessionent.ExpiresAtGT(input.Now),
	).Order(authent.Desc(authrefreshsessionent.FieldCreatedAt), authent.Desc(authrefreshsessionent.FieldID)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active auth refresh sessions by user id: %w", err)
	}
	sessions := make([]store.RefreshSession, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, toStoreRefreshSession(record))
	}
	return sessions, nil
}

func (r *sessionStore) RotateRefreshSession(ctx context.Context, input store.RotateRefreshSessionInput) (store.RefreshSession, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return store.RefreshSession{}, fmt.Errorf("begin auth refresh-session rotation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := tx.AuthRefreshSession.Query().Where(authrefreshsessionent.TokenIDEQ(input.CurrentTokenID)).Only(ctx)
	if err != nil {
		if authent.IsNotFound(err) {
			return store.RefreshSession{}, store.ErrRefreshSessionNotFound
		}
		return store.RefreshSession{}, fmt.Errorf("query current auth refresh session for rotation: %w", err)
	}
	if current.RevokedAt != nil || !current.ExpiresAt.After(input.Now) {
		return store.RefreshSession{}, store.ErrRefreshSessionNotFound
	}
	affected, err := tx.AuthRefreshSession.Update().Where(
		authrefreshsessionent.IDEQ(current.ID),
		authrefreshsessionent.RevokedAtIsNil(),
		authrefreshsessionent.ExpiresAtGT(input.Now),
	).SetRevokedAt(input.RevokedAt).SetReplacedByTokenID(input.NewTokenID).Save(ctx)
	if err != nil {
		return store.RefreshSession{}, fmt.Errorf("revoke auth refresh session during rotation: %w", err)
	}
	if affected == 0 {
		return store.RefreshSession{}, store.ErrRefreshSessionNotFound
	}
	next, err := tx.AuthRefreshSession.Create().SetUserID(current.UserID).SetTokenID(input.NewTokenID).SetExpiresAt(input.NewExpiresAt).Save(ctx)
	if err != nil {
		return store.RefreshSession{}, fmt.Errorf("create rotated auth refresh session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return store.RefreshSession{}, fmt.Errorf("commit auth refresh-session rotation: %w", err)
	}
	return toStoreRefreshSession(next), nil
}

// toStoreRefreshSession 将 Ent 刷新会话记录转换为存储层刷新会话模型。
func toStoreRefreshSession(record *authent.AuthRefreshSession) store.RefreshSession {
	return store.RefreshSession{
	//nolint:gosec // Ent ID 来自受控 schema，且保持为正数。
		ID:                uint64(record.ID),
		UserID:            record.UserID,
		TokenID:           record.TokenID,
		ExpiresAt:         record.ExpiresAt,
		RevokedAt:         record.RevokedAt,
		ReplacedByTokenID: record.ReplacedByTokenID,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

var _ store.SessionStore = (*sessionStore)(nil)
