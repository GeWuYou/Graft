package rbacopenapi

//go:generate go tool oapi-codegen --include-operation-ids getPermissions,getRoles,getRolePermissions --generate types --package rbacopenapi -o zz_generated.read.go ../../../../../openapi/openapi.yaml
