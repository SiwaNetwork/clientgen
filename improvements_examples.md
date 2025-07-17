# Примеры улучшений кода ClientGen

## 1. Добавление Graceful Shutdown

### Текущая реализация (main.go)
```go
func main() {
    // ... код инициализации ...
    
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(float64(time.Second)*testConfig.TimeoutSec))
    defer cancel()
    
    // ... запуск ...
    fmt.Println("Done!")
}
```

### Улучшенная реализация
```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    // ... другие импорты ...
)

func main() {
    // ... код инициализации ...
    
    // Создание контекста с отменой
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Обработка сигналов для graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    // Запуск обработчика сигналов
    go func() {
        sig := <-sigChan
        log.Infof("Received signal: %v, initiating graceful shutdown...", sig)
        cancel()
    }()
    
    // Создание контекста с таймаутом
    timeoutCtx, timeoutCancel := context.WithTimeout(ctx, time.Duration(float64(time.Second)*testConfig.TimeoutSec))
    defer timeoutCancel()
    
    errg, errCtx := errgroup.WithContext(timeoutCtx)
    testConfig.Eg = errg
    testConfig.Ctx = &errCtx
    
    log.Infof("Starting clientgen!")
    
    // Запуск с обработкой ошибок
    if err := genlib.StartClientGen(testConfig); err != nil {
        log.Errorf("ClientGen failed: %v", err)
        os.Exit(1)
    }
    
    // Ожидание завершения всех горутин
    if err := errg.Wait(); err != nil {
        log.Errorf("Error during execution: %v", err)
        os.Exit(1)
    }
    
    log.Info("ClientGen shutdown completed successfully")
}
```

## 2. Валидация конфигурации

### Новый файл: clientgenlib/config_validator.go
```go
package clientgenlib

import (
    "fmt"
    "net"
    "time"
)

// ValidateConfig проверяет корректность конфигурации
func (cfg *ClientGenConfig) Validate() error {
    // Проверка сетевого интерфейса
    if cfg.Iface == "" {
        return fmt.Errorf("interface name (Iface) is required")
    }
    
    // Проверка MAC адреса сервера
    if _, err := net.ParseMAC(cfg.ServerMAC); err != nil {
        return fmt.Errorf("invalid server MAC address: %v", err)
    }
    
    // Проверка IP адресов
    serverIP := net.ParseIP(cfg.ServerAddress)
    if serverIP == nil {
        return fmt.Errorf("invalid server IP address: %s", cfg.ServerAddress)
    }
    
    startIP := net.ParseIP(cfg.ClientIPStart)
    if startIP == nil {
        return fmt.Errorf("invalid client start IP: %s", cfg.ClientIPStart)
    }
    
    endIP := net.ParseIP(cfg.ClientIPEnd)
    if endIP == nil {
        return fmt.Errorf("invalid client end IP: %s", cfg.ClientIPEnd)
    }
    
    // Проверка диапазона IP
    if !IpBetween(startIP, endIP, startIP) {
        return fmt.Errorf("client IP start must be less than or equal to end IP")
    }
    
    // Проверка временных параметров
    if cfg.TimeoutSec <= 0 {
        return fmt.Errorf("timeout must be positive")
    }
    
    if cfg.DurationSec <= 0 {
        return fmt.Errorf("duration must be positive")
    }
    
    if cfg.DurationSec >= cfg.TimeoutSec {
        return fmt.Errorf("duration must be less than timeout")
    }
    
    // Проверка параметров производительности
    if cfg.NumTXWorkers <= 0 {
        return fmt.Errorf("NumTXWorkers must be positive")
    }
    
    if cfg.NumRXWorkers <= 0 {
        return fmt.Errorf("NumRXWorkers must be positive")
    }
    
    if cfg.NumPacketParsers <= 0 {
        return fmt.Errorf("NumPacketParsers must be positive")
    }
    
    if cfg.NumPacketProcessors <= 0 {
        return fmt.Errorf("NumPacketProcessors must be positive")
    }
    
    // Проверка размеров очередей (если добавлены в конфигурацию)
    if cfg.RawOutQueueSize > 0 && cfg.RawOutQueueSize < 100 {
        log.Warnf("RawOutQueueSize is very small (%d), may cause performance issues", cfg.RawOutQueueSize)
    }
    
    return nil
}

// SetDefaults устанавливает значения по умолчанию для неуказанных параметров
func (cfg *ClientGenConfig) SetDefaults() {
    if cfg.ClientIPStep == 0 {
        cfg.ClientIPStep = 1
    }
    
    if cfg.TimeBetweenDelayReqSec == 0 {
        cfg.TimeBetweenDelayReqSec = 1
    }
    
    if cfg.ClientRetranTimeWhenNoResponseSec == 0 {
        cfg.ClientRetranTimeWhenNoResponseSec = 1
    }
    
    if cfg.CounterPrintIntervalSecs == 0 {
        cfg.CounterPrintIntervalSecs = 1
    }
    
    // Установка размеров очередей по умолчанию
    if cfg.RawOutQueueSize == 0 {
        cfg.RawOutQueueSize = 10000
    }
    
    if cfg.RawInQueueSize == 0 {
        cfg.RawInQueueSize = 10000
    }
}
```

