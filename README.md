# Airlock

Autonomous bug fixing inside disposable VMs.

Give Airlock a GitHub issue URL. It reproduces the bug, synthesizes bounded candidate fixes for supported bug classes, validates them in an isolated VM, and publishes a draft PR.

```bash
airlock fix https://github.com/owner/repo/issues/123
```

## What it does

```
resolve issue
  → clone repo
  → classify + route (host vs VM)
  → classify issue signals (service-dependent? setup vs repro command?)
  → reproduce bug in VM
  → synthesize candidate fixes
  → multi-round fix loop (with duplicate suppression + strategy switching)
  → promote winning attempt
  → emit review-packet.md + draft-pr.md
  → (optional) publish GitHub draft PR + issue comment
  → append to run ledger
```

## Install

```bash
go install github.com/vmaliwal/airlock/cmd/airlock@latest
```

Optional convenience installer:

```bash
curl -fsSL https://raw.githubusercontent.com/vmaliwal/airlock/main/install.sh | bash
```

Requires Go 1.23+. Homebrew is not the current distribution path.

## Quick start

```bash
# Fix a public GitHub issue
airlock fix https://github.com/owner/repo/issues/123

# View run metrics
airlock metrics

# Check VM backend prerequisites
airlock check
```

## Environment variables

| Variable | Purpose |
|----------|---------|
| `GITHUB_TOKEN` | GitHub auth — required for private repos and PR publishing |
| `AIRLOCK_GITHUB_CREATE_DRAFT_PR=1` | Enable automatic draft PR creation after a credible fix |
| `AIRLOCK_PLANNER_PROVIDER=anthropic` | Enable LLM-backed synthesis |
| `ANTHROPIC_API_KEY` | API key for LLM planner |
| `AIRLOCK_PLANNER_MODEL` | Override planner model (default: `claude-sonnet-4-5`) |
| `AIRLOCK_ALLOW_HOST_EXEC_EXCEPTION=1` | Allow host execution of repo code (use sparingly) |
| `AIRLOCK_METRICS_DIR` | Custom path for `runs.jsonl` ledger |
| `AIRLOCK_CUSTOMER_ID` | Customer identifier for multi-tenant metric rollups |
| `AIRLOCK_LESSONS_ROOT` | Path to a broader lesson corpus for planning |

## Backends

- **lima** — macOS local Linux VM via Lima / Virtualization.framework
- **firecracker** — Linux/cloud VM via Firecracker host shim

Lima is fully operational. Firecracker is implemented and documented; full parity requires a validated host shim driver. See `docs/backends.md`.

## `airlock fix` behaviour

- Resolves the GitHub issue and infers a reproduction command from the issue body
- Detects service-dependent issues (HMR, dev server, browser repro) early
- Flags setup-only commands vs real test assertions
- Runs a read-only VM reproduction to establish honest `repro_status`
- Synthesizes candidate fix attempts for supported bug classes:
  - Go: expected/got normalization, resource lifecycle (missing defer/close)
  - Python: unclosed code block, empty-string guard, isinstance/None type guard, missing return of accumulator
  - Optional LLM planner for broader coverage
- Runs a bounded multi-round autofix loop with:
  - duplicate attempt suppression
  - prior-round failure memory
  - mutation-kind strategy switching
  - winner promotion with checkpoint recording
- Emits `review-packet.md` and `draft-pr.md` artifacts always
- Optionally creates a GitHub draft PR and posts the link back to the issue

## Supported Tier-1 languages

Go, Python, TypeScript/JavaScript.

C# and other languages are classified and routed honestly but are outside the current fix promise.

## Inspection commands

Read-only. Safe to run anytime. No side effects.

```bash
airlock probe <repo>               # classify repo type and runability
airlock investigate <repo>         # full investigation report
airlock plan <repo|input.json>     # repair strategy without execution
airlock preflight <repo>           # routing decision (host vs VM)
airlock metrics [runs.jsonl]       # view run ledger and scorecards
```

## Advanced / escape-hatch commands

Use these when debugging Airlock itself or isolating a specific pipeline stage.
Normal usage should go through `airlock fix`.

```bash
airlock synthesize <input.json>          # generate candidate attempts only
airlock eval-planner <cases.json>        # run planner eval corpus
airlock intake-compile <input>           # compile issue intake to research contract

airlock attempt-run <attempt.json>       # run a single bounded attempt
airlock autofix-run <plan.json>          # run a prepared multi-attempt autofix plan
airlock research-run <contract.json>     # run a full research contract in VM
airlock research-validate <contract>     # validate research contract (no exec)
airlock campaign-run <plan.json>         # run a multi-issue campaign
airlock campaign-validate <plan>         # validate campaign plan (no exec)
airlock run <contract.json>              # run a raw VM contract
airlock validate <contract.json>         # validate a raw VM contract (no exec)
airlock template <type>                  # print a contract template
```

## Build and test

```bash
go build ./cmd/airlock
go test ./...
```

## Documentation

- `docs/autoresearch-protocol.md` — autoresearch and fix loop protocol
- `docs/backends.md` — Lima and Firecracker backend details
- `docs/contract.md` — research contract schema
- `docs/product-issues.md` — issue backlog
- `docs/next-phase-gaps.md` — roadmap and gap tracker
- `docs/security-model.md` — security model
- `docs/firecracker-host-shim.md` — Firecracker host shim contract
- `docs/firecracker-host-setup.md` — Linux bring-up guide
