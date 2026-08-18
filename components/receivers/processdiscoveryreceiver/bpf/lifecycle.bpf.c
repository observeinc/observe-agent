// SPDX-License-Identifier: GPL-2.0 OR Apache-2.0
//
// eBPF programs for process lifecycle tracking. Attaches to
// sched_process_exec and sched_process_exit tracepoints and emits
// lightweight records into a ring buffer consumed by the Go-side
// lifecycle reader.

#include "vmlinux.h"

// BPF helper macros — avoids a build-time dependency on libbpf headers.
#define SEC(name) __attribute__((section(name), used))
#define __uint(name, val) int(*name)[val]
#define __type(name, val) typeof(val) *name

static long (*bpf_get_current_pid_tgid)(void) = (void *)14;
static long (*bpf_ktime_get_ns)(void) = (void *)5;
static long (*bpf_get_current_cgroup_id)(void) = (void *)80;
static void *(*bpf_ringbuf_reserve)(void *ringbuf, __u64 size, __u64 flags) = (void *)131;
static void (*bpf_ringbuf_submit)(void *data, __u64 flags) = (void *)132;

#define LIFECYCLE_EXEC 1
#define LIFECYCLE_EXIT 2

struct lifecycle_event {
	__u32 type;
	__u32 pid;
	__u32 parent_pid;
	__s32 fd;
	__u64 timestamp_ns;
	__u64 cgroup_id;
};

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 20);
} process_lifecycle SEC(".maps");

SEC("tracepoint/sched/sched_process_exec")
int tracepoint__sched_process_exec(void *ctx) {
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 pid = pid_tgid >> 32;

	struct lifecycle_event *evt =
		bpf_ringbuf_reserve(&process_lifecycle, sizeof(*evt), 0);
	if (!evt)
		return 0;

	evt->type = LIFECYCLE_EXEC;
	evt->pid = pid;
	evt->parent_pid = 0;
	evt->fd = 0;
	evt->timestamp_ns = bpf_ktime_get_ns();
	evt->cgroup_id = bpf_get_current_cgroup_id();

	bpf_ringbuf_submit(evt, 0);
	return 0;
}

SEC("tracepoint/sched/sched_process_exit")
int tracepoint__sched_process_exit(void *ctx) {
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 pid = pid_tgid >> 32;
	__u32 tid = (__u32)pid_tgid;

	// Only emit for the thread-group leader (pid == tid) to avoid
	// duplicate exit events from individual threads.
	if (pid != tid)
		return 0;

	struct lifecycle_event *evt =
		bpf_ringbuf_reserve(&process_lifecycle, sizeof(*evt), 0);
	if (!evt)
		return 0;

	evt->type = LIFECYCLE_EXIT;
	evt->pid = pid;
	evt->parent_pid = 0;
	evt->fd = 0;
	evt->timestamp_ns = bpf_ktime_get_ns();
	evt->cgroup_id = bpf_get_current_cgroup_id();

	bpf_ringbuf_submit(evt, 0);
	return 0;
}

char LICENSE[] SEC("license") = "GPL";
