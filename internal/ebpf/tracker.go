package ebpf

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"
	"unsafe"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/pedrospdc/gespann/pkg/types"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -target bpfel ConnTracker ../../bpf/simple_tracker.c

type ConnEvent struct {
	PID           uint32
	TID           uint32
	SAddr         uint32
	DAddr         uint32
	SPort         uint16
	DPort         uint16
	EventType     uint8
	Protocol      uint8
	Timestamp     uint64
	BytesSent     uint64
	BytesReceived uint64
	RTTMicros     uint32
	DurationMS    uint32
	TCPState      uint8
	ResetReason   uint8
}

type Tracker struct {
	objs   ConnTrackerObjects
	links  []link.Link
	reader *ringbuf.Reader
	logger *slog.Logger
}

func NewTracker(logger *slog.Logger) (*Tracker, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock limit: %w", err)
	}

	spec, err := LoadConnTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to load eBPF spec: %w", err)
	}

	var objs ConnTrackerObjects
	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		return nil, fmt.Errorf("failed to load eBPF objects: %w", err)
	}

	reader, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		return nil, fmt.Errorf("failed to create ringbuf reader: %w", err)
	}

	return &Tracker{
		objs:   objs,
		reader: reader,
		logger: logger,
	}, nil
}

func (t *Tracker) Start(ctx context.Context) error {
	// Attach kprobe for connect system calls
	connectLink, err := link.Kprobe("sys_connect", t.objs.TraceConnectEntry, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sys_connect kprobe: %w", err)
	}
	t.links = append(t.links, connectLink)

	// Attach kprobe for close system calls
	closeLink, err := link.Kprobe("sys_close", t.objs.TraceCloseEntry, nil)
	if err != nil {
		return fmt.Errorf("failed to attach sys_close kprobe: %w", err)
	}
	t.links = append(t.links, closeLink)

	t.logger.Info("eBPF programs attached successfully")
	return nil
}

func (t *Tracker) ReadEvents(ctx context.Context, eventCh chan<- types.ConnEvent) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			record, err := t.reader.Read()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				t.logger.Error("failed to read from ringbuf", "error", err)
				continue
			}

			if len(record.RawSample) < int(unsafe.Sizeof(ConnEvent{})) {
				t.logger.Warn("received truncated event")
				continue
			}

			var rawEvent ConnEvent
			if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &rawEvent); err != nil {
				t.logger.Error("failed to parse event", "error", err)
				continue
			}

			event := types.ConnEvent{
				PID:           rawEvent.PID,
				TID:           rawEvent.TID,
				SAddr:         rawEvent.SAddr,
				DAddr:         rawEvent.DAddr,
				SPort:         rawEvent.SPort,
				DPort:         rawEvent.DPort,
				Type:          types.EventType(rawEvent.EventType),
				Protocol:      types.ProtocolType(rawEvent.Protocol),
				Timestamp:     time.Unix(0, int64(rawEvent.Timestamp)),
				BytesSent:     rawEvent.BytesSent,
				BytesReceived: rawEvent.BytesReceived,
				RTTMicros:     rawEvent.RTTMicros,
				DurationMS:    rawEvent.DurationMS,
				TCPState:      rawEvent.TCPState,
				ResetReason:   types.ResetReason(rawEvent.ResetReason),
			}

			select {
			case eventCh <- event:
			case <-ctx.Done():
				return ctx.Err()
			default:
				t.logger.Warn("event channel full, dropping event")
			}
		}
	}
}

func (t *Tracker) Close() error {
	var errs []error

	for _, l := range t.links {
		if err := l.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if t.reader != nil {
		if err := t.reader.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := t.objs.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close tracker: %v", errs)
	}

	return nil
}
