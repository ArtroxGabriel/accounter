FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache \
    gcc \
    musl-dev \
    ca-certificates

COPY go.mod go.sum /app/

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target="/root/.cache/go-build" \
    CGO_ENABLED=0 GOOS=linux go build -o go-worker  -ldflags '-s -w -extldflags "-static"' ./cmd/app

FROM ubuntu:oracular AS user
RUN useradd -u 10001 scratchuser

FROM scratch
WORKDIR /app


COPY --from=builder /app/go-worker ./
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=user /etc/passwd /etc/passwd

USER scratchuser
STOPSIGNAL SIGINT

CMD ["/app/app"]
