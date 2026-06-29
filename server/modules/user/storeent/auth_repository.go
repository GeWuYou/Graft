package storeent

import (
	"context"
	"errors"
	"fmt"

	usercontract "graft/server/modules/user/contract"
	ent "graft/server/modules/user/ent"
	refreshsessionent "graft/server/modules/user/ent/refreshsession"
	userstore "graft/server/modules/user/store"
)

type authRepository struct {
	client *ent.Client
}

// NewAuthRepository builds the user module's Ent-backed auth/session repository.
func NewAuthRepository(client *ent.Client) (userstore.AuthRepository, error) {
	return newAuthRepository(client)
}

// newAuthRepository 使用给定的 Ent 客户端创建 authRepository。
// 当 client 为空时返回错误。
func newAuthRepository(client *ent.Client) (*authRepository, error) {
	if client == nil {
		return nil, fmt.Errorf("user storeent requires a non-nil ent client")
	}

	return &authRepository{client: client}, nil
}

func (r *authRepository) GetUserCredentialByUsername(ctx context.Context, username string) (userstore.UserCredential, error) {
	record, err := r.queryUserCredentialByUsername(ctx, username)
	if err != nil {
		return userstore.UserCredential{}, err
	}

	return toStoreUserCredential(record), nil
}

func (r *authRepository) SetPasswordHash(ctx context.Context, input userstore.SetPasswordHashInput) error {
	userID, err := userAuthUserID(input.UserID)
	if err != nil {
		return err
	}

	return r.updatePasswordHash(ctx, userID, input)
}

func (r *authRepository) ChangePasswordAndRevokeOtherRefreshSessions(
	ctx context.Context,
	input userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput,
) error {
	userID, err := userAuthUserID(input.UserID)
	if err != nil {
		return err
	}

	tx, rollback, err := beginUserAuthTx(ctx, r.client, "password change")
	if err != nil {
		return err
	}
	defer rollback()

	if err := setUserPasswordInTx(
		ctx,
		tx,
		passwordUpdateTxInput{
			userID:             userID,
			passwordHash:       input.PasswordHash,
			mustChangePassword: input.MustChangePassword,
			changedAt:          input.ChangedAt,
			requireActiveUser:  false,
			contextMessage:     "set user password hash during password change",
		},
	); err != nil {
		return err
	}
	if err := revokeRefreshSessionsInTx(
		ctx,
		tx,
		userID,
		input.ChangedAt,
		&input.CurrentTokenID,
		"revoke other refresh sessions during password change",
	); err != nil {
		return err
	}

	if err := commitPasswordChange(tx); err != nil {
		return err
	}

	return nil
}

func (r *authRepository) EnsureUserCredential(ctx context.Context, input userstore.EnsureUserCredentialInput) (userstore.UserCredential, error) {
	record, err := r.queryUserCredentialByUsername(ctx, input.Username)
	if err == nil {
		return toStoreUserCredential(record), nil
	}
	if !errors.Is(err, userstore.ErrUserNotFound) {
		return userstore.UserCredential{}, fmt.Errorf("query ensured user credential by username: %w", err)
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
				return userstore.UserCredential{}, fmt.Errorf("re-query ensured user credential after conflict: %w", lookupErr)
			}
			return credential, nil
		}

		return userstore.UserCredential{}, fmt.Errorf("create ensured user credential: %w", err)
	}

	return toStoreUserCredential(record), nil
}

func (r *authRepository) CreateRefreshSession(ctx context.Context, input userstore.CreateRefreshSessionInput) (userstore.RefreshSession, error) {
	userID, err := toEntID(input.UserID)
	if err != nil {
		return userstore.RefreshSession{}, err
	}

	record, err := r.client.RefreshSession.Create().
		SetUserID(userID).
		SetTokenID(input.TokenID).
		SetExpiresAt(input.ExpiresAt).
		Save(ctx)
	if err != nil {
		return userstore.RefreshSession{}, fmt.Errorf("create refresh session: %w", err)
	}

	return toStoreRefreshSession(record), nil
}

