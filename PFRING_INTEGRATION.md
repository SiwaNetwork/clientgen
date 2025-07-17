# PF_RING Integration Guide

## Обзор

Этот документ описывает полную интеграцию PF_RING в clientgen для высокопроизводительной обработки сетевых пакетов с поддержкой hardware timestamps.

## Изменения в коде

### 1. Обновление структур данных

В файле `clientgenlib/clientData.go` добавлены поля для статистики PF_RING:

```go
type GlobalStatistics struct {
    // ... existing fields ...
    
    // PF_RING statistics
    PFRingRXPackets      uint64
    PFRingRXBytes        uint64
    PFRingRXDropped      uint64
    PFRingTXPackets      uint64
    PFRingTXBytes        uint64
    PFRingHWTimestamps   uint64
}
```

### 2. Улучшения в RX Worker

В файле `clientgenlib/ioWkr.go` внесены следующие изменения:

#### Инициализация PF_RING для приема:
- Увеличен размер буфера с 4096 до 65536 для лучшей производительности
- Добавлен флаг `pfring.FlagLongHeader` для расширенной информации о пакетах
- Оптимизированы параметры для низкой задержки (`SetPollDuration(0)`)

#### Hardware Timestamps:
- Добавлена поддержка чтения hardware timestamps через `ReadPacketDataExtended()`
- Автоматический подсчет пакетов с HW timestamps

#### Статистика:
- Добавлен периодический сбор статистики PF_RING каждые 5 секунд
- Статистика включает количество принятых и потерянных пакетов

### 3. Интеграция TX через PF_RING

#### Создание TX Ring:
```go
txFlags := pfring.FlagPromisc|pfring.FlagHWTimestamp|pfring.FlagLongHeader
txRing, err := pfring.NewRing(cfg.Iface, 65536, txFlags)
```

#### Отправка пакетов:
- Используется `txRing.WritePacketData()` для отправки
- Автоматический fallback на raw socket при ошибках

### 4. Вывод статистики

В файле `clientgenlib/counterProcessor.go` добавлен вывод статистики PF_RING:

```
==PF_RING Statistics=========
PF_RING RX packets: 1234567
PF_RING RX dropped: 0
PF_RING TX packets: 1234567
PF_RING HW timestamps: 1234567
```

## Требования

### Системные требования:
1. PF_RING kernel module должен быть загружен
2. Сетевая карта должна поддерживать hardware timestamps
3. Пользователь должен иметь права для доступа к PF_RING

### Установка PF_RING:

```bash
# Клонирование репозитория
git clone https://github.com/ntop/PF_RING.git
cd PF_RING

# Компиляция kernel module
cd kernel
make
sudo make install

# Компиляция userspace библиотек
cd ../userland
make
sudo make install

# Загрузка модуля
sudo modprobe pf_ring
```

### Настройка сетевой карты:

```bash
# Включение hardware timestamps
sudo ethtool -T eth0

# Настройка RX/TX очередей для оптимальной производительности
sudo ethtool -G eth0 rx 4096 tx 4096

# Отключение offloading для точных timestamps
sudo ethtool -K eth0 gro off gso off tso off
```

## Производительность

### Оптимизации:
1. **Размер буфера**: Увеличен до 65536 байт для снижения потерь пакетов
2. **Poll параметры**: Настроены для минимальной задержки
3. **Hardware timestamps**: Используются когда доступны для точности
4. **Параллельная обработка**: Поддержка нескольких RX/TX workers

### Ожидаемые улучшения:
- Снижение CPU использования на 30-50%
- Уменьшение задержки обработки пакетов
- Точность timestamps до наносекунд (с HW поддержкой)
- Возможность обработки 10+ Mpps на современном оборудовании

## Мониторинг

### Проверка работы PF_RING:

```bash
# Статус модуля
lsmod | grep pf_ring

# Информация о ring
cat /proc/net/pf_ring/info

# Статистика по интерфейсу
cat /proc/net/pf_ring/stats/eth0
```

### Debug режим:

Включите debug флаги в конфигурации для детальной информации:
```json
{
    "DebugPrint": true,
    "DebugIoWkrRX": true,
    "DebugIoWkrTX": true,
    "PrintTxRxCounts": true
}
```

## Troubleshooting

### Проблема: "pfring.h: No such file or directory"
Решение: Установите PF_RING development headers

### Проблема: "Failed to create PF_RING socket"
Решение: 
- Проверьте что модуль pf_ring загружен
- Убедитесь что у пользователя есть права (CAP_NET_RAW)

### Проблема: "No hardware timestamps"
Решение:
- Проверьте поддержку HW timestamps: `ethtool -T eth0`
- Включите timestamps на NIC

## Заключение

Интеграция PF_RING значительно улучшает производительность clientgen, особенно при высоких нагрузках. Hardware timestamps обеспечивают точность измерений, необходимую для PTP протокола.