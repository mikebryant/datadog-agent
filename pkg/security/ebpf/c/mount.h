#ifndef _MOUNT_H_
#define _MOUNT_H_

#include "syscalls.h"

#define FSTYPE_LEN 16

struct mount_event_t {
    struct kevent_t event;
    struct process_context_t process;
    struct container_context_t container;
    struct syscall_t syscall;
    u32 mount_id;
    u32 group_id;
    dev_t device;
    u32 parent_mount_id;
    unsigned long parent_ino;
    unsigned long root_ino;
    u32 root_mount_id;
    u32 padding;
    char fstype[FSTYPE_LEN];
};

SYSCALL_COMPAT_KPROBE3(mount, const char*, source, const char*, target, const char*, fstype) {
    struct syscall_cache_t syscall = {
        .type = EVENT_MOUNT,
    };

    cache_syscall(&syscall);
    return 0;
}

SEC("kprobe/attach_recursive_mnt")
int kprobe__attach_recursive_mnt(struct pt_regs *ctx) {
    struct syscall_cache_t *syscall = peek_syscall(EVENT_MOUNT);
    if (!syscall)
        return 0;

    syscall->mount.src_mnt = (struct mount *)PT_REGS_PARM1(ctx);
    syscall->mount.dest_mnt = (struct mount *)PT_REGS_PARM2(ctx);
    syscall->mount.dest_mountpoint = (struct mountpoint *)PT_REGS_PARM3(ctx);

    // resolve root dentry
    struct dentry *dentry = get_vfsmount_dentry(get_mount_vfsmount(syscall->mount.src_mnt));
    syscall->mount.root_key.mount_id = get_mount_mount_id(syscall->mount.src_mnt);
    syscall->mount.root_key.ino = get_dentry_ino(dentry);
    resolve_dentry(dentry, syscall->mount.root_key, 0);

    struct super_block *sb = get_dentry_sb(dentry);
    struct file_system_type *s_type = get_super_block_fs(sb);
    bpf_probe_read(&syscall->mount.fstype, sizeof(syscall->mount.fstype), &s_type->name);

    return 0;
}

SEC("kprobe/propagate_mnt")
int kprobe__propagate_mnt(struct pt_regs *ctx) {
    struct syscall_cache_t *syscall = peek_syscall(EVENT_MOUNT);
    if (!syscall)
        return 0;

    syscall->mount.dest_mnt = (struct mount *)PT_REGS_PARM1(ctx);
    syscall->mount.dest_mountpoint = (struct mountpoint *)PT_REGS_PARM2(ctx);
    syscall->mount.src_mnt = (struct mount *)PT_REGS_PARM3(ctx);

    // resolve root dentry
    struct dentry *dentry = get_vfsmount_dentry(get_mount_vfsmount(syscall->mount.src_mnt));
    syscall->mount.root_key.mount_id = get_mount_mount_id(syscall->mount.src_mnt);
    syscall->mount.root_key.ino = get_dentry_ino(dentry);
    resolve_dentry(dentry, syscall->mount.root_key, 0);

    struct super_block *sb = get_dentry_sb(dentry);
    struct file_system_type *s_type = get_super_block_fs(sb);
    bpf_probe_read(&syscall->mount.fstype, sizeof(syscall->mount.fstype), &s_type->name);

    return 0;
}

int __attribute__((always_inline)) do_sys_mount_ret(void *ctx, struct syscall_cache_t *syscall, int retval) {
    if (retval)
        return 0;

    struct dentry *dentry = get_mountpoint_dentry(syscall->mount.dest_mountpoint);
    struct path_key_t path_key = {
        .mount_id = get_mount_mount_id(syscall->mount.dest_mnt),
        .ino = get_dentry_ino(dentry),
    };

    struct mount_event_t event = {
        .syscall.retval = retval,
        .mount_id = get_mount_mount_id(syscall->mount.src_mnt),
        .group_id = get_mount_peer_group_id(syscall->mount.src_mnt),
        .device = get_mount_dev(syscall->mount.src_mnt),
        .parent_mount_id = path_key.mount_id,
        .parent_ino = path_key.ino,
        .root_ino = syscall->mount.root_key.ino,
        .root_mount_id = syscall->mount.root_key.mount_id,
    };
    bpf_probe_read_str(&event.fstype, FSTYPE_LEN, (void*) syscall->mount.fstype);

    if (event.mount_id == 0 && event.device == 0) {
        return 0;
    }

    struct proc_cache_t *entry = fill_process_context(&event.process);
    fill_container_context(entry, &event.container);

    resolve_dentry(dentry, path_key, 0);

    send_event(ctx, EVENT_MOUNT, event);

    return 0;
}

SEC("tracepoint/handle_sys_mount_exit")
int handle_sys_mount_exit(struct tracepoint_raw_syscalls_sys_exit_t *args) {
    struct syscall_cache_t *syscall = pop_syscall(EVENT_MOUNT);
    if (!syscall)
        return 0;

    return do_sys_mount_ret(args, syscall, args->ret);
}

SYSCALL_COMPAT_KRETPROBE(mount) {
    struct syscall_cache_t *syscall = pop_syscall(EVENT_MOUNT);
    if (!syscall)
        return 0;

    int retval = PT_REGS_RC(ctx);
    return do_sys_mount_ret(ctx, syscall, retval);
}

#endif
