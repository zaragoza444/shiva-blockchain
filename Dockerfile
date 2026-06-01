FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download 2>/dev/null || true
COPY cmd ./cmd
COPY internal ./internal
COPY configs ./configs
RUN go mod tidy && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /onexd ./cmd/onexd && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /onex ./cmd/onex && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /onex-bridge ./cmd/onex-bridge

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=builder /onexd /onex /onex-bridge /usr/local/bin/
COPY configs /app/configs
RUN adduser -D -H onex && mkdir -p /data /bridge-data && chown -R onex:onex /data /bridge-data
USER onex
EXPOSE 8545 30303 9338
VOLUME ["/data", "/bridge-data"]
ENV ONEX_DATADIR=/data
ENV ONEX_PROJECT_ROOT=/app
ENTRYPOINT ["/usr/local/bin/onexd"]
CMD ["-datadir", "/data", "-api", ":8545", "-listen", ":30303", "-seeds", "/app/configs/seeds-mainnet.json"]
