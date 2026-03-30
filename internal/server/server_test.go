package server_test

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
	"github.com/nznyx/se-control/internal/server"
	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

const bufSize = 1024 * 1024

// newBufconnServer создаёт Server с in-memory listener для тестов.
func newBufconnServer(t *testing.T) (*server.Server, *bufconn.Listener) {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := server.New(0, server.WithListener(lis))
	return srv, lis
}

// newBufconnClient создаёт gRPC-клиент, подключённый через bufconn.
func newBufconnClient(t *testing.T, lis *bufconn.Listener) pb.ChatServiceClient {
	t.Helper()
	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewChatServiceClient(conn)
}

// TestServer_Start проверяет, что сервер успешно запускается (T-SRV-01).
func TestServer_Start(t *testing.T) {
	t.Parallel()

	srv, lis := newBufconnServer(t)
	t.Cleanup(func() { srv.Stop() })

	err := srv.Start()
	require.NoError(t, err, "server should start without error")

	// Проверяем, что listener принимает соединения.
	assert.NotNil(t, lis, "listener should be active after Start")
}

// TestServer_Start_PortBusy проверяет ошибку при занятом порту (T-SRV-01 edge case).
func TestServer_Start_PortBusy(t *testing.T) {
	t.Parallel()

	// Занимаем порт.
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { _ = lis.Close() }()

	port := lis.Addr().(*net.TCPAddr).Port

	// Пытаемся запустить сервер на том же порту.
	srv := server.New(port)
	err = srv.Start()
	assert.Error(t, err, "should fail when port is already in use")
}

// TestServer_Stop проверяет корректный graceful shutdown (T-SRV-02).
func TestServer_Stop(t *testing.T) {
	t.Parallel()

	srv, _ := newBufconnServer(t)

	err := srv.Start()
	require.NoError(t, err)

	// Stop не должен паниковать.
	assert.NotPanics(t, func() { srv.Stop() }, "Stop should not panic")

	// Повторный Stop тоже не должен паниковать.
	assert.NotPanics(t, func() { srv.Stop() }, "repeated Stop should not panic")
}

// TestServer_Stop_DeadlockWhenClientConnected проверяет отсутствие deadlock при Stop()
// когда клиент подключён.
func TestServer_Stop_DeadlockWhenClientConnected(t *testing.T) {
	t.Parallel()

	srv, lis := newBufconnServer(t)
	err := srv.Start()
	require.NoError(t, err)

	// Подключаем клиента, но НЕ отключаем его.
	grpcClient := newBufconnClient(t, lis)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = grpcClient.Chat(ctx)
	require.NoError(t, err)

	// Даём время на установку bidirectional stream.
	time.Sleep(100 * time.Millisecond)

	// Stop() должен завершиться без deadlock.
	done := make(chan struct{})
	go func() {
		srv.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Успех: Stop() завершился без deadlock.
	case <-time.After(2 * time.Second):
		t.Fatal("DEADLOCK: srv.Stop() завис. GracefulStop() ждёт завершения клиента, а cancel() вызывается после.")
	}
}

// TestServer_Stop_ClosesIncomingChannel проверяет, что Stop() закрывает канал Incoming(),
// чтобы forwardIncoming() в ChatService мог корректно завершиться.
func TestServer_Stop_ClosesIncomingChannel(t *testing.T) {
	t.Parallel()

	srv, _ := newBufconnServer(t)
	err := srv.Start()
	require.NoError(t, err)

	incoming := srv.Incoming()

	srv.Stop()

	// Канал должен быть закрыт после Stop().
	select {
	case _, ok := <-incoming:
		if ok {
			t.Fatal("ожидалось, что канал будет закрыт, но получено значение")
		}
		// Успех: канал закрыт.
	case <-time.After(time.Second):
		t.Fatal("GOROUTINE LEAK: канал Incoming() не был закрыт после Stop()")
	}
}

