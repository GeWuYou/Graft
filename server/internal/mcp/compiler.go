package mcp

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	mcpExtensionName       = "x-graft-mcp"
	mcpExtensionSchemaName = "x-graft-mcp-schema"

	inputBodyName                         = "body"
	confirmationTokenInputName            = "confirmation_token"
	resourceTemplatePlaceholderMatchParts = 2
)

var (
	resourceTemplatePlaceholderPattern = regexp.MustCompile(`\{([^{}]+)\}`)
	isoDurationPattern                 = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)W)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)
)

const isoWeeksCaptureIndex = 3

// DocumentationCatalog 是由 canonical OpenAPI 的 MCP capability 投影生成的只读文档清单。
// 它与 runtime 共用编译结果，避免 Explorer 维护第二份 Tool、Resource 或 Action 定义。
type DocumentationCatalog struct {
	Tools     []DocumentationItem `json:"tools"`
	Resources []DocumentationItem `json:"resources"`
	Actions   []DocumentationItem `json:"actions"`
}

// DocumentationItem 描述可由 MCP runtime 注册的单个 capability。
// Resource 会填充 URI 模板；Tool 和 Action 则保留其对应的 REST operation 与输入 schema。
type DocumentationItem struct {
	Name         string                    `json:"name"`
	Description  string                    `json:"description"`
	Method       string                    `json:"method"`
	Path         string                    `json:"path"`
	InputSchema  map[string]any            `json:"input_schema"`
	URITemplate  string                    `json:"uri_template,omitempty"`
	Risk         DocumentationRisk         `json:"risk"`
	Confirmation DocumentationConfirmation `json:"confirmation"`
}

// DocumentationRisk 保留 operation 对 Agent 调用决策有意义的风险说明。
type DocumentationRisk struct {
	Level      string `json:"level"`
	Reason     string `json:"reason,omitempty"`
	Reversible *bool  `json:"reversible,omitempty"`
	Impact     string `json:"impact,omitempty"`
}

// DocumentationConfirmation 描述危险 Action 所需的二阶段确认策略。
type DocumentationConfirmation struct {
	Required bool   `json:"required"`
	Strategy string `json:"strategy"`
	TTL      string `json:"ttl"`
}

// toolDefinition 是从 canonical OpenAPI operation 编译出的只读 MCP Tool。
// 它保留 REST path、参数绑定和输入 JSON Schema，运行时不能再手写补充业务 Tool。
type toolDefinition struct {
	name        string
	description string
	method      string
	path        string
	inputSchema map[string]any
	inputs      []inputBinding
	metadata    mcpMetadata
}

// capabilityDefinitions 是从同一份 OpenAPI operation 投影出的 MCP 能力集合。
// Tool、Resource Template 与 Action 共用 operationId、参数和风险元数据，避免维护第二套清单。
type capabilityDefinitions struct {
	tools     []toolDefinition
	resources []resourceDefinition
}

type resourceDefinition struct {
	name        string
	description string
	uriTemplate string
	tool        toolDefinition
}

// Name 返回由 operationId 规范化得到的稳定 MCP Tool 名称。
func (d toolDefinition) Name() string {
	return d.name
}

type inputBinding struct {
	name     string
	location string
	required bool
	style    string
	explode  *bool
}

type mcpMetadata struct {
	resourceURITemplates         []string
	resourceURIParameterBindings map[string]string
	risk                         mcpRisk
	confirmation                 mcpConfirmation
}

type mcpRisk struct {
	level      string
	reason     string
	reversible *bool
	impact     string
}

type mcpConfirmation struct {
	required bool
	strategy string
	ttl      string
}

