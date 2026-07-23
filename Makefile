# Alex Mackay 2025

# Build folder
BUILD_FOLDER = build
COVERAGE_BUILD_FOLDER    ?= $(BUILD_FOLDER)/coverage
UNIT_COVERAGE_OUT        ?= $(COVERAGE_BUILD_FOLDER)/ut_cov.out
BIN                      ?= $(BUILD_FOLDER)/checkout

# Packages
PKG                      ?= github.com/ATMackay/checkout
CONSTANTS_PKG            ?= $(PKG)/constants


# Version.
#
# The commit SHA, commit date and dirty flag are stamped automatically by the Go
# toolchain (-buildvcs=true) and read at runtime via runtime/debug.ReadBuildInfo
# — see the constants package. Only the semver tag and the wall-clock build date
# need injecting (the build date is not reproducible, hence not from VCS).
VERSION    ?= $(shell git describe --tags 2>/dev/null || echo dev)
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := -s -w \
  -X '$(CONSTANTS_PKG).Version=$(VERSION)' \
  -X '$(CONSTANTS_PKG).BuildDate=$(BUILD_DATE)'

# Static build settings.
#
# go-sqlite3 is a cgo package, so CGO_ENABLED=1 is mandatory — a CGO_ENABLED=0
# binary compiles but fails at runtime with "go-sqlite3 requires cgo to work".
# Linking statically against musl lets the resulting binary run on a base image
# with no libc at all (distroless/static, scratch).
#
#   osusergo,netgo               use the pure-Go user and DNS resolvers rather
#                                than libc NSS, which does not work statically
#   sqlite_omit_load_extension   drop SQLite's dlopen-based extension loading,
#                                which cannot work in a static binary
STATIC_TAGS    := osusergo,netgo,sqlite_omit_load_extension
STATIC_LDFLAGS := $(LDFLAGS) -linkmode external -extldflags "-static"

build:
	@mkdir -p build
	@echo ">> building $(BIN) (version=$(VERSION))"
	GO111MODULE=on go build -buildvcs=true -ldflags "$(LDFLAGS)" -o $(BIN)
	@echo  "Checkout server successfully built. To run the application execute './$(BIN) run'"

# build-static produces a fully static binary for container images. Requires a
# C toolchain (build-base on Alpine).
build-static:
	@mkdir -p build
	@echo ">> building static $(BIN) (version=$(VERSION))"
	CGO_ENABLED=1 GO111MODULE=on go build \
	  -buildvcs=true \
	  -tags "$(STATIC_TAGS)" \
	  -ldflags '$(STATIC_LDFLAGS)' \
	  -o $(BIN)

install: build
	mv $(BIN) $(GOBIN)

# Run orders service with DEBUG logging
run-orders: build
	@./$(BUILD_FOLDER)/checkout run orders --memory-db --log-level debug

build/coverage:
	@mkdir -p $(COVERAGE_BUILD_FOLDER)

test: build/coverage
	@go test -cover -coverprofile $(UNIT_COVERAGE_OUT) ./...

test-integration:
	@echo "🧪 Running integration tests..."
	@go test -cover -tags=integration ./integration ./messaging/kafka -count=1 -timeout=15m

test-coverage: test
	@go tool cover -html=$(UNIT_COVERAGE_OUT)

docker:
	@./build-docker.sh
	@echo  "To run the application execute 'docker run -p 8080:8080 -e DB_HOST=<DB_HOST> -e DB_PASSWORD=<DB_PASSWORD> checkout'"

docker-run-postgres:
	@docker compose -f docker-compose.yml --profile postgres up --force-recreate

docker-run-sqlite:
	@docker compose -f docker-compose.yml --profile sqlite up --force-recreate

# Run the full event system: postgres + kafka + orders + notifier.
# Postgres publishes on host 5433 (not 5432) so it does not collide with a local
# Postgres — the services reach it over the compose network regardless. Override
# PG_HOST_PORT / ORDERS_PORT / NOTIFIER_PORT if those host ports are taken.
# Requires the checkout image (run `make docker` first).
docker-run-events:
	@PG_HOST_PORT=$${PG_HOST_PORT:-5433} docker compose -f docker-compose.yml --profile events up --force-recreate

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
		-o ./docs/openapi/ \
		-ot json
	@echo "✅ Wrote OpenAPI to docs/openapi/swagger.json"

api-docs: openapi
	@echo "✅ All docs generated."

mocks:
	@go install go.uber.org/mock/mockgen@latest
	@go generate ./...

.PHONY: build build-static run docker test test-coverage docker-run-postgres docker-run-sqlite docker-run-events swag-install openapi api-docs mocks