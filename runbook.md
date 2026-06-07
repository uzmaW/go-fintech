# Runbook: Go-Fintech Transaction Processing System

## Overview
This runbook covers operational procedures for the fintech transaction processing system handling 1M transactions/minute.

## Service Architecture
| Service | Port | Health Port | Metrics Port | Purpose |
|---------|------|-------------|--------------|---------|
| api-gateway | 8080 | 8080 | 8080 | HTTP ingress, transaction ingestion |
| fraud-service | 8081 | 8081 | 8181 | Real-time fraud detection |
| authorization-service | 8083 | 8083 | 8183 | Balance holds, authorization |
| ledger-processor | 8084 | 8084 | 8184 | Double-entry ledger writes |
| notification-service | 8082 | 8082 | 8182 | Email/SMS/push notifications |
| settlement-job | 8085 | 8085 | 8185 | Nightly settlement CronJob |

## Alert Response Procedures

### HighErrorRate (Critical)
**Alert**: Error rate > 5% for 2 minutes
**Impact**: Transactions failing, revenue loss

1. Check service logs: `kubectl logs -l app=<service> --tail=200`
2. Verify Kafka connectivity: `kubectl exec -it kafka-0 -- kafka-topics.sh --bootstrap-server localhost:9092 --list`
3. Check Redis connectivity: `kubectl exec -it redis-0 -- redis-cli ping`
4. Check PostgreSQL: `kubectl exec -it postgres-0 -- pg_isready -U postgres`
5. If database issue, check connection pool: look for `MaxOpenConns` in logs
6. **Escalation**: If persists > 5 min, page on-call DBA

### HighLatency (Warning)
**Alert**: P95 latency > 500ms for 5 minutes

1. Check CPU/memory: `kubectl top pods -l app=<service>`
2. Check consumer lag: `kubectl exec -it kafka-0 -- kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group <service>`
3. If consumer lag > 10000, check for slow DB queries
4. Consider scaling HPA: `kubectl scale deployment/<service> --replicas=<N>`

### KafkaConsumerLag (Critical)
**Alert**: Consumer lag > 50000 for 5 minutes

1. Check if consumer pod is running: `kubectl get pods -l app=<service>`
2. Check consumer group status: `kafka-consumer-groups.sh --describe --group <service>`
3. If partition assignment is stuck, restart consumer: `kubectl rollout restart deployment/<service>`
4. Check for rebalancing storms in logs

### ServiceDown (Critical)
**Alert**: Service unhealthy for > 1 minute

1. Check pod status: `kubectl get pods -l app=<service>`
2. Check events: `kubectl describe pod <pod-name>`
3. Check OOMKilled: `kubectl get pod <pod-name> -o jsonpath='{.status.containerStatuses[0].lastState}'`
4. If OOM, increase memory limit in deployment
5. Check readiness probe: `kubectl exec -it <pod> -- wget -qO- http://localhost:<port>/readyz`

### DatabaseHighLatency (Warning)
**Alert**: DB query P95 > 100ms for 5 minutes

1. Check active connections: `SELECT count(*) FROM pg_stat_activity;`
2. Check slow queries: `SELECT * FROM pg_stat_activity WHERE state = 'active' AND query_start < now() - interval '10 seconds';`
3. Check for lock contention: `SELECT * FROM pg_locks WHERE NOT granted;`
4. Consider increasing `MaxOpenConns` if connection pool is saturated

## Manual Procedures

### Scaling a Service
```bash
kubectl scale deployment/<service> --replicas=<N>
```
Verify HPA is configured and不会 override manual scaling.

### Rolling Restart
```bash
kubectl rollout restart deployment/<service>
```
Monitor rollout status: `kubectl rollout status deployment/<service>`

### Force Settlement Job Run
```bash
kubectl create job --from=cronjob/settlement-job settlement-manual-$(date +%s) -n fintech
```

### Kafka Topic Operations
```bash
# List topics
kubectl exec -it kafka-0 -- kafka-topics.sh --bootstrap-server localhost:9092 --list

# Describe topic
kubectl exec -it kafka-0 -- kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic transactions.pending

# Check consumer group
kubectl exec -it kafka-0 -- kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group fraud-service
```

### Database Operations
```bash
# Connect to psql
kubectl exec -it postgres-0 -- psql -U postgres -d fintech

# Check partition sizes
SELECT schemaname, relname, pg_size_pretty(pg_total_relation_size(relid))
FROM pg_stat_user_tables ORDER BY pg_total_relation_size(relid) DESC;
```

## Disaster Recovery

### Database Backup
```bash
kubectl exec -it postgres-0 -- pg_dump -U postgres fintech > backup_$(date +%Y%m%d).sql
```

### Kafka Backup (Topic Data)
```bash
kubectl exec -it kafka-0 -- kafka-console-producer.sh \
  --bootstrap-server localhost:9092 \
  --topic transactions.pending \
  --property parse.key=true \
  --property key.separator=: < backup.txt
```

### Full Restore
1. Restore PostgreSQL: `kubectl exec -it postgres-0 -- psql -U postgres fintech < backup.sql`
2. Kafka topics are durable; restore from MirrorMaker if cross-DC backup exists

## Escalation Matrix
| Severity | Response Time | Contact |
|----------|---------------|---------|
| P0 (Service Down) | 5 min | On-call engineer + Engineering Lead |
| P1 (High Error Rate) | 15 min | On-call engineer |
| P2 (High Latency) | 30 min | Engineering team |
| P3 (Warning) | Next business day | Engineering team |
