package entstore

import (
	"context"
	"errors"
	"fmt"
	"time"

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
		UserID:             toStoreID(record.ID),
		Username:           record.Username,
		PasswordHash:       record.PasswordHash,
		MustChangePassword: record.MustChangePassword,
		PasswordChangedAt:  record.PasswordChangedAt,
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

	updater := r.client.User.UpdateOneID(id).
		SetPasswordHash(input.PasswordHash).
		SetMustChangePassword(input.MustChangePassword)
	if input.ChangedAt != nil {
		updater = updater.SetPasswordChangedAt(*input.ChangedAt)
	}

	if err := updater.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return store.ErrUserNotFound
		}
		return fmt.Errorf("set user password hash: %w", err)
	}

	return nil
}

// ChangePasswordAndRevokeOtherRefreshSessions 以事务方式完成改密并保留当前会话。
func (r *authRepository) ChangePasswordAndRevokeOtherRefreshSessions(
	ctx context.Context,
	input store.ChangePasswordAndRevokeOtherRefreshSessionsInput,
) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		if err == store.ErrInvalidID {
			return store.ErrUserNotFound
		}
		return err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin password change transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := tx.User.UpdateOneID(userID).
		SetPasswordHash(input.PasswordHash).
		SetMustChangePassword(input.MustChangePassword).
		SetPasswordChangedAt(input.ChangedAt).
		Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return store.ErrUserNotFound
		}
		return fmt.Errorf("set user password hash during password change: %w", err)
	}

	if _, err := tx.RefreshSession.Update().
		Where(
			entrefreshsession.UserIDEQ(userID),
			entrefreshsession.RevokedAtIsNil(),
			entrefreshsession.TokenIDNEQ(input.CurrentTokenID),
		).
		SetRevokedAt(input.ChangedAt).
		Save(ctx); err != nil {
		return fmt.Errorf("revoke other refresh sessions during password change: %w", err)
	}

	if err := commitPasswordChange(tx); err != nil {
		return err
	}
	committed = true

	return nil
}

// EnsureUserCredential 幂等确保目标用户名的最小认证记录存在。
func (r *authRepository) EnsureUserCredential(ctx context.Context, input store.EnsureUserCredentialInput) (store.UserCredential, error) {
	record, err := r.client.User.Query().
		Where(entuser.UsernameEQ(input.Username)).
		Only(ctx)
	if err == nil {
		return toStoreUserCredential(record), nil
	}
	if !ent.IsNotFound(err) {
		return store.UserCredential{}, fmt.Errorf("query ensured user credential by username: %w", err)
	}

	record, err = r.client.User.Create().
		SetUsername(input.Username).
		SetDisplay(input.Display).
		SetPasswordHash(input.PasswordHash).
		SetMustChangePassword(input.MustChangePassword).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			credential, lookupErr := r.GetUserCredentialByUsername(ctx, input.Username)
			if lookupErr != nil {
				return store.UserCredential{}, fmt.Errorf("re-query ensured user credential after conflict: %w", lookupErr)
			}
			return credential, nil
		}

		return store.UserCredential{}, fmt.Errorf("create ensured user credential: %w", err)
	}

	return toStoreUserCredential(record), nil
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

// RevokeOtherRefreshSessionsByUserID 吊销某个用户除当前 token 外的其它有效 refresh session。
func (r *authRepository) RevokeOtherRefreshSessionsByUserID(ctx context.Context, input store.RevokeOtherRefreshSessionsInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return err
	}

	if _, err := r.client.RefreshSession.Update().
		Where(
			entrefreshsession.UserIDEQ(userID),
			entrefreshsession.RevokedAtIsNil(),
			entrefreshsession.TokenIDNEQ(input.CurrentTokenID),
		).
		SetRevokedAt(input.RevokedAt).
		Save(ctx); err != nil {
		return fmt.Errorf("revoke other refresh sessions by user id: %w", err)
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
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	current, err := loadActiveRefreshSessionForRotation(ctx, tx, input.CurrentTokenID, input.Now)
	if err != nil {
		return store.RefreshSession{}, err
	}
	if err := revokeRefreshSessionForRotation(ctx, tx, current.ID, input); err != nil {
		return store.RefreshSession{}, err
	}
	next, err := createRotatedRefreshSession(ctx, tx, current.UserID, input)
	if err != nil {
		return store.RefreshSession{}, err
	}
	if err := commitRefreshRotation(tx); err != nil {
		return store.RefreshSession{}, err
	}
	committed = true

	return toStoreRefreshSession(next), nil
}

