package systemconfigopenapi

//go:generate go tool oapi-codegen --include-operation-ids getSystemConfigs,getSystemConfig,putSystemConfig,postSystemConfigReset --generate types --package systemconfigopenapi -o zz_generated.systemconfig.go ../../../../../openapi/openapi.yaml
