// Package server — gRPC сервер для обработки P2P чат-соединений.
// Поддерживает bidirectional streaming для обмена сообщениями.
package server

import (
	"context"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"

	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// bufferSize — размер буферизованных каналов сообщений.
const bufferSize = 100

// Server — gRPC сервер, обрабатывающий bidirectional streaming.
// Работает с protobuf-типами; конвертация в доменные типы выполняется в chat.Service.
type Server struct {
	pb.UnimplementedChatServiceServer

	port       int
	listener   net.Listener
	grpcServer *grpc.Server

	// incoming — канал входящих protobuf-сообщений от peer-а.
	incoming chan *pb.ChatMessage
	// sendCh — канал исходящих protobuf-сообщений для отправки peer-у.
	sendCh chan *pb.ChatMessage

	// mu защищает доступ к полям stream и cancel.
	mu     sync.Mutex
	stream pb.ChatService_ChatServer

	// cancel завершает контекст активного соединения.
	cancel context.CancelFunc
}

// Option — функциональная опция для конфигурации Server.
type Option func(*Server)

// WithListener задаёт кастомный net.Listener (используется в тестах с bufconn).
func WithListener(lis net.Listener) Option {
	return func(s *Server) {
		s.listener = lis
	}
}

// New создаёт новый экземпляр Server.
func New(port int, opts ...Option) *Server {
	s := &Server{
		port:     port,
		incoming: make(chan *pb.ChatMessage, bufferSize),
		sendCh:   make(chan *pb.ChatMessage, bufferSize),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start запускает gRPC сервер на указанном порту.
// Если задан кастомный listener (через WithListener), использует его.
// Возвращает ошибку, если порт уже занят или listener не удалось создать.
func (s *Server) Start() error {
	if s.listener == nil {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
		if err != nil {
			return fmt.Errorf("listening on port %d: %w", s.port, err)
		}
		s.listener = lis
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterChatServiceServer(s.grpcServer, s)

	// Запускаем сервер в отдельной горутине, чтобы не блокировать вызывающий код.
	go func() {
		_ = s.grpcServer.Serve(s.listener)
	}()

	return nil
}

// Stop корректно останавливает сервер.
// Безопасно вызывать повторно.
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
}

// Chat — обработчик bidirectional streaming RPC.
// Запускает горутины для одновременного приёма и отправки сообщений.
func (s *Server) Chat(stream pb.ChatService_ChatServer) error {
	ctx, cancel := context.WithCancel(stream.Context())

	s.mu.Lock()
	s.stream = stream
	s.cancel = cancel
	s.mu.Unlock()

	defer cancel()

	var wg sync.WaitGroup

	// Горутина приёма сообщений от peer-а.
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.receiveLoop(ctx, stream)
	}()

	// Горутина отправки сообщений peer-у.
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.sendLoop(ctx, stream)
	}()

	wg.Wait()
	return nil
}

// receiveLoop читает сообщения из stream и пишет в канал incoming.
func (s *Server) receiveLoop(ctx context.Context, stream pb.ChatService_ChatServer) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			pbMsg, err := stream.Recv()
			if err != nil {
				// Клиент закрыл соединение или произошла ошибка — завершаем цикл.
				return
			}
			select {
			case s.incoming <- pbMsg:
			case <-ctx.Done():
				return
			}
		}
	}
}

// sendLoop читает сообщения из sendCh и отправляет их через stream.
func (s *Server) sendLoop(ctx context.Context, stream pb.ChatService_ChatServer) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-s.sendCh:
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
func (s *Server) Send(msg *pb.ChatMessage) error {
	select {
	case s.sendCh <- msg:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// Incoming возвращает канал входящих protobuf-сообщений от peer-а.
func (s *Server) Incoming() <-chan *pb.ChatMessage {
	return s.incoming
}
