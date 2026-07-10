package storeent

import (
	"context"
	"errors"
	"fmt"
	"time"

	authent "graft/server/modules/auth/ent"
	authcredentialent "graft/server/modules/auth/ent/authcredential"
	authrefreshsessionent "graft/server/modules/auth/ent/authrefreshsession"
	"graft/server/modules/auth/store"
)

func (r *credentialStore) savePasswordHash(ctx context.Context, userID uint64, hash string, mustChange bool, changedAt *time.Time) error {
	record, err := r.client.AuthCredential.Query().Where(authcredentialent.UserIDEQ(userID)).Only(ctx)
	if err != nil && !authent.IsNotFound(err) {
		return fmt.Errorf("query auth credential for password update: %w", err)
	}
	if authent.IsNotFound(err) {
		builder := r.client.AuthCredential.Create().SetUserID(userID).SetPasswordHash(hash).SetMustChangePassword(mustChange)
		if changedAt != nil {
			builder.SetPasswordChangedAt(*changedAt)
		}
		if _, createErr := builder.Save(ctx); createErr != nil {
			return fmt.Errorf("create auth credential: %w", createErr)
		}
		return nil
	}

	updater := record.Update().SetPasswordHash(hash).SetMustChangePassword(mustChange)
	if changedAt != nil {
		updater.SetPasswordChangedAt(*changedAt)
	}
	if err := updater.Exec(ctx); err != nil {
		return fmt.Errorf("update auth credential password: %w", err)
	}
	return nil
}

func (r *credentialStore) updatePasswordAndRevokeSessions(ctx context.Context, userID uint64, hash string, mustChange bool, changedAt time.Time, currentTokenID string) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin auth credential transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	credential, err := tx.AuthCredential.Query().Where(authcredentialent.UserIDEQ(userID)).Only(ctx)
	if err != nil {
		if authent.IsNotFound(err) {
			return store.ErrCredentialNotFound
		}
		return fmt.Errorf("query auth credential in transaction: %w", err)
	}
	if err := credential.Update().SetPasswordHash(hash).SetMustChangePassword(mustChange).SetPasswordChangedAt(changedAt).Exec(ctx); err != nil {
		return fmt.Errorf("update auth credential password in transaction: %w", err)
	}

	updater := tx.AuthRefreshSession.Update().Where(authrefreshsessionent.UserIDEQ(userID), authrefreshsessionent.RevokedAtIsNil())
	if currentTokenID != "" {
		updater.Where(authrefreshsessionent.TokenIDNEQ(currentTokenID))
	}
	if _, err := updater.SetRevokedAt(changedAt).Save(ctx); err != nil {
		return fmt.Errorf("revoke auth refresh sessions in transaction: %w", err)
	}
	if err := tx.Commit(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return fmt.Errorf("commit auth credential transaction: %w", err)
	}
	return nil
}
