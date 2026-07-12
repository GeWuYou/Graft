package user

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"graft/server/internal/config"
	servicecontainer "graft/server/internal/container"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/moduleapi"
)

// bootstrapReader 收敛 web 启动阶段依赖的最小后端快照装配。
//
// 该读模型继续停留在 user 模块边界内，避免为了一个受保护的 bootstrap
// 契约，把菜单过滤、locale 快照或权限聚合拆散到 core 或新增共享抽象里。
type bootstrapReader struct {
	rbac         moduleapi.RBACAccessService
	menuRegistry *menu.Registry
	systemConfig moduleapi.SystemConfigResolver
	localizer    *i18n.Service
	localeConfig config.I18nConfig
}

const localeFallbackCapacity = 2

type bootstrapResponse struct {
	User               bootstrapUserResponse   `json:"user"`
	MustChangePassword bool                    `json:"must_change_password"`
	Roles              []string                `json:"roles"`
	Permissions        []string                `json:"permissions"`
	Menus              []bootstrapMenuResponse `json:"menus"`
	Locale             bootstrapLocaleSnapshot `json:"locale"`
}

type bootstrapUserResponse struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type bootstrapMenuResponse struct {
	Code       string `json:"code"`
	ParentCode string `json:"parent_code,omitempty"`
	Kind       string `json:"kind"`
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

// newBootstrapReader 创建用于生成用户启动快照的读取器，并解析系统配置服务。
func newBootstrapReader(
	localeConfig config.I18nConfig,
	localizer *i18n.Service,
	menuRegistry *menu.Registry,
	services servicecontainer.Resolver,
	rbac moduleapi.RBACAccessService,
) bootstrapReader {
	return bootstrapReader{
		rbac:         rbac,
		menuRegistry: menuRegistry,
		systemConfig: resolveBootstrapSystemConfig(services),
		localizer:    localizer,
		localeConfig: localeConfig,
	}
}

// Read 返回当前请求主体可见的最小 bootstrap 载荷。
func (r bootstrapReader) Read(ctx context.Context, request *http.Request) (bootstrapResponse, error) {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil || requestAuth.User.ID == 0 {
		return bootstrapResponse{}, moduleapi.ErrUnauthenticated
	}
	if r.menuRegistry != nil {
		if err := r.menuRegistry.Validate(); err != nil {
			return bootstrapResponse{}, err
		}
	}
	permissionCodes, permissionSet, err := r.listPermissionCodes(ctx, requestAuth.User.ID)
	if err != nil {
		return bootstrapResponse{}, err
	}
	roleNames, err := r.listRoleNames(ctx, requestAuth.User.ID)
	if err != nil {
		return bootstrapResponse{}, err
	}
	return bootstrapResponse{
		User: bootstrapUserResponse{
			ID:          requestAuth.User.ID,
			Username:    requestAuth.User.Username,
			DisplayName: requestAuth.User.DisplayName,
		},
		MustChangePassword: false,
		Roles:              roleNames,
		Permissions:        permissionCodes,
		Menus:              r.filterBootstrapMenus(ctx, permissionSet),
		Locale:             r.localeSnapshot(request),
	}, nil
}

// ReadBootstrap exposes only the user-owned bootstrap snapshot. Auth overlays
// credential state from its own store before serving the stable route payload.
func (r bootstrapReader) ReadBootstrap(ctx context.Context, request *http.Request) (moduleapi.AuthBootstrapPayload, error) {
	payload, err := r.Read(ctx, request)
	if err != nil {
		return moduleapi.AuthBootstrapPayload{}, err
	}
	menus := make([]moduleapi.AuthBootstrapMenuItem, 0, len(payload.Menus))
	for _, item := range payload.Menus {
		menus = append(menus, moduleapi.AuthBootstrapMenuItem{Code: item.Code, ParentCode: item.ParentCode, Kind: item.Kind, Title: item.Title, TitleKey: item.TitleKey, Path: item.Path, Icon: item.Icon, Order: item.Order, Permission: item.Permission})
	}
	return moduleapi.AuthBootstrapPayload{
		User: moduleapi.CurrentUser{
			ID:          payload.User.ID,
			Username:    payload.User.Username,
			DisplayName: payload.User.DisplayName,
		},
		Roles:       payload.Roles,
		Permissions: payload.Permissions,
		Menus:       menus,
		Locale: moduleapi.AuthBootstrapLocaleSnapshot{
			CurrentLocale:    payload.Locale.CurrentLocale,
			DefaultLocale:    payload.Locale.DefaultLocale,
			FallbackLocale:   payload.Locale.FallbackLocale,
			SupportedLocales: payload.Locale.SupportedLocales,
		},
	}, nil
}

var _ moduleapi.UserBootstrapProvider = bootstrapReader{}

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

func (r bootstrapReader) filterBootstrapMenus(ctx context.Context, granted map[string]struct{}) []bootstrapMenuResponse {
	return filterBootstrapMenus(ctx, r.menuRegistry, granted, r.systemConfig)
}

// filterBootstrapMenus 根据授予的权限和系统配置可见性过滤菜单项，去重并排序，同时移除没有可见子项的菜单组。
// registry 为空时返回空切片；调用方在进入该过滤阶段前验证导航图。
func filterBootstrapMenus(
	ctx context.Context,
	registry *menu.Registry,
	granted map[string]struct{},
	systemConfig moduleapi.SystemConfigResolver,
) []bootstrapMenuResponse {
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
		if !bootstrapMenuFeatureGateVisible(ctx, item, systemConfig) {
			continue
		}

		response := bootstrapMenuResponse{
			Code:       item.Code,
			ParentCode: item.ParentCode,
			Kind:       string(item.Kind),
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
	menus = pruneEmptyBootstrapGroups(menus)

	slices.SortStableFunc(menus, compareBootstrapMenus)

	return menus
}

// pruneEmptyBootstrapGroups removes group menu items that have no visible children.
func pruneEmptyBootstrapGroups(menus []bootstrapMenuResponse) []bootstrapMenuResponse {
	visible := make(map[string]bootstrapMenuResponse, len(menus))
	for _, item := range menus {
		visible[item.Code] = item
	}
	for changed := true; changed; {
		changed = false
		children := make(map[string]bool, len(visible))
		for _, item := range visible {
			if item.ParentCode != "" {
				children[item.ParentCode] = true
			}
		}
		for code, item := range visible {
			if item.Kind == string(menu.NodeKindGroup) && !children[code] {
				delete(visible, code)
				changed = true
			}
		}
	}
	pruned := make([]bootstrapMenuResponse, 0, len(visible))
	for _, item := range menus {
		if _, ok := visible[item.Code]; ok {
			pruned = append(pruned, item)
		}
	}
	return pruned
}

// resolveBootstrapSystemConfig resolves the system configuration service from resolver.
// It returns nil when the resolver is unavailable, resolution fails, or the resolved value has an incompatible type.
func resolveBootstrapSystemConfig(resolver servicecontainer.Resolver) moduleapi.SystemConfigResolver {
	if resolver == nil {
		return nil
	}
	resolved, err := resolver.Resolve((*moduleapi.SystemConfigResolver)(nil))
	if err != nil {
		return nil
	}
	systemConfig, ok := resolved.(moduleapi.SystemConfigResolver)
	if !ok {
		return nil
	}
	return systemConfig
}

func bootstrapMenuFeatureGateVisible(
	ctx context.Context,
	item menu.Item,
	systemConfig moduleapi.SystemConfigResolver,
) bool {
	key := strings.TrimSpace(item.VisibleWhenConfigEnabled)
	if key == "" {
		return true
	}
	if systemConfig == nil {
		return true
	}
	return systemConfig.IsBooleanConfigEnabled(ctx, key, true)
}

func bootstrapMenuIdentity(item bootstrapMenuResponse) string {
	code := strings.TrimSpace(item.Code)
	if code != "" {
		return "code:" + code
	}

	return "path:" + strings.TrimSpace(item.Path)
}

// mergeBootstrapMenu 合并具有相同标识的菜单项，补充缺失字段并保留较小的排序值。
func mergeBootstrapMenu(existing, next bootstrapMenuResponse) bootstrapMenuResponse {
	merged := existing
	if merged.Title == "" {
		merged.Title = next.Title
	}
	if merged.ParentCode == "" {
		merged.ParentCode = next.ParentCode
	}
	if merged.Kind == "" {
		merged.Kind = next.Kind
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

	if parentCode := strings.Compare(strings.TrimSpace(left.ParentCode), strings.TrimSpace(right.ParentCode)); parentCode != 0 {
		return parentCode
	}

	return strings.Compare(left.Code, right.Code)
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
