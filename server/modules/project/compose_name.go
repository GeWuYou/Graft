package project

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	projectcompose "graft/server/modules/project/compose"
)

// ensureComposeProjectName preserves a declared Compose top-level name or injects
// ensureComposeProjectName 确保 Compose YAML 包含有效的顶层 name，并在缺少时根据显示名称生成并写入。
// 返回项目名称、处理后的 YAML 内容以及处理错误。
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

// deriveComposeProjectName 根据显示名称生成规范的 Compose 项目名称；非字母数字字符会替换为连字符。
// 成功时返回生成的名称；名称无效时返回错误。
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
