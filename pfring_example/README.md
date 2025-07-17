# Пример интеграции PF_RING

Этот проект демонстрирует различные способы интеграции PF_RING в ваше приложение.

## Структура проекта

```
pfring_example/
├── src/
│   └── packet_capture.c    # Пример программы захвата пакетов
├── external/               # Директория для локальных копий PF_RING
├── CMakeLists.txt         # Файл сборки CMake
├── Makefile              # Традиционный Makefile
└── README.md            # Этот файл
```

## Способы сборки

### 1. Используя Makefile с внешним PF_RING

```bash
# Предполагается, что PF_RING клонирован в ../PF_RING
cd /workspace
# (PF_RING уже клонирован)

# Сборка библиотеки PF_RING
cd PF_RING/userland/lib
./configure
make

# Сборка примера
cd /workspace/pfring_example
make
```

### 2. Локальная копия PF_RING

```bash
cd /workspace/pfring_example
make local
```

Эта команда скопирует необходимые файлы PF_RING в директорию `external/pfring/` и соберет программу с ними.

### 3. Используя CMake

```bash
cd /workspace/pfring_example
mkdir build && cd build

# Вариант 1: Использовать PF_RING из ../PF_RING
cmake ..
make

# Вариант 2: Использовать системный PF_RING
cmake -DUSE_SYSTEM_PFRING=ON ..
make

# Вариант 3: Использовать как git submodule
cmake -DUSE_SUBMODULE=ON ..
make
```

### 4. Git Submodule (рекомендуется для production)

```bash
cd /workspace/pfring_example
git init  # если еще не git репозиторий
git submodule add https://github.com/ntop/PF_RING.git external/PF_RING
git submodule update --init --recursive

# Сборка с CMake
mkdir build && cd build
cmake -DUSE_SUBMODULE=ON ..
make
```

## Использование программы

```bash
# Запуск требует прав root для доступа к сетевому интерфейсу
sudo ./packet_capture eth0

# Или для другого интерфейса
sudo ./packet_capture wlan0
```

## Функциональность примера

Программа `packet_capture` демонстрирует:
- Инициализацию PF_RING
- Захват пакетов с сетевого интерфейса
- Базовый анализ пакетов (IP адреса, протоколы)
- Вывод статистики
- Корректное завершение работы

## Требования

- Linux (с поддержкой PF_RING)
- GCC или совместимый компилятор C
- libpcap-dev
- pthread
- CMake 3.10+ (для сборки через CMake)

## Установка зависимостей

### Ubuntu/Debian:
```bash
sudo apt-get install build-essential libpcap-dev cmake
```

### CentOS/RHEL:
```bash
sudo yum install gcc make libpcap-devel cmake
```

## Примечания по производительности

- Для максимальной производительности используйте ZC (Zero Copy) драйверы
- Запускайте с привязкой к CPU core: `taskset -c 1 ./packet_capture zc:eth0`
- Отключите гиперпоточность для критичных к производительности приложений

## Лицензия

Пример кода распространяется под MIT лицензией.
PF_RING имеет двойную лицензию: GPLv2 для модуля ядра, LGPLv2.1 для библиотеки.