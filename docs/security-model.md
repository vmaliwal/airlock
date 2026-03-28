# Security model

## Defaults

- execution backend required
- disposable VM required
- network denied by default
- temp HOME required
- temp XDG dirs required
- host secret scrubbing required
- no SSH agent forwarding
- no writable host mounts
- artifact export must be explicit

## Threats addressed

- lifecycle scripts editing host dotfiles
- secret exfiltration via inherited env
- arbitrary host filesystem writes
- opportunistic network egress
- persistence outside run lifetime

## Threats reduced but not eliminated

- malicious code inside the guest
- guest-to-network abuse if network allowlists are granted
- supply-chain compromise inside guest package registries
- guest escape bugs in virtualization stack

## Host requirements

### macOS / Lima

- Lima installed and available as `limactl`
- Apple Virtualization.framework-capable environment
- no reliance on Lima default home mounts; Airlock provides explicit config with mounts disabled

### Linux / Firecracker

- Firecracker installed on the runner host
- a prepared rootfs and kernel image
- privileged orchestration account on the Linux runner host

## Artifact rules

Only these may return to the host:
- stdout/stderr logs
- structured run summary JSON
- declared output files
- git diff / patch if requested

No full guest home export by default.
