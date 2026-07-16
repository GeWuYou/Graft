package project

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	projectcompose "graft/server/modules/project/compose"
)

// ensureComposeProjectName 保留有效的 Compose 顶层 name；缺少时使用应用名称补充，并返回处理后的 YAML。
func ensureComposeProjectName(content, applicationName string) (string, string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return "", "", invalidComposeError("invalid compose yaml")
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return "", "", invalidComposeError("compose root must be a mapping")
	}
	mapping := root.Content[0]
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value != "name" {
			continue
		}
		name := strings.TrimSpace(mapping.Content[index+1].Value)
		if !projectcompose.IsValidComposeProjectName(name) {
			return "", "", invalidComposeError("invalid compose name")
		}
		return name, content, nil
	}
	name := strings.TrimSpace(applicationName)
	if !projectcompose.IsValidComposeProjectName(name) {
		return "", "", invalidComposeError("invalid application name")
	}
	mapping.Content = append([]*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: name}}, mapping.Content...)
	encoded, err := yaml.Marshal(&root)
	if err != nil {
		return "", "", invalidComposeError("encode compose yaml")
	}
	return name, string(encoded), nil
}

func invalidComposeError(reason string) error {
	return fmt.Errorf("%w: %s", errors.Join(errProjectInvalidArgument, errProjectInvalidCompose), reason)
}
