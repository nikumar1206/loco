FROM golang:1.24.4-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY service/go.mod service/go.sum ./


RUN go mod download

COPY service/ .

RUN go build -o loco-api .

FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=builder /app/loco-api .

ENTRYPOINT ["/loco-api"]
