# Clientgen
Генератор трафика PTP клиентов с открытым исходным кодом, основанный на PF_RING и примере simpleclient.

## Детали работы

Эта утилита предназначена для симуляции большого количества PTP клиентов, которые проходят через следующую последовательность:
1. Запрос Sync Unicast Grant на указанную продолжительность
2. Запрос Announce Unicast Grant на указанную продолжительность
3. Запрос Delay Response Unicast Grant на указанную продолжительность
4. Периодическая отправка запросов DelayResp на сервер, пока Grants активны
5. Потенциальный перезапуск после истечения всех Grants

### Конфигурация
Утилита настраивается через единый конфигурационный файл json, подробно описанный ниже, clientgen_config.json
* Клиенты указываются начальным IP-адресом, конечным IP-адресом и размером шага IP-адреса, аналогично коммерческим генераторам трафика

### Работа

Утилита запускается через CLI и периодически выводит подробную информацию (период указывается в конфигурации), настраиваемую на основе конфигурации.
* Пример вывода CLI приведен ниже. Каждая выводимая секция может быть отключена при желании в конфигурации.

### Входящие пакеты
Утилита основана на конвейерной обработке входящих пакетов следующим образом:
1. RX ioWkr: Чтение входящего пакета из буфера PF_RING (включая RX timestamp), передача его в packetParser
2. packetParser: Парсинг входящего пакета с использованием gopacket для декодирования слоев пакета, передача в packetProcessor
3. packetProcessor:
   * Если пакет ARP (для IPv4) или ICMPv6 (для IPv6), создание ответа на основе конфигурации клиента. Передача ответа в txWkr
   * Если пакет UDP, проверка UDP пакета для определения, предназначен ли он для симулируемого клиента, и создание ответа при необходимости. Передача ответа в txWkr.
   * Если пакет был из TX пути (см. ниже), то этот пакет используется только для определения места хранения timestamp, который пришел с ним (TX timestamp)

Утилита работает, используя низкоуровневое создание пакетов и захват пакетов для контроля каждого отдельного отправленного пакета и обработки каждого полученного пакета.
* Поскольку каждый пакет создается вручную, утилита требует предварительного знания MAC-адреса DUT в конфигурационном файле
* Утилита также требует имя интерфейса, например ens1f0, в конфигурационном файле для привязки сокетов и библиотеки pcap
* PF_RING используется для приема пакетов.
  * Он позволяет распределять все входящие пакеты на интерфейсе по принципу round-robin между произвольным количеством рабочих goroutines
  * Он позволяет использовать гораздо большую буферизацию пакетов, буферизируя пакеты в памяти CPU, а не в буферных пространствах NIC
  * Он убирает опрос NIC из пользовательского пространства, вместо этого напрямую опрашивая NIC в ядре и буферизируя пакеты в буферах пользовательского пространства

### Исходящие пакеты
Утилита имеет txWkrs для обработки отправки пакетов:
* Каждый пакет, отправленный в txWkr, имеет флаг, указывающий, требуется ли его timestamping или нет
* Если timestamping не требуется, пакет отправляется с простым сокетом
* Если timestamping требуется, пакет отправляется на сокет с включенным timestamping.
* Каждый txWkr имеет несколько goroutines, которые опрашивают свой включенный в timestamping сокет, чтобы как можно скорее извлечь TX timestamps
* Для каждого TX timestamp, полный пакет, отправленный, также читается назад. Этот пакет передается в packetParser с флагом, указывающим, что это был отправленный пакет

### Обработка клиентов
Утилита имеет retransmitWorkers для обработки повторных передач пакетов:
* Каждый retransmitWorker поддерживает кучу на основе времени для указания, когда клиент должен быть повторно передан, если ответа не было.
* Каждый элемент кучи связан с определенным клиентом.
* Эта куча используется для повторной передачи grants. Например, если клиент не получил Sync Grant от сервера в течение времени, этот процессор повторно передаст Sync Grant Request
* Он также используется для повторной передачи, если это необходимо. Например, когда клиент имеет все grants, он используется для периодической отправки DelayReq.

Утилита имеет restartWorkers для обработки перезапусков клиентов:
* Каждый restartWorker поддерживает кучу на основе времени, если перезапуск включен в конфигурации
* Когда клиент получает все три grants от сервера, он помещает себя в кучу для перезапуска после истечения всех grants.

