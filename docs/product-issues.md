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
- Status: `validated`
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
- validated fix now makes Python bootstrap explicit in generated intake contracts:
  - generated setup includes `bootstrap python venv`
  - Python commands are rewritten through `.venv/bin/python`
  - install policy is explicit for `requirements.txt` and `pyproject.toml`
- tests:
  - `GOTOOLCHAIN=local go test ./internal/research/... -count=1`
  - `GOTOOLCHAIN=local go test ./... -count=1`
- real command check via `airlock intake-compile` on LangChain text-splitters confirmed the generated setup and command policy

### User impact
Causes avoidable bootstrap failures or extra operator work in Python repos.

### Expected
Python repos should route toward clearer venv-first bootstrap defaults and stronger planning hints.

### Current workaround
Previously: manually author venv-aware setup commands.

### Notes
This now exists as explicit runtime policy for compiled intake flows. More advanced Python setup synthesis may still be needed for harder repos.

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
- Status: `validated`
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
- validated fix now emits first-class proof-state fields:
  - `repro_status`
  - `validation_scope`
  - `fix_confidence`
  - `confidence_reason`
- artifacts now include proof-state in:
  - `proof-state.json`
  - `validation-results.json`
  - `outcome.md`
- tests:
  - `GOTOOLCHAIN=local go test ./internal/research/... -count=1`
  - `GOTOOLCHAIN=local go test ./... -count=1`

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
Previously: operator manually inspects the evidence chain and narrates confidence in prose.

### Notes
This is now productized as a first proof layer; future work can still deepen the confidence model.

## AIR-007 — Bug intake still does not compile directly into runnable research execution
- Status: `validated`
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
- prior execution still required a manually authored/adapted research contract to proceed into Lima-backed validation
- validated fix added:
  - `airlock intake-compile <repo-path|plan-input.json> [output.json]`
- real command check:
  - compiled `langchain-36186-plan-input.json` into a runnable research contract artifact
  - `research-validate` accepted the generated artifact successfully
- tests:
  - `GOTOOLCHAIN=local go test ./internal/research/... -count=1`
  - `GOTOOLCHAIN=local go test ./cmd/airlock/... -count=1`
  - `GOTOOLCHAIN=local go test ./... -count=1`

### User impact
The product still feels like separate tools rather than one issue-intake-to-fix flow.

### Expected
A bug signal should compile into an executable bounded run plan or research contract artifact without manual contract authoring in the common case.

### Current workaround
Previously: operator adapts an existing research contract or authors a new one by hand.

### Notes
Fixed by adding an executable intake compiler that emits runnable read-only research contracts from local bug intake. This is an honest bridge, not yet full mutate-contract synthesis.

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

## AIR-009 — Planner-backed autonomous attempt synthesis is still too narrow
- Status: `in_progress`
- Severity: `sev1`
- Type: `planner`
- First seen: `2026-03-29`
- Reported by: `operator`
- Source repo: `multiple`
- Source issue: `n/a`
- Affected command: `airlock synthesize`, `airlock autofix-run`

### Problem
Airlock now has a first autonomy bridge via `airlock synthesize`, but candidate-fix generation is still limited to a narrow set of heuristic bug classes. The product still lacks a general planner-backed path from reproduced failure evidence into multiple structured mutation attempts.

### Evidence
- current synthesis is validated for a narrow supported set:
  - unclosed code-block EOF preservation
  - empty-string reasoning-content guard tightening
- `autofix-run` can execute structured attempts well once they exist
- the main remaining gap is generating those attempts broadly from failure fingerprints, repo shape, and relevant code context

### User impact
Without broader planner-backed synthesis, Airlock remains closer to a safe autoresearch/repair executor than a general autonomous bug fixer.

### Expected
Airlock should support an end-to-end planner loop:
- bug signal / repro / fingerprint
- relevant files + narrowed context
- LLM planner generates multiple structured candidate attempts in Airlock mutation schema
- lessons rank them
- `autofix-run` executes them
- proof state and reviewer-facing outputs summarize the result

