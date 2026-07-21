package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const objectQueryPartsPerEntry = 2

// dispatcher 将 OpenAPI 编译出的 Tool 请求原地派发回同一 Gin engine。
// 它不重写模块 handler、鉴权或审计逻辑，REST 路由仍是业务行为的唯一执行入口。
type dispatcher struct {
	engine *gin.Engine
}

func newDispatcher(engine *gin.Engine) (*dispatcher, error) {
	if engine == nil {
		return nil, fmt.Errorf("MCP dispatcher engine is unavailable")
	}
	return &dispatcher{engine: engine}, nil
}

func (d *dispatcher) toolHandler(definition toolDefinition) mcpsdk.ToolHandler {
	return func(ctx context.Context, request *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		arguments, err := toolArguments(request)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return d.dispatch(ctx, definition, arguments)
	}
}

func toolArguments(request *mcpsdk.CallToolRequest) (map[string]any, error) {
	if request == nil || request.Params == nil || len(request.Params.Arguments) == 0 {
		return map[string]any{}, nil
	}
	var arguments map[string]any
	if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
		return nil, fmt.Errorf("MCP tool arguments must be a JSON object: %w", err)
	}
	if arguments == nil {
		return map[string]any{}, nil
	}
	return arguments, nil
}

func (d *dispatcher) dispatch(ctx context.Context, definition toolDefinition, arguments map[string]any) (*mcpsdk.CallToolResult, error) {
	if d == nil || d.engine == nil {
		return nil, fmt.Errorf("MCP dispatcher engine is unavailable")
	}
	target, body, err := definition.request(arguments)
	if err != nil {
		return toolErrorResult(err), nil
	}
	var requestBody io.Reader
	if body != nil {
		requestBody = body
	}
	request, err := http.NewRequestWithContext(ctx, definition.method, target, requestBody)
	if err != nil {
		return nil, fmt.Errorf("build in-process REST request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	d.engine.ServeHTTP(recorder, request)
	return restResponseResult(recorder), nil
}

func (d toolDefinition) request(arguments map[string]any) (string, *bytes.Reader, error) {
	if err := d.validateArguments(arguments); err != nil {
		return "", nil, err
	}
	bound, err := d.bindArguments(arguments)
	if err != nil {
		return "", nil, err
	}
	if strings.ContainsAny(bound.path, "{}") {
		return "", nil, fmt.Errorf("MCP tool %q has unresolved path parameters", d.name)
	}
	if encoded := bound.query.Encode(); encoded != "" {
		bound.path += "?" + encoded
	}
	return bound.path, bound.body, nil
}

func (d toolDefinition) validateArguments(arguments map[string]any) error {
	allowed := make(map[string]inputBinding, len(d.inputs))
	for _, input := range d.inputs {
		allowed[input.name] = input
	}
	for name := range arguments {
		if _, exists := allowed[name]; !exists {
			return fmt.Errorf("MCP tool %q does not accept argument %q", d.name, name)
		}
	}
	return nil
}

type boundToolArguments struct {
	path  string
	query url.Values
	body  *bytes.Reader
}

func (d toolDefinition) bindArguments(arguments map[string]any) (boundToolArguments, error) {
	bound := boundToolArguments{path: d.path, query: make(url.Values)}
	for _, input := range d.inputs {
		value, present := arguments[input.name]
		if input.required && !present {
			return boundToolArguments{}, fmt.Errorf("MCP tool %q requires argument %q", d.name, input.name)
		}
		if !present {
			continue
		}
		var err error
		bound.path, bound.body, err = d.bindInput(bound.path, bound.query, bound.body, input, value)
		if err != nil {
			return boundToolArguments{}, err
		}
	}
	return bound, nil
}

func (d toolDefinition) bindInput(path string, query url.Values, body *bytes.Reader, input inputBinding, value any) (string, *bytes.Reader, error) {
	switch input.location {
	case "path":
		return d.bindPathInput(path, body, input, value)
	case "query":
		appendQueryArgument(query, input, value)
		return path, body, nil
	case inputBodyName:
		return d.bindBodyInput(path, value)
	}
	return path, body, fmt.Errorf("MCP tool %q has unsupported input location %q", d.name, input.location)
}

func (d toolDefinition) bindPathInput(path string, body *bytes.Reader, input inputBinding, value any) (string, *bytes.Reader, error) {
	placeholder := "{" + input.name + "}"
	if !strings.Contains(path, placeholder) {
		return "", nil, fmt.Errorf("MCP tool %q path does not contain parameter %q", d.name, input.name)
	}
	return strings.ReplaceAll(path, placeholder, url.PathEscape(argumentString(value))), body, nil
}

func (d toolDefinition) bindBodyInput(path string, value any) (string, *bytes.Reader, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", nil, fmt.Errorf("encode MCP tool %q body: %w", d.name, err)
	}
	return path, bytes.NewReader(encoded), nil
}

func appendQueryArgument(query url.Values, input inputBinding, value any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			query.Add(input.name, argumentString(item))
		}
	case []string:
		for _, item := range typed {
			query.Add(input.name, item)
		}
	case map[string]any:
		appendObjectQueryArgument(query, input, typed)
	default:
		query.Add(input.name, argumentString(value))
	}
}

func appendObjectQueryArgument(query url.Values, input inputBinding, values map[string]any) {
	if input.explode == nil || *input.explode {
		for key, value := range values {
			query.Add(key, argumentString(value))
		}
		return
	}
	parts := make([]string, 0, len(values)*objectQueryPartsPerEntry)
	for key, value := range values {
		parts = append(parts, key, argumentString(value))
	}
	query.Add(input.name, strings.Join(parts, ","))
}

func argumentString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(typed)
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func restResponseResult(recorder *httptest.ResponseRecorder) *mcpsdk.CallToolResult {
	if recorder == nil {
		return toolErrorResult(fmt.Errorf("in-process REST response is unavailable"))
	}
	body := recorder.Body.Bytes()
	result := &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(body)}},
		IsError: recorder.Code < http.StatusOK || recorder.Code >= http.StatusMultipleChoices,
	}
	if !result.IsError {
		var structured map[string]any
		if json.Unmarshal(body, &structured) == nil {
			result.StructuredContent = structured
		}
	}
	return result
}

func toolErrorResult(err error) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: err.Error()}},
		IsError: true,
	}
}
