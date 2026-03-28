# Airlock

Airlock is a disposable VM sandbox runner for executing untrusted repository workflows outside the host machine.

It also contains the canonical command-first autoresearch engine: probe, classify, reproduce, attempt bounded fixes, validate, learn, and run those loops inside disposable VMs when host execution is not trustworthy or not viable.

Principles:
- untrusted repos never run on the host
- execution happens inside disposable VMs
- host secrets are scrubbed by default
- HOME/XDG dirs are temporary inside the guest
- network is denied by default and optionally allowlisted
- only declared artifacts come back

Backends:
- `lima` — macOS local Linux VM backend via Lima / Virtualization.framework
- `firecracker` — Linux/cloud backend via Firecracker host orchestration

## Why Go

Airlock is an infra/security tool. Go gives us:
- single static binary distribution
- strong subprocess/file/network ergonomics
- easy macOS/Linux support
- no Node runtime dependency on the operator machine

## Status

- Lima backend: implemented at the orchestration layer for macOS
- Firecracker backend: implemented at the orchestration layer for Linux/cloud hosts
- Guest runner payload generation: implemented
- End-to-end guest validation depends on host backend availability (`limactl` on macOS, Firecracker host shim on Linux)

## Build

```bash
go build ./cmd/airlock
```

## Test

```bash
go test ./...
```

## Check host prerequisites

```bash
./airlock check
```

## Probe a repo before running research

```bash
./airlock probe /path/to/repo-or-subdir
./airlock preflight /path/to/repo-or-subdir
```

Probe now distinguishes between:
- `ready` — host execution is viable
- `structurally_blocked` — missing sources/bootstrap makes honest execution impossible
- `host_toolchain_blocked_vm_runnable` — host execution should not proceed, but VM execution is still viable

When a repo falls into `host_toolchain_blocked_vm_runnable`, Airlock will prefer a VM-backed path instead of trying to validate on the host.
Currently this auto-routing applies to:
- `airlock attempt-run ...`
- `airlock autofix-run ...`

## Print contract templates

```bash
./airlock template research
./airlock template campaign
./airlock template attempt
./airlock template autofix
```

## Run a native git-centric attempt locally

```bash
./airlock attempt-run path/to/attempt.json
```

## Run a bounded multi-attempt autofix loop

```bash
./airlock autofix-run path/to/autofix.json
```

Autofix plans can now use:
- `search_replace`
- `insert_after`
- `replace_line`

And attempt ordering can be influenced by prior lessons plus optional `fingerprint_hints`.

## Example run

```bash
./airlock run examples/lima-contract.json
```

## Research flows

```bash
./airlock research-validate examples/beats-kafka-alias-research.json
./airlock research-run examples/beats-kafka-alias-research.json
./airlock campaign-validate examples/beats-kafka-alias-campaign.json
./airlock campaign-run examples/beats-kafka-alias-campaign.json
./airlock campaign-validate examples/beats-three-issue-campaign.json
./airlock campaign-run examples/beats-three-issue-campaign.json
```

See `docs/contract.md`, `docs/autoresearch-protocol.md`, and `examples/`.
