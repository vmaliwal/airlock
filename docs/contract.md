# Contract

```json
{
  "backend": { "kind": "lima" },
  "sandbox": {
    "namePrefix": "beats-kafka",
    "artifactsDir": "/absolute/host/path/artifacts",
    "cpu": 4,
    "memoryGiB": 8,
    "diskGiB": 20,
    "ttlMinutes": 60
  },
  "repo": {
    "cloneUrl": "https://github.com/elastic/beats.git",
    "ref": "main",
    "subdir": "libbeat/common/kafka"
  },
  "security": {
    "network": "deny",
    "allowHosts": [],
    "allowedEnv": [],
    "exportPaths": ["/airlock/artifacts"],
    "includePatch": true
  },
  "steps": [
    { "name": "repro", "run": "go test ./libbeat/common/kafka -run TestRepro_MajorVersionAliasUsesLatestMinor -count=1", "timeoutSeconds": 600 }
  ]
}
```

## Fields

### `backend`
- `kind`: `lima | firecracker`
- `firecrackerHost`: required for remote Firecracker orchestration when not local to a Linux host

### `sandbox`
- `namePrefix`: human-readable prefix
- `artifactsDir`: host path where artifacts are exported
- `cpu`, `memoryGiB`, `diskGiB`: guest sizing
- `ttlMinutes`: hard upper bound for the guest lifetime

### `repo`
- `cloneUrl`: git URL cloned inside the guest
- `ref`: branch / tag / sha
- `subdir`: optional working subdir within repo

### `security`
- `bootstrapNetwork`: `deny | allowlist` for guest bootstrap package installation
- `bootstrapAllowHosts`: allowlist for bootstrap phase if package installation is needed
- `bootstrapAptPackages`: minimal packages to install inside guest before repo execution
- `network`: `deny | allowlist` for repo execution phase
- `allowHosts`: explicit host allowlist for execution phase
- `allowedEnv`: host env vars permitted into guest
- `exportPaths`: guest paths allowed to be copied back
- `includePatch`: export `git diff` if repo is mutated inside guest

### `steps`
- `name`: label
- `run`: shell command executed inside guest repo workspace
- `timeoutSeconds`: optional per-step timeout
- `allowFailure`: optional flag when gathering inventory
