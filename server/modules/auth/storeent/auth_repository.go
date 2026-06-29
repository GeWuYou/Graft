// Package storeent provides the Ent-backed auth-owned persistence implementation.
package storeent

import (
	"context"
	"errors"
	"fmt"

	authstore "graft/server/modules/auth/store"
	usercontract "graft/server/modules/user/contract"
	ent "graft/server/modules/user/ent"
	refreshsessionent "graft/server/modules/user/ent/refreshsession"
	userstore "graft/server/modules/user/store"
)

type authRepository struct {
	client *ent.Client
}

// NewAuthRepository builds the auth module's Ent-backed auth/session repository.
func NewAuthRepository(client *ent.Client) (authstore.AuthRepository, error) {
	return newAuthRepository(client)
}

// 如果 client 为空，则返回错误。
func newAuthRepository(client *ent.Client) (*authRepository, error) {
	if client == nil {
		return nil, fmt.Errorf("auth storeent requires a non-nil ent client")
	}

	return &authRepository{client: client}, nil
}

func (r *authRepository) GetUserCredentialByUsername(ctx context.Context, username string) (authstore.UserCredential, error) {
	record, err := r.queryUserCredentialByUsername(ctx, username)
	if err != nil {
		return authstore.UserCredential{}, err
	}

	return toStoreUserCredential(record), nil
}

func (r *authRepository) SetPasswordHash(ctx context.Context, input authstore.SetPasswordHashInput) error {
	userID, err := authUserID(input.UserID)
	if err != nil {
		return err
	}

	return r.updatePasswordHash(ctx, userID, input)
}

func (r *authRepository) ChangePasswordAndRevokeOtherRefreshSessions(
	ctx context.Context,
	input authstore.ChangePasswordAndRevokeOtherRefreshSessionsInput,
) error {
	userID, err := authUserID(input.UserID)
	if err != nil {
		return err
	}

	return runAuthTx(ctx, r.client, "password change", func(tx *ent.Tx) error {
		if err := setUserPasswordInTx(
			ctx,
			tx,
			passwordUpdateTxInput{
				userID:             userID,
				passwordHash:       input.PasswordHash,
				mustChangePassword: input.MustChangePassword,
				changedAt:          input.ChangedAt,
				contextMessage:     "set user password hash during password change",
			},
		); err != nil {
			return err
		}
		if err := revokeOtherRefreshSessionsInTx(
			ctx,
			tx,
			userID,
			input.CurrentTokenID,
			input.ChangedAt,
			"revoke other refresh sessions during password change",
		); err != nil {
			return err
		}

		return nil
	})
}

func (r *authRepository) EnsureUserCredential(ctx context.Context, input authstore.EnsureUserCredentialInput) (authstore.UserCredential, error) {
	record, err := r.queryUserCredentialByUsername(ctx, input.Username)
	if err == nil {
		return toStoreUserCredential(record), nil
	}
	if !errors.Is(err, userstore.ErrUserNotFound) {
		return authstore.UserCredential{}, fmt.Errorf("query ensured user credential by username: %w", err)
	}

	record, err = r.client.User.Create().
		SetUsername(input.Username).
		SetDisplay(input.Display).
		SetStatus(usercontract.UserStatusEnabled).
		SetPasswordHash(input.PasswordHash).
		SetMustChangePassword(input.MustChangePassword).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			credential, lookupErr := r.GetUserCredentialByUsername(ctx, input.Username)
			if lookupErr != nil {
				return authstore.UserCredential{}, fmt.Errorf("re-query ensured user credential after conflict: %w", lookupErr)
			}
			return credential, nil
		}

		return authstore.UserCredential{}, fmt.Errorf("create ensured user credential: %w", err)
	}

	return toStoreUserCredential(record), nil
}

func (r *authRepository) CreateRefreshSession(ctx context.Context, input authstore.CreateRefreshSessionInput) (authstore.RefreshSession, error) {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return authstore.RefreshSession{}, err
	}

	record, err := r.client.RefreshSession.Create().
		SetUserID(userID).
		SetTokenID(input.TokenID).
		SetExpiresAt(input.ExpiresAt).
		Save(ctx)
	if err != nil {
		return authstore.RefreshSession{}, fmt.Errorf("create refresh session: %w", err)
	}

	return toStoreRefreshSession(record), nil
}

func (r *authRepository) GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (authstore.RefreshSession, error) {
	record, err := r.client.RefreshSession.Query().
		Where(refreshsessionent.TokenIDEQ(tokenID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return authstore.RefreshSession{}, authstore.ErrRefreshSessionNotFound
		}
		return authstore.RefreshSession{}, fmt.Errorf("query refresh session by token id: %w", err)
	}

	return toStoreRefreshSession(record), nil
}

func (r *authRepository) RevokeRefreshSession(ctx context.Context, input authstore.RevokeRefreshSessionInput) error {
	updater := r.client.RefreshSession.Update().
		Where(refreshsessionent.TokenIDEQ(input.TokenID)).
		SetRevokedAt(input.RevokedAt)
	if input.ReplacedByTokenID != nil {
		updater = updater.SetReplacedByTokenID(*input.ReplacedByTokenID)
	}

	affected, err := updater.Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	if affected == 0 {
		return authstore.ErrRefreshSessionNotFound
	}

	return nil
}

