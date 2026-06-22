package app

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	openapiJSONPath           = "/openapi.json"
	openapiDocsPath           = "/docs"
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

//go:embed openapi.bundle.json
var embeddedOpenAPIBundleJSON []byte

type openAPIDocsAssets struct {
	json []byte
}

func loadOpenAPIDocsAssets() (*openAPIDocsAssets, error) {
	if len(embeddedOpenAPIBundleJSON) == 0 {
		return nil, fmt.Errorf("embedded bundled openapi spec is empty")
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	document, err := loader.LoadFromData(embeddedOpenAPIBundleJSON)
	if err != nil {
		return nil, fmt.Errorf("load embedded bundled openapi spec: %w", err)
	}
	if err := document.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate embedded bundled openapi spec: %w", err)
	}
	if bytes.Contains(embeddedOpenAPIBundleJSON, []byte("./paths/")) || bytes.Contains(embeddedOpenAPIBundleJSON, []byte("./components/")) {
		return nil, fmt.Errorf("embedded bundled openapi spec still contains external file refs")
	}

	return &openAPIDocsAssets{
		json: embeddedOpenAPIBundleJSON,
	}, nil
}

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
