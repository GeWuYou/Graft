package openapi

//go:generate node ../../../../scripts/ensure-openapi-bundle.mjs
//go:generate go tool oapi-codegen --generate types --package generated -o generated/types.gen.go ../../../../openapi/dist/openapi.bundle.json
