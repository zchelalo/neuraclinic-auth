ifneq ("$(wildcard .env)", "")
	include .env
	export $(shell sed 's/=.*//' .env)
endif

DOCKER_COMPOSE_FILE = ./.docker/compose.yml
DOCKER_NETWORK = neuraclinic-network
URI_DB = postgresql://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)
MIGRATE = docker run --rm -v $(shell pwd)/internal/db/migrations:/migrations --network $(DOCKER_NETWORK) migrate/migrate -path /migrations -database "$(URI_DB)" -verbose

setup:
	$(MAKE) create-envs
	$(MAKE) jwt-keys
	$(MAKE) tls-generate-dev
	$(MAKE) create-network
	$(MAKE) compose-build-detached

create-envs:
	test -f .env || cp .env.example .env

jwt-keys:
	./scripts/create-jwt-keys.sh

tls-generate-dev:
	./scripts/generate-dev-tls-certs.sh

create-network:
	docker network inspect $(DOCKER_NETWORK) >/dev/null 2>&1 || docker network create $(DOCKER_NETWORK)

proto:
	buf generate buf.build/zchelalo-labs/neuraclinic-proto-contracts \
		--path auth/v1/auth.proto \
		--path shared/v1/shared.proto \
		--path user/v1/user.proto

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down

compose:
	$(MAKE) create-network
	docker compose -f $(DOCKER_COMPOSE_FILE) up

compose-detached:
	$(MAKE) create-network
	docker compose -f $(DOCKER_COMPOSE_FILE) up -d

compose-build:
	$(MAKE) create-network
	docker compose -f $(DOCKER_COMPOSE_FILE) up --build

compose-build-detached:
	$(MAKE) create-network
	docker compose -f $(DOCKER_COMPOSE_FILE) up --build -d

compose-down:
	docker compose -f $(DOCKER_COMPOSE_FILE) down

fmt:
	go fmt ./...

lint:
	go vet ./...

test:
	go test ./...

coverage:
	go test ./... -coverprofile=coverage.out

build:
	mkdir -p dist
	go build -buildvcs=false -trimpath -o dist/neuraclinic-auth ./cmd

sqlc:
	sqlc generate

.PHONY: setup create-envs jwt-keys tls-generate-dev create-network proto migrate-up migrate-down compose compose-detached compose-build compose-build-detached compose-down fmt lint test coverage build sqlc
