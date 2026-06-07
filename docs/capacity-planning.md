# Capacity Planning

## Current System Capacity

### Target Load
- **Transactions per minute**: 1,000,000
- **Transactions per second**: 16,700 TPS
- **Peak load (2x)**: 33,400 TPS
- **Burst capacity (3x)**: 50,100 TPS

### Resource Requirements

#### Compute (per service replica)
| Service | CPU Request | CPU Limit | Memory Request | Memory Limit |
|---------|-------------|-----------|----------------|--------------|
| api-gateway | 500m | 2000m | 512Mi | 1Gi |
| fraud-service | 1000m | 4000m | 1Gi | 2Gi |
| authorization-service | 500m | 1000m | 256Mi | 512Mi |
| ledger-processor | 1000m | 2000m | 1Gi | 2Gi |
| notification-service | 250m | 1000m | 256Mi | 512Mi |
| settlement-job | 500m | 1000m | 512Mi | 1Gi |

#### Database (PostgreSQL)
| Resource | Current | Recommended | Peak |
|----------|---------|-------------|------|
| CPU | 4 cores | 8 cores | 16 cores |
| Memory | 16GB | 32GB | 64GB |
| Storage | 100GB | 500GB | 1TB |
| Connections | 100 | 200 | 400 |
| IOPS | 10,000 | 20,000 | 40,000 |

#### Redis
| Resource | Current | Recommended | Peak |
|----------|---------|-------------|------|
| CPU | 2 cores | 4 cores | 8 cores |
| Memory | 4GB | 16GB | 32GB |
| Connections | 500 | 1000 | 2000 |
| Network | 1Gbps | 10Gbps | 25Gbps |

#### Kafka
| Resource | Current | Recommended | Peak |
|----------|---------|-------------|------|
| Brokers | 3 | 5 | 7 |
| Partitions per topic | 12 | 24 | 36 |
| Replication factor | 3 | 3 | 3 |
| Retention | 7 days | 14 days | 30 days |
| Disk per broker | 500GB | 1TB | 2TB |
| Network | 1Gbps | 10Gbps | 25Gbps |

## Scaling Strategies

### Horizontal Scaling (Recommended)
| Service | Min Replicas | Max Replicas | Target CPU | Scale-up Cooldown |
|---------|--------------|--------------|------------|-------------------|
| api-gateway | 3 | 20 | 60% | 60s |
| fraud-service | 3 | 15 | 70% | 120s |
| authorization-service | 3 | 10 | 60% | 60s |
| ledger-processor | 3 | 10 | 60% | 120s |
| notification-service | 2 | 10 | 70% | 60s |
| settlement-job | 1 | 1 | N/A | N/A |

### Vertical Scaling Triggers
- **Database**: Scale up when connection count > 80% of max, or IOPS > 70% of provisioned
- **Redis**: Scale up when memory > 75%, or connection count > 80%
- **Kafka**: Add brokers when disk usage > 70%, or request latency > 100ms

## Capacity Thresholds

### Warning Thresholds (80% capacity)
- CPU utilization > 80% sustained for 5 minutes
- Memory utilization > 80% sustained for 5 minutes
- Database connections > 80% of max
- Kafka consumer lag > 10,000 messages
- Redis memory > 80%

### Critical Thresholds (90% capacity)
- CPU utilization > 90% sustained for 2 minutes
- Memory utilization > 90% sustained for 2 minutes
- Database connections > 90% of max
- Kafka consumer lag > 50,000 messages
- Redis memory > 90%

## Load Testing Results

### Baseline Performance (3 replicas)
| Metric | Result |
|--------|--------|
| Max TPS achieved | 25,000 TPS |
| P50 latency | 45ms |
| P95 latency | 180ms |
| P99 latency | 420ms |
| Error rate | 0.01% |
| CPU utilization | 65% |
| Memory utilization | 55% |

### Stress Test (peak load)
| Metric | Result |
|--------|--------|
| Max TPS achieved | 50,000 TPS |
| P50 latency | 120ms |
| P95 latency | 450ms |
| P99 latency | 1200ms |
| Error rate | 0.1% |
| CPU utilization | 85% |
| Memory utilization | 75% |

## Cost Optimization

### Right-Sizing Recommendations
- **api-gateway**: Can reduce to 250m CPU / 256Mi memory during off-peak (nightly)
- **notification-service**: Scale to 1 replica during off-peak hours
- **settlement-job**: Use spot instances for batch processing

### Reserved Capacity
- **Database**: Reserved instances for 1-year term (30% savings)
- **Redis**: Reserved nodes for 1-year term (25% savings)
- **Kafka**: Reserved instances for 1-year term (20% savings)

## Growth Projections

### 6-Month Projection
- **Target**: 2M TPS (2x current)
- **Required resources**: 2x current compute, 1.5x storage
- **Estimated cost increase**: 40% (due to reserved capacity discounts)

### 12-Month Projection
- **Target**: 5M TPS (5x current)
- **Required resources**: 4x current compute, 3x storage
- **Architecture changes**: Consider sharding, multi-region deployment

## Monitoring Dashboard
- Grafana: `Fintech Capacity Overview`
- Key panels: CPU/Memory utilization, TPS, latency, consumer lag, connection pool usage
- Alert rules: PrometheusRule `fintech-capacity-alerts`

## Review Cadence
- **Weekly**: Review capacity metrics and scaling events
- **Monthly**: Cost optimization review
- **Quarterly**: Architecture review and growth planning
