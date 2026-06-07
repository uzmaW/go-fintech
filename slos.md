# Service Level Objectives (SLOs)

## System Overview
The Go-Fintech transaction processing system targets 1,000,000 transactions per minute (≈16,700 TPS).

## SLIs (Service Level Indicators)

### Availability
| Service | SLI | Target | Measurement |
|---------|-----|--------|-------------|
| api-gateway | Successful HTTP responses (non-5xx) | 99.95% | Prometheus `fintech_http_requests_total` |
| fraud-service | Messages processed without error | 99.99% | Kafka consumer commit rate |
| authorization-service | Authorization decisions per request | 99.99% | Redis operation success rate |
| ledger-processor | Ledger entries committed successfully | 99.999% | PostgreSQL transaction success |
| notification-service | Notifications delivered to Kafka | 99.95% | Kafka producer acknowledge rate |

### Latency
| Operation | Target (P50) | Target (P95) | Target (P99) |
|-----------|--------------|--------------|--------------|
| Transaction ingestion (api-gateway) | < 50ms | < 200ms | < 500ms |
| Fraud scoring (fraud-service) | < 100ms | < 300ms | < 1000ms |
| Authorization hold (authorization-service) | < 30ms | < 100ms | < 200ms |
| Ledger write (ledger-processor) | < 50ms | < 150ms | < 300ms |
| End-to-end (ingestion → settlement) | < 2s | < 5s | < 10s |

### Throughput
| Metric | Target | Minimum |
|--------|--------|---------|
| Transaction ingestion rate | 16,700 TPS | 10,000 TPS |
| Kafka consumer throughput | 20,000 msg/s | 12,000 msg/s |
| Ledger entries per second | 33,400 entries/s | 20,000 entries/s |

### Error Budget
| Period | Budget | Burn Rate |
|--------|--------|-----------|
| Rolling 30 days | 21.6 minutes (99.95%) | 1x = 14.4 min/day |
| Monthly | 21.6 minutes | 2x = 7.2 min/day (fast burn alert) |

## SLOs Definitions

### SLO-1: Transaction Availability
- **SLI**: Percentage of API requests returning non-5xx responses
- **Target**: 99.95% over 30-day rolling window
- **Error Budget**: 21.6 minutes of downtime per month
- **Alert**: Fast burn (14.4x hourly burn rate) triggers page

### SLO-2: Transaction Latency
- **SLI**: Percentage of transactions processed within latency targets
- **Target**: 95% of transactions < 200ms end-to-end
- **Measurement**: P95 latency from HTTP request to Kafka acknowledgment
- **Alert**: P95 > 500ms for 5 minutes

### SLO-3: Data Durability
- **SLI**: Percentage of settled transactions with correct ledger entries
- **Target**: 99.999% (5 nines)
- **Measurement**: Ledger entry count vs. settled transaction count
- **Alert**: Discrepancy > 0.001% detected

### SLO-4: Fraud Detection Coverage
- **SLI**: Percentage of transactions evaluated by fraud scoring
- **Target**: 100% (every transaction scored before settlement)
- **Measurement**: Kafka messages on `transactions.authorized` vs. `transactions.pending`
- **Alert**: Coverage drops below 99.9%

### SLO-5: Notification Delivery
- **SLI**: Percentage of notifications successfully delivered
- **Target**: 99.95%
- **Measurement**: Messages on `notifications.sent` vs. `notifications.outbox`
- **Alert**: Delivery rate < 99% for 5 minutes

## Consequences of SLO Violations

### Availability Breach
- **< 99.95%**: Engineering review required within 24 hours
- **< 99.9%**: Incident post-mortem mandatory
- **< 99.5%**: Customer credit issuance evaluation

### Latency Breach
- **P95 > 500ms**: Auto-scaling triggered, capacity review
- **P95 > 1s**: Traffic shedding for non-critical endpoints
- **P95 > 5s**: Service degradation notice to customers

### Data Durability Breach
- Any discrepancy triggers immediate investigation
- Reconciliation job runs every 15 minutes
- Financial audit trail maintained for 7 years

## Monitoring Stack
- **Prometheus**: Metrics collection (15s scrape interval)
- **Grafana**: Dashboards and visualization
- **AlertManager**: Alert routing and escalation
- **OpenTelemetry**: Distributed tracing
- **PagerDuty**: On-call rotation and escalation
