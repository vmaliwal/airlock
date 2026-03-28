# Next-Phase Gaps

Last updated: 2026-03-28
Baseline version: `v0.1.0` (`c93e806`)

This document is the durable gap tracker for Airlock.

Purpose:
- record the delta between the current implementation and the intended vision
- keep a running list of what is incomplete, risky, or underpowered
- append new learnings instead of re-deriving them from scratch
- anchor roadmap work in real evidence from runs against actual repositories
- serve as the single roadmap document for gaps, candidate repos, policy changes, and security posture

## Vision Reference

Target vision:
- safe command-first autoresearch
- git-native repair loops
- reproducible evidence before mutation
- bounded multi-attempt autofix
- lessons-informed improvement over time
- broad issue-class coverage
- VM-first execution substrate
- eventually: “give me a bug, I’ll fix it”

## Operating Policy

### New policy: no handcrafted repo contracts by default
Effective now:
- do not use hand-authored repo-specific research contracts as the normal path
- allow manual contracts only when the goal is debugging Airlock itself or isolating a missing Airlock capability
- every time a manual contract is required, record the missing planner/runtime capability here as a gap

Implication:
- normal repo work should start from a bug signal, not a handcrafted execution plan
- contracts remain as compiled/internal artifacts, not the primary operator interface

### Security posture
Required stance:
- do not run unknown OSS repo code directly on the host as the default path
- use VM-backed execution for untrusted repos:
  - Lima on macOS
  - Firecracker on Linux/cloud when parity is complete
- host-side work is allowed only for:
  - static inspection
  - git metadata inspection
  - non-executing file edits
  - tightly bounded local validation against repos already under explicit operator control
- if a repo requires execution and is not clearly trusted, route it into a disposable VM

Current assessment:
- current direction is materially safer than earlier host execution
- Lima-backed execution is the active safe path today
- Firecracker parity is still incomplete, so Linux/cloud security parity is not yet fully realized
- manual local repros of unknown code should be treated as exceptions, not the standard workflow

## Gap Summary

## 1. Repair planning is still weak
Current:
- Airlock can execute bounded attempts well.
- Mutation specs exist and can be ordered.
- Attempts are still mostly human-authored.

Gap:
- limited automatic synthesis of candidate fixes
- weak root-cause-to-strategy mapping
- no strong planner for “likely repairs for this fingerprint/class of failure”

Why it matters:
- the system is still closer to a powerful repair executor than an autonomous fixer

Desired next state:
- generate candidate attempts from fingerprints, repo structure, and failure class
- support multiple repair families per bug class
- rank attempts based on more than hand ordering

## 2. Lessons exist, but self-improvement is shallow
Current:
- `lessons.jsonl`
- success/failure-aware ranking
- mutation-kind + fingerprint-aware ordering for autofix

Gap:
- no real policy engine over lessons
- weak generalization across repos/languages/failure families
- no persistent strategy selection layer beyond lightweight scoring

Why it matters:
- learning is being recorded, but not yet strongly changing future behavior

Desired next state:
- fingerprint-to-strategy priors
- attempt family scoring by issue class
- repo/ecosystem-aware strategy transfer

## 3. Research, attempt, autofix, and campaign modes are still not fully unified
Current:
- all four exist
- VM routing now applies to attempt and autofix
- research/campaign are strong but still somewhat separate mentally and structurally

Gap:
- mode transitions are still not one coherent ladder
- artifact conventions and route decisions should feel more uniform
- planning logic is duplicated conceptually across modes

Why it matters:
- the final system should feel like one engine at different scopes, not adjacent subtools

Desired next state:
- unified execution model
- shared preflight/routing logic across all modes
- consistent artifact model and summaries

## 4. Mutation taxonomy is still too narrow
Current:
- `search_replace`
- `insert_after`
- `replace_line`

Gap:
- missing common repair primitives:
  - nil guards
  - early return on error
  - import or require fixups
  - test gating patterns
  - config/bootstrap remediation
  - file creation/scaffold edits
  - AST-aware transformations
  - dependency/bootstrap fixes

Why it matters:
- current repair power is strongest on deterministic, localized text edits
- broader real-world bug fixing needs richer strategy families

Desired next state:
- a mutation library that maps naturally to recurring bug classes

