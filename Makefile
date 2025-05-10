PWD = $(shell pwd)

MODULE = rio
IMAGE_TAG ?= $(MODULE)
GITHUB_SHA ?= $(MODULE)

SRC = `go list -f {{.Dir}} ./... | grep -v /vendor/`

ndef = $(if $(value $(1)),,$(error $(1) not set))

all: fmt lint test

fmt:
	@echo "==> Formatting source code..."
	@go fmt $(SRC)

lint:
	@echo "==> Running lint check..."
	@go vet $(SRC)

test:
	@echo "==> Running tests..."
	@go clean ./...
	@go test -vet=off `go list ./... | grep -v cmd` -race -p 1 --cover

generate:
	@echo "==> Generating code..."
	@go generate ./...

build:
	@docker build \
		--build-arg GITHUB_TOKEN=$(GITHUB_TOKEN) \
		--build-arg COMMIT_HASH=$(COMMIT_HASH) \
		--target release \
		-f docker/Dockerfile \
		-t $(IMAGE_TAG) .

test-mariadb-up:
	@COMPOSE_HTTP_TIMEOUT=180 docker-compose \
		-f docker/docker-compose-mariadb.test.yml \
		-p mariadb_$(GITHUB_SHA) up \
		--force-recreate \
		--abort-on-container-exit \
		--exit-code-from app \
		--build

test-mariadb-down:
	@COMPOSE_HTTP_TIMEOUT=180 docker-compose \
		-f docker/docker-compose-mariadb.test.yml \
		-p mariadb_$(GITHUB_SHA) down \
 		-v --rmi local

test-mysql-up:
	@COMPOSE_HTTP_TIMEOUT=180 docker compose \
		-f docker/docker-compose-mysql.test.yml \
		-p mysql_$(GITHUB_SHA) up \
		--force-recreate \
		--abort-on-container-exit \
		--exit-code-from app \
		--build

test-mysql-down:
	@COMPOSE_HTTP_TIMEOUT=180 docker-compose \
		-f docker/docker-compose-mysql.test.yml \
		-p mysql_$(GITHUB_SHA) down \
 		-v --rmi local

dev-up:
	@docker compose \
		-f docker/docker-compose.dev.yml \
		-p $(GITHUB_SHA) up --build -d

dev-down:
	@docker compose \
		-f docker/docker-compose.dev.yml \
		-p $(GITHUB_SHA) down \
 		-v --rmi local

dev-ps:
	@docker compose \
		-f docker/docker-compose.dev.yml \
		-p $(GITHUB_SHA) ps -a

gen-docs:
	@swag init -g cmd/server/main.go --ot yaml --overridesFile docker/.swaggo -o docs
	
.PHONY: all fmt lint test install test-mariadb-up test-mariadb-down test-mysql-up test-mysql-down dev-up dev-down dev-ps build generate
