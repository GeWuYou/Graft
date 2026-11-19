// Package registryopenapi 持有由共享 OpenAPI 生成的镜像仓库请求与服务端绑定。
package registryopenapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml -o zz_generated.registry.go ../../../../../openapi/openapi.yaml
