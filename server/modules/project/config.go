package project

import (
	"encoding/json"
	"errors"
	"fmt"

	"graft/server/internal/configregistry"
	projectcontract "graft/server/modules/project/contract"
)

const (
	projectConfigDomain          = "ops"
	projectConfigDomainLabel     = "Operations"
	projectConfigGroupCreate     = "ops.project.create"
	projectConfigGroupCreateDesc = "Managed project create authority and root workflow."
	projectConfigOrderBase       = 7100
	maxManagedRootLength         = 1024
)

const defaultManagedRootDirectory = ""

// registerConfig 注册本模块定义的配置项，并按基础顺序为每项设置排序。
// 当 registry 为空时返回错误；任一配置注册失败时返回包装后的错误。
func registerConfig(registry *configregistry.Registry) error {
	if registry == nil {
		return errors.New("config registry is unavailable")
	}
	for index, definition := range configDefinitions() {
		definition.Order = projectConfigOrderBase + index
		if err := registry.Register(definition); err != nil {
			return fmt.Errorf("register project config definition %s: %w", definition.Key, err)
		}
	}
	return nil
}

// configDefinitions 返回本模块管理的配置定义列表。
func configDefinitions() []configregistry.Definition {
	return []configregistry.Definition{
		projectManagedRootDefinition(),
	}
}

// projectManagedRootDefinition 构建项目托管根目录配置的定义。
// 该配置用于项目创建流程中的托管根目录设置，包含其所属域、分组、文案键、字符串类型、默认值、运行时热更新应用模式以及对应权限。
func projectManagedRootDefinition() configregistry.Definition {
	return configregistry.Definition{
		Key:              projectcontract.ProjectManagedRootConfig.String(),
		Module:           moduleID,
		Domain:           projectConfigDomain,
		DomainLabel:      projectConfigDomainLabel,
		Group:            projectConfigGroupCreate,
		GroupLabel:       "Project Create",
		GroupDescription: projectConfigGroupCreateDesc,
		TitleKey:         projectcontract.ProjectManagedRootConfigTitle.String(),
		DescriptionKey:   projectcontract.ProjectManagedRootConfigDescription.String(),
		Type:             configregistry.ValueTypeString,
		Schema:           json.RawMessage(projectManagedRootSchema()),
		DefaultValue:     mustRawJSON(defaultManagedRootDirectory),
		RuntimeApplyMode: configregistry.RuntimeApplyModeRuntimeHot,
		Permission:       projectcontract.ProjectCreatePermission.String(),
		RestartRequired:  false,
		Required:         false,
		Sensitive:        false,
	}
}

// projectManagedRootSchema 返回项目托管根目录配置的 JSON Schema 字符串。
// 该 Schema 约束值为字符串，包含最大长度限制，并描述其用于项目创建流程的托管根目录。
func projectManagedRootSchema() string {
	return fmt.Sprintf(
		`{"type":"string","minLength":0,"maxLength":%d,"description":"Absolute managed root directory for project create flows. Empty means managed create stays unavailable until explicitly configured."}`,
		maxManagedRootLength,
	)
}

// mustRawJSON 将 value 编码为 JSON，并返回对应的 json.RawMessage。
// 编码失败时，函数会 panic。
func mustRawJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
