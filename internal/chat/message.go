package chat

import (
	"time"

	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// Message — доменная структура сообщения чата.
// Используется для внутренней коммуникации между компонентами.
type Message struct {
	Sender    string
	Text      string
	Timestamp time.Time
}

// ToProto конвертирует доменное сообщение в protobuf-представление.
func (m Message) ToProto() *pb.ChatMessage {
	return &pb.ChatMessage{
		Sender:    m.Sender,
		Text:      m.Text,
		Timestamp: m.Timestamp.Unix(),
	}
}

// MessageFromProto конвертирует protobuf-сообщение в доменную структуру.
func MessageFromProto(msg *pb.ChatMessage) Message {
	return Message{
		Sender:    msg.GetSender(),
		Text:      msg.GetText(),
		Timestamp: time.Unix(msg.GetTimestamp(), 0),
	}
}
