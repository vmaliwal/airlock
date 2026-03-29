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
- `airlock plan <repo-path|plan-input.json>`
- `airlock intake-compile <repo-path|plan-input.json> [output.json]`
- `airlock preflight <repo-path>`

Important probe statuses:
- `ready`: repo is runnable with no immediate structural warning
- `structurally_blocked`: missing source/bootstrap paths prevent an honest run
- `monorepo_target_required`: root-level scope is too broad; choose a concrete package/module target
- `host_toolchain_blocked_vm_runnable`: the host toolchain is too old, so local validation should stop and the repo should be routed into a VM-backed run instead
- `bootstrap_needed_vm_preferred`: bootstrap/install setup is likely needed before honest execution
- `partial_runnable_scope`: a concrete subdir/package scope is selected and should stay narrow
- `env_config_blocked`: execution context is still underspecified

Current warning taxonomy:
- `service_dependent`
- `integration_blocked`
- `flaky_candidate`

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

Execution note:
- campaign execution now uses the same compiled-contract execution helper path as other backend-driven flows
- compiled research contracts now carry a concrete plan object when a real local planning target exists
- compile-time plan synthesis is now omitted when that local target context is unavailable, rather than fabricating control-repo context
- bug intake can now compile directly into runnable read-only research contracts via `airlock intake-compile ...`
- run artifacts now include `execution-policy.json` so host-exception/provenance is visible after execution
- this reduces backend/prereq drift between campaign mode and direct contract execution

Autofix and planning learning:
- attempt lessons are stored in `lessons.jsonl`
- autofix ranking uses prior success/failure, mutation kind, and optional `fingerprint_hints`
- `airlock plan` now ranks mutation families using:
  - repo-type defaults
  - failure-text-derived fingerprint hints
  - prior lessons
- `airlock plan` accepts either a repo path or a JSON plan input carrying issue URL, failing command, and failure text
- set `AIRLOCK_LESSONS_ROOT` to point planning at a broader lesson corpus
- this is an early step toward fingerprint-aware planning instead of static candidate order

Supported campaign inputs:
- a single research contract with a top-level `campaign` section
- a campaign plan with `entries[]` pointing at multiple research contracts
