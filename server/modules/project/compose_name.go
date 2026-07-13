package project

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	projectcompose "graft/server/modules/project/compose"
)

// ensureComposeProjectName preserves a declared Compose top-level name or injects
// a deterministic name derived from the display name before a managed workspace is written.
func ensureComposeProjectName(content, displayName string) (string, string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return "", "", fmt.Errorf("%w: invalid compose yaml", errProjectInvalidArgument)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return "", "", fmt.Errorf("%w: compose root must be a mapping", errProjectInvalidArgument)
	}
	mapping := root.Content[0]
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value != "name" {
			continue
		}
		name := strings.TrimSpace(mapping.Content[index+1].Value)
		if !projectcompose.IsValidCanonicalProjectName(name) {
			return "", "", fmt.Errorf("%w: invalid compose name", errProjectInvalidArgument)
		}
		return name, content, nil
	}
	name, err := deriveComposeProjectName(displayName)
	if err != nil {
		return "", "", err
	}
	mapping.Content = append([]*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: name}}, mapping.Content...)
	encoded, err := yaml.Marshal(&root)
	if err != nil {
		return "", "", fmt.Errorf("%w: encode compose yaml", errProjectInvalidArgument)
	}
	return name, string(encoded), nil
}

func deriveComposeProjectName(displayName string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(displayName))
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, value)
	value = strings.Trim(value, "-")
	if !projectcompose.IsValidCanonicalProjectName(value) {
		return "", fmt.Errorf("%w: unable to derive compose name", errProjectInvalidArgument)
	}
	return value, nil
}
