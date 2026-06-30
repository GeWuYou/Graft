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

func configDefinitions() []configregistry.Definition {
	return []configregistry.Definition{
		projectManagedRootDefinition(),
	}
}

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
		Schema:           mustRawJSON(projectManagedRootSchema()),
		DefaultValue:     mustRawJSON(defaultManagedRootDirectory),
		RuntimeApplyMode: configregistry.RuntimeApplyModeRuntimeHot,
		Permission:       projectcontract.ProjectCreatePermission.String(),
		RestartRequired:  false,
		Required:         false,
		Sensitive:        false,
	}
}

func projectManagedRootSchema() string {
	return fmt.Sprintf(
		`{"type":"string","minLength":0,"maxLength":%d,"description":"Absolute managed root directory for project create flows. Empty means managed create stays unavailable until explicitly configured."}`,
		maxManagedRootLength,
	)
}

func mustRawJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
