package useropenapi

//go:generate go tool oapi-codegen --include-operation-ids getUsers,getUserById,getUserSessions,postUsers,postUserUpdate,postUserStatus,postUserResetPassword,postUserDelete,postUserSessionsRevokeAll,postUserSessionRevoke --generate types --package useropenapi -o zz_generated.management.go ../../../../../openapi/openapi.yaml
