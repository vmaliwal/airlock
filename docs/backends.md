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
- host shim interface as defined in `docs/firecracker-host-shim.md`

Flow:
1. upload payload bundle to Linux runner host
2. invoke host shim to stage ingress and call the Firecracker driver
3. run guest payload inside microVM
4. collect artifacts from runner host
5. destroy microVM resources

Reference assets:
- shim contract: `docs/firecracker-host-shim.md`
- host setup guide: `docs/firecracker-host-setup.md`
- reference shim script: `scripts/firecracker/airlock-firecracker-host.sh`

Current parity note:
- Firecracker orchestration now stages guest helper binaries when contracts reference `/tmp/airlock` or `/tmp/airlock-researchguest`
- local mode passes explicit `--copy-in host:guest` mappings to `airlock-firecracker-host.sh`
- ssh mode uploads helper binaries to the remote workdir and passes the same `--copy-in` mappings remotely
- full parity still depends on the host shim implementing `--copy-in` correctly and on validated end-to-end Firecracker runs

## Why both exist

- Lima gives strong local isolation on macOS
- Firecracker gives stronger and cheaper scale-out isolation on Linux/cloud
