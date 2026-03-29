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
- Status: `validated`
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
- prior observed plan output reported `repoType: unknown` and no Python baseline commands
- validated fix now detects a concrete scope root and classifies the target correctly
- tests:
  - `GOTOOLCHAIN=local go test ./internal/research/... -count=1`
  - `GOTOOLCHAIN=local go test ./... -count=1`

### User impact
Weakens planning quality, hides correct baseline commands, and risks turning runnable package targets into unnecessary manual work.

### Expected
If the selected target path contains `pyproject.toml`/`package.json`/`go.mod`/`Cargo.toml`, Airlock should classify that concrete target correctly even when repo root is higher.

### Current workaround
Previously: point Airlock at a subdir that it already handles correctly or manually supply more context.

### Notes
Fixed by introducing concrete `scopeRoot` detection for targeted packages so classification/bootstrap hints use the package scope instead of only the git root.

## AIR-006 — Fix confidence and proof state are not first-class artifacts
- Status: `new`
- Severity: `sev2`
- Type: `eval`
- First seen: `2026-03-28`
- Reported by: `operator`
- Source repo: `multiple`
- Source issue: `n/a`
- Affected command: `airlock attempt-run`, `airlock autofix-run`, `airlock research-run`

### Problem
Airlock does not currently record a structured distinction between:
- plausible fix
- reproduced-before / passing-after fix
- exact issue-level fix confidence

As a result, operators can over-read a passing targeted test as stronger evidence than it really is.

### Evidence
- recent repo loops required manual reasoning about confidence level for:
  - `charmbracelet/gum#1017`
  - `cli/cli#12585`
- no first-class artifact fields currently encode:
  - proof that failure was reproduced before patch
  - proof that the exact repro passed after patch
  - validation scope ring
  - resulting confidence level

### User impact
Makes it harder to tell whether Airlock fixed:
- the exact reported bug
- the underlying bug class
- or only a plausible nearby issue

### Expected
Airlock should emit explicit proof-state / confidence metadata such as:
- `repro_status`
- `validation_scope`
- `fix_confidence`
- `confidence_reason`

### Current workaround
Operator manually inspects the evidence chain and narrates confidence in prose.

### Notes
This should become a product-level evaluation artifact, not only a human judgment step.

## AIR-007 — Bug intake still does not compile directly into runnable research execution
- Status: `new`
- Severity: `sev2`
- Type: `ux`
- First seen: `2026-03-29`
- Reported by: `repo validation`
- Source repo: `langchain-ai/langchain`
- Source issue: `#36194`
- Affected command: `airlock plan`, `airlock research-run`

### Problem
Airlock can intake a bug signal via `plan`, but it still does not provide a clean direct path from that intake into a runnable VM-backed research execution. For real issue work, operators still need to hand-author or adapt a research contract.

### Evidence
- `airlock plan` produced useful investigation output for `langchain-ai/langchain#36194`
- execution still requires a manually authored/adapted research contract to proceed into Lima-backed validation
- this is especially visible for Python repos where VM routing is correct but the top-level intake-to-run path remains operator-heavy

### User impact
The product still feels like separate tools rather than one issue-intake-to-fix flow.

### Expected
A bug signal should compile into an executable bounded run plan or research contract artifact without manual contract authoring in the common case.

### Current workaround
Operator adapts an existing research contract or authors a new one by hand.

### Notes
Manual contracts are still acceptable for isolating missing capability, but this gap should be tracked explicitly instead of normalized.

## AIR-008 — Compiled research plan can be synthesized against the wrong host repo context
- Status: `validated`
- Severity: `sev2`
- Type: `planner`
- First seen: `2026-03-29`
- Reported by: `repo validation`
- Source repo: `langchain-ai/langchain`
- Source issue: `#36194`
- Affected command: `airlock research-validate`

### Problem
When validating a research contract, Airlock may synthesize and embed a `plan` using the control-plane working directory instead of the target repo context. This can produce misleading compiled plans with incorrect repo roots, repo types, and candidate commands.

### Evidence
- prior `research-validate` for `langchain-empty-reasoning-36194-research.json` embedded a plan with:
  - `targetRepo: /Users/varun/repos/airlock`
  - candidate commands like `go test ./libs/core`
- the actual target repo is `langchain-ai/langchain` subdir `libs/core`
- validated fix now omits synthesized plans unless a real local `TargetPath` exists
- tests:
  - `GOTOOLCHAIN=local go test ./internal/research/... -count=1`
  - `GOTOOLCHAIN=local go test ./... -count=1`
- real command check:
  - `go run ./cmd/airlock research-validate sessions/.../langchain-unclosed-code-block-36186-research.json`
  - result no longer embeds a bogus compiled `plan`

### User impact
Makes compiled artifacts less trustworthy and can mislead operators or future automation layers that consume compiled plan data.

### Expected
If a plan is synthesized during compilation, it should use the intended target repo context or be omitted when that context is unavailable.

### Current workaround
Previously: treat the compiled plan as advisory only and rely on the explicitly authored contract for execution.

### Notes
Fixed by only synthesizing a plan during compilation when a real local planning target exists.