// CompileReadTools 从已打包的 canonical OpenAPI 文档生成 MCP read Tool。
// 它只接受显式声明 x-graft-mcp 的 GET operation，拒绝不完整 metadata，且不提供 tag、path 或 summary 回退。
func CompileReadTools(bundle []byte) ([]toolDefinition, error) {
	capabilities, err := CompileCapabilities(bundle)
	if err != nil {
		return nil, err
	}
	tools := make([]toolDefinition, 0, len(capabilities.tools))
	for _, tool := range capabilities.tools {
		if tool.method == http.MethodGet {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

// CompileCapabilities 从 canonical OpenAPI 生成所有已批准的 MCP capability 投影。
// capability 不能仅按 HTTP method 分类：GET 与显式声明为低风险、无确认的 POST 可作为查询 Tool；确认保护的 POST 才作为 Action，身份始终来自 operationId。
func CompileCapabilities(bundle []byte) (capabilityDefinitions, error) {
	document, err := loadOpenAPIDocument(bundle)
	if err != nil {
		return capabilityDefinitions{}, err
	}
	return compileDocumentCapabilities(document)
}

// CompileDocumentationCatalog 从 canonical OpenAPI 生成供 MCP Explorer 序列化的 capability 清单。
// 它不会扩展 runtime 行为，且不推断 operation 未声明的权限或 scope。
func CompileDocumentationCatalog(bundle []byte) (DocumentationCatalog, error) {
	capabilities, err := CompileCapabilities(bundle)
	if err != nil {
		return DocumentationCatalog{}, err
	}
	catalog := DocumentationCatalog{
		Tools:     make([]DocumentationItem, 0),
		Resources: make([]DocumentationItem, 0, len(capabilities.resources)),
		Actions:   make([]DocumentationItem, 0),
	}
	for _, tool := range capabilities.tools {
		item := documentationItemFromTool(tool)
		if tool.metadata.confirmation.required {
			catalog.Actions = append(catalog.Actions, item)
			continue
		}
		catalog.Tools = append(catalog.Tools, item)
	}
	for _, resource := range capabilities.resources {
		item := documentationItemFromTool(resource.tool)
		item.URITemplate = resource.uriTemplate
		catalog.Resources = append(catalog.Resources, item)
	}
	return catalog, nil
}

func documentationItemFromTool(tool toolDefinition) DocumentationItem {
	item := DocumentationItem{
		Name:        tool.name,
		Description: tool.description,
		Method:      tool.method,
		Path:        tool.path,
		InputSchema: tool.inputSchema,
		Risk: DocumentationRisk{
			Level:      tool.metadata.risk.level,
			Reason:     tool.metadata.risk.reason,
			Reversible: tool.metadata.risk.reversible,
			Impact:     tool.metadata.risk.impact,
		},
		Confirmation: DocumentationConfirmation{
			Required: tool.metadata.confirmation.required,
			Strategy: tool.metadata.confirmation.strategy,
			TTL:      tool.metadata.confirmation.ttl,
		},
	}
	if len(tool.metadata.resourceURITemplates) == 1 {
		item.URITemplate = tool.metadata.resourceURITemplates[0]
	}
	return item
}

func loadOpenAPIDocument(bundle []byte) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	document, err := loader.LoadFromData(bundle)
	if err != nil {
		return nil, fmt.Errorf("load bundled OpenAPI document: %w", err)
	}
	if err := document.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate bundled OpenAPI document: %w", err)
	}
	if err := validateMetadataAnchor(document); err != nil {
		return nil, err
	}
	return document, nil
}

func compileDocumentCapabilities(document *openapi3.T) (capabilityDefinitions, error) {
	paths := slices.Collect(maps.Keys(document.Paths.Map()))
	sort.Strings(paths)
	allNames := make(map[string]string)
	capabilities := capabilityDefinitions{tools: make([]toolDefinition, 0), resources: make([]resourceDefinition, 0)}
	for _, path := range paths {
		pathItem := document.Paths.Find(path)
		if pathItem == nil {
			return capabilityDefinitions{}, fmt.Errorf("OpenAPI path %q is unavailable", path)
		}
		methods := slices.Collect(maps.Keys(pathItem.Operations()))
		sort.Strings(methods)
		for _, method := range methods {
			definitions, err := compileOpenAPIOperation(pathItem, path, method, allNames)
			if err != nil {
				return capabilityDefinitions{}, err
			}
			if definitions.tool != nil {
				capabilities.tools = append(capabilities.tools, *definitions.tool)
			}
			if definitions.resource != nil {
				capabilities.resources = append(capabilities.resources, *definitions.resource)
			}
		}
	}
	slices.SortFunc(capabilities.tools, func(left, right toolDefinition) int {
		return strings.Compare(left.name, right.name)
	})
	slices.SortFunc(capabilities.resources, func(left, right resourceDefinition) int {
		if compared := strings.Compare(left.uriTemplate, right.uriTemplate); compared != 0 {
			return compared
		}
		return strings.Compare(left.name, right.name)
	})
	return capabilities, nil
}

type compiledOperation struct {
	tool     *toolDefinition
	resource *resourceDefinition
}

func compileOpenAPIOperation(pathItem *openapi3.PathItem, path, method string, allNames map[string]string) (compiledOperation, error) {
	operation := pathItem.GetOperation(method)
	if operation == nil || operation.Extensions == nil {
		return compiledOperation{}, nil
	}
	rawMetadata, optedIn := operation.Extensions[mcpExtensionName]
	if !optedIn {
		return compiledOperation{}, nil
	}
	metadata, parameters, err := compileOperationMetadata(pathItem, operation, method, path, rawMetadata)
	if err != nil {
		return compiledOperation{}, err
	}
	toolName, err := registerOperationToolName(allNames, operation.OperationID, method, path)
	if err != nil {
		return compiledOperation{}, err
	}
	return compileOperationProjection(toolName, operation, path, method, parameters, metadata)
}

func compileOperationProjection(toolName string, operation *openapi3.Operation, path, method string, parameters map[string]*openapi3.Parameter, metadata mcpMetadata) (compiledOperation, error) {
	switch strings.ToUpper(method) {
	case http.MethodGet:
		return compileReadProjection(toolName, operation, path, parameters, metadata)
	case http.MethodPost:
		return compileActionProjection(toolName, operation, path, parameters, metadata)
	default:
		return compiledOperation{}, fmt.Errorf("OpenAPI %s %s is not supported for MCP projection", strings.ToUpper(method), path)
	}
}

func compileReadProjection(toolName string, operation *openapi3.Operation, path string, parameters map[string]*openapi3.Parameter, metadata mcpMetadata) (compiledOperation, error) {
	if err := validateReadOnlyMetadata(metadata); err != nil {
		return compiledOperation{}, fmt.Errorf("OpenAPI GET %s is not safe to compile as a read tool: %w", path, err)
	}
	definition, err := compileTool(toolName, operation, path, http.MethodGet, parameters, metadata)
	if err != nil {
		return compiledOperation{}, fmt.Errorf("compile OpenAPI GET %s: %w", path, err)
	}
	if len(metadata.resourceURITemplates) == 0 {
		return compiledOperation{tool: &definition}, nil
	}
	resource, err := compileResource(definition, metadata)
	if err != nil {
		return compiledOperation{}, fmt.Errorf("compile OpenAPI GET %s resource: %w", path, err)
	}
	return compiledOperation{tool: &definition, resource: resource}, nil
}

func compileActionProjection(toolName string, operation *openapi3.Operation, path string, parameters map[string]*openapi3.Parameter, metadata mcpMetadata) (compiledOperation, error) {
	if !metadata.confirmation.required {
		if err := validateReadOnlyMetadata(metadata); err != nil {
			return compiledOperation{}, fmt.Errorf("OpenAPI POST %s is not safe to compile as a query tool: %w", path, err)
		}
		definition, err := compileTool(toolName, operation, path, http.MethodPost, parameters, metadata)
		if err != nil {
			return compiledOperation{}, fmt.Errorf("compile OpenAPI POST %s query tool: %w", path, err)
		}
		return compiledOperation{tool: &definition}, nil
	}
	definition, err := compileTool(toolName, operation, path, http.MethodPost, parameters, metadata)
	if err != nil {
		return compiledOperation{}, fmt.Errorf("compile OpenAPI POST %s: %w", path, err)
	}
	return compiledOperation{tool: &definition}, nil
}

func compileOperationMetadata(pathItem *openapi3.PathItem, operation *openapi3.Operation, method, path string, rawMetadata any) (mcpMetadata, map[string]*openapi3.Parameter, error) {
	metadata, err := parseMetadata(rawMetadata)
	if err != nil {
		return mcpMetadata{}, nil, fmt.Errorf("OpenAPI %s %s metadata: %w", strings.ToUpper(method), path, err)
	}
	parameters, err := collectParameters(pathItem, operation)
	if err != nil {
		return mcpMetadata{}, nil, fmt.Errorf("OpenAPI %s %s parameters: %w", strings.ToUpper(method), path, err)
	}
	if err := validateMetadata(metadata, parameters); err != nil {
		return mcpMetadata{}, nil, fmt.Errorf("OpenAPI %s %s metadata: %w", strings.ToUpper(method), path, err)
	}
	return metadata, parameters, nil
}

func registerOperationToolName(allNames map[string]string, operationID, method, path string) (string, error) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return "", fmt.Errorf("OpenAPI %s %s opted into MCP without operationId", strings.ToUpper(method), path)
	}
	toolName := snakeCaseOperationID(operationID)
	if toolName == "" {
		return "", fmt.Errorf("OpenAPI %s %s has invalid operationId %q", strings.ToUpper(method), path, operationID)
	}
	if existing, exists := allNames[toolName]; exists {
		return "", fmt.Errorf("MCP tool name %q collides between %s and %s %s", toolName, existing, strings.ToUpper(method), path)
	}
	allNames[toolName] = strings.ToUpper(method) + " " + path
	return toolName, nil
}

