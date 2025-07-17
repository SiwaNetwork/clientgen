#!/bin/bash

# Скрипт для настройки PF_RING как git submodule

echo "=== Настройка PF_RING как git submodule ==="

# Проверяем, что мы в git репозитории
if [ ! -d .git ]; then
    echo "Инициализируем git репозиторий..."
    git init
fi

# Проверяем, существует ли уже submodule
if [ -d "external/PF_RING/.git" ]; then
    echo "PF_RING submodule уже существует"
else
    echo "Добавляем PF_RING как submodule..."
    git submodule add https://github.com/ntop/PF_RING.git external/PF_RING
fi

# Обновляем submodule
echo "Обновляем submodule..."
git submodule update --init --recursive

# Собираем библиотеку PF_RING
echo "Собираем библиотеку PF_RING..."
cd external/PF_RING/userland/lib
./configure --prefix=$(pwd)/../../install
make
make install

cd ../../../../

# Создаем .gitignore если не существует
if [ ! -f .gitignore ]; then
    echo "Создаем .gitignore..."
    cat > .gitignore << EOF
# Собранные файлы
packet_capture
*.o
*.a
*.so

# Директории сборки
build/
cmake-build-*/

# Временные файлы
*.swp
*~
.DS_Store

# Локальная установка PF_RING
external/install/
EOF
fi

echo "=== Настройка завершена ==="
echo ""
echo "Теперь вы можете собрать проект:"
echo "  mkdir build && cd build"
echo "  cmake -DUSE_SUBMODULE=ON .."
echo "  make"
echo ""
echo "Или используя Makefile:"
echo "  make local"