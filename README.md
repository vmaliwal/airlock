# Airlock

Airlock is a disposable VM sandbox runner for executing untrusted repository workflows outside the host machine.

It also contains the canonical command-first autoresearch engine: probe, classify, reproduce, synthesize bounded candidate fixes for supported bug classes, validate, learn, and run those loops inside disposable VMs when host execution is not trustworthy or not viable.

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

Current parity note:
- Lima has the proven guest-binary path for `/tmp/airlock` and `/tmp/airlock-researchguest`
- Firecracker now stages guest helper binaries and passes explicit `--copy-in` mappings to the host shim
- the expected shim interface is documented in `docs/firecracker-host-shim.md`
- a reference shim now exists at `scripts/firecracker/airlock-firecracker-host.sh`
- host bring-up is documented in `docs/firecracker-host-setup.md`
- full Firecracker parity still depends on validated driver + end-to-end Linux/cloud runs

## Why Go

Airlock is an infra/security tool. Go gives us:
- single static binary distribution
- strong subprocess/file/network ergonomics
- easy macOS/Linux support
- no Node runtime dependency on the operator machine

Current minimum toolchain: **Go 1.23+**
- this is now required by the official Anthropic Go SDK used for planner-backed synthesis

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
./airlock investigate /path/to/repo-or-subdir
./airlock plan /path/to/repo-or-subdir
./airlock plan path/to/plan-input.json
./airlock intake-compile path/to/plan-input.json
./airlock intake-compile path/to/plan-input.json /tmp/issue-readonly.json
./airlock synthesize path/to/plan-input.json
./airlock synthesize path/to/plan-input.json /tmp/issue-autofix.json
./airlock preflight /path/to/repo-or-subdir
```

`plan-input.json` can include:
- `repoPath`
- `issueUrl`
- `failingCommand`
- `failureText`
- `notes`

`intake-compile` is the current bridge from issue intake to execution:
- it compiles a local bug signal into a runnable **read-only** research contract
- the generated artifact can go straight into `research-validate` or `research-run`
- this removes the old need to hand-author a starting research contract in the common local-intake case

`synthesize` is the first autonomy bridge for repair generation:
- by default it uses built-in narrow synthesis heuristics for supported bug classes
- it can now also use a planner-backed structured synthesis path when configured with:
  - `AIRLOCK_PLANNER_PROVIDER=anthropic`
  - `ANTHROPIC_API_KEY=...`
  - optional: `AIRLOCK_PLANNER_MODEL=claude-sonnet-4-5`
- planner-backed synthesis still returns bounded native Airlock mutation attempts, not arbitrary patch blobs
- the output can go straight into `autofix-run`
- current honest positioning remains: supported-class autonomous candidate-fix generation, not broad autonomous bug fixing yet

Probe now distinguishes between:
- `ready` — repo is runnable with no immediate structural warning
- `structurally_blocked` — missing sources/bootstrap makes honest execution impossible
- `monorepo_target_required` — repo root is too broad; choose a concrete package/module target
- `host_toolchain_blocked_vm_runnable` — host execution should not proceed, but VM execution is still viable
- `bootstrap_needed_vm_preferred` — bootstrap/install setup is likely needed before honest execution
- `partial_runnable_scope` — a concrete subdir/package scope is selected and should stay scoped
- `env_config_blocked` — execution context is still underspecified

Host execution policy:
- unknown repo code should not execute on the host by default
- `airlock attempt-run ...` and `airlock autofix-run ...` will route to a VM when possible unless an explicit host exception is declared
- declare an explicit host exception only with:
  - `AIRLOCK_ALLOW_HOST_EXEC_EXCEPTION=1`

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

Autofix/attempt mutations can now use:
- `search_replace`
- `insert_after`
- `replace_line`
- `create_file`
- `apply_patch`
- `ensure_line`
- `nil_guard`
- `error_return`

Planning/attempt ordering can now use:
- repo-type defaults
- failure-text-derived fingerprint hints
- prior lessons plus optional `fingerprint_hints`

Set `AIRLOCK_LESSONS_ROOT=/path/to/lessons` to feed a broader lesson corpus into planning.

## Example run

```bash
./airlock run examples/lima-contract.json
```

## Research flows

```bash
./airlock research-validate examples/beats-kafka-alias-research.json
./airlock research-run examples/beats-kafka-alias-research.json
./airlock intake-compile path/to/plan-input.json /tmp/issue-readonly.json
./airlock research-validate /tmp/issue-readonly.json
./airlock research-run /tmp/issue-readonly.json
./airlock campaign-validate examples/beats-kafka-alias-campaign.json
./airlock campaign-run examples/beats-kafka-alias-campaign.json
./airlock campaign-validate examples/beats-three-issue-campaign.json
./airlock campaign-run examples/beats-three-issue-campaign.json
```

Recent product/backlog progress reflected in the current tree:
- `AIR-005` validated: concrete package scope detection now classifies Python subdirs correctly
- `AIR-007` validated: bug intake now compiles into runnable read-only research contracts
- `AIR-008` validated: `research-validate` no longer fabricates bogus compiled plans against the control repo

See `docs/contract.md`, `docs/autoresearch-protocol.md`, `docs/product-issues.md`, and `examples/`.
