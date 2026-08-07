#!/usr/bin/env bash
set -euo pipefail

# Heavier burst against POST /orders to stress the pipeline + dashboards.
#
# Usage:
#   ./scripts/stress-orders.sh              # 500 req, 25 parallel
#   ./scripts/stress-orders.sh 1000 50      # custom count / concurrency
#   SHARED_CLIENT=1 ./scripts/stress-orders.sh 100   # also stress rate limiter

COUNT="${1:-500}"
PARALLEL="${2:-25}"
BASE_URL="${BASE_URL:-http://localhost:8080}"
RUN_ID="$(date +%s)"

if ! curl -sf "${BASE_URL}/healthz" >/dev/null; then
  echo "ERROR: orders-service not reachable at ${BASE_URL}" >&2
  echo "Run: make demo" >&2
  exit 1
fi

echo "STRESS  POST ${BASE_URL}/orders"
echo "        count=${COUNT}  parallel=${PARALLEL}  shared_client=${SHARED_CLIENT:-0}"
echo "        Grafana: http://localhost:3000/d/ecommerce-observability"
echo

ok=0
limited=0
other=0
start_ns="$(date +%s)"

post_one() {
  local i="$1"
  local client
  if [[ "${SHARED_CLIENT:-}" == "1" ]]; then
    client="stress-shared-${RUN_ID}"
  else
    client="stress-${RUN_ID}-${i}"
  fi

  local code
  code="$(curl -sS -o /dev/null -w "%{http_code}" --max-time 10 -X POST "${BASE_URL}/orders" \
    -H "Content-Type: application/json" \
    -H "X-Client-Id: ${client}" \
    -d "{\"items\":[{\"sku\":\"STRESS-${i}\",\"quantity\":1,\"unit_price\":100}]}" 2>/dev/null || echo "000")"

  case "${code}" in
    201) echo OK ;;
    429) echo LIMITED ;;
    *) echo "OTHER:${code}" ;;
  esac
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

for i in $(seq 1 "${COUNT}"); do
  while [[ "$(jobs -pr | wc -l | tr -d ' ')" -ge "${PARALLEL}" ]]; do
    sleep 0.02
  done
  (
    post_one "${i}" >"${tmp}/${i}.out"
  ) &
  if (( i % 50 == 0 )); then
    echo "  queued ${i}/${COUNT}..."
  fi
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

end_ns="$(date +%s)"
elapsed=$((end_ns - start_ns))
[[ "${elapsed}" -lt 1 ]] && elapsed=1
rps=$((COUNT / elapsed))

echo
echo "done in ${elapsed}s (~${rps} req/s attempted)"
echo "  201=${ok}  429=${limited}  other=${other}"
echo
echo "Watch for ~30s while Prometheus scrapes:"
echo "  http://localhost:3000/d/ecommerce-observability"
echo
echo "Quick metrics:"
curl -sf http://localhost:8080/metrics 2>/dev/null | grep -E '^orders_created_total|^orders_rate_limit_rejected_total' || true
curl -sf http://localhost:8081/metrics 2>/dev/null | grep -E '^notifications_events_consumed_total|^notifications_webhook_success_total|^notifications_consumer_lag' || true
