SHELL := /bin/bash

.PHONY: up down logs health fmt lint test test-go test-stage2 migrate tree \
	dev-gateway dev-core dev-admin-web dev-manager dev-desktop-client dev-mobile

up:
	docker compose up -d --build

down:
	docker compose down -v

logs:
	docker compose logs -f --tail=200

health:
	curl -fsS http://127.0.0.1:$${API_GATEWAY_PORT:-8080}/v1/health

fmt:
	@echo "TODO: gofmt ./... && prettier --write . && dart format ."

lint:
	@echo "TODO: go vet ./... && eslint . && flutter analyze"

test:
	@echo "Use make test-go for Go unit tests, and make test-stage2 for stage2+stage3+stage4 integration tests"

test-go:
	GOCACHE=$$(pwd)/tmpcache/gobuild GOMODCACHE=$$(pwd)/tmpcache/gomod go -C apps/api-gateway test ./...
	GOCACHE=$$(pwd)/tmpcache/gobuild GOMODCACHE=$$(pwd)/tmpcache/gomod go -C services/core-platform test ./...

test-stage2:
	bash ops/scripts/stage2-integration.sh

migrate:
	@echo "Migrations are verified in stage2/stage3 integration tests via TEST_MIGRATIONS_DIR"

tree:
	find . -maxdepth 4 -type d | sort

dev-gateway:
	cd apps/api-gateway && go run ./cmd/api-gateway

dev-core:
	cd services/core-platform && go run ./cmd/core-platform

dev-admin-web:
	pnpm --dir apps/admin-web dev

dev-manager:
	pnpm --dir apps/manager-desktop dev

dev-desktop-client:
	pnpm --dir apps/desktop-client dev

dev-mobile:
	cd apps/mobile-client && flutter run
