# Build stage
FROM golang:1.22.10-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata wget

# Copy dependency files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/api \
    ./cmd/api

# Production stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy timezone data for proper timezone handling
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy required binaries
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /usr/bin/wget /usr/bin/wget

# Run as non-root user
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/api"]
