package authopenapi

//go:generate go tool oapi-codegen --include-operation-ids postAuthLogin,postAuthRefresh,postAuthLogout,getAuthBootstrap,getAuthSessions,postAuthSessionsRevokeAll,postAuthSessionsRevokeOthers,postAuthSessionRevoke,postAuthChangePassword,postAuthCompleteRequiredPasswordChange,getAuthPersonalAccessTokens,postAuthPersonalAccessTokens,postAuthPersonalAccessTokenRevoke --generate types --package authopenapi -o zz_generated.auth.go ../../../../../openapi/openapi.yaml
