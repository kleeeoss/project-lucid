# Lucid-CI — Dynamic Sandbox Runner (`/infra-ai/sandbox`)

**Domain Owner:** Garv (AI & Cloud Architect)  
**Security Model:** Least-privilege ephemeral container detonation.

## Isolation Guarantees
1. **Network Denial:** Container must ALWAYS be executed with `--network none`.
2. **Filesystem Defense:** Container root filesystem is mounted `--read-only`, with writable `/workspace` and `/tmp` mounted via memory tmpfs.
3. **Capability Dropping:** All Linux kernel capabilities dropped (`--cap-drop ALL`).
4. **Resource Caps:** Container hard memory ceiling `--memory 512m`, CPU ceiling `--cpus 1.0`, process limit `--pids-limit 100`.
5. **Execution Timeout:** 60-second hard watchdog timer.
6. **Kernel Proxy (Cloud EC2):** Executed using gVisor user-space kernel (`--runtime=runsc`).

## Build Command
```bash
docker build -f sandbox/Dockerfile.runner -t lucid-sandbox-runner:latest sandbox/
