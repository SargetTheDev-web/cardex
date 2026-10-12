# Build stage
FROM golang:1.26.3 AS builder

WORKDIR /app

# Download dependencies first for better Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build a statically linked production binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /app/cardex-api ./cmd/api

# Runtime stage
FROM alpine:3.22

WORKDIR /app

# CA certificates are needed for HTTPS/TLS connections
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/cardex-api /app/cardex-api

EXPOSE 8080

USER 65532:65532

ENTRYPOINT ["/app/cardex-api"]