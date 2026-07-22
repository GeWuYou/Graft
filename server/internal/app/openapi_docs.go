package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"graft/server/internal/buildinfo"
)

const (
	openapiJSONPath           = "/openapi.json"
	openapiDocsPath           = "/docs"
	openapiBundleSourcePath   = "openapi/dist/openapi.bundle.json"
	scalarDocsScriptURL       = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.57.5/dist/browser/standalone.js"
	scalarDocsScriptIntegrity = "sha384-t5h38o34qqR7GUJVk2SXZl4p7wXfwNuV04PZALl5ae4ih2PEwQtGRPLiAax9r7V8"
)

const scalarDocsCustomCSS = `html, body {
  height: 100%;
  overflow: hidden;
}

.scalar-api-reference.references-layout {
  height: 100%;
  max-height: 100%;
  min-height: 0;
  grid-template-rows: var(--scalar-header-height, 0px) minmax(0, 1fr) auto;
}

.scalar-api-reference.references-layout .references-rendered {
  min-height: 0;
  overflow-y: auto;
  scrollbar-color: var(--scalar-scrollbar-color, transparent) transparent;
  scrollbar-width: thin;
}

.scalar-api-reference.references-layout .references-rendered::-webkit-scrollbar {
  width: 12px;
}

.scalar-api-reference.references-layout .references-rendered::-webkit-scrollbar-track {
  background: transparent;
}

.scalar-api-reference.references-layout .references-rendered::-webkit-scrollbar-thumb {
  background: var(--scalar-scrollbar-color, transparent);
  background-clip: content-box;
  border: 3px solid transparent;
  border-radius: 20px;
}

.scalar-api-reference.references-layout .references-rendered::-webkit-scrollbar-thumb:active {
  background: var(--scalar-scrollbar-color-active, transparent);
  background-clip: content-box;
  border: 3px solid transparent;
}

.graft-docs-operation-method {
  color: var(--graft-docs-operation-color);
  font-family: var(--scalar-font-code, ui-monospace, SFMono-Regular, Consolas, monospace);
  font-size: 0.875em;
  font-weight: 650;
}

.graft-docs-operation-method-get { --graft-docs-operation-color: #0082d0; }
.graft-docs-operation-method-post { --graft-docs-operation-color: #0d9f6e; }
.graft-docs-operation-method-put { --graft-docs-operation-color: #d97706; }
.graft-docs-operation-method-patch { --graft-docs-operation-color: #c2410c; }
.graft-docs-operation-method-delete { --graft-docs-operation-color: #dc2626; }

@media (max-width: 1000px) {
  .scalar-api-reference.references-layout {
    grid-template-rows: var(--scalar-header-height, 0px) 0 minmax(0, 1fr) auto;
  }
}`

