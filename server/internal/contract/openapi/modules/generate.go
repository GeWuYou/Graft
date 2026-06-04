package modulesopenapi

//go:generate go tool oapi-codegen --include-operation-ids getModulesRuntime,getModulesRuntimeModule --generate types --package modulesopenapi -o zz_generated.modules.go ../../../../../openapi/openapi.yaml
