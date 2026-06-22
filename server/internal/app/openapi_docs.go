package app

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	openapiJSONPath           = "/openapi.json"
	openapiDocsPath           = "/docs"
	openapiBundleSourcePath   = "openapi/dist/openapi.bundle.json"
	scalarDocsScriptURL       = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.57.5/dist/browser/standalone.js"
	scalarDocsScriptIntegrity = "sha384-t5h38o34qqR7GUJVk2SXZl4p7wXfwNuV04PZALl5ae4ih2PEwQtGRPLiAax9r7V8"
)

var scalarDocsPageTemplate = template.Must(template.New("scalar-docs").Parse(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Graft API Docs</title>
    <style>
      body { margin: 0; }
    </style>
  </head>
  <body>
    <script id="api-reference" data-url="{{ .SpecURL }}"></script>
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
	return buildOpenAPIDocsAssets(generatedOpenAPIBundleJSON)
}

// buildOpenAPIDocsAssets 从规范字节构建 OpenAPI 文档资源。它验证规范的有效性，确保规范为完整的打包内容且不包含外部文件引用。
func buildOpenAPIDocsAssets(spec []byte) (*openAPIDocsAssets, error) {
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
	if bytes.Contains(spec, []byte("./paths/")) || bytes.Contains(spec, []byte("./components/")) {
		return nil, fmt.Errorf("generated bundled openapi spec still contains external file refs")
	}

	return &openAPIDocsAssets{
		json: spec,
	}, nil
}

// renderScalarDocsHTML 渲染 Scalar 文档 HTML 页面，其中包含指定的 OpenAPI 规范 URL。
func renderScalarDocsHTML(specURL string) ([]byte, error) {
	var buffer bytes.Buffer
	data := struct {
		SpecURL string
	}{
		SpecURL: specURL,
	}
	if err := scalarDocsPageTemplate.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("render scalar docs html: %w", err)
	}
	return buffer.Bytes(), nil
}
