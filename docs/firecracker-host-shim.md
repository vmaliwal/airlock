# Firecracker Host Shim Contract

This document defines the contract Airlock expects from `airlock-firecracker-host.sh`.

Status:
- required for real Firecracker parity
- backend integration in Airlock now assumes this interface
- reference shim script now exists at `scripts/firecracker/airlock-firecracker-host.sh`
- end-to-end validation remains blocked until a real Linux/Firecracker driver path is implemented and validated

## Purpose

The host shim is the **small trusted host-side adapter** between Airlock's control plane and a Linux host capable of launching Firecracker microVMs.

In this repo, the reference shim is intentionally thin and delegates raw microVM mechanics to a narrower driver hook.

It is responsible for:
- preparing a disposable microVM root/work area
- copying declared files into the guest
- executing the guest contract script inside the guest
- exporting artifacts back to the declared artifact directory
- cleaning up VM resources afterward

It is **not** responsible for:
- planning
- repo probing
- mutation generation
- repair logic
- policy authoring

Those belong to Airlock proper.

## Required CLI

Command:

```bash
airlock-firecracker-host.sh run \
  --name <sandbox-name> \
  --contract <host-path-to-guest-run.sh> \
  --artifacts <host-artifacts-dir> \
  [--copy-in <host-src>:<guest-dst>] ...
```

### Arguments

| Flag | Required | Meaning |
|------|----------|---------|
| `run` | yes | run a single disposable microVM job |
| `--name` | yes | sandbox/job name; safe for filesystem and VM resource naming |
| `--contract` | yes | host path to the generated guest shell script |
| `--artifacts` | yes | host directory where the shim must place run outputs |
| `--copy-in` | no, repeatable | file to inject into guest before execution, format `HOST_SRC:GUEST_DST` |

## `--copy-in` semantics

Each `--copy-in` pair means:
- `HOST_SRC` exists on the Linux host where the shim runs
- copy that exact file into guest path `GUEST_DST`
- preserve executability where relevant
- parent directories inside guest must be created if needed

Current Airlock usage depends on these mappings for:
- `/tmp/airlock`
- `/tmp/airlock-researchguest`

The shim should treat `--copy-in` as a **bounded ingress list**, not an unrestricted host mount.

## Required output behavior

On success or failure, the shim must leave artifacts under `--artifacts`.

Minimum required outputs:
- `summary.json`

Strongly recommended outputs when available:
- full guest artifact directory copy/tarball
- guest serial console log
- host shim stderr/stdout logs

### `summary.json`

Airlock currently expects the guest run to create a summary file compatible with the guest contract flow. In practice this means the shim must ensure that the guest script can populate and export:

- `summary.json`
- optional `repo.patch`
- step logs
- other declared artifact files

The simplest honest behavior is:
1. make `--artifacts` available inside the guest as `/airlock/artifacts`
2. run the contract script
3. copy resulting files back to the same host artifacts directory

## Execution contract inside guest

The shim must execute the contract script as:
- an executable shell script inside the guest
- with `/airlock` writable
- with `/airlock/artifacts` writable and exported back out

Expected guest paths used by Airlock-generated scripts:
- `/airlock/artifacts`
- `/airlock/work`
- `/airlock/home`
- `/airlock/xdg/config`
- `/airlock/xdg/cache`
- `/airlock/xdg/data`
- `/airlock/tmp`

## Security invariants

The shim should preserve these invariants:

1. **Disposable execution**
   - every run gets a fresh VM or a fresh snapshot restore with isolated writable state

2. **Bounded ingress**
   - only the contract script and declared `--copy-in` files enter the guest intentionally

3. **Bounded egress**
   - only declared artifact paths leave the guest

4. **No implicit host repo execution**
   - the shim must not replace VM execution with host execution for convenience

5. **No ambient host mounts**
   - do not mount the Airlock source repo, operator home directory, or arbitrary host paths into the guest

6. **Cleanup**
   - VM resources, temp dirs, and sockets should be removed after the run

## Recommended implementation model

Recommended phases:

1. Validate args
2. Create per-run workdir
3. Prepare microVM rootfs/overlay
4. Copy in:
   - contract script
   - all `--copy-in` files
5. Start microVM
6. Run guest script
7. Export `/airlock/artifacts`
8. Destroy VM and cleanup

## Transport suggestions

Good implementation options:
- overlayfs or copy-on-write rootfs for per-run state
- vsock or a mounted scratch disk for artifacts
- a small jailer wrapper for Firecracker process lifecycle

Avoid for first implementation:
- broad host mounts
- mutable shared base rootfs
- hidden cache state affecting correctness

## Relationship to E2B / Modal / Firecracker-style systems

Design principles worth borrowing:
- immutable base image
- explicit copy-in / copy-out
- fast disposable overlay
- snapshotting only after correctness
- tiny trusted host shim

That is the model this shim contract is designed to support.

## Minimal acceptance test

A real shim implementation should pass this shape of test:

```bash
airlock-firecracker-host.sh run \
  --name smoke \
  --contract /tmp/guest-run.sh \
  --artifacts /tmp/airlock-firecracker-smoke \
  --copy-in /tmp/airlock:/tmp/airlock \
  --copy-in /tmp/airlock-researchguest:/tmp/airlock-researchguest
```

Expected result:
- microVM starts
- injected binaries are present at requested guest paths
- guest contract runs
- `summary.json` is written back to `/tmp/airlock-firecracker-smoke`

## Current gap

Airlock now supports the control-plane side of guest binary staging for Firecracker.

Remaining blockers for full parity:
- a real `airlock-firecracker-host.sh` implementation
- verified `--copy-in` support
- at least one validated end-to-end Firecracker research run
