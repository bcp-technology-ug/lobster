.PHONY: build install test lint clean

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

lint:
	go vet ./...

clean:
	rm -rf bin/
