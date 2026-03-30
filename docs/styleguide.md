# Стайлгайд — P2P gRPC Chat

## 1. Общие принципы

Проект следует стандартным Go-конвенциям и принципам качественной разработки:

| Принцип | Применение |
|---------|-----------|
| **SOLID** | Разделение ответственности по пакетам, зависимость от интерфейсов |
| **KISS** | Минимальная сложность, один бинарник, простые структуры данных |
| **DRY** | Общие типы в `internal/chat/message.go`, переиспользование интерфейсов |
| **YAGNI** | Реализуем только требуемый функционал: 1-на-1 чат, текстовые сообщения |

---

## 2. Форматирование кода

### 2.1 Инструменты

| Инструмент | Назначение | Команда |
|-----------|-----------|---------|
| `gofmt` | Стандартное форматирование Go | `gofmt -w .` |
| `goimports` | Форматирование + сортировка импортов | `goimports -w .` |
| `golangci-lint` | Агрегатор линтеров | `golangci-lint run` |

Весь код **обязательно** форматируется через `gofmt` перед коммитом. CI проверяет это автоматически.

### 2.2 Длина строки

- Мягкий лимит: **100 символов**
- Жёсткий лимит: **120 символов**
- Исключения: строковые литералы, URL в комментариях

### 2.3 Отступы

- Только **табы** (стандарт Go, `gofmt` обеспечивает автоматически)

---

## 3. Именование

### 3.1 Пакеты

- Имена пакетов — **строчные**, одно слово, без подчёркиваний
- Имя пакета совпадает с именем директории
- Избегать generic-имён: `utils`, `helpers`, `common`

```go
// ✅ Хорошо
package server
package client
package chat

// ❌ Плохо
package chatServer
package chat_server
package utils
```

### 3.2 Переменные и функции

- **camelCase** для неэкспортируемых
- **PascalCase** для экспортируемых
- Имена должны быть осмысленными и отражать назначение
- Избегать однобуквенных имён (кроме `i`, `j` в циклах, `err` для ошибок)

```go
// ✅ Хорошо
func (s *Server) Start(port int) error
func (c *Client) Connect(address string) error
var messageChannel chan Message

// ❌ Плохо
func (s *Server) S(p int) error
func (c *Client) Do(a string) error
var mc chan Message
```

### 3.3 Интерфейсы

- Интерфейсы с одним методом именуются суффиксом `-er`: `Reader`, `Writer`, `Sender`
- Интерфейсы определяются **в пакете-потребителе**, а не в пакете-реализации

```go
// ✅ Хорошо — интерфейс в пакете chat, реализация в пакете server
// internal/chat/interfaces.go
type MessageSender interface {
    Send(msg Message) error
}
```

### 3.4 Константы

- **PascalCase** для экспортируемых
- **camelCase** для неэкспортируемых
- Группировка через `const ()` блоки
- Никаких magic numbers/strings — все значения именованы

```go
// ✅ Хорошо
const (
    DefaultPort    = 50051
    MaxMessageSize = 4096
)

// ❌ Плохо — magic numbers
grpc.NewServer(grpc.MaxRecvMsgSize(4096))
```

### 3.5 Файлы

- Имена файлов — **snake_case**: `chat_service.go`, `grpc_server.go`
- Тестовые файлы: `*_test.go`
- Один основной тип/структура на файл (при необходимости)

---

## 4. Структура файла

Порядок элементов в файле:

1. Пакет (`package`)
2. Импорты (сгруппированные: stdlib, external, internal)
3. Константы
4. Типы (интерфейсы, структуры)
5. Конструкторы (`New...`)
6. Методы
7. Вспомогательные функции (неэкспортируемые)

