.PHONY: all build test test-coverage lint proto clean

# Переменные
BINARY_NAME := chat
BUILD_DIR := bin
PROTO_DIR := api/proto
PROTO_OUT := pkg/proto/chat

all: proto build

# Сборка бинарника
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/chat/

# Запуск тестов
test:
	go test -v -race ./...

# Тесты с покрытием
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Линтер
lint:
	golangci-lint run ./...

# Генерация protobuf кода
proto:
	mkdir -p $(PROTO_OUT)
	protoc \
		--go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
		-I $(PROTO_DIR) \
		$(PROTO_DIR)/chat.proto

# Форматирование кода
fmt:
	gofmt -w .
	goimports -w .

# Очистка артефактов
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out
