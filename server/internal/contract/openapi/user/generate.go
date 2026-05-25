package useropenapi

//go:generate go tool oapi-codegen --include-operation-ids postUsers,postUserUpdate --generate types --package useropenapi -o zz_generated.write.go ../../../../../openapi/openapi.yaml
