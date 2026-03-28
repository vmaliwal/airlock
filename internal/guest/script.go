package guest

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vmaliwal/airlock/internal/contract"
)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func BuildScript(c contract.Contract, sandboxName string, allowedEnv map[string]string) string {
	keys := make([]string, 0, len(allowedEnv))
	for k := range allowedEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var envExports []string
	for _, k := range keys {
		envExports = append(envExports, fmt.Sprintf("export %s=%s", k, shellQuote(allowedEnv[k])))
	}

	stepsJSON := stepsJSON(c.Steps)
	ref := shellQuote(c.Repo.Ref)
	subdir := shellQuote(c.Repo.Subdir)
	cloneURL := shellQuote(c.Repo.CloneURL)
	backend := shellQuote(string(c.Backend.Kind))
	networkMode := shellQuote(string(c.Security.Network))
	allowHosts := make([]string, 0, len(c.Security.AllowHosts))
	for _, host := range c.Security.AllowHosts {
		allowHosts = append(allowHosts, shellQuote(host))
	}
	bootstrapHosts := make([]string, 0, len(c.Security.BootstrapAllowHosts))
	for _, host := range c.Security.BootstrapAllowHosts {
		bootstrapHosts = append(bootstrapHosts, shellQuote(host))
	}
	bootstrapNetworkMode := shellQuote(string(c.Security.BootstrapNetwork))
	bootstrapPackages := make([]string, 0, len(c.Security.BootstrapAptPackages))
	for _, pkg := range c.Security.BootstrapAptPackages {
		bootstrapPackages = append(bootstrapPackages, shellQuote(pkg))
	}
	includePatch := "0"
	if c.Security.IncludePatch {
		includePatch = "1"
	}

	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

export AIRLOCK_ROOT=/airlock
export AIRLOCK_WORK=/airlock/work
export AIRLOCK_HOME=/airlock/home
export AIRLOCK_XDG_CONFIG_HOME=/airlock/xdg/config
export AIRLOCK_XDG_CACHE_HOME=/airlock/xdg/cache
export AIRLOCK_XDG_DATA_HOME=/airlock/xdg/data
export AIRLOCK_ARTIFACTS=/airlock/artifacts
export HOME="$AIRLOCK_HOME"
export XDG_CONFIG_HOME="$AIRLOCK_XDG_CONFIG_HOME"
export XDG_CACHE_HOME="$AIRLOCK_XDG_CACHE_HOME"
export XDG_DATA_HOME="$AIRLOCK_XDG_DATA_HOME"
export TMPDIR=/airlock/tmp
export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
export LANG=C.UTF-8
export LC_ALL=C.UTF-8
%s
export AIRLOCK_INCLUDE_PATCH=%s
export AIRLOCK_BACKEND=%s
export AIRLOCK_SANDBOX_NAME=%s
export AIRLOCK_REPO_CLONE_URL=%s
export AIRLOCK_REPO_REF=%s
export AIRLOCK_REPO_SUBDIR=%s

mkdir -p "$AIRLOCK_WORK" "$AIRLOCK_HOME" "$AIRLOCK_XDG_CONFIG_HOME" "$AIRLOCK_XDG_CACHE_HOME" "$AIRLOCK_XDG_DATA_HOME" "$AIRLOCK_ARTIFACTS" "$TMPDIR"
cd "$AIRLOCK_WORK"

reset_iptables() {
  if command -v sudo >/dev/null 2>&1 && command -v iptables >/dev/null 2>&1; then
    sudo iptables -F OUTPUT || true
    sudo iptables -P OUTPUT ACCEPT || true
  fi
}

apply_network_policy() {
  local mode="$1"
  shift
  local hosts=("$@")
  reset_iptables
  if [[ "$mode" == "deny" ]]; then
    if command -v sudo >/dev/null 2>&1 && command -v iptables >/dev/null 2>&1; then
      sudo iptables -P OUTPUT DROP || true
      sudo iptables -A OUTPUT -o lo -j ACCEPT || true
    fi
  elif [[ "$mode" == "allowlist" ]]; then
    if command -v getent >/dev/null 2>&1 && command -v sudo >/dev/null 2>&1 && command -v iptables >/dev/null 2>&1; then
      sudo iptables -P OUTPUT DROP || true
      sudo iptables -A OUTPUT -o lo -j ACCEPT || true
      for host in "${hosts[@]}"; do
        while read -r ip _; do
          [[ -n "$ip" ]] && sudo iptables -A OUTPUT -d "$ip" -j ACCEPT || true
        done < <(getent ahosts "$host" | awk '{print $1}' | sort -u)
      done
    fi
  fi
}

BOOTSTRAP_NETWORK_MODE=%s
BOOTSTRAP_ALLOW_HOSTS=(%s)
BOOTSTRAP_APT_PACKAGES=(%s)
NETWORK_MODE=%s
ALLOW_HOSTS=(%s)

if [[ ${#BOOTSTRAP_APT_PACKAGES[@]} -gt 0 ]]; then
  apply_network_policy "$BOOTSTRAP_NETWORK_MODE" "${BOOTSTRAP_ALLOW_HOSTS[@]}"
  if command -v sudo >/dev/null 2>&1 && command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo DEBIAN_FRONTEND=noninteractive apt-get install -y "${BOOTSTRAP_APT_PACKAGES[@]}"
  fi
fi

apply_network_policy "$NETWORK_MODE" "${ALLOW_HOSTS[@]}"

clone_repo() {
  local attempt=1
  while [[ $attempt -le 3 ]]; do
    rm -rf repo
    if git clone --depth 1 --filter=blob:none %s repo; then
      return 0
    fi
    attempt=$((attempt + 1))
    sleep 3
  done
  return 1
}

clone_repo
cd repo
if [[ -n %s && %s != '' ]]; then
  git checkout %s
fi
if [[ -n %s && %s != '' ]]; then
  cd %s
fi

cat > "$AIRLOCK_ARTIFACTS/steps.json" <<'JSON'
%s
JSON

python3 - <<'PY'
import json, os, subprocess, time, pathlib
artifacts = pathlib.Path(os.environ['AIRLOCK_ARTIFACTS'])
steps = json.loads((artifacts / 'steps.json').read_text())
results = []
success = True
for step in steps:
    name = step['name']
    stdout_path = artifacts / f"{name}.stdout.log"
    stderr_path = artifacts / f"{name}.stderr.log"
    started = time.time()
    with open(stdout_path, 'w') as out, open(stderr_path, 'w') as err:
        proc = subprocess.run(step['run'], shell=True, stdout=out, stderr=err, timeout=step.get('timeoutSeconds', 600), executable='/bin/bash')
    finished = time.time()
    result = {
        'name': name,
        'command': step['run'],
        'exitCode': proc.returncode,
        'stdoutPath': str(stdout_path),
        'stderrPath': str(stderr_path),
        'startedAt': time.strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', time.gmtime(started)),
        'finishedAt': time.strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', time.gmtime(finished)),
        'durationMs': int((finished - started) * 1000),
        'allowedFailure': bool(step.get('allowFailure', False)),
    }
    results.append(result)
    if proc.returncode != 0 and not step.get('allowFailure', False):
        success = False
        break
if os.environ.get('AIRLOCK_INCLUDE_PATCH') == '1':
    with open(artifacts / 'repo.patch', 'w') as f:
        subprocess.run('git diff --binary HEAD', shell=True, stdout=f, stderr=subprocess.DEVNULL, executable='/bin/bash')
summary = {
    'backend': os.environ.get('AIRLOCK_BACKEND', 'unknown'),
    'sandboxName': os.environ.get('AIRLOCK_SANDBOX_NAME', 'unknown'),
    'repo': {
        'cloneUrl': os.environ.get('AIRLOCK_REPO_CLONE_URL', ''),
        'ref': os.environ.get('AIRLOCK_REPO_REF') or None,
        'subdir': os.environ.get('AIRLOCK_REPO_SUBDIR') or None,
    },
    'startedAt': results[0]['startedAt'] if results else time.strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', time.gmtime()),
    'finishedAt': results[-1]['finishedAt'] if results else time.strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', time.gmtime()),
    'success': success,
    'steps': results,
    'patchPath': str(artifacts / 'repo.patch') if (artifacts / 'repo.patch').exists() else None,
    'guestArtifactDir': str(artifacts),
}
(artifacts / 'summary.json').write_text(json.dumps(summary, indent=2))
PY
`, strings.Join(envExports, "\n"), includePatch, backend, shellQuote(sandboxName), cloneURL, ref, subdir, bootstrapNetworkMode, strings.Join(bootstrapHosts, " "), strings.Join(bootstrapPackages, " "), networkMode, strings.Join(allowHosts, " "), cloneURL, ref, ref, ref, subdir, subdir, subdir, stepsJSON)
}

func stepsJSON(steps []contract.Step) string {
	data, _ := json.Marshal(steps)
	return string(data)
}
