# План тестирования — P2P gRPC Chat

## 1. Стратегия тестирования

### 1.1 Уровни тестирования

| Уровень | Описание | Инструменты |
|---------|----------|-------------|
| **Unit-тесты** | Тестирование отдельных функций и структур в изоляции | `testing`, `testify/assert`, `testify/require` |
| **Интеграционные тесты** | Тестирование взаимодействия компонентов (gRPC server + client) | `testing`, `grpc/test`, `bufconn` |
| **E2E тесты** | Полный сценарий: запуск двух экземпляров, обмен сообщениями | `testing`, shell scripts |

### 1.2 Подход

- **Тесты пишутся параллельно с кодом** — каждый PR должен содержать тесты для нового функционала
- **Mock-объекты** через интерфейсы (`MessageSender`, `MessageReceiver`, `UI`) — без внешних mock-фреймворков
- **Table-driven tests** — идиоматичный Go-подход для параметризованных тестов
- **`bufconn`** — in-memory gRPC соединение для интеграционных тестов без реальной сети

---

## 2. Чеклист тестов

### 2.1 Пакет `internal/chat/` — бизнес-логика

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-CH-01 | `TestNewMessage` — создание сообщения с корректными полями (sender, text, timestamp) | Unit | 🔲 Реализовать |
| T-CH-02 | `TestMessageTimestamp` — timestamp устанавливается автоматически при создании | Unit | 🔲 Реализовать |
| T-CH-03 | `TestChatServiceSend` — отправка сообщения через mock sender | Unit | 🔲 Реализовать |
| T-CH-04 | `TestChatServiceReceive` — получение сообщения из канала | Unit | 🔲 Реализовать |
| T-CH-05 | `TestChatServiceStop` — корректное завершение сервиса | Unit | 🔲 Реализовать |
| T-CH-06 | `TestEmptyMessage` — обработка пустого текста сообщения | Unit | 🔲 Реализовать |
| T-CH-07 | `TestUnicodeMessage` — корректная обработка Unicode символов (кириллица, эмодзи) | Unit | 🔲 Реализовать |

### 2.2 Пакет `internal/server/` — gRPC сервер

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-SRV-01 | `TestServerStart` — сервер запускается на указанном порту | Unit | 🔲 Реализовать |
| T-SRV-02 | `TestServerStop` — graceful shutdown сервера | Unit | 🔲 Реализовать |
| T-SRV-03 | `TestServerChatStream` — обработка bidirectional stream через bufconn | Integration | 🔲 Реализовать |
| T-SRV-04 | `TestServerClientDisconnect` — обработка отключения клиента | Integration | 🔲 Реализовать |

### 2.3 Пакет `internal/client/` — gRPC клиент

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-CLI-01 | `TestClientConnect` — подключение к серверу через bufconn | Integration | 🔲 Реализовать |
| T-CLI-02 | `TestClientSend` — отправка сообщения через stream | Integration | 🔲 Реализовать |
| T-CLI-03 | `TestClientReceive` — получение сообщения из stream | Integration | 🔲 Реализовать |
| T-CLI-04 | `TestClientClose` — корректное закрытие соединения | Integration | 🔲 Реализовать |
| T-CLI-05 | `TestClientConnectFail` — ошибка при подключении к несуществующему адресу | Unit | 🔲 Реализовать |

### 2.4 Пакет `internal/ui/` — консольный интерфейс

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-UI-01 | `TestDisplayMessage` — форматирование сообщения: имя, дата, текст | Unit | 🔲 Реализовать |
| T-UI-02 | `TestDisplaySystem` — вывод системного сообщения | Unit | 🔲 Реализовать |
| T-UI-03 | `TestReadInput` — чтение строки из reader | Unit | 🔲 Реализовать |
| T-UI-04 | `TestMessageFormat` — проверка формата `[2026-03-30 12:00:00] Alice: Hello` | Unit | 🔲 Реализовать |

### 2.5 Пакет `internal/app/` — координатор

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-APP-01 | `TestConfigIsServer` — `IsServer()` возвращает true при пустом адресе | Unit | 🔲 Реализовать |
| T-APP-02 | `TestConfigIsClient` — `IsServer()` возвращает false при указанном адресе | Unit | 🔲 Реализовать |

### 2.6 Интеграционные / E2E тесты

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-E2E-01 | `TestFullChatSession` — сервер и клиент обмениваются сообщениями через bufconn | Integration | 🔲 Реализовать |
| T-E2E-02 | `TestGracefulShutdown` — оба пира корректно завершают работу | Integration | 🔲 В плане |
| T-E2E-03 | `TestReconnectScenario` — поведение при разрыве соединения | Integration | 🔲 В плане |

---

## 3. Приоритеты реализации тестов

### Обязательные (основной костяк)

Эти тесты должны быть реализованы в первую очередь:

1. **T-CH-01, T-CH-03, T-CH-04** — базовая бизнес-логика сообщений
2. **T-CH-07** — Unicode поддержка (ключевое требование)
3. **T-SRV-01, T-SRV-03** — запуск сервера и streaming
4. **T-CLI-01, T-CLI-02, T-CLI-03** — подключение клиента и обмен
5. **T-UI-01, T-UI-04** — форматирование вывода
6. **T-APP-01, T-APP-02** — определение режима работы
7. **T-E2E-01** — полный сценарий обмена сообщениями

### Желательные (в плане)

Эти тесты реализуются при наличии времени:

- **T-CH-02, T-CH-05, T-CH-06** — edge cases бизнес-логики
- **T-SRV-02, T-SRV-04** — graceful shutdown и disconnect
- **T-CLI-04, T-CLI-05** — закрытие и ошибки подключения
- **T-UI-02, T-UI-03** — системные сообщения и чтение ввода
- **T-E2E-02, T-E2E-03** — shutdown и reconnect сценарии

---

## 4. Тестовая инфраструктура

### 4.1 Запуск тестов

```bash
# Все тесты
make test

# С покрытием
make test-coverage

# Только unit-тесты (быстрые)
go test ./internal/chat/... ./internal/ui/... ./internal/app/...

# Интеграционные тесты
go test ./internal/server/... ./internal/client/... -tags=integration
```

### 4.2 Mock-объекты

Моки реализуются вручную через интерфейсы (без codegen-фреймворков):

```go
// mockSender реализует интерфейс MessageSender для тестов.
type mockSender struct {
    sent []Message
    err  error
}

func (m *mockSender) Send(msg Message) error {
    if m.err != nil {
        return m.err
    }
    m.sent = append(m.sent, msg)
    return nil
}
```

### 4.3 bufconn для gRPC тестов

```go
// Используем bufconn для in-memory gRPC соединений в тестах.
// Это позволяет тестировать gRPC без реальной сети.
const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
    lis = bufconn.Listen(bufSize)
}
```

### 4.4 CI интеграция

Тесты запускаются автоматически в GitHub Actions при каждом push и PR:

```yaml
- name: Run tests
  run: go test -v -race -coverprofile=coverage.out ./...

- name: Check coverage
  run: go tool cover -func=coverage.out
```

---

## 5. Критерии приёмки тестов

| Критерий | Порог |
|----------|-------|
| Все обязательные тесты проходят | ✅ |
| Нет race conditions (`-race` флаг) | ✅ |
| Покрытие ключевых пакетов > 60% | ✅ |
| Тесты выполняются < 30 секунд | ✅ |
| CI pipeline зелёный | ✅ |
