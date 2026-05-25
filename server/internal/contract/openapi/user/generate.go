package useropenapi

//go:generate go tool oapi-codegen --include-operation-ids getUsers,getUserById,postUsers,postUserUpdate,postUserStatus,postUserResetPassword,postUserDelete --generate types --package useropenapi -o zz_generated.management.go ../../../../../openapi/openapi.yaml
