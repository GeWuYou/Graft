package healthopenapi

//go:generate go tool oapi-codegen --include-operation-ids getHealthz --generate types --package healthopenapi -o zz_generated.health.go ../../../../../openapi/openapi.yaml
