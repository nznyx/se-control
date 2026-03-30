// Package client — gRPC клиент для подключения к peer-у.
// Поддерживает bidirectional streaming для обмена сообщениями.
package client

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nznyx/se-control/internal/chat"
	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// bufferSize — размер буферизованных каналов сообщений.
const bufferSize = 100

// Client — gRPC клиент для установки соединения и обмена сообщениями.
// Реализует интерфейс chat.Peer: Send(chat.Message), Incoming() <-chan chat.Message, Close().
// Конвертация proto ↔ domain выполняется внутри пакета.
type Client struct {
	address string
	opts    []grpc.DialOption

	conn   *grpc.ClientConn
	stream pb.ChatService_ChatClient

	// incoming — канал входящих доменных сообщений от peer-а.
	// Пересоздаётся при каждом новом Connect().
	incoming chan chat.Message

	// sendCh — канал исходящих доменных сообщений для отправки peer-у.
	// Пересоздаётся при каждом новом Connect().
	sendCh chan chat.Message

	// cancel завершает контекст активного соединения.
	cancel context.CancelFunc
	// mu защищает доступ к полям conn, stream, incoming, sendCh.
	mu sync.Mutex

	// closeOnce гарантирует однократное закрытие канала incoming.
	closeOnce sync.Once
}

// Option — функциональная опция для конфигурации Client.
type Option func(*Client)

// WithDialOptions задаёт дополнительные grpc.DialOption (используется в тестах с bufconn).
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(c *Client) {
		c.opts = append(c.opts, opts...)
	}
}

// New создаёт новый экземпляр Client.
func New(address string, opts ...Option) *Client {
	c := &Client{
		address: address,
		opts:    []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Connect устанавливает соединение с peer-ом и запускает bidirectional stream.
// Создаёт свежие каналы для каждого нового соединения.
// Возвращает ошибку, если соединение или создание stream не удалось.
func (c *Client) Connect() error {
	conn, err := grpc.NewClient(c.address, c.opts...)
	if err != nil {
		return fmt.Errorf("connecting to peer %s: %w", c.address, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	grpcClient := pb.NewChatServiceClient(conn)
	stream, err := grpcClient.Chat(ctx)
	if err != nil {
		cancel()
		_ = conn.Close()
		return fmt.Errorf("creating chat stream: %w", err)
	}

	incoming := make(chan chat.Message, bufferSize)
	sendCh := make(chan chat.Message, bufferSize)

	c.mu.Lock()
	c.conn = conn
	c.stream = stream
	c.cancel = cancel
	c.incoming = incoming
	c.sendCh = sendCh
	c.closeOnce = sync.Once{} // сбрасываем для нового соединения
	c.mu.Unlock()

	// Запускаем горутины приёма и отправки сообщений.
	go c.receiveLoop(ctx, stream, incoming)
	go c.sendLoop(ctx, stream, sendCh)

	return nil
}

// receiveLoop читает сообщения из stream, конвертирует в доменные и пишет в канал incoming.
func (c *Client) receiveLoop(ctx context.Context, stream pb.ChatService_ChatClient, incoming chan<- chat.Message) {
	defer func() {
		// Закрываем канал incoming при завершении горутины,
		// чтобы forwardIncoming() в ChatService корректно завершился.
		c.closeOnce.Do(func() {
			close(incoming)
		})
	}()

	for {
		pbMsg, err := stream.Recv()
		if err != nil {
			// Сервер закрыл соединение или произошла ошибка — завершаем цикл.
			return
		}
		select {
		case incoming <- chat.MessageFromProto(pbMsg):
		case <-ctx.Done():
			return
		}
	}
}

// sendLoop читает доменные сообщения из sendCh, конвертирует в proto и отправляет через stream.
func (c *Client) sendLoop(ctx context.Context, stream pb.ChatService_ChatClient, sendCh <-chan chat.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sendCh:
			if !ok {
				return
			}
			if err := stream.Send(msg.ToProto()); err != nil {
				return
			}
		}
	}
}

// Send отправляет доменное сообщение peer-у через активный stream.
// Реализует интерфейс chat.MessageSender.
func (c *Client) Send(msg chat.Message) error {
	c.mu.Lock()
	sendCh := c.sendCh
	c.mu.Unlock()

	if sendCh == nil {
		return fmt.Errorf("not connected")
	}

	select {
	case sendCh <- msg:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// Incoming возвращает канал входящих доменных сообщений от peer-а.
// Реализует интерфейс chat.MessageReceiver.
func (c *Client) Incoming() <-chan chat.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.incoming
}

// Close корректно закрывает соединение и stream.
// Реализует интерфейс chat.Peer.
// Безопасно вызывать повторно.
func (c *Client) Close() {
	c.mu.Lock()
	cancel := c.cancel
	stream := c.stream
	conn := c.conn
	c.cancel = nil
	c.stream = nil
	c.conn = nil
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if stream != nil {
		_ = stream.CloseSend()
	}

	if conn != nil {
		_ = conn.Close()
	}
}