func validateMetadataAnchor(document *openapi3.T) error {
	if document == nil {
		return fmt.Errorf("bundled OpenAPI document is missing %s", mcpExtensionSchemaName)
	}
	anchor, err := extensionObject(document.Extensions, mcpExtensionSchemaName)
	if err != nil {
		return err
	}
	return validateMetadataAnchorShape(anchor)
}

func extensionObject(extensions map[string]any, name string) (map[string]any, error) {
	if extensions == nil {
		return nil, fmt.Errorf("bundled OpenAPI document is missing %s", name)
	}
	raw, ok := extensions[name]
	if !ok {
		return nil, fmt.Errorf("bundled OpenAPI document is missing %s", name)
	}
	anchor, ok := objectValue(raw)
	if !ok {
		return nil, fmt.Errorf("%s must define an object schema", name)
	}
	return anchor, nil
}

func validateMetadataAnchorShape(anchor map[string]any) error {
	if strings.TrimSpace(stringValue(anchor["type"])) != "object" {
		return fmt.Errorf("%s must define an object schema", mcpExtensionSchemaName)
	}
	properties, ok := objectValue(anchor["properties"])
	if !ok || properties["risk"] == nil || properties["confirmation"] == nil {
		return fmt.Errorf("%s must define risk and confirmation properties", mcpExtensionSchemaName)
	}
	if !containsString(anyStringSlice(anchor["required"]), "risk") || !containsString(anyStringSlice(anchor["required"]), "confirmation") {
		return fmt.Errorf("%s must require risk and confirmation", mcpExtensionSchemaName)
	}
	return nil
}

