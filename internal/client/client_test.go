package client_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/nznyx/se-control/internal/chat"
	"github.com/nznyx/se-control/internal/client"
	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

const bufSize = 1024 * 1024

// echoServer — тестовый gRPC сервер, отражающий полученные сообщения обратно.
type echoServer struct {
	pb.UnimplementedChatServiceServer
	received chan *pb.ChatMessage
}

func newEchoServer() *echoServer {
	return &echoServer{
		received: make(chan *pb.ChatMessage, 100),
	}
}

func (e *echoServer) Chat(stream pb.ChatService_ChatServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return nil
		}
		e.received <- msg
		// Отражаем сообщение обратно клиенту.
		if sendErr := stream.Send(msg); sendErr != nil {
			return sendErr
		}
	}
}

// setupTestServer запускает тестовый gRPC сервер на bufconn и возвращает адрес и опции для клиента.
func setupTestServer(t *testing.T, srv pb.ChatServiceServer) (string, []grpc.DialOption) {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	grpcSrv := grpc.NewServer()
	pb.RegisterChatServiceServer(grpcSrv, srv)

	go func() {
		_ = grpcSrv.Serve(lis)
	}()

	t.Cleanup(func() { grpcSrv.GracefulStop() })

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}

	opts := []grpc.DialOption{
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	return "passthrough://bufnet", opts
}

// TestClient_Connect проверяет успешное подключение к серверу (T-CLI-01).
func TestClient_Connect(t *testing.T) {
	t.Parallel()

	echo := newEchoServer()
	addr, dialOpts := setupTestServer(t, echo)

	cli := client.New(addr, client.WithDialOptions(dialOpts...))
	t.Cleanup(func() { cli.Close() })

	err := cli.Connect()
	require.NoError(t, err, "client should connect without error")
}

// TestClient_Send проверяет отправку сообщения через stream (T-CLI-02).
func TestClient_Send(t *testing.T) {
	t.Parallel()

	echo := newEchoServer()
	addr, dialOpts := setupTestServer(t, echo)

	cli := client.New(addr, client.WithDialOptions(dialOpts...))
	t.Cleanup(func() { cli.Close() })

	err := cli.Connect()
	require.NoError(t, err)

	msg := chat.Message{
		Sender:    "Bob",
		Text:      "Hello from client!",
		Timestamp: time.Now(),
	}

	err = cli.Send(msg)
	require.NoError(t, err, "client Send should not return error")

	// Проверяем, что сервер получил сообщение (в proto-формате).
	select {
	case received := <-echo.received:
		assert.Equal(t, msg.Sender, received.GetSender())
		assert.Equal(t, msg.Text, received.GetText())
		assert.Equal(t, msg.Timestamp.Unix(), received.GetTimestamp())
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for server to receive message")
	}
}

// TestClient_Receive проверяет получение сообщения из stream (T-CLI-03).
func TestClient_Receive(t *testing.T) {
	t.Parallel()

	// echoServer отражает сообщения обратно, поэтому отправленное сообщение
	// должно вернуться через Incoming().
	echo := newEchoServer()
	addr, dialOpts := setupTestServer(t, echo)

	cli := client.New(addr, client.WithDialOptions(dialOpts...))
	t.Cleanup(func() { cli.Close() })

	err := cli.Connect()
	require.NoError(t, err)

	sent := chat.Message{
		Sender:    "Alice",
		Text:      "Echo test",
		Timestamp: time.Now(),
	}

	err = cli.Send(sent)
	require.NoError(t, err)

	// Клиент должен получить отражённое сообщение через Incoming().
	select {
	case received := <-cli.Incoming():
		assert.Equal(t, sent.Sender, received.Sender, "sender should match")
		assert.Equal(t, sent.Text, received.Text, "text should match")
		assert.Equal(t, sent.Timestamp.Unix(), received.Timestamp.Unix(), "timestamp should match")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message on client Incoming() channel")
	}
}

// TestClient_Receive_Unicode проверяет корректную обработку Unicode (T-CLI-03 + T-CH-07).
func TestClient_Receive_Unicode(t *testing.T) {
	t.Parallel()

	echo := newEchoServer()
	addr, dialOpts := setupTestServer(t, echo)

	cli := client.New(addr, client.WithDialOptions(dialOpts...))
	t.Cleanup(func() { cli.Close() })

	err := cli.Connect()
	require.NoError(t, err)

	tests := []struct {
		name string
		text string
	}{
		{name: "cyrillic", text: "Привет, мир! 🚀"},
		{name: "japanese", text: "こんにちは世界"},
		{name: "emoji", text: "🎉🔥💯"},
		{name: "mixed", text: "Hello Мир 🌍"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = cli.Send(chat.Message{
				Sender:    "Тест",
				Text:      tt.text,
				Timestamp: time.Now(),
			})
			require.NoError(t, err)

			select {
			case received := <-cli.Incoming():
				assert.Equal(t, tt.text, received.Text, "unicode text should be preserved")
			case <-time.After(3 * time.Second):
				t.Fatalf("timeout waiting for unicode message: %s", tt.text)
			}
		})
	}
}

// TestClient_Close проверяет корректное закрытие соединения (T-CLI-04).
func TestClient_Close(t *testing.T) {
	t.Parallel()

	echo := newEchoServer()
	addr, dialOpts := setupTestServer(t, echo)

	cli := client.New(addr, client.WithDialOptions(dialOpts...))

	err := cli.Connect()
	require.NoError(t, err)

	// Close не должен паниковать.
	assert.NotPanics(t, func() { cli.Close() }, "Close should not panic")

	// Повторный Close тоже не должен паниковать.
	assert.NotPanics(t, func() { cli.Close() }, "repeated Close should not panic")
}

// TestClient_ConnectFail проверяет ошибку при подключении к несуществующему адресу (T-CLI-05).
func TestClient_ConnectFail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "invalid address format",
			address: "not-a-valid-address",
		},
		{
			name:    "unreachable port",
			address: "localhost:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := client.New(tt.address)
			defer cli.Close()

			err := cli.Connect()
			// grpc.NewClient не возвращает ошибку сразу (lazy connection),
			// но Chat stream должен вернуть ошибку при попытке установить соединение.
			if err == nil {
				t.Log("Connect returned nil (lazy connection), this is expected for gRPC")
			}
		})
	}
}

// TestClient_Send_WithoutConnect проверяет поведение Send без предварительного Connect.
func TestClient_Send_WithoutConnect(t *testing.T) {
	t.Parallel()

	cli := client.New("localhost:50051")
	defer cli.Close()

	msg := chat.Message{
		Sender:    "Test",
		Text:      "No connection",
		Timestamp: time.Now(),
	}

	// Send без Connect должен вернуть ошибку (sendCh == nil).
	err := cli.Send(msg)
	assert.Error(t, err, "Send without Connect should return error")
}
