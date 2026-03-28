# Architecture

Airlock separates the **control plane** from the **execution plane**.

## Control plane

Runs on the operator machine.

Responsibilities:
- validate run contracts
- probe/classify repo readiness
- preflight route decisions explicitly
- detect when host execution is blocked but VM execution is still viable
- generate backend-specific plans
- create disposable sandboxes
- upload guest payloads
- trigger guest execution
- collect declared artifacts
- destroy the sandbox

## Execution plane

Runs inside a disposable Linux VM.

Responsibilities:
- create a temp HOME and XDG dirs
- scrub environment variables
- apply network policy
- clone the target repository
- execute declared steps
- write structured artifacts
- export only declared outputs

## Backends

### Lima backend

Use case:
- macOS operator machine
- local disposable Linux VM using Lima / Apple Virtualization.framework

Properties:
- no host repo mounts
- no host home mounts
- temp guest workspace
- host-to-guest file copy only for the run payload

### Firecracker backend

Use case:
- Linux/cloud runner host
- stronger isolation using a microVM

Properties:
- intended for ephemeral cloud execution hosts
- host orchestration occurs over SSH or local command execution on the Linux runner
- guest payload is uploaded, executed, and destroyed with the VM

## Trust boundaries

1. Host is trusted for orchestration only.
2. Guest VM is treated as disposable and potentially hostile after repo code runs.
3. Artifact export is allowlisted.
4. No host secrets are inherited unless explicitly allowlisted.

## Non-goals

- running untrusted repos directly on the host
- sharing host home directories into sandboxes
- silently allowing unrestricted network egress
