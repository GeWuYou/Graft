// Package networkopenapi 持有由 canonical OpenAPI 派生的 Network 连通性类型。
package networkopenapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml -o zz_generated.connectivity.go ../../../../../openapi/openapi.yaml