func parseMetadata(raw any) (mcpMetadata, error) {
	value, ok := objectValue(raw)
	if !ok {
		return mcpMetadata{}, fmt.Errorf("%s must be an object", mcpExtensionName)
	}
	resources, err := parseResourceMetadata(value)
	if err != nil {
		return mcpMetadata{}, err
	}
	risk, err := parseRiskMetadata(value["risk"])
	if err != nil {
		return mcpMetadata{}, err
	}
	confirmation, err := parseConfirmationMetadata(value["confirmation"])
	if err != nil {
		return mcpMetadata{}, err
	}
	return mcpMetadata{resourceURITemplates: resources.templates, resourceURIParameterBindings: resources.bindings, risk: risk, confirmation: confirmation}, nil
}

type resourceMetadata struct {
	templates []string
	bindings  map[string]string
}

func parseResourceMetadata(value map[string]any) (resourceMetadata, error) {
	resources := resourceMetadata{
		templates: anyStringSlice(value["resource_uri_templates"]),
		bindings:  stringMap(value["resource_uri_parameter_bindings"]),
	}
	if value["resource_uri_templates"] != nil && len(resources.templates) == 0 {
		return resourceMetadata{}, fmt.Errorf("resource_uri_templates must be a non-empty string array")
	}
	if value["resource_uri_parameter_bindings"] != nil && resources.bindings == nil {
		return resourceMetadata{}, fmt.Errorf("resource_uri_parameter_bindings must be an object")
	}
	return resources, nil
}

