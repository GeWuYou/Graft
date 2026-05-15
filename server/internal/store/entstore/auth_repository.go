package entstore

import (
	"context"
	"errors"
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

// RevokeRefreshSessionsByUserID 吊销某个用户名下全部尚未吊销的刷新会话。
func (r *authRepository) RevokeRefreshSessionsByUserID(ctx context.Context, input store.RevokeRefreshSessionsByUserIDInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return err
	}

	if _, err := r.client.RefreshSession.Update().
		Where(
			entrefreshsession.UserIDEQ(userID),
			entrefreshsession.RevokedAtIsNil(),
		).
		SetRevokedAt(input.RevokedAt).
		Save(ctx); err != nil {
		return fmt.Errorf("revoke refresh sessions by user id: %w", err)
	}

	return nil
}

// RevokeRefreshSessionByUserID 按用户定向吊销一条当前有效的 refresh session。
func (r *authRepository) RevokeRefreshSessionByUserID(ctx context.Context, input store.RevokeRefreshSessionByUserIDInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return err
	}

	affected, err := r.client.RefreshSession.Update().
		Where(
			entrefreshsession.UserIDEQ(userID),
			entrefreshsession.TokenIDEQ(input.TokenID),
			entrefreshsession.RevokedAtIsNil(),
			entrefreshsession.ExpiresAtGT(input.RevokedAt),
		).
		SetRevokedAt(input.RevokedAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke refresh session by user id: %w", err)
	}
	if affected == 0 {
		return store.ErrRefreshSessionNotFound
	}

	return nil
}

// ListActiveRefreshSessionsByUserID 按用户读取当前有效的 refresh session 列表。
func (r *authRepository) ListActiveRefreshSessionsByUserID(ctx context.Context, input store.ListActiveRefreshSessionsByUserIDInput) ([]store.RefreshSession, error) {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return nil, err
	}

	records, err := r.client.RefreshSession.Query().
		Where(
			entrefreshsession.UserIDEQ(userID),
			entrefreshsession.RevokedAtIsNil(),
			entrefreshsession.ExpiresAtGT(input.Now),
		).
		Order(ent.Desc(entrefreshsession.FieldCreatedAt), ent.Desc(entrefreshsession.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active refresh sessions by user id: %w", err)
	}

	sessions := make([]store.RefreshSession, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, toStoreRefreshSession(record))
	}

	return sessions, nil
}

// RotateRefreshSession 以事务方式完成一次 refresh session 轮换。
func (r *authRepository) RotateRefreshSession(ctx context.Context, input store.RotateRefreshSessionInput) (store.RefreshSession, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return store.RefreshSession{}, fmt.Errorf("begin refresh session rotation transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	current, err := tx.RefreshSession.Query().
		Where(entrefreshsession.TokenIDEQ(input.CurrentTokenID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return store.RefreshSession{}, store.ErrRefreshSessionNotFound
		}
		return store.RefreshSession{}, fmt.Errorf("query current refresh session for rotation: %w", err)
	}
	if current.RevokedAt != nil {
		return store.RefreshSession{}, store.ErrRefreshSessionNotFound
	}
	if !current.ExpiresAt.After(input.Now) {
		return store.RefreshSession{}, store.ErrRefreshSessionNotFound
	}

	if _, err = tx.RefreshSession.UpdateOneID(current.ID).
		SetRevokedAt(input.RevokedAt).
		SetReplacedByTokenID(input.NewTokenID).
		Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return store.RefreshSession{}, store.ErrRefreshSessionNotFound
		}
		return store.RefreshSession{}, fmt.Errorf("revoke current refresh session during rotation: %w", err)
	}

	next, err := tx.RefreshSession.Create().
		SetUserID(current.UserID).
		SetTokenID(input.NewTokenID).
		SetExpiresAt(input.NewExpiresAt).
		Save(ctx)
	if err != nil {
		return store.RefreshSession{}, fmt.Errorf("create rotated refresh session: %w", err)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		if errors.Is(commitErr, context.Canceled) || errors.Is(commitErr, context.DeadlineExceeded) {
			return store.RefreshSession{}, commitErr
		}
		return store.RefreshSession{}, fmt.Errorf("commit refresh session rotation transaction: %w", commitErr)
	}

	return toStoreRefreshSession(next), nil
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
