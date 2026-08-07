#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PID_DIR="${ROOT_DIR}/.pids"
LOG_DIR="${ROOT_DIR}/.logs"
mkdir -p "${PID_DIR}" "${LOG_DIR}"

export PATH="/opt/homebrew/bin:${PATH}"

echo "==> Starting infra (Redpanda, Redis, Prometheus, Grafana)..."
docker compose -f "${ROOT_DIR}/infra/docker-compose.yml" up -d

echo "==> Waiting for Redis..."
for i in $(seq 1 30); do
  if docker compose -f "${ROOT_DIR}/infra/docker-compose.yml" exec -T redis redis-cli ping 2>/dev/null | grep -q PONG; then
    break
  fi
  sleep 1
done

start_bg() {
  local name="$1"
  shift
  local pidfile="${PID_DIR}/${name}.pid"
  local logfile="${LOG_DIR}/${name}.log"

  if [[ -f "${pidfile}" ]] && kill -0 "$(cat "${pidfile}")" 2>/dev/null; then
    echo "    ${name} already running (pid $(cat "${pidfile}"))"
    return
  fi

  nohup "$@" >"${logfile}" 2>&1 &
  echo $! >"${pidfile}"
  echo "    started ${name} (pid $!, log ${logfile})"
}

echo "==> Starting app processes..."
start_bg orders \
  bash -c "cd '${ROOT_DIR}/orders-service' && exec go run ./cmd/server"

start_bg mock-webhook \
  bash -c "cd '${ROOT_DIR}/notifications-service' && exec bun run mock-webhook"

start_bg notifications \
  bash -c "cd '${ROOT_DIR}/notifications-service' && exec bun run start"

echo "==> Waiting for orders API..."
for i in $(seq 1 60); do
  if curl -sf http://localhost:8080/healthz >/dev/null 2>&1; then
    echo "    orders-service is up"
    break
  fi
  if [[ "$i" -eq 60 ]]; then
    echo "ERROR: orders-service did not become ready. Check .logs/orders.log" >&2
    exit 1
  fi
  sleep 1
done

cat <<EOF

Stack is up.

  Orders API:     http://localhost:8080
  Mock webhook:   http://localhost:8090/deliveries
  Notif metrics:  http://localhost:8081/metrics
  Prometheus:     http://localhost:9090
  Grafana:        http://localhost:3000/d/ecommerce-observability
                  (admin / admin)

Load test:  make load
            ./scripts/load-orders.sh 100 10
Stop apps:  make stop-apps
Stop all:   make down

EOF