func parseRiskMetadata(raw any) (mcpRisk, error) {
	riskValue, ok := objectValue(raw)
	if !ok {
		return mcpRisk{}, fmt.Errorf("risk must be an object")
	}
	level, ok := riskValue["level"].(string)
	if !ok || strings.TrimSpace(level) == "" {
		return mcpRisk{}, fmt.Errorf("risk.level is required")
	}
	risk := mcpRisk{
		level:  strings.TrimSpace(level),
		reason: strings.TrimSpace(stringValue(riskValue["reason"])),
		impact: strings.TrimSpace(stringValue(riskValue["impact"])),
	}
	if reversible, exists := riskValue["reversible"]; exists {
		parsed, ok := reversible.(bool)
		if !ok {
			return mcpRisk{}, fmt.Errorf("risk.reversible must be a boolean")
		}
		risk.reversible = &parsed
	}
	if err := validateRiskTextFields(riskValue); err != nil {
		return mcpRisk{}, err
	}
	return risk, nil
}

func validateRiskTextFields(risk map[string]any) error {
	for _, field := range []string{"reason", "impact"} {
		rawText, exists := risk[field]
		if !exists {
			continue
		}
		text, ok := rawText.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return fmt.Errorf("risk.%s must be a non-empty string", field)
		}
	}
	return nil
}

func parseConfirmationMetadata(raw any) (mcpConfirmation, error) {
	confirmationValue, ok := objectValue(raw)
	if !ok {
		return mcpConfirmation{}, fmt.Errorf("confirmation must be an object")
	}
	required, ok := confirmationValue["required"].(bool)
	if !ok {
		return mcpConfirmation{}, fmt.Errorf("confirmation.required must be a boolean")
	}
	strategy, ok := confirmationValue["strategy"].(string)
	if !ok || strings.TrimSpace(strategy) == "" {
		return mcpConfirmation{}, fmt.Errorf("confirmation.strategy is required")
	}
	ttl, ok := confirmationValue["ttl"].(string)
	if !ok || strings.TrimSpace(ttl) == "" {
		return mcpConfirmation{}, fmt.Errorf("confirmation.ttl is required")
	}
	return mcpConfirmation{required: required, strategy: strings.TrimSpace(strategy), ttl: strings.TrimSpace(ttl)}, nil
}

func validateMetadata(metadata mcpMetadata, parameters map[string]*openapi3.Parameter) error {
	if !containsString([]string{"low", "medium", "high", "critical"}, metadata.risk.level) {
		return fmt.Errorf("risk.level %q is unsupported", metadata.risk.level)
	}
	if err := validateConfirmation(metadata.confirmation); err != nil {
		return err
	}
	return validateResourceURIs(metadata, parameters)
}

