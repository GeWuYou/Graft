package storeent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"graft/server/internal/ent"
	entrefreshsession "graft/server/internal/ent/refreshsession"
	entuser "graft/server/internal/ent/user"
	userstore "graft/server/plugins/user/store"
)

type authRepository struct {
	client *ent.Client
}

// NewAuthRepository builds the user plugin's Ent-backed auth/session repository.
func NewAuthRepository(client *ent.Client) (userstore.AuthRepository, error) {
	if client == nil {
		return nil, fmt.Errorf("user storeent requires a non-nil ent client")
	}

	return &authRepository{client: client}, nil
}

func (r *authRepository) GetUserCredentialByUsername(ctx context.Context, username string) (userstore.UserCredential, error) {
	record, err := r.client.User.Query().
		Where(entuser.UsernameEQ(username)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return userstore.UserCredential{}, userstore.ErrUserNotFound
		}
		return userstore.UserCredential{}, fmt.Errorf("query user credential by username: %w", err)
	}

	return toStoreUserCredential(record), nil
}

func (r *authRepository) SetPasswordHash(ctx context.Context, input userstore.SetPasswordHashInput) error {
	id, err := toEntID(input.UserID)
	if err != nil {
		if err == userstore.ErrInvalidID {
			return userstore.ErrUserNotFound
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
			return userstore.ErrUserNotFound
		}
		return fmt.Errorf("set user password hash: %w", err)
	}

	return nil
}

func (r *authRepository) ChangePasswordAndRevokeOtherRefreshSessions(
	ctx context.Context,
	input userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput,
) error {
	userID, err := toEntID(input.UserID)
	if err != nil {
		if err == userstore.ErrInvalidID {
			return userstore.ErrUserNotFound
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
			return userstore.ErrUserNotFound
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

func (r *authRepository) EnsureUserCredential(ctx context.Context, input userstore.EnsureUserCredentialInput) (userstore.UserCredential, error) {
	record, err := r.client.User.Query().
		Where(entuser.UsernameEQ(input.Username)).
		Only(ctx)
	if err == nil {
		return toStoreUserCredential(record), nil
	}
	if !ent.IsNotFound(err) {
		return userstore.UserCredential{}, fmt.Errorf("query ensured user credential by username: %w", err)
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
		Where(entrefreshsession.TokenIDEQ(tokenID)).
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
		return userstore.ErrRefreshSessionNotFound
	}

	return nil
}

func (r *authRepository) RevokeRefreshSessionsByUserID(ctx context.Context, input userstore.RevokeRefreshSessionsByUserIDInput) error {
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

func (r *authRepository) RevokeOtherRefreshSessionsByUserID(ctx context.Context, input userstore.RevokeOtherRefreshSessionsInput) error {
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

func (r *authRepository) RevokeRefreshSessionByUserID(ctx context.Context, input userstore.RevokeRefreshSessionByUserIDInput) error {
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
			entrefreshsession.UserIDEQ(userID),
			entrefreshsession.RevokedAtIsNil(),
			entrefreshsession.ExpiresAtGT(input.Now),
		).
		Order(ent.Desc(entrefreshsession.FieldCreatedAt), ent.Desc(entrefreshsession.FieldID)).
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
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return userstore.RefreshSession{}, fmt.Errorf("begin refresh session rotation transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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
			return nil, userstore.ErrRefreshSessionNotFound
		}
		return nil, fmt.Errorf("query current refresh session for rotation: %w", err)
	}
	if current.RevokedAt != nil || !current.ExpiresAt.After(now) {
		return nil, userstore.ErrRefreshSessionNotFound
	}

	return current, nil
}

func revokeRefreshSessionForRotation(
	ctx context.Context,
	tx *ent.Tx,
	sessionID int,
	input userstore.RotateRefreshSessionInput,
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
		return userstore.ErrRefreshSessionNotFound
	}

	return nil
}

func createRotatedRefreshSession(
	ctx context.Context,
	tx *ent.Tx,
	userID int,
	input userstore.RotateRefreshSessionInput,
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
