package chat

import (
	"errors"
	"time"
)

// Service — сервис чата, координирующий отправку и получение сообщений.
// Зависит от абстракции Peer, а не от конкретных реализаций gRPC сервера/клиента.
type Service struct {
	username string
	messages chan Message
	// peer — активное соединение (сервер или клиент), инжектируется через SetPeer.
	peer Peer
}

// NewService создаёт новый экземпляр Service.
func NewService(username string) *Service {
	return &Service{
		username: username,
		messages: make(chan Message, 100),
	}
}

// SetPeer инжектирует транспортный уровень (server или client).
// Вызывается из App после создания и запуска соответствующего компонента.
func (s *Service) SetPeer(peer Peer) {
	s.peer = peer
	go s.forwardIncoming()
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

	return s.peer.Send(msg)
}

// Incoming возвращает канал входящих сообщений.
func (s *Service) Incoming() <-chan Message {
	return s.messages
}

// Stop корректно завершает работу сервиса.
func (s *Service) Stop() {
	if s.peer != nil {
		s.peer.Close()
	}
}

// forwardIncoming читает входящие доменные сообщения от peer-а
// и записывает их в канал messages.
// Внешний цикл позволяет переключиться на новый канал (например, когда
// сервер заменяет начальный пустой канал при первом подключении клиента).
func (s *Service) forwardIncoming() {
	defer close(s.messages)
	var current <-chan Message
	for {
		newCh := s.peer.Incoming()
		if newCh == nil || newCh == current {
			break
		}
		current = newCh
		for msg := range current {
			s.messages <- msg
		}
	}
}
