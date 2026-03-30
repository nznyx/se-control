// Package server — gRPC сервер для обработки P2P чат-соединений.
// Поддерживает bidirectional streaming для обмена сообщениями.
package server

import (
	"context"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"

	"github.com/nznyx/se-control/internal/chat"
	pb "github.com/nznyx/se-control/pkg/proto/chat"
)

// bufferSize — размер буферизованных каналов сообщений.
const bufferSize = 100

// Server — gRPC сервер, обрабатывающий bidirectional streaming.
// Реализует интерфейс chat.Peer: Send(chat.Message), Incoming() <-chan chat.Message, Close().
// Конвертация proto ↔ domain выполняется внутри пакета.
type Server struct {
	pb.UnimplementedChatServiceServer

	port       int
	listener   net.Listener
	grpcServer *grpc.Server

	// mu защищает доступ к полям incoming, sendCh и cancel.
	mu sync.Mutex

	// incoming — канал входящих доменных сообщений от peer-а.
	// Пересоздаётся при каждом новом соединении.
	incoming chan chat.Message

	// sendCh — канал исходящих доменных сообщений для отправки peer-у.
	// Пересоздаётся при каждом новом соединении.
	sendCh chan chat.Message

	// cancel завершает контекст активного соединения.
	cancel context.CancelFunc

	// closeOnce гарантирует однократное закрытие канала incoming.
	closeOnce sync.Once
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
		incoming: make(chan chat.Message, bufferSize),
		sendCh:   make(chan chat.Message, bufferSize),
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
// cancel() вызывается до grpcServer.Stop(), чтобы сигнализировать горутинам
// о завершении до принудительного закрытия стримов.
// Канал incoming закрывается в defer внутри Chat() при завершении стрима.
// Безопасно вызывать повторно.
func (s *Server) Stop() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	if s.grpcServer != nil {
		// Используем Stop() вместо GracefulStop() для немедленного закрытия стримов.
		// GracefulStop() ждёт завершения Chat(), который ждёт завершения stream.Recv(),
		// который разблокируется только после принудительного закрытия соединения.
		s.grpcServer.Stop()
	}

	// Если Chat() никогда не вызывался (нет активного соединения),
	// закрываем канал incoming здесь, чтобы forwardIncoming() мог завершиться.
	s.closeOnce.Do(func() {
		s.mu.Lock()
		close(s.incoming)
		s.mu.Unlock()
	})
}

// Close реализует интерфейс chat.Peer.
// Является псевдонимом для Stop().
func (s *Server) Close() {
	s.Stop()
}

// Chat — обработчик bidirectional streaming RPC.
// Создаёт свежие каналы для каждого нового соединения,
// чтобы данные от предыдущего соединения не смешивались с новыми.
func (s *Server) Chat(stream pb.ChatService_ChatServer) error {
	ctx, cancel := context.WithCancel(stream.Context())

	incoming := make(chan chat.Message, bufferSize)
	sendCh := make(chan chat.Message, bufferSize)

	s.mu.Lock()
	// Закрываем старый канал incoming перед заменой (если он ещё открыт).
	// Используем отдельный closeOnce для каждого соединения.
	s.incoming = incoming
	s.sendCh = sendCh
	s.cancel = cancel
	s.closeOnce = sync.Once{} // сбрасываем для нового соединения
	s.mu.Unlock()

	defer func() {
		cancel()
		// Закрываем канал incoming при завершении стрима,
		// чтобы forwardIncoming() мог корректно завершиться.
		s.closeOnce.Do(func() {
			s.mu.Lock()
			close(s.incoming)
			s.mu.Unlock()
		})
	}()

	var wg sync.WaitGroup

	// Горутина приёма сообщений от peer-а.
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.receiveLoop(ctx, stream, incoming)
	}()

	// Горутина отправки сообщений peer-у.
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.sendLoop(ctx, stream, sendCh)
	}()

	wg.Wait()
	return nil
}

// receiveLoop читает сообщения из stream, конвертирует в доменные и пишет в канал incoming.
func (s *Server) receiveLoop(ctx context.Context, stream pb.ChatService_ChatServer, incoming chan<- chat.Message) {
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
			case incoming <- chat.MessageFromProto(pbMsg):
			case <-ctx.Done():
				return
			}
		}
	}
}

// sendLoop читает доменные сообщения из sendCh, конвертирует в proto и отправляет через stream.
func (s *Server) sendLoop(ctx context.Context, stream pb.ChatService_ChatServer, sendCh <-chan chat.Message) {
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
func (s *Server) Send(msg chat.Message) error {
	s.mu.Lock()
	sendCh := s.sendCh
	s.mu.Unlock()

	select {
	case sendCh <- msg:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// Incoming возвращает канал входящих доменных сообщений от peer-а.
// Реализует интерфейс chat.MessageReceiver.
func (s *Server) Incoming() <-chan chat.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.incoming
}
