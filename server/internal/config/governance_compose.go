package config

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidateComposeFile 校验 Compose 服务拓扑及 Schema 声明的部署字段契约。
func ValidateComposeFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	contents, err := os.ReadFile(path) // #nosec G304 -- 路径仅来自显式 CLI 参数或受控 Compose 挂载。
	if err != nil {
		return fmt.Errorf("read compose file %s: %w", path, err)
	}
	var document composeDocument
	if err := yaml.Unmarshal(contents, &document); err != nil {
		return fmt.Errorf("parse compose file %s: %w", path, err)
	}
	if len(document.Services) == 0 {
		return fmt.Errorf("compose file %s must declare a non-empty services mapping", path)
	}
	schema, err := CurrentSchema()
	if err != nil {
		return err
	}
	findings := validateComposeDocument(document, schema.Compose)
	if len(findings) == 0 {
		return nil
	}
	sort.Strings(findings)
	validationErrors := make([]error, 0, len(findings))
	for _, finding := range findings {
		validationErrors = append(validationErrors, errors.New(finding))
	}
	return fmt.Errorf("compose configuration validation failed: %w", errors.Join(validationErrors...))
}

type composeDocument struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Environment yaml.Node `yaml:"environment"`
	Volumes     yaml.Node `yaml:"volumes"`
	Ports       yaml.Node `yaml:"ports"`
	Labels      yaml.Node `yaml:"labels"`
	Secrets     yaml.Node `yaml:"secrets"`
	Restart     string    `yaml:"restart"`
	User        string    `yaml:"user"`
	Command     yaml.Node `yaml:"command"`
	Entrypoint  yaml.Node `yaml:"entrypoint"`
	DependsOn   yaml.Node `yaml:"depends_on"`
}

func validateComposeDocument(document composeDocument, contract ComposeRule) []string {
	var findings []string
	for _, rule := range contract.Services {
		service, exists := document.Services[rule.Name]
		if !exists {
			if rule.Required {
				findings = append(findings, fmt.Sprintf("missing required compose service %s", rule.Name))
			}
			continue
		}
		findings = append(findings, validateComposeService(service, rule)...)
	}
	return findings
}

func validateComposeService(service composeService, rule ComposeServiceRule) []string {
	findings := validateComposeServiceFields(service, rule)
	for _, target := range rule.Volumes {
		volume, found := findVolume(service.Volumes, target.Target)
		if !found {
			findings = append(findings, fmt.Sprintf("compose service %s missing required volume target %s", rule.Name, target.Target))
			continue
		}
		if target.ReadOnly != nil && volume.readOnly != *target.ReadOnly {
			findings = append(findings, fmt.Sprintf("compose service %s volume target %s must set read_only=%t", rule.Name, target.Target, *target.ReadOnly))
		}
	}
	for _, dependency := range rule.DependsOn {
		if !hasDependency(service.DependsOn, dependency) {
			findings = append(findings, fmt.Sprintf("compose service %s missing required dependency %s", rule.Name, dependency.Service))
		}
	}
	return findings
}

func validateComposeServiceFields(service composeService, rule ComposeServiceRule) []string {
	findings := validateEnvironmentFields(service, rule)
	findings = append(findings, validatePortFields(service, rule)...)
	findings = append(findings, validateLabelSecretFields(service, rule)...)
	findings = append(findings, validateRuntimeFields(service, rule)...)
	return findings
}

func validateEnvironmentFields(service composeService, rule ComposeServiceRule) []string {
	var findings []string
	for _, key := range rule.Environment {
		if !nodeHasKey(service.Environment, key) {
			findings = append(findings, fmt.Sprintf("compose service %s missing required environment %s", rule.Name, key))
		}
	}
	return findings
}

func validatePortFields(service composeService, rule ComposeServiceRule) []string {
	var findings []string
	for _, port := range rule.Ports {
		if !hasPort(service.Ports, port) {
			findings = append(findings, fmt.Sprintf("compose service %s missing required container port %s", rule.Name, port))
		}
	}
	return findings
}

func validateLabelSecretFields(service composeService, rule ComposeServiceRule) []string {
	var findings []string
	for _, label := range rule.Labels {
		if !nodeHasKey(service.Labels, label) {
			findings = append(findings, fmt.Sprintf("compose service %s missing required label %s", rule.Name, label))
		}
	}
	for _, secret := range rule.Secrets {
		if !hasSecret(service.Secrets, secret) {
			findings = append(findings, fmt.Sprintf("compose service %s missing required secret %s", rule.Name, secret))
		}
	}
	return findings
}

func validateRuntimeFields(service composeService, rule ComposeServiceRule) []string {
	var findings []string
	if rule.Restart != "" && service.Restart != rule.Restart {
		findings = append(findings, fmt.Sprintf("compose service %s must set restart to %s", rule.Name, rule.Restart))
	}
	if rule.User != "" && service.User != rule.User {
		findings = append(findings, fmt.Sprintf("compose service %s must set user to %s", rule.Name, rule.User))
	}
	if !nodeContains(service.Command, rule.CommandContains) {
		findings = append(findings, fmt.Sprintf("compose service %s command does not satisfy required runtime arguments", rule.Name))
	}
	if !nodeContains(service.Entrypoint, rule.EntrypointContains) {
		findings = append(findings, fmt.Sprintf("compose service %s entrypoint does not satisfy required runtime arguments", rule.Name))
	}
	return findings
}

