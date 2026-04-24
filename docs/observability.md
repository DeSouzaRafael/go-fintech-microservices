# Observability

## Stack

| Tool | Role | Port |
|------|------|------|
| OpenTelemetry SDK | Traces + metrics in every service | — |
| Jaeger (OTLP) | Distributed trace collection & UI | 16686 |
| Prometheus | Metrics scraping (15s interval) | 9090 |
| Grafana | Dashboards | 13000 |

All 7 services expose `/metrics` (OTel Prometheus exporter) and send OTLP traces to Jaeger on startup.

---

## Service Health — Prometheus Targets

All 7 microservices registered and scraped successfully.

![Prometheus targets — all UP](screenshots/prometheus-targets-all-up.png)

### Live Runtime Stats (collected 2026-04-24)

| Service | Goroutines | Heap | Open FDs |
|---------|-----------|------|----------|
| identity | 20 | 3.38 MB | 15 |
| wallet | 16 | 5.51 MB | 11 |
| transaction | 14 | 5.36 MB | 10 |
| fraud | 19 | 7.20 MB | 13 |
| notification | 14 | 6.22 MB | 10 |
| query | 14 | 5.51 MB | 10 |
| gateway | 35 | 3.98 MB | 12 |

Low heap footprint — smallest service (identity) runs at **3.4 MB** under load. Gateway has more goroutines due to HTTP connection handling.

---

## Distributed Tracing — Jaeger

Every gRPC call is instrumented via the `UnaryTracing` interceptor in `pkg/middleware`. Traces ship to Jaeger over OTLP gRPC on port 4317.

### Trace list — Identity Service (Login endpoint, 80 requests)

![Jaeger traces list](screenshots/jaeger-traces-list.png)

Observed p50 ≈ **49–51ms** for `/fintech.identity.v1.IdentityService/Login` — includes bcrypt verification (~48ms) + Postgres token write.

### Single trace detail

![Jaeger trace detail](screenshots/jaeger-trace-detail.png)

Span shows the full gRPC duration with OTel attributes. Service name, operation, and trace ID are propagated automatically.

---

## Metrics — Prometheus + Grafana

Grafana is provisioned automatically from `deploy/grafana/dashboards/`. Two dashboards in the **Fintech** folder:

- **Fintech Services — RED Metrics** — request rate, error rate, p50/p95/p99 latency per service
- **Fintech — Kafka & Outbox** — consumer lag, unpublished outbox depth, circuit breaker state

### Go runtime metrics across all services

![Prometheus — goroutines per service](screenshots/prometheus-all-services.png)

### Grafana Explore — real-time goroutine count per service

![Grafana Explore — goroutines](screenshots/grafana-explore-goroutines.png)

---

## Starting the Stack

```bash
# 1. Infrastructure
docker compose -f deploy/docker-compose.yml up -d

# 2. Migrations
make migrate-up

# 3. Services (example — identity)
PORT=50052 METRICS_PORT=9102 \
  DB_DSN="postgres://fintech:fintech@localhost:15432/identity?sslmode=disable" \
  JWT_SECRET="<secret>" \
  OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317" \
  ./services/identity/cmd/server/server

# 4. Open UIs
open http://localhost:16686   # Jaeger
open http://localhost:13000   # Grafana (admin/admin)
open http://localhost:9090    # Prometheus
```

## Adding Metrics to a Service

Every `cmd/server/main.go` calls `pkgmetrics.Setup()`:

```go
metricsSrv, err := pkgmetrics.Setup(pkgmetrics.Config{
    ServiceName: "identity",
    Port:        9102,          // matches prometheus.yml scrape config
})
defer func() { _ = metricsSrv.Shutdown(ctx) }()
```

Port assignment per `deploy/grafana/provisioning/prometheus.yml`:

| Service | Metrics port |
|---------|-------------|
| wallet | 9101 |
| identity | 9102 |
| transaction | 9103 |
| fraud | 9104 |
| notification | 9105 |
| query | 9106 |
| gateway | 9107 |
