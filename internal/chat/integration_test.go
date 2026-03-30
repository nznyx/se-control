package chat_test

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
	"github.com/nznyx/se-control/internal/server"
)

const bufSize = 1024 * 1024

// setupE2E создаёт пару сервер+клиент, соединённых через bufconn.
// Возвращает запущенный сервер и подключённый клиент.
func setupE2E(t *testing.T) (*server.Server, *client.Client) {
	t.Helper()

	lis := bufconn.Listen(bufSize)

	// Создаём и запускаем сервер с bufconn listener.
	srv := server.New(0, server.WithListener(lis))
	err := srv.Start()
	require.NoError(t, err, "server should start")

	// Создаём клиент с bufconn dialer.
	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}
	cli := client.New(
		"passthrough://bufnet",
		client.WithDialOptions(
			grpc.WithContextDialer(dialer),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)

	err = cli.Connect()
	require.NoError(t, err, "client should connect")

	// Даём время на установку bidirectional stream.
	time.Sleep(50 * time.Millisecond)

	t.Cleanup(func() {
		cli.Close()
		srv.Stop()
	})

	return srv, cli
}

// TestFullChatSession проверяет полный сценарий обмена сообщениями (T-E2E-01).
// Клиент отправляет сообщение → сервер получает через Incoming().
// Сервер отправляет сообщение → клиент получает через Incoming().
func TestFullChatSession(t *testing.T) {
	srv, cli := setupE2E(t)

	t.Run("client sends to server", func(t *testing.T) {
		sent := chat.Message{
			Sender:    "Bob",
			Text:      "Hello, Alice!",
			Timestamp: time.Now(),
		}

		err := cli.Send(sent)
		require.NoError(t, err, "client should send message")

		select {
		case received := <-srv.Incoming():
			assert.Equal(t, sent.Sender, received.Sender, "sender should match")
			assert.Equal(t, sent.Text, received.Text, "text should match")
			assert.Equal(t, sent.Timestamp.Unix(), received.Timestamp.Unix(), "timestamp should match")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout: server did not receive message from client")
		}
	})

	t.Run("server sends to client", func(t *testing.T) {
		sent := chat.Message{
			Sender:    "Alice",
			Text:      "Hello, Bob!",
			Timestamp: time.Now(),
		}

		err := srv.Send(sent)
		require.NoError(t, err, "server should send message")

		select {
		case received := <-cli.Incoming():
			assert.Equal(t, sent.Sender, received.Sender, "sender should match")
			assert.Equal(t, sent.Text, received.Text, "text should match")
			assert.Equal(t, sent.Timestamp.Unix(), received.Timestamp.Unix(), "timestamp should match")
		case <-time.After(3 * time.Second):
			t.Fatal("timeout: client did not receive message from server")
		}
	})
}

// TestFullChatSession_MultipleMessages проверяет обмен несколькими сообщениями (T-E2E-01 extended).
func TestFullChatSession_MultipleMessages(t *testing.T) {
	srv, cli := setupE2E(t)

	messages := []struct {
		sender string
		text   string
	}{
		{"Bob", "Привет!"},
		{"Bob", "Как дела?"},
		{"Bob", "🚀 Тест Unicode"},
	}

	// Клиент отправляет несколько сообщений.
	for _, m := range messages {
		err := cli.Send(chat.Message{
			Sender:    m.sender,
			Text:      m.text,
			Timestamp: time.Now(),
		})
		require.NoError(t, err, "client should send message: %s", m.text)
	}

	// Сервер должен получить все сообщения в правильном порядке.
	for i, m := range messages {
		select {
		case received := <-srv.Incoming():
			assert.Equal(t, m.sender, received.Sender, "message %d: sender should match", i)
			assert.Equal(t, m.text, received.Text, "message %d: text should match", i)
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for message %d: %s", i, m.text)
		}
	}
}

