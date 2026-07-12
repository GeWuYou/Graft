// Package runtimetargetopenapi owns generated OpenAPI route boundary types.
package runtimetargetopenapi

//go:generate go tool oapi-codegen --include-operation-ids getRuntimeTargets,getRuntimeTarget,postRuntimeTargetRefresh --generate types --package runtimetargetopenapi -o zz_generated.runtime_target.go ../../../../../openapi/openapi.yaml
