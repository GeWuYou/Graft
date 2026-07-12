package savedview

import (
	"errors"

	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

// Module registers the generic saved-view service. It intentionally owns no menu or HTTP routes.
type Module struct{ service *Service }

// NewModule constructs a saved-view module with the provided service.
func NewModule(service *Service) *Module { return &Module{service: service} }

// Register exposes only the consumer-neutral service boundary.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil || ctx == nil || ctx.Services == nil {
		return errors.New("saved view module service is unavailable")
	}
	return ctx.Services.RegisterSingleton((*moduleapi.SavedViewService)(nil), func(containerdi.Resolver) (any, error) {
		return m.service, nil
	})
}

// Boot has no background work.
func (*Module) Boot(*module.Context) error { return nil }

// Shutdown has no persistent resources beyond the platform database pool.
func (*Module) Shutdown(*module.Context) error { return nil }
