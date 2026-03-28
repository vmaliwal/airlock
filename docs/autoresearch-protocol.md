# Autoresearch Protocol

Airlock is the canonical home of the command-first autoresearch system.

## Core principles

1. Measure before mutate
2. Command-first execution
3. Reproduce before mutate
4. Isolate before experiment
5. Separate repro scaffolding from candidate fixes
6. Narrow scope first
7. Prefer reversible changes
8. Validate harder than you patch
9. Fingerprint failures, not just logs
10. Draft PR, never silent merge
11. Log every failed attempt
12. Stop on uncertainty

## Layering

- `internal/research/*` owns protocol logic
- `internal/backend/*` owns secure execution backends
- `internal/guest/*` owns guest payload generation
- `internal/contract/*` owns contract schema

## Git-native direction

Autonomy should be git-centric, not patch-string-centric.

The intended repair loop is:
- reproduce on a clean repo state
- create an attempt mutation
- inspect diff / budget / patch artifact
- validate
- commit or reset cleanly between attempts

This keeps every attempt reversible, inspectable, and learnable.

## Why this moved out of Cowork

Cowork is a notes/second-brain workspace.
Airlock is the execution engine.
Autoresearch belongs with the execution engine, not with note-taking infrastructure.

## Verified OSS examples

These contracts have been run successfully through the Lima backend against public OSS repos:

- `examples/beats-kafka-alias-research.json`
- `examples/beats-kafka-panic-research.json`
- `examples/beats-httpjson-precision-research.json`

All runs:
- cloned the repo inside the guest VM
- reproduced a real failure before patching
- applied a bounded patch
- passed target and neighbor validation

## CLI

Probe / classification:
- `airlock probe <repo-path>`
- `airlock investigate <repo-path>`
- `airlock plan <repo-path>`
- `airlock preflight <repo-path>`

Important probe statuses:
- `ready`: host execution is viable
- `structurally_blocked`: missing source/bootstrap paths prevent an honest run
- `host_toolchain_blocked_vm_runnable`: the host toolchain is too old, so local validation should stop and the repo should be routed into a VM-backed run instead

Template scaffolding:
- `airlock template research`
- `airlock template campaign`
- `airlock template attempt`
- `airlock template autofix`

Native git-centric attempt execution:
- `airlock attempt-run <attempt.json>`
- `airlock autofix-run <autofix.json>`

For repos classified as `host_toolchain_blocked_vm_runnable`, both commands route into a disposable VM automatically when a VM backend is available.

Policy note:
- unknown repo code should not execute on the host by default
- use `AIRLOCK_ALLOW_HOST_EXEC_EXCEPTION=1` only for an explicit, declared host exception
- otherwise prefer VM-backed execution even when the host could technically run the repo

Research flows:
- `airlock research-validate <contract.json>`
- `airlock research-run <contract.json>`

Campaign flows:
- `airlock campaign-validate <contract.json>`
- `airlock campaign-run <contract.json>`

Autofix and planning learning:
- attempt lessons are stored in `lessons.jsonl`
- autofix ranking uses prior success/failure, mutation kind, and optional `fingerprint_hints`
- `airlock plan` now ranks mutation families using repo-type defaults plus prior lessons
- set `AIRLOCK_LESSONS_ROOT` to point planning at a broader lesson corpus
- this is an early step toward fingerprint-aware planning instead of static candidate order

Supported campaign inputs:
- a single research contract with a top-level `campaign` section
- a campaign plan with `entries[]` pointing at multiple research contracts
