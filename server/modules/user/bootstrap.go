package user

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/moduleapi"
	userstore "graft/server/modules/user/store"
)

// bootstrapReader 收敛 web 启动阶段依赖的最小后端快照装配。
//
// 该读模型继续停留在 user 模块边界内，避免为了一个受保护的 bootstrap
// 契约，把菜单过滤、locale 快照或权限聚合拆散到 core 或新增共享抽象里。
type bootstrapReader struct {
	auth         userstore.AuthRepository
	rbac         moduleapi.RBACAccessService
	menuRegistry *menu.Registry
	localizer    *i18n.Service
	localeConfig config.I18nConfig
}

const localeFallbackCapacity = 2

type bootstrapResponse struct {
	User               loginUserResponse       `json:"user"`
	MustChangePassword bool                    `json:"must_change_password"`
	Roles              []string                `json:"roles"`
	Permissions        []string                `json:"permissions"`
	Menus              []bootstrapMenuResponse `json:"menus"`
	Locale             bootstrapLocaleSnapshot `json:"locale"`
}

type bootstrapMenuResponse struct {
	Code       string `json:"code"`
	Title      string `json:"title"`
	TitleKey   string `json:"title_key,omitempty"`
	Path       string `json:"path"`
	Icon       string `json:"icon"`
	Order      int    `json:"order"`
	Permission string `json:"permission"`
}

type bootstrapLocaleSnapshot struct {
	CurrentLocale    string   `json:"current_locale"`
	DefaultLocale    string   `json:"default_locale"`
	FallbackLocale   string   `json:"fallback_locale"`
	SupportedLocales []string `json:"supported_locales"`
}

func newBootstrapReader(
	localeConfig config.I18nConfig,
	localizer *i18n.Service,
	menuRegistry *menu.Registry,
	auth userstore.AuthRepository,
	rbac moduleapi.RBACAccessService,
) bootstrapReader {
	return bootstrapReader{
		auth:         auth,
		rbac:         rbac,
		menuRegistry: menuRegistry,
		localizer:    localizer,
		localeConfig: localeConfig,
	}
}

// Read 返回当前请求主体可见的最小 bootstrap 载荷。
func (r bootstrapReader) Read(ctx context.Context, request *http.Request) (bootstrapResponse, error) {
	if r.auth == nil {
		return bootstrapResponse{}, errors.New("auth repository is unavailable")
	}

	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 {
		return bootstrapResponse{}, moduleapi.ErrUnauthenticated
	}

	permissionCodes, permissionSet, err := r.listPermissionCodes(ctx, requestAuth.User.ID)
	if err != nil {
		return bootstrapResponse{}, err
	}
	roleNames, err := r.listRoleNames(ctx, requestAuth.User.ID)
	if err != nil {
		return bootstrapResponse{}, err
	}
	credential, err := r.auth.GetUserCredentialByUsername(ctx, requestAuth.User.Username)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return bootstrapResponse{}, moduleapi.ErrUnauthenticated
		}
		return bootstrapResponse{}, err
	}

	return bootstrapResponse{
		User: loginUserResponse{
			ID:          requestAuth.User.ID,
			Username:    requestAuth.User.Username,
			DisplayName: requestAuth.User.DisplayName,
		},
		MustChangePassword: credential.MustChangePassword,
		Roles:              roleNames,
		Permissions:        permissionCodes,
		Menus:              filterBootstrapMenus(r.menuRegistry, permissionSet),
		Locale:             r.localeSnapshot(request),
	}, nil
}

func (r bootstrapReader) listPermissionCodes(ctx context.Context, userID uint64) ([]string, map[string]struct{}, error) {
	if r.rbac == nil {
		return nil, nil, errors.New("rbac access service is unavailable")
	}

	permissions, err := r.rbac.ListPermissionCodesByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	codeSet := make(map[string]struct{}, len(permissions))
	codes := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		code := strings.TrimSpace(permission)
		if code == "" {
			continue
		}
		if _, exists := codeSet[code]; exists {
			continue
		}

		codeSet[code] = struct{}{}
		codes = append(codes, code)
	}

	return codes, codeSet, nil
}

func (r bootstrapReader) listRoleNames(ctx context.Context, userID uint64) ([]string, error) {
	if r.rbac == nil {
		return nil, errors.New("rbac access service is unavailable")
	}

	return r.rbac.ListRoleNamesByUserID(ctx, userID)
}

