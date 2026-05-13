package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Server wraps the Gin engine used by the runtime shell.
type Server struct {
	engine *gin.Engine
	server *http.Server
}

// NewServer creates the minimal Gin engine used by the MVP shell.
func NewServer() *Server {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())
	return &Server{engine: engine}
}

// Engine returns the root router used by core and plugins.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Run starts the HTTP server and keeps it bound to the provided lifecycle context.
func (s *Server) Run(ctx context.Context, addr string) error {
	if s.server != nil {
		return errors.New("http server already running")
	}

	s.server = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	errCh := make(chan error, 1)
	go func() {
		err := s.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err, ok := <-errCh:
		s.server = nil
		if !ok {
			return nil
		}
		return fmt.Errorf("listen and serve: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		shutdownErr := s.Shutdown(shutdownCtx)
		<-errCh
		return shutdownErr
	}
}

// Shutdown stops the underlying HTTP server gracefully when it is running.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	server := s.server
	s.server = nil
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}
