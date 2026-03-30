// Package chat — бизнес-логика чата.
// Содержит интерфейсы и сервис для управления обменом сообщениями.
package chat

import (
	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// MessageSender — интерфейс для отправки сообщений.
type MessageSender interface {
	Send(msg *pb.ChatMessage) error
}

// MessageReceiver — интерфейс для получения сообщений.
type MessageReceiver interface {
	Incoming() <-chan *pb.ChatMessage
}
