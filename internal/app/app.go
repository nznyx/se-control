// Package app — координатор приложения.
// Отвечает за инициализацию компонентов и управление жизненным циклом.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nznyx/se-control/internal/chat"
	"github.com/nznyx/se-control/internal/client"
	"github.com/nznyx/se-control/internal/server"
	"github.com/nznyx/se-control/internal/ui"
)

// Config содержит конфигурацию приложения, полученную из CLI-аргументов.
type Config struct {
	Username    string
	PeerAddress string
	Port        int
}

// IsServer возвращает true, если приложение запущено в режиме сервера
// (адрес peer-а не указан).
func (c Config) IsServer() bool {
	return c.PeerAddress == ""
}

// App — координатор приложения, управляющий жизненным циклом компонентов.
type App struct {
	config Config

	cancel context.CancelFunc
	mu     sync.Mutex
}

// New создаёт новый экземпляр App с указанной конфигурацией.
func New(cfg Config) *App {
	return &App{
		config: cfg,
	}
}

// Run запускает приложение. Блокирует до завершения.
func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()

	// Настройка graceful shutdown по сигналам ОС
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		a.Shutdown()
	}()

	chatService := chat.NewService(a.config.Username)
	consoleUI := ui.New(os.Stdin, os.Stdout)

	if a.config.IsServer() {
		srv := server.New(a.config.Port)
		if err := srv.Start(); err != nil {
			return fmt.Errorf("starting server: %w", err)
		}
		consoleUI.DisplaySystem(fmt.Sprintf("Server is listening on port %d, waiting for peer...", a.config.Port))
		chatService.SetPeer(srv)
	} else {
		cli := client.New(a.config.PeerAddress)
		consoleUI.DisplaySystem(fmt.Sprintf("Connecting to peer at %s...", a.config.PeerAddress))
		if err := cli.Connect(); err != nil {
			return fmt.Errorf("connecting to peer: %w", err)
		}
		consoleUI.DisplaySystem("Connected to peer.")
		chatService.SetPeer(cli)
	}

	// Горутина для вывода входящих сообщений
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-chatService.Incoming():
				if !ok {
					// FR-08: сообщение при обрыве соединения
					if ctx.Err() == nil {
						consoleUI.DisplaySystem("Peer connection closed. Exiting...")
						a.Shutdown()
					}
					return
				}
				consoleUI.DisplayMessage(msg)
			}
		}
	}()

	// Горутина для чтения пользовательского ввода
	go func() {
		for {
			// Проверка завершения перед чтением
			if ctx.Err() != nil {
				return
			}
			text, err := consoleUI.ReadInput()
			if err != nil {
				if ctx.Err() == nil {
					consoleUI.DisplaySystem(fmt.Sprintf("Error reading input: %v", err))
					a.Shutdown()
				}
				return
			}
			if text != "" {
				if err := chatService.Send(text); err != nil {
					if ctx.Err() == nil {
						consoleUI.DisplaySystem(fmt.Sprintf("Error sending message: %v", err))
					}
				}
			}
		}
	}()

	// Ожидаем завершения (обрыв соединения или системный сигнал)
	<-ctx.Done()

	// Корректно завершаем сервисы (сервер/клиент)
	chatService.Stop()

	return nil
}

// Shutdown корректно завершает работу приложения.
func (a *App) Shutdown() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}
