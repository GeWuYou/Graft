package config

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed schema/v1.yaml
var currentSchemaYAML []byte

// Source 标识最终配置值的生效来源。
type Source string

const (
	// SourceCLI 表示值来自显式 CLI 覆盖。
	SourceCLI Source = "cli"
	// SourceEnvironment 表示值来自进程环境。
	SourceEnvironment Source = "environment"
	// SourceEnvFile 表示值来自只读 .env 文件。
	SourceEnvFile Source = "env_file"
	// SourceDefault 表示值来自配置 Schema 默认值。
	SourceDefault Source = "default"
)

// Severity 描述配置 finding 是否阻断启动。
type Severity string

const (
	// SeverityError 表示 finding 阻断后续启动。
	SeverityError Severity = "error"
	// SeverityWarning 表示 finding 仅提示迁移风险。
	SeverityWarning Severity = "warning"
)

// EnvironmentRule 描述一个版本化环境配置契约。
type EnvironmentRule struct {
	Name        string   `yaml:"name" json:"name"`
	Type        string   `yaml:"type" json:"type"`
	Required    bool     `yaml:"required" json:"required"`
	Default     *string  `yaml:"default" json:"default,omitempty"`
	Description string   `yaml:"description" json:"description"`
	Introduced  string   `yaml:"introduced" json:"introduced"`
	Deprecated  bool     `yaml:"deprecated" json:"deprecated"`
	Removed     bool     `yaml:"removed" json:"removed"`
	Replacement *string  `yaml:"replacement" json:"replacement,omitempty"`
	Severity    Severity `yaml:"severity" json:"severity,omitempty"`
	Sensitive   bool     `yaml:"sensitive" json:"sensitive"`

	replacementDeclared bool
}

// UnmarshalYAML 保留 replacement 是否显式声明，使 null 与缺失可被配置 Schema 校验区分。
func (r *EnvironmentRule) UnmarshalYAML(value *yaml.Node) error {
	type rawEnvironmentRule EnvironmentRule
	var decoded rawEnvironmentRule
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*r = EnvironmentRule(decoded)
	for index := 0; index+1 < len(value.Content); index += 2 {
		if value.Content[index].Value == "replacement" {
			r.replacementDeclared = true
			break
		}
	}
	return nil
}

// SchemaChange 描述当前 Schema 相对前一版本的操作员可读变更记录。
type SchemaChange struct {
	Kind        string `yaml:"kind" json:"kind"`
	Key         string `yaml:"key" json:"key"`
	Description string `yaml:"description" json:"description"`
	Migration   string `yaml:"migration" json:"migration"`
}

// AnyOfRule 描述至少一个非空配置必须存在的联合约束。
type AnyOfRule struct {
	Keys        []string `yaml:"keys" json:"keys"`
	Description string   `yaml:"description" json:"description"`
}

// Schema 是嵌入式版本化配置契约。
type Schema struct {
	Version     int               `yaml:"schema_version" json:"schema_version"`
	Environment []EnvironmentRule `yaml:"environment" json:"environment"`
	AnyOf       []AnyOfRule       `yaml:"any_of" json:"any_of"`
	Compose     ComposeRule       `yaml:"compose" json:"compose"`
	Changes     []SchemaChange    `yaml:"changes" json:"changes"`
}

// ComposeRule 描述官方 Compose 部署所需的服务拓扑与字段契约。
type ComposeRule struct {
	Services []ComposeServiceRule `yaml:"services" json:"services"`
}

// ComposeServiceRule 描述一个 Compose 服务的必需声明。
type ComposeServiceRule struct {
	Name               string                  `yaml:"name" json:"name"`
	Required           bool                    `yaml:"required" json:"required"`
	Environment        []string                `yaml:"environment" json:"environment"`
	Volumes            []ComposeVolumeRule     `yaml:"volumes" json:"volumes"`
	Ports              []string                `yaml:"ports" json:"ports"`
	Labels             []string                `yaml:"labels" json:"labels"`
	Secrets            []string                `yaml:"secrets" json:"secrets"`
	Restart            string                  `yaml:"restart" json:"restart"`
	User               string                  `yaml:"user" json:"user"`
	CommandContains    []string                `yaml:"command_contains" json:"command_contains"`
	EntrypointContains []string                `yaml:"entrypoint_contains" json:"entrypoint_contains"`
	DependsOn          []ComposeDependencyRule `yaml:"depends_on" json:"depends_on"`
}

