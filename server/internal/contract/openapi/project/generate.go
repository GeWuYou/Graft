// Package projectopenapi 持有由共享 OpenAPI 生成的应用请求与服务端绑定。
package projectopenapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml -o zz_generated.project.go ../../../../../openapi/openapi.yaml
