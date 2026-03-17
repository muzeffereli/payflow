
GO     := $(shell which go 2>/dev/null || echo "$(HOME)/sdk/go1.26.1/bin/go")
COMPOSE := docker compose -f infra/docker-compose.yml

.PHONY: help up down restart infra build test migrate clean logs

help:
	@echo ""
	@echo "  make up        Start all services (build + run)"
	@echo "  make infra     Start only infrastructure (postgres, redis, nats)"
	@echo "  make down      Stop everything"
	@echo "  make restart   Rebuild and restart all services"
	@echo "  make build     Compile all services locally (no Docker)"
	@echo "  make swag      Regenerate Swagger docs for all services"
	@echo "  make migrate   Run SQL migrations (requires psql)"
	@echo "  make test      Run all tests"
	@echo "  make logs      Tail logs from all services"
	@echo "  make clean     Remove all containers and volumes"
	@echo ""
	@echo "  Swagger UIs (after make up):"
	@echo "    API Gateway:      http://localhost:8080/swagger/index.html"
	@echo "    Order Service:    http://localhost:8081/swagger/index.html"
	@echo "    Payment Service:  http://localhost:8082/swagger/index.html"
	@echo "    Wallet Service:   http://localhost:8083/swagger/index.html"
	@echo ""


up:
	$(COMPOSE) up --build -d
	@echo ""
	@echo "  API Gateway:  http://localhost:8080"
	@echo "  Prometheus:   http://localhost:9090"
	@echo "  Grafana:      http://localhost:3000  (admin/admin)"
	@echo "  NATS Monitor: http://localhost:8222"
	@echo ""

infra:
	$(COMPOSE) up -d postgres redis nats
	@echo "Waiting for postgres to be healthy..."
	@$(COMPOSE) run --rm -e PGPASSWORD=postgres postgres \
		sh -c 'until pg_isready -h postgres -U postgres; do sleep 1; done'
	@echo "Infrastructure ready."

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) down
	$(COMPOSE) up --build -d

logs:
	$(COMPOSE) logs -f

logs-%:
	$(COMPOSE) logs -f $*

clean:
	$(COMPOSE) down -v --remove-orphans
	@echo "All containers and volumes removed."


build:
	$(GO) build ./...

test:
	$(GO) test -race -count=1 ./...

test-v:
	$(GO) test -race -v -count=1 ./...

test-unit:
	$(GO) test -race -short ./...

migrate:
	@bash scripts/migrate.sh

token:
	@curl -s -X POST http://localhost:8080/api/v1/tokens \
		-H "Content-Type: application/json" \
		-d '{"user_id":"user-test-123"}' | jq .

order:
	@curl -s -X POST http://localhost:8080/api/v1/orders \
		-H "Authorization: Bearer $(TOKEN)" \
		-H "Content-Type: application/json" \
		-H "Idempotency-Key: test-$(shell date +%s)" \
		-d '{"currency":"USD","items":[{"product_id":"prod-1","quantity":2,"price":2500}]}' \
		| jq .


swag:
	swag init -g cmd/api-gateway/main.go   -o cmd/api-gateway/docs   --parseInternal
	swag init -g cmd/order-service/main.go  -o cmd/order-service/docs  --parseInternal
	swag init -g cmd/wallet-service/main.go -o cmd/wallet-service/docs --parseInternal
	swag init -g cmd/payment-service/main.go -o cmd/payment-service/docs --parseInternal

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy
