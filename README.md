# Go-Fintech Transaction Processing System

Production-grade, scalable fintech transaction processing platform targeting **1,000,000 transactions per minute** (~16,700 TPS).

## Architecture

```
                          ┌─────────────────────────────────────────────────┐
                          │                  CLIENTS                        │
                          │   Mobile App  │  Web App  │  Partner APIs       │
                          └──────┬────────┴─────┬─────┴──────┬──────────────┘
                                 │              │            │
                                 ▼              ▼            ▼
                      ┌──────────────────────────────────────────┐
                      │           API GATEWAY (Go/Gin)           │
                      │   POST /api/v1/transactions              │
                      │   POST /api/v1/transfers                 │
                      │   GET  /api/v1/transactions/:id          │
                      │   Prometheus metrics + OTel tracing      │
                      └──────────────────┬───────────────────────┘
                                         │
                                         ▼
                      ┌──────────────────────────────────────────┐
                      │              KAFKA CLUSTER                │
                      │                                          │
                      │  ┌─────────────────┐  ┌──────────────┐  │
                      │  │ transactions.    │  │ fraud.events │  │
                      │  │   pending        │  │              │  │
                      │  └────────┬────────┘  └──────┬───────┘  │
                      │           │                   │          │
                      │  ┌────────▼────────┐  ┌──────▼───────┐  │
                      │  │ transactions.    │  │ notifications│  │
                      │  │   authorized     │  │   .sent      │  │
                      │  └────────┬────────┘  │   .failed    │  │
                      │           │           └──────────────┘  │
                      │  ┌────────▼────────┐                    │
                      │  │ transactions.    │                    │
                      │  │   settled        │                    │
                      │  └─────────────────┘                    │
                      └──┬──────────┬──────────┬────────────────┘
                         │          │          │
            ┌────────────▼──┐  ┌────▼────┐  ┌──▼──────────────────┐
            │ FRAUD SERVICE │  │ LEDGER  │  │ AUTHORIZATION       │
            │ (Go)          │  │PROCESSOR│  │ SERVICE (Go)        │
            │               │  │ (Go)    │  │                     │
            │ • Score txns  │  │         │  │ • Redis Lua atomic  │
            │ • APPROVE/    │  │ • Double│  │   hold placement    │
            │   DECLINE/    │  │   entry │  │ • Balance check     │
            │   REVIEW      │  │ • Batch │  │ • Fail-closed       │
            │               │  │   write │  │   on Redis miss     │
            └───────┬───────┘  └────┬────┘  └─────────┬──────────┘
                    │               │                  │
                    │               ▼                  │
                    │     ┌──────────────────┐         │
                    │     │   POSTGRESQL     │◄────────┘
                    │     │                  │
                    │     │ • accounts (16)  │
                    │     │ • transactions   │
                    │     │ • ledger_entries │
                    │     │ • holds          │
                    │     │ • cards          │
                    │     │ • audit_log      │
                    │     └──────────────────┘
                    │
                    ▼
          ┌──────────────────┐        ┌──────────────────┐
          │   REDIS CLUSTER  │        │   NOTIFICATION   │
          │                  │        │   SERVICE (Go)   │
          │ • Balance holds  │        │                  │
          │ • Idempotency    │        │ • Email (SMTP)   │
          │ • Session cache  │        │ • SMS (Twilio)   │
          │ • Lua scripts    │        │ • Push (FCM)     │
          └──────────────────┘        └──────────────────┘
                                                │
                                  ┌─────────────▼─────────────┐
                                  │     SETTLEMENT JOB        │
                                  │     (Go CronJob)          │
                                  │                           │
                                  │ • Nightly batch (02:00)   │
                                  │ • AUTHORIZED → SETTLED    │
                                  │ • Reconciliation          │
                                  └───────────────────────────┘
```

## Services

| Service | Language | Port | Purpose |
|---------|----------|------|---------|
| `api-gateway` | Go/Gin | 8080 | HTTP ingress, transaction ingestion, Avro serialization |
| `fraud-service` | Go | 8081 | Real-time fraud scoring, APPROVE/DECLINE/REVIEW decisions |
| `authorization-service` | Go | 8083 | Redis atomic balance holds via Lua scripts, fail-closed |
| `ledger-processor` | Go | 8084 | Double-entry ledger writes with batch commit |
| `notification-service` | Go | 8082 | Email/SMS/push dispatch via Kafka outbox pattern |
| `settlement-job` | Go | 8085 | Nightly batch settlement (CronJob) |

## Tech Stack

| Component | Technology |
|-----------|-----------|
| **Services** | Go 1.23+ with Gin HTTP framework |
| **Streaming** | Apache Kafka (kafka-go) |
| **Database** | PostgreSQL 16 (pgx pool, hash partitioning) |
| **Cache** | Redis 7 (go-redis, Lua scripts) |
| **Observability** | Prometheus + Grafana, OpenTelemetry |
| **Messaging** | Avro schemas (goavro) |
| **CI/CD** | GitHub Actions → Docker → ArgoCD GitOps |
| **Orchestration** | Kubernetes (HPA, PDB, NetworkPolicy) |
| **Python** | Workers for analytics, reconciliation, admin CLI |

## Quick Start

### Prerequisites
- Go 1.23+
- Docker
- kubectl + Helm
- Kafka, PostgreSQL, Redis (or use Docker Compose)

### Build

```bash
make build          # Build all Go binaries
make docker-build   # Build all Docker images
make test           # Run all unit tests with race detection
```

### Run Locally

```bash
# Start infrastructure
docker-compose up -d kafka postgres redis

# Run services (each in separate terminal)
make run-api
make run-fraud
make run-auth
make run-ledger
make run-notification
```

### Deploy to Kubernetes

```bash
# Apply all manifests
make k8s-apply

# Sync ArgoCD
make argocd-sync

# Install monitoring stack
make monitoring-install
```

## Project Structure

```
├── cmd/                          # Service entrypoints
│   ├── api-gateway/
│   ├── authorization-service/
│   ├── fraud-service/
│   ├── ledger-processor/
│   ├── notification-service/
│   └── settlement-job/
├── internal/                     # Shared packages
│   ├── config/                   # YAML + env-var configuration
│   ├── db/                       # PostgreSQL connection pool
│   ├── health/                   # /healthz, /readyz, /livez
│   ├── kafka/                    # Producer/consumer wrappers + Avro
│   ├── metrics/                  # Prometheus metrics + Gin middleware
│   ├── models/                   # Domain types
│   ├── schema/                   # Avro schema registry
│   ├── tlsutil/                  # TLS/mTLS + NetworkPolicy generation
│   └── tracing/                  # OpenTelemetry OTLP init
├── k8s/                          # Kubernetes manifests
├── migrations/                   # SQL migrations
├── python/                       # Python workers & tools
├── mobile/                       # React Native mobile app
├── tests/                        # Integration & load tests
└── docs/                         # Runbook, SLOs, capacity planning
```

## Testing

```bash
make test                        # Unit tests + race detector
go test -tags integration ./...  # Integration tests (requires services)
k6 run tests/load/transaction_load.js  # Load test (requires running API)
```

### Load Test Scenarios

| Scenario | VUs | Duration | Target |
|----------|-----|----------|--------|
| Baseline | 100 | 2 min | Steady state |
| Spike | 100 → 5000 | 3 min | Burst traffic |
| Stress | 0 → 2000 | 8 min | Find breaking point |
| Soak | 200 | 10 min | Memory leaks |
| Throughput | — | 2 min | 16,700 TPS sustained |

## Configuration

All services use `internal/config` with YAML files and environment variable overrides:

```bash
# Environment variables
KAFKA_BROKERS=kafka-0:9092,kafka-1:9092
DATABASE_HOST=postgres.fintech.svc
REDIS_ADDR=redis.fintech.svc:6379
OTEL_ENABLED=true
OTEL_COLLECTOR_URL=otel-collector:4317
AVRO_SCHEMAS_DIR=./internal/schema/schemas
```

## API

### Create Transaction
```bash
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "idempotency_key": "unique-key-123",
    "transaction_type": "TRANSFER",
    "source_account_id": "acc-001",
    "target_account_id": "acc-002",
    "amount": 150.00,
    "currency": "USD"
  }'
# Response: 202 Accepted
# {"transaction_id": "uuid", "status": "PENDING"}
```

### Health Check
```bash
curl http://localhost:8080/api/v1/health
# {"status": "healthy", "service": "api-gateway", "env": "production"}
```

## SLOs

| SLI | Target | Error Budget |
|-----|--------|-------------|
| Availability | 99.95% | 21.6 min/month |
| P95 Latency | < 200ms | — |
| Data Durability | 99.999% | — |
| Fraud Coverage | 100% | — |

## Monitoring

- **Prometheus**: 15s scrape, 30-day retention
- **Grafana**: 10-panel dashboard (TPS, errors, latency, Kafka lag, DB/Redis metrics)
- **Alerts**: 10 rules (error rate, latency, consumer lag, service down, OOM, etc.)
- **Traces**: OpenTelemetry OTLP → Jaeger/Tempo

## Documentation

- [Runbook](docs/runbook.md) - Alert response, manual procedures, DR
- [SLOs](docs/slos.md) - Service level objectives and error budgets
- [Capacity Planning](docs/capacity-planning.md) - Resource sizing and scaling strategies

## License

Proprietary - Internal use only.
