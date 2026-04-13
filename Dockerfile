# MXKeys - Matrix Key Notary Server
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o mxkeys ./cmd/mxkeys

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 mxkeys

# Create directories
RUN mkdir -p /var/lib/mxkeys/keys /etc/mxkeys && \
    chown -R mxkeys:mxkeys /var/lib/mxkeys /etc/mxkeys

# Copy binary
COPY --from=builder /build/mxkeys /usr/local/bin/mxkeys

# Copy default config
COPY config.example.yaml /etc/mxkeys/config.yaml

USER mxkeys

EXPOSE 8448

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8448/_mxkeys/ready || exit 1

ENTRYPOINT ["/usr/local/bin/mxkeys"]
