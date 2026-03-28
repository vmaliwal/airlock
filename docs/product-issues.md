# Airlock Product Issues

This file is the durable, issue-style backlog for failures exposed by real usage.

Purpose:
- capture gaps before fixing them
- treat repo validations like real customer reports
- separate **issue intake** from **implementation**
- keep evidence attached to every product problem

Rule:
- when a repo run exposes an Airlock failure, log it here first unless the run is blocked by an actively known issue already listed here
- do not silently fix newly discovered gaps without recording them

## Status values

- `new`
- `triaged`
- `planned`
- `in_progress`
- `validated`
- `closed`
- `wont_fix`

## Severity values

- `sev0` — safety/integrity risk
- `sev1` — blocks core user flow
- `sev2` — painful but workaround exists
- `sev3` — polish / UX / non-core

## Types

- `safety`
- `planner`
- `classification`
- `mutation`
- `runtime`
- `backend`
- `artifact`
- `ux`
- `docs`
- `eval`

## Issue template

```md
## AIR-XXX — Short title
- Status: `new`
- Severity: `sev1`
- Type: `runtime`
- First seen: `YYYY-MM-DD`
- Reported by: `repo validation` | `operator` | `customer`
- Source repo: `owner/repo`
- Source issue: `#123` or URL
- Affected command: `airlock ...`

### Problem
One-paragraph description of the failure.

### Evidence
- artifact:
- summary:
- stderr:
- fingerprint:

### User impact
What this blocks in the product flow.

### Expected
What Airlock should have done instead.

### Current workaround
If any.

### Notes
Any hypotheses or follow-ups.
```

---

## AIR-001 — Firecracker host shim lacks validated end-to-end driver path
- Status: `planned`
- Severity: `sev1`
- Type: `backend`
- First seen: `2026-03-28`
- Reported by: `repo validation`
- Source repo: `multiple`
- Source issue: `n/a`
- Affected command: `airlock research-run ...` with Firecracker backend

### Problem
Airlock's Firecracker backend now stages guest binaries and passes `--copy-in`, but there is still no validated end-to-end Linux/Firecracker driver path proving that a real microVM run completes and exports artifacts successfully.

### Evidence
- docs: `docs/firecracker-host-shim.md`
- docs: `docs/firecracker-host-setup.md`
- commit: `fc89243`
- commit: `212f80c`

### User impact
Blocks first-class Linux/cloud parity for the main safe execution path.

### Expected
A real Firecracker-backed run should execute guest scripts, honor `--copy-in`, and export `summary.json` plus declared artifacts.

### Current workaround
Use Lima-backed VM execution on macOS.

### Notes
This is now a narrow driver/validation problem, not a vague backend design problem.

## AIR-002 — Complex setup authored as inline shell is brittle
- Status: `triaged`
- Severity: `sev2`
- Type: `ux`
- First seen: `2026-03-28`
- Reported by: `repo validation`
- Source repo: `cli/cli`
- Source issue: `#12927`
- Affected command: `airlock research-run ...`

### Problem
Complex setup mutations encoded as inline shell commands are easy to misquote, hard to review, and cause contract-authoring failures unrelated to the actual repo bug.

### Evidence
- source learning recorded in `docs/next-phase-gaps.md`
- repo validation: `cli/cli` worktree-sync run

### User impact
Raises operator burden and causes false starts in bug reproduction/remediation setup.

### Expected
Airlock should prefer first-class file mutation helpers, patch assets, or compiled setup artifacts over brittle inline shell.

### Current workaround
Use patch-based setup or simpler bounded file-mutation helpers.

### Notes
This is a product-quality issue even if the underlying runtime can still succeed.

## AIR-003 — Python guest bootstrap policy is still too implicit
- Status: `triaged`
- Severity: `sev2`
- Type: `runtime`
- First seen: `2026-03-28`
- Reported by: `repo validation`
- Source repo: `langchain-ai/langchain`
- Source issue: `#36297`
- Affected command: `airlock research-run ...`

### Problem
Python repos often require venv-first bootstrap inside the guest, but this is still more a learned workaround than an explicit default policy and strategy layer.

### Evidence
- source learning recorded in `docs/next-phase-gaps.md`
- repo validation: `langchain-ai/langchain` (`libs/core`)

### User impact
Causes avoidable bootstrap failures or extra operator work in Python repos.

### Expected
Python repos should route toward clearer venv-first bootstrap defaults and stronger planning hints.

### Current workaround
Manually author venv-aware setup commands.

### Notes
Likely should become planner/runtime policy, not only docs.

## AIR-004 — Host toolchain policy is correct but underexplained
- Status: `triaged`
- Severity: `sev3`
- Type: `docs`
- First seen: `2026-03-28`
- Reported by: `repo validation`
- Source repo: `cli/cli`, `gum`, others
- Source issue: `n/a`
- Affected command: `airlock probe`, `airlock preflight`

### Problem
Some repos can technically pass on host via toolchain auto-download, while Airlock still classifies them as host-blocked under stricter local-toolchain policy. The behavior is intentional but easy to misread as inconsistency.

### Evidence
- source learning recorded in `docs/next-phase-gaps.md`

### User impact
Creates confusion about why Airlock routes to VM or stops despite apparent host viability.

### Expected
Policy stance should be explicit in CLI/docs/artifacts.

### Current workaround
Read the roadmap/policy docs.

### Notes
This is important for trust and operator understanding, even if not a runtime blocker.

## AIR-005 — Subdir package detection misses concrete Python target manifests
- Status: `new`
- Severity: `sev1`
- Type: `classification`
- First seen: `2026-03-28`
- Reported by: `repo validation`
- Source repo: `langchain-ai/langchain`
- Source issue: `#36186`
- Affected command: `airlock plan`, `airlock probe`, `airlock investigate`

### Problem
When targeting `libs/text-splitters` in LangChain, Airlock classified the target as `repoType: unknown` even though `libs/text-splitters/pyproject.toml` exists and the target path is already a concrete package scope.

### Evidence
- target path: `/Users/varun/repos/experiments/langchain/libs/text-splitters`
- manifest present: `libs/text-splitters/pyproject.toml`
- observed plan output reported `repoType: unknown` and no Python baseline commands

### User impact
Weakens planning quality, hides correct baseline commands, and risks turning runnable package targets into unnecessary manual work.

### Expected
If the selected target path contains `pyproject.toml`/`package.json`/`go.mod`/`Cargo.toml`, Airlock should classify that concrete target correctly even when repo root is higher.

### Current workaround
Point Airlock at a subdir that it already handles correctly or manually supply more context.

### Notes
This is a real product issue exposed by repo intake and should be fixed only after triage/prioritization, not silently during repo work.