### Использование в main.go
```go
// После загрузки конфигурации
decoder := json.NewDecoder(file)
err = decoder.Decode(&testConfig)
if err != nil {
    log.Errorf("Failed to decode config file err %v", err)
    return
}

// Установка значений по умолчанию
testConfig.SetDefaults()

// Валидация конфигурации
if err := testConfig.Validate(); err != nil {
    log.Errorf("Invalid configuration: %v", err)
    return
}
```

## 3. Использование пулов объектов

### Улучшенный clientgenlib/client.go
```go
package clientgenlib

import (
    "sync"
    // ... другие импорты ...
)

// Пулы объектов для уменьшения нагрузки на GC
var (
    outPacketPool = sync.Pool{
        New: func() interface{} {
            return &outPacket{
                data: new(gopacket.SerializeBuffer),
            }
        },
    }
    
    inPacketPool = sync.Pool{
        New: func() interface{} {
            return &inPacket{
                data: make([]byte, 1500), // MTU size
            }
        },
    }
    
    pktDecoderPool = sync.Pool{
        New: func() interface{} {
            return &PktDecoder{}
        },
    }
)

// Использование пула в коде
func (cfg *ClientGenConfig) getOutPacket() *outPacket {
    pkt := outPacketPool.Get().(*outPacket)
    pkt.data.Clear() // Очистка буфера
    pkt.getTS = false
    pkt.pktType = pktIgnore
    pkt.cl = nil
    return pkt
}

func (cfg *ClientGenConfig) putOutPacket(pkt *outPacket) {
    outPacketPool.Put(pkt)
}

// Пример использования в функции
func singleClientSendRequest(cfg *ClientGenConfig, cl *SingleClientGen, reqType int) error {
    // Получение объекта из пула
    out := cfg.getOutPacket()
    defer cfg.putOutPacket(out) // Возврат в пул
    
    // ... использование out ...
    
    return nil
}
```

## 4. Улучшенная обработка ошибок