// ComposeVolumeRule 描述一个按容器挂载目标匹配的卷要求。
type ComposeVolumeRule struct {
	Target   string `yaml:"target" json:"target"`
	ReadOnly *bool  `yaml:"read_only" json:"read_only,omitempty"`
}

// ComposeDependencyRule 描述一个服务启动依赖及其可选条件。
type ComposeDependencyRule struct {
	Service   string `yaml:"service" json:"service"`
	Condition string `yaml:"condition" json:"condition"`
}

// CurrentSchema 返回当前二进制嵌入的配置 Schema。
func CurrentSchema() (*Schema, error) {
	var schema Schema
	if err := yaml.Unmarshal(currentSchemaYAML, &schema); err != nil {
		return nil, fmt.Errorf("decode embedded configuration schema: %w", err)
	}
	if err := schema.Validate(); err != nil {
		return nil, err
	}
	return &schema, nil
}

// Validate 校验 Schema 自身，使二进制不会使用含糊或重复的配置契约。
func (s *Schema) Validate() error {
	if s == nil || s.Version < 1 {
		return fmt.Errorf("configuration schema_version must be greater than zero")
	}
	if err := validateEnvironmentRules(s.Environment); err != nil {
		return err
	}
	return validateComposeRules(s.Compose.Services)
}

func validateEnvironmentRules(rules []EnvironmentRule) error {
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		if strings.TrimSpace(rule.Name) == "" || strings.TrimSpace(rule.Type) == "" {
			return fmt.Errorf("configuration schema environment rule requires name and type")
		}
		if _, exists := seen[rule.Name]; exists {
			return fmt.Errorf("configuration schema contains duplicate key %s", rule.Name)
		}
		seen[rule.Name] = struct{}{}
		if rule.Removed && rule.Required {
			return fmt.Errorf("removed configuration %s cannot be required", rule.Name)
		}
		if err := validateMigrationRule(rule); err != nil {
			return err
		}
	}
	return nil
}

func validateMigrationRule(rule EnvironmentRule) error {
	if !rule.Deprecated && !rule.Removed {
		return nil
	}
	if !rule.replacementDeclared {
		return fmt.Errorf("deprecated or removed configuration %s requires replacement declaration", rule.Name)
	}
	if rule.Severity != SeverityError && rule.Severity != SeverityWarning {
		return fmt.Errorf("deprecated or removed configuration %s requires severity error or warning", rule.Name)
	}
	return nil
}

func validateComposeRules(rules []ComposeServiceRule) error {
	composeServices := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		if strings.TrimSpace(rule.Name) == "" {
			return fmt.Errorf("configuration schema compose service rule requires name")
		}
		if _, exists := composeServices[rule.Name]; exists {
			return fmt.Errorf("configuration schema contains duplicate compose service %s", rule.Name)
		}
		composeServices[rule.Name] = struct{}{}
		if err := validateComposeServiceRule(rule); err != nil {
			return err
		}
	}
	return nil
}

func validateComposeServiceRule(rule ComposeServiceRule) error {
	for _, volume := range rule.Volumes {
		if strings.TrimSpace(volume.Target) == "" {
			return fmt.Errorf("configuration schema compose service %s volume rule requires target", rule.Name)
		}
	}
	for _, dependency := range rule.DependsOn {
		if strings.TrimSpace(dependency.Service) == "" {
			return fmt.Errorf("configuration schema compose service %s dependency rule requires service", rule.Name)
		}
	}
	return nil
}

func sortedRules(schema *Schema) []EnvironmentRule {
	rules := append([]EnvironmentRule(nil), schema.Environment...)
	sort.Slice(rules, func(i, j int) bool { return rules[i].Name < rules[j].Name })
	return rules
}
