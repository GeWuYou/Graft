package auth

import (
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
	"graft/server/internal/permission"
	rbaclocales "graft/server/modules/rbac/locales"
)

// TestPermissionSeedsFromItemsResolvesDockerImagePermissionLocales 验证新增 Docker 镜像权限的 server locale 能进入默认管理员种子。
func TestPermissionSeedsFromItemsResolvesDockerImagePermissionLocales(t *testing.T) {
	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "en-US",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	resources, err := rbaclocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load rbac locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register rbac locale resources: %v", err)
	}

	items := []permission.Item{
		{
			Code:           "ops.container.image.pull",
			DisplayKey:     "rbac.permissionCatalog.dockerImagePull.display",
			DescriptionKey: "rbac.permissionCatalog.dockerImagePull.description",
			Module:         "container",
		},
		{
			Code:           "ops.container.image.tag",
			DisplayKey:     "rbac.permissionCatalog.dockerImageTag.display",
			DescriptionKey: "rbac.permissionCatalog.dockerImageTag.description",
			Module:         "container",
		},
		{
			Code:           "ops.container.image.remove",
			DisplayKey:     "rbac.permissionCatalog.dockerImageRemove.display",
			DescriptionKey: "rbac.permissionCatalog.dockerImageRemove.description",
			Module:         "container",
		},
	}

	seeds, err := permissionSeedsFromItems(localizer, items)
	if err != nil {
		t.Fatalf("build Docker image permission seeds: %v", err)
	}
	if len(seeds) != len(items) {
		t.Fatalf("expected %d permission seeds, got %d", len(items), len(seeds))
	}

	wantDisplays := []string{"拉取镜像", "为镜像打标签", "删除镜像"}
	wantDescriptions := []string{
		"允许从 Docker 守护进程已配置的仓库拉取镜像。",
		"允许为本地 Docker 镜像创建新的仓库标签。",
		"允许删除本地 Docker 镜像，强制删除需额外确认。",
	}
	for index, seed := range seeds {
		if seed.Display != wantDisplays[index] {
			t.Errorf("seed %d display = %q, want %q", index, seed.Display, wantDisplays[index])
		}
		if seed.Description != wantDescriptions[index] {
			t.Errorf("seed %d description = %q, want %q", index, seed.Description, wantDescriptions[index])
		}
	}
}
