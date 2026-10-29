# Build Stage
FROM golang:1.26-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o medtrust_backend ./cmd/main.go

# Runtime Stage
FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=builder /app/medtrust_backend .
COPY config/config.yaml ./config/config.yaml
COPY fabric_crypto ./fabric_crypto
RUN mkdir -p ./data/ipfs_storage ./data/fabric_ledger

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && chown -R appuser:appgroup /app
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=3 \
  CMD wget -q -O - http://localhost:8080/api/v1/health || exit 1

CMD ["./medtrust_backend"]
