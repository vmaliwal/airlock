#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
usage:
  airlock-firecracker-host.sh run \
    --name <sandbox-name> \
    --contract <host-path-to-guest-run.sh> \
    --artifacts <host-artifacts-dir> \
    [--copy-in <host-src>:<guest-dst>] ...

Environment:
  AIRLOCK_FIRECRACKER_STATE_DIR   Optional state root for staging workdirs.
                                  Default: /tmp/airlock-firecracker-host
  AIRLOCK_FIRECRACKER_DRIVER      Required for real execution. Executable called as:
                                  $AIRLOCK_FIRECRACKER_DRIVER run --manifest <manifest.json>
  AIRLOCK_FIRECRACKER_KEEP_WORKDIR=1
                                  Keep staged workdir after execution for debugging.
EOF
}

fail() {
  echo "error: $*" >&2
  exit 1
}

json_escape() {
  python3 - <<'PY' "$1"
import json, sys
print(json.dumps(sys.argv[1]))
PY
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 not found on PATH"
}

copy_file() {
  local src="$1"
  local dst="$2"
  mkdir -p "$(dirname "$dst")"
  cp "$src" "$dst"
  if [[ -x "$src" ]]; then
    chmod +x "$dst"
  fi
}

cmd="${1:-}"
if [[ -z "$cmd" ]]; then
  usage
  exit 1
fi
shift || true

[[ "$cmd" == "run" ]] || fail "unsupported command: $cmd"

name=""
contract=""
artifacts=""
declare -a copyins=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --name)
      [[ $# -ge 2 ]] || fail "--name requires a value"
      name="$2"
      shift 2
      ;;
    --contract)
      [[ $# -ge 2 ]] || fail "--contract requires a value"
      contract="$2"
      shift 2
      ;;
    --artifacts)
      [[ $# -ge 2 ]] || fail "--artifacts requires a value"
      artifacts="$2"
      shift 2
      ;;
    --copy-in)
      [[ $# -ge 2 ]] || fail "--copy-in requires a value"
      copyins+=("$2")
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

[[ -n "$name" ]] || fail "--name is required"
[[ -n "$contract" ]] || fail "--contract is required"
[[ -n "$artifacts" ]] || fail "--artifacts is required"
[[ -f "$contract" ]] || fail "contract file not found: $contract"

require_cmd python3
require_cmd cp
require_cmd mkdir

state_root="${AIRLOCK_FIRECRACKER_STATE_DIR:-/tmp/airlock-firecracker-host}"
run_id="${name}-$(date +%s)"
workdir="$state_root/$run_id"
staged_contract="$workdir/ingress/guest-run.sh"
staged_artifacts="$workdir/exports"
manifest="$workdir/manifest.json"

mkdir -p "$workdir/ingress/files" "$staged_artifacts" "$artifacts"
copy_file "$contract" "$staged_contract"
chmod +x "$staged_contract"

copyins_json="[]"
if [[ ${#copyins[@]} -gt 0 ]]; then
  entries=()
  idx=0
  for item in "${copyins[@]}"; do
    host_src="${item%%:*}"
    guest_dst="${item#*:}"
    [[ -n "$host_src" && -n "$guest_dst" && "$host_src" != "$guest_dst" ]] || fail "invalid --copy-in value: $item"
    [[ -f "$host_src" ]] || fail "copy-in source not found: $host_src"
    base="$(basename "$host_src")"
    staged_src="$workdir/ingress/files/${idx}-${base}"
    copy_file "$host_src" "$staged_src"
    entries+=("{\"hostSource\":$(json_escape "$host_src"),\"stagedSource\":$(json_escape "$staged_src"),\"guestDestination\":$(json_escape "$guest_dst")}")
    idx=$((idx + 1))
  done
  copyins_json="[$(IFS=,; echo "${entries[*]}")]"
fi

cat > "$manifest" <<EOF
{
  "name": $(json_escape "$name"),
  "workdir": $(json_escape "$workdir"),
  "contract": {
    "hostPath": $(json_escape "$contract"),
    "stagedPath": $(json_escape "$staged_contract")
  },
  "artifacts": {
    "requestedHostDir": $(json_escape "$artifacts"),
    "stagedHostDir": $(json_escape "$staged_artifacts")
  },
  "copyIn": $copyins_json
}
EOF

cleanup() {
  if [[ "${AIRLOCK_FIRECRACKER_KEEP_WORKDIR:-0}" != "1" ]]; then
    rm -rf "$workdir"
  fi
}
trap cleanup EXIT

if [[ -z "${AIRLOCK_FIRECRACKER_DRIVER:-}" ]]; then
  echo "manifest written to $manifest" >&2
  fail "AIRLOCK_FIRECRACKER_DRIVER is not set; cannot perform real Firecracker execution"
fi

if [[ ! -x "$AIRLOCK_FIRECRACKER_DRIVER" ]]; then
  fail "AIRLOCK_FIRECRACKER_DRIVER is not executable: $AIRLOCK_FIRECRACKER_DRIVER"
fi

"$AIRLOCK_FIRECRACKER_DRIVER" run --manifest "$manifest"

if [[ ! -f "$artifacts/summary.json" && -f "$staged_artifacts/summary.json" ]]; then
  cp "$staged_artifacts"/* "$artifacts"/ 2>/dev/null || true
fi

[[ -f "$artifacts/summary.json" ]] || fail "driver completed but summary.json was not exported to $artifacts"
