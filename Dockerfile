# Multi-stage Dockerfile for goupdate
# Supports multi-architecture builds (linux/amd64, linux/arm64)
# Works on both Linux and macOS Docker hosts
#
# Standalone usage (DockerHub):
#   docker run -v $(pwd):/workspace goupdate:latest outdated
#   docker run -v $(pwd):/workspace goupdate:latest update --patch --yes
#   docker run -v $(pwd):/workspace goupdate:latest scan
#
# Build multi-arch:
#   docker buildx build --platform linux/amd64,linux/arm64 -t goupdate:latest .

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with CGO disabled for static binary
# TARGETOS and TARGETARCH are automatically set by Docker buildx
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w -X github.com/user/goupdate/cmd.Version=${VERSION}" -o goupdate main.go

# Final stage - minimal alpine image
FROM alpine:3.20

# Re-declare VERSION in final stage for labels
ARG VERSION=dev

# Image metadata
LABEL org.opencontainers.image.title="goupdate"
LABEL org.opencontainers.image.description="Scan, list, and update dependencies across npm, Go, Composer, pip, and .NET from one CLI. Open-source, runs locally, no cloud services or git required."
LABEL org.opencontainers.image.source="https://github.com/ajxudir/goupdate"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="${VERSION}"

# Install runtime dependencies for package managers
# - git: for version control operations
# - ca-certificates: for HTTPS requests
# - nodejs/npm: for JavaScript ecosystem (npm, pnpm, yarn)
# - go: for Go modules
# - php/composer: for PHP ecosystem
# - python3/pip: for Python ecosystem
RUN apk add --no-cache \
    git \
    ca-certificates \
    nodejs \
    npm \
    go \
    php83 \
    php83-phar \
    php83-mbstring \
    php83-openssl \
    php83-curl \
    composer \
    python3 \
    py3-pip

# Create non-root user for security
RUN addgroup -g 1000 goupdate && \
    adduser -u 1000 -G goupdate -s /bin/sh -D goupdate

# Copy binary from builder
COPY --from=builder /build/goupdate /usr/local/bin/goupdate

# Set proper permissions
RUN chmod +x /usr/local/bin/goupdate

# Create config and work directories
RUN mkdir -p /home/goupdate/.config/goupdate /workspace && \
    chown -R goupdate:goupdate /home/goupdate /workspace

# Switch to non-root user
USER goupdate

WORKDIR /workspace

# Default command shows help
ENTRYPOINT ["goupdate"]
CMD ["--help"]
