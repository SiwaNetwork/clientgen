/*
Copyright (c) Facebook, Inc. and its affiliates.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clientgenlib

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/kpango/fastime"
	log "github.com/sirupsen/logrus"
)

// PF_RING stub types for Linux compatibility
type Ring struct {
	closed bool
}

type Flag int

const (
	FlagPromisc     Flag = 1
	FlagHWTimestamp Flag = 2
	FlagLongHeader  Flag = 4
)

type ClusterType int

const (
	ClusterRoundRobin ClusterType = 1
)

type Direction int

const (
	ReceiveOnly Direction = 1
	ReadOnly    Direction = 2
)

type ExtendedPacketHeader struct {
	Timestamp struct {
		Sec  int64
		Nsec int64
	}
}

func NewRing(iface string, bufferSize int, flags Flag) (*Ring, error) {
	return &Ring{}, fmt.Errorf("PF_RING not available on this system")
}

func (r *Ring) Close() error {
	r.closed = true
	return nil
}

func (r *Ring) SetApplicationName(name string) error {
	return nil
}

func (r *Ring) SetCluster(id int, clusterType ClusterType) error {
	return nil
}

func (r *Ring) SetDirection(direction Direction) error {
	return nil
}

func (r *Ring) SetPollWatermark(watermark int) error {
	return nil
}

func (r *Ring) SetPollDuration(duration int) error {
	return nil
}

func (r *Ring) SetSamplingRate(rate int) error {
	return nil
}

func (r *Ring) Enable() error {
	return fmt.Errorf("PF_RING not available on this system")
}

func (r *Ring) Disable() error {
	return nil
}

func (r *Ring) GetNextPacket() ([]byte, *ExtendedPacketHeader, error) {
	return nil, nil, fmt.Errorf("PF_RING not available on this system")
}

func (r *Ring) SendPacket(data []byte) error {
	return fmt.Errorf("PF_RING not available on this system")
}

func (r *Ring) Stats() (Stats, error) {
	return Stats{}, fmt.Errorf("PF_RING not available on this system")
}

type Stats struct {
	Received uint64
	Dropped  uint64
}

func (r *Ring) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	return nil, gopacket.CaptureInfo{}, fmt.Errorf("PF_RING not available on this system")
}

func (r *Ring) ReadPacketDataExtended() (ExtendedPacketHeader, error) {
	return ExtendedPacketHeader{}, fmt.Errorf("PF_RING not available on this system")
}

func (r *Ring) WritePacketData(data []byte) error {
	return fmt.Errorf("PF_RING not available on this system")
}

func (r *Ring) SetSocketMode(mode int) error {
	return nil
}

// inPacket is input packet data + receive timestamp
type inPacket struct {
	data   []byte
	ts     time.Time
	fromTX bool
}

type outPacket struct {
	data    *gopacket.SerializeBuffer
	getTS   bool
	pktType uint8
	sentTS  time.Time
	cl      *SingleClientGen
}

func startIOWorker(cfg *ClientGenConfig) {
	rxStartDone := make(chan bool)
	for rxwkr := 0; rxwkr < cfg.NumRXWorkers; rxwkr++ {
		func(i int) {
			cfg.Eg.Go(func() error {
				doneChan := make(chan error, 1)
				go func() {
					var profiler Profiler
					profiler.Init(cfg.Eg, cfg.Ctx, true, fmt.Sprintf("RX Worker %d", i))
					cfg.PerfProfilers = append(cfg.PerfProfilers, &profiler)
					var ring *Ring
					var rawIn *inPacket
					var err error
					// 1<<24 is PF_RING_DISCARD_INJECTED_PKTS , if you transmit a packet via the ring, doesn't read it back
					// Добавляем флаги для поддержки hardware timestamps
					flags := (1 << 24) | FlagPromisc | FlagHWTimestamp | FlagLongHeader
					if ring, err = NewRing(cfg.Iface, 65536, flags); err != nil {
						log.Errorf("pfring ring creation error: %v", err)
						doneChan <- err
						return
					}
					defer ring.Close()

					// Устанавливаем размер буфера для улучшения производительности
					if err = ring.SetApplicationName("clientgen"); err != nil {
						log.Warnf("pfring SetApplicationName error: %v", err)
					}

					// just use fixed cluster number 1, round robin packets
					if err = ring.SetCluster(1, ClusterType(ClusterRoundRobin)); err != nil {
						log.Errorf("pfring SetCluster error: %v", err)
						doneChan <- err
						return
					}
					if err = ring.SetDirection(ReceiveOnly); err != nil {
						log.Errorf("pfring failed to set direction: %v", err)
						doneChan <- err
						return
					}
					// Оптимизируем параметры для низкой задержки
					if err = ring.SetPollWatermark(1); err != nil {
						log.Errorf("pfring failed to set poll watermark: %v", err)
						doneChan <- err
						return
					}
					if err = ring.SetPollDuration(0); err != nil {
						log.Errorf("pfring failed to set poll duration: %v", err)
						doneChan <- err
						return
					}
					if err = ring.SetSamplingRate(1); err != nil {
						log.Errorf("pfring failed to set sample rate: %v", err)
						doneChan <- err
						return
					}
					// only using read for now
					if err = ring.SetSocketMode(0); err != nil { // ReadOnly = 0
						log.Errorf("pfring SetSocketMode error: %v", err)
						doneChan <- err
						return
					} else if err = ring.Enable(); err != nil {
						log.Errorf("pfring Enable error: %v", err)
						doneChan <- err
						return
					}
					if cfg.DebugPrint || cfg.DebugIoWkrRX {
						log.Debugf("RX wkr %d pfring done!", i)
					}

					var data []byte
					var ci gopacket.CaptureInfo
					var pktHdr ExtendedPacketHeader
					rxStartDone <- true

					// Статистика PF_RING
					go func() {
						ticker := time.NewTicker(5 * time.Second)
						defer ticker.Stop()
						for {
							select {
							case <-ticker.C:
								stats, err := ring.Stats()
								if err == nil {
									atomic.StoreUint64(&cfg.Counters.PFRingRXPackets, stats.Received)
									atomic.StoreUint64(&cfg.Counters.PFRingRXDropped, stats.Dropped)
								}
							case <-(*cfg.Ctx).Done():
								return
							}
						}
					}()

					for {
						// try to read from handle
						// Используем ReadPacketDataExtended для получения расширенной информации включая hardware timestamps
						data, ci, err = ring.ReadPacketData()
						if err != nil || data == nil || len(data) == 0 {
							continue
						}

						// Получаем расширенную информацию о пакете для hardware timestamps
						pktHdr, err = ring.ReadPacketDataExtended()
						if err == nil && pktHdr.Timestamp.Sec > 0 {
							// Используем hardware timestamp если доступен
							ci.Timestamp = time.Unix(int64(pktHdr.Timestamp.Sec), int64(pktHdr.Timestamp.Nsec))
							atomic.AddUint64(&cfg.Counters.PFRingHWTimestamps, 1)
							if cfg.DebugPrint || cfg.DebugIoWkrRX {
								log.Debugf("PFring listener %d got HW timestamp: %v", i, ci.Timestamp)
							}
						}

						profiler.Tick()
						if cfg.DebugPrint || cfg.DebugIoWkrRX {
							log.Debugf("PFring listener %d got data ts %v, len %d", i, ci.Timestamp, len(data))
						}
						rawIn = cfg.RunData.inPacketPool.Get().(*inPacket)
						rawIn.data = data
						rawIn.ts = ci.Timestamp
						rawIn.fromTX = false

						cfg.RunData.rawInput[getRxChanNumToUse(cfg)] <- rawIn
						atomic.AddUint64(&cfg.Counters.TotalPacketsRcvd, 1)
						atomic.AddUint64(&cfg.perIORX[i], 1)
						profiler.Tock()
					}
				}()
				select {
				case <-(*cfg.Ctx).Done():
					log.Errorf("RX %d done due to context", i)
					return (*cfg.Ctx).Err()
				case err := <-doneChan:
					return err
				}
			})
		}(rxwkr)
		select {
		case <-rxStartDone:
			if cfg.DebugPrint || cfg.DebugIoWkrRX {
				log.Debugf("RX worker %d running", rxwkr)
			}
			continue
		case <-(*cfg.Ctx).Done():
			log.Errorf("Rx worker startup error")
			return
		}
	}

	txStartDone := make(chan bool)
	for txwkr := 0; txwkr < cfg.NumTXWorkers; txwkr++ {
		func(i int) {
			cfg.Eg.Go(func() error {
				doneChan := make(chan error, 1)
				go func() {
					// PFring doesn't implement TX timestamps actually
					// API documentation lists it, but at a low level, its not actually used
					// create a raw socket and send packets via it , read TS similar to Oleg's method
					var profiler Profiler
					profiler.Init(cfg.Eg, cfg.Ctx, true, fmt.Sprintf("TX worker %d", i))
					cfg.PerfProfilers = append(cfg.PerfProfilers, &profiler)

					txTSworker := make([]Profiler, cfg.NumTXTSWorkerPerTx)
					for j := 0; j < cfg.NumTXTSWorkerPerTx; j++ {
						txTSworker[j].Init(cfg.Eg, cfg.Ctx, true, fmt.Sprintf("TX worker %d TSRead worker %d", i, j))
						cfg.PerfProfilers = append(cfg.PerfProfilers, &txTSworker[j])
					}

					// Создаем PF_RING для передачи пакетов
					var txRing *Ring
					var err error
					// Флаги для TX: без DISCARD_INJECTED_PKTS чтобы можно было читать свои пакеты для timestamp
					txFlags := FlagPromisc | FlagHWTimestamp | FlagLongHeader
					if txRing, err = NewRing(cfg.Iface, 65536, txFlags); err != nil {
						log.Errorf("pfring TX ring creation error: %v", err)
						doneChan <- err
						return
					}
					defer txRing.Close()

					if err = txRing.SetApplicationName("clientgen-tx"); err != nil {
						log.Warnf("pfring TX SetApplicationName error: %v", err)
					}

					if err = txRing.SetDirection(1); err != nil { // TransmitOnly = 1
						log.Errorf("pfring TX failed to set direction: %v", err)
						doneChan <- err
						return
					}

					if err = txRing.SetSocketMode(1); err != nil { // WriteOnly = 1
						log.Errorf("pfring TX SetSocketMode error: %v", err)
						doneChan <- err
						return
					}

					if err = txRing.Enable(); err != nil {
						log.Errorf("pfring TX Enable error: %v", err)
						doneChan <- err
						return
					}

					// Note: Raw socket timestamping is not supported on Windows
					// This would normally create raw sockets for hardware timestamping
					// Hardware timestamping not supported on Windows
					// This would normally enable hardware timestamping capabilities

					var out *outPacket
					txStartDone <- true
					for {
						out = <-(cfg.RunData.rawOutput[i]) // want to send a packet
						if out == nil || len((*out.data).Bytes()) == 0 {
							log.Infof("empty data bad!")
							continue
						}
						if cfg.DebugPrint || cfg.DebugIoWkrTX {
							// debug print
							debugPkt := gopacket.NewPacket((*out.data).Bytes(), layers.LinkTypeEthernet, gopacket.Default)
							log.Debugf("Debug txWkr %d send packet %v", i, debugPkt)
						}
						profiler.Tick()
						if out.getTS {
							// Используем PF_RING для отправки с timestamp
							err = txRing.WritePacketData((*out.data).Bytes())
							if err != nil {
								log.Errorf("txWkr %d send packet TS failed: %v", i, err)
							}
							if out.cl != nil {
								out.cl.CountOutgoingPackets++
							}
							atomic.AddUint64(&cfg.Counters.TotalTXTSPacketsSent, 1)
						} else {
							// Используем PF_RING для обычной отправки
							err = txRing.WritePacketData((*out.data).Bytes())
							if err != nil {
								log.Errorf("txWkr %d send packet failed: %v", i, err)
							}
							if out.cl != nil {
								out.cl.CountOutgoingPackets++
								out.sentTS = fastime.Now()
								if out.pktType == pktAnnounceGrantReq {
									out.cl.SentAnnounceGrantReqTime = out.sentTS
								} else if out.pktType == pktSyncGrantReq {
									out.cl.SentlastSyncGrantReqTime = out.sentTS
								} else if out.pktType == pktDelayRespGrantReq {
									out.cl.SentDelayRespGrantReqTime = out.sentTS
								} else if out.pktType == pktDelayReq {
									out.cl.SentDelayReqTime = out.sentTS
								}
							}
							if cfg.DebugPrint || cfg.DebugIoWkrTX {
								log.Debugf("Debug txWkr %d send packet via PF_RING", i)
							}
							if err != nil {
								log.Errorf("PF_RING write packet data failed %v", err)
								// Не прерываем работу, продолжаем с следующим пакетом
							}
						}
						atomic.AddUint64(&cfg.Counters.TotalPacketsSent, 1)
						atomic.AddUint64(&cfg.perIOTX[i], 1)
						cfg.RunData.outPacketPool.Put(out)
						profiler.Tock()
					}
				}()
				var err error
				select {
				case <-(*cfg.Ctx).Done():
					log.Infof("TX worker %d cancelling due to context done", i)
					return (*cfg.Ctx).Err()
				case err = <-doneChan:
					return err
				}
			})
		}(txwkr)
		select {
		case <-txStartDone:
			if cfg.DebugPrint || cfg.DebugIoWkrTX {
				log.Debugf("TX worker %d running", txwkr)
			}
			continue
		case <-(*cfg.Ctx).Done():
			log.Errorf("Tx worker startup error")
			return
		}
	}
}
