#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/infra/docker-compose.yml"
COMPOSE=(docker compose -f "${COMPOSE_FILE}")
TOPIC="order.created"
PAYLOAD='{"event_id":"smoke-001","order_id":"ord-001","status":"created"}'

echo "==> Waiting for Redpanda to become healthy..."
for i in $(seq 1 30); do
  if "${COMPOSE[@]}" exec -T redpanda rpk cluster health 2>/dev/null | grep -Eq 'Healthy:.+true'; then
    echo "    Redpanda is healthy."
    break
  fi
  if [[ "$i" -eq 30 ]]; then
    echo "ERROR: Redpanda did not become healthy in time." >&2
    exit 1
  fi
  sleep 2
done

echo "==> Creating topic ${TOPIC} (idempotent)..."
"${COMPOSE[@]}" exec -T redpanda rpk topic create "${TOPIC}" \
  --partitions 1 \
  --replicas 1 \
  || true

echo "==> Producing test message..."
echo "${PAYLOAD}" | "${COMPOSE[@]}" exec -T redpanda rpk topic produce "${TOPIC}"

echo "==> Consuming from beginning (expect one matching message)..."
# Consume with a short offset window; pipe through grep to assert payload
CONSUMED="$("${COMPOSE[@]}" exec -T redpanda rpk topic consume "${TOPIC}" \
  --offset start \
  --num 1 \
  --format '%v\n' 2>/dev/null || true)"

if echo "${CONSUMED}" | grep -Fq 'smoke-001'; then
  echo "OK: produced and consumed test message on ${TOPIC}"
  echo "    ${CONSUMED}"
else
  echo "ERROR: did not find expected payload in consume output:" >&2
  echo "${CONSUMED}" >&2
  exit 1
fi
