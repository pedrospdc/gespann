/* SPDX-License-Identifier: GPL-2.0 */
/* Minimal vmlinux.h for gespann Docker build */

#ifndef _VMLINUX_H_
#define _VMLINUX_H_

typedef unsigned char __u8;
typedef unsigned short __u16;
typedef unsigned int __u32;
typedef unsigned long long __u64;
typedef signed char __s8;
typedef signed short __s16;
typedef signed int __s32;
typedef signed long long __s64;

typedef __u16 __be16;
typedef __u32 __be32;
typedef __u64 __be64;
typedef __u32 __wsum;

// BPF map types
enum bpf_map_type {
    BPF_MAP_TYPE_HASH = 1,
    BPF_MAP_TYPE_RINGBUF = 27,
};

// BPF map update flags
enum {
    BPF_ANY = 0,
    BPF_NOEXIST = 1,
    BPF_EXIST = 2,
};

struct sock {
    void *__placeholder[64];
};

struct inet_sock {
    __u32 inet_saddr;
    __u32 inet_daddr;
    __u16 inet_sport;
    __u16 inet_dport;
    void *__placeholder[32];
};

#ifdef __TARGET_ARCH_arm64
struct pt_regs {
    __u64 regs[31];
    __u64 sp;
    __u64 pc;
    __u64 pstate;
};
#define PT_REGS_PARM1(x) ((x)->regs[0])
#define PT_REGS_PARM2(x) ((x)->regs[1])
#define PT_REGS_PARM3(x) ((x)->regs[2])
#else
struct pt_regs {
    unsigned long long regs[16];
};
#define PT_REGS_PARM1(x) ((x)->regs[7])
#define PT_REGS_PARM2(x) ((x)->regs[6])
#define PT_REGS_PARM3(x) ((x)->regs[5])
#endif

#endif /* _VMLINUX_H_ */