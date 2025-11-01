FROM golang:1.25.1-alpine3.22 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY api/go.mod api/go.sum ./


RUN go mod download

RUN go build -o loco-api .

FROM debian:bookworm-slim

WORKDIR /

COPY --from=builder /app/loco-api .

ENTRYPOINT ["/loco-api"]
