package types

import "time"

type EventType uint8

const (
	ConnOpen EventType = iota + 1
	ConnClose
	ConnIdle
	ConnReset
	ConnFailed
	ConnData
)

type ProtocolType uint8

const (
	ProtoUnknown ProtocolType = 0
	ProtoTCP     ProtocolType = 6
	ProtoUDP     ProtocolType = 17
)

type ResetReason uint8

const (
	ResetNormal  ResetReason = 0
	ResetTimeout ResetReason = 1
	ResetRefused ResetReason = 2
	ResetAbort   ResetReason = 3
)

type ConnEvent struct {
	PID           uint32       `json:"pid"`
	TID           uint32       `json:"tid"`
	SAddr         uint32       `json:"saddr"`
	DAddr         uint32       `json:"daddr"`
	SPort         uint16       `json:"sport"`
	DPort         uint16       `json:"dport"`
	Type          EventType    `json:"event_type"`
	Protocol      ProtocolType `json:"protocol"`
	Timestamp     time.Time    `json:"timestamp"`
	BytesSent     uint64       `json:"bytes_sent"`
	BytesReceived uint64       `json:"bytes_received"`
	RTTMicros     uint32       `json:"rtt_microseconds"`
	DurationMS    uint32       `json:"duration_ms"`
	TCPState      uint8        `json:"tcp_state"`
	ResetReason   ResetReason  `json:"reset_reason"`
}

type ConnMetrics struct {
	// Connection counts
	OpenConnections   int64 `json:"open_connections"`
	ClosedConnections int64 `json:"closed_connections"`
	IdleConnections   int64 `json:"idle_connections"`
	ResetConnections  int64 `json:"reset_connections"`
	FailedConnections int64 `json:"failed_connections"`
	TotalConnections  int64 `json:"total_connections"`

	// Performance metrics
	TotalBytesSent        uint64  `json:"total_bytes_sent"`
	TotalBytesReceived    uint64  `json:"total_bytes_received"`
	AvgConnectionDuration float64 `json:"avg_connection_duration_ms"`
	AvgRTT                float64 `json:"avg_rtt_microseconds"`

	// Protocol distribution
	TCPConnections int64 `json:"tcp_connections"`
	UDPConnections int64 `json:"udp_connections"`
}