func validateConfirmation(confirmation mcpConfirmation) error {
	if confirmation.required {
		if confirmation.strategy != "two_phase" {
			return fmt.Errorf("confirmation.required=true requires strategy two_phase")
		}
		if !positiveISODuration(confirmation.ttl) {
			return fmt.Errorf("confirmation.required=true requires a positive ISO 8601 ttl")
		}
		return nil
	}
	if confirmation.strategy != "none" || confirmation.ttl != "PT0S" {
		return fmt.Errorf("confirmation.required=false requires strategy none and ttl PT0S")
	}
	return nil
}

func validateResourceURIs(metadata mcpMetadata, parameters map[string]*openapi3.Parameter) error {
	if len(metadata.resourceURITemplates) == 0 {
		return nil
	}
	if metadata.resourceURIParameterBindings == nil {
		return fmt.Errorf("resource_uri_parameter_bindings is required when resource_uri_templates are declared")
	}
	placeholders, err := collectResourceURIPlaceholders(metadata.resourceURITemplates)
	if err != nil {
		return err
	}
	return validateResourceURIParameterBindings(placeholders, metadata.resourceURIParameterBindings, parameters)
}

func collectResourceURIPlaceholders(templates []string) (map[string]struct{}, error) {
	placeholders := make(map[string]struct{})
	for _, template := range templates {
		template = strings.TrimSpace(template)
		if !isCanonicalResourceURI(template) {
			return nil, fmt.Errorf("resource URI template %q is not canonical", template)
		}
		matches := resourceTemplatePlaceholderPattern.FindAllStringSubmatch(template, -1)
		if strings.ContainsAny(resourceTemplatePlaceholderPattern.ReplaceAllString(template, ""), "{}") {
			return nil, fmt.Errorf("resource URI template %q has an invalid placeholder", template)
		}
		for _, match := range matches {
			placeholders[match[1]] = struct{}{}
		}
	}
	return placeholders, nil
}

func validateResourceURIParameterBindings(placeholders map[string]struct{}, bindings map[string]string, parameters map[string]*openapi3.Parameter) error {
	for placeholder := range placeholders {
		binding, exists := bindings[placeholder]
		if !exists || strings.TrimSpace(binding) == "" {
			return fmt.Errorf("resource URI placeholder %q has no parameter binding", placeholder)
		}
		parameter, exists := parameters["path:"+strings.TrimSpace(binding)]
		if !exists || parameter == nil {
			return fmt.Errorf("resource URI placeholder %q must bind to a declared path parameter %q", placeholder, binding)
		}
	}
	for placeholder := range bindings {
		if _, exists := placeholders[placeholder]; !exists {
			return fmt.Errorf("resource URI parameter binding %q has no matching placeholder", placeholder)
		}
	}
	return nil
}

func isCanonicalResourceURI(template string) bool {
	return strings.HasPrefix(template, "graft://docker/containers/") ||
		strings.HasPrefix(template, "graft://applications/") ||
		strings.HasPrefix(template, "graft://runtime-targets/")
}

func validateReadOnlyMetadata(metadata mcpMetadata) error {
	if metadata.risk.level != "low" {
		return fmt.Errorf("risk.level must be low, got %q", metadata.risk.level)
	}
	if metadata.risk.reversible == nil || !*metadata.risk.reversible {
		return fmt.Errorf("risk.reversible must be true")
	}
	if metadata.confirmation.required || metadata.confirmation.strategy != "none" || metadata.confirmation.ttl != "PT0S" {
		return fmt.Errorf("read tools must not require confirmation")
	}
	return nil
}

