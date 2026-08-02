package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// ResolveOptions 控制配置治理解析所读取的来源。
type ResolveOptions struct {
	EnvFile     string
	Set         []string
	Environment map[string]string
}

type resolvedInputs struct {
	cli, environment, envFile map[string]string
}

// ResolvedValue 保存最终值及其来源；Value 永远不应写入诊断输出。
type ResolvedValue struct {
	Value  string `json:"-"`
	Source Source `json:"source"`
}

// Finding 是可机器读取且不包含敏感值的配置问题。
type Finding struct {
	Code        string   `json:"code"`
	Severity    Severity `json:"severity"`
	Key         string   `json:"key"`
	Source      Source   `json:"source,omitempty"`
	Description string   `json:"description"`
	Introduced  string   `json:"introduced,omitempty"`
	Replacement string   `json:"replacement,omitempty"`
}

// Report 是一次配置解析与校验的结构化结果。
type Report struct {
	SchemaVersion int                      `json:"schema_version"`
	Values        map[string]ResolvedValue `json:"values,omitempty"`
	Findings      []Finding                `json:"findings"`
}

// ErrorCount 返回阻断启动的 finding 数量。
func (r Report) ErrorCount() int {
	count := 0
	for _, finding := range r.Findings {
		if finding.Severity == SeverityError {
			count++
		}
	}
	return count
}

// ValidationError 让 CLI 和运行时能识别配置治理失败而不泄露配置值。
type ValidationError struct{ Report Report }

const configurationValidationExitCode = 2

// Error 返回稳定的简短失败摘要。
func (e *ValidationError) Error() string {
	return fmt.Sprintf("configuration validation failed: %d error(s)", e.Report.ErrorCount())
}

// ExitCode 让调用方区分可修复的契约错误与命令或 I/O 故障。
func (*ValidationError) ExitCode() int { return configurationValidationExitCode }

// ResolveAndValidate 根据 CLI、进程环境、.env 和 Schema 默认值的优先级解析并校验配置。
func ResolveAndValidate(options ResolveOptions) (Report, error) {
	schema, err := CurrentSchema()
	if err != nil {
		return Report{}, err
	}
	inputs, err := resolveInputs(options)
	if err != nil {
		return Report{}, err
	}
	report := Report{SchemaVersion: schema.Version, Values: make(map[string]ResolvedValue), Findings: []Finding{}}

	for _, rule := range sortedRules(schema) {
		validateRule(&report, rule, inputs)
	}
	validateAnyOfRules(&report, schema.AnyOf)
	validateSchemaVersion(&report)
	if report.ErrorCount() > 0 {
		return report, &ValidationError{Report: report}
	}
	return report, nil
}

func resolveInputs(options ResolveOptions) (resolvedInputs, error) {
	envFile, err := readGovernedEnvFile(options.EnvFile, options.Environment == nil)
	if err != nil {
		return resolvedInputs{}, err
	}
	cli, err := parseSetValues(options.Set)
	if err != nil {
		return resolvedInputs{}, err
	}
	environment := options.Environment
	if environment == nil {
		environment = environmentMap()
	}
	return resolvedInputs{cli: cli, environment: environment, envFile: envFile}, nil
}

func validateRule(report *Report, rule EnvironmentRule, inputs resolvedInputs) {
	value, source, present := resolveValue(rule, inputs.cli, inputs.environment, inputs.envFile)
	if present {
		report.Values[rule.Name] = ResolvedValue{Value: value, Source: source}
	}
	if rule.Removed && present {
		report.Findings = append(report.Findings, finding("removed", removalSeverity(rule), rule, source))
		return
	}
	report.Findings = append(report.Findings, validateRuleFindings(rule, value, source, present)...)
}

func removalSeverity(rule EnvironmentRule) Severity {
	if rule.Severity != "" {
		return rule.Severity
	}
	return SeverityError
}

