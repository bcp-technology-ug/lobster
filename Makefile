.PHONY: build install test test-unit test-all test-cli test-daemon test-docker lint clean generate proto sqlc \
        docker-build docker-push docker-up docker-down

IMAGE_CLI    ?= ghcr.io/bcp-technology-ug/lobster
IMAGE_DAEMON ?= ghcr.io/bcp-technology-ug/lobsterd
TAG          ?= dev

# Build both binaries into ./bin/
build:
	go build -o bin/lobster   ./cmd/lobster
	go build -o bin/lobsterd  ./cmd/lobsterd

# Install the CLI into $GOPATH/bin (or $GOBIN if set)
install:
	go install ./cmd/lobster
	go install ./cmd/lobsterd

test: test-unit test-all

test-unit:
	go test ./...

# Run ALL integration scenarios in a single lobster run: CLI tests + daemon
# tests.  lobsterd is started automatically by lobster via Docker Compose
# (configured in tests/lobster.yaml) and torn down after the run completes.
# Requires Docker on the host.
COMPOSE_FILES    := -f config/docker-compose.yml

test-all:
	go install ./cmd/lobster
	docker compose $(COMPOSE_FILES) build --quiet
	# Clean up any containers from a previous run before starting fresh.
	docker compose $(COMPOSE_FILES) down -v 2>/dev/null || true
	docker rm -f lobster-self-tests-lobsterd-1 2>/dev/null || true
	cd tests && lobster run --ci

# Run only the CLI scenarios (no daemon, no Docker).
test-cli:
	go install ./cmd/lobster
	cd tests && lobster run \
	  --tags '~@docker ~@daemon ~@integration' \
	  --no-compose \
	  --ci

# Run only the @daemon scenarios against an already-running lobsterd.
# Builds and starts lobsterd automatically via lobster.yaml compose config.
test-daemon:
	go install ./cmd/lobster
	docker compose $(COMPOSE_FILES) build --quiet
	docker compose $(COMPOSE_FILES) down -v 2>/dev/null || true
	docker rm -f lobster-self-tests-lobsterd-1 2>/dev/null || true
	cd tests && lobster run --tags '@daemon' --ci

# Run the Docker Compose image-lifecycle tests (@docker tag).
# These test image build, compose up/down, health endpoint, persistence.
# Requires Docker on the host.
test-docker:
	go install ./cmd/lobster
	lobster run \
	  --features 'tests/features/docker-*.feature' \
	  --tags "@docker" \
	  --env LOBSTER_ROOT=$(CURDIR) \
	  --ci \
	  --migrations-dir migrations

lint:
	go vet ./...

clean:
	rm -rf bin/

# Code generation targets
# ─────────────────────────────────────────────────────────────────────────────

# generate runs all code generators in the correct order.
generate: proto sqlc

# proto regenerates Go + OpenAPI stubs from .proto files using buf.
proto:
	buf generate

# sqlc regenerates type-safe Go query code from SQL files.
sqlc:
	sqlc generate

# ─────────────────────────────────────────────────────────────────────────────
# Docker targets
# ─────────────────────────────────────────────────────────────────────────────

# Build both Docker images locally.
docker-build:
	docker build -f config/Dockerfile.lobster  -t $(IMAGE_CLI):$(TAG)    .
	docker build -f config/Dockerfile.lobsterd -t $(IMAGE_DAEMON):$(TAG) .

# Push both images to the registry (requires prior docker login).
docker-push: docker-build
	docker push $(IMAGE_CLI):$(TAG)
	docker push $(IMAGE_DAEMON):$(TAG)

# Start the daemon stack defined in config/docker-compose.yml.
docker-up:
	docker compose -f config/docker-compose.yml up -d --build

# Stop the daemon stack (data volume is preserved).
docker-down:
	docker compose -f config/docker-compose.yml down
