.PHONY: build test run clean tidy lint docker-build docker-push k8s-apply argocd-sync

BINARIES=api-gateway fraud-service authorization-service ledger-processor notification-service settlement-job
REGISTRY?=ghcr.io/company/go-fintech
TAG?=latest

build:
	@echo "Building all services..."
	@for svc in $(BINARIES); do \
		echo "Building cmd/$$svc..."; \
		go build -o bin/$$svc ./cmd/$$svc; \
	done

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

run-api:
	go run ./cmd/api-gateway

run-fraud:
	go run ./cmd/fraud-service

run-auth:
	go run ./cmd/authorization-service

run-ledger:
	go run ./cmd/ledger-processor

run-notification:
	go run ./cmd/notification-service

run-settlement:
	go run ./cmd/settlement-job

clean:
	rm -rf bin/
	go clean

tidy:
	go mod tidy

deps:
	go mod download

lint:
	golangci-lint run ./...

vet:
	go vet ./...

security:
	govulncheck ./...

docker-build:
	@for svc in $(BINARIES); do \
		echo "Building docker image for $$svc..."; \
		docker build -t $(REGISTRY)/$$svc:$(TAG) --build-arg SERVICE=$$svc -f Dockerfile .; \
	done

docker-push:
	@for svc in $(BINARIES); do \
		echo "Pushing docker image for $$svc..."; \
		docker push $(REGISTRY)/$$svc:$(TAG); \
	done

docker-build-single:
	@echo "Building docker image for $(SERVICE)..."
	docker build -t $(REGISTRY)/$(SERVICE):$(TAG) --build-arg SERVICE=$(SERVICE) -f Dockerfile .

# Kubernetes
k8s-apply:
	kubectl apply -f k8s/postgres.yaml
	kubectl apply -f k8s/redis.yaml
	kubectl apply -f k8s/kafka.yaml
	kubectl apply -f k8s/api-gateway.yaml
	kubectl apply -f k8s/fraud-service.yaml
	kubectl apply -f k8s/authorization-service.yaml
	kubectl apply -f k8s/ledger-processor.yaml
	kubectl apply -f k8s/notification-service.yaml
	kubectl apply -f k8s/settlement-job.yaml
	kubectl apply -f k8s/prometheus-rules.yaml
	kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
	kubectl create configmap grafana-fintech-dashboard \
		--from-file=fintech-dashboard.json=k8s/grafana-dashboard.json \
		-n monitoring --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -f k8s/network-policies.yaml

k8s-delete:
	kubectl delete -f k8s/ --ignore-not-found

k8s-status:
	kubectl get all -n fintech

k8s-logs:
	kubectl logs -f -l app=$(SERVICE) -n fintech --tail=100

# ArgoCD
argocd-install:
	kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

argocd-sync:
	kubectl apply -f k8s/argocd.yaml
	argocd app sync fintech-infrastructure
	argocd app sync fintech-services

argocd-status:
	argocd app list
	argocd app get fintech-services

# Monitoring
monitoring-install:
	kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
		-n monitoring \
		--set grafana.enabled=true \
		--set prometheus.prometheusSpec.retention=30d \
		--set alertmanager.enabled=true

# Database
migrate-up:
	psql -h localhost -U postgres -d fintech -f migrations/001_initial_schema.sql

migrate-down:
	psql -h localhost -U postgres -d fintech -c "DROP TABLE IF EXISTS ledger_entries, transactions, accounts, cards, holds, audit_log, reconciliation_summary CASCADE;"

# Kafka topics
kafka-create-topics:
	python3 python/admin-cli/admin_tool.py create-topics --bootstrap-servers localhost:9092

kafka-list-topics:
	python3 python/admin-cli/admin_tool.py list-topics --bootstrap-servers localhost:9092

# Python tools
python-admin:
	python3 -m python.admin-cli.admin_tool

python-reconcile:
	python3 -m python.reconciliation.reconciliation

python-analytics:
	python3 -m python.analytics.batch_analytics

help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Test:"
	@echo "  build              - Build all Go binaries"
	@echo "  test               - Run Go tests with race detection"
	@echo "  lint               - Run golangci-lint"
	@echo "  vet                - Run go vet"
	@echo "  security           - Run govulncheck"
	@echo ""
	@echo "Run Services:"
	@echo "  run-api            - Run API Gateway"
	@echo "  run-fraud          - Run Fraud Detection Service"
	@echo "  run-auth           - Run Authorization Service"
	@echo "  run-ledger         - Run Ledger Processor"
	@echo "  run-notification   - Run Notification Service"
	@echo "  run-settlement     - Run Settlement Job"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build       - Build all Docker images"
	@echo "  docker-push        - Push all Docker images"
	@echo "  docker-build-single SERVICE=xxx - Build single service"
	@echo ""
	@echo "Kubernetes:"
	@echo "  k8s-apply          - Apply all K8s manifests"
	@echo "  k8s-delete         - Delete all K8s resources"
	@echo "  k8s-status         - Show K8s resource status"
	@echo "  k8s-logs SERVICE=x - Tail logs for a service"
	@echo ""
	@echo "ArgoCD:"
	@echo "  argocd-install     - Install ArgoCD in cluster"
	@echo "  argocd-sync        - Sync ArgoCD applications"
	@echo "  argocd-status      - Show ArgoCD app status"
	@echo ""
	@echo "Monitoring:"
	@echo "  monitoring-install - Install Prometheus + Grafana"
	@echo ""
	@echo "Database & Kafka:"
	@echo "  migrate-up         - Run database migrations"
	@echo "  migrate-down       - Drop all tables"
	@echo "  kafka-create-topics - Create Kafka topics"
	@echo "  kafka-list-topics   - List Kafka topics"
	@echo ""
	@echo "Utility:"
	@echo "  clean              - Remove built binaries"
	@echo "  tidy               - Tidy Go modules"
	@echo "  deps               - Download Go dependencies"