## 5. Probe/classification taxonomy is incomplete
Current:
- structural blockers
- empty replace target detection
- host toolchain blocked vs VM runnable

Gap:
- limited classification of:
  - service-dependent failures
  - env/config blockers
  - flaky/stability failures
  - external-sidecar requirements
  - partial runnable scope (unit runnable, integration blocked)
  - bootstrap-needed but not structurally impossible repos

Why it matters:
- much of autonomy is routing, not mutation

Desired next state:
- richer status taxonomy
- more precise route recommendations
- clearer distinction between “can’t run”, “shouldn’t run here”, and “can run if enriched”

## 6. VM backend parity is incomplete
Current:
- Lima path is real and validated
- VM routing works for attempt/autofix

Gap:
- Firecracker parity is incomplete
- no fully validated guest runner parity on Linux/cloud
- backend behavior is not yet symmetric across all modes

Why it matters:
- the architecture explicitly called for both macOS local and Linux/cloud first-class backends

Desired next state:
- Firecracker parity with Lima for research/attempt/autofix/campaign
- validated Linux/cloud runs with artifact symmetry

## 7. Safety policy is stronger, but still partly implicit
Current:
- disposable VMs
- scrubbed env
- network allowlisting
- bounded diffs and path budgets

Gap:
- bootstrap vs validation policy still deserves more explicit rules
- allowed outbound network policy is still coarse
- unsafe-to-continue conditions should be more explicit
- mutation class review thresholds are not yet formalized

Why it matters:
- the vision was deterministic safety and enforceable policy, not ad hoc care

Desired next state:
- more safety policy encoded as first-class checks
- stronger safety docs linked to behavior

## 8. Evals are not yet first-class
Current:
- many real validations have been run manually
- several real OSS wins exist

Gap:
- no dedicated eval suite yet
- no persistent benchmark corpus
- no cross-version score tracking
- no quality dashboard for repair performance

Why it matters:
- without evals, improvement remains anecdotal

Desired next state:
- first-class eval tasks and scoring
- benchmark suites for classification, routing, reproduction, repair, and validation

## 9. Top-level UX is still operator-heavy
Current:
- powerful CLI primitives exist
- route-to-VM now reduces some operator burden

Gap:
- still too much manual contract/attempt authoring
- no polished “here is the bug, go work it” top-level flow
- summary artifacts are useful but not yet ideal operator UX

Why it matters:
- the long-term promise is agentic bug fixing, not just a collection of capable subcommands

Desired next state:
- tighter end-to-end task entrypoint
- better auto-scoping, auto-planning, and final summaries

## 10. PR-quality outputs are underdeveloped
Current:
- patches and machine-readable summaries exist
- attempt/campaign artifacts are useful

Gap:
- final reviewer-facing output is still thin
- missing polished generation of:
  - issue summary
  - repro summary
  - root cause
  - fix rationale
  - evidence table
  - residual uncertainty
  - draft PR body

Why it matters:
- real bug fixing ends in something maintainers can consume, not only internal artifacts

Desired next state:
- first-class review packet / PR draft generation

## Priority View

Highest-priority gaps for the next phase:
0. codify and enforce: no execution of unknown repo code on host unless an explicit exception is declared
1. stronger repair planning
2. better self-improvement from lessons
3. unified engine behavior across modes
4. broader mutation taxonomy
5. broader probe/classification taxonomy
6. full backend parity
7. stronger explicit safety policy
8. first-class evals
9. higher-level bug intake UX
10. better PR/reviewer-facing outputs

## Execution Plan

This section translates the roadmap into concrete workstreams. The immediate focus is Priority 0 plus items 1 through 7, with the new no-handcrafted-contract policy enforced throughout.

### Workstream 0 — Host execution policy enforcement
Addresses:
- Priority 0
- Gap 7

Deliverables:
- explicit host-execution gate in CLI/runtime paths
- default VM routing for unknown repo execution when a backend is available
- explicit override path for declared host exceptions
- preflight output that makes policy decisions visible

Success signals:
- unknown repo code no longer executes on host silently
- every host execution is either trusted-by-design or an explicit visible exception

### Workstream 1 — Planner and strategy synthesis
Addresses:
- Gap 1