func validateRuleFindings(rule EnvironmentRule, value string, source Source, present bool) []Finding {
	var findings []Finding
	if rule.Deprecated && present {
		findings = append(findings, finding("deprecated", SeverityWarning, rule, source))
	}
	if rule.Required && strings.TrimSpace(value) == "" {
		return append(findings, finding("required", SeverityError, rule, source))
	}
	if present && strings.TrimSpace(value) != "" && !validType(rule.Type, value) {
		findings = append(findings, finding("invalid_type", SeverityError, rule, source))
	}
	return findings
}

func validateAnyOfRules(report *Report, rules []AnyOfRule) {
	for _, rule := range rules {
		if anyOfSatisfied(rule, report.Values) {
			continue
		}
		key := strings.Join(rule.Keys, " or ")
		report.Findings = append(report.Findings, Finding{Code: "required_any_of", Severity: SeverityError, Key: key, Description: rule.Description})
	}
}

func finding(code string, severity Severity, rule EnvironmentRule, source Source) Finding {
	return Finding{Code: code, Severity: severity, Key: rule.Name, Source: source, Description: rule.Description, Introduced: rule.Introduced, Replacement: rule.Replacement}
}

func resolveValue(rule EnvironmentRule, cli, environment, envFile map[string]string) (string, Source, bool) {
	for _, candidate := range []struct {
		values map[string]string
		source Source
	}{{cli, SourceCLI}, {environment, SourceEnvironment}, {envFile, SourceEnvFile}} {
		if value, ok := candidate.values[rule.Name]; ok {
			return value, candidate.source, true
		}
	}
	if rule.Default != nil {
		return *rule.Default, SourceDefault, true
	}
	return "", "", false
}

func anyOfSatisfied(rule AnyOfRule, values map[string]ResolvedValue) bool {
	for _, key := range rule.Keys {
		if value, ok := values[key]; ok && strings.TrimSpace(value.Value) != "" {
			return true
		}
	}
	return false
}

func validateSchemaVersion(report *Report) {
	value, ok := report.Values["GRAFT_CONFIG_SCHEMA_VERSION"]
	if !ok {
		return
	}
	version, err := strconv.Atoi(strings.TrimSpace(value.Value))
	if err != nil || version != report.SchemaVersion {
		report.Findings = append(report.Findings, Finding{Code: "schema_version", Severity: SeverityError, Key: "GRAFT_CONFIG_SCHEMA_VERSION", Source: value.Source, Description: fmt.Sprintf("Configuration schema version must be %d.", report.SchemaVersion)})
	}
}

func validType(kind string, value string) bool {
	switch kind {
	case "string":
		return true
	case "integer":
		_, err := strconv.ParseInt(value, 10, 64)
		return err == nil
	case "boolean":
		_, err := strconv.ParseBool(value)
		return err == nil
	case "duration":
		_, err := time.ParseDuration(value)
		return err == nil
	default:
		return false
	}
}

func readGovernedEnvFile(path string, discover bool) (map[string]string, error) {
	resolvedPath := strings.TrimSpace(path)
	if resolvedPath == "" && discover {
		var err error
		resolvedPath, err = ResolveEnvFile("")
		if err != nil {
			return nil, err
		}
	}
	if resolvedPath == "" {
		return map[string]string{}, nil
	}
	values, err := godotenv.Read(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("read env file %s: %w", resolvedPath, err)
	}
	return values, nil
}

func parseSetValues(values []string) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for _, value := range values {
		key, content, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid --set value %q: expected KEY=VALUE", value)
		}
		result[strings.TrimSpace(key)] = content
	}
	return result, nil
}

func environmentMap() map[string]string {
	result := map[string]string{}
	for _, entry := range osEnviron() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			result[key] = value
		}
	}
	return result
}

var osEnviron = os.Environ

// IsValidationError 判断错误是否为可由操作者修复的配置契约失败。
func IsValidationError(err error) bool {
	var validationError *ValidationError
	return errors.As(err, &validationError)
}
