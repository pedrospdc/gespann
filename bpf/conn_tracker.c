//go:build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>

// Avoid system bpf_tracing.h PT_REGS macros, use our own
#undef PT_REGS_PARM1
#undef PT_REGS_PARM2  
#undef PT_REGS_PARM3

typedef __u64 size_t;

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

struct conn_state {
    __u64 start_time;
    __u64 bytes_sent;
    __u64 bytes_received;
    __u32 last_rtt;
    __u8 tcp_state;
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

enum reset_reason {
    RESET_NORMAL = 0,
    RESET_TIMEOUT = 1,
    RESET_REFUSED = 2,
    RESET_ABORT = 3,
};

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} events SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);
    __type(value, struct conn_state);
    __uint(max_entries, MAX_ENTRIES);
} conn_state_map SEC(".maps");

static __always_inline __u32 make_conn_key(__u32 saddr, __u32 daddr, __u16 sport, __u16 dport) {
    return saddr ^ daddr ^ ((__u32)sport << 16) ^ (__u32)dport;
}

SEC("kprobe/tcp_connect")
int trace_tcp_connect(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    struct conn_event *event;
    struct conn_state state = {};
    
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event)
        return 0;

    __u64 now = bpf_ktime_get_ns();
    event->pid = bpf_get_current_pid_tgid() >> 32;
    event->tid = bpf_get_current_pid_tgid();
    event->timestamp = now;
    event->event_type = CONN_OPEN;
    event->protocol = PROTO_TCP;
    event->bytes_sent = 0;
    event->bytes_received = 0;
    event->rtt_us = 0;
    event->duration_ms = 0;
    event->tcp_state = 1; // TCP_ESTABLISHED
    event->reset_reason = RESET_NORMAL;

    struct inet_sock *inet = (struct inet_sock *)sk;
    BPF_CORE_READ_INTO(&event->saddr, inet, inet_saddr);
    BPF_CORE_READ_INTO(&event->daddr, inet, inet_daddr);
    BPF_CORE_READ_INTO(&event->sport, inet, inet_sport);
    BPF_CORE_READ_INTO(&event->dport, inet, inet_dport);

    __u32 conn_key = make_conn_key(event->saddr, event->daddr, event->sport, event->dport);
    
    // Initialize connection state
    state.start_time = now;
    state.bytes_sent = 0;
    state.bytes_received = 0;
    state.last_rtt = 0;
    state.tcp_state = 1;
    
    bpf_map_update_elem(&conn_state_map, &conn_key, &state, BPF_ANY);

    bpf_ringbuf_submit(event, 0);
    return 0;
}

SEC("kprobe/tcp_close")
int trace_tcp_close(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    struct conn_event *event;
    struct conn_state *state;
    
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event)
        return 0;

    __u64 now = bpf_ktime_get_ns();
    event->pid = bpf_get_current_pid_tgid() >> 32;
    event->tid = bpf_get_current_pid_tgid();
    event->timestamp = now;
    event->event_type = CONN_CLOSE;
    event->protocol = PROTO_TCP;
    event->reset_reason = RESET_NORMAL;

    struct inet_sock *inet = (struct inet_sock *)sk;
    BPF_CORE_READ_INTO(&event->saddr, inet, inet_saddr);
    BPF_CORE_READ_INTO(&event->daddr, inet, inet_daddr);
    BPF_CORE_READ_INTO(&event->sport, inet, inet_sport);
    BPF_CORE_READ_INTO(&event->dport, inet, inet_dport);

    __u32 conn_key = make_conn_key(event->saddr, event->daddr, event->sport, event->dport);
    
    // Get connection state for duration and byte counts
    state = bpf_map_lookup_elem(&conn_state_map, &conn_key);
    if (state) {
        event->duration_ms = (now - state->start_time) / 1000000; // ns to ms
        event->bytes_sent = state->bytes_sent;
        event->bytes_received = state->bytes_received;
        event->rtt_us = state->last_rtt;
        event->tcp_state = state->tcp_state;
        
        bpf_map_delete_elem(&conn_state_map, &conn_key);
    } else {
        event->duration_ms = 0;
        event->bytes_sent = 0;
        event->bytes_received = 0;
        event->rtt_us = 0;
        event->tcp_state = 0;
    }

    bpf_ringbuf_submit(event, 0);
    return 0;
}

SEC("kprobe/tcp_keepalive_timer")
int trace_tcp_keepalive(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    struct conn_event *event;
    struct conn_state *state;
    __u64 now = bpf_ktime_get_ns();
    
    struct inet_sock *inet = (struct inet_sock *)sk;
    __u32 saddr, daddr;
    __u16 sport, dport;
    
    BPF_CORE_READ_INTO(&saddr, inet, inet_saddr);
    BPF_CORE_READ_INTO(&daddr, inet, inet_daddr);
    BPF_CORE_READ_INTO(&sport, inet, inet_sport);
    BPF_CORE_READ_INTO(&dport, inet, inet_dport);

    __u32 conn_key = make_conn_key(saddr, daddr, sport, dport);
    state = bpf_map_lookup_elem(&conn_state_map, &conn_key);
    
    if (state && (now - state->start_time) > 30000000000ULL) {
        event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
        if (!event)
            return 0;

        event->pid = bpf_get_current_pid_tgid() >> 32;
        event->tid = bpf_get_current_pid_tgid();
        event->timestamp = now;
        event->event_type = CONN_IDLE;
        event->protocol = PROTO_TCP;
        event->saddr = saddr;
        event->daddr = daddr;
        event->sport = sport;
        event->dport = dport;
        event->duration_ms = (now - state->start_time) / 1000000;
        event->bytes_sent = state->bytes_sent;
        event->bytes_received = state->bytes_received;
        event->rtt_us = state->last_rtt;
        event->tcp_state = state->tcp_state;
        event->reset_reason = RESET_NORMAL;

        bpf_ringbuf_submit(event, 0);
    }

    return 0;
}

