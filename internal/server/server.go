// Package server — gRPC сервер для обработки P2P чат-соединений.
package server

// Server — gRPC сервер, обрабатывающий bidirectional streaming.
type Server struct {
	port int
}

// New создаёт новый экземпляр Server.
func New(port int) *Server {
	return &Server{
		port: port,
	}
}

// Start запускает gRPC сервер на указанном порту.
func (s *Server) Start() error {
	// TODO: реализовать запуск gRPC сервера.
	return nil
}

// Stop корректно останавливает сервер.
func (s *Server) Stop() {
	// TODO: реализовать graceful shutdown gRPC сервера.
}
