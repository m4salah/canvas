package server

import "canvas/handlers"

func (s *Server) setupRoutes() {
	handlers.Static(s.mux)
	handlers.Health(s.mux)
}
