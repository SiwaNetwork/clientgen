# Руководство по установке и использованию ClientGen

## Содержание
1. [Требования](#требования)
2. [Установка зависимостей](#установка-зависимостей)
3. [Сборка проекта](#сборка-проекта)
4. [Конфигурация](#конфигурация)
5. [Запуск](#запуск)
6. [Мониторинг и отладка](#мониторинг-и-отладка)
7. [Решение проблем](#решение-проблем)

## Требования

### Системные требования
- **ОС**: Linux (рекомендуется Ubuntu 20.04+ или CentOS 8+)
- **Архитектура**: x86_64
- **Память**: минимум 4GB RAM (8GB+ для больших нагрузок)
- **Сеть**: сетевая карта с поддержкой аппаратных timestamp'ов

### Программные требования
- **Go**: версия 1.16 или выше
- **PF_RING**: последняя стабильная версия
- **Git**: для клонирования репозитория
- **Права**: root доступ для работы с PF_RING

## Установка зависимостей

### 1. Установка Go

```bash
# Для Ubuntu/Debian
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Проверка установки
go version
```

### 2. Установка PF_RING

```bash
# Клонирование репозитория PF_RING
git clone https://github.com/ntop/PF_RING.git
cd PF_RING

# Установка зависимостей для сборки
sudo apt-get update
sudo apt-get install -y build-essential bison flex linux-headers-$(uname -r)

# Сборка и установка kernel модуля
cd kernel
make
sudo make install

# Сборка и установка библиотеки
cd ../userland/lib
./configure
make
sudo make install

# Загрузка модуля
sudo modprobe pf_ring

# Проверка установки
lsmod | grep pf_ring
```

### 3. Настройка системы

```bash
# Увеличение лимитов системы
sudo sysctl -w net.core.rmem_max=134217728
sudo sysctl -w net.core.wmem_max=134217728
sudo sysctl -w net.core.netdev_max_backlog=5000
sudo sysctl -w net.ipv4.udp_mem="4096 87380 134217728"

# Сохранение настроек
echo "net.core.rmem_max=134217728" | sudo tee -a /etc/sysctl.conf
echo "net.core.wmem_max=134217728" | sudo tee -a /etc/sysctl.conf
echo "net.core.netdev_max_backlog=5000" | sudo tee -a /etc/sysctl.conf
echo "net.ipv4.udp_mem=4096 87380 134217728" | sudo tee -a /etc/sysctl.conf
```

## Сборка проекта

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd clientgen
```

### 2. Установка Go зависимостей

```bash
# Инициализация модулей (если необходимо)
go mod download

# Проверка зависимостей
go mod verify
```

### 3. Сборка бинарного файла

```bash
# Сборка с оптимизациями
CGO_CFLAGS="-I/usr/local/include" \
CGO_LDFLAGS="-L/usr/local/lib -lpfring -lpcap" \
go build -ldflags="-s -w" -o clientgen main.go

# Проверка сборки
./clientgen -h
```

## Конфигурация

### Основной конфигурационный файл

Создайте или отредактируйте файл `clientgen_config.json`:

```json
{
    // Сетевые настройки
    "Iface": "eth0",                    // Имя сетевого интерфейса
    "ServerMAC": "00:11:22:33:44:55",   // MAC адрес PTP сервера
    "ServerAddress": "192.168.1.100",    // IP адрес PTP сервера
    
    // Настройки клиентов
    "ClientIPStart": "192.168.1.10",     // Начальный IP клиентов
    "ClientIPEnd": "192.168.1.20",       // Конечный IP клиентов
    "ClientIPStep": 1,                   // Шаг между IP адресами
    
    // Временные параметры
    "TimeoutSec": 90,                    // Общий таймаут
    "DurationSec": 60,                   // Длительность теста
    "TimeBetweenDelayReqSec": 1,         // Интервал между DelayReq
    
    // Параметры производительности
    "NumTXWorkers": 4,                   // Количество TX воркеров
    "NumRXWorkers": 4,                   // Количество RX воркеров
    "NumPacketParsers": 4,               // Количество парсеров пакетов
    "NumPacketProcessors": 4,            // Количество обработчиков
    
    // Параметры отладки
    "DebugPrint": false,                 // Подробный вывод
    "PrintPerformance": true,            // Вывод статистики
    "CounterPrintIntervalSecs": 1        // Интервал вывода счетчиков
}
```

### Примеры конфигураций

#### Минимальная конфигурация (10 клиентов)
```json
{
    "Iface": "eth0",
    "ServerMAC": "00:11:22:33:44:55",
    "ServerAddress": "192.168.1.100",
    "ClientIPStart": "192.168.1.10",
    "ClientIPEnd": "192.168.1.19",
    "ClientIPStep": 1,
    "TimeoutSec": 30,
    "DurationSec": 10,
    "NumTXWorkers": 1,
    "NumRXWorkers": 1
}
```

#### Высоконагруженная конфигурация (10000 клиентов)
```json
{
    "Iface": "eth0",
    "ServerMAC": "00:11:22:33:44:55",
    "ServerAddress": "10.0.0.100",
    "ClientIPStart": "10.1.0.1",
    "ClientIPEnd": "10.1.39.16",
    "ClientIPStep": 1,
    "TimeoutSec": 300,
    "DurationSec": 120,
    "NumTXWorkers": 8,
    "NumRXWorkers": 8,
    "NumPacketParsers": 8,
    "NumPacketProcessors": 8
}
```

## Запуск

### Базовый запуск

```bash
# Запуск с правами root (необходимо для PF_RING)
sudo ./clientgen -config clientgen_config.json
```

### Запуск с профилированием

```bash
# С CPU профилированием
sudo ./clientgen -config clientgen_config.json -profilelog cpu_profile.prof

# Анализ профиля
go tool pprof cpu_profile.prof
```

### Запуск в фоне с логированием

```bash
# Запуск в фоне с перенаправлением вывода
sudo nohup ./clientgen -config clientgen_config.json > clientgen.log 2>&1 &

# Мониторинг логов
tail -f clientgen.log
```

### Systemd сервис

Создайте файл `/etc/systemd/system/clientgen.service`:

```ini
[Unit]
Description=ClientGen PTP Traffic Generator
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/clientgen
ExecStart=/opt/clientgen/clientgen -config /opt/clientgen/clientgen_config.json
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Управление сервисом:
```bash
sudo systemctl daemon-reload
sudo systemctl enable clientgen
sudo systemctl start clientgen
sudo systemctl status clientgen
```

## Мониторинг и отладка

### Просмотр статистики

ClientGen выводит статистику в реальном времени:

```
=== Performance Statistics ===
Total Clients: 1000
Packets Sent: 45234
Packets Received: 45100
Active Grants: 950

=== Latency Statistics ===
Announce Grant Latency - Min: 1.2ms, Max: 5.3ms, Avg: 2.1ms
Sync Grant Latency - Min: 0.8ms, Max: 4.1ms, Avg: 1.5ms
```

### Отладочный режим

Для включения подробного вывода:

```json
{
    "DebugPrint": true,
    "DebugLogClient": true,
    "DebugIoWkrRX": true,
    "DebugIoWkrTX": true
}
```

### Анализ результатов

Используйте скрипт анализа:

```bash
./analyze_clientgen.sh
```

## Решение проблем

### Проблема: "pfring.h: No such file or directory"

**Решение**: Установите PF_RING следуя инструкциям:

1. **Установка PF_RING**:
```bash
# Клонирование репозитория
git clone https://github.com/ntop/PF_RING.git
cd PF_RING

# Компиляция kernel module
cd kernel
make
sudo make install

# Компиляция userspace библиотек
cd ../userland/lib
./configure
make
sudo make install

# Компиляция libpcap с поддержкой PF_RING
cd ../libpcap
./configure
make
sudo make install

# Загрузка модуля
sudo modprobe pf_ring
```

2. **Настройка переменных окружения**:
```bash
export CGO_CFLAGS="-I/usr/local/include"
export CGO_LDFLAGS="-L/usr/local/lib -lpfring -lpcap"
export LD_LIBRARY_PATH="/usr/local/lib:$LD_LIBRARY_PATH"
```

3. **Проверка установки**:
```bash
# Запустите скрипт проверки
./check_pfring.sh
```

### Проблема: "Permission denied"

**Решение**: Запускайте с правами root или настройте capabilities:
```bash
sudo setcap cap_net_raw,cap_net_admin=eip ./clientgen
```

### Проблема: "Interface not found"

**Решение**: Проверьте имя интерфейса:
```bash
ip link show
# Используйте правильное имя в конфигурации
```

### Проблема: Высокая потеря пакетов

**Решение**:
1. Увеличьте количество воркеров в конфигурации
2. Проверьте настройки системы (sysctl)
3. Используйте CPU affinity:
```bash
taskset -c 0-7 ./clientgen -config config.json
```

### Проблема: Out of memory

**Решение**:
1. Уменьшите размеры буферов в коде
2. Уменьшите количество клиентов
3. Увеличьте память системы

## Оптимизация производительности

### 1. Настройка CPU affinity

```bash
# Привязка к конкретным ядрам
sudo taskset -c 0-15 ./clientgen -config config.json
```

### 2. Отключение CPU frequency scaling

```bash
# Установка производительного режима
sudo cpupower frequency-set -g performance
```

### 3. Настройка прерываний сетевой карты

```bash
# Распределение прерываний по ядрам
sudo set_irq_affinity.sh eth0
```

### 4. Huge pages

```bash
# Включение huge pages
echo 1024 | sudo tee /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages
```

## Дополнительные ресурсы

- [PF_RING Documentation](https://www.ntop.org/products/packet-capture/pf_ring/)
- [Go Performance Tips](https://github.com/golang/go/wiki/Performance)
- [PTP Protocol Specification](https://www.ieee802.org/1588/)

## Поддержка

При возникновении проблем:
1. Проверьте логи системы: `journalctl -xe`
2. Включите debug режим в конфигурации
3. Соберите информацию о системе: `uname -a`, `go version`, `lsmod | grep pf_ring`
4. Создайте issue в репозитории с подробным описанием проблемы