func collectParameters(pathItem *openapi3.PathItem, operation *openapi3.Operation) (map[string]*openapi3.Parameter, error) {
	parameters := make(map[string]*openapi3.Parameter)
	for _, refs := range []openapi3.Parameters{pathItem.Parameters, operation.Parameters} {
		for _, ref := range refs {
			if ref == nil || ref.Value == nil {
				return nil, fmt.Errorf("parameter reference is unresolved")
			}
			parameter := ref.Value
			name := strings.TrimSpace(parameter.Name)
			location := strings.TrimSpace(parameter.In)
			if name == "" || location == "" {
				return nil, fmt.Errorf("parameter name and location are required")
			}
			key := location + ":" + name
			parameters[key] = parameter
		}
	}
	return parameters, nil
}

func compileTool(name string, operation *openapi3.Operation, path string, method string, parameters map[string]*openapi3.Parameter, metadata mcpMetadata) (toolDefinition, error) {
	parameterInputs, err := compileParameterInputs(parameters)
	if err != nil {
		return toolDefinition{}, err
	}
	properties := parameterInputs.properties
	required := parameterInputs.required
	inputs := parameterInputs.inputs
	bodyInput, err := compileRequestBodyInput(operation)
	if err != nil {
		return toolDefinition{}, err
	}
	if bodyInput != nil {
		properties[inputBodyName] = bodyInput.schema
		if bodyInput.required {
			required = append(required, inputBodyName)
		}
		inputs = append(inputs, bodyInput.binding)
	}
	if metadata.confirmation.required {
		properties[confirmationTokenInputName] = map[string]any{"type": "string", "minLength": 1}
	}
	slices.Sort(required)
	return toolDefinition{
		name:        name,
		description: strings.TrimSpace(operation.Description),
		method:      method,
		path:        path,
		inputSchema: map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false},
		inputs:      inputs,
		metadata:    metadata,
	}, nil
}

func compileResource(tool toolDefinition, metadata mcpMetadata) (*resourceDefinition, error) {
	if len(metadata.resourceURITemplates) != 1 {
		return nil, fmt.Errorf("read operation must declare exactly one resource URI template")
	}
	return &resourceDefinition{
		name:        tool.name,
		description: tool.description,
		uriTemplate: metadata.resourceURITemplates[0],
		tool:        tool,
	}, nil
}

type compiledParameterInputs struct {
	properties map[string]any
	required   []string
	inputs     []inputBinding
}

func compileParameterInputs(parameters map[string]*openapi3.Parameter) (compiledParameterInputs, error) {
	result := compiledParameterInputs{
		properties: make(map[string]any),
		required:   make([]string, 0),
		inputs:     make([]inputBinding, 0),
	}
	keys := slices.Collect(maps.Keys(parameters))
	sort.Strings(keys)
	for _, key := range keys {
		parameter := parameters[key]
		if !isToolInputParameter(parameter) {
			continue
		}
		schema, binding, err := compileParameterInput(parameter, result.properties)
		if err != nil {
			return compiledParameterInputs{}, err
		}
		result.properties[parameter.Name] = schema
		if parameter.Required {
			result.required = append(result.required, parameter.Name)
		}
		result.inputs = append(result.inputs, binding)
	}
	return result, nil
}

func isToolInputParameter(parameter *openapi3.Parameter) bool {
	return parameter.In == "path" || parameter.In == "query"
}

func compileParameterInput(parameter *openapi3.Parameter, properties map[string]any) (any, inputBinding, error) {
	if parameter.Schema == nil || parameter.Schema.Value == nil {
		return nil, inputBinding{}, fmt.Errorf("%s parameter %q is missing a schema", parameter.In, parameter.Name)
	}
	if parameter.Name == inputBodyName {
		return nil, inputBinding{}, fmt.Errorf("%s parameter %q conflicts with reserved body input", parameter.In, parameter.Name)
	}
	if _, exists := properties[parameter.Name]; exists {
		return nil, inputBinding{}, fmt.Errorf("parameter %q has ambiguous input bindings", parameter.Name)
	}
	schema, err := schemaValue(parameter.Schema.Value)
	if err != nil {
		return nil, inputBinding{}, fmt.Errorf("encode %s parameter %q schema: %w", parameter.In, parameter.Name, err)
	}
	return schema, inputBinding{name: parameter.Name, location: parameter.In, required: parameter.Required, style: parameter.Style, explode: parameter.Explode}, nil
}

