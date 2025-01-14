#ifndef _CHMOD_H_
#define _CHMOD_H_

#include "syscalls.h"

struct chmod_event_t {
    struct kevent_t event;
    struct process_context_t process;
    struct container_context_t container;
    struct syscall_t syscall;
    struct file_t file;
    u32 mode;
    u32 padding;
};

int __attribute__((always_inline)) chmod_approvers(struct syscall_cache_t *syscall) {
    return basename_approver(syscall, syscall->setattr.dentry, EVENT_CHMOD);
}

int __attribute__((always_inline)) trace__sys_chmod(umode_t mode) {
    struct policy_t policy = fetch_policy(EVENT_CHMOD);
    if (is_discarded_by_process(policy.mode, EVENT_CHMOD)) {
        return 0;
    }

    struct syscall_cache_t syscall = {
        .type = EVENT_CHMOD,
        .policy = policy,
        .setattr = {
            .mode = mode & S_IALLUGO,
        }
    };

    cache_syscall(&syscall);

    return 0;
}

SYSCALL_KPROBE2(chmod, const char*, filename, umode_t, mode) {
    return trace__sys_chmod(mode);
}

SYSCALL_KPROBE2(fchmod, int, fd, umode_t, mode) {
    return trace__sys_chmod(mode);
}

SYSCALL_KPROBE3(fchmodat, int, dirfd, const char*, filename, umode_t, mode) {
    return trace__sys_chmod(mode);
}

int __attribute__((always_inline)) do_sys_chmod_ret(void *ctx, struct syscall_cache_t *syscall, int retval) {
    if (IS_UNHANDLED_ERROR(retval))
        return 0;

    struct chmod_event_t event = {
        .syscall.retval = retval,
        .file = syscall->setattr.file,
        .padding = 0,
        .mode = syscall->setattr.mode,
    };

    struct proc_cache_t *entry = fill_process_context(&event.process);
    fill_container_context(entry, &event.container);

    // dentry resolution in setattr.h

    send_event(ctx, EVENT_CHMOD, event);

    return 0;
}

SEC("tracepoint/handle_sys_chmod_exit")
int handle_sys_chmod_exit(struct tracepoint_raw_syscalls_sys_exit_t *args) {
    struct syscall_cache_t *syscall = pop_syscall(EVENT_CHMOD);
    if (!syscall)
        return 0;

    return do_sys_chmod_ret(args, syscall, args->ret);
}

int __attribute__((always_inline)) trace__sys_chmod_ret(struct pt_regs *ctx) {
   struct syscall_cache_t *syscall = pop_syscall(EVENT_CHMOD);
    if (!syscall)
        return 0;

    int retval = PT_REGS_RC(ctx);
    return do_sys_chmod_ret(ctx, syscall, retval);
}

SYSCALL_KRETPROBE(chmod) {
    return trace__sys_chmod_ret(ctx);
}

SYSCALL_KRETPROBE(fchmod) {
    return trace__sys_chmod_ret(ctx);
}

SYSCALL_KRETPROBE(fchmodat) {
    return trace__sys_chmod_ret(ctx);
}

#endif
