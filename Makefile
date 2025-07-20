# Makefile для clientgen с поддержкой PF_RING

# Переменные
BINARY_NAME = clientgen
WEBSERVER_NAME = webserver
GO = go
GOFLAGS = -v

# PF_RING пути
PFRING_INCLUDE = /usr/local/include
PFRING_LIB = /usr/local/lib

# CGO флаги для PF_RING
export CGO_CFLAGS = -I$(PFRING_INCLUDE)
export CGO_LDFLAGS = -L$(PFRING_LIB) -lpfring -lpcap

# Цели
.PHONY: all build build-web clean test install deps check-pfring web run-web

all: build

# Проверка установки PF_RING
check-pfring:
	@echo "Checking PF_RING installation..."
	@./check_pfring.sh

# Установка зависимостей
deps:
	$(GO) mod download
	$(GO) mod tidy

# Сборка
build: deps
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

# Сборка веб-сервера (без PF_RING зависимостей)
build-web: deps
	@echo "Building $(WEBSERVER_NAME)..."
	env -u CGO_CFLAGS -u CGO_LDFLAGS $(GO) build $(GOFLAGS) -o $(WEBSERVER_NAME) webserver.go

# Сборка обоих приложений
web: build build-web

# Сборка с оптимизациями
build-release: deps
	@echo "Building optimized $(BINARY_NAME)..."
	$(GO) build -ldflags="-s -w" -o $(BINARY_NAME) .

# Установка
install: build
	@echo "Installing $(BINARY_NAME)..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo setcap cap_net_raw,cap_net_admin=eip /usr/local/bin/$(BINARY_NAME)

# Тесты
test:
	$(GO) test -v ./clientgenlib/...

# Очистка
clean:
	@echo "Cleaning..."
	$(GO) clean
	rm -f $(BINARY_NAME) $(WEBSERVER_NAME)

# Запуск с конфигурацией по умолчанию
run: build
	sudo ./$(BINARY_NAME) -config clientgen_config.json

# Запуск веб-сервера
run-web: build-web
	./$(WEBSERVER_NAME)

# Помощь
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make build-web     - Build the web server"
	@echo "  make web           - Build both binary and web server"
	@echo "  make build-release - Build optimized binary"
	@echo "  make install       - Install binary with capabilities"
	@echo "  make test          - Run tests"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make check-pfring  - Check PF_RING installation"
	@echo "  make run           - Build and run with default config"
	@echo "  make run-web       - Build and run web server"
	@echo "  make help          - Show this help"