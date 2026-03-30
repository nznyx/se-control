package chat

import (
	"errors"
	"time"

	"github.com/nznyx/se-control/internal/app"
	"github.com/nznyx/se-control/internal/client"
	"github.com/nznyx/se-control/internal/server"
)

// Service — сервис чата, координирующий отправку и получение сообщений.
type Service struct {
	username string
	messages chan Message
	server   *server.Server
	client   *client.Client
}

// NewService создаёт новый экземпляр Service.
func NewService(username string) *Service {
	return &Service{
		username: username,
		messages: make(chan Message, 100),
	}
}

// Start инициализирует и запускает сервис в зависимости от конфигурации.
// Если config.IsServer() — запускает gRPC-сервер, иначе — подключается как клиент.
func (s *Service) Start(config app.Config) error {
	if config.IsServer() {
		s.server = server.New(config.Port)
		return s.server.Start()
	}

	s.client = client.New(config.PeerAddress)
	return s.client.Connect()
}

// Send отправляет текстовое сообщение от имени текущего пользователя.
func (s *Service) Send(text string) error {
	if text == "" {
		return errors.New("cannot send empty message")
	}

	_ = Message{
		Sender:    s.username,
		Text:      text,
		Timestamp: time.Now(),
	}

	// TODO: реализовать интеграцию с s.server и s.client после их готовности
	return nil
}

// Incoming возвращает канал входящих сообщений.
func (s *Service) Incoming() <-chan Message {
	return s.messages
}

// Stop корректно завершает работу сервиса.
func (s *Service) Stop() {
	if s.server != nil {
		s.server.Stop()
	}

	if s.client != nil {
		s.client.Close()
	}
}
