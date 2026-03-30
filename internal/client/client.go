// Package client — gRPC клиент для подключения к peer-у.
// Поддерживает bidirectional streaming для обмена сообщениями.
package client

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// bufferSize — размер буферизованных каналов сообщений.
const bufferSize = 100

// Client — gRPC клиент для установки соединения и обмена сообщениями.
// Работает с protobuf-типами; конвертация в доменные типы выполняется в chat.Service.
type Client struct {
	address string
	opts    []grpc.DialOption

	conn   *grpc.ClientConn
	stream pb.ChatService_ChatClient

	// incoming — канал входящих protobuf-сообщений от peer-а.
	incoming chan *pb.ChatMessage
	// sendCh — канал исходящих protobuf-сообщений для отправки peer-у.
	sendCh chan *pb.ChatMessage

	// cancel завершает контекст активного соединения.
	cancel context.CancelFunc
	// mu защищает доступ к полям conn и stream.
	mu sync.Mutex
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
		address:  address,
		incoming: make(chan *pb.ChatMessage, bufferSize),
		sendCh:   make(chan *pb.ChatMessage, bufferSize),
		opts:     []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Connect устанавливает соединение с peer-ом и запускает bidirectional stream.
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

	c.mu.Lock()
	c.conn = conn
	c.stream = stream
	c.cancel = cancel
	c.mu.Unlock()

	// Запускаем горутины приёма и отправки сообщений.
	go c.receiveLoop(ctx, stream)
	go c.sendLoop(ctx, stream)

	return nil
}

// receiveLoop читает сообщения из stream и пишет в канал incoming.
func (c *Client) receiveLoop(ctx context.Context, stream pb.ChatService_ChatClient) {
	for {
		pbMsg, err := stream.Recv()
		if err != nil {
			// Сервер закрыл соединение или произошла ошибка — завершаем цикл.
			return
		}
		select {
		case c.incoming <- pbMsg:
		case <-ctx.Done():
			return
		}
	}
}

// sendLoop читает сообщения из sendCh и отправляет их через stream.
func (c *Client) sendLoop(ctx context.Context, stream pb.ChatService_ChatClient) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.sendCh:
			if !ok {
				return
			}
			if err := stream.Send(msg); err != nil {
				return
			}
		}
	}
}

// Send ставит protobuf-сообщение в очередь на отправку peer-у.
func (c *Client) Send(msg *pb.ChatMessage) error {
	select {
	case c.sendCh <- msg:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// Incoming возвращает канал входящих protobuf-сообщений от peer-а.
func (c *Client) Incoming() <-chan *pb.ChatMessage {
	return c.incoming
}

// Close корректно закрывает соединение и stream.
// Безопасно вызывать повторно.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	if c.stream != nil {
		_ = c.stream.CloseSend()
		c.stream = nil
	}

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}
