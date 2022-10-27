BIN := "./bin/calendar"
BIN_SENDER := "$(BIN)_sender"
BIN_SCHEDULER := "$(BIN)_scheduler"
DOCKER_IMG="calendar:develop"
DEFAULT_CONFIG_PATH="$(shell pwd)/.calendar_config.yaml"
MIGRATIONS_FOLDER="migrations"

DB_DRIVER:=${GOOSE_DRIVER}
ifeq ($(DB_DRIVER),)
DB_DRIVER:="postgres"
endif
DB_STRING:=${GOOSE_DBSTRING}
ifeq ($(DB_STRING),)
DB_STRING:="host=localhost port=5432 user=postgres dbname=postgres sslmode=disable"
endif

GIT_HASH := $(shell git log --format="%h" -n 1)
LDFLAGS := -X main.release="develop" -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S) -X main.gitHash=$(GIT_HASH)

build:
	go build -v -o $(BIN) -ldflags "$(LDFLAGS)" ./cmd/calendar
	go build -v -o $(BIN_SENDER) -ldflags "$(LDFLAGS)" ./cmd/sender
	go build -v -o $(BIN_SCHEDULER) -ldflags "$(LDFLAGS)" ./cmd/scheduler

run: build
	$(BIN) -config $(DEFAULT_CONFIG_PATH)

build-img:
	docker build \
		--build-arg=LDFLAGS="$(LDFLAGS)" \
		-t $(DOCKER_IMG) \
		-f build/Dockerfile .

run-img: build-img
	docker run $(DOCKER_IMG)

version: build
	$(BIN) version

test:
	go test -race ./...

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.41.1

lint: install-lint-deps
	golangci-lint run ./...

.PHONY: build run build-img run-img version test lint

migrate-up:
	GOOSE_DRIVER=$(DB_DRIVER) GOOSE_DBSTRING=$(DB_STRING) goose -dir $(MIGRATIONS_FOLDER) up

migrate-down:
	GOOSE_DRIVER=$(DB_DRIVER) GOOSE_DBSTRING=$(DB_STRING) goose -dir $(MIGRATIONS_FOLDER) down

go-generate:
	buf generate --path "api/event"
	go generate ./...

up:
	docker compose up -d

down:
	docker compose down

integration-tests:
	docker compose -f docker-compose-test.yaml up --exit-code-from "tests"
