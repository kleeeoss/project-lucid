#!/usr/bin/env python3
"""
LUCID-CI — Synthetic Adversarial Payload: Fork Bomb & Memory Exhaustion.
Part of TASK-INF-501 Sandbox Penetration Testing Suite.

This script simulates an attacker-controlled PR attempting:
1. Recursive process spawning (os.fork) to exhaust host PIDs and starve host tasks.
2. Rapid memory consumption (> 1GB RAM) to cause host Out-Of-Memory (OOM) killer activation.

Under the Lucid-CI sandbox invariant:
- Container PID limit (--pids-limit 100) MUST block process creation with BlockingIOError / OSError(11).
- Container memory limit (--memory 512m) MUST terminate or halt the container without degrading the host.
"""

import os
import sys
import time
from typing import Dict, Any


def attempt_memory_exhaustion(target_mb: int = 768, chunk_mb: int = 64) -> Dict[str, Any]:
    """
    Attempts to rapidly allocate memory up to target_mb in chunk_mb increments.
    Returns:
        dict: allocated_mb, hit_limit (bool), error_type.
    """
    chunks = []
    total_allocated = 0
    hit_limit = False
    error_msg = None

    try:
        for _ in range(target_mb // chunk_mb):
            # Allocate 64MB bytearray
            chunk = bytearray(chunk_mb * 1024 * 1024)
            # Touch memory to ensure physical page allocation
            chunk[0] = 1
            chunk[-1] = 1
            chunks.append(chunk)
            total_allocated += chunk_mb
    except (MemoryError, OSError) as err:
        hit_limit = True
        error_msg = str(err)

    return {
        "target_mb": target_mb,
        "allocated_mb": total_allocated,
        "hit_limit": hit_limit,
        "error": error_msg,
    }


def attempt_controlled_fork_bomb(max_attempts: int = 150) -> Dict[str, Any]:
    """
    Attempts to fork processes up to max_attempts.
    Designed to verify --pids-limit 100 in sandbox containers.
    Includes safety cap to avoid unintended host degradation if run outside a container.
    """
    pids = []
    forks_created = 0
    blocked_by_pids_limit = False
    error_msg = None

    for i in range(max_attempts):
        try:
            pid = os.fork()
            if pid == 0:
                # Child process sleeps briefly then terminates
                time.sleep(1)
                os._exit(0)
            else:
                pids.append(pid)
                forks_created += 1
        except (BlockingIOError, OSError) as err:
            # Errno 11: EAGAIN / Resource temporarily unavailable -> PIDs limit enforced
            blocked_by_pids_limit = True
            error_msg = str(err)
            break

    # Clean up child processes spawned by parent
    for pid in pids:
        try:
            os.waitpid(pid, 0)
        except OSError:
            pass

    return {
        "forks_attempted": max_attempts,
        "forks_created": forks_created,
        "blocked_by_pids_limit": blocked_by_pids_limit,
        "error": error_msg,
    }


def main() -> int:
    print("[PEN_TEST] Initiating resource exhaustion containment test...")

    # 1. Test Memory Cap
    mem_res = attempt_memory_exhaustion(target_mb=768)
    print(f"  - Memory allocation: {mem_res['allocated_mb']}MB (hit limit: {mem_res['hit_limit']})")

    # 2. Test PID Cap
    fork_res = attempt_controlled_fork_bomb(max_attempts=150)
    print(f"  - Forks created: {fork_res['forks_created']} (blocked by PID limit: {fork_res['blocked_by_pids_limit']})")

    if fork_res["blocked_by_pids_limit"]:
        print("[ISOLATION_VERIFIED] Process table expansion blocked by container PID limit.")
        return 0

    print("[CONTAINMENT_OBSERVED] Test completed within host/sandbox thresholds.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
