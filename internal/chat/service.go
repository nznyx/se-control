package chat

import (
	"errors"
	"time"

	"github.com/nznyx/se-control/internal/app"
	"github.com/nznyx/se-control/internal/client"
	"github.com/nznyx/se-control/internal/server"
	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// protoSender — интерфейс для отправки protobuf-сообщений.
// Реализуется как *server.Server, так и *client.Client.
type protoSender interface {
	Send(msg *pb.ChatMessage) error
	Incoming() <-chan *pb.ChatMessage
}

// Service — сервис чата, координирующий отправку и получение сообщений.
type Service struct {
	username string
	messages chan Message
	// peer — активное соединение (сервер или клиент).
	peer protoSender
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
// После успешного запуска начинает пересылку входящих сообщений в канал messages.
func (s *Service) Start(config app.Config) error {
	if config.IsServer() {
		srv := server.New(config.Port)
		if err := srv.Start(); err != nil {
			return err
		}
		s.peer = srv
	} else {
		cli := client.New(config.PeerAddress)
		if err := cli.Connect(); err != nil {
			return err
		}
		s.peer = cli
	}

	go s.forwardIncoming()
	return nil
}

// Send отправляет текстовое сообщение от имени текущего пользователя.
func (s *Service) Send(text string) error {
	if text == "" {
		return errors.New("cannot send empty message")
	}

	if s.peer == nil {
		return errors.New("not connected to peer")
	}

	msg := Message{
		Sender:    s.username,
		Text:      text,
		Timestamp: time.Now(),
	}

	return s.peer.Send(msg.ToProto())
}

// Incoming возвращает канал входящих сообщений.
func (s *Service) Incoming() <-chan Message {
	return s.messages
}

// Stop корректно завершает работу сервиса.
func (s *Service) Stop() {
	if srv, ok := s.peer.(*server.Server); ok {
		srv.Stop()
	}
	if cli, ok := s.peer.(*client.Client); ok {
		cli.Close()
	}
}

// forwardIncoming читает входящие protobuf-сообщения от peer-а
// и конвертирует их в доменные Message, записывая в канал messages.
func (s *Service) forwardIncoming() {
	for pbMsg := range s.peer.Incoming() {
		s.messages <- MessageFromProto(pbMsg)
	}
}
