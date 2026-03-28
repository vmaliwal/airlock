# Backends

## Lima backend

Implemented and intended for macOS hosts.

Requirements:
- `limactl` on PATH
- a Linux image retrievable by Lima
- passwordless sudo in guest user (Lima default)

Flow:
1. generate Lima instance config with mounts disabled
2. start disposable instance
3. upload payload bundle
4. execute guest runner
5. pull artifacts back
6. destroy instance

Current extra capability:
- on macOS, Airlock can auto-route host-toolchain-blocked autofix runs into Lima
- for those runs, the control plane can copy guest helper binaries and bootstrap a declared Go toolchain inside the VM before validation

## Firecracker backend

Implemented at the orchestration layer for Linux/cloud hosts.

Requirements:
- Firecracker installed on Linux runner host
- prepared kernel and rootfs
- `airlock-firecracker-host.sh` available on the Linux runner host
- SSH reachable host if orchestrated remotely

Flow:
1. upload payload bundle to Linux runner host
2. invoke host shim to create microVM
3. run guest payload inside microVM
4. collect artifacts from runner host
5. destroy microVM resources

## Why both exist

- Lima gives strong local isolation on macOS
- Firecracker gives stronger and cheaper scale-out isolation on Linux/cloud
