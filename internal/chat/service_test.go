package chat

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// mockPeer реализует интерфейс protoSender для тестов.
// Позволяет проверять отправленные сообщения и симулировать ошибки.
type mockPeer struct {
	sent     []*pb.ChatMessage
	err      error
	incoming chan *pb.ChatMessage
}

func newMockPeer() *mockPeer {
	return &mockPeer{
		incoming: make(chan *pb.ChatMessage, 100),
	}
}

func (m *mockPeer) Send(msg *pb.ChatMessage) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockPeer) Incoming() <-chan *pb.ChatMessage {
	return m.incoming
}

// injectPeer подставляет mock-peer в сервис через поле peer.
// Используется только в тестах (white-box testing внутри пакета).
func injectPeer(s *Service, peer protoSender) {
	s.peer = peer
}

// TestNewMessage проверяет создание сообщения с корректными полями (T-CH-01).
func TestNewMessage(t *testing.T) {
	t.Parallel()

	msg := Message{
		Sender:    "Alice",
		Text:      "Hello",
		Timestamp: time.Now(),
	}

	assert.Equal(t, "Alice", msg.Sender, "sender should be set")
	assert.Equal(t, "Hello", msg.Text, "text should be set")
	assert.False(t, msg.Timestamp.IsZero(), "timestamp should not be zero")
}

// TestMessageTimestamp проверяет, что timestamp устанавливается автоматически (T-CH-02).
func TestMessageTimestamp(t *testing.T) {
	t.Parallel()

	service := NewService("Alice")
	mock := newMockPeer()
	injectPeer(service, mock)

	before := time.Now()
	err := service.Send("Hello")
	after := time.Now()

	require.NoError(t, err)
	require.Len(t, mock.sent, 1, "should have sent one message")

	ts := time.Unix(mock.sent[0].GetTimestamp(), 0)
	assert.True(t, !ts.Before(before.Truncate(time.Second)) && !ts.After(after.Add(time.Second)),
		"timestamp should be set at send time, got: %v", ts)
}

// TestService_Send_WithMock проверяет отправку сообщения через mock sender (T-CH-03).
func TestService_Send_WithMock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		text      string
		mockErr   error
		wantError bool
		wantSent  bool
	}{
		{
			name:      "T-CH-06: empty message returns error",
			text:      "",
			wantError: true,
			wantSent:  false,
		},
		{
			name:      "T-CH-01, T-CH-03: valid message is sent",
			text:      "Hello world",
			wantError: false,
			wantSent:  true,
		},
		{
			name:      "T-CH-07: unicode message is sent",
			text:      "Привет, мир! 🚀",
			wantError: false,
			wantSent:  true,
		},
		{
			name:      "mock sender returns error",
			text:      "Hello",
			mockErr:   errors.New("send failed"),
			wantError: true,
			wantSent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewService("Alice")
			mock := newMockPeer()
			mock.err = tt.mockErr
			injectPeer(service, mock)

			err := service.Send(tt.text)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.wantSent {
				require.Len(t, mock.sent, 1, "should have sent exactly one message")
				assert.Equal(t, "Alice", mock.sent[0].GetSender(), "sender should be username")
				assert.Equal(t, tt.text, mock.sent[0].GetText(), "text should match")
				assert.NotZero(t, mock.sent[0].GetTimestamp(), "timestamp should be set")
			} else {
				assert.Empty(t, mock.sent, "should not have sent any message")
			}
		})
	}
}

// TestService_Receive проверяет получение сообщения из канала (T-CH-04).
func TestService_Receive(t *testing.T) {
	t.Parallel()

	service := NewService("Bob")

	// Напрямую пишем в канал messages (white-box test внутри пакета).
	expected := Message{
		Sender:    "Alice",
		Text:      "Hello, Bob!",
		Timestamp: time.Now(),
	}

	go func() {
		service.messages <- expected
	}()

	select {
	case received := <-service.Incoming():
		assert.Equal(t, expected.Sender, received.Sender, "sender should match")
		assert.Equal(t, expected.Text, received.Text, "text should match")
		assert.Equal(t, expected.Timestamp.Unix(), received.Timestamp.Unix(), "timestamp should match")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message on Incoming() channel")
	}
}

// TestService_Stop проверяет корректное завершение сервиса (T-CH-05).
func TestService_Stop(t *testing.T) {
	t.Parallel()

	service := NewService("Alice")

	// Stop без запуска не должен паниковать.
	assert.NotPanics(t, func() { service.Stop() }, "Stop without Start should not panic")
}

// TestService_Send_NotConnected проверяет ошибку при отправке без подключения.
func TestService_Send_NotConnected(t *testing.T) {
	t.Parallel()

	service := NewService("Alice")

	err := service.Send("Hello")
	assert.Error(t, err, "Send without connection should return error")
	assert.EqualError(t, err, "not connected to peer")
}

// TestMessageToProto проверяет конвертацию доменного сообщения в protobuf.
func TestMessageToProto(t *testing.T) {
	t.Parallel()

	ts := time.Unix(1711800000, 0)
	msg := Message{
		Sender:    "Alice",
		Text:      "Hello",
		Timestamp: ts,
	}

	proto := msg.ToProto()

	assert.Equal(t, "Alice", proto.GetSender())
	assert.Equal(t, "Hello", proto.GetText())
	assert.Equal(t, ts.Unix(), proto.GetTimestamp())
}

// TestMessageFromProto проверяет конвертацию protobuf-сообщения в доменную структуру.
func TestMessageFromProto(t *testing.T) {
	t.Parallel()

	pbMsg := &pb.ChatMessage{
		Sender:    "Bob",
		Text:      "Привет! 🎉",
		Timestamp: 1711800000,
	}

	msg := MessageFromProto(pbMsg)

	assert.Equal(t, "Bob", msg.Sender)
	assert.Equal(t, "Привет! 🎉", msg.Text)
	assert.Equal(t, int64(1711800000), msg.Timestamp.Unix())
}
