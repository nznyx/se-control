# P2P gRPC Chat

КР за 3 модуль по SE. Выполнили: Токарев Алексей и Деружинский Дмитрий.

Консольный P2P чат на Go с использованием gRPC bidirectional streaming.

## Возможности

- Peer-to-peer обмен текстовыми сообщениями (Unicode)
- Режим сервера (ожидание подключения) и клиента (подключение к peer-у)
- Отображение имени отправителя, даты и текста сообщения
- Graceful shutdown по Ctrl+C

## Требования

- Go 1.26+
- `protoc` (Protocol Buffers compiler)
- `protoc-gen-go`, `protoc-gen-go-grpc`
- `golangci-lint` (для линтинга)

## Установка зависимостей

```bash
# Установка protoc плагинов для Go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Загрузка Go-зависимостей
go mod download
```

## Сборка

```bash
# Генерация protobuf кода и сборка
make build

# Или вручную:
# 1. Генерация proto
make proto

# 2. Сборка бинарника
go build -o bin/chat ./cmd/chat/
```

## Запуск

### Режим сервера

Запуск без указания адреса — приложение ожидает входящее подключение:

```bash
./bin/chat --name Alice --port 50051
```

### Режим клиента

Запуск с указанием адреса peer-а — приложение подключается к серверу:

```bash
./bin/chat --name Bob --address localhost:50051
```

### Параметры

| Флаг | Описание | Обязательный |
|------|----------|-------------|
| `--name` | Имя пользователя | Да |
| `--port` | Порт для прослушивания (режим сервера) | Нет (по умолчанию 50051) |
| `--address` | Адрес peer-а для подключения (режим клиента) | Нет |

Если `--address` не указан — запуск в режиме сервера. Если указан — в режиме клиента.

## Тестирование

```bash
# Все тесты
make test

# С покрытием
make test-coverage

# Линтер
make lint
```

## Структура проекта

```
├── api/proto/          # Protobuf определения
├── cmd/chat/           # Точка входа
├── internal/
│   ├── app/            # Координатор приложения
│   ├── chat/           # Бизнес-логика чата
│   ├── client/         # gRPC клиент
│   ├── server/         # gRPC сервер
│   └── ui/             # Консольный ввод/вывод
├── pkg/proto/chat/     # Сгенерированный protobuf код
└── docs/               # Документация
```

## Документация

- [Архитектура](docs/architecture.md) — диаграммы компонентов, классов, обоснование решений
- [Тестирование](docs/testing.md) — план тестирования, чеклист тестов
- [Стайлгайд](docs/styleguide.md) — конвенции кода, именование, Git workflow

## Лицензия

MIT