## Руководство по установке

### PF_RING
1. Скачайте и установите pf_ring из ntop.org
```console
git clone https://github.com/ntop/PF_RING.git
```
2. Соберите ядро модуля pf_ring
```console
cd PF_RING/kernel
make 
sudo make install
insmod ./pf_ring.ko
```
3. Соберите и установите API библиотеки для pf_ring
```console
cd PF_RING/userland/lib
./configure && make
sudo make install
cd ../libpcap
./configure && make
sudo make install
```
Подробнее можно найти здесь
	* https://www.ntop.org/guides/pf_ring/get_started/git_installation.html

### Pull clientgen
```console
git clone https://github.com/opencomputeproject/Time-Appliance-Project
```

### Build clientgen
1. Убедитесь, что LDD связан с тем, где библиотеки pf_ring находятся
```console
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/usr/local/lib/
```
2. Убедитесь, что путь к заголовочному файлу pf_ring. Измените этот путь на тот, где PF_RING был собран.
```console
C_INCLUDE_PATH=~/ptp/PF_RING/kernel; export C_INCLUDE_PATH
```
3. Перейдите в директорию clientgen и соберите
```console
cd Software/Experimental/clientgen/
go build
```

## Руководство по использованию
Clientgen настраивается конфигурационным файлом json, clientgen_config.json, в той же директории, что и исполняемый файл. Запустите его, запустив Clientgen в директории clientgen. Измените конфигурацию json перед запуском в консоль для выполнения.

```console
cd clientgen/
./clientgen -config clientgen_config.json [-profilelog <cpuprofiler file>]
```

Описание элементов в этом json файле:
### Traffic and client configuration
* "Iface" - интерфейс на сервере для генерации трафика клиентов, например "ens1f0"
* "ServerMAC" - MAC-адрес PTP Grandmaster сервера, например "0c:42:a1:80:31:66"
* "ServerAddress" - IPv4 или IPv6 адрес PTP Grandmaster сервера, например "10.254.254.254"
* "ClientIPStart" - IPv4 или IPv6. Для диапазона клиентов, это IP-адрес первого клиента. Например "10.1.1.2"
* "ClientIPEnd" - IPv4 или IPv6. Для диапазона клиентов, это последний IP-адрес клиента. Например "10.1.1.10"
* "ClientIPStep" - Для генерации клиентов, насколько увеличивать ClientIPStart для каждого клиента. Если ClientIPStart равен 10.1.1.2, а это 2, то будут сгенерированы клиенты 10.1.1.2 -> 10.1.1.4 -> 10.1.1.6 -> 10.1.1.8 и т.д. до ClientIPEnd
* "SoftStartRate" - Максимальное количество клиентов для запуска в секунду
* "TimeoutSec" - сколько секунд запустить clientgen, после чего программа остановит генерацию трафика.
* "DurationSec" - Продолжительность Grant каждого клиента при попытке подписаться на PTP grandmaster при запросе UDP Grants, например Sync/Announce/DelayResp.
* "TimeAfterDurationBeforeRestartSec" - Время после истечения последнего grant клиента для ожидания перед перезапуском клиента в секундах
* "TimeBetweenDelayReqSec" - После того, как клиент имеет все свои grants, сколько DelayReqs отправлять в секунду
* "ClientRetranTimeWhenNoResponseSec" - Сколько секунд клиент должен ждать при запросе grant перед повторной передачей запроса grant, если ответа не получено в секундах
### Performance controls
* "NumTXWorkers" - Сколько goroutines запускать для обработки отправки пакетов. Это может быть главным узким местом, из-за производительности timestamping TX.
* "NumTXTSWorkerPerTx" - Сколько goroutines запускать на TX worker для чтения TX timestamps.
* "NumRXWorkers" - Сколько goroutines запускать для обработки RX работы. 
* "NumPacketParsers" - Сколько goroutines запускать для парсинга каждого полученного пакета в формат gopacket для внутреннего использования.
* "NumPacketProcessors" - Сколько goroutines запускать для обработки каждого парсированного пакета и возможной генерации ответа.
* "NumClientRetransmitProcs" - Сколько goroutines запускать для управления внутренними таймерами для возможной повторной передачи для каждого клиента. Работа на goroutine будет масштабироваться с количеством клиентов.
* "NumClientRestartProcs" - Сколько goroutines запускать для управления внутренними таймерами для перезапуска клиентов после истечения их grants. Работа на goroutine будет масштабироваться с количеством клиентов.
### Debug logging
* Включайте только для разработки и отладки, должны быть выключены в большинстве случаев.
### Периодическая печать статистики
* "PrintPerformance" - Печатает процент занятости каждого рабочего goroutine. Используйте это, чтобы помочь настроить Performance Controls выше, чтобы получить желаемую производительность.
* "PrintClientData" - Печатает информацию обо всех клиентах, например общее количество запросов Announce, общее количество полученных Announce Grants
* "PrintTxRxCounts" - Печатает простые счетчики TX и RX пакетов
* "PrintClientReqData" - Печатает гистограмму информации для Announce Requests / Sync Requests / Delay Response Grant Requests / Delay Requests для всех клиентов
* "PrintLatencyData" - Печатает статистическую информацию о латентности сервера при ответе на Announce Requests / Sync Requests / Delay Response Grant Requests / Delay Requests , а также статистическую информацию о времени между Sync пакетами от grandmaster.
* "CounterPrintIntervalSecs" - Сколько секунд печатать включенные статистики