func (r *authRepository) RevokeRefreshSessionsByUserID(ctx context.Context, input authstore.RevokeRefreshSessionsByUserIDInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return err
	}

	if _, err := r.client.RefreshSession.Update().
		Where(
			refreshsessionent.UserIDEQ(userID),
			refreshsessionent.RevokedAtIsNil(),
		).
		SetRevokedAt(input.RevokedAt).
		Save(ctx); err != nil {
		return fmt.Errorf("revoke refresh sessions by user id: %w", err)
	}

	return nil
}

func (r *authRepository) RevokeOtherRefreshSessionsByUserID(ctx context.Context, input authstore.RevokeOtherRefreshSessionsInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return err
	}

	if _, err := r.client.RefreshSession.Update().
		Where(
			refreshsessionent.UserIDEQ(userID),
			refreshsessionent.RevokedAtIsNil(),
			refreshsessionent.TokenIDNEQ(input.CurrentTokenID),
		).
		SetRevokedAt(input.RevokedAt).
		Save(ctx); err != nil {
		return fmt.Errorf("revoke other refresh sessions by user id: %w", err)
	}

	return nil
}

func (r *authRepository) RevokeRefreshSessionByUserID(ctx context.Context, input authstore.RevokeRefreshSessionByUserIDInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return err
	}

	affected, err := r.client.RefreshSession.Update().
		Where(
			refreshsessionent.UserIDEQ(userID),
			refreshsessionent.TokenIDEQ(input.TokenID),
			refreshsessionent.RevokedAtIsNil(),
			refreshsessionent.ExpiresAtGT(input.RevokedAt),
		).
		SetRevokedAt(input.RevokedAt).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("revoke refresh session by user id: %w", err)
	}
	if affected == 0 {
		return authstore.ErrRefreshSessionNotFound
	}

	return nil
}

func (r *authRepository) ListActiveRefreshSessionsByUserID(ctx context.Context, input authstore.ListActiveRefreshSessionsByUserIDInput) ([]authstore.RefreshSession, error) {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return nil, err
	}

	records, err := r.client.RefreshSession.Query().
		Where(
			refreshsessionent.UserIDEQ(userID),
			refreshsessionent.RevokedAtIsNil(),
			refreshsessionent.ExpiresAtGT(input.Now),
		).
		Order(ent.Desc(refreshsessionent.FieldCreatedAt), ent.Desc(refreshsessionent.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active refresh sessions by user id: %w", err)
	}

	sessions := make([]authstore.RefreshSession, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, toStoreRefreshSession(record))
	}

	return sessions, nil
}

func (r *authRepository) RotateRefreshSession(ctx context.Context, input authstore.RotateRefreshSessionInput) (authstore.RefreshSession, error) {
	tx, rollback, err := beginAuthTx(ctx, r.client, "refresh session rotation")
	if err != nil {
		return authstore.RefreshSession{}, err
	}
	defer rollback()

	current, err := loadActiveRefreshSessionForRotation(ctx, tx, input.CurrentTokenID, input.Now)
	if err != nil {
		return authstore.RefreshSession{}, err
	}
	if err := revokeRefreshSessionForRotation(ctx, tx, current.ID, input); err != nil {
		return authstore.RefreshSession{}, err
	}
	next, err := createRotatedRefreshSession(ctx, tx, current.UserID, input)
	if err != nil {
		return authstore.RefreshSession{}, err
	}
	if err := commitRefreshRotation(tx); err != nil {
		return authstore.RefreshSession{}, err
	}

	return toStoreRefreshSession(next), nil
}

func (r *authRepository) ResetPasswordAndRevokeRefreshSessions(
	ctx context.Context,
	input authstore.ResetPasswordAndRevokeSessionsInput,
) error {
	userID, err := authUserID(input.UserID)
	if err != nil {
		return err
	}

	return runAuthTx(ctx, r.client, "reset password", func(tx *ent.Tx) error {
		if err := setUserPasswordInTx(
			ctx,
			tx,
			passwordUpdateTxInput{
				userID:             userID,
				passwordHash:       input.PasswordHash,
				mustChangePassword: input.MustChangePassword,
				changedAt:          input.ChangedAt,
				contextMessage:     "set user password hash during reset",
			},
		); err != nil {
			return err
		}
		if err := revokeOtherRefreshSessionsInTx(
			ctx,
			tx,
			userID,
			"",
			input.ChangedAt,
			"revoke refresh sessions during reset",
		); err != nil {
			return err
		}

		return nil
	})
}

// toStoreUserCredential 将 Ent 用户记录映射为存储层用户凭据。
// UserID 使用存储层标识转换，其他字段保持记录中的值。
func toStoreUserCredential(record *ent.User) authstore.UserCredential {
	return authstore.UserCredential{
		UserID:             toStoreID(record.ID),
		Username:           record.Username,
		PasswordHash:       record.PasswordHash,
		MustChangePassword: record.MustChangePassword,
		PasswordChangedAt:  record.PasswordChangedAt,
	}
}

func toStoreRefreshSession(record *ent.RefreshSession) authstore.RefreshSession {
	return authstore.RefreshSession{
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
