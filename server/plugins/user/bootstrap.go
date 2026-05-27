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
	"graft/server/internal/pluginapi"
	userstore "graft/server/plugins/user/store"
)

// bootstrapReader 收敛 web 启动阶段依赖的最小后端快照装配。
//
// 该读模型继续停留在 user 插件边界内，避免为了一个受保护的 bootstrap
// 契约，把菜单过滤、locale 快照或权限聚合拆散到 core 或新增共享抽象里。
type bootstrapReader struct {
	auth         userstore.AuthRepository
	rbac         pluginapi.RBACAccessService
	menuRegistry *menu.Registry
	localizer    *i18n.Service
	localeConfig config.I18nConfig
}

const localeFallbackCapacity = 2

const (
	accessControlOrderRoot = iota
	accessControlOrderOverview
	accessControlOrderUsers
	accessControlOrderRoles
	accessControlOrderPermissions
)

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
	rbac pluginapi.RBACAccessService,
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

	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 {
		return bootstrapResponse{}, pluginapi.ErrUnauthenticated
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
			return bootstrapResponse{}, pluginapi.ErrUnauthenticated
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
	menus := make([]bootstrapMenuResponse, 0, len(items))
	for _, item := range items {
		required := strings.TrimSpace(item.Permission)
		if required != "" {
			if _, ok := granted[required]; !ok {
				continue
			}
		}

		menus = append(menus, bootstrapMenuResponse{
			Code:       item.Code,
			Title:      item.Title,
			TitleKey:   item.TitleKey,
			Path:       item.Path,
			Icon:       item.Icon,
			Permission: item.Permission,
		})
	}

	slices.SortStableFunc(menus, compareBootstrapMenus)

	return menus
}

func compareBootstrapMenus(left, right bootstrapMenuResponse) int {
	leftPath := strings.TrimSpace(left.Path)
	rightPath := strings.TrimSpace(right.Path)

	leftOrder, leftManaged := accessControlBootstrapOrder(leftPath)
	rightOrder, rightManaged := accessControlBootstrapOrder(rightPath)
	if leftManaged && rightManaged {
		return leftOrder - rightOrder
	}
	if leftManaged {
		return -1
	}
	if rightManaged {
		return 1
	}

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

func accessControlBootstrapOrder(path string) (int, bool) {
	switch path {
	case "/access-control":
		return accessControlOrderRoot, true
	case "/access-control/overview":
		return accessControlOrderOverview, true
	case "/access-control/users":
		return accessControlOrderUsers, true
	case "/access-control/roles":
		return accessControlOrderRoles, true
	case "/access-control/permissions":
		return accessControlOrderPermissions, true
	default:
		return 0, false
	}
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
