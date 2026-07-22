package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"

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
  height: 100dvh;
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
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script id="api-reference" data-configuration="{{ .Configuration }}"></script>
    <script src="` + scalarDocsScriptURL + `" integrity="` + scalarDocsScriptIntegrity + `" crossorigin="anonymous"></script>
  </body>
</html>`))

type openAPIDocsAssets struct {
	json []byte
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

	document.Info.Version = buildinfo.Normalize(build).Version
	runtimeSpec, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode runtime openapi spec: %w", err)
	}
	if bytes.Contains(runtimeSpec, []byte("./paths/")) || bytes.Contains(runtimeSpec, []byte("./components/")) {
		return nil, fmt.Errorf("generated bundled openapi spec still contains external file refs")
	}

	return &openAPIDocsAssets{
		json: runtimeSpec,
	}, nil
}

// renderScalarDocsHTML 根据指定的 OpenAPI 规范 URL 渲染 Scalar 文档 HTML 页面。
//
// specURL 指定页面加载的 OpenAPI 规范地址。
//
// 返回渲染后的 HTML 内容；如果配置编码或模板渲染失败，则返回错误。
func renderScalarDocsHTML(specURL string) ([]byte, error) {
	configuration, err := json.Marshal(struct {
		URL       string `json:"url"`
		Layout    string `json:"layout"`
		CustomCSS string `json:"customCss"`
	}{
		URL:       specURL,
		Layout:    "modern",
		CustomCSS: scalarDocsCustomCSS,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Scalar docs configuration: %w", err)
	}

	var buffer bytes.Buffer
	data := struct {
		Configuration string
	}{
		Configuration: string(configuration),
	}
	if err := scalarDocsPageTemplate.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("render scalar docs html: %w", err)
	}
	return buffer.Bytes(), nil
}
