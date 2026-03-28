# Repo Candidates

Last updated: 2026-03-28
Baseline version: `v0.1.0` (`c93e806`)

This file tracks candidate repositories and issues for Airlock validation.

Selection criteria:
- public OSS
- credible open issue or reproducible failure
- bounded validation command available
- preferably deterministic
- ideally covers a new gap area, not only the same failure family

## Current Working Set

### Proven useful
- `elastic/beats`
  - strong real bug corpus
  - deterministic code-level failures
  - good for command-first repair and campaign validation
- `ashupednekar/litefunctions`
  - useful for host-toolchain/VM-routing and test ergonomics
- `sindresorhus/execa`
  - useful for safe local worktree validation and protocol sanity
- `meetcli`
  - useful as a structural blocker honesty target

## Fresh Candidates to Evaluate

### `asyncapi/cli`
Potential issues:
- `#2027` — CLI hangs indefinitely when registry host is unreachable
- `#2026` — AsyncAPI document double-stringified in ZIP output
- `#2018` — watch mode crash

Why interesting:
- covers Node CLI behavior
- potential timeout / hang handling
- could exercise non-Go ecosystem strategies

Risks:
- some issues may be integration-heavy or environment-sensitive

### `langchain-ai/langchain`
Potential issues:
- `#36339` — unexpected keyword argument `x_title`
- `#36312` — missing attribute on model object
- `#36297` — field dropped during content conversion

Why interesting:
- Python ecosystem coverage
- API drift / adapter / structured-output bug classes

Risks:
- repo is large
- issue reproduction may require tighter package scoping

### `cli/cli`
Potential issues:
- `#12927` — worktree corruption in `gh repo sync`
- `#12895` — status logic around cancelled runs
- `#12812` — stale cached rate-limit header behavior

Why interesting:
- Go CLI
- git/worktree semantics are relevant to Airlock’s own domain

Risks:
- some issues may be hard to reproduce hermetically

### `charmbracelet/gum`
Potential issues:
- `#797` — pager last line render issue
- `#701` — fuzzy sort scoring bug
- `#681` — markdown table formatting breakage

Why interesting:
- smaller surface area
- potentially good for deterministic text-processing/rendering bugs

Risks:
- some issues may be UI/TTY sensitive

### `sindresorhus/execa`
Potential issues:
- `#1219` — using both `stdin: 'inherit'` and `input`
- `#1214` — duplex with fd3 writable stream constraint
- `#1194` — Windows-specific signal behavior

Why interesting:
- existing familiarity
- likely easier bounded Node test cases

Risks:
- avoid overfitting to one repo
- some issues platform-specific

## Gap Coverage Matrix

Use repo choices to intentionally cover missing capability areas.

- repair planning quality
  - `elastic/beats`
  - `langchain-ai/langchain`
- VM routing / host blocker handling
  - `ashupednekar/litefunctions`
- Node CLI / timeout / hanging behavior
  - `asyncapi/cli`
  - `sindresorhus/execa`
- text/rendering deterministic bugs
  - `charmbracelet/gum`
- structural blocker honesty
  - `meetcli`

## Next Repos to Try

Recommended next three:
1. `asyncapi/cli` — likely best new ecosystem + timeout/CLI behavior coverage
2. `langchain-ai/langchain` — Python/API-drift class coverage
3. `charmbracelet/gum` — smaller Go target for deterministic rendering/logic issues

## Append Log

### 2026-03-28
- Refreshed open issue candidates from GitHub search.
- `litefunctions/portal` graduated from candidate to validated VM-routing target.
- `asyncapi/cli` looks promising again, especially for timeout/hang handling rather than the previously dropped generic test-pass claim.
- Added popularity/issue-volume evidence for three strong candidates:
  - `langchain-ai/langchain` — ~131k stars, ~504 open issues
  - `cli/cli` — ~43k stars, ~949 open issues
  - `charmbracelet/gum` — ~23k stars, ~150 open issues
- First Airlock pass results:
  - `langchain-ai/langchain` root initially exposed a probe gap for monorepo root detection; package subdirs like `libs/core` probed correctly.
  - `cli/cli` is a strong VM-routed Go candidate.
  - `charmbracelet/gum` is another strong VM-routed Go candidate with a smaller/more tractable surface.
- `langchain-ai/langchain` (`libs/core`) is now a validated success target for issue `#36297`.
- `charmbracelet/gum` is now a validated success target for file height accounting in `file/file.go`.