```go
package server

import (
    "context"
    "fmt"
    "net"

    "google.golang.org/grpc"

    pb "github.com/nznyx/se-control/pkg/proto/chat"
)

const defaultBufferSize = 100

// Server — gRPC сервер для обработки чат-соединений.
type Server struct {
    pb.UnimplementedChatServiceServer
    grpcServer *grpc.Server
    messages   chan *pb.ChatMessage
    port       int
}

// New создаёт новый экземпляр Server.
func New(port int) *Server {
    return &Server{
        port:     port,
        messages: make(chan *pb.ChatMessage, defaultBufferSize),
    }
}

// Start запускает gRPC сервер на указанном порту.
func (s *Server) Start() error {
    // ...
}

// Stop корректно останавливает сервер.
func (s *Server) Stop() {
    // ...
}
```

---

## 5. Импорты

Импорты группируются в три блока, разделённые пустой строкой:

1. **Стандартная библиотека**
2. **Внешние зависимости**
3. **Внутренние пакеты проекта**

```go
import (
    "context"
    "fmt"
    "time"

    "google.golang.org/grpc"
    "github.com/stretchr/testify/assert"

    "github.com/nznyx/se-control/internal/chat"
    pb "github.com/nznyx/se-control/pkg/proto/chat"
)
```

`goimports` автоматически обеспечивает правильную группировку.

---

## 6. Обработка ошибок

### 6.1 Правила

- **Всегда** проверять возвращаемые ошибки
- **Не игнорировать** ошибки (кроме явно документированных случаев)
- Оборачивать ошибки с контекстом через `fmt.Errorf("...: %w", err)`
- Определять sentinel errors через `errors.New()` для ожидаемых ошибок

```go
// ✅ Хорошо
conn, err := grpc.Dial(address, grpc.WithInsecure())
if err != nil {
    return fmt.Errorf("connecting to peer %s: %w", address, err)
}

// ✅ Хорошо — sentinel error
var ErrNotConnected = errors.New("not connected to peer")

// ❌ Плохо — игнорирование ошибки
conn, _ := grpc.Dial(address, grpc.WithInsecure())

// ❌ Плохо — потеря контекста
if err != nil {
    return err
}
```

### 6.2 Логирование ошибок

- Используем стандартный `log` пакет (без внешних логгеров — YAGNI)
- Логируем на уровне, где ошибка обрабатывается, а не на каждом уровне стека

---

## 7. Комментарии и документация

### 7.1 GoDoc комментарии

- **Все экспортируемые** типы, функции, методы и константы должны иметь GoDoc комментарий
- Комментарий начинается с имени элемента
- Полные предложения с точкой в конце

```go
// Server — gRPC сервер для обработки P2P чат-соединений.
// Поддерживает bidirectional streaming для обмена сообщениями.
type Server struct { ... }

// Start запускает gRPC сервер на указанном порту.
// Возвращает ошибку, если порт уже занят.
func (s *Server) Start() error { ... }

// DefaultPort — порт по умолчанию для gRPC сервера.
const DefaultPort = 50051
```

### 7.2 Inline комментарии

- Только для неочевидной логики
- Объясняют **почему**, а не **что**
- На русском или английском (единообразно в рамках файла)

```go
// ✅ Хорошо — объясняет почему
// Используем буферизованный канал, чтобы отправитель не блокировался
// при медленном чтении получателем.
messages := make(chan Message, 100)

// ❌ Плохо — объясняет что (и так видно из кода)
// Создаём канал сообщений
messages := make(chan Message, 100)
```

---

## 8. Конкурентность

### 8.1 Goroutines

- Каждая goroutine должна иметь чёткий механизм завершения (context, done channel)
- Документировать goroutine в комментарии к функции, которая её запускает
- Не запускать goroutine без возможности её остановить

```go
// ✅ Хорошо
func (s *Server) listenMessages(ctx context.Context, stream pb.ChatService_ChatServer) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            msg, err := stream.Recv()
            if err != nil {
                return
            }
            s.messages <- msg
        }
    }
}
```

### 8.2 Channels

- Предпочитать channels мьютексам
- Закрывать канал только на стороне отправителя
- Использовать буферизованные каналы для предотвращения блокировок

---

## 9. Тестирование (стиль)

### 9.1 Именование тестов

```go
// Формат: Test<Struct><Method>_<Scenario>
func TestServer_Start_Success(t *testing.T) { ... }
func TestServer_Start_PortBusy(t *testing.T) { ... }
func TestClient_Connect_InvalidAddress(t *testing.T) { ... }
```

