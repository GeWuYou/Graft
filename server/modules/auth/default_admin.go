package auth

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	authstore "graft/server/modules/auth/store"
)

//nolint:cyclop,gocognit,gocyclo,nestif // Bootstrap preserves distinct existing-admin and first-provision semantics.
func ensureDefaultAdmin(ctx context.Context, localizer *i18n.Service, credentials authstore.CredentialStore, identity moduleapi.UserIdentityProvider, rbac moduleapi.RBACBootstrapService, permissions []permission.Item) error {
	if credentials == nil || identity == nil {
		return errors.New("auth default-admin dependencies are unavailable")
	}
	if rbac == nil {
		return errors.New("rbac bootstrap service is unavailable")
	}
	profile, err := identity.EnsureDefaultAdminProfile(ctx)
	if err != nil {
		return fmt.Errorf("ensure default admin profile: %w", err)
	}
	credential, err := credentials.GetUserCredentialByUsername(ctx, profile.Username)
	if errors.Is(err, authstore.ErrCredentialNotFound) {
		hash, hashErr := newPasswordHasher().Hash(defaultAdminPassword)
		if hashErr != nil {
			return fmt.Errorf("hash default admin password: %w", hashErr)
		}
		if setErr := credentials.SetPasswordHash(ctx, authstore.SetPasswordHashInput{UserID: profile.ID, PasswordHash: hash, MustChangePassword: true}); setErr != nil {
			return fmt.Errorf("provision default admin credential: %w", setErr)
		}
		credential, err = credentials.GetUserCredentialByUsername(ctx, profile.Username)
	}
	if err != nil {
		return fmt.Errorf("get default admin credential: %w", err)
	}
	if !credential.MustChangePassword && credential.PasswordHash != nil && *credential.PasswordHash != "" {
		if compareErr := newPasswordHasher().Compare(*credential.PasswordHash, defaultAdminPassword); compareErr == nil {
			if setErr := credentials.SetPasswordHash(ctx, authstore.SetPasswordHashInput{UserID: credential.UserID, PasswordHash: *credential.PasswordHash, MustChangePassword: true, ChangedAt: credential.PasswordChangedAt}); setErr != nil {
				return fmt.Errorf("mark default admin credential for password change: %w", setErr)
			}
		} else if !errors.Is(compareErr, bcrypt.ErrMismatchedHashAndPassword) {
			return fmt.Errorf("compare default admin password hash: %w", compareErr)
		}
	}
	seeds, err := permissionSeedsFromItems(localizer, permissions)
	if err != nil {
		return fmt.Errorf("build default admin permission seeds: %w", err)
	}
	if err := rbac.EnsureDefaultAdminAccess(ctx, credential.UserID, seeds); err != nil {
		return fmt.Errorf("ensure default admin access: %w", err)
	}
	return nil
}

func permissionSeedsFromItems(localizer *i18n.Service, items []permission.Item) ([]moduleapi.PermissionSeed, error) {
	if localizer == nil {
		return nil, errors.New("permission seed localization requires i18n service")
	}
	seeds := make([]moduleapi.PermissionSeed, 0, len(items))
	for _, item := range items {
		if item.DisplayKey == "" {
			return nil, fmt.Errorf("permission seed localization requires stable locale key for %s", item.Code)
		}
		if len(localizer.RegisteredMessageResources(i18n.LocaleTag(localizer.DefaultLocale()), i18n.MessageKey(item.DisplayKey))) == 0 {
			return nil, fmt.Errorf("permission seed localization key missing for %s: %s", item.Code, item.DisplayKey)
		}
		description := item.Description
		if item.DescriptionKey != "" {
			description = localizer.Lookup(i18n.LookupRequest{Locale: i18n.LocaleTag(localizer.DefaultLocale()), Key: i18n.MessageKey(item.DescriptionKey), FallbackMessage: description})
		}
		seeds = append(seeds, moduleapi.PermissionSeed{Code: item.Code, Display: localizer.Lookup(i18n.LookupRequest{Locale: i18n.LocaleTag(localizer.DefaultLocale()), Key: i18n.MessageKey(item.DisplayKey), FallbackMessage: item.Name}), DisplayKey: item.DisplayKey, Description: description, DescriptionKey: item.DescriptionKey, Module: item.Module})
	}
	return seeds, nil
}
