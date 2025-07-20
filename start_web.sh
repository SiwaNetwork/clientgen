#!/bin/bash

# ClientGen Web Interface Startup Script
# Copyright (c) Facebook, Inc. and its affiliates.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default port
PORT=${1:-8080}

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}  ClientGen Web Interface${NC}"
echo -e "${BLUE}================================${NC}"
echo

# Check if web files exist
if [ ! -d "web" ]; then
    echo -e "${RED}❌ Ошибка: Директория 'web' не найдена${NC}"
    echo -e "${YELLOW}💡 Убедитесь, что вы запускаете скрипт из корневой директории проекта${NC}"
    exit 1
fi

if [ ! -f "web/templates/index.html" ]; then
    echo -e "${RED}❌ Ошибка: Файл index.html не найден${NC}"
    exit 1
fi

# Check if webserver exists, build if not
if [ ! -f "webserver" ]; then
    echo -e "${YELLOW}🔨 Веб-сервер не найден, выполняется сборка...${NC}"
    if command -v make >/dev/null 2>&1; then
        make build-web
    else
        echo -e "${YELLOW}⚠️  Make не найден, выполняется сборка напрямую...${NC}"
        env -u CGO_CFLAGS -u CGO_LDFLAGS go build -o webserver webserver.go
    fi
    
    if [ ! -f "webserver" ]; then
        echo -e "${RED}❌ Ошибка: Не удалось собрать веб-сервер${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Веб-сервер успешно собран${NC}"
fi

# Check if port is available
if command -v netstat >/dev/null 2>&1; then
    if netstat -tuln | grep ":$PORT " >/dev/null; then
        echo -e "${RED}❌ Ошибка: Порт $PORT уже используется${NC}"
        echo -e "${YELLOW}💡 Попробуйте другой порт: $0 <port>${NC}"
        exit 1
    fi
elif command -v ss >/dev/null 2>&1; then
    if ss -tuln | grep ":$PORT " >/dev/null; then
        echo -e "${RED}❌ Ошибка: Порт $PORT уже используется${NC}"
        echo -e "${YELLOW}💡 Попробуйте другой порт: $0 <port>${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}🚀 Запуск веб-сервера на порту $PORT...${NC}"
echo -e "${BLUE}📱 Откройте браузер и перейдите по адресу:${NC}"
echo -e "${GREEN}   http://localhost:$PORT${NC}"
echo
echo -e "${YELLOW}📋 Возможности интерфейса:${NC}"
echo -e "   • 🏠 Панель управления - контроль состояния системы"
echo -e "   • ⚙️  Конфигурация - настройка параметров"
echo -e "   • 📊 Статистика - подробные метрики"
echo -e "   • 📈 Производительность - графики в реальном времени"
echo
echo -e "${YELLOW}🛑 Для остановки нажмите Ctrl+C${NC}"
echo

# Start the web server
./webserver $PORT