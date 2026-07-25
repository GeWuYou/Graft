package storeent

import (
	"context"
	"fmt"
	"time"

	authent "graft/server/modules/auth/ent"
	authcredentialent "graft/server/modules/auth/ent/authcredential"
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