Deliverables:
- first-class `investigate` / `plan` flow
- bug-signal intake from issue URL, failing command, or failure text
- automatic target/subdir selection
- reproduction candidate generation and ranking
- validation candidate generation and ranking
- strategy library keyed by failure fingerprint and repo class

Success signals:
- fewer hand-authored plans
- better first-attempt localization
- better first-attempt success rate on repeated issue families

### Workstream 2 — Lessons-driven self-improvement
Addresses:
- Gap 2

Deliverables:
- persistent strategy priors driven by previous runs
- fingerprint-to-strategy ranking
- repo/ecosystem-aware bootstrap priors
- mutation-family priors that materially affect future attempt ordering

Success signals:
- measurable reduction in attempts-to-success over time
- new repos benefit from prior repo learnings without manual transfer

### Workstream 3 — Unify runtime modes into one engine
Addresses:
- Gap 3

Deliverables:
- one shared planning/routing model across `attempt`, `autofix`, `research`, and `campaign`
- common artifact and summary conventions
- shared route explanations and stop conditions

Success signals:
- same repo/bug signal leads to coherent behavior regardless of entrypoint
- less duplicated decision logic

### Workstream 4 — Expand mutation taxonomy and make mutations typed
Addresses:
- Gap 4

Deliverables:
- first-class typed actions instead of opaque inline shell as the default:
  - create file
  - edit file
  - search/replace
  - insert after
  - replace line
  - apply patch
  - run command
  - bootstrap helpers
- additional repair families:
  - nil guards
  - early error returns
  - import/config/bootstrap fixups
  - scaffold/file creation
  - dependency/bootstrap remediations
  - eventually AST-aware transforms

Success signals:
- fewer brittle repo-specific shell blobs
- more reusable and learnable repair actions

### Workstream 5 — Expand probe and classification taxonomy
Addresses:
- Gap 5

Deliverables:
- richer routing states for:
  - service-dependent failures
  - env/config blockers
  - flaky failures
  - partial-runnable scopes
  - bootstrap-needed repos
- better route recommendations and stop reasons

Success signals:
- fewer false starts
- cleaner distinction between blocked, VM-runnable, partially-runnable, and enriched-required repos

### Workstream 6 — Backend parity
Addresses:
- Gap 6

Deliverables:
- Firecracker parity with Lima for guest runner behavior, routing, and artifact export
- validated Linux/cloud runs for the same bug-fixing flows already proven on Lima

Success signals:
- same research classes succeed on both backends
- Linux/cloud is no longer a design-only backend

### Workstream 7 — Explicit safety policy enforcement
Addresses:
- Gap 7

Deliverables:
- safety matrix encoded as enforceable checks
- clearer host-vs-VM execution rules
- explicit unsafe-to-continue stop conditions
- stronger bootstrap vs validation network policy enforcement
- policy checks that prevent casual execution of unknown repo code on the host

Success signals:
- security posture is inspectable, deterministic, and enforced by code
- host execution of unknown code becomes an exceptional, visible override path

### Workstream 8 — First-class evals
Addresses:
- Gap 8

Deliverables:
- `evals/` benchmark corpus
- machine-readable eval summaries
- cross-version score tracking

Success signals:
- progress is measurable instead of anecdotal

### Workstream 9 — Higher-level bug intake UX
Addresses:
- Gap 9

Deliverables:
- top-level `fix <bug-signal>` style operator UX
- contracts become compiled/internal artifacts rather than the main authoring surface

Success signals:
- operator burden decreases significantly

### Workstream 10 — PR/reviewer-facing outputs
Addresses:
- Gap 10

Deliverables:
- issue summary
- repro summary
- root cause
- fix rationale
- evidence table
- residual uncertainty
- draft PR body

Success signals:
- outputs are maintainer-consumable by default

## Real Evidence So Far

Repos/issues already used as grounding evidence:
- `elastic/beats`
  - `#49491`
  - `#49376`
  - `#49599`
- `ashupednekar/litefunctions/portal`
  - default `go test` ergonomics / integration-test gating issue
- `sindresorhus/execa`
  - synthetic and safe worktree validations
- `meetcli`
  - honest structural blocker classification

## Append Log

Use this section as an ongoing journal of gap discoveries and refinements.

