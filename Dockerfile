# Build stage
ARG GO_IMAGE=golang:1.24.11-alpine
ARG RUNTIME_IMAGE=alpine:3.22
FROM ${GO_IMAGE} AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Copy source code
COPY . .

# Module downloads and the build can hit transient proxy resets on some
# networks. Retry in-place so already-downloaded modules persist between
# attempts, then fail loudly if the build still did not complete.
RUN set -eu; \
    for attempt in $(seq 1 6); do \
        if CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o agentos ./cmd/agentos; then \
            break; \
        fi; \
        echo "go build attempt ${attempt} failed; retrying"; \
        sleep 3; \
    done; \
    test -x agentos

# Final stage
FROM ${RUNTIME_IMAGE}

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 agno && \
    adduser -D -u 1000 -G agno agno

# Set working directory
WORKDIR /home/agno

# Copy binary from builder
COPY --from=builder /app/agentos .

# Change ownership
RUN chown -R agno:agno /home/agno

# Switch to non-root user
USER agno

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./agentos"]
