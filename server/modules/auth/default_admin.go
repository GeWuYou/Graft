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

// ensureDefaultAdmin 按 user 提供的 profile identity 初始化 auth credential，再向 RBAC 写入默认访问权限。
// 该流程允许重复执行，但不会覆盖已变更的默认管理员密码；依赖或本地化权限资源缺失时返回错误。
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
	credential, err := ensureDefaultAdminCredential(ctx, credentials, profile)
	if err != nil {
		return err
	}
	if err := requireDefaultAdminPasswordChange(ctx, credentials, credential); err != nil {
		return err
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

func ensureDefaultAdminCredential(ctx context.Context, credentials authstore.CredentialStore, profile moduleapi.CurrentUser) (authstore.UserCredential, error) {
	credential, err := credentials.GetUserCredentialByUsername(ctx, profile.Username)
	if errors.Is(err, authstore.ErrCredentialNotFound) {
		return provisionDefaultAdminCredential(ctx, credentials, profile)
	}
	if err != nil {
		return authstore.UserCredential{}, fmt.Errorf("get default admin credential: %w", err)
	}
	return credential, nil
}

func provisionDefaultAdminCredential(ctx context.Context, credentials authstore.CredentialStore, profile moduleapi.CurrentUser) (authstore.UserCredential, error) {
	hash, err := newPasswordHasher().Hash(defaultAdminPassword)
	if err != nil {
		return authstore.UserCredential{}, fmt.Errorf("hash default admin password: %w", err)
	}
	if err := credentials.SetPasswordHash(ctx, authstore.SetPasswordHashInput{UserID: profile.ID, PasswordHash: hash, MustChangePassword: true}); err != nil {
		return authstore.UserCredential{}, fmt.Errorf("provision default admin credential: %w", err)
	}
	credential, err := credentials.GetUserCredentialByUsername(ctx, profile.Username)
	if err != nil {
		return authstore.UserCredential{}, fmt.Errorf("get default admin credential: %w", err)
	}
	return credential, nil
}

func requireDefaultAdminPasswordChange(ctx context.Context, credentials authstore.CredentialStore, credential authstore.UserCredential) error {
	if credential.MustChangePassword || credential.PasswordHash == nil || *credential.PasswordHash == "" {
		return nil
	}
	compareErr := newPasswordHasher().Compare(*credential.PasswordHash, defaultAdminPassword)
	if errors.Is(compareErr, bcrypt.ErrMismatchedHashAndPassword) {
		return nil
	}
	if compareErr != nil {
		return fmt.Errorf("compare default admin password hash: %w", compareErr)
	}
	if err := credentials.SetPasswordHash(ctx, authstore.SetPasswordHashInput{UserID: credential.UserID, PasswordHash: *credential.PasswordHash, MustChangePassword: true, ChangedAt: credential.PasswordChangedAt}); err != nil {
		return fmt.Errorf("mark default admin credential for password change: %w", err)
	}
	return nil
}

// permissionSeedsFromItems 将权限注册项转换为 RBAC bootstrap 所需的本地化种子；缺少稳定显示键或资源时拒绝引导。
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
			description = lookupPermissionSeedMessage(localizer, item.DescriptionKey, description)
		}
		seeds = append(seeds, moduleapi.PermissionSeed{Code: item.Code, Display: lookupPermissionSeedMessage(localizer, item.DisplayKey, item.Name), DisplayKey: item.DisplayKey, Description: description, DescriptionKey: item.DescriptionKey, Module: item.Module})
	}
	return seeds, nil
}

// lookupPermissionSeedMessage 按 canonical message key 读取已注册资源，避免 owner-local key 被误套用默认 core namespace。
func lookupPermissionSeedMessage(localizer *i18n.Service, key string, fallback string) string {
	resources := localizer.RegisteredMessageResources(
		i18n.LocaleTag(localizer.DefaultLocale()),
		i18n.MessageKey(key),
	)
	if len(resources) > 0 {
		return resources[0].Text
	}
	return fallback
}
