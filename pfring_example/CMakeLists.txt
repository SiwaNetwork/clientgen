cmake_minimum_required(VERSION 3.10)
project(pfring_example C)

set(CMAKE_C_STANDARD 99)

# Опции сборки
option(USE_SYSTEM_PFRING "Использовать системную установку PF_RING" OFF)
option(USE_SUBMODULE "Использовать PF_RING как git submodule" OFF)

# Исходники
set(SOURCES src/packet_capture.c)

# Создаем исполняемый файл
add_executable(packet_capture ${SOURCES})

if(USE_SYSTEM_PFRING)
    # Ищем системную установку PF_RING
    find_path(PFRING_INCLUDE_DIR 
        NAMES pfring.h
        PATHS /usr/local/include /usr/include
    )
    
    find_library(PFRING_LIBRARY 
        NAMES pfring
        PATHS /usr/local/lib /usr/lib
    )
    
    if(PFRING_INCLUDE_DIR AND PFRING_LIBRARY)
        message(STATUS "Найден системный PF_RING")
        target_include_directories(packet_capture PRIVATE ${PFRING_INCLUDE_DIR})
        target_link_libraries(packet_capture ${PFRING_LIBRARY})
    else()
        message(FATAL_ERROR "Системный PF_RING не найден")
    endif()
    
elseif(USE_SUBMODULE)
    # Используем PF_RING как submodule
    if(NOT EXISTS "${CMAKE_SOURCE_DIR}/external/PF_RING/userland/lib/pfring.h")
        message(FATAL_ERROR "PF_RING submodule не найден. Выполните:\n"
                "git submodule add https://github.com/ntop/PF_RING.git external/PF_RING\n"
                "git submodule update --init --recursive")
    endif()
    
    # Добавляем пути к заголовкам
    target_include_directories(packet_capture PRIVATE 
        ${CMAKE_SOURCE_DIR}/external/PF_RING/userland/lib
        ${CMAKE_SOURCE_DIR}/external/PF_RING/kernel
    )
    
    # Компилируем PF_RING как часть проекта
    add_library(pfring_static STATIC
        external/PF_RING/userland/lib/pfring.c
        external/PF_RING/userland/lib/pfring_utils.c
    )
    
    target_include_directories(pfring_static PUBLIC
        ${CMAKE_SOURCE_DIR}/external/PF_RING/userland/lib
        ${CMAKE_SOURCE_DIR}/external/PF_RING/kernel
    )
    
    target_link_libraries(packet_capture pfring_static)
    
else()
    # Используем локальную копию из ../PF_RING
    set(PFRING_PATH "${CMAKE_SOURCE_DIR}/../PF_RING")
    
    if(NOT EXISTS "${PFRING_PATH}/userland/lib/pfring.h")
        message(FATAL_ERROR "PF_RING не найден в ${PFRING_PATH}")
    endif()
    
    # Добавляем пути к заголовкам
    target_include_directories(packet_capture PRIVATE 
        ${PFRING_PATH}/userland/lib
        ${PFRING_PATH}/kernel
    )
    
    # Ищем скомпилированную библиотеку
    find_library(PFRING_LIB
        NAMES pfring
        PATHS ${PFRING_PATH}/userland/lib
        NO_DEFAULT_PATH
    )
    
    if(PFRING_LIB)
        target_link_libraries(packet_capture ${PFRING_LIB})
    else()
        # Компилируем сами
        add_library(pfring_local STATIC
            ${PFRING_PATH}/userland/lib/pfring.c
            ${PFRING_PATH}/userland/lib/pfring_utils.c
        )
        
        target_include_directories(pfring_local PUBLIC
            ${PFRING_PATH}/userland/lib
            ${PFRING_PATH}/kernel
        )
        
        target_link_libraries(packet_capture pfring_local)
    endif()
endif()

# Общие библиотеки
find_package(Threads REQUIRED)
target_link_libraries(packet_capture 
    Threads::Threads
    pcap
    rt
)

# Флаги компиляции
target_compile_options(packet_capture PRIVATE -Wall -O2)

# Установка
install(TARGETS packet_capture DESTINATION bin)

# Вывод информации о конфигурации
message(STATUS "Конфигурация сборки:")
message(STATUS "  USE_SYSTEM_PFRING: ${USE_SYSTEM_PFRING}")
message(STATUS "  USE_SUBMODULE: ${USE_SUBMODULE}")