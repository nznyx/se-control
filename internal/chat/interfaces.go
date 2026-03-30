// Package chat — бизнес-логика чата.
// Содержит интерфейсы и сервис для управления обменом сообщениями.
package chat

// MessageSender — интерфейс для отправки доменных сообщений.
type MessageSender interface {
	Send(msg Message) error
}

// MessageReceiver — интерфейс для получения доменных сообщений.
type MessageReceiver interface {
	Incoming() <-chan Message
}

// Peer — объединённый интерфейс транспортного уровня.
// Реализуется как *server.Server, так и *client.Client.
// Позволяет ChatService работать через абстракцию, не завися от конкретных реализаций.
type Peer interface {
	MessageSender
	MessageReceiver
	// Close корректно завершает соединение.
	Close()
}
