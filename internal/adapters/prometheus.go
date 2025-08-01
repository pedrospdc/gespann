package adapters

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pedrospdc/gespann/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusAdapter struct {
	registry *prometheus.Registry
	server   *http.Server

	// Connection counts
	openConnections   prometheus.Gauge
	closedConnections prometheus.Counter
	idleConnections   prometheus.Gauge
	resetConnections  prometheus.Counter
	failedConnections prometheus.Counter
	totalConnections  prometheus.Counter

	// Performance metrics
	totalBytesSent        prometheus.Counter
	totalBytesReceived    prometheus.Counter
	avgConnectionDuration prometheus.Gauge
	avgRTT                prometheus.Gauge

	// Protocol distribution
	tcpConnections prometheus.Counter
	udpConnections prometheus.Counter

	// Event tracking
	connectionEvents    *prometheus.CounterVec
	connectionBandwidth *prometheus.CounterVec
}

func NewPrometheusAdapter(settings map[string]string) (*PrometheusAdapter, error) {
	port := settings["port"]
	if port == "" {
		port = "8080"
	}

	registry := prometheus.NewRegistry()

	// Connection count metrics
	openConnections := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gespann_open_connections",
		Help: "Number of currently open connections",
	})

	closedConnections := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gespann_closed_connections_total",
		Help: "Total number of closed connections",
	})

	idleConnections := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gespann_idle_connections",
		Help: "Number of idle connections",
	})

	resetConnections := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gespann_reset_connections_total",
		Help: "Total number of reset connections",
	})

	failedConnections := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gespann_failed_connections_total",
		Help: "Total number of failed connection attempts",
	})

	totalConnections := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gespann_total_connections",
		Help: "Total number of connections seen",
	})

	// Performance metrics
	totalBytesSent := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gespann_bytes_sent_total",
		Help: "Total bytes sent across all connections",
	})

	totalBytesReceived := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gespann_bytes_received_total",
		Help: "Total bytes received across all connections",
	})

	avgConnectionDuration := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gespann_avg_connection_duration_ms",
		Help: "Average connection duration in milliseconds",
	})

	avgRTT := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gespann_avg_rtt_microseconds",
		Help: "Average round trip time in microseconds",
	})

	// Protocol distribution
	tcpConnections := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gespann_tcp_connections_total",
		Help: "Total number of TCP connections",
	})

	udpConnections := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gespann_udp_connections_total",
		Help: "Total number of UDP connections",
	})

	// Event tracking
	connectionEvents := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gespann_connection_events_total",
			Help: "Total number of connection events by type",
		},
		[]string{"event_type", "protocol", "reset_reason"},
	)

	connectionBandwidth := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gespann_connection_bandwidth_bytes_total",
			Help: "Total bandwidth usage by connection",
		},
		[]string{"direction", "protocol"},
	)

	registry.MustRegister(
		openConnections, closedConnections, idleConnections,
		resetConnections, failedConnections, totalConnections,
		totalBytesSent, totalBytesReceived, avgConnectionDuration, avgRTT,
		tcpConnections, udpConnections, connectionEvents, connectionBandwidth,
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	adapter := &PrometheusAdapter{
		registry:              registry,
		server:                server,
		openConnections:       openConnections,
		closedConnections:     closedConnections,
		idleConnections:       idleConnections,
		resetConnections:      resetConnections,
		failedConnections:     failedConnections,
		totalConnections:      totalConnections,
		totalBytesSent:        totalBytesSent,
		totalBytesReceived:    totalBytesReceived,
		avgConnectionDuration: avgConnectionDuration,
		avgRTT:                avgRTT,
		tcpConnections:        tcpConnections,
		udpConnections:        udpConnections,
		connectionEvents:      connectionEvents,
		connectionBandwidth:   connectionBandwidth,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Prometheus server error: %v\n", err)
		}
	}()

	return adapter, nil
}

func (p *PrometheusAdapter) SendMetrics(ctx context.Context, metrics types.ConnMetrics) error {
	// Connection counts
	p.openConnections.Set(float64(metrics.OpenConnections))
	p.idleConnections.Set(float64(metrics.IdleConnections))
	p.closedConnections.Add(float64(metrics.ClosedConnections))
	p.resetConnections.Add(float64(metrics.ResetConnections))
	p.failedConnections.Add(float64(metrics.FailedConnections))
	p.totalConnections.Add(float64(metrics.TotalConnections))

	// Performance metrics
	p.totalBytesSent.Add(float64(metrics.TotalBytesSent))
	p.totalBytesReceived.Add(float64(metrics.TotalBytesReceived))
	p.avgConnectionDuration.Set(metrics.AvgConnectionDuration)
	p.avgRTT.Set(metrics.AvgRTT)

	// Protocol distribution
	p.tcpConnections.Add(float64(metrics.TCPConnections))
	p.udpConnections.Add(float64(metrics.UDPConnections))

	return nil
}

func (p *PrometheusAdapter) SendEvent(ctx context.Context, event types.ConnEvent) error {
	eventType := ""
	switch event.Type {
	case types.ConnOpen:
		eventType = "open"
	case types.ConnClose:
		eventType = "close"
	case types.ConnIdle:
		eventType = "idle"
	case types.ConnReset:
		eventType = "reset"
	case types.ConnFailed:
		eventType = "failed"
	case types.ConnData:
		eventType = "data"
	}

	protocol := ""
	switch event.Protocol {
	case types.ProtoTCP:
		protocol = "tcp"
	case types.ProtoUDP:
		protocol = "udp"
	default:
		protocol = "unknown"
	}

	resetReason := ""
	switch event.ResetReason {
	case types.ResetNormal:
		resetReason = "normal"
	case types.ResetTimeout:
		resetReason = "timeout"
	case types.ResetRefused:
		resetReason = "refused"
	case types.ResetAbort:
		resetReason = "abort"
	}

	p.connectionEvents.WithLabelValues(eventType, protocol, resetReason).Inc()

	// Track bandwidth
	if event.BytesSent > 0 {
		p.connectionBandwidth.WithLabelValues("sent", protocol).Add(float64(event.BytesSent))
	}
	if event.BytesReceived > 0 {
		p.connectionBandwidth.WithLabelValues("received", protocol).Add(float64(event.BytesReceived))
	}

	return nil
}

func (p *PrometheusAdapter) Close() error {
	return p.server.Shutdown(context.Background())
}
