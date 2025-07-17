#!/bin/bash

# Скрипт для проверки установки и настройки PF_RING

echo "=== PF_RING Installation Check ==="
echo

# Проверка загруженного модуля
echo "1. Checking PF_RING kernel module..."
if lsmod | grep -q pf_ring; then
    echo "   ✓ PF_RING module is loaded"
    echo "   Module info:"
    lsmod | grep pf_ring
else
    echo "   ✗ PF_RING module is NOT loaded"
    echo "   Try: sudo modprobe pf_ring"
fi
echo

# Проверка /proc/net/pf_ring
echo "2. Checking PF_RING proc interface..."
if [ -d "/proc/net/pf_ring" ]; then
    echo "   ✓ PF_RING proc interface exists"
    if [ -r "/proc/net/pf_ring/info" ]; then
        echo "   PF_RING info:"
        cat /proc/net/pf_ring/info | sed 's/^/   /'
    fi
else
    echo "   ✗ PF_RING proc interface NOT found"
fi
echo

# Проверка библиотек
echo "3. Checking PF_RING libraries..."
if ldconfig -p | grep -q libpfring; then
    echo "   ✓ PF_RING libraries found:"
    ldconfig -p | grep libpfring | sed 's/^/   /'
else
    echo "   ✗ PF_RING libraries NOT found"
    echo "   Make sure to run 'sudo make install' in PF_RING/userland"
fi
echo

# Проверка заголовочных файлов
echo "4. Checking PF_RING headers..."
if [ -f "/usr/local/include/pfring.h" ]; then
    echo "   ✓ PF_RING headers found at /usr/local/include/pfring.h"
elif [ -f "/usr/include/pfring.h" ]; then
    echo "   ✓ PF_RING headers found at /usr/include/pfring.h"
else
    echo "   ✗ PF_RING headers NOT found"
fi
echo

# Проверка сетевого интерфейса
echo "5. Checking network interface capabilities..."
IFACE=$(ip route | grep default | awk '{print $5}' | head -1)
if [ -n "$IFACE" ]; then
    echo "   Default interface: $IFACE"
    
    # Проверка hardware timestamps
    echo "   Hardware timestamp capabilities:"
    if command -v ethtool &> /dev/null; then
        ethtool -T $IFACE 2>/dev/null | grep -E "hardware-transmit|hardware-receive|hardware-raw" | sed 's/^/   /'
    else
        echo "   ethtool not found - cannot check HW timestamp support"
    fi
else
    echo "   ✗ No default network interface found"
fi
echo

# Проверка прав пользователя
echo "6. Checking user capabilities..."
if [ "$EUID" -eq 0 ]; then
    echo "   ✓ Running as root"
else
    echo "   ⚠ Not running as root"
    echo "   Checking CAP_NET_RAW capability..."
    if getcap $(which clientgen) 2>/dev/null | grep -q cap_net_raw; then
        echo "   ✓ clientgen has CAP_NET_RAW capability"
    else
        echo "   ✗ clientgen does NOT have CAP_NET_RAW capability"
        echo "   Try: sudo setcap cap_net_raw+ep $(which clientgen)"
    fi
fi
echo

# Итоговые рекомендации
echo "=== Recommendations ==="
if ! lsmod | grep -q pf_ring; then
    echo "• Load PF_RING module: sudo modprobe pf_ring"
fi
if ! ldconfig -p | grep -q libpfring; then
    echo "• Install PF_RING libraries: cd PF_RING/userland && make && sudo make install"
fi
echo "• For best performance, consider using PF_RING ZC drivers for your NIC"
echo "• Enable hardware timestamps on your NIC if supported"
echo