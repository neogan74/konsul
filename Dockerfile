# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with version info
ARG VERSION=dev
ARG BUILD_DATE
ARG VCS_REF
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags "-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE} -X main.gitCommit=${VCS_REF}" \
    -o konsul \
    ./cmd/konsul/

# Build konsulctl CLI
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags "-s -w" \
    -o konsulctl \
    ./cmd/konsulctl/

# Final stage
FROM alpine:latest

# Add metadata labels
LABEL org.opencontainers.image.title="Konsul" \
      org.opencontainers.image.description="Lightweight service discovery and KV store" \
      org.opencontainers.image.vendor="Konsul" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}"

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 -S konsul && \
    adduser -u 1000 -S konsul -G konsul

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/konsul .
COPY --from=builder /app/konsulctl /usr/local/bin/

# Create data, backup, and certs directories with proper ownership
RUN mkdir -p /app/data /app/backups /app/certs && \
    chown -R konsul:konsul /app

# Switch to non-root user
USER konsul

# Expose ports (HTTP/HTTPS and DNS)
EXPOSE 8888 8600/udp 8600/tcp

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8888/health/live || exit 1

# Environment variables with defaults
ENV KONSUL_HOST="" \
    KONSUL_PORT=8888 \
    KONSUL_LOG_LEVEL=info \
    KONSUL_LOG_FORMAT=json

# Run the application
CMD ["./konsul"]