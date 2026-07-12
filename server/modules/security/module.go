package security

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	securitycontract "graft/server/modules/security/contract"
)

const securityMenuOrderOverview = 1

// Module owns the Security-domain aggregate read surface.
type Module struct{}

// NewModule creates the Security overview module.
func NewModule() *Module { return &Module{} }

// Register declares the Security overview permission, menu, and HTTP route.
func (m *Module) Register(ctx *module.Context) error {
	if ctx == nil || ctx.Services == nil {
		return errors.New("security module context services are unavailable")
	}
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	registerPermissions(ctx.PermissionRegistry)
	registerMenu(ctx.MenuRegistry)

	authService, err := module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return fmt.Errorf("resolve route authorizer: %w", err)
	}
	rbacPosture, err := module.ResolveService[moduleapi.RBACSecurityPostureService](ctx.Services, (*moduleapi.RBACSecurityPostureService)(nil))
	if err != nil {
		return fmt.Errorf("resolve rbac security posture service: %w", err)
	}
	auditReader, err := module.ResolveService[moduleapi.AuditSecurityReader](ctx.Services, (*moduleapi.AuditSecurityReader)(nil))
	if err != nil {
		return fmt.Errorf("resolve audit security reader: %w", err)
	}
	if ctx.Router == nil {
		return nil
	}

	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	group := ctx.Router.Group(securitycontract.SecurityGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(securitycontract.OverviewCollection,
		httpx.RequirePermission(ctx.I18n, authService, authorizer, securitycontract.OverviewReadPermission.String(), publisher),
		handleOverview(ctx, rbacPosture, auditReader),
	)
	return nil
}

// Boot has no owned runtime work.
func (m *Module) Boot(_ *module.Context) error { return nil }

// Shutdown has no owned runtime resources.
func (m *Module) Shutdown(_ *module.Context) error { return nil }

// registerPermissions 将 Security Overview 的读取权限注册到权限注册表中。
func registerPermissions(registry *permission.Registry) {
	if registry == nil {
		return
	}
	registry.Register(permission.Item{
		Code:           securitycontract.OverviewReadPermission.String(),
		DisplayKey:     securitycontract.OverviewReadDisplay.String(),
		DescriptionKey: securitycontract.OverviewReadDescription.String(),
		Module:         moduleID,
	})
}

// registerMenu registers the Security Overview menu entry when a menu registry is available.
func registerMenu(registry *menu.Registry) {
	if registry == nil {
		return
	}
	registry.Register(menu.Item{
		Code:       "security.overview",
		ParentCode: "domain.security",
		Kind:       menu.NodeKindEntry,
		Title:      "Security Overview",
		TitleKey:   securitycontract.OverviewMenuTitle.String(),
		Path:       securitycontract.OverviewMenuPath,
		Icon:       "security-overview",
		Order:      securityMenuOrderOverview,
		Permission: securitycontract.OverviewReadPermission.String(),
		Module:     moduleID,
	})
}

// registerMessages 验证安全模块所需的中英文国际化消息资源是否已注册。
// 如果本地化服务不可用或任一语言缺少所需消息资源，则返回错误。
func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}
	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range []securitycontract.MessageKey{
			securitycontract.OverviewMenuTitle,
			securitycontract.OverviewReadDisplay,
			securitycontract.OverviewReadDescription,
		} {
			if len(localizer.RegisteredMessageResources(locale, i18n.MessageKey(key.String()))) == 0 {
				return fmt.Errorf("register security module messages: locale resource %s missing key %s", locale, key)
			}
		}
	}
	return nil
}

type overviewResponse struct {
	TimePreset    moduleapi.AuditOverviewPreset   `json:"time_preset"`
	AccessControl moduleapi.SecurityPosture       `json:"access_control"`
	Audit         moduleapi.AuditSecuritySnapshot `json:"audit"`
}

// handleOverview 创建用于返回安全概览数据的 Gin 处理函数。
// @param ctx 模块上下文，用于本地化错误响应。
// @param rbacPosture 读取访问控制安全态势的服务。
// @param auditReader 读取指定时间范围安全审计快照的服务。
// @return 处理安全概览请求并返回统一响应的 Gin 处理函数。
func handleOverview(
	ctx *module.Context,
	rbacPosture moduleapi.RBACSecurityPostureService,
	auditReader moduleapi.AuditSecurityReader,
) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		preset, ok := parseOverviewPreset(ginCtx.Query("preset"))
		if !ok {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "preset"})
			return
		}
		requestCtx := ginCtx.Request.Context()
		posture, err := rbacPosture.ReadSecurityPosture(requestCtx)
		if err != nil {
			ctx.Logger.Error("read security posture failed", zap.Error(err))
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}
		auditSnapshot, err := auditReader.ReadSecuritySnapshot(requestCtx, preset)
		if err != nil {
			ctx.Logger.Error("read security audit snapshot failed", zap.Error(err))
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}
		httpx.WriteSuccess(ginCtx, http.StatusOK, overviewResponse{TimePreset: preset, AccessControl: posture, Audit: auditSnapshot})
	}
}

// parseOverviewPreset 将审计概览时间参数解析为受支持的时间范围。
// 返回规范化的时间范围及其是否有效。
func parseOverviewPreset(value string) (moduleapi.AuditOverviewPreset, bool) {
	switch moduleapi.AuditOverviewPreset(strings.TrimSpace(value)) {
	case "", moduleapi.AuditOverviewPresetLast24Hours:
		return moduleapi.AuditOverviewPresetLast24Hours, true
	case moduleapi.AuditOverviewPresetLast7Days:
		return moduleapi.AuditOverviewPresetLast7Days, true
	case moduleapi.AuditOverviewPresetLast30Days:
		return moduleapi.AuditOverviewPresetLast30Days, true
	default:
		return "", false
	}
}

var _ module.Module = (*Module)(nil)