### Current workaround
Use the narrow heuristic synthesis path where supported, or hand-author attempts/contracts for unsupported bug classes.

### Notes
A first provider-backed implementation now exists behind `airlock synthesize` when configured with `AIRLOCK_PLANNER_PROVIDER=anthropic` plus `ANTHROPIC_API_KEY`.
It packages investigation context, candidate file snippets, and allowed mutation kinds into a structured planner call, then normalizes the response back into native Airlock `AttemptFile` mutations.

Additional progress now shipped:
- first planner eval harness via `airlock eval-planner`
- stronger planner file/context narrowing via token scoring, symbol extraction, path-aware ranking, and simple source/test pairing
- broader validated heuristic coverage including a Go expected/got normalization class

This remains in progress because the product still needs:
- broader eval corpus and top-N proof across more real repos
- broader bug-class coverage
- stronger reviewer-facing PR summary output
- more evidence across real OSS repos

## AIR-010 — Top-level UX is still too operator-heavy
- Status: `in_progress`
- Severity: `sev1`
- Type: `ux`
- First seen: `2026-03-29`
- Reported by: `operator`
- Source repo: `multiple`
- Source issue: `n/a`
- Affected command: `airlock fix <github-issue-url>`

### Problem
Airlock’s underlying capabilities are improving, but the main user flow still requires operator familiarity with lower-level commands like `plan`, `intake-compile`, `synthesize`, and `autofix-run`. The intended product entry point should be one visible command that resolves an issue URL, shows progress, and produces a real output artifact.

### Evidence
- current flow still exposes plan-input files and multiple subcommands to reach a repair attempt
- roadmap intent is converging on `airlock fix <github-issue-url>` with visible progress and tangible output

### User impact
Makes the product feel like an expert toolchain rather than a simple autonomous fixing product.

### Expected
Airlock should provide a top-level command like:
- `airlock fix <github-issue-url>`
that performs intake, routing, repro, synthesis, attempt execution, proof capture, and PR/output preparation with clear progress stages.

### Current workaround
Use the lower-level command ladder directly, or the early `airlock fix <github-issue-url>` path for public GitHub issues.

### Notes
A first implementation now exists:
- `airlock fix <github-issue-url>` resolves the issue, clones the repo, attempts readonly reproduction when it can infer a command, synthesizes candidate fixes, executes autofix, and prints visible progress

This issue remains open because it still needs:
- tighter repro inference from issue content beyond the currently recognized command forms
- richer progress/proof presentation
- more coverage across real repos
- better private-repo and authenticated integration support
- simple distribution via:
  - primary install `go install github.com/vmaliwal/airlock/cmd/airlock@latest`
  - optional convenience installer `install.sh`
  - explicitly no Homebrew path for now

## AIR-011 — VM helper binary builds are pinned to local Go toolchain
- Status: `validated`
- Severity: `sev1`
- Type: `runtime`
- First seen: `2026-03-29`
- Reported by: `repo validation`
- Source repo: `elastic/beats`
- Source issue: `#49491`
- Affected command: `airlock fix <github-issue-url>`, VM-backed `autofix-run` / `research-run`

### Problem
When Airlock routes into a VM-backed path that requires building helper binaries like `/tmp/airlock` or `/tmp/airlock-researchguest`, the host-side build still forces `GOTOOLCHAIN=local`. If the locally installed Go is older than Airlock's own module requirement, VM execution fails before reproduction starts.

### Evidence
- real `airlock fix https://github.com/elastic/beats/issues/49491` run reached VM routing, then failed with:
  - `go.mod requires go >= 1.23.0 (running go 1.21.3; GOTOOLCHAIN=local)`
- failure occurred while building `./cmd/researchguest` for Lima guest injection

### User impact
Breaks the top-level issue flow and any VM-backed execution path on hosts whose default Go install is older than the repo's required toolchain, even when the operator otherwise has `GOTOOLCHAIN=auto` available.