type requestBodyInput struct {
	schema   any
	binding  inputBinding
	required bool
}

func compileRequestBodyInput(operation *openapi3.Operation) (*requestBodyInput, error) {
	if operation.RequestBody == nil {
		return nil, nil
	}
	body := operation.RequestBody.Value
	if body == nil {
		return nil, fmt.Errorf("request body reference is unresolved")
	}
	content := body.Content.Get("application/json")
	if content == nil || content.Schema == nil || content.Schema.Value == nil {
		return nil, fmt.Errorf("request body must declare application/json schema")
	}
	schema, err := schemaValue(content.Schema.Value)
	if err != nil {
		return nil, fmt.Errorf("encode request body schema: %w", err)
	}
	return &requestBodyInput{schema: schema, binding: inputBinding{name: inputBodyName, location: inputBodyName, required: body.Required}, required: body.Required}, nil
}

func schemaValue(schema *openapi3.Schema) (any, error) {
	encoded, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func snakeCaseOperationID(operationID string) string {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return ""
	}
	var builder strings.Builder
	runes := []rune(operationID)
	for index, current := range runes {
		if isOperationIDWordRune(current) {
			appendOperationIDWord(&builder, runes, index)
			continue
		}
		if isOperationIDDelimiter(current) {
			appendOperationIDDelimiter(&builder)
			continue
		}
		return ""
	}
	return strings.Trim(builder.String(), "_")
}

func isOperationIDWordRune(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value)
}

func appendOperationIDWord(builder *strings.Builder, runes []rune, index int) {
	current := runes[index]
	if unicode.IsUpper(current) && index > 0 && needsWordBoundary(runes, index) {
		appendOperationIDDelimiter(builder)
	}
	builder.WriteRune(unicode.ToLower(current))
}

func isOperationIDDelimiter(value rune) bool {
	return value == '_' || value == '-' || unicode.IsSpace(value)
}

func appendOperationIDDelimiter(builder *strings.Builder) {
	if builder.Len() > 0 && !strings.HasSuffix(builder.String(), "_") {
		builder.WriteByte('_')
	}
}

func needsWordBoundary(runes []rune, index int) bool {
	previous := runes[index-1]
	if unicode.IsLower(previous) || unicode.IsDigit(previous) {
		return true
	}
	return unicode.IsUpper(previous) && index+1 < len(runes) && unicode.IsLower(runes[index+1])
}

func positiveISODuration(value string) bool {
	matches := isoDurationPattern.FindStringSubmatch(value)
	if matches == nil || strings.HasSuffix(value, "T") {
		return false
	}
	return durationHasPositiveComponent(matches) && durationUsesOnlyWeeks(matches)
}

func durationHasPositiveComponent(matches []string) bool {
	for _, component := range matches[1:] {
		if strings.TrimLeft(component, "0") != "" {
			return true
		}
	}
	return false
}

func durationUsesOnlyWeeks(matches []string) bool {
	if matches[isoWeeksCaptureIndex] == "" {
		return true
	}
	for index, component := range matches[1:] {
		if index != isoWeeksCaptureIndex-1 && component != "" {
			return false
		}
	}
	return true
}

func objectValue(value any) (map[string]any, bool) {
	object, ok := value.(map[string]any)
	return object, ok
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func anyStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil
		}
		result = append(result, strings.TrimSpace(text))
	}
	return result
}

func stringMap(value any) map[string]string {
	object, ok := objectValue(value)
	if !ok {
		return nil
	}
	result := make(map[string]string, len(object))
	for key, raw := range object {
		text, ok := raw.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return nil
		}
		result[strings.TrimSpace(key)] = strings.TrimSpace(text)
	}
	return result
}

func containsString(values []string, target string) bool {
	return slices.Contains(values, target)
}
