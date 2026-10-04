# ==========================================
# Stage 1: Build Environment
# ==========================================
FROM golang:1.25-alpine AS builder

# Install build dependencies (git, ca-certificates)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache Go modules by copying dependency manifests first
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source tree
COPY . .

# Compile static, highly-optimized binaries (CGO disabled for pure Go builds)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /bin/api ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /bin/worker ./cmd/worker

# ==========================================
# Stage 2: Minimal Runtime Environment
# ==========================================
FROM alpine:3.19

# Install CA certificates for TLS/HTTPS communication with AWS APIs (SQS, Bedrock, DynamoDB)
# and tzdata for accurate RFC3339 timestamp handling
RUN apk add --no-cache ca-certificates tzdata

# Run as a non-root user for security best practices
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binaries from the builder stage
COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/worker /app/worker

# Grant permissions to the non-root user
RUN chown -R appuser:appgroup /app
USER appuser

EXPOSE 8081

# Default entrypoint runs the API gateway.
# This is overridden by docker-compose for the worker service (`command: ["/app/worker"]`).
CMD ["/app/api"]