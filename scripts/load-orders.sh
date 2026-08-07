#!/usr/bin/env bash
set -euo pipefail

# Burst of POST /orders for Grafana / Prometheus demos.
#
# Usage:
#   ./scripts/load-orders.sh                 # 50 sequential requests
#   ./scripts/load-orders.sh 100             # 100 requests
#   ./scripts/load-orders.sh 100 10          # 100 requests, 10 in parallel
#   SHARED_CLIENT=1 ./scripts/load-orders.sh 20   # same X-Client-Id → hit rate limit

COUNT="${1:-50}"
PARALLEL="${2:-1}"
BASE_URL="${BASE_URL:-http://localhost:8080}"
RUN_ID="$(date +%s)"

echo "POST ${BASE_URL}/orders  count=${COUNT}  parallel=${PARALLEL}"

ok=0
limited=0
other=0
active=0

post_one() {
  local i="$1"
  local client
  if [[ "${SHARED_CLIENT:-}" == "1" ]]; then
    client="load-shared-${RUN_ID}"
  else
    client="load-${RUN_ID}-${i}"
  fi

  local code
  code="$(curl -sS -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/orders" \
    -H "Content-Type: application/json" \
    -H "X-Client-Id: ${client}" \
    -d "{\"items\":[{\"sku\":\"LOAD-${i}\",\"quantity\":1,\"unit_price\":100}]}")"

  case "${code}" in
    201) echo OK ;;
    429) echo LIMITED ;;
    *) echo "OTHER:${code}" ;;
  esac
}

if [[ "${PARALLEL}" -le 1 ]]; then
  for i in $(seq 1 "${COUNT}"); do
    result="$(post_one "${i}")"
    case "${result}" in
      OK) ok=$((ok + 1)) ;;
      LIMITED) limited=$((limited + 1)) ;;
      *) other=$((other + 1)) ;;
    esac
  done
else
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT

  for i in $(seq 1 "${COUNT}"); do
    while [[ "$(jobs -pr | wc -l | tr -d ' ')" -ge "${PARALLEL}" ]]; do
      sleep 0.05
    done
    (
      post_one "${i}" >"${tmp}/${i}.out"
    ) &
  done
  wait

  for f in "${tmp}"/*.out; do
    [[ -f "$f" ]] || continue
    result="$(cat "$f")"
    case "${result}" in
      OK) ok=$((ok + 1)) ;;
      LIMITED) limited=$((limited + 1)) ;;
      *) other=$((other + 1)) ;;
    esac
  done
fi

echo "done: 201=${ok}  429=${limited}  other=${other}"
echo "Watch Grafana: http://localhost:3000/d/ecommerce-observability"
