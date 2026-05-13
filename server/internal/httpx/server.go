package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Server wraps the Gin engine used by the runtime shell.
type Server struct {
	engine *gin.Engine
	mu     sync.Mutex
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
	srv := &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}
	if err := s.bindRunningServer(srv); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err, ok := <-errCh:
		s.clearRunningServer(srv)
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
	server := s.detachRunningServer()
	if server == nil {
		return nil
	}

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}

func (s *Server) bindRunningServer(server *http.Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize lifecycle transitions so Run and Shutdown share one explicit
	// running server pointer instead of racing on partially applied state.
	if s.server != nil {
		return errors.New("http server already running")
	}

	s.server = server
	return nil
}

func (s *Server) detachRunningServer() *http.Server {
	s.mu.Lock()
	defer s.mu.Unlock()

	server := s.server
	s.server = nil
	return server
}

func (s *Server) clearRunningServer(server *http.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == server {
		s.server = nil
	}
}
