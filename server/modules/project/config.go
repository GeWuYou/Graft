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
	projectConfigGroupCreate     = "ops.application.create"
	projectConfigGroupImport     = "ops.application.import"
	projectConfigGroupWorkspace  = "ops.application.workspace"
	projectConfigOrderBase       = 7100
	minWorkspaceJSONLength       = 2
	maxManagedRootLength         = 1024
	maxImportRootsLength         = 8192
	maxWorkspaceHiddenDirsLength = 4096
	maxWorkspaceTooltipRulesJSON = 16384
)

const defaultApplicationRootDirectory = "/opt/graft/apps"
const defaultImportAllowedRoots = "[]"
const defaultWorkspaceHiddenDirectories = `[".git",".github","node_modules","vendor","target","build","dist","coverage","data","logs","tmp","cache",".idea",".vscode"]`
const defaultWorkspaceFileTooltipRules = `[{"pattern":"^docker-compose(?:\\.[^.]+)?\\.ya?ml$","tooltip":"Compose 配置","enabled":true},{"pattern":"^\\.env(?:\\..+)?$","tooltip":"环境变量文件","enabled":true}]`
const defaultWorkspaceDirectoryTooltipRules = `[{"pattern":"^logs$","tooltip":"日志目录","enabled":true}]`
const projectConfigDomainKey = "systemConfig.domains.ops"

type projectConfigDefinitionSpec struct {
	key                 string
	group               string
	groupKey            string
	groupDescriptionKey string
	titleKey            string
	descriptionKey      string
	schema              string
	defaultValue        any
	permission          string
}

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
		applicationRootDirectoryDefinition(),
		projectImportAllowedRootsDefinition(),
		projectWorkspaceHiddenDirectoriesDefinition(),
		projectWorkspaceFileTooltipRulesDefinition(),
		projectWorkspaceDirectoryTooltipRulesDefinition(),
	}
}

func applicationRootDirectoryDefinition() configregistry.Definition {
	return buildProjectConfigDefinition(projectConfigDefinitionSpec{
		key:                 projectcontract.ApplicationRootDirectoryConfig.String(),
		group:               projectConfigGroupCreate,
		groupKey:            projectcontract.ApplicationCreateConfigGroupTitle.String(),
		groupDescriptionKey: projectcontract.ApplicationCreateConfigGroupDescription.String(),
		titleKey:            projectcontract.ApplicationRootDirectoryConfigTitle.String(),
		descriptionKey:      projectcontract.ApplicationRootDirectoryConfigDescription.String(),
		schema:              applicationRootDirectorySchema(),
		defaultValue:        defaultApplicationRootDirectory,
		permission:          projectcontract.ApplicationCreatePermission.String(),
	})
}

func projectImportAllowedRootsDefinition() configregistry.Definition {
	return buildProjectConfigDefinition(projectConfigDefinitionSpec{
		key:                 projectcontract.ApplicationImportAllowedRootsConfig.String(),
		group:               projectConfigGroupImport,
		groupKey:            projectcontract.ApplicationImportConfigGroupTitle.String(),
		groupDescriptionKey: projectcontract.ApplicationImportConfigGroupDescription.String(),
		titleKey:            projectcontract.ApplicationImportAllowedRootsConfigTitle.String(),
		descriptionKey:      projectcontract.ApplicationImportAllowedRootsConfigDescription.String(),
		schema:              projectImportAllowedRootsSchema(),
		defaultValue:        defaultImportAllowedRoots,
		permission:          projectcontract.ApplicationImportPermission.String(),
	})
}

func projectWorkspaceHiddenDirectoriesDefinition() configregistry.Definition {
	return buildProjectConfigDefinition(projectConfigDefinitionSpec{
		key:                 projectcontract.ApplicationWorkspaceHiddenDirectoriesConfig.String(),
		group:               projectConfigGroupWorkspace,
		groupKey:            projectcontract.ApplicationWorkspaceConfigGroupTitle.String(),
		groupDescriptionKey: projectcontract.ApplicationWorkspaceConfigGroupDescription.String(),
		titleKey:            projectcontract.ApplicationWorkspaceHiddenDirectoriesConfigTitle.String(),
		descriptionKey:      projectcontract.ApplicationWorkspaceHiddenDirectoriesConfigDescription.String(),
		schema:              projectWorkspaceHiddenDirectoriesSchema(),
		defaultValue:        defaultWorkspaceHiddenDirectories,
		permission:          projectcontract.ApplicationViewPermission.String(),
	})
}

func projectWorkspaceFileTooltipRulesDefinition() configregistry.Definition {
	return buildProjectConfigDefinition(projectConfigDefinitionSpec{
		key:                 projectcontract.ApplicationWorkspaceFileTooltipRulesConfig.String(),
		group:               projectConfigGroupWorkspace,
		groupKey:            projectcontract.ApplicationWorkspaceConfigGroupTitle.String(),
		groupDescriptionKey: projectcontract.ApplicationWorkspaceConfigGroupDescription.String(),
		titleKey:            projectcontract.ApplicationWorkspaceFileTooltipRulesConfigTitle.String(),
		descriptionKey:      projectcontract.ApplicationWorkspaceFileTooltipRulesConfigDescription.String(),
		schema:              projectWorkspaceTooltipRulesSchema(projectcontract.ApplicationWorkspaceFileTooltipRulesConfigDescription.String()),
		defaultValue:        defaultWorkspaceFileTooltipRules,
		permission:          projectcontract.ApplicationViewPermission.String(),
	})
}

