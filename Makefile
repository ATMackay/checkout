# Alex Mackay 2025

# Build folder
BUILD_FOLDER = build
COVERAGE_BUILD_FOLDER    ?= $(BUILD_FOLDER)/coverage
UNIT_COVERAGE_OUT        ?= $(COVERAGE_BUILD_FOLDER)/ut_cov.out
BIN                      ?= $(BUILD_FOLDER)/checkout

# Packages
PKG                      ?= github.com/ATMackay/checkout
CONSTANTS_PKG            ?= $(PKG)/constants


# Git based version
VERSION_TAG    ?= $(shell git describe --tags)
GIT_COMMIT     ?= $(shell git rev-parse HEAD)
BUILD_DATE     ?= $(shell date -u +'%Y-%m-%d %H:%M:%S')
COMMIT_DATE    ?= $(shell git show -s --format="%ci" $(shell git rev-parse HEAD))
DIRTY          ?= false

LDFLAGS := -s -w \
  -X '$(CONSTANTS_PKG).Version=$(VERSION_TAG)' \
  -X '$(CONSTANTS_PKG).CommitDate=$(COMMIT_DATE)' \
  -X '$(CONSTANTS_PKG).GitCommit=$(GIT_COMMIT)' \
  -X '$(CONSTANTS_PKG).BuildDate=$(BUILD_DATE)' \
  -X '$(CONSTANTS_PKG).Dirty=$(DIRTY)'

build:
	@mkdir -p build
	@echo ">> building $(BIN) (version=$(VERSION_TAG) commit=$(GIT_COMMIT) dirty=$(DIRTY))"
	GO111MODULE=on go build -ldflags "$(LDFLAGS)" -o $(BIN)
	@echo  "Checkout server successfully built. To run the application execute './$(BIN) run'"

run: build
	@./$(BUILD_FOLDER)/checkout run --memory-db

build/coverage:
	@mkdir -p $(COVERAGE_BUILD_FOLDER)

test: build/coverage
	@go test -cover -coverprofile $(UNIT_COVERAGE_OUT) -v ./...

test-coverage: test
	@go tool cover -html=$(UNIT_COVERAGE_OUT)

docker:
	@./build-docker.sh
	@echo  "To run the application execute 'docker run -p 8080:8080 -e DB_HOST=<DB_HOST> -e DB_PASSWORD=<DB_PASSWORD> checkout'"

docker-run-db:
	@docker compose -f docker-compose.yml up -d database

openapi-clean:
	rm -rf ./docs/openapi/*
	@echo "Deleted docs/openapi/openapi.json"

swag-install:
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo  "Installed swag"

openapi: swag-install openapi-clean
	@swag init \
		-g main.go \
		--parseDependency --parseInternal \
		-o ./docs/openapi/openapi.json \
		-ot json
	@echo "✅ Wrote OpenAPI to docs/openapi/openapi.json"

api-docs: openapi
	@echo "✅ All docs generated."

.PHONY: build run docker test test-coverage docker-run-db swag-install api-docs