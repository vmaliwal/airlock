# Next-Phase Gaps

Last updated: 2026-03-28
Baseline version: `v0.1.0` (`c93e806`)

This document is the durable gap tracker for Airlock.

Purpose:
- record the delta between the current implementation and the intended vision
- keep a running list of what is incomplete, risky, or underpowered
- append new learnings instead of re-deriving them from scratch
- anchor roadmap work in real evidence from runs against actual repositories

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
1. stronger repair planning
2. better self-improvement from lessons
3. first-class evals
4. broader issue taxonomy + mutation taxonomy
5. full backend parity

## Execution Plan

This section translates the 10 gaps into concrete workstreams.

### Workstream A — Planning + Strategy Synthesis
Addresses:
- Gap 1
- Gap 2
- part of Gap 4

Deliverables:
- strategy library keyed by fingerprint class
- attempt synthesis rules for common failure families
- ranking policy that uses lessons beyond exact attempt names
- first strategy reports showing which mutation families win for which fingerprints

Success signals:
- reduced attempts-to-success on repeated issue classes
- better first-attempt win rate on benchmark tasks

### Workstream B — Unify Runtime Modes
Addresses:
- Gap 3
- part of Gap 5
- part of Gap 9

Deliverables:
- common preflight/routing layer across attempt/autofix/research/campaign
- shared artifact naming and summary structures
- route explanations visible in all mode outputs

Success signals:
- less duplicated routing logic
- same repo yields coherent route decisions regardless of entry mode

### Workstream C — Mutation + Classification Expansion
Addresses:
- Gap 4
- Gap 5

Deliverables:
- additional mutation families:
  - nil guard
  - error return hardening
  - import/config/bootstrap helpers
  - file scaffold helpers
- richer probe/preflight statuses for integration/env/service/bootstrap cases

Success signals:
- more real repo issues become addressable without hand-crafted mutations
- fewer false starts on repos that should have been routed or stopped earlier

### Workstream D — Backend Parity + Safety Codification
Addresses:
- Gap 6
- Gap 7

Deliverables:
- Firecracker guest parity with Lima
- safety policy matrix documented and enforced
- clearer bootstrap-vs-validation network policy checks
- more explicit unsafe-to-continue conditions

Success signals:
- same contract classes work across Lima and Firecracker
- safety decisions become more inspectable and deterministic

### Workstream E — Evals + UX + Review Output
Addresses:
- Gap 8
- Gap 9
- Gap 10

Deliverables:
- `evals/` benchmark corpus
- machine-readable eval summaries
- higher-level task UX
- PR/reviewer-facing output packet generation

Success signals:
- capability gains become measurable over time
- outputs are useful to maintainers, not just to Airlock itself

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
- Added host-toolchain-blocked-but-VM-runnable classification.
- Added VM auto-routing for `attempt-run` and `autofix-run`.
- Validated real VM-routed fixes on `litefunctions/portal`.
- Confirmed one important gap pattern for nested-module repos: safety allowlists must be interpreted relative to git root in diff accounting.
- Confirmed that route quality is now improving, but repair planning quality remains the larger strategic gap.
