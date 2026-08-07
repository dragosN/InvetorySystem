#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="${ROOT_DIR}/scripts/stress-orders.k6.js"

if ! curl -sf http://localhost:8080/healthz >/dev/null; then
  echo "ERROR: orders-service not up. Run: make demo" >&2
  exit 1
fi

echo "Watch Grafana: http://localhost:3000/d/ecommerce-observability"
echo

K6_ARGS=(run)
[[ -n "${VUS:-}" ]] && K6_ARGS+=(-e "VUS=${VUS}")
[[ -n "${DURATION:-}" ]] && K6_ARGS+=(-e "DURATION=${DURATION}")
[[ -n "${SHARED_CLIENT:-}" ]] && K6_ARGS+=(-e "SHARED_CLIENT=${SHARED_CLIENT}")

if command -v k6 >/dev/null 2>&1; then
  echo "Using local k6"
  BASE_URL="${BASE_URL:-http://localhost:8080}" \
    k6 "${K6_ARGS[@]}" -e "BASE_URL=${BASE_URL:-http://localhost:8080}" "${SCRIPT}"
  exit 0
fi

echo "Local k6 not found — running via Docker image grafana/k6"
# host.docker.internal reaches services on the Mac/Colima host
docker run --rm -i \
  --add-host=host.docker.internal:host-gateway \
  -v "${ROOT_DIR}/scripts:/scripts:ro" \
  -e BASE_URL=http://host.docker.internal:8080 \
  -e VUS="${VUS:-25}" \
  -e DURATION="${DURATION:-45s}" \
  -e SHARED_CLIENT="${SHARED_CLIENT:-0}" \
  grafana/k6 \
  run /scripts/stress-orders.k6.js
