# syntax=docker/dockerfile:1
# ─────────────────────────────────────────────────────────────────────────────
# Build stage
# CGO is required by mattn/go-sqlite3 which embeds the SQLite C library.
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src

# Download dependencies separately so Docker can cache this layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux \
    go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /bin/lobster \
      ./cmd/lobster

# ─────────────────────────────────────────────────────────────────────────────
# Runtime stage
# ─────────────────────────────────────────────────────────────────────────────
FROM alpine:3.21

# ca-certificates: needed for TLS connections to remote endpoints.
# tzdata: needed for time-zone aware scheduling steps.
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/lobster /usr/local/bin/lobster

ENTRYPOINT ["lobster"]
