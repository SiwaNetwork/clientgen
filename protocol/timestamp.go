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

package protocol

// Here we have basic HW and SW timestamping support

import (
	"fmt"
	"net"
	"time"
)

const (
	// Control is a socket control message containing TX/RX timestamp
	// If the read fails we may endup with multiple timestamps in the buffer
	// which is best to read right away
	ControlSizeBytes = 128
	// ptp packets usually up to 66 bytes
	PayloadSizeBytes = 128
	// HWTIMESTAMP is a hardware timestamp
	HWTIMESTAMP = "hardware"
	// SWTIMESTAMP is a software timestmap
	SWTIMESTAMP = "software"
)

func Timestamping() int {
	return 37 // SO_TIMESTAMPING_NEW
}

// ConnFd returns file descriptor of a connection
func ConnFd(conn *net.UDPConn) (int, error) {
	sc, err := conn.SyscallConn()
	if err != nil {
		return -1, err
	}
	var intfd int
	err = sc.Control(func(fd uintptr) {
		intfd = int(fd)
	})
	if err != nil {
		return -1, err
	}
	return intfd, nil
}

func IoctlTimestamp(fd int, ifname string) error {
	// Stub implementation - hardware timestamping not supported
	return fmt.Errorf("hardware timestamping not supported on this platform")
}

// EnableHWTimestampsSocket enables HW timestamps on the socket
func EnableHWTimestampsSocket(connFd int, iface string) error {
	return fmt.Errorf("hardware timestamping not supported on this platform")
}

// EnableSWTimestampsSocket enables SW timestamps on the socket
func EnableSWTimestampsSocket(connFd int) error {
	return fmt.Errorf("software timestamping not supported on this platform")
}

// SocketControlMessageTimestamp is a very optimised version of ParseSocketControlMessage
func SocketControlMessageTimestamp(b []byte) (time.Time, error) {
	return time.Time{}, fmt.Errorf("timestamping not supported on this platform")
}

// ReadTXtimestampBuf returns HW TX timestamp, needs to be provided 2 buffers which all can be re-used after ReadTXtimestampBuf finishes.
func ReadTXtimestampBuf(connFd int, oob, toob []byte) (time.Time, int, error) {
	return time.Time{}, 0, fmt.Errorf("timestamping not supported on this platform")
}

// ReadTXtimestamp returns HW TX timestamp
func ReadTXtimestamp(connFd int) (time.Time, int, error) {
	return time.Time{}, 0, fmt.Errorf("timestamping not supported on this platform")
}

// ReadPacketWithRXTimestamp returns byte packet and HW RX timestamp
func ReadPacketWithRXTimestamp(connFd int) ([]byte, interface{}, time.Time, error) {
	return nil, nil, time.Time{}, fmt.Errorf("timestamping not supported on this platform")
}

// ReadPacketWithRXTimestampBuf writes byte packet into provide buffer buf, and returns number of bytes copied to the buffer, client ip and HW RX timestamp.
func ReadPacketWithRXTimestampBuf(connFd int, buf, oob []byte) (int, interface{}, time.Time, error) {
	return 0, nil, time.Time{}, fmt.Errorf("timestamping not supported on this platform")
}

// IPToSockaddr converts IP + port into a socket address
func IPToSockaddr(ip net.IP, port int) interface{} {
	return nil
}

// SockaddrToIP converts socket address to an IP
func SockaddrToIP(sa interface{}) net.IP {
	return nil
}