func projectWorkspaceDirectoryTooltipRulesDefinition() configregistry.Definition {
	return buildProjectConfigDefinition(projectConfigDefinitionSpec{
		key:                 projectcontract.ApplicationWorkspaceDirectoryTooltipRulesConfig.String(),
		group:               projectConfigGroupWorkspace,
		groupKey:            projectcontract.ApplicationWorkspaceConfigGroupTitle.String(),
		groupDescriptionKey: projectcontract.ApplicationWorkspaceConfigGroupDescription.String(),
		titleKey:            projectcontract.ApplicationWorkspaceDirectoryTooltipRulesConfigTitle.String(),
		descriptionKey:      projectcontract.ApplicationWorkspaceDirectoryTooltipRulesConfigDescription.String(),
		schema:              projectWorkspaceTooltipRulesSchema(projectcontract.ApplicationWorkspaceDirectoryTooltipRulesConfigDescription.String()),
		defaultValue:        defaultWorkspaceDirectoryTooltipRules,
		permission:          projectcontract.ApplicationViewPermission.String(),
	})
}

func buildProjectConfigDefinition(spec projectConfigDefinitionSpec) configregistry.Definition {
	return configregistry.Definition{
		Key:                 spec.key,
		Module:              moduleID,
		Domain:              projectConfigDomain,
		DomainKey:           projectConfigDomainKey,
		DomainLabel:         "",
		Group:               spec.group,
		GroupKey:            spec.groupKey,
		GroupLabel:          "",
		GroupDescription:    "",
		GroupDescriptionKey: spec.groupDescriptionKey,
		TitleKey:            spec.titleKey,
		DescriptionKey:      spec.descriptionKey,
		Type:                configregistry.ValueTypeString,
		Schema:              json.RawMessage(spec.schema),
		DefaultValue:        mustRawJSON(spec.defaultValue),
		RuntimeApplyMode:    configregistry.RuntimeApplyModeRuntimeHot,
		Permission:          spec.permission,
		RestartRequired:     false,
		Required:            false,
		Sensitive:           false,
	}
}

// applicationRootDirectorySchema 返回应用根目录配置的 JSON Schema；空字符串表示禁用应用创建。
func applicationRootDirectorySchema() string {
	return fmt.Sprintf(
		`{"type":"string","minLength":0,"maxLength":%d,"x-i18n":{"titleKey":"%s","descriptionKey":"%s"}}`,
		maxManagedRootLength,
		projectcontract.ApplicationRootDirectoryConfigTitle.String(),
		projectcontract.ApplicationRootDirectoryConfigDescription.String(),
	)
}

// 该 Schema 定义为字符串类型，表示一个 JSON 数组字符串，数组项应包含稳定的 id、显示标签和绝对本地路径。
func projectImportAllowedRootsSchema() string {
	return fmt.Sprintf(
		`{"type":"string","minLength":2,"maxLength":%d,"description":"JSON array string for import browse roots. Each item should include stable id, operator label, and absolute local path.","examples":["[{\"id\":\"srv\",\"label\":\"/srv\",\"path\":\"/srv\"}]"]}`,
		maxImportRootsLength,
	)
}

func projectWorkspaceHiddenDirectoriesSchema() string {
	return fmt.Sprintf(
		`{"type":"string","minLength":2,"maxLength":%d,"description":"JSON array string for configuration workspace directories hidden by default. Each item is a directory basename such as node_modules or .git.","examples":["[\".git\",\"node_modules\",\"dist\"]"],"x-i18n":{"descriptionKey":"%s","placeholderKey":"%s"},"x-graft":{"editor":"string-array-json-list"}}`,
		maxWorkspaceHiddenDirsLength,
		projectcontract.ApplicationWorkspaceHiddenDirectoriesConfigDescription.String(),
		projectcontract.ApplicationWorkspaceHiddenDirectoriesPlaceholder.String(),
	)
}

func projectWorkspaceTooltipRulesSchema(descriptionKey string) string {
	exampleRules := []workspaceTooltipRuleSchemaExample{
		{
			Pattern: `^docker-compose(?:\.[^.]+)?\.ya?ml$`,
			Tooltip: "Compose 配置",
			Enabled: true,
		},
	}
	schema := map[string]any{
		"type":        "string",
		"minLength":   minWorkspaceJSONLength,
		"maxLength":   maxWorkspaceTooltipRulesJSON,
		"description": "JSON array string of ordered workspace tooltip rules. Each rule contains basename regex pattern, tooltip text, and enabled flag. Later enabled matches override earlier matches.",
		"examples":    []string{string(mustRawJSON(exampleRules))},
		"x-i18n": map[string]string{
			"descriptionKey": descriptionKey,
		},
		"x-graft": map[string]string{
			"editor": "workspace-tooltip-rule-list",
		},
	}
	return string(mustRawJSON(schema))
}

type workspaceTooltipRuleSchemaExample struct {
	Pattern string `json:"pattern"`
	Tooltip string `json:"tooltip"`
	Enabled bool   `json:"enabled"`
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