SEC("kprobe/tcp_sendmsg")
int trace_tcp_sendmsg(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    size_t size = (size_t)PT_REGS_PARM3(ctx);
    struct conn_state *state;
    struct inet_sock *inet = (struct inet_sock *)sk;
    
    __u32 saddr, daddr;
    __u16 sport, dport;
    
    BPF_CORE_READ_INTO(&saddr, inet, inet_saddr);
    BPF_CORE_READ_INTO(&daddr, inet, inet_daddr);
    BPF_CORE_READ_INTO(&sport, inet, inet_sport);
    BPF_CORE_READ_INTO(&dport, inet, inet_dport);

    __u32 conn_key = make_conn_key(saddr, daddr, sport, dport);
    state = bpf_map_lookup_elem(&conn_state_map, &conn_key);
    
    if (state) {
        state->bytes_sent += size;
        bpf_map_update_elem(&conn_state_map, &conn_key, state, BPF_EXIST);
    }

    return 0;
}

SEC("kprobe/tcp_recvmsg")
int trace_tcp_recvmsg(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    size_t size = (size_t)PT_REGS_PARM3(ctx);
    struct conn_state *state;
    struct inet_sock *inet = (struct inet_sock *)sk;
    
    __u32 saddr, daddr;
    __u16 sport, dport;
    
    BPF_CORE_READ_INTO(&saddr, inet, inet_saddr);
    BPF_CORE_READ_INTO(&daddr, inet, inet_daddr);
    BPF_CORE_READ_INTO(&sport, inet, inet_sport);
    BPF_CORE_READ_INTO(&dport, inet, inet_dport);

    __u32 conn_key = make_conn_key(saddr, daddr, sport, dport);
    state = bpf_map_lookup_elem(&conn_state_map, &conn_key);
    
    if (state) {
        state->bytes_received += size;
        bpf_map_update_elem(&conn_state_map, &conn_key, state, BPF_EXIST);
    }

    return 0;
}

SEC("kprobe/tcp_reset")
int trace_tcp_reset(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    struct conn_event *event;
    struct conn_state *state;
    
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event)
        return 0;

    __u64 now = bpf_ktime_get_ns();
    event->pid = bpf_get_current_pid_tgid() >> 32;
    event->tid = bpf_get_current_pid_tgid();
    event->timestamp = now;
    event->event_type = CONN_RESET;
    event->protocol = PROTO_TCP;
    event->reset_reason = RESET_ABORT;

    struct inet_sock *inet = (struct inet_sock *)sk;
    BPF_CORE_READ_INTO(&event->saddr, inet, inet_saddr);
    BPF_CORE_READ_INTO(&event->daddr, inet, inet_daddr);
    BPF_CORE_READ_INTO(&event->sport, inet, inet_sport);
    BPF_CORE_READ_INTO(&event->dport, inet, inet_dport);

    __u32 conn_key = make_conn_key(event->saddr, event->daddr, event->sport, event->dport);
    
    state = bpf_map_lookup_elem(&conn_state_map, &conn_key);
    if (state) {
        event->duration_ms = (now - state->start_time) / 1000000;
        event->bytes_sent = state->bytes_sent;
        event->bytes_received = state->bytes_received;
        event->rtt_us = state->last_rtt;
        event->tcp_state = state->tcp_state;
        
        bpf_map_delete_elem(&conn_state_map, &conn_key);
    } else {
        event->duration_ms = 0;
        event->bytes_sent = 0;
        event->bytes_received = 0;
        event->rtt_us = 0;
        event->tcp_state = 0;
    }

    bpf_ringbuf_submit(event, 0);
    return 0;
}

SEC("kprobe/tcp_connect_fail")
int trace_tcp_connect_fail(struct pt_regs *ctx)
{
    struct sock *sk = (struct sock *)PT_REGS_PARM1(ctx);
    struct conn_event *event;
    
    event = bpf_ringbuf_reserve(&events, sizeof(*event), 0);
    if (!event)
        return 0;

    event->pid = bpf_get_current_pid_tgid() >> 32;
    event->tid = bpf_get_current_pid_tgid();
    event->timestamp = bpf_ktime_get_ns();
    event->event_type = CONN_FAILED;
    event->protocol = PROTO_TCP;
    event->reset_reason = RESET_REFUSED;
    event->duration_ms = 0;
    event->bytes_sent = 0;
    event->bytes_received = 0;
    event->rtt_us = 0;
    event->tcp_state = 0;

    struct inet_sock *inet = (struct inet_sock *)sk;
    BPF_CORE_READ_INTO(&event->saddr, inet, inet_saddr);
    BPF_CORE_READ_INTO(&event->daddr, inet, inet_daddr);
    BPF_CORE_READ_INTO(&event->sport, inet, inet_sport);
    BPF_CORE_READ_INTO(&event->dport, inet, inet_dport);

    bpf_ringbuf_submit(event, 0);
    return 0;
}

char _license[] SEC("license") = "GPL";