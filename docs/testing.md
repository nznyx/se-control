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
| T-CH-01 | `TestNewMessage` — создание сообщения с корректными полями (sender, text, timestamp) | Unit | ✅ Реализован |
| T-CH-02 | `TestMessageTimestamp` — timestamp устанавливается автоматически при создании | Unit | ✅ Реализован |
| T-CH-03 | `TestService_Send_WithMock` — отправка сообщения через mock sender | Unit | ✅ Реализован |
| T-CH-04 | `TestService_Receive` — получение сообщения из канала | Unit | ✅ Реализован |
| T-CH-05 | `TestService_Stop` — корректное завершение сервиса | Unit | ✅ Реализован |
| T-CH-06 | `TestService_Send_WithMock/empty_message` — обработка пустого текста сообщения | Unit | ✅ Реализован |
| T-CH-07 | `TestService_Send_WithMock/unicode` — корректная обработка Unicode символов (кириллица, эмодзи) | Unit | ✅ Реализован |

### 2.2 Пакет `internal/server/` — gRPC сервер

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-SRV-01 | `TestServer_Start` — сервер запускается на указанном порту | Unit | ✅ Реализован |
| T-SRV-02 | `TestServer_Stop` — graceful shutdown сервера | Unit | ✅ Реализован |
| T-SRV-03 | `TestServer_ChatStream` — обработка bidirectional stream через bufconn | Integration | ✅ Реализован |
| T-SRV-04 | `TestServer_ClientDisconnect` — обработка отключения клиента | Integration | ✅ Реализован |

### 2.3 Пакет `internal/client/` — gRPC клиент

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-CLI-01 | `TestClient_Connect` — подключение к серверу через bufconn | Integration | ✅ Реализован |
| T-CLI-02 | `TestClient_Send` — отправка сообщения через stream | Integration | ✅ Реализован |
| T-CLI-03 | `TestClient_Receive` — получение сообщения из stream | Integration | ✅ Реализован |
| T-CLI-04 | `TestClient_Close` — корректное закрытие соединения | Integration | ✅ Реализован |
| T-CLI-05 | `TestClient_ConnectFail` — ошибка при подключении к несуществующему адресу | Unit | ✅ Реализован |

### 2.4 Пакет `internal/ui/` — консольный интерфейс

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-UI-01 | `TestConsole_DisplayMessage` — форматирование сообщения: имя, дата, текст | Unit | ✅ Реализован |
| T-UI-02 | `TestConsole_DisplaySystem` — вывод системного сообщения | Unit | ✅ Реализован |
| T-UI-03 | `TestConsole_ReadInput` — чтение строки из reader | Unit | ✅ Реализован |
| T-UI-04 | `TestConsole_DisplayMessage` — проверка формата `[2026-03-30 12:00:00] Alice: Hello` | Unit | ✅ Реализован |

### 2.5 Пакет `internal/app/` — координатор

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-APP-01 | `TestConfig_IsServer/empty_address` — `IsServer()` возвращает true при пустом адресе | Unit | ✅ Реализован |
| T-APP-02 | `TestConfig_IsServer/with_address` — `IsServer()` возвращает false при указанном адресе | Unit | ✅ Реализован |

### 2.6 Интеграционные / E2E тесты

| ID | Тест | Тип | Статус |
|----|------|-----|--------|
| T-E2E-01 | `TestFullChatSession` — сервер и клиент обмениваются сообщениями через bufconn | Integration | ✅ Реализован |
| T-E2E-02 | `TestGracefulShutdown` — оба пира корректно завершают работу | Integration | ✅ Реализован |
| T-E2E-03 | `TestReconnectScenario` — поведение при разрыве соединения | Integration | ✅ Реализован |

---

## 3. Приоритеты реализации тестов

### Обязательные (основной костяк)

Все обязательные тесты реализованы ✅:

1. **T-CH-01, T-CH-03, T-CH-04** — базовая бизнес-логика сообщений ✅
2. **T-CH-07** — Unicode поддержка (ключевое требование) ✅
3. **T-SRV-01, T-SRV-03** — запуск сервера и streaming ✅
4. **T-CLI-01, T-CLI-02, T-CLI-03** — подключение клиента и обмен ✅
5. **T-UI-01, T-UI-04** — форматирование вывода ✅
6. **T-APP-01, T-APP-02** — определение режима работы ✅
7. **T-E2E-01** — полный сценарий обмена сообщениями ✅

### Желательные

Реализованы дополнительно:

- **T-CH-02, T-CH-05, T-CH-06** — edge cases бизнес-логики ✅
- **T-SRV-02, T-SRV-04** — graceful shutdown и disconnect ✅
- **T-CLI-04, T-CLI-05** — закрытие и ошибки подключения ✅
- **T-UI-02, T-UI-03** — системные сообщения и чтение ввода ✅
- **T-E2E-02** — graceful shutdown обоих пиров ✅
- **T-E2E-03** — reconnect сценарий ✅

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

Моки реализуются вручную через интерфейс `protoSender` (без codegen-фреймворков):

```go
// mockPeer реализует интерфейс protoSender для тестов.
type mockPeer struct {
    sent     []*pb.ChatMessage
    err      error
    incoming chan *pb.ChatMessage
}

func (m *mockPeer) Send(msg *pb.ChatMessage) error {
    if m.err != nil {
        return m.err
    }
    m.sent = append(m.sent, msg)
    return nil
}

func (m *mockPeer) Incoming() <-chan *pb.ChatMessage {
    return m.incoming
}
```

### 4.3 bufconn для gRPC тестов

```go
// Используем bufconn для in-memory gRPC соединений в тестах.
// Это позволяет тестировать gRPC без реальной сети.
const bufSize = 1024 * 1024

lis := bufconn.Listen(bufSize)
srv := server.New(0, server.WithListener(lis))
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
