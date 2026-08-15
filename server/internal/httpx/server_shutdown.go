package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"graft/server/internal/logger/logsafe"
)

func shutdownHTTPServer(ctx context.Context, server *http.Server, logger *zap.Logger, name string) error {
	err := server.Shutdown(ctx)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		logsafe.Warn(logger, name+" graceful shutdown timed out; forcing close", zap.Error(err))
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			return errors.Join(err, fmt.Errorf("force close %s: %w", name, closeErr))
		}
	}
	return err
}
