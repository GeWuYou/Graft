package container

import (
	"errors"
	"fmt"

	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	containercontract "graft/server/modules/container/contract"
)

const (
	operationsMenuOrderRoot = 50
	containerMenuOrderList  = 51
	dockerImageMenuOrder    = 52
	dockerVolumeMenuOrder   = 53
)

// registerMessages verifies that all required container module messages are registered for the supported locales.
// It returns an error when the localizer is unavailable or any required locale resource is missing.
func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range containerLocaleBackedMessageKeys() {
			matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
			if len(matches) == 0 {
				return fmt.Errorf("register container module messages: locale resource %s missing key %s", locale, key)
			}
		}
	}
	return nil
}

// containerLocaleBackedMessageKeys returns the message keys backed by localized resources.
func containerLocaleBackedMessageKeys() []string {
	return append([]string(nil), containerMessageKeys...)
}

var containerMessageKeys = []string{
	containercontract.ContainerMenuTitle.String(),
	containercontract.ContainerListMenuTitle.String(),
	containercontract.DockerImageMenuTitle.String(),
	containercontract.DockerVolumeMenuTitle.String(),
	containercontract.DockerVolumeNotFound.String(),
	containercontract.DockerVolumeConflict.String(),
	containercontract.ContainerRuntimeDisabled.String(),
	containercontract.ContainerRuntimeSocketMissing.String(),
	containercontract.ContainerRuntimePermissionDenied.String(),
	containercontract.ContainerRuntimeUnavailable.String(),
	containercontract.ContainerNotFound.String(),
	containercontract.ContainerMountNotFound.String(),
	containercontract.ContainerInvalidRef.String(),
	containercontract.ContainerInvalidListQuery.String(),
	containercontract.ContainerInvalidBatchAction.String(),
	containercontract.ContainerInvalidState.String(),
	containercontract.ContainerEventsUnavailable.String(),
	containercontract.ContainerLogsTooLarge.String(),
	containercontract.ContainerInvalidLogQuery.String(),
	containercontract.ContainerShellDisabled.String(),
	containercontract.ContainerShellForbidden.String(),
	containercontract.ContainerShellTicketInvalid.String(),
	containercontract.ContainerShellTicketExpired.String(),
	containercontract.ContainerShellTicketUsed.String(),
	containercontract.ContainerShellOriginDenied.String(),
	containercontract.ContainerShellContainerNotRunning.String(),
	containercontract.ContainerShellCommandNotFound.String(),
	containercontract.ContainerShellSessionFailed.String(),
	containercontract.ContainerShellUnsupportedControlMessage.String(),
	containercontract.ContainerTimeout.String(),
	containercontract.ContainerMountUsageUnsupported.String(),
	containercontract.ContainerDangerousActionsDisabled.String(),
	containercontract.DockerImageInvalidReference.String(),
	containercontract.DockerImageInUse.String(),
	containercontract.DockerImageReferencedByMultipleTags.String(),
	containercontract.DockerImageNotFound.String(),
	containercontract.DockerImageCommunicationError.String(),
	containercontract.DockerImagePullFailed.String(),
	containercontract.DockerImageTagFailed.String(),
	containercontract.DockerImageRemoveFailed.String(),
	containercontract.DockerImageTagNotAssociated.String(),
	containercontract.DockerImagePullCompleted.String(),
	containercontract.DockerImageTagCompleted.String(),
	containercontract.DockerImageUntagCompleted.String(),
	containercontract.DockerImageRemoveCompleted.String(),
	containercontract.ContainerAuditShellSessionRequested.String(),
	containercontract.ContainerAuditShellTicketIssued.String(),
	containercontract.ContainerAuditShellTicketRejected.String(),
	containercontract.ContainerAuditShellSessionStarted.String(),
	containercontract.ContainerAuditShellSessionClosed.String(),
	containercontract.ContainerAuditShellSessionFailed.String(),
	containercontract.ContainerActionStartCompleted.String(),
	containercontract.ContainerActionStopCompleted.String(),
	containercontract.ContainerActionRestartCompleted.String(),
	containercontract.ContainerActionRemoveCompleted.String(),
	containercontract.ContainerBatchActionCompleted.String(),
	containercontract.ContainerBatchActionPartial.String(),
	containercontract.ContainerBatchActionFailed.String(),
}

// registerPermissions 注册容器模块的权限项。
// 当权限注册表不可用时返回错误。
func registerPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}

	for _, item := range permissionItems(moduleName) {
		registry.Register(item)
	}
	return nil
}