### Expected
Airlock should build its own guest helper binaries using a toolchain policy consistent with the main CLI invocation, rather than hard-pinning to an older local toolchain.

### Current workaround
Upgrade the host's installed Go manually so `GOTOOLCHAIN=local` is sufficient.

### Notes
This is a real product/runtime regression exposed by the new Go 1.23 planner dependency and the new `airlock fix` path.

## AIR-012 — Issue-provided repro scaffolding is ignored in readonly fix runs
- Status: `validated`
- Severity: `sev1`
- Type: `runtime`
- First seen: `2026-03-30`
- Reported by: `repo validation`
- Source repo: `elastic/beats`
- Source issue: `#49491`
- Affected command: `airlock fix <github-issue-url>`

### Problem
`airlock fix` can infer a failing command from an issue body, but it currently ignores issue-provided repro scaffolding such as temporary test files or setup snippets that must exist before the command becomes meaningful. As a result, readonly reproduction may execute successfully yet honestly report `not_reproduced` because the issue's minimal repro fixture was never materialized.

### Evidence
- real rerun of `airlock fix https://github.com/elastic/beats/issues/49491`
- readonly VM path correctly bootstrapped Go after the recent runtime fix
- reproduction command ran successfully, but output showed `ok ... [no tests to run]`
- issue body explicitly instructed Airlock to create `libbeat/common/kafka/zzz_repro_test.go` before running the test command
- resulting repro artifact reported `repro_status: not_reproduced`

### User impact
Blocks honest top-level reproduction on issues that include inline minimal repro files, especially bug-hunter style reports and issues that provide a temporary failing test snippet.

### Expected
Readonly issue runs should compile bounded repro scaffolding from the issue body into setup steps before executing the inferred reproduction command.

### Current workaround
Hand-author a readonly research contract or manually create the repro file before running the command.

### Notes
This is narrower than general planner autonomy. The immediate need is bounded support for issue-provided temporary repro files, not freeform issue execution.

Validated fix:
- readonly intake-compiled runs now extract bounded issue repro file scaffolding from fenced code blocks when the first line encodes a repo-relative file path (e.g. `// path/to/file_test.go`)
- that scaffolding is compiled into setup before executing the inferred reproduction command
- real rerun of `airlock fix https://github.com/elastic/beats/issues/49491` now reports `repro_status: reproduced` instead of `not_reproduced`

## AIR-013 — GitHub draft PR and reviewer packet output are still missing
- Status: `in_progress`
- Severity: `sev1`
- Type: `ux`
- First seen: `2026-03-30`
- Reported by: `operator`
- Source repo: `multiple`
- Source issue: `n/a`
- Affected command: `airlock fix <github-issue-url>`

### Problem
Airlock can now run a materially honest fix loop, but it still stops short of the main artifact humans expect: a draft PR or PR-grade reviewer packet. Without a clear maintainer-facing output, successful work remains hard to review, trust, and monetize.

### Evidence
- current artifacts include proof state, advancement decision, validation summaries, and patch/checkpoint data
- `airlock fix` still does not create a GitHub draft PR, PR comment, or polished reviewer packet by default
- `docs/next-phase-gaps.md` still lists PR/reviewer-facing outputs as an open top-level gap

### User impact
Blocks the first commercially legible workflow. Even when Airlock does the technical work, the result is still too internal and operator-mediated.

### Expected
Airlock should produce a GitHub-first review artifact by default:
- draft PR body or ready-to-post PR packet
- issue summary
- reproduction summary
- root cause
- fix rationale
- evidence table
- residual uncertainty

### Current workaround
Operator manually converts internal artifacts into a PR description or hand-opens a PR.

### Notes
Recommendation is to solve this first for GitHub before building a broader output-adapter/plugin system.

Progress now shipped:
- `airlock fix` writes a maintainer-oriented `review-packet.md`
- `airlock fix` also writes a first `draft-pr.md` artifact for GitHub-first reviewer output
- these artifacts surface directly in `FixResult` as `reviewPacketPath` and `draftPRPath`