func filterBootstrapMenus(registry *menu.Registry, granted map[string]struct{}) []bootstrapMenuResponse {
	if registry == nil {
		return []bootstrapMenuResponse{}
	}

	items := registry.Items()
	menusByKey := make(map[string]bootstrapMenuResponse, len(items))
	menuKeys := make([]string, 0, len(items))
	for _, item := range items {
		required := strings.TrimSpace(item.Permission)
		if required != "" {
			if _, ok := granted[required]; !ok {
				continue
			}
		}

		response := bootstrapMenuResponse{
			Code:       item.Code,
			Title:      item.Title,
			TitleKey:   item.TitleKey,
			Path:       item.Path,
			Icon:       item.Icon,
			Order:      item.Order,
			Permission: item.Permission,
		}
		key := bootstrapMenuIdentity(response)
		if existing, ok := menusByKey[key]; ok {
			menusByKey[key] = mergeBootstrapMenu(existing, response)
			continue
		}

		menusByKey[key] = response
		menuKeys = append(menuKeys, key)
	}

	menus := make([]bootstrapMenuResponse, 0, len(menuKeys))
	for _, key := range menuKeys {
		menus = append(menus, menusByKey[key])
	}

	slices.SortStableFunc(menus, compareBootstrapMenus)

	return menus
}

func bootstrapMenuIdentity(item bootstrapMenuResponse) string {
	code := strings.TrimSpace(item.Code)
	if code != "" {
		return "code:" + code
	}

	return "path:" + strings.TrimSpace(item.Path)
}

func mergeBootstrapMenu(existing, next bootstrapMenuResponse) bootstrapMenuResponse {
	merged := existing
	if merged.Title == "" {
		merged.Title = next.Title
	}
	if merged.TitleKey == "" {
		merged.TitleKey = next.TitleKey
	}
	if merged.Path == "" {
		merged.Path = next.Path
	}
	if merged.Icon == "" {
		merged.Icon = next.Icon
	}
	if merged.Permission == "" {
		merged.Permission = next.Permission
	}
	if next.Order < merged.Order {
		merged.Order = next.Order
	}

	return merged
}

func compareBootstrapMenus(left, right bootstrapMenuResponse) int {
	if left.Order != right.Order {
		return left.Order - right.Order
	}

	leftPath := strings.TrimSpace(left.Path)
	rightPath := strings.TrimSpace(right.Path)
	leftDepth := bootstrapPathDepth(leftPath)
	rightDepth := bootstrapPathDepth(rightPath)
	if leftDepth != rightDepth {
		return leftDepth - rightDepth
	}
	if leftPath != rightPath {
		return strings.Compare(leftPath, rightPath)
	}

	return strings.Compare(left.Code, right.Code)
}

func bootstrapPathDepth(path string) int {
	if path == "" {
		return 0
	}

	return strings.Count(strings.Trim(path, "/"), "/") + 1
}

func (r bootstrapReader) localeSnapshot(request *http.Request) bootstrapLocaleSnapshot {
	defaultLocale := strings.TrimSpace(r.localeConfig.DefaultLocale)
	fallbackLocale := strings.TrimSpace(r.localeConfig.FallbackLocale)
	currentLocale := defaultLocale
	if r.localizer != nil {
		if defaultLocale == "" {
			defaultLocale = r.localizer.DefaultLocale()
		}
		if fallbackLocale == "" {
			fallbackLocale = r.localizer.FallbackLocale()
		}
		currentLocale = r.localizer.ResolveRequestLocale(request, "")
	}
	if currentLocale == "" {
		currentLocale = defaultLocale
	}

	supportedLocales := append([]string(nil), r.localeConfig.SupportedLocales...)
	if len(supportedLocales) == 0 {
		seen := make(map[string]struct{}, localeFallbackCapacity)
		for _, locale := range []string{defaultLocale, fallbackLocale} {
			locale = strings.TrimSpace(locale)
			if locale == "" {
				continue
			}
			if _, exists := seen[locale]; exists {
				continue
			}
			seen[locale] = struct{}{}
			supportedLocales = append(supportedLocales, locale)
		}
	}

	return bootstrapLocaleSnapshot{
		CurrentLocale:    currentLocale,
		DefaultLocale:    defaultLocale,
		FallbackLocale:   fallbackLocale,
		SupportedLocales: supportedLocales,
	}
}
