package storeent

import (
	"context"
	"errors"
	"fmt"
	"time"

	ent "graft/server/modules/user/ent"
	refreshsessionent "graft/server/modules/user/ent/refreshsession"
	userent "graft/server/modules/user/ent/user"
	userstore "graft/server/modules/user/store"
)

func userAuthUserID(inputUserID uint64) (int, error) {
	userID, err := toEntID(inputUserID)
	if err != nil {
		if errors.Is(err, userstore.ErrInvalidID) {
			return 0, userstore.ErrUserNotFound
		}
		return 0, err
	}
	return userID, nil
}

func (r *authRepository) queryUserCredentialByUsername(ctx context.Context, username string) (*ent.User, error) {
	record, err := r.client.User.Query().
		Where(
			userent.UsernameEQ(username),
			userent.DeletedAtEQ(0),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, userstore.ErrUserNotFound
		}
		return nil, fmt.Errorf("query user credential by username: %w", err)
	}
	return record, nil
}

func (r *authRepository) updatePasswordHash(ctx context.Context, userID int, input userstore.SetPasswordHashInput) error {
	updater := r.client.User.UpdateOneID(userID).
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

func beginUserAuthTx(ctx context.Context, client *ent.Client, action string) (*ent.Tx, func(), error) {
	tx, err := client.Tx(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin %s transaction: %w", action, err)
	}

	cleanup := func() {
		_ = tx.Rollback()
	}
	return tx, cleanup, nil
}

func commitUserAuthTx(tx *ent.Tx, action string) error {
	if commitErr := tx.Commit(); commitErr != nil {
		if errors.Is(commitErr, context.Canceled) || errors.Is(commitErr, context.DeadlineExceeded) {
			return commitErr
		}
		return fmt.Errorf("commit %s transaction: %w", action, commitErr)
	}

	return nil
}

type passwordUpdateTxInput struct {
	userID             int
	passwordHash       string
	mustChangePassword bool
	changedAt          time.Time
	requireActiveUser  bool
	contextMessage     string
}

func setUserPasswordInTx(
	ctx context.Context,
	tx *ent.Tx,
	input passwordUpdateTxInput,
) error {
	updater := tx.User.UpdateOneID(input.userID)
	if input.requireActiveUser {
		updater = updater.Where(userent.DeletedAtEQ(0))
	}
	updater = updater.
		SetPasswordHash(input.passwordHash).
		SetMustChangePassword(input.mustChangePassword).
		SetPasswordChangedAt(input.changedAt)

	if err := updater.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return userstore.ErrUserNotFound
		}
		return fmt.Errorf("%s: %w", input.contextMessage, err)
	}

	return nil
}

func revokeRefreshSessionsInTx(
	ctx context.Context,
	tx *ent.Tx,
	userID int,
	revokedAt time.Time,
	currentTokenID *string,
	contextMessage string,
) error {
	updater := tx.RefreshSession.Update().
		Where(
			refreshsessionent.UserIDEQ(userID),
			refreshsessionent.RevokedAtIsNil(),
		)
	if currentTokenID != nil {
		updater = updater.Where(refreshsessionent.TokenIDNEQ(*currentTokenID))
	}

	if _, err := updater.SetRevokedAt(revokedAt).Save(ctx); err != nil {
		return fmt.Errorf("%s: %w", contextMessage, err)
	}

	return nil
}

func loadActiveRefreshSessionForRotation(
	ctx context.Context,
	tx *ent.Tx,
	currentTokenID string,
	now time.Time,
) (*ent.RefreshSession, error) {
	current, err := tx.RefreshSession.Query().
		Where(refreshsessionent.TokenIDEQ(currentTokenID)).
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
			refreshsessionent.IDEQ(sessionID),
			refreshsessionent.RevokedAtIsNil(),
			refreshsessionent.ExpiresAtGT(input.Now),
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
	return commitUserAuthTx(tx, "refresh session rotation")
}

func commitPasswordChange(tx *ent.Tx) error {
	return commitUserAuthTx(tx, "password change")
}

func commitResetPassword(tx *ent.Tx) error {
	return commitUserAuthTx(tx, "reset password")
}
