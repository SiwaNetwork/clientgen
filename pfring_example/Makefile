# Makefile для примера интеграции PF_RING

# Компилятор
CC = gcc

# Пути к PF_RING (используем клонированный репозиторий)
PFRING_PATH = ../PF_RING/userland/lib
PFRING_KERNEL = ../PF_RING/kernel

# Флаги компиляции
CFLAGS = -Wall -O2 -I$(PFRING_PATH) -I$(PFRING_KERNEL)
LDFLAGS = -L$(PFRING_PATH) -lpfring -lpcap -lpthread -lrt

# Исходники и цель
SRCS = src/packet_capture.c
TARGET = packet_capture

# Правила сборки
all: check_pfring $(TARGET)

# Проверка наличия PF_RING
check_pfring:
	@if [ ! -d "$(PFRING_PATH)" ]; then \
		echo "Ошибка: PF_RING не найден в $(PFRING_PATH)"; \
		echo "Выполните следующие команды:"; \
		echo "  cd .. && git clone https://github.com/ntop/PF_RING.git"; \
		echo "  cd PF_RING/userland/lib && ./configure && make"; \
		exit 1; \
	fi

# Сборка программы
$(TARGET): $(SRCS)
	$(CC) $(CFLAGS) -o $@ $^ $(LDFLAGS)

# Копирование необходимых файлов PF_RING локально (для автономной сборки)
setup_local:
	mkdir -p external/pfring
	cp $(PFRING_PATH)/pfring.h external/pfring/
	cp $(PFRING_PATH)/pfring.c external/pfring/
	cp $(PFRING_PATH)/pfring_utils.h external/pfring/
	cp $(PFRING_PATH)/pfring_utils.c external/pfring/
	cp $(PFRING_KERNEL)/linux/pf_ring.h external/pfring/linux/
	@echo "Файлы PF_RING скопированы в external/pfring/"

# Альтернативная сборка с локальными файлами
local: setup_local
	$(CC) -Wall -O2 -Iexternal/pfring -o $(TARGET) src/packet_capture.c external/pfring/pfring.c external/pfring/pfring_utils.c -lpcap -lpthread -lrt

# Очистка
clean:
	rm -f $(TARGET)
	rm -rf external/pfring

# Установка (требует root)
install: $(TARGET)
	@echo "Для установки требуются права root"
	sudo cp $(TARGET) /usr/local/bin/

# Помощь
help:
	@echo "Доступные цели:"
	@echo "  make         - собрать программу (требуется PF_RING в ../PF_RING)"
	@echo "  make local   - собрать с локальной копией PF_RING"
	@echo "  make clean   - очистить собранные файлы"
	@echo "  make install - установить программу в систему"
	@echo "  make help    - показать эту справку"

.PHONY: all check_pfring setup_local local clean install help