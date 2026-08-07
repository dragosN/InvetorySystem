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

## Load / stress test

**Prefer k6** for real stress testing (VU ramp, p95 latency, fail thresholds). Curl scripts remain for quick bursts.

```bash
make stress                 # k6: ramp to 25 VUs for ~70s (Docker if k6 not installed)
make stress-rate            # shared X-Client-Id → exercise Redis rate limit
VUS=50 DURATION=2m make stress

# lighter curl alternative
make load                   # 100 orders / 10 parallel
make load-rate
```

| Script | Role |
|--------|------|
| [`scripts/stress-orders.k6.js`](../scripts/stress-orders.k6.js) | k6 scenario + thresholds |
| [`scripts/run-k6.sh`](../scripts/run-k6.sh) | Local k6 or `grafana/k6` Docker |
| [`scripts/load-orders.sh`](../scripts/load-orders.sh) | Simple curl burst |

After the transactional outbox change, the same `make stress` profile typically shows **p95 of a few ms** and much higher request counts. See [PERFORMANCE.md](./PERFORMANCE.md).
| [`scripts/stress-orders.sh`](../scripts/stress-orders.sh) | Heavier curl burst (optional) |

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
