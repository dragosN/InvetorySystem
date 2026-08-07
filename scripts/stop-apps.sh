#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_DIR="${ROOT_DIR}/.pids"

if [[ ! -d "${PID_DIR}" ]]; then
  echo "No app pids found."
  exit 0
fi

for pidfile in "${PID_DIR}"/*.pid; do
  [[ -f "${pidfile}" ]] || continue
  name="$(basename "${pidfile}" .pid)"
  pid="$(cat "${pidfile}")"
  if kill -0 "${pid}" 2>/dev/null; then
    # Kill process group children if any (go run spawns a child)
    kill "${pid}" 2>/dev/null || true
    # Also try killing child go/bun processes by port/name lightly
    echo "stopped ${name} (pid ${pid})"
  else
    echo "${name} not running"
  fi
  rm -f "${pidfile}"
done

# Best-effort cleanup of go run / bun leftovers on known ports
for port in 8080 8081 8090; do
  if command -v lsof >/dev/null 2>&1; then
    pids="$(lsof -tiTCP:${port} -sTCP:LISTEN 2>/dev/null || true)"
    if [[ -n "${pids}" ]]; then
      # shellcheck disable=SC2086
      kill ${pids} 2>/dev/null || true
      echo "freed port ${port}"
    fi
  fi
done

echo "apps stopped"
