# Firecracker Host Setup Guide

This guide describes how to prepare a Linux host for Airlock's Firecracker backend.

Status:
- reference setup
- intended to make the remaining parity gap concrete
- not yet validated end-to-end in this repo on a live Firecracker host

Read first:
- `docs/firecracker-host-shim.md`

## Goal

Bring up a Linux host that can satisfy this interface:

```bash
airlock-firecracker-host.sh run \
  --name smoke \
  --contract /path/to/guest-run.sh \
  --artifacts /path/to/artifacts \
  --copy-in /path/to/airlock:/tmp/airlock \
  --copy-in /path/to/airlock-researchguest:/tmp/airlock-researchguest
```

## Recommended host shape

| Item | Recommendation |
|------|----------------|
| OS | Ubuntu 24.04 LTS or similar Linux host |
| Hypervisor | Firecracker |
| Privilege model | dedicated runner account with access to required Firecracker resources |
| Workspace root | `/var/lib/airlock-firecracker` |
| Kernel image | prebuilt, pinned, read-only |
| Rootfs | pinned base image, reused read-only with disposable overlay |

## High-level architecture

Recommended layering:

1. **Airlock control plane** on operator machine
2. **Host shim** on Linux runner host
3. **Driver** on Linux runner host that does actual Firecracker launch
4. **Disposable microVM** for guest execution

The host shim should remain small.
The driver may evolve as needed for raw Firecracker mechanics.

## What to install on the Linux host

Minimum:
- `firecracker`
- `python3`
- `bash`
- `cp`, `mkdir`, `tar`
- a prepared kernel image
- a prepared rootfs image

Commonly useful:
- `iptables`
- `sudo`
- `jq`
- `rsync`

## Reference deployment layout

```text
/var/lib/airlock-firecracker/
  bin/
    airlock-firecracker-host.sh
    airlock-firecracker-driver.sh
  images/
    vmlinux
    ubuntu-rootfs.ext4
  runs/
    ...ephemeral per-run dirs...
```

Suggested environment:

```bash
export AIRLOCK_FIRECRACKER_STATE_DIR=/var/lib/airlock-firecracker/runs
export AIRLOCK_FIRECRACKER_DRIVER=/var/lib/airlock-firecracker/bin/airlock-firecracker-driver.sh
```

## Reference shim in this repo

Reference script path:

```text
scripts/firecracker/airlock-firecracker-host.sh
```

What it already does:
- parses the stable CLI
- validates inputs
- stages the guest contract
- stages all `--copy-in` files into a bounded ingress area
- writes a manifest JSON for the driver
- invokes a narrow driver hook
- checks that `summary.json` was exported

What it deliberately does **not** do yet:
- raw Firecracker microVM launch
- rootfs overlay creation
- guest drive attachment
- artifact disk/vsock plumbing

Those belong in the driver.

## Driver contract

The reference shim calls:

```bash
$AIRLOCK_FIRECRACKER_DRIVER run --manifest <manifest.json>
```

The driver is expected to:
- read the manifest
- create a disposable VM workspace
- inject:
  - staged contract script
  - staged `copyIn` files
- run the contract in the guest
- export guest artifacts so that `summary.json` ends up in the requested artifacts dir

### Manifest shape

Current manifest includes:
- `name`
- `workdir`
- `contract.hostPath`
- `contract.stagedPath`
- `artifacts.requestedHostDir`
- `artifacts.stagedHostDir`
- `copyIn[]` entries with:
  - `hostSource`
  - `stagedSource`
  - `guestDestination`

## Recommended driver design

Borrow the best ideas from E2B / Modal / Firecracker-style systems:

### 1. Immutable base image
- keep kernel and rootfs pinned
- don’t mutate the base rootfs per run

### 2. Disposable overlay
- create per-run writable overlay/diff disk
- destroy it after run completion

### 3. Explicit ingress only
- copy in only:
  - contract script
  - declared `copyIn` files
- avoid broad host mounts

### 4. Explicit egress only
- export only `/airlock/artifacts`
- don’t export guest HOME or arbitrary paths by default

### 5. Snapshot later
- correctness first
- snapshot/warm pool only after one honest end-to-end path works reliably

## First real acceptance target

A good first milestone is not a full research run.
It is a **smoke run** that proves:
- contract script enters guest
- copy-in file enters guest at requested destination
- guest script can write `/airlock/artifacts/summary.json`
- host sees the summary afterward

## Suggested smoke test

Use a guest contract that only verifies file presence and writes a tiny summary.

Success criteria:
- exit code 0
- `summary.json` exported
- one copied binary visible at expected guest path

## Security notes

Do not weaken these:
- no host repo mounts
- no implicit host fallback execution
- no shared mutable rootfs between runs
- no unbounded export of guest filesystem

## Honest current blocker

Airlock now has:
- control-plane-side Firecracker guest-binary staging
- explicit `--copy-in` contract
- a reference host shim

Still missing for full parity:
- real driver implementation on a Linux Firecracker host
- one validated end-to-end Firecracker run
