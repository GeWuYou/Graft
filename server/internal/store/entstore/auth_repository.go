package entstore

import (
	"context"
	"fmt"

	"graft/server/internal/ent"
	entrefreshsession "graft/server/internal/ent/refreshsession"
	entuser "graft/server/internal/ent/user"
	"graft/server/internal/store"
)

type authRepository struct {
	client *ent.Client
}

// GetUserCredentialByUsername 按用户名读取认证所需的最小用户口令信息。
func (r *authRepository) GetUserCredentialByUsername(ctx context.Context, username string) (store.UserCredential, error) {
	record, err := r.client.User.Query().
		Where(entuser.UsernameEQ(username)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return store.UserCredential{}, store.ErrUserNotFound
		}
		return store.UserCredential{}, fmt.Errorf("query user credential by username: %w", err)
	}

	return store.UserCredential{
		UserID:            uint64(record.ID),
		Username:          record.Username,
		PasswordHash:      record.PasswordHash,
		PasswordChangedAt: record.PasswordChangedAt,
	}, nil
}

// SetPasswordHash 为指定用户写入口令散列与最近变更时间。
func (r *authRepository) SetPasswordHash(ctx context.Context, input store.SetPasswordHashInput) error {
	id, err := toEntID(input.UserID)
	if err != nil {
		if err == store.ErrInvalidID {
			return store.ErrUserNotFound
		}
		return err
	}

	if err := r.client.User.UpdateOneID(id).
		SetPasswordHash(input.PasswordHash).
		SetPasswordChangedAt(input.ChangedAt).
		Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return store.ErrUserNotFound
		}
		return fmt.Errorf("set user password hash: %w", err)
	}

	return nil
}

// CreateRefreshSession 持久化一条新的刷新会话记录。
func (r *authRepository) CreateRefreshSession(ctx context.Context, input store.CreateRefreshSessionInput) (store.RefreshSession, error) {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return store.RefreshSession{}, err
	}

	record, err := r.client.RefreshSession.Create().
		SetUserID(userID).
		SetTokenID(input.TokenID).
		SetExpiresAt(input.ExpiresAt).
		Save(ctx)
	if err != nil {
		return store.RefreshSession{}, fmt.Errorf("create refresh session: %w", err)
	}

	return toStoreRefreshSession(record), nil
}

// GetRefreshSessionByTokenID 按 token 标识读取刷新会话状态。
func (r *authRepository) GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (store.RefreshSession, error) {
	record, err := r.client.RefreshSession.Query().
		Where(entrefreshsession.TokenIDEQ(tokenID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return store.RefreshSession{}, store.ErrRefreshSessionNotFound
		}
		return store.RefreshSession{}, fmt.Errorf("query refresh session by token id: %w", err)
	}

	return toStoreRefreshSession(record), nil
}

// RevokeRefreshSession 吊销一条刷新会话，并可选记录轮换后的 token 标识。
func (r *authRepository) RevokeRefreshSession(ctx context.Context, input store.RevokeRefreshSessionInput) error {
	updater := r.client.RefreshSession.Update().
		Where(entrefreshsession.TokenIDEQ(input.TokenID)).
		SetRevokedAt(input.RevokedAt)
	if input.ReplacedByTokenID != nil {
		updater = updater.SetReplacedByTokenID(*input.ReplacedByTokenID)
	}

	affected, err := updater.Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	if affected == 0 {
		return store.ErrRefreshSessionNotFound
	}

	return nil
}

func toStoreRefreshSession(record *ent.RefreshSession) store.RefreshSession {
	return store.RefreshSession{
		ID:                uint64(record.ID),
		UserID:            uint64(record.UserID),
		TokenID:           record.TokenID,
		ExpiresAt:         record.ExpiresAt,
		RevokedAt:         record.RevokedAt,
		ReplacedByTokenID: record.ReplacedByTokenID,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}
