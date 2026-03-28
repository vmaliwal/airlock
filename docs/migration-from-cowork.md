# Migration from cowork-supercharged

Airlock is now the source of truth for autoresearch.

## Moved concepts

From `cowork-supercharged/scripts/autoresearch/*`:
- repo probe
- result evaluation
- failure fingerprinting
- campaign mode / many-failure handling
- contracts and preflight logic
- safe execution policy

From `cowork-supercharged/specs/autoresearch/*`:
- protocol
- infra policy
- contract templates
- command-first architecture

## New ownership

- `~/repos/airlock` = implementation + docs + examples
- `cowork-supercharged` = consumer/integration + notes/artifacts only

## Cowork cleanup status

Completed:
- removed `scripts/autoresearch/*` implementation
- replaced `specs/autoresearch/*` with archive pointers
- removed Cowork autoresearch tests

Remaining direction:
- Cowork should only call Airlock or link to Airlock artifacts
- all protocol evolution should happen in Airlock