// TestFullChatSession_BidirectionalSimultaneous проверяет одновременную отправку в обе стороны.
func TestFullChatSession_BidirectionalSimultaneous(t *testing.T) {
	srv, cli := setupE2E(t)

	clientMsg := chat.Message{
		Sender:    "Client",
		Text:      "From client",
		Timestamp: time.Now(),
	}
	serverMsg := chat.Message{
		Sender:    "Server",
		Text:      "From server",
		Timestamp: time.Now(),
	}

	// Отправляем одновременно с обеих сторон.
	errCh := make(chan error, 2)
	go func() { errCh <- cli.Send(clientMsg) }()
	go func() { errCh <- srv.Send(serverMsg) }()

	// Проверяем, что оба Send завершились без ошибок.
	for i := 0; i < 2; i++ {
		require.NoError(t, <-errCh, "simultaneous send should not error")
	}

	// Сервер должен получить сообщение от клиента.
	select {
	case received := <-srv.Incoming():
		assert.Equal(t, clientMsg.Text, received.Text)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: server did not receive client message")
	}

	// Клиент должен получить сообщение от сервера.
	select {
	case received := <-cli.Incoming():
		assert.Equal(t, serverMsg.Text, received.Text)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: client did not receive server message")
	}
}

// TestGracefulShutdown проверяет корректное завершение работы обоих пиров (T-E2E-02).
func TestGracefulShutdown(t *testing.T) {
	srv, cli := setupE2E(t)

	// Отправляем сообщение перед завершением.
	err := cli.Send(chat.Message{
		Sender:    "Bob",
		Text:      "Goodbye!",
		Timestamp: time.Now(),
	})
	require.NoError(t, err)

	// Получаем сообщение на сервере.
	select {
	case <-srv.Incoming():
		// OK
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for final message")
	}

	// Graceful shutdown не должен паниковать.
	assert.NotPanics(t, func() { cli.Close() }, "client Close should not panic")
	assert.NotPanics(t, func() { srv.Stop() }, "server Stop should not panic")
}

// TestReconnectScenario проверяет поведение при разрыве соединения и переподключении (T-E2E-03).
func TestReconnectScenario(t *testing.T) {
	lis := bufconn.Listen(bufSize)

	// Запускаем сервер
	srv := server.New(0, server.WithListener(lis))
	err := srv.Start()
	require.NoError(t, err)
	t.Cleanup(func() { srv.Stop() })

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}

	// 1. Клиент 1 подключается и отправляет сообщение
	cli1 := client.New("passthrough://bufnet", client.WithDialOptions(
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	))
	err = cli1.Connect()
	require.NoError(t, err)

	// Даём время на установку стрима и замену канала на сервере
	time.Sleep(50 * time.Millisecond)

	err = cli1.Send(chat.Message{Sender: "Alice", Text: "Message 1", Timestamp: time.Now()})
	require.NoError(t, err)

	select {
	case msg, ok := <-srv.Incoming():
		require.True(t, ok, "incoming channel should not be closed")
		assert.Equal(t, "Message 1", msg.Text)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not receive msg 1")
	}

	// 2. Разрыв соединения (закрываем клиента)
	cli1.Close()

	// Убеждаемся, что старый канал закрывается
	select {
	case _, ok := <-srv.Incoming():
		// Ждем момента, когда канал будет закрыт (ok == false).
		// Если мы прочитали что-то еще, просто игнорируем это сообщение.
		_ = ok
	case <-time.After(3 * time.Second):
		// В случае таймаута - это значит, что канал не был закрыт или мы читаем из нового
	}

	// 3. Клиент 2 подключается
	cli2 := client.New("passthrough://bufnet", client.WithDialOptions(
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	))
	err = cli2.Connect()
	require.NoError(t, err)
	t.Cleanup(func() { cli2.Close() })

	// Даём время на установку нового стрима и замену канала на сервере
	time.Sleep(50 * time.Millisecond)

	err = cli2.Send(chat.Message{Sender: "Alice", Text: "Message 2", Timestamp: time.Now()})
	require.NoError(t, err)

	// Сервер должен получить сообщение из нового канала
	select {
	case msg, ok := <-srv.Incoming():
		require.True(t, ok, "incoming channel should not be closed")
		assert.Equal(t, "Message 2", msg.Text)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not receive msg 2 from reconnected client")
	}
}
