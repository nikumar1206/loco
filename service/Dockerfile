# Stage 1: Build
FROM golang:1.24.4-alpine AS builder

# Set necessary env vars
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Set working directory inside container
WORKDIR /app

# Copy go.mod and go.sum from parent directory
COPY service/go.mod service/go.sum ./


# Download dependencies
RUN go mod download

# Copy rest of the app source
COPY service/ .

# Build the Go binary
RUN go build -o loco-api .

# Stage 2: Run
FROM gcr.io/distroless/static-debian12

WORKDIR /

# Copy the binary from builder
COPY --from=builder /app/loco-api .

# Run the app
ENTRYPOINT ["/loco-api"]