This issue remains open because automated GitHub draft PR creation/posting is still incomplete.

Additional progress now shipped:
- `airlock fix` can optionally attempt GitHub draft PR publication when explicitly enabled with `AIRLOCK_GITHUB_CREATE_DRAFT_PR=1`
- publication is gated on current run evidence and requires `GITHUB_TOKEN`
- the first implementation creates a branch from the promoted fix state, pushes it, creates a draft PR via the GitHub API, and posts the PR link back to the issue when possible

Remaining gap:
- this path still needs live end-to-end validation on a safe target plus follow-up PR/comment polish

## AIR-014 — Private repo auth inside the guest is still missing
- Status: `in_progress`
- Severity: `sev1`
- Type: `runtime`
- First seen: `2026-03-30`
- Reported by: `operator`
- Source repo: `private/enterprise`
- Source issue: `n/a`
- Affected command: `airlock fix <github-issue-url>`, VM-backed clone/research/autofix flows

### Problem
Airlock can clone and validate public repos, but private-repo usage is still blocked because guest-side git/auth setup is not yet first-class. This blocks one of the most important real-world deployment paths for paying teams.

### Evidence
- current documented flows focus on public GitHub issue URLs and public clone-based validation
- commercial planning identified lack of private repo auth in the guest as a first-order blocker
- no first-class guest credential injection path is documented as a validated capability

### User impact
Blocks design-partner and enterprise-style usage on the repos that matter most.

### Expected
Airlock should support bounded authenticated private-repo access inside the guest for GitHub-first workflows, with explicit policy and minimal credential exposure.

### Current workaround
Use public repos only, or manually prepare local trusted checkouts and avoid the intended end-to-end product flow.

### Notes
Recommendation is to solve this in the GitHub-first commercialization path before generalized multi-integration/plugin architecture.

Progress now shipped:
- guest env scrubbing now preserves explicitly allowlisted sensitive vars instead of dropping them unconditionally
- intake-compiled GitHub clone flows now allowlist `GITHUB_TOKEN`
- guest clone scripts now use bounded GitHub HTTPS auth when `GITHUB_TOKEN` is present and the clone target is `https://github.com/...`

This issue remains open because the end-to-end private-repo story still needs broader coverage beyond clone auth, including authenticated fetch/push paths and clearer credential lifecycle policy.

## AIR-015 — Autofix VM guest path broken for issue-flow cloned repos
- Status: `new`
- Severity: `sev1`
- Type: `runtime`
- First seen: `2026-03-30`
- Reported by: `repo validation`
- Source repo: `hashicorp/terraform`
- Source issue: `#38302`
- Affected command: `airlock fix <github-issue-url>`

### Problem
`airlock fix` clones a repo into a macOS temp dir (e.g. `/var/folders/.../airlock-fix-hashicorp-...`). When synthesis generates autofix attempts, `CompileAutofixPlanToVMContract` uses that host path as the plan's repo path. Inside the Lima guest, that macOS temp path does not exist, so the guest script's `cd` command fails.

### Evidence
- `airlock fix https://github.com/hashicorp/terraform/issues/38302` after Slice 2
- synthesis generated 2 attempts (defer-close heuristic)
- autofix VM run failed with: `/tmp/guest-run.sh: line 105: cd: ../../../../../../../var/folders/.../airlock-fix-hashicorp-...: No such file or directory`

### User impact
Synthesis works but autofix execution breaks for any issue where `airlock fix` clones the repo to a host temp dir. Blocks the main `airlock fix` path from completing end-to-end even when synthesis succeeds.

### Expected
The autofix VM contract should clone the repo fresh inside the guest (same as readonly research runs) rather than referencing the host-side temp clone path.

### Current workaround
Use a pre-cloned local repo path and run `airlock autofix-run` directly.

### Notes
The readonly reproduction path already works correctly because `CompilePlanInputToRunContract` uses the issue's clone URL, not the local path. The same pattern needs to apply to the autofix VM compilation path when initiated from `airlock fix`.
