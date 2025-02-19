# Alex Mackay 2025

# Build folder
BUILD_FOLDER = build

# Test coverage variables
COVERAGE_BUILD_FOLDER = $(BUILD_FOLDER)/coverage
UNIT_COVERAGE_OUT  = $(COVERAGE_BUILD_FOLDER)/ut_cov.out

# Git based version
VERSION_TAG ?= $(shell git describe --tags)
GIT_COMMIT ?= $(shell git rev-parse HEAD)
BUILD_DATE ?= $(shell date -u +'%Y-%m-%d %H:%M:%S')
COMMIT_DATE ?= $(shell git show -s --format="%ci" $(shell git rev-parse HEAD))

build:
	@GO111MODULE=on go build -o $(BUILD_FOLDER)/checkout \
	-ldflags=" -X 'github.com/ATMackay/checkout/constants.Version=$(VERSION_TAG)' -X 'github.com/ATMackay/checkout/constants.CommitDate=$(COMMIT_DATE)' -X 'github.com/ATMackay/checkout/constants.BuildDate=$(BUILD_DATE)' -X 'github.com/ATMackay/checkout/constants.GitCommit=$(GIT_COMMIT)'"
	@echo  "Checkout server successfully built. To run the application execute './$(BUILD_FOLDER)/checkout run'"

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
	@echo  "To run the application execute 'docker run -p 8000:8000 -e DB_HOST=<DB_HOST> -e DB_PASSWORD=<DB_PASSWORD> checkout'"

docker-run-db:
	@docker compose -f docker-compose.yml up -d database

swag-install:
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo  "Installed swag"

docs: swag-install
	@swag init -g main.go --output docs/generated
	@echo "Swagger documentation generated"

.PHONY: build run docker test test-coverage docker-run-db swag-install docs