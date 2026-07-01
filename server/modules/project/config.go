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
	projectConfigGroupImport     = "ops.project.import"
	projectConfigGroupImportDesc = "Import authority roots and bounded directory browsing."
	projectConfigOrderBase       = 7100
	maxManagedRootLength         = 1024
	maxImportRootsLength         = 8192
)

const defaultManagedRootDirectory = ""
const defaultImportAllowedRoots = "[]"

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
		projectImportAllowedRootsDefinition(),
	}
}

// projectManagedRootDefinition 构造项目创建流程中托管根目录配置的定义。
// projectManagedRootDefinition 构造并返回项目创建流程所使用的受管根目录配置定义。
// 该定义包含配置键、展示分组、JSON Schema、默认值以及写入该配置所需权限。
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

// projectImportAllowedRootsDefinition 构造项目导入浏览允许根目录配置的定义。
// projectImportAllowedRootsDefinition 构造项目导入允许根目录的配置定义。
// 该配置以字符串形式存储 JSON 数组，数组元素包含稳定的 id、label 和绝对本地路径。
func projectImportAllowedRootsDefinition() configregistry.Definition {
	return configregistry.Definition{
		Key:              projectcontract.ProjectImportAllowedRootsConfig.String(),
		Module:           moduleID,
		Domain:           projectConfigDomain,
		DomainLabel:      projectConfigDomainLabel,
		Group:            projectConfigGroupImport,
		GroupLabel:       "Project Import",
		GroupDescription: projectConfigGroupImportDesc,
		TitleKey:         projectcontract.ProjectImportAllowedRootsConfigTitle.String(),
		DescriptionKey:   projectcontract.ProjectImportAllowedRootsConfigDescription.String(),
		Type:             configregistry.ValueTypeString,
		Schema:           json.RawMessage(projectImportAllowedRootsSchema()),
		DefaultValue:     mustRawJSON(defaultImportAllowedRoots),
		RuntimeApplyMode: configregistry.RuntimeApplyModeRuntimeHot,
		Permission:       projectcontract.ProjectImportPermission.String(),
		RestartRequired:  false,
		Required:         false,
		Sensitive:        false,
	}
}

// projectManagedRootSchema 返回项目托管根目录配置的 JSON Schema 字符串。
// projectManagedRootSchema 返回用于项目创建流程托管根目录配置的 JSON Schema。
// 该 Schema 约束值为字符串，并包含最大长度限制；空值表示托管创建在显式配置前不可用。
func projectManagedRootSchema() string {
	return fmt.Sprintf(
		`{"type":"string","minLength":0,"maxLength":%d,"description":"Absolute managed root directory for project create flows. Empty means managed create stays unavailable until explicitly configured."}`,
		maxManagedRootLength,
	)
}

// 该 Schema 定义为字符串类型，表示一个 JSON 数组字符串，数组项应包含稳定的 id、显示标签和绝对本地路径。
func projectImportAllowedRootsSchema() string {
	return fmt.Sprintf(
		`{"type":"string","minLength":2,"maxLength":%d,"description":"JSON array string for import browse roots. Each item should include stable id, operator label, and absolute local path.","examples":["[{\"id\":\"srv\",\"label\":\"/srv\",\"path\":\"/srv\"}]"]}`,
		maxImportRootsLength,
	)
}

// mustRawJSON 将 value 编码为 JSON，并返回对应的 json.RawMessage。
// mustRawJSON 将值编码为 JSON 原始消息。
//
// 编码失败时会 panic。
func mustRawJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
