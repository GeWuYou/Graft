package project

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	projectcompose "graft/server/modules/project/compose"
)

// ensureComposeProjectName preserves a declared Compose top-level name or injects
// ensureComposeProjectName 确保 Compose YAML 包含有效的顶层 name，并在缺少时使用应用名写入。
// 返回项目名称、处理后的 YAML 内容以及处理错误。
func ensureComposeProjectName(content, applicationName string) (string, string, error) {
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
	name := strings.TrimSpace(applicationName)
	if !projectcompose.IsValidCanonicalProjectName(name) {
		return "", "", fmt.Errorf("%w: invalid application name", errProjectInvalidArgument)
	}
	mapping.Content = append([]*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"}, {Kind: yaml.ScalarNode, Tag: "!!str", Value: name}}, mapping.Content...)
	encoded, err := yaml.Marshal(&root)
	if err != nil {
		return "", "", fmt.Errorf("%w: encode compose yaml", errProjectInvalidArgument)
	}
	return name, string(encoded), nil
}
