# Multi-stage Dockerfile for Go backend

# Stage 1: build the Go binary
FROM golang:1.25-alpine AS builder

# Install git for go mod
RUN apk add --no-cache git

WORKDIR /app

# Copy module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source
COPY . .

# Build binary for linux/amd64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o backend ./cmd/server


# Stage 2: create minimal image with non-root user
FROM alpine:3.18

# Create non-root user
RUN addgroup -S app && adduser -S -G app app

# Create /data directory
RUN mkdir -p /data/scenarios /data/runs && \
    chown -R app:app /data

# Copy binary from builder
COPY --from=builder /app/backend /usr/local/bin/backend

# Use non-root user
USER app

# Expose port
EXPOSE 8080

# Default environment variables
ENV ADDR=:8080
ENV DATA_DIR=/data

# Create entrypoint script to handle env vars if needed
ENTRYPOINT ["/usr/local/bin/backend"]