### 2026-03-28
- Codified Priority 0: unknown repo code should not execute on host unless an explicit exception is declared.
- Implemented explicit host-execution exception gate via `AIRLOCK_ALLOW_HOST_EXEC_EXCEPTION=1`.
- Added first-class `investigate` and `plan` entrypoints.
- `plan` now accepts either a repo path or structured JSON input carrying issue URL, failing command, and failure text.
- Unified attempt/autofix host-vs-VM routing through shared preflight policy instead of ad hoc local checks.
- Added shared compiled-contract execution helper so campaign execution and direct contract execution use the same backend/prereq path.
- Tightened contract validation: `security.exportPaths` is now required, and Firecracker mode must be `local` or `ssh`.
- Firecracker backend now fails honestly for contracts that require guest binary injection (`/tmp/airlock`, `/tmp/airlock-researchguest`) instead of implying parity that does not yet exist.
- Expanded typed mutation support with `create_file` and `apply_patch`.
- Expanded semantic mutation support with `ensure_line`, `nil_guard`, and `error_return`.
- Added lessons-aware mutation-family ranking in planning, with optional lesson corpus input via `AIRLOCK_LESSONS_ROOT`.
- Planning now uses failure-text-derived fingerprint hints plus prior lessons to rank mutation families more intelligently.
- Expanded classification with `bootstrap_needed_vm_preferred`, `partial_runnable_scope`, and `env_config_blocked`.
- Added warning-level taxonomy for `service_dependent`, `integration_blocked`, and `flaky_candidate`.
- Fixed a real probe issue: repo root detection now prefers the git root over the nearest nested manifest so subdir/package scope is preserved honestly.
- Added host-toolchain-blocked-but-VM-runnable classification.
- Added VM auto-routing for `attempt-run` and `autofix-run`.
- Validated real VM-routed fixes on `litefunctions/portal`.
- Confirmed one important gap pattern for nested-module repos: safety allowlists must be interpreted relative to git root in diff accounting.
- Confirmed that route quality is now improving, but repair planning quality remains the larger strategic gap.
- Probed new OSS targets:
  - `langchain-ai/langchain` root exposed a real monorepo detection gap: Airlock reported `repoType: unknown` at repo root because manifests live in nested package dirs.
  - `langchain-ai/langchain/libs/core` showed that subdir probing works fine once pointed at a concrete package.
  - `cli/cli` and `charmbracelet/gum` both validated the usefulness of the `host_toolchain_blocked_vm_runnable` route.
- New nuance discovered: host toolchain classification is policy-sensitive. Raw host `go test ./...` can succeed for some repos via Go's auto-toolchain download, while Airlock intentionally treats host execution as blocked under local-toolchain-only policy. This likely needs clearer policy documentation or an explicit configurable stance.
- Ran full VM-backed research successfully on `langchain-ai/langchain` (`libs/core`) for issue `#36297`.
  - Real Airlock gap fixed: monorepo roots now stop with `monorepo_target_required` and enumerate concrete package targets.
  - Real Airlock gap fixed: `research-run` now rebases safety allowlists when `repo.subdir` is set.
  - Real workflow lesson: Python repos need venv-first bootstrap; system `pip install` inside Ubuntu guests is not sufficient because of PEP 668.
  - Real workflow lesson: setup steps that only mutate local environment should not require a checkpoint commit.
- Ran full VM-backed research successfully on `charmbracelet/gum` for issue `#1025`-style file height accounting.
  - Confirmed a new real repo success outside Beats and LangChain.
  - Learned that repo refs in contracts must be verified; `gum` required `main`, not `master`.
  - Confirmed command-first, test-added-in-setup remediation works cleanly for smaller Go OSS repos.
- Ran full VM-backed research successfully on `cli/cli` for issue `#12927`-style worktree corruption in `gh repo sync`.
  - Confirmed another large Go CLI repo can be handled cleanly through the same reproduce → patch → validate loop.
  - Repo lesson: `git update-ref` on a branch checked out in another worktree is a real corruption hazard and should be blocked unless a worktree-aware path is implemented.
  - Airlock workflow gap: authoring complex setup mutations as inline command strings is too brittle; patch-based or first-class file-mutation contract helpers would reduce contract failure modes.
