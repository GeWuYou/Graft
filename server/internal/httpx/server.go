// Package httpx owns the Gin engine assembly for the platform runtime.
package httpx

import "github.com/gin-gonic/gin"

// Server wraps the Gin engine used by the runtime shell.
type Server struct {
	engine *gin.Engine
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

// Run starts the HTTP server on the provided address.
func (s *Server) Run(addr string) error {
	return s.engine.Run(addr)
}
