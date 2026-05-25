package authopenapi

//go:generate go tool oapi-codegen --include-operation-ids postAuthLogin,getAuthBootstrap --generate types --package authopenapi -o zz_generated.auth.go ../../../../../openapi/openapi.yaml
