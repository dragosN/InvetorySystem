.PHONY: help up down demo stop-apps load load-rate stress stress-rate smoke status

help:
	@echo "Targets:"
	@echo "  make demo        Start infra + apps (one command)"
	@echo "  make up          Start Docker infra only"
	@echo "  make stop-apps   Stop Go/Bun app processes"
	@echo "  make down        Stop apps + Docker infra"
	@echo "  make load        Light curl burst (100 / 10)"
	@echo "  make load-rate   Curl burst that hits rate limit"
	@echo "  make stress      k6 stress test (25 VUs, ~70s)"
	@echo "  make stress-rate k6 with shared client (rate limit)"
	@echo "  make smoke       Kafka produce/consume smoke test"
	@echo "  make status      Show health endpoints"

up:
	docker compose -f infra/docker-compose.yml up -d

demo:
	./scripts/start-stack.sh

stop-apps:
	./scripts/stop-apps.sh

down: stop-apps
	docker compose -f infra/docker-compose.yml down

load:
	./scripts/load-orders.sh 100 10

load-rate:
	SHARED_CLIENT=1 ./scripts/load-orders.sh 20

# Prefer local k6; otherwise run via Docker against host services.
stress:
	@./scripts/run-k6.sh

stress-rate:
	@SHARED_CLIENT=1 ./scripts/run-k6.sh

smoke:
	./scripts/smoke-kafka.sh

status:
	@curl -sf http://localhost:8080/healthz && echo " orders :8080" || echo " orders DOWN"
	@curl -sf http://localhost:8090/healthz && echo " webhook :8090" || echo " webhook DOWN"
	@curl -sf http://localhost:8081/healthz && echo " notif metrics :8081" || echo " notif metrics DOWN"
	@curl -sf http://localhost:9090/-/healthy >/dev/null && echo " prometheus :9090" || echo " prometheus DOWN"
	@curl -sf http://localhost:3000/api/health >/dev/null && echo " grafana :3000" || echo " grafana DOWN"