func (r *authRepository) GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (userstore.RefreshSession, error) {
	record, err := r.client.RefreshSession.Query().
		Where(refreshsessionent.TokenIDEQ(tokenID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return userstore.RefreshSession{}, userstore.ErrRefreshSessionNotFound
		}
		return userstore.RefreshSession{}, fmt.Errorf("query refresh session by token id: %w", err)
	}

	return toStoreRefreshSession(record), nil
}

func (r *authRepository) RevokeRefreshSession(ctx context.Context, input userstore.RevokeRefreshSessionInput) error {
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
		return userstore.ErrRefreshSessionNotFound
	}

	return nil
}

func (r *authRepository) RevokeRefreshSessionsByUserID(ctx context.Context, input userstore.RevokeRefreshSessionsByUserIDInput) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		if errors.Is(err, userstore.ErrInvalidID) {
			return userstore.ErrUserNotFound
		}
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

func (r *authRepository) RevokeOtherRefreshSessionsByUserID(ctx context.Context, input userstore.RevokeOtherRefreshSessionsInput) error {
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

func (r *authRepository) RevokeRefreshSessionByUserID(ctx context.Context, input userstore.RevokeRefreshSessionByUserIDInput) error {
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
		return userstore.ErrRefreshSessionNotFound
	}

	return nil
}

func (r *authRepository) ListActiveRefreshSessionsByUserID(ctx context.Context, input userstore.ListActiveRefreshSessionsByUserIDInput) ([]userstore.RefreshSession, error) {
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

	sessions := make([]userstore.RefreshSession, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, toStoreRefreshSession(record))
	}

	return sessions, nil
}

func (r *authRepository) RotateRefreshSession(ctx context.Context, input userstore.RotateRefreshSessionInput) (userstore.RefreshSession, error) {
	tx, rollback, err := beginUserAuthTx(ctx, r.client, "refresh session rotation")
	if err != nil {
		return userstore.RefreshSession{}, err
	}
	defer rollback()

	current, err := loadActiveRefreshSessionForRotation(ctx, tx, input.CurrentTokenID, input.Now)
	if err != nil {
		return userstore.RefreshSession{}, err
	}
	if err := revokeRefreshSessionForRotation(ctx, tx, current.ID, input); err != nil {
		return userstore.RefreshSession{}, err
	}
	next, err := createRotatedRefreshSession(ctx, tx, current.UserID, input)
	if err != nil {
		return userstore.RefreshSession{}, err
	}
	if err := commitRefreshRotation(tx); err != nil {
		return userstore.RefreshSession{}, err
	}

	return toStoreRefreshSession(next), nil
}

func (r *authRepository) ResetPasswordAndRevokeRefreshSessions(
	ctx context.Context,
	input userstore.ResetPasswordAndRevokeSessionsInput,
) error {
	userID, err := userAuthUserID(input.UserID)
	if err != nil {
		return err
	}

	tx, rollback, err := beginUserAuthTx(ctx, r.client, "reset password")
	if err != nil {
		return err
	}
	defer rollback()

	if err := setUserPasswordInTx(
		ctx,
		tx,
		passwordUpdateTxInput{
			userID:             userID,
			passwordHash:       input.PasswordHash,
			mustChangePassword: input.MustChangePassword,
			changedAt:          input.ChangedAt,
			requireActiveUser:  true,
			contextMessage:     "reset user password",
		},
	); err != nil {
		return err
	}
	if err := revokeRefreshSessionsInTx(
		ctx,
		tx,
		userID,
		input.ChangedAt,
		nil,
		"revoke refresh sessions during password reset",
	); err != nil {
		return err
	}

	if err := commitResetPassword(tx); err != nil {
		return err
	}

	return nil
}

func toStoreRefreshSession(record *ent.RefreshSession) userstore.RefreshSession {
	return userstore.RefreshSession{
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

func toStoreUserCredential(record *ent.User) userstore.UserCredential {
	return userstore.UserCredential{
		UserID:             toStoreID(record.ID),
		Username:           record.Username,
		PasswordHash:       record.PasswordHash,
		MustChangePassword: record.MustChangePassword,
		PasswordChangedAt:  record.PasswordChangedAt,
	}
}
