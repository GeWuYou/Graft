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

.scalar-app,
.scalar-app .references-layout {
  height: 100dvh;
  min-height: 0;
}

.scalar-app .references-layout {
  grid-template-rows: var(--scalar-header-height, 0px) minmax(0, 1fr) auto;
}

.scalar-app .references-rendered {
  min-height: 0;
  overflow-y: auto;
  scrollbar-color: var(--scalar-scrollbar-color, transparent) transparent;
  scrollbar-width: thin;
}

.scalar-app .references-rendered::-webkit-scrollbar {
  width: 12px;
}

.scalar-app .references-rendered::-webkit-scrollbar-track {
  background: transparent;
}

.scalar-app .references-rendered::-webkit-scrollbar-thumb {
  background: var(--scalar-scrollbar-color, transparent);
  background-clip: content-box;
  border: 3px solid transparent;
  border-radius: 20px;
}

.scalar-app .references-rendered::-webkit-scrollbar-thumb:active {
  background: var(--scalar-scrollbar-color-active, transparent);
  background-clip: content-box;
  border: 3px solid transparent;
}

@media (max-width: 1000px) {
  .scalar-app .references-layout {
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

// OpenAPIDocsBundleSourcePath returns the canonical bundled OpenAPI source path in the repository.
func OpenAPIDocsBundleSourcePath() string {
	return openapiBundleSourcePath
}

// OpenAPIDocsBundleSHA256 returns the digest of the embedded bundled OpenAPI asset.
func OpenAPIDocsBundleSHA256() string {
	return generatedOpenAPIBundleSHA256
}

// loadOpenAPIDocsAssets loads and validates the embedded OpenAPI documentation assets.
func loadOpenAPIDocsAssets() (*openAPIDocsAssets, error) {
	return buildOpenAPIDocsAssets(generatedOpenAPIBundleJSON, buildinfo.Current())
}

// buildOpenAPIDocsAssets 从规范字节构建 OpenAPI 文档资源。它验证规范的有效性，确保规范为完整的打包内容且不包含外部文件引用。
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

// renderScalarDocsHTML 渲染 Scalar 文档 HTML 页面，其中包含指定的 OpenAPI 规范 URL。
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