func nodeHasKey(node yaml.Node, key string) bool {
	node = unwrapNode(node)
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			if node.Content[index].Value == key {
				return true
			}
		}
		return false
	}
	return nodeHasValue(node, key) || nodeHasValue(node, key+"=")
}

func nodeHasValue(node yaml.Node, expected string) bool {
	for _, value := range nodeValues(node) {
		if value == expected || strings.HasPrefix(value, expected+"=") {
			return true
		}
	}
	return false
}

func nodeValues(node yaml.Node) []string {
	node = unwrapNode(node)
	switch node.Kind {
	case yaml.ScalarNode:
		return []string{node.Value}
	case yaml.SequenceNode:
		values := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			if child.Kind == yaml.MappingNode {
				for index := 0; index+1 < len(child.Content); index += 2 {
					values = append(values, child.Content[index].Value)
				}
				continue
			}
			values = append(values, child.Value)
		}
		return values
	case yaml.MappingNode:
		values := make([]string, 0, len(node.Content)/keyValueNodeWidth)
		for index := 0; index+1 < len(node.Content); index += 2 {
			values = append(values, node.Content[index].Value)
		}
		return values
	default:
		return nil
	}
}

type composeVolume struct{ readOnly bool }

func findVolume(node yaml.Node, target string) (composeVolume, bool) {
	node = unwrapNode(node)
	if node.Kind != yaml.SequenceNode {
		return composeVolume{}, false
	}
	for _, volume := range node.Content {
		if found, ok := findVolumeEntry(*volume, target); ok {
			return found, true
		}
	}
	return composeVolume{}, false
}

func findVolumeEntry(volume yaml.Node, target string) (composeVolume, bool) {
	if volume.Kind == yaml.MappingNode && mappingValue(volume, "target") == target {
		return composeVolume{readOnly: strings.EqualFold(mappingValue(volume, "read_only"), "true")}, true
	}
	if volume.Kind != yaml.ScalarNode {
		return composeVolume{}, false
	}
	parts := splitComposeVolume(volume.Value)
	if len(parts) < volumePartsMinimum {
		return composeVolume{}, false
	}
	if matchesVolumeTarget(parts[len(parts)-2], target) {
		return composeVolume{readOnly: parts[len(parts)-1] == "ro"}, true
	}
	if matchesVolumeTarget(parts[len(parts)-1], target) {
		return composeVolume{}, true
	}
	return composeVolume{}, false
}

const volumePartsMinimum = 2

const composeVolumePartsCapacity = 3

func splitComposeVolume(value string) []string {
	parts := make([]string, 0, composeVolumePartsCapacity)
	start, interpolationDepth := 0, 0
	for index := 0; index < len(value); index++ {
		switch value[index] {
		case '$':
			if index+1 < len(value) && value[index+1] == '{' {
				interpolationDepth++
				index++
			}
		case '}':
			if interpolationDepth > 0 {
				interpolationDepth--
			}
		case ':':
			if interpolationDepth == 0 {
				parts = append(parts, value[start:index])
				start = index + 1
			}
		}
	}
	return append(parts, value[start:])
}

// matchesVolumeTarget 仅接受完整目标或 Compose 默认插值，避免相邻路径名称发生子串误匹配。
func matchesVolumeTarget(value, target string) bool {
	if value == target {
		return true
	}
	if !strings.HasPrefix(value, "${") || !strings.HasSuffix(value, "}") {
		return false
	}
	defaultOffset := strings.LastIndex(value, ":-")
	return defaultOffset > 1 && value[defaultOffset+2:len(value)-1] == target
}

func hasPort(node yaml.Node, target string) bool {
	node = unwrapNode(node)
	for _, port := range nodeValues(node) {
		port = strings.TrimSuffix(port, "/tcp")
		if port == target || strings.HasSuffix(port, ":"+target) {
			return true
		}
	}
	if node.Kind == yaml.SequenceNode {
		for _, port := range node.Content {
			if port.Kind == yaml.MappingNode && mappingValue(*port, "target") == target {
				return true
			}
		}
	}
	return false
}

func hasSecret(node yaml.Node, expected string) bool {
	node = unwrapNode(node)
	if node.Kind == yaml.MappingNode {
		return nodeHasKey(node, expected)
	}
	if node.Kind != yaml.SequenceNode {
		return node.Value == expected
	}
	for _, secret := range node.Content {
		if secret.Kind == yaml.ScalarNode && secret.Value == expected {
			return true
		}
		if secret.Kind == yaml.MappingNode && mappingValue(*secret, "source") == expected {
			return true
		}
	}
	return false
}

func nodeContains(node yaml.Node, required []string) bool {
	if len(required) == 0 {
		return true
	}
	contents := strings.Join(nodeValues(node), "\n")
	for _, item := range required {
		if !strings.Contains(contents, item) {
			return false
		}
	}
	return true
}

func hasDependency(node yaml.Node, rule ComposeDependencyRule) bool {
	node = unwrapNode(node)
	if node.Kind != yaml.MappingNode {
		return nodeHasValue(node, rule.Service) && rule.Condition == ""
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value != rule.Service {
			continue
		}
		if rule.Condition == "" || mappingValue(*node.Content[index+1], "condition") == rule.Condition {
			return true
		}
	}
	return false
}

func mappingValue(node yaml.Node, key string) string {
	node = unwrapNode(node)
	if node.Kind != yaml.MappingNode {
		return ""
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1].Value
		}
	}
	return ""
}

const keyValueNodeWidth = 2

func unwrapNode(node yaml.Node) yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		return *node.Content[0]
	}
	return node
}
