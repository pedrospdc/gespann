//go:build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>

#define MAX_ENTRIES 10240

struct conn_event {
    __u32 pid;
    __u32 tid;
    __u32 saddr;
    __u32 daddr;
    __u16 sport;
    __u16 dport;
    __u8 event_type;
    __u8 protocol;
    __u64 timestamp;
    __u64 bytes_sent;
    __u64 bytes_received;
    __u32 rtt_us;
    __u32 duration_ms;
    __u8 tcp_state;
    __u8 reset_reason;
};

enum event_type {
    CONN_OPEN = 1,
    CONN_CLOSE = 2,
    CONN_IDLE = 3,
    CONN_RESET = 4,
    CONN_FAILED = 5,
    CONN_DATA = 6,
};

enum protocol_type {
    PROTO_TCP = 6,
    PROTO_UDP = 17,
    PROTO_UNKNOWN = 0,
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} events SEC(".maps");

SEC("kprobe/sys_connect")
int trace_connect_entry(struct pt_regs *ctx)
{
    struct conn_event *event;
    
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event)
        return 0;

    event->pid = bpf_get_current_pid_tgid() >> 32;
    event->tid = bpf_get_current_pid_tgid();
    event->timestamp = bpf_ktime_get_ns();
    event->event_type = CONN_OPEN;
    event->protocol = PROTO_TCP;
    event->bytes_sent = 0;
    event->bytes_received = 0;
    event->rtt_us = 0;
    event->duration_ms = 0;
    event->tcp_state = 1;
    event->reset_reason = 0;
    
    // For demo purposes, use placeholder values
    event->saddr = 0x0100007f; // 127.0.0.1
    event->daddr = 0x0100007f; // 127.0.0.1
    event->sport = 8080;
    event->dport = 80;

    bpf_ringbuf_submit(event, 0);
    return 0;
}

SEC("kprobe/sys_close")
int trace_close_entry(struct pt_regs *ctx)
{
    struct conn_event *event;
    
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event)
        return 0;

    event->pid = bpf_get_current_pid_tgid() >> 32;
    event->tid = bpf_get_current_pid_tgid();
    event->timestamp = bpf_ktime_get_ns();
    event->event_type = CONN_CLOSE;
    event->protocol = PROTO_TCP;
    event->bytes_sent = 1024;
    event->bytes_received = 2048;
    event->rtt_us = 500;
    event->duration_ms = 5000;
    event->tcp_state = 0;
    event->reset_reason = 0;
    
    // For demo purposes, use placeholder values
    event->saddr = 0x0100007f; // 127.0.0.1
    event->daddr = 0x0100007f; // 127.0.0.1
    event->sport = 8080;
    event->dport = 80;

    bpf_ringbuf_submit(event, 0);
    return 0;
}

char _license[] SEC("license") = "GPL";