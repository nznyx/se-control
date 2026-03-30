package chat

import (
	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// Service — сервис чата, координирующий отправку и получение сообщений.
type Service struct {
	username string
	messages chan *pb.ChatMessage
}

// NewService создаёт новый экземпляр Service.
func NewService(username string) *Service {
	return &Service{
		username: username,
		messages: make(chan *pb.ChatMessage, 100),
	}
}

// Send отправляет текстовое сообщение от имени текущего пользователя.
func (s *Service) Send(_ string) error {
	// TODO: реализовать отправку сообщения через gRPC stream.
	return nil
}

// Incoming возвращает канал входящих сообщений.
func (s *Service) Incoming() <-chan *pb.ChatMessage {
	return s.messages
}

// Stop корректно завершает работу сервиса.
func (s *Service) Stop() {
	// TODO: реализовать остановку сервиса.
}
