// Package app — координатор приложения.
// Отвечает за инициализацию компонентов и управление жизненным циклом.
package app

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
}

// New создаёт новый экземпляр App с указанной конфигурацией.
func New(cfg Config) *App {
	return &App{
		config: cfg,
	}
}

// Run запускает приложение. Блокирует до завершения.
func (a *App) Run() error {
	// TODO: реализовать инициализацию и запуск компонентов.
	return nil
}

// Shutdown корректно завершает работу приложения.
func (a *App) Shutdown() {
	// TODO: реализовать graceful shutdown.
}
