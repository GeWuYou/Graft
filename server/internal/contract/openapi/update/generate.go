// Package updateopenapi 持有由共享 OpenAPI 生成的平台更新请求与服务端绑定。
package updateopenapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml -o zz_generated.update.go ../../../../../openapi/openapi.yaml
