# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o konsul ./cmd/konsul/

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 -S konsul && \
    adduser -u 1000 -S konsul -G konsul

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/konsul .

# Create data and backup directories with proper ownership
RUN mkdir -p /app/data /app/backups && \
    chown -R konsul:konsul /app

# Switch to non-root user
USER konsul

# Expose port
EXPOSE 8888

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8888/services/ || exit 1

# Run the application
CMD ["./konsul"]