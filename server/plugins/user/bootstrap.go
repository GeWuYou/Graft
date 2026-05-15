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
	"graft/server/internal/store"
)

type bootstrapResponse struct {
	User        loginUserResponse       `json:"user"`
	Permissions []string                `json:"permissions"`
	Menus       []bootstrapMenuResponse `json:"menus"`
	Locale      bootstrapLocaleSnapshot `json:"locale"`
}

type bootstrapMenuResponse struct {
	Code       string `json:"code"`
	Title      string `json:"title"`
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

// bootstrapReader 收敛 web 启动阶段依赖的最小后端快照装配。
//
// 该读模型继续停留在 user 插件边界内，避免为了一个受保护的 bootstrap
// 契约，把菜单过滤、locale 快照或权限聚合拆散到 core 或新增共享抽象里。
type bootstrapReader struct {
	rbac         store.RBACRepository
	menuRegistry *menu.Registry
	localizer    *i18n.Service
	localeConfig config.I18nConfig
}

func newBootstrapReader(
	localeConfig config.I18nConfig,
	localizer *i18n.Service,
	menuRegistry *menu.Registry,
	rbac store.RBACRepository,
) bootstrapReader {
	return bootstrapReader{
		rbac:         rbac,
		menuRegistry: menuRegistry,
		localizer:    localizer,
		localeConfig: localeConfig,
	}
}

// Read 返回当前请求主体可见的最小 bootstrap 载荷。
func (r bootstrapReader) Read(ctx context.Context, request *http.Request) (bootstrapResponse, error) {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 {
		return bootstrapResponse{}, pluginapi.ErrUnauthenticated
	}

	permissionCodes, permissionSet, err := r.listPermissionCodes(ctx, requestAuth.User.ID)
	if err != nil {
		return bootstrapResponse{}, err
	}

	return bootstrapResponse{
		User: loginUserResponse{
			ID:          requestAuth.User.ID,
			Username:    requestAuth.User.Username,
			DisplayName: requestAuth.User.DisplayName,
		},
		Permissions: permissionCodes,
		Menus:       filterBootstrapMenus(r.menuRegistry, permissionSet),
		Locale:      r.localeSnapshot(request),
	}, nil
}

func (r bootstrapReader) listPermissionCodes(ctx context.Context, userID uint64) ([]string, map[string]struct{}, error) {
	if r.rbac == nil {
		return nil, nil, errors.New("rbac repository is unavailable")
	}

	permissions, err := r.rbac.ListPermissionsByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	codes := make([]string, 0, len(permissions))
	codeSet := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		code := strings.TrimSpace(permission.Code)
		if code == "" {
			continue
		}
		if _, exists := codeSet[code]; exists {
			continue
		}

		codeSet[code] = struct{}{}
		codes = append(codes, code)
	}

	slices.Sort(codes)
	return codes, codeSet, nil
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
			Path:       item.Path,
			Icon:       item.Icon,
			Permission: item.Permission,
		})
	}

	return menus
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
		for _, locale := range []string{defaultLocale, fallbackLocale} {
			locale = strings.TrimSpace(locale)
			if locale == "" {
				continue
			}
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