var scalarDocsPageTemplate = template.Must(template.New("scalar-docs").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="icon" type="image/svg+xml" href="/favicon.svg?v=3">
    <title>Graft API Docs</title>
    <style>
      :root { color-scheme: light dark; }
      html, body { height: 100%; }
      body { margin: 0; overflow: hidden; }
      .graft-docs-shell { display: grid; grid-template-rows: auto minmax(0, 1fr); height: 100dvh; }
      .graft-docs-overview { display: flex; align-items: center; gap: 16px; min-width: 0; padding: 8px 20px; border-bottom: 1px solid color-mix(in srgb, CanvasText 18%, transparent); background: Canvas; color: CanvasText; font: 12px/1.3 ui-sans-serif, system-ui, sans-serif; }
      .graft-docs-overview-title { flex: 0 0 auto; font-weight: 650; }
      .graft-docs-stat-list { display: flex; flex: 1 1 auto; flex-wrap: wrap; gap: 6px; min-width: 0; margin: 0; }
      .graft-docs-stat { display: flex; align-items: baseline; gap: 5px; margin: 0; padding: 4px 8px; border: 1px solid color-mix(in srgb, CanvasText 16%, transparent); border-radius: 4px; }
      .graft-docs-stat dt { color: color-mix(in srgb, CanvasText 68%, transparent); font-weight: 500; }
      .graft-docs-stat dd { margin: 0; font: 650 13px/1 ui-monospace, SFMono-Regular, Consolas, monospace; }
      .graft-docs-stat[data-operation-method] dt, .graft-docs-stat[data-operation-method] dd { color: var(--graft-operation-color); }
      .graft-docs-stat[data-operation-method="GET"] { --graft-operation-color: #0082d0; }
      .graft-docs-stat[data-operation-method="POST"] { --graft-operation-color: #0d9f6e; }
      .graft-docs-stat[data-operation-method="PUT"] { --graft-operation-color: #d97706; }
      .graft-docs-stat[data-operation-method="PATCH"] { --graft-operation-color: #c2410c; }
      .graft-docs-stat[data-operation-method="DELETE"] { --graft-operation-color: #dc2626; }
      .graft-scalar-container { min-height: 0; overflow: hidden; }
      .graft-scalar-container > div, .graft-scalar-container > div > div { height: 100%; min-height: 0; }
      @media (max-width: 640px) {
        .graft-docs-overview { align-items: flex-start; flex-direction: column; gap: 6px; padding: 10px 14px; }
        .graft-docs-stat-list { width: 100%; }
      }
    </style>
  </head>
  <body>
    <main class="graft-docs-shell">
      <section class="graft-docs-overview" aria-label="Documentation health summary">
        <span class="graft-docs-overview-title">Documentation Health</span>
        <dl class="graft-docs-stat-list">
          <div class="graft-docs-stat"><dt>Total APIs</dt><dd data-operation-count="{{ .Summary.Total }}">{{ .Summary.Total }}</dd></div>
          <div class="graft-docs-stat"><dt>Deprecated</dt><dd data-operation-count="{{ .Summary.Deprecated }}">{{ .Summary.Deprecated }}</dd></div>
          <div class="graft-docs-stat"><dt>OpenAPI Version</dt><dd>{{ .Summary.OpenAPIVersion }}</dd></div>
          <div class="graft-docs-stat"><dt>Tags</dt><dd>{{ .Summary.Tags }}</dd></div>
          <div class="graft-docs-stat"><dt>Tagged Operations</dt><dd>{{ .Summary.TaggedOperations }}</dd></div>
          {{ range .Summary.Methods }}
          <div class="graft-docs-stat" data-operation-method="{{ .Method }}"><dt>{{ .Method }}</dt><dd data-operation-count="{{ .Count }}">{{ .Count }}</dd></div>
          {{ end }}
        </dl>
      </section>
      <div class="graft-scalar-container">
        <script id="api-reference" data-configuration="{{ .Configuration }}"></script>
      </div>
    </main>
    <script src="` + scalarDocsScriptURL + `" integrity="` + scalarDocsScriptIntegrity + `" crossorigin="anonymous"></script>
  </body>
</html>`))

type openAPIDocsAssets struct {
	json    []byte
	summary openAPIDocsOperationSummary
}

type openAPIDocsOperationSummary struct {
	Total            int
	Deprecated       int
	OpenAPIVersion   string
	Tags             int
	TaggedOperations int
	Methods          []openAPIDocsMethodCount
}

type openAPIDocsMethodCount struct {
	Method string
	Count  int
}

var openAPIDocsMethodOrder = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodHead,
	http.MethodOptions,
	http.MethodTrace,
	http.MethodConnect,
}

// OpenAPIDocsBundleSourcePath 返回仓库中规范 OpenAPI 打包源文件的路径。
func OpenAPIDocsBundleSourcePath() string {
	return openapiBundleSourcePath
}

// OpenAPIDocsBundleSHA256 返回嵌入式 OpenAPI 打包资源的摘要。
func OpenAPIDocsBundleSHA256() string {
	return generatedOpenAPIBundleSHA256
}

// OpenAPIDocsBundle 返回产品 runtime 使用的 canonical OpenAPI 打包文档副本。
// MCP compiler 必须消费这份确定性生成物，不能从磁盘重新拼装或维护第二套 Tool 清单。
func OpenAPIDocsBundle() []byte {
	return append([]byte(nil), generatedOpenAPIBundleJSON...)
}

// loadOpenAPIDocsAssets 加载并校验嵌入式 OpenAPI 文档资源。
func loadOpenAPIDocsAssets() (*openAPIDocsAssets, error) {
	return buildOpenAPIDocsAssets(generatedOpenAPIBundleJSON, buildinfo.Current())
}

// buildOpenAPIDocsAssets 验证 OpenAPI 规范并构建运行时文档资源。
// 它根据构建信息设置规范版本，并拒绝仍包含外部文件引用的规范。
func buildOpenAPIDocsAssets(spec []byte, build buildinfo.Info) (*openAPIDocsAssets, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("generated bundled openapi spec is empty")
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	document, err := loader.LoadFromData(spec)
	if err != nil {
		return nil, fmt.Errorf("load generated bundled openapi spec: %w", err)
	}
	if err := document.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate generated bundled openapi spec: %w", err)
	}
	if document.Info == nil {
		return nil, fmt.Errorf("generated bundled openapi spec is missing info")
	}

	enrichOpenAPITagDescriptions(document)
	document.Info.Version = buildinfo.Normalize(build).Version
	runtimeSpec, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode runtime openapi spec: %w", err)
	}
	if bytes.Contains(runtimeSpec, []byte("./paths/")) || bytes.Contains(runtimeSpec, []byte("./components/")) {
		return nil, fmt.Errorf("generated bundled openapi spec still contains external file refs")
	}

	summary := summarizeOpenAPIOperations(document.Paths)
	summary.OpenAPIVersion = document.OpenAPI
	summary.Tags = len(document.Tags)
	summary.TaggedOperations = countTaggedOpenAPIOperations(document.Paths)

	return &openAPIDocsAssets{json: runtimeSpec, summary: summary}, nil
}

func countTaggedOpenAPIOperations(paths *openapi3.Paths) int {
	count := 0
	if paths == nil {
		return count
	}
	for _, pathItem := range paths.Map() {
		if pathItem == nil {
			continue
		}
		for _, operation := range pathItem.Operations() {
			if operation != nil && len(operation.Tags) > 0 {
				count++
			}
		}
	}
	return count
}

func summarizeOpenAPIOperations(paths *openapi3.Paths) openAPIDocsOperationSummary {
	counts := make(map[string]int)
	summary := openAPIDocsOperationSummary{}
	for _, pathItem := range paths.Map() {
		if pathItem == nil {
			continue
		}
		for method, operation := range pathItem.Operations() {
			counts[method]++
			if operation != nil && operation.Deprecated {
				summary.Deprecated++
			}
		}
	}

	for _, method := range openAPIDocsMethodOrder {
		count := counts[method]
		if count == 0 {
			continue
		}
		summary.Total += count
		summary.Methods = append(summary.Methods, openAPIDocsMethodCount{Method: method, Count: count})
	}
	return summary
}

type openAPIDocsTagSummary struct {
	Total              int
	Deprecated         int
	AuthenticatedTotal int
	MethodCounts       map[string]int
}

func enrichOpenAPITagDescriptions(document *openapi3.T) {
	if document == nil || document.Paths == nil || len(document.Tags) == 0 {
		return
	}

	summaries := collectOpenAPITagSummaries(document.Paths, document.Security)
	appendOpenAPITagDashboards(document.Tags, summaries)
}

func collectOpenAPITagSummaries(paths *openapi3.Paths, defaultRequirements openapi3.SecurityRequirements) map[string]*openAPIDocsTagSummary {
	summaries := make(map[string]*openAPIDocsTagSummary)
	if paths == nil {
		return summaries
	}
	for _, pathItem := range paths.Map() {
		collectPathItemTagSummaries(pathItem, defaultRequirements, summaries)
	}
	return summaries
}

func collectPathItemTagSummaries(pathItem *openapi3.PathItem, defaultRequirements openapi3.SecurityRequirements, summaries map[string]*openAPIDocsTagSummary) {
	if pathItem == nil {
		return
	}
	for method, operation := range pathItem.Operations() {
		collectOperationTagSummary(method, operation, defaultRequirements, summaries)
	}
}

func collectOperationTagSummary(method string, operation *openapi3.Operation, defaultRequirements openapi3.SecurityRequirements, summaries map[string]*openAPIDocsTagSummary) {
	if operation == nil {
		return
	}
	authenticated := operationRequiresAuthentication(operation, defaultRequirements)
	for _, tagName := range operation.Tags {
		summary := summaries[tagName]
		if summary == nil {
			summary = &openAPIDocsTagSummary{MethodCounts: make(map[string]int)}
			summaries[tagName] = summary
		}
		summary.Total++
		summary.MethodCounts[method]++
		if operation.Deprecated {
			summary.Deprecated++
		}
		if authenticated {
			summary.AuthenticatedTotal++
		}
	}
}

func appendOpenAPITagDashboards(tags openapi3.Tags, summaries map[string]*openAPIDocsTagSummary) {
	for _, tag := range tags {
		if tag == nil || tag.Name == "" {
			continue
		}
		summary := summaries[tag.Name]
		if summary == nil || summary.Total == 0 {
			continue
		}
		tag.Description = joinOpenAPITagDescription(tag.Description, renderOpenAPITagDashboard(*summary))
	}
}

func operationRequiresAuthentication(operation *openapi3.Operation, defaultRequirements openapi3.SecurityRequirements) bool {
	if operation == nil {
		return false
	}
	if operation.Security == nil {
		return len(defaultRequirements) > 0
	}
	return len(*operation.Security) > 0
}

func joinOpenAPITagDescription(description string, dashboard string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return dashboard
	}
	return description + "\n\n---\n\n" + dashboard
}

func renderOpenAPITagDashboard(summary openAPIDocsTagSummary) string {
	lines := []string{
		"### Overview",
		fmt.Sprintf("%d operations", summary.Total),
	}
	for _, method := range openAPIDocsMethodOrder {
		if count := summary.MethodCounts[method]; count > 0 {
			lines = append(lines, fmt.Sprintf(`- <span class="graft-docs-operation-method graft-docs-operation-method-%s">%s</span>: %d`, strings.ToLower(method), method, count))
		}
	}
	lines = append(
		lines,
		fmt.Sprintf("Deprecated: %d operations.", summary.Deprecated),
		"",
		"### Security",
		fmt.Sprintf("Authentication required for %d of %d operations.", summary.AuthenticatedTotal, summary.Total),
	)
	return strings.Join(lines, "\n")
}

// renderScalarDocsHTML 根据指定的 OpenAPI 规范 URL 渲染 Scalar 文档 HTML 页面。
//
// specURL 指定页面加载的 OpenAPI 规范地址。
//
// 返回渲染后的 HTML 内容；如果配置编码或模板渲染失败，则返回错误。
func renderScalarDocsHTML(specURL string, summary openAPIDocsOperationSummary) ([]byte, error) {
	configuration, err := json.Marshal(struct {
		URL               string `json:"url"`
		Layout            string `json:"layout"`
		ShowSidebar       bool   `json:"showSidebar"`
		CustomCSS         string `json:"customCss"`
		DefaultHTTPClient struct {
			TargetKey string `json:"targetKey"`
			ClientKey string `json:"clientKey"`
		} `json:"defaultHttpClient"`
	}{
		URL:         specURL,
		Layout:      "modern",
		ShowSidebar: true,
		CustomCSS:   scalarDocsCustomCSS,
		DefaultHTTPClient: struct {
			TargetKey string `json:"targetKey"`
			ClientKey string `json:"clientKey"`
		}{TargetKey: "shell", ClientKey: "curl"},
	})
	if err != nil {
		return nil, fmt.Errorf("encode Scalar docs configuration: %w", err)
	}

	var buffer bytes.Buffer
	data := struct {
		Configuration string
		Summary       openAPIDocsOperationSummary
	}{
		Configuration: string(configuration),
		Summary:       summary,
	}
	if err := scalarDocsPageTemplate.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("render scalar docs html: %w", err)
	}
	return buffer.Bytes(), nil
}