// TestServer_ChatStream проверяет bidirectional streaming через bufconn (T-SRV-03).
// Клиент отправляет сообщение — сервер получает его через Incoming().
func TestServer_ChatStream(t *testing.T) {
	t.Parallel()

	srv, lis := newBufconnServer(t)
	t.Cleanup(func() { srv.Stop() })

	err := srv.Start()
	require.NoError(t, err)

	grpcClient := newBufconnClient(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := grpcClient.Chat(ctx)
	require.NoError(t, err, "should establish chat stream")

	// Клиент отправляет сообщение.
	sent := &pb.ChatMessage{
		Sender:    "Alice",
		Text:      "Hello, server!",
		Timestamp: time.Now().Unix(),
	}
	err = stream.Send(sent)
	require.NoError(t, err, "client should send message without error")

	// Сервер должен получить сообщение через Incoming() как доменный тип.
	select {
	case received := <-srv.Incoming():
		assert.Equal(t, sent.GetSender(), received.Sender, "sender should match")
		assert.Equal(t, sent.GetText(), received.Text, "text should match")
		assert.Equal(t, sent.GetTimestamp(), received.Timestamp.Unix(), "timestamp should match")
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message on server Incoming() channel")
	}
}

// TestServer_ChatStream_Unicode проверяет корректную обработку Unicode (T-SRV-03 + T-CH-07).
func TestServer_ChatStream_Unicode(t *testing.T) {
	t.Parallel()

	srv, lis := newBufconnServer(t)
	t.Cleanup(func() { srv.Stop() })

	err := srv.Start()
	require.NoError(t, err)

	grpcClient := newBufconnClient(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := grpcClient.Chat(ctx)
	require.NoError(t, err)

	unicodeTexts := []string{
		"Привет, мир!",
		"こんにちは",
		"🚀🎉✨",
		"Héllo Wörld",
	}

	for _, text := range unicodeTexts {
		err = stream.Send(&pb.ChatMessage{
			Sender:    "Алексей",
			Text:      text,
			Timestamp: time.Now().Unix(),
		})
		require.NoError(t, err)

		select {
		case received := <-srv.Incoming():
			assert.Equal(t, text, received.Text, "unicode text should be preserved: %s", text)
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for unicode message: %s", text)
		}
	}
}

// TestServer_Send проверяет отправку сообщения от сервера клиенту.
func TestServer_Send(t *testing.T) {
	t.Parallel()

	srv, lis := newBufconnServer(t)
	t.Cleanup(func() { srv.Stop() })

	err := srv.Start()
	require.NoError(t, err)

	grpcClient := newBufconnClient(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := grpcClient.Chat(ctx)
	require.NoError(t, err)

	// Даём серверу время установить stream.
	time.Sleep(50 * time.Millisecond)

	// Сервер отправляет доменное сообщение.
	toSend := chat.Message{
		Sender:    "Server",
		Text:      "Hello from server!",
		Timestamp: time.Now(),
	}
	err = srv.Send(toSend)
	require.NoError(t, err, "server Send should not return error")

	// Клиент должен получить сообщение.
	received, err := stream.Recv()
	require.NoError(t, err, "client should receive message")
	assert.Equal(t, toSend.Sender, received.GetSender())
	assert.Equal(t, toSend.Text, received.GetText())
}

// TestServer_ClientDisconnect проверяет корректную обработку отключения клиента (T-SRV-04).
func TestServer_ClientDisconnect(t *testing.T) {
	t.Parallel()

	srv, lis := newBufconnServer(t)
	t.Cleanup(func() { srv.Stop() })

	err := srv.Start()
	require.NoError(t, err)

	grpcClient := newBufconnClient(t, lis)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	stream, err := grpcClient.Chat(ctx)
	require.NoError(t, err)

	// Клиент закрывает соединение.
	cancel()
	_ = stream.CloseSend()

	// Сервер должен корректно обработать отключение без паники.
	time.Sleep(100 * time.Millisecond)

	assert.NotPanics(t, func() { srv.Stop() }, "server should handle client disconnect gracefully")
}
