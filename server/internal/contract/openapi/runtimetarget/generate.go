// Package runtimetargetopenapi 持有由共享 OpenAPI 生成的运行时目标请求与服务端绑定。
package runtimetargetopenapi

//go:generate go tool oapi-codegen -config oapi-codegen.yaml -o zz_generated.runtime_target.go ../../../../../openapi/openapi.yaml
