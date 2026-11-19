// Package buildopenapi 持有由共享 OpenAPI 生成的构建请求与服务端绑定。
package buildopenapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml -o zz_generated.build.go ../../../../../openapi/openapi.yaml
