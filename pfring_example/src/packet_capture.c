/*
 * Пример интеграции PF_RING для захвата и анализа сетевых пакетов
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <unistd.h>
#include <sys/time.h>
#include <netinet/in.h>
#include <netinet/ip.h>
#include <netinet/tcp.h>
#include <netinet/udp.h>
#include <arpa/inet.h>

// Включаем PF_RING
#include "../external/pfring/pfring.h"

// Глобальные переменные
static int keep_running = 1;
static pfring *ring = NULL;
static u_int64_t num_packets = 0;
static u_int64_t num_bytes = 0;

// Обработчик сигнала для корректного завершения
void sighandler(int sig) {
    printf("\nПолучен сигнал %d, завершаем работу...\n", sig);
    keep_running = 0;
}

// Функция для вывода статистики
void print_stats() {
    pfring_stat stats;
    
    if (pfring_stats(ring, &stats) >= 0) {
        printf("\n=== Статистика ===\n");
        printf("Получено пакетов: %llu\n", (unsigned long long)num_packets);
        printf("Получено байт: %llu\n", (unsigned long long)num_bytes);
        printf("Потеряно пакетов: %llu\n", (unsigned long long)stats.drop);
        printf("==================\n");
    }
}

// Функция для анализа пакета
void analyze_packet(const struct pfring_pkthdr *header, const u_char *packet) {
    struct ether_header *eth_header;
    struct iphdr *ip_header;
    char src_ip[INET_ADDRSTRLEN];
    char dst_ip[INET_ADDRSTRLEN];
    
    // Ethernet заголовок
    eth_header = (struct ether_header *)packet;
    
    // Проверяем, что это IP пакет
    if (ntohs(eth_header->ether_type) == 0x0800) {
        // IP заголовок
        ip_header = (struct iphdr *)(packet + sizeof(struct ether_header));
        
        // Конвертируем IP адреса в строки
        inet_ntop(AF_INET, &(ip_header->saddr), src_ip, INET_ADDRSTRLEN);
        inet_ntop(AF_INET, &(ip_header->daddr), dst_ip, INET_ADDRSTRLEN);
        
        printf("Пакет #%llu: %s -> %s, ", 
               (unsigned long long)num_packets, src_ip, dst_ip);
        
        // Определяем протокол
        switch (ip_header->protocol) {
            case IPPROTO_TCP:
                printf("TCP");
                break;
            case IPPROTO_UDP:
                printf("UDP");
                break;
            case IPPROTO_ICMP:
                printf("ICMP");
                break;
            default:
                printf("Proto=%d", ip_header->protocol);
        }
        
        printf(", размер=%u байт\n", header->len);
    }
}

// Основная функция обработки пакетов
void packet_handler(const struct pfring_pkthdr *header, const u_char *packet, const u_char *user) {
    num_packets++;
    num_bytes += header->len;
    
    // Анализируем каждый 100-й пакет для примера
    if (num_packets % 100 == 0) {
        analyze_packet(header, packet);
    }
    
    // Выводим статистику каждые 10000 пакетов
    if (num_packets % 10000 == 0) {
        print_stats();
    }
}

int main(int argc, char *argv[]) {
    char *device = NULL;
    u_int32_t flags = 0;
    int rc;
    
    // Проверка аргументов
    if (argc != 2) {
        printf("Использование: %s <интерфейс>\n", argv[0]);
        printf("Пример: %s eth0\n", argv[0]);
        return 1;
    }
    
    device = argv[1];
    
    // Устанавливаем обработчик сигналов
    signal(SIGINT, sighandler);
    signal(SIGTERM, sighandler);
    
    printf("Открываем устройство %s для захвата пакетов...\n", device);
    
    // Открываем PF_RING
    flags |= PF_RING_PROMISC;  // Promiscuous режим
    flags |= PF_RING_TIMESTAMP; // Включаем временные метки
    
    ring = pfring_open(device, 1536, flags);
    
    if (ring == NULL) {
        fprintf(stderr, "Ошибка: не удалось открыть устройство %s\n", device);
        fprintf(stderr, "Возможно требуются права root или модуль PF_RING не загружен\n");
        return -1;
    }
    
    // Устанавливаем направление захвата (rx+tx)
    pfring_set_direction(ring, rx_and_tx_direction);
    
    // Включаем кольцо
    rc = pfring_enable_ring(ring);
    if (rc != 0) {
        fprintf(stderr, "Ошибка: не удалось включить ring (%d)\n", rc);
        pfring_close(ring);
        return -1;
    }
    
    printf("Начинаем захват пакетов. Нажмите Ctrl+C для остановки.\n\n");
    
    // Основной цикл обработки пакетов
    while (keep_running) {
        struct pfring_pkthdr hdr;
        u_char *buffer;
        
        rc = pfring_recv(ring, &buffer, 0, &hdr, 1 /* wait_for_packet */);
        
        if (rc > 0) {
            // Обрабатываем пакет
            packet_handler(&hdr, buffer, NULL);
        } else if (rc == 0) {
            // Таймаут, продолжаем
            continue;
        } else {
            // Ошибка
            fprintf(stderr, "Ошибка pfring_recv: %d\n", rc);
            break;
        }
    }
    
    // Выводим финальную статистику
    print_stats();
    
    // Закрываем PF_RING
    pfring_close(ring);
    
    printf("\nПрограмма завершена.\n");
    
    return 0;
}