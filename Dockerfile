# ──────────────────────────────────────────────
# Stage 1 — Build
# ──────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Install build dependencies for CGO-free build
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependency downloads separately from source code
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source tree
COPY . .

# Build a statically linked binary (no CGO needed for lib/pq)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/bin/exchange ./cmd/exchange

# ──────────────────────────────────────────────
# Stage 2 — Run
# ──────────────────────────────────────────────
FROM alpine:3.20

# Non-root user for security
RUN addgroup -S zaraba && adduser -S zaraba -G zaraba

WORKDIR /app

# Certs + timezone data (needed for HTTPS calls and time ops)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo                 /usr/share/zoneinfo

# Copy the compiled binary
COPY --from=builder /app/bin/exchange ./exchange

# Copy embedded static assets (ui/ is embedded at compile time via efs.go)
# They are already baked into the binary, so no separate copy is needed.

RUN chown -R zaraba:zaraba /app
USER zaraba

# HTTP and gRPC ports
EXPOSE 8080 50051

# DATABASE_URL must be supplied at runtime (see docker-compose.yml)
ENV APP_ENV=production

ENTRYPOINT ["./exchange"]
CMD ["-addr", ":8080", "-grpc-addr", ":50051"]