// permissionItems 为容器与镜像管理操作构建 RBAC 权限项。
// moduleName 用于设置每个权限项的模块标识。
func permissionItems(moduleName string) []permission.Item {
	items := []permission.Item{
		{
			Code:           containercontract.ContainerViewPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerView.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerView.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerDetailPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerDetail.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerDetail.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerEventsPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerEvents.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerEvents.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerEnvironmentPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerEnvironment.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerEnvironment.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerLogsPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerLogs.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerLogs.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerShellPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerShell.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerShell.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerStartPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerStart.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerStart.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerStopPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerStop.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerStop.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerRestartPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerRestart.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerRestart.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerRemovePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.containerRemove.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.containerRemove.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.ContainerVolumeRemovePermission.String(),
			DisplayKey:     "rbac.permissionCatalog.containerVolumeRemove.display",
			DescriptionKey: "rbac.permissionCatalog.containerVolumeRemove.description",
			Module:         moduleName,
		},
	}
	return append(items, dockerImagePermissionItems(moduleName)...)
}

// dockerImagePermissionItems 返回与 Image ID 和 Repository:Tag 动作对应的最小权限集合。
func dockerImagePermissionItems(moduleName string) []permission.Item {
	return []permission.Item{
		{
			Code:           containercontract.DockerImagePullPermission.String(),
			DisplayKey:     "rbac.permissionCatalog.dockerImagePull.display",
			DescriptionKey: "rbac.permissionCatalog.dockerImagePull.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.DockerImageTagPermission.String(),
			DisplayKey:     "rbac.permissionCatalog.dockerImageTag.display",
			DescriptionKey: "rbac.permissionCatalog.dockerImageTag.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.DockerImageUntagPermission.String(),
			DisplayKey:     "rbac.permissionCatalog.dockerImageUntag.display",
			DescriptionKey: "rbac.permissionCatalog.dockerImageUntag.description",
			Module:         moduleName,
		},
		{
			Code:           containercontract.DockerImageRemovePermission.String(),
			DisplayKey:     "rbac.permissionCatalog.dockerImageRemove.display",
			DescriptionKey: "rbac.permissionCatalog.dockerImageRemove.description",
			Module:         moduleName,
		},
	}
}

// registerMenu registers the container list menu item with the specified module name.
// registerMenu 注册容器列表菜单项。
// @param moduleName 菜单项所属的模块名称。
// registerMenu 注册容器列表菜单项。
// registerMenu 注册容器运行时菜单组及其容器列表入口。
// 当菜单注册器不可用时返回错误，注册成功时返回 nil。
func registerMenu(registry *menu.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}

	registry.Register(menu.Item{
		Code:                     "docker",
		ParentCode:               "domain.infrastructure",
		Kind:                     menu.NodeKindGroup,
		Title:                    "",
		TitleKey:                 containercontract.ContainerMenuTitle.String(),
		SectionKey:               menu.RuntimeSectionKey,
		SectionTitleKey:          containercontract.ContainerMenuSectionTitle.String(),
		Icon:                     "docker-provider",
		Order:                    operationsMenuOrderRoot,
		VisibleWhenConfigEnabled: containercontract.ContainerRuntimeEnabledConfig.String(),
		Module:                   moduleName,
	})
	registry.Register(menu.Item{
		Code:                     "container.list",
		ParentCode:               "docker",
		Kind:                     menu.NodeKindEntry,
		Title:                    "",
		TitleKey:                 containercontract.ContainerListMenuTitle.String(),
		Path:                     containercontract.ContainerMenuPath,
		Icon:                     "container-workload",
		Order:                    containerMenuOrderList,
		Permission:               containercontract.ContainerViewPermission.String(),
		VisibleWhenConfigEnabled: containercontract.ContainerRuntimeEnabledConfig.String(),
		Module:                   moduleName,
	})
	registry.Register(menu.Item{
		Code:                     "docker.image.list",
		ParentCode:               "docker",
		Kind:                     menu.NodeKindEntry,
		TitleKey:                 containercontract.DockerImageMenuTitle.String(),
		Path:                     containercontract.DockerImageMenuPath,
		Icon:                     "image-artifact",
		Order:                    dockerImageMenuOrder,
		Permission:               containercontract.ContainerViewPermission.String(),
		VisibleWhenConfigEnabled: containercontract.ContainerRuntimeEnabledConfig.String(),
		Module:                   moduleName,
	})
	registry.Register(menu.Item{
		Code:                     "docker.volume.list",
		ParentCode:               "docker",
		Kind:                     menu.NodeKindEntry,
		TitleKey:                 containercontract.DockerVolumeMenuTitle.String(),
		Path:                     containercontract.DockerVolumeMenuPath,
		Icon:                     "persistent-volume",
		Order:                    dockerVolumeMenuOrder,
		Permission:               containercontract.ContainerViewPermission.String(),
		VisibleWhenConfigEnabled: containercontract.ContainerRuntimeEnabledConfig.String(),
		Module:                   moduleName,
	})
	return nil
}
