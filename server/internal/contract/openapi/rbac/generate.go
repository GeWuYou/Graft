package rbacopenapi

//go:generate go tool oapi-codegen --include-operation-ids getPermissions --generate types --package rbacopenapi -o zz_generated.permissions.go ../../../../../openapi/openapi.yaml