## Пример вывода CLI
При запуске со всеми печатными, это пример того, что охватывают статистические данные.

```console
========================Statistics after 999.684ms============
==ClientData=============
0: TotalClients = 228, rate 0
1: TotalPacketsSent = 912, rate 912
2: TotalPacketsRcvd = 1596, rate 1596
3: TotalTXTSPacketsSent = 912, rate 912
4: TotalTXTSRead = 912, rate 912
5: MaxTXTSBytesOutstanding = 3332, rate 3332
6: TotalGenMsgSent = 684, rate 684
7: TotalGenMsgRcvd = 1140, rate 1140
8: TotalEventMsgSent = 228, rate 228
9: TotalEventMsgRcvd = 456, rate 456
10: TotalClientAnnounceReq = 228, rate 228
11: TotalClientAnnounceReqResend = 0, rate 0
12: TotalClientAnnounceGrant = 228, rate 228
13: TotalClientSyncReq = 228, rate 228
14: TotalClientSyncReqResend = 0, rate 0
15: TotalClientSyncGrant = 228, rate 228
16: TotalClientDelayRespReq = 228, rate 228
17: TotalClientDelayRespReqResend = 0, rate 0
18: TotalClientDelayRespGrant = 228, rate 228
19: TotalSyncRcvd = 228, rate 228
20: TotalPDelayRespRcvd = 0, rate 0
21: TotalFollowUpRcvd = 228, rate 228
22: TotalPDelayRespFollowUpRcvd = 0, rate 0
23: TotalAnnounceRcvd = 228, rate 228
24: TotalDelayReqSent = 228, rate 228
25: TotalDelayRespRcvd = 228, rate 228
26: TotalRetransmitHeapAdd = 0, rate 0
27: TotalRetransmitHeapAddAlreadyIn = 0, rate 0
28: TotalRetransmitHeapAddNotIn = 0, rate 0
29: TotalRetransmitHeapTryRemove = 0, rate 0
30: TotalRetransmitHeapRemove = 0, rate 0
31: TotalRetransmitHeapPop = 228, rate 228
Client states
Total: 228, Max: 1 , Min: 1
Count 1:228
==Tx Rx Counters=============
TX worker 0 pkt send: 912
RX worker 0 pkt recv: 1596
==Client Request Data============
Announce Grant Requests sent
Total: 228, Max: 1 , Min: 1
Count 1:228
Sync Grant Requests sent
Total: 228, Max: 1 , Min: 1
Count 1:228
Delay Resp Grant Requests sent
Total: 228, Max: 1 , Min: 1
Count 1:228
Delay Requests sent
Total: 228, Max: 1 , Min: 1
Count 1:228
Announce Grant Latency
 228 samples of 228 events
Cumulative:	320.040312ms
HMean:		1.024843ms
Avg.:		1.403685ms
p50: 		1.16235ms
p75:		1.683062ms
p95:		3.27251ms
p99:		4.523026ms
p999:		5.568122ms
Long 5%:	4.00771ms
Short 5%:	471.566µs
Max:		5.568122ms
Min:		307.806µs
Range:		5.260316ms
StdDev:		888.435µs
Rate/sec.:	712.41
Sync Grant Latency
 228 samples of 228 events
Cumulative:	300.154828ms
HMean:		247.17µs
Avg.:		1.316468ms
p50: 		1.18881ms
p75:		1.857054ms
p95:		3.285182ms
p99:		4.467874ms
p999:		5.387614ms
Long 5%:	3.841688ms
Short 5%:	42.421µs
Max:		5.387614ms
Min:		35.566µs
Range:		5.352048ms
StdDev:		1.003697ms
Rate/sec.:	759.61
Delay Resp Grant Latency
 228 samples of 228 events
Cumulative:	229.549872ms
HMean:		145.619µs
Avg.:		1.006797ms
p50: 		1.095026ms
p75:		1.194654ms
p95:		2.357494ms
p99:		3.319574ms
p999:		3.398314ms
Long 5%:	3.005904ms
Short 5%:	29.675µs
Max:		3.398314ms
Min:		22.114µs
Range:		3.3762ms
StdDev:		814.495µs
Rate/sec.:	993.25
Delay Req Latency
 228 samples of 228 events
Cumulative:	173.381584ms
HMean:		127.768µs
Avg.:		760.445µs
p50: 		997.978µs
p75:		1.193058ms
p95:		2.155666ms
p99:		2.336506ms
p999:		3.223606ms
Long 5%:	2.408821ms
Short 5%:	29.538µs
Max:		3.223606ms
Min:		24.546µs
Range:		3.19906ms
StdDev:		672.425µs
Rate/sec.:	1315.02
Time Between Syncs
 0 samples of 0 events
Cumulative:	0s
HMean:		0s
Avg.:		0s
p50: 		0s
p75:		0s
p95:		0s
p99:		0s
p999:		0s
Long 5%:	0s
Short 5%:	0s
Max:		0s
Min:		0s
Range:		0s
StdDev:		0s
Rate/sec.:	0.00
==Software Performance=============
time="2021-08-24T10:46:40-07:00" level=info msg="Profiler RX Worker 0 last busy 0.50%"
time="2021-08-24T10:46:40-07:00" level=info msg="Profiler PacketParser 0 last busy 0.50%"
time="2021-08-24T10:46:40-07:00" level=info msg="Profiler TX worker 0 last busy 0.50%"
time="2021-08-24T10:46:40-07:00" level=info msg="Profiler TX worker 0 TSRead worker 0 last busy 0.50%"
time="2021-08-24T10:46:40-07:00" level=info msg="Profiler PacketProcessor 0 last busy 0.50%"
time="2021-08-24T10:46:40-07:00" level=info msg="Profiler CounterProcessor last busy 0.00%"
time="2021-08-24T10:46:40-07:00" level=info msg="Profiler Client Retransmit Proc 0 last busy 0.00%"
========================Statistics after 2.005134s============
==ClientData=============
0: TotalClients = 228, rate 0
1: TotalPacketsSent = 1140, rate 228
2: TotalPacketsRcvd = 1824, rate 228
3: TotalTXTSPacketsSent = 1140, rate 228
4: TotalTXTSRead = 1140, rate 228
5: MaxTXTSBytesOutstanding = 3332, rate 0
6: TotalGenMsgSent = 684, rate 0
7: TotalGenMsgRcvd = 1140, rate 0
8: TotalEventMsgSent = 456, rate 228
9: TotalEventMsgRcvd = 684, rate 228
10: TotalClientAnnounceReq = 228, rate 0
11: TotalClientAnnounceReqResend = 0, rate 0
12: TotalClientAnnounceGrant = 228, rate 0
13: TotalClientSyncReq = 228, rate 0
14: TotalClientSyncReqResend = 0, rate 0
15: TotalClientSyncGrant = 228, rate 0
16: TotalClientDelayRespReq = 228, rate 0
17: TotalClientDelayRespReqResend = 0, rate 0
18: TotalClientDelayRespGrant = 228, rate 0
19: TotalSyncRcvd = 228, rate 0
20: TotalPDelayRespRcvd = 0, rate 0
21: TotalFollowUpRcvd = 228, rate 0
22: TotalPDelayRespFollowUpRcvd = 0, rate 0
23: TotalAnnounceRcvd = 228, rate 0
24: TotalDelayReqSent = 456, rate 228
25: TotalDelayRespRcvd = 456, rate 228
26: TotalRetransmitHeapAdd = 0, rate 0
27: TotalRetransmitHeapAddAlreadyIn = 0, rate 0
28: TotalRetransmitHeapAddNotIn = 0, rate 0
29: TotalRetransmitHeapTryRemove = 0, rate 0
30: TotalRetransmitHeapRemove = 0, rate 0
31: TotalRetransmitHeapPop = 456, rate 228
Client states
Total: 228, Max: 1 , Min: 1
Count 1:228
==Tx Rx Counters=============
TX worker 0 pkt send: 1140
RX worker 0 pkt recv: 1824
==Client Request Data============
Announce Grant Requests sent
Total: 228, Max: 1 , Min: 1
Count 1:228
Sync Grant Requests sent
Total: 228, Max: 1 , Min: 1
Count 1:228
Delay Resp Grant Requests sent
Total: 228, Max: 1 , Min: 1
Count 1:228
Delay Requests sent
Total: 228, Max: 2 , Min: 2
Count 2:228
Announce Grant Latency
 228 samples of 228 events
Cumulative:	320.040312ms
HMean:		1.024843ms
Avg.:		1.403685ms
p50: 		1.16235ms
p75:		1.683062ms
p95:		3.27251ms
p99:		4.523026ms
p999:		5.568122ms
Long 5%:	4.00771ms
Short 5%:	471.566µs
Max:		5.568122ms
Min:		307.806µs
Range:		5.260316ms
StdDev:		888.435µs
Rate/sec.:	712.41
Sync Grant Latency
 228 samples of 228 events
Cumulative:	300.154828ms
HMean:		247.17µs
Avg.:		1.316468ms
p50: 		1.18881ms
p75:		1.857054ms
p95:		3.285182ms
p99:		4.467874ms
p999:		5.387614ms
Long 5%:	3.841688ms
Short 5%:	42.421µs
Max:		5.387614ms
Min:		35.566µs
Range:		5.352048ms
StdDev:		1.003697ms
Rate/sec.:	759.61
Delay Resp Grant Latency
 228 samples of 228 events
Cumulative:	229.549872ms
HMean:		145.619µs
Avg.:		1.006797ms
p50: 		1.095026ms
p75:		1.194654ms
p95:		2.357494ms
p99:		3.319574ms
p999:		3.398314ms
Long 5%:	3.005904ms
Short 5%:	29.675µs
Max:		3.398314ms
Min:		22.114µs
Range:		3.3762ms
StdDev:		814.495µs
Rate/sec.:	993.25
Delay Req Latency
 228 samples of 228 events
Cumulative:	143.902592ms
HMean:		469.397µs
Avg.:		631.151µs
p50: 		654.054µs
p75:		798.782µs
p95:		1.171114ms
p99:		1.304222ms
p999:		1.486662ms
Long 5%:	1.282575ms
Short 5%:	173.455µs
Max:		1.486662ms
Min:		130.678µs
Range:		1.355984ms
StdDev:		298.877µs
Rate/sec.:	1584.41
Time Between Syncs
 0 samples of 0 events
Cumulative:	0s
HMean:		0s
Avg.:		0s
p50: 		0s
p75:		0s
p95:		0s
p99:		0s
p999:		0s
Long 5%:	0s
Short 5%:	0s
Max:		0s
Min:		0s
Range:		0s
StdDev:		0s
Rate/sec.:	0.00
==Software Performance=============
time="2021-08-24T10:46:41-07:00" level=info msg="Profiler RX Worker 0 last busy 0.00%"
time="2021-08-24T10:46:41-07:00" level=info msg="Profiler PacketParser 0 last busy 0.00%"
time="2021-08-24T10:46:41-07:00" level=info msg="Profiler TX worker 0 last busy 0.00%"
time="2021-08-24T10:46:41-07:00" level=info msg="Profiler TX worker 0 TSRead worker 0 last busy 0.00%"
time="2021-08-24T10:46:41-07:00" level=info msg="Profiler PacketProcessor 0 last busy 0.00%"
time="2021-08-24T10:46:41-07:00" level=info msg="Profiler CounterProcessor last busy 0.00%"
time="2021-08-24T10:46:41-07:00" level=info msg="Profiler Client Retransmit Proc 0 last busy 0.00%"
```


