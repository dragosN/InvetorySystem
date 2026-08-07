.PHONY: help up down demo stop-apps load load-rate smoke status

help:
	@echo "Targets:"
	@echo "  make demo       Start infra + apps (one command)"
	@echo "  make up         Start Docker infra only"
	@echo "  make stop-apps  Stop Go/Bun app processes"
	@echo "  make down       Stop apps + Docker infra"
	@echo "  make load       Burst 100 orders (10 parallel)"
	@echo "  make load-rate  Burst with shared client (hit rate limit)"
	@echo "  make smoke      Kafka produce/consume smoke test"
	@echo "  make status     Show health endpoints"

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

smoke:
	./scripts/smoke-kafka.sh

status:
	@curl -sf http://localhost:8080/healthz && echo " orders :8080" || echo " orders DOWN"
	@curl -sf http://localhost:8090/healthz && echo " webhook :8090" || echo " webhook DOWN"
	@curl -sf http://localhost:8081/healthz && echo " notif metrics :8081" || echo " notif metrics DOWN"
	@curl -sf http://localhost:9090/-/healthy >/dev/null && echo " prometheus :9090" || echo " prometheus DOWN"
	@curl -sf http://localhost:3000/api/health >/dev/null && echo " grafana :3000" || echo " grafana DOWN"
