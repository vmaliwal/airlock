# OSS Validation Queue

This file tracks active and candidate repo validations.

Policy:
- prefer real public OSS repos with clear bug reports or honest failure signals
- use repo validations to expose both:
  - repo bugs Airlock may fix
  - Airlock product issues Airlock must log
- when a run exposes an Airlock issue, record it in `docs/product-issues.md` before implementation

## Selection criteria

Pick repos that are:
- popular enough to matter
- active enough to have real issue flow
- reproducible enough to validate honestly
- diverse across language/runtime classes

## Current proven repos

- `elastic/beats`
- `langchain-ai/langchain`
- `cli/cli`
- `charmbracelet/gum`
- `ashupednekar/litefunctions/portal`

## Recommended next wave

### 1. `cli/cli`
Why:
- large, active Go CLI
- issue volume is high
- command-first fit is strong
- worktree / git / UX bugs are a good match for Airlock

Goals:
- find 1–2 more reproducible issues beyond `#12927`
- see whether current planning/mutation families are enough without handholding
- log any new Airlock planner/runtime gaps as product issues

### 2. `charmbracelet/gum`
Why:
- fast iteration
- relatively small Go surface area
- good for validating shorter repro → patch → validate loops

Goals:
- collect more than one success so Gum becomes a repeated-flywheel repo rather than a one-off win
- test whether smaller CLI repos expose planner quality gaps differently than large repos

### 3. `langchain-ai/langchain` (`libs/core` or another concrete subdir)
Why:
- exercises Python + monorepo + bootstrap complexity
- already exposed real Airlock lessons

Goals:
- validate whether Python defaults are still too implicit
- turn repeated Python pain into issue-driven product backlog, not ad hoc fixes

## Candidate new repos

### `astral-sh/uv`
Why:
- high-visibility Python tooling repo
- fast-moving
- likely to expose Python/bootstrap/CLI issue classes

### `moby/moby`
Why:
- large issue volume
- probably too heavy for early generic success, but useful as a classification stress test

### `hashicorp/terraform`
Why:
- large Go CLI with broad issue volume
- may expose command reproduction + acceptance/integration classification boundaries

### `jqlang/jq`
Why:
- compact CLI codebase
- simpler reproduction loops
- useful for smoke-testing smaller C/tooling repos if we want language expansion pressure

## Suggested immediate sequence

1. `cli/cli` — 1 new issue
2. `gum` — 1 new issue
3. `langchain` — 1 new issue or one honest blocked run

For each:
1. intake bug signal
2. run investigate/plan/preflight honestly
3. attempt reproduction/repair
4. if Airlock fails, log `AIR-*` issue first
5. only then decide whether to implement or defer
