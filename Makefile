# Makefile для clientgen с поддержкой PF_RING

# Переменные
BINARY_NAME = clientgen
GO = go
GOFLAGS = -v

# PF_RING пути
PFRING_INCLUDE = /usr/local/include
PFRING_LIB = /usr/local/lib

# CGO флаги для PF_RING
export CGO_CFLAGS = -I$(PFRING_INCLUDE)
export CGO_LDFLAGS = -L$(PFRING_LIB) -lpfring -lpcap
export LD_LIBRARY_PATH = $(PFRING_LIB):$(LD_LIBRARY_PATH)

# Цели
.PHONY: all build clean test install deps check-pfring

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
	rm -f $(BINARY_NAME)

# Запуск с конфигурацией по умолчанию
run: build
	sudo ./$(BINARY_NAME) -config clientgen_config.json

# Помощь
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make build-release - Build optimized binary"
	@echo "  make install       - Install binary with capabilities"
	@echo "  make test          - Run tests"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make check-pfring  - Check PF_RING installation"
	@echo "  make run           - Build and run with default config"
	@echo "  make help          - Show this help"