.PHONY: build install test lint clean test-cli

# Build both binaries into ./bin/
build:
	go build -o bin/lobster   ./cmd/lobster
	go build -o bin/lobsterd  ./cmd/lobsterd

# Install the CLI into $GOPATH/bin (or $GOBIN if set)
install:
	go install ./cmd/lobster
	go install ./cmd/lobsterd

test:
	go test ./...

# Run the self-testing CLI integration suite using lobster itself.
# Requires lobster to be installed (runs `go install` first).
# Runs from tests/ so the suite has its own isolated store.
test-cli:
	go install ./cmd/lobster
	cd tests && lobster run --features 'features/*.feature' --ci --migrations-dir ../migrations

lint:
	go vet ./...

clean:
	rm -rf bin/
