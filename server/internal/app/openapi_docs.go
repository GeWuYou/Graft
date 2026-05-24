package app

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	openapiRootSpecRelativePath   = "openapi/openapi.yaml"
	openapiBundleSpecRelativePath = "openapi/dist/openapi.bundle.json"
	openapiJSONPath               = "/openapi.json"
	openapiYAMLPath               = "/openapi.yaml"
	openapiDocsPath               = "/docs"
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
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference" integrity="sha384-t5h38o34qqR7GUJVk2SXZl4p7wXfwNuV04PZALl5ae4ih2PEwQtGRPLiAax9r7V8" crossorigin="anonymous"></script>
  </body>
</html>`))

type openAPIDocsAssets struct {
	json []byte
	yaml []byte
}

func loadOpenAPIDocsAssets() (*openAPIDocsAssets, error) {
	repositoryRoot, err := resolveRepositoryRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}

	rootSpecPath := filepath.Join(repositoryRoot, filepath.FromSlash(openapiRootSpecRelativePath))
	// #nosec G304 -- rootSpecPath is constrained to the repository-owned openapi spec under the resolved repo root.
	yamlContent, err := os.ReadFile(rootSpecPath)
	if err != nil {
		return nil, fmt.Errorf("read openapi spec %q: %w", rootSpecPath, err)
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	rootDocument, err := loader.LoadFromDataWithPath(yamlContent, &url.URL{Path: filepath.ToSlash(rootSpecPath)})
	if err != nil {
		return nil, fmt.Errorf("load openapi spec %q: %w", rootSpecPath, err)
	}
	if err := rootDocument.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate openapi spec %q: %w", rootSpecPath, err)
	}

	bundleSpecPath := filepath.Join(repositoryRoot, filepath.FromSlash(openapiBundleSpecRelativePath))
	// #nosec G304 -- bundleSpecPath is constrained to the repository-owned bundled openapi spec under the resolved repo root.
	jsonContent, err := os.ReadFile(bundleSpecPath)
	if err != nil {
		return nil, fmt.Errorf("read bundled openapi spec %q: %w", bundleSpecPath, err)
	}

	bundleDocument, err := loader.LoadFromData(jsonContent)
	if err != nil {
		return nil, fmt.Errorf("load bundled openapi spec %q: %w", bundleSpecPath, err)
	}
	if err := bundleDocument.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate bundled openapi spec %q: %w", bundleSpecPath, err)
	}
	if bytes.Contains(jsonContent, []byte("./paths/")) || bytes.Contains(jsonContent, []byte("./components/")) {
		return nil, fmt.Errorf("bundled openapi spec %q still contains external file refs", bundleSpecPath)
	}

	return &openAPIDocsAssets{
		json: jsonContent,
		yaml: yamlContent,
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

func resolveRepositoryRoot() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	current := workingDirectory
	for {
		if _, openapiErr := os.Stat(filepath.Join(current, "openapi", "openapi.yaml")); openapiErr == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("find repository root from %q", strings.TrimSpace(workingDirectory))
}