### 9.2 Table-driven tests

```go
func TestConfig_IsServer(t *testing.T) {
    tests := []struct {
        name        string
        peerAddress string
        want        bool
    }{
        {
            name:        "empty address means server mode",
            peerAddress: "",
            want:        true,
        },
        {
            name:        "with address means client mode",
            peerAddress: "localhost:50051",
            want:        false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cfg := Config{PeerAddress: tt.peerAddress}
            assert.Equal(t, tt.want, cfg.IsServer())
        })
    }
}
```

### 9.3 Assertions

- Используем `testify/assert` для читаемых assertion-ов
- `assert` — для non-fatal проверок (тест продолжается)
- `require` — для fatal проверок (тест останавливается)

---

## 10. Git конвенции

### 10.1 Commit messages

Формат: `<type>: <description>`

| Тип | Описание |
|-----|----------|
| `feat` | Новая функциональность |
| `fix` | Исправление бага |
| `docs` | Документация |
| `test` | Тесты |
| `refactor` | Рефакторинг без изменения поведения |
| `ci` | Изменения CI/CD |
| `chore` | Прочие изменения (зависимости, конфиг) |

Примеры:
```
feat: add gRPC server with bidirectional streaming
fix: handle client disconnect gracefully
docs: add architecture documentation
test: add unit tests for chat service
ci: add GitHub Actions workflow
```

### 10.2 Ветки

| Ветка | Назначение |
|-------|-----------|
| `main` | Стабильная версия, только через PR |
| `feature/<name>` | Новая функциональность |
| `fix/<name>` | Исправление бага |
| `docs/<name>` | Документация |

### 10.3 Pull Requests

- Заголовок PR совпадает с форматом commit message
- Описание PR содержит: что сделано, как тестировалось, связанные issues
- Каждый PR проходит code review вторым участником
- CI должен быть зелёным перед merge

---

## 11. Линтер — конфигурация

Файл `.golangci.yml` в корне проекта:

```yaml
linters:
  enable:
    - errcheck      # проверка обработки ошибок
    - govet         # подозрительные конструкции
    - staticcheck   # статический анализ
    - unused        # неиспользуемый код
    - gosimple      # упрощение кода
    - ineffassign   # неэффективные присваивания
    - gofmt         # форматирование
    - goimports     # импорты
    - misspell      # опечатки в комментариях

linters-settings:
  govet:
    check-shadowing: true

issues:
  exclude-use-default: false
```

---

## 12. Антипаттерны — чего избегаем

| Антипаттерн | Как избегаем |
|-------------|-------------|
| **Spaghetti Code** | Чёткое разделение по пакетам, каждый пакет — одна ответственность |
| **God Object** | Нет единого объекта, управляющего всем; `App` только координирует |
| **Boat Anchor** | Не добавляем код «на будущее»; только требуемый функционал |
| **Golden Hammer** | gRPC используется для сети, стандартная библиотека — для остального |
| **Copy-paste Coding** | Общие типы и интерфейсы в `internal/chat/` |
| **Magic Numbers** | Все числовые значения — именованные константы |
| **Dependency Hell** | Минимум зависимостей: grpc, protobuf, testify, cobra |
| **Hardcoding** | Конфигурация через CLI-флаги, константы для значений по умолчанию |
| **Big Ball of Mud** | Чёткая структура пакетов, интерфейсы между слоями |
| **Premature Optimization** | Простые структуры данных, оптимизация только при необходимости |
| **Reinventing the Wheel** | Используем gRPC вместо своего протокола, `cobra` для CLI |

---

## 13. Зависимости проекта

| Зависимость | Версия | Назначение |
|-------------|--------|-----------|
| `google.golang.org/grpc` | latest | gRPC фреймворк |
| `google.golang.org/protobuf` | latest | Protobuf runtime |
| `github.com/spf13/cobra` | latest | CLI фреймворк |
| `github.com/stretchr/testify` | latest | Test assertions |

Минимальный набор зависимостей — только то, что действительно необходимо.
