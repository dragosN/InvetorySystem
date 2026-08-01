# Local infrastructure

Docker Compose stack for local development:

- **Redpanda** — Kafka-compatible broker (`localhost:19092`)
- **Redis** — caching / rate limiting (`localhost:6379`)
- **Prometheus** — metrics (`localhost:9090`)
- **Grafana** — dashboards (`localhost:3000`, admin/admin)

```bash
docker compose up -d
docker compose ps
docker compose down
```
