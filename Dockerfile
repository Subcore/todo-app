# Stage 1: Build
FROM golang:1.24.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# CGO_ENABLED=0 for a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api/main.go

# Stage 2: Final
FROM alpine:3.21.3

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/main .
# Copy only docs.html (frontend now served by separate Nginx container)
COPY --from=builder /app/web/docs.html ./web/docs.html
COPY --from=builder /app/openapi.yaml ./openapi.yaml

# Use the non-root user
USER appuser

# Expose the application port
EXPOSE 8080

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run the binary
ENTRYPOINT ["./main"]
