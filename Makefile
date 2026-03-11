SHELL := /bin/bash

.PHONY: up down logs health test-go test-stage2 test-installer test-node-agent test-sdk-go \
	validate-contracts build-admin-web build-manager-desktop build-sdk-ts \
	dev-gateway dev-core dev-installer-help dev-admin-web dev-manager dev-mobile \
	deploy-install deploy-upgrade deploy-uninstall deploy-config

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f --tail=200

health:
	curl -fsS http://127.0.0.1:$${API_GATEWAY_PORT:-8080}/v1/health

test-go:
	GOCACHE=$$(pwd)/tmpcache/gobuild GOMODCACHE=$$(pwd)/tmpcache/gomod go -C services/core-platform test ./...
	GOCACHE=$$(pwd)/tmpcache/gobuild GOMODCACHE=$$(pwd)/tmpcache/gomod go -C apps/api-gateway test ./...

test-stage2:
	bash ops/scripts/stage2-integration.sh

test-installer:
	GOCACHE=$$(pwd)/tmpcache/gobuild GOMODCACHE=$$(pwd)/tmpcache/gomod go -C apps/installer-cli test ./...

test-node-agent:
	GOCACHE=$$(pwd)/tmpcache/gobuild GOMODCACHE=$$(pwd)/tmpcache/gomod go -C apps/node-agent test ./...

test-sdk-go:
	GOCACHE=$$(pwd)/tmpcache/gobuild GOMODCACHE=$$(pwd)/tmpcache/gomod go -C packages/sdk-go test ./...

validate-contracts:
	node packages/api-contracts/scripts/validate-openapi.mjs

build-admin-web:
	pnpm --dir apps/admin-web build

build-manager-desktop:
	pnpm --dir apps/manager-desktop build

build-sdk-ts:
	pnpm --dir packages/shared-types build
	pnpm --dir packages/sdk-ts build

deploy-install:
	bash deploy/install.sh

deploy-upgrade:
	bash deploy/upgrade.sh

deploy-uninstall:
	bash deploy/uninstall.sh

deploy-config:
	docker compose --env-file deploy/.env.example -f deploy/docker-compose.yml config

dev-gateway:
	cd apps/api-gateway && go run ./cmd/api-gateway

dev-core:
	cd services/core-platform && go run ./cmd/core-platform

dev-installer-help:
	go -C apps/installer-cli run ./cmd/installer -h

dev-admin-web:
	pnpm --dir apps/admin-web dev

dev-manager:
	pnpm --dir apps/manager-desktop dev

dev-mobile:
	cd apps/mobile-client && flutter run