### Новый файл: clientgenlib/errors.go
```go
package clientgenlib

import (
    "fmt"
    "time"
)

// Типы ошибок
type ErrorType int

const (
    ErrTypeNetwork ErrorType = iota
    ErrTypeProtocol
    ErrTypeTimeout
    ErrTypeResource
)

// ClientGenError представляет ошибку с контекстом
type ClientGenError struct {
    Type      ErrorType
    Operation string
    ClientIP  string
    Err       error
    Timestamp time.Time
}

func (e *ClientGenError) Error() string {
    return fmt.Sprintf("[%s] %s error for client %s in operation %s: %v",
        e.Timestamp.Format(time.RFC3339),
        e.typeString(),
        e.ClientIP,
        e.Operation,
        e.Err)
}

func (e *ClientGenError) typeString() string {
    switch e.Type {
    case ErrTypeNetwork:
        return "Network"
    case ErrTypeProtocol:
        return "Protocol"
    case ErrTypeTimeout:
        return "Timeout"
    case ErrTypeResource:
        return "Resource"
    default:
        return "Unknown"
    }
}

// Retry механизм с экспоненциальной задержкой
type RetryConfig struct {
    MaxAttempts     int
    InitialDelay    time.Duration
    MaxDelay        time.Duration
    BackoffFactor   float64
}

func RetryWithBackoff(cfg RetryConfig, operation func() error) error {
    var lastErr error
    delay := cfg.InitialDelay
    
    for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
        if err := operation(); err == nil {
            return nil
        } else {
            lastErr = err
            
            if attempt < cfg.MaxAttempts {
                log.Warnf("Operation failed (attempt %d/%d): %v, retrying in %v",
                    attempt, cfg.MaxAttempts, err, delay)
                
                time.Sleep(delay)
                
                // Увеличение задержки
                delay = time.Duration(float64(delay) * cfg.BackoffFactor)
                if delay > cfg.MaxDelay {
                    delay = cfg.MaxDelay
                }
            }
        }
    }
    
    return fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// Пример использования
func (cfg *ClientGenConfig) sendPacketWithRetry(pkt *outPacket) error {
    retryConfig := RetryConfig{
        MaxAttempts:   3,
        InitialDelay:  100 * time.Millisecond,
        MaxDelay:      5 * time.Second,
        BackoffFactor: 2.0,
    }
    
    return RetryWithBackoff(retryConfig, func() error {
        // Отправка пакета
        if err := cfg.sendPacket(pkt); err != nil {
            return &ClientGenError{
                Type:      ErrTypeNetwork,
                Operation: "sendPacket",
                ClientIP:  pkt.cl.ClientIP.String(),
                Err:       err,
                Timestamp: time.Now(),
            }
        }
        return nil
    })
}
```

## 5. Метрики производительности

### Новый файл: clientgenlib/metrics.go
```go
package clientgenlib

import (
    "sync/atomic"
    "time"
)

// MetricsCollector собирает метрики производительности
type MetricsCollector struct {
    // Счетчики
    PacketsSent      atomic.Uint64
    PacketsReceived  atomic.Uint64
    PacketsDropped   atomic.Uint64
    ErrorsTotal      atomic.Uint64
    
    // Латентность (в наносекундах)
    latencyBuckets   []atomic.Uint64 // Гистограмма
    bucketBoundaries []time.Duration
    
    // Текущие значения
    ActiveClients    atomic.Int32
    QueueDepth       atomic.Int32
}

// NewMetricsCollector создает новый коллектор метрик
func NewMetricsCollector() *MetricsCollector {
    boundaries := []time.Duration{
        100 * time.Microsecond,
        500 * time.Microsecond,
        1 * time.Millisecond,
        5 * time.Millisecond,
        10 * time.Millisecond,
        50 * time.Millisecond,
        100 * time.Millisecond,
        500 * time.Millisecond,
        1 * time.Second,
    }
    
    return &MetricsCollector{
        bucketBoundaries: boundaries,
        latencyBuckets:   make([]atomic.Uint64, len(boundaries)+1),
    }
}

// RecordLatency записывает значение латентности
func (m *MetricsCollector) RecordLatency(latency time.Duration) {
    bucketIndex := len(m.bucketBoundaries)
    for i, boundary := range m.bucketBoundaries {
        if latency <= boundary {
            bucketIndex = i
            break
        }
    }
    m.latencyBuckets[bucketIndex].Add(1)
}

// GetMetricsSnapshot возвращает снимок текущих метрик
func (m *MetricsCollector) GetMetricsSnapshot() MetricsSnapshot {
    snapshot := MetricsSnapshot{
        Timestamp:       time.Now(),
        PacketsSent:     m.PacketsSent.Load(),
        PacketsReceived: m.PacketsReceived.Load(),
        PacketsDropped:  m.PacketsDropped.Load(),
        ErrorsTotal:     m.ErrorsTotal.Load(),
        ActiveClients:   int(m.ActiveClients.Load()),
        QueueDepth:      int(m.QueueDepth.Load()),
        LatencyHistogram: make(map[string]uint64),
    }
    
    // Копирование гистограммы латентности
    for i, count := range m.latencyBuckets {
        var label string
        if i < len(m.bucketBoundaries) {
            label = fmt.Sprintf("≤%v", m.bucketBoundaries[i])
        } else {
            label = fmt.Sprintf(">%v", m.bucketBoundaries[len(m.bucketBoundaries)-1])
        }
        snapshot.LatencyHistogram[label] = count.Load()
    }
    
    return snapshot
}

// MetricsSnapshot представляет снимок метрик
type MetricsSnapshot struct {
    Timestamp        time.Time
    PacketsSent      uint64
    PacketsReceived  uint64
    PacketsDropped   uint64
    ErrorsTotal      uint64
    ActiveClients    int
    QueueDepth       int
    LatencyHistogram map[string]uint64
}

// Интеграция с Prometheus (опционально)
func (m *MetricsCollector) ServePrometheusMetrics() {
    // Здесь можно добавить HTTP endpoint для Prometheus
}
```

## 6. Абстракция сетевого уровня

### Новый файл: clientgenlib/network_interface.go
```go
package clientgenlib

import (
    "net"
    "time"
)

// NetworkInterface определяет интерфейс для работы с сетью
type NetworkInterface interface {
    // Инициализация
    Init(iface string, config NetworkConfig) error
    Close() error
    
    // Отправка и получение
    SendPacket(data []byte, needsTimestamp bool) error
    ReceivePacket() (data []byte, timestamp time.Time, err error)
    
    // Получение timestamp'ов
    GetTXTimestamp() (timestamp time.Time, err error)
    
    // Статистика
    GetStats() NetworkStats
}

// NetworkConfig содержит конфигурацию сети
type NetworkConfig struct {
    NumRXQueues    int
    NumTXQueues    int
    RXBufferSize   int
    TXBufferSize   int
    EnableHWTimestamp bool
}

// NetworkStats содержит статистику сети
type NetworkStats struct {
    PacketsReceived uint64
    PacketsSent     uint64
    BytesReceived   uint64
    BytesSent       uint64
    Errors          uint64
    Dropped         uint64
}

// PFRingNetwork реализация для PF_RING
type PFRingNetwork struct {
    // ... поля для PF_RING ...
}

func (p *PFRingNetwork) Init(iface string, config NetworkConfig) error {
    // Инициализация PF_RING
    return nil
}

// StandardSocketNetwork реализация на стандартных сокетах
type StandardSocketNetwork struct {
    conn     *net.UDPConn
    iface    string
    config   NetworkConfig
}

func (s *StandardSocketNetwork) Init(iface string, config NetworkConfig) error {
    // Инициализация стандартного сокета
    s.iface = iface
    s.config = config
    
    // Создание UDP сокета
    addr, err := net.ResolveUDPAddr("udp", ":319") // PTP event port
    if err != nil {
        return err
    }
    
    s.conn, err = net.ListenUDP("udp", addr)
    if err != nil {
        return err
    }
    
    // Настройка буферов
    if err := s.conn.SetReadBuffer(config.RXBufferSize); err != nil {
        return err
    }
    
    if err := s.conn.SetWriteBuffer(config.TXBufferSize); err != nil {
        return err
    }
    
    return nil
}

// NetworkFactory создает правильную реализацию
func NewNetworkInterface(backend string) (NetworkInterface, error) {
    switch backend {
    case "pfring":
        return &PFRingNetwork{}, nil
    case "socket":
        return &StandardSocketNetwork{}, nil
    default:
        return nil, fmt.Errorf("unknown network backend: %s", backend)
    }
}
```

## Заключение

Эти примеры демонстрируют основные направления улучшения кода:

1. **Надежность**: Graceful shutdown, обработка ошибок, retry механизмы
2. **Производительность**: Пулы объектов, метрики, оптимизация памяти
3. **Поддерживаемость**: Валидация конфигурации, абстракции, модульность
4. **Тестируемость**: Интерфейсы для моков, разделение ответственности

Внедрение этих улучшений сделает код более профессиональным, надежным и готовым к production использованию.