# Day 7 — Load testing, polish, release

## Goal

Someone can clone the repo, run **one command**, open Grafana, fire a load burst, and see the distributed system react live.

## One-command stack

```bash
make demo          # infra + apps
make load          # 100 orders / 10 parallel
make down          # tear down
```

| Script | Role |
|--------|------|
| [`scripts/start-stack.sh`](../scripts/start-stack.sh) | `docker compose up` + background Go/Bun processes |
| [`scripts/stop-apps.sh`](../scripts/stop-apps.sh) | Stop app PIDs / free ports 8080/8081/8090 |
| [`scripts/load-orders.sh`](../scripts/load-orders.sh) | Curl burst for dashboard demos |
| [`Makefile`](../Makefile) | `demo`, `load`, `load-rate`, `smoke`, `status`, `down` |

## Load test

```bash
./scripts/load-orders.sh 100 10
# or
make load
```

Unique `X-Client-Id` per request by default (avoids the 5/min rate limit). Shared client for limiter demos:

```bash
make load-rate
# SHARED_CLIENT=1 ./scripts/load-orders.sh 20
```

While it runs, watch http://localhost:3000/d/ecommerce-observability — throughput, lag, webhook rates, and API latency should move within a scrape interval (~15s).

## README polish

Root README now includes:

- Architecture diagram (Mermaid)
- “What this demonstrates” (interview framing)
- One-command setup
- Load / stop instructions
- Links to Day 1–7 docs

## Release

Tag `v1.0.0` after Day 7 lands on `main` — portfolio baseline of the full 7-day system.
