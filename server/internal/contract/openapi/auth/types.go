package authopenapi

// ServerInterface is the minimal generated handler contract for guarded auth login/bootstrap migration.
type ServerInterface interface {
	PostAuthLogin(params PostAuthLoginParams, body PostAuthLoginJSONRequestBody)
	GetAuthBootstrap(params GetAuthBootstrapParams)
}
