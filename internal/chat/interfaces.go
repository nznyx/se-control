// Package chat — бизнес-логика чата.
// Содержит интерфейсы и сервис для управления обменом сообщениями.
package chat

// MessageSender — интерфейс для отправки сообщений.
type MessageSender interface {
	Send(msg Message) error
}

// MessageReceiver — интерфейс для получения сообщений.
type MessageReceiver interface {
	Incoming() <-chan Message
}