func loadActiveRefreshSessionForRotation(
	ctx context.Context,
	tx *ent.Tx,
	currentTokenID string,
	now time.Time,
) (*ent.RefreshSession, error) {
	current, err := tx.RefreshSession.Query().
		Where(entrefreshsession.TokenIDEQ(currentTokenID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, store.ErrRefreshSessionNotFound
		}
		return nil, fmt.Errorf("query current refresh session for rotation: %w", err)
	}
	if current.RevokedAt != nil || !current.ExpiresAt.After(now) {
		return nil, store.ErrRefreshSessionNotFound
	}

	return current, nil
}

func revokeRefreshSessionForRotation(
	ctx context.Context,
	tx *ent.Tx,
	sessionID int,
	input store.RotateRefreshSessionInput,
) error {
	affected, err := tx.RefreshSession.Update().
		Where(
			entrefreshsession.IDEQ(sessionID),
			entrefreshsession.RevokedAtIsNil(),
			entrefreshsession.ExpiresAtGT(input.Now),
		).
		SetRevokedAt(input.RevokedAt).
		SetReplacedByTokenID(input.NewTokenID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke current refresh session during rotation: %w", err)
	}
	if affected == 0 {
		return store.ErrRefreshSessionNotFound
	}

	return nil
}

func createRotatedRefreshSession(
	ctx context.Context,
	tx *ent.Tx,
	userID int,
	input store.RotateRefreshSessionInput,
) (*ent.RefreshSession, error) {
	next, err := tx.RefreshSession.Create().
		SetUserID(userID).
		SetTokenID(input.NewTokenID).
		SetExpiresAt(input.NewExpiresAt).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create rotated refresh session: %w", err)
	}

	return next, nil
}

func commitRefreshRotation(tx *ent.Tx) error {
	if commitErr := tx.Commit(); commitErr != nil {
		if errors.Is(commitErr, context.Canceled) || errors.Is(commitErr, context.DeadlineExceeded) {
			return commitErr
		}
		return fmt.Errorf("commit refresh session rotation transaction: %w", commitErr)
	}

	return nil
}

func commitPasswordChange(tx *ent.Tx) error {
	if commitErr := tx.Commit(); commitErr != nil {
		if errors.Is(commitErr, context.Canceled) || errors.Is(commitErr, context.DeadlineExceeded) {
			return commitErr
		}
		return fmt.Errorf("commit password change transaction: %w", commitErr)
	}

	return nil
}

// toStoreRefreshSession 把 Ent refresh session 记录收敛为稳定仓储 DTO。
func toStoreRefreshSession(record *ent.RefreshSession) store.RefreshSession {
	return store.RefreshSession{
		ID:                toStoreID(record.ID),
		UserID:            toStoreID(record.UserID),
		TokenID:           record.TokenID,
		ExpiresAt:         record.ExpiresAt,
		RevokedAt:         record.RevokedAt,
		ReplacedByTokenID: record.ReplacedByTokenID,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

func toStoreUserCredential(record *ent.User) store.UserCredential {
	return store.UserCredential{
		UserID:             toStoreID(record.ID),
		Username:           record.Username,
		PasswordHash:       record.PasswordHash,
		MustChangePassword: record.MustChangePassword,
		PasswordChangedAt:  record.PasswordChangedAt,
	}
}
