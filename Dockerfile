FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build HTTP service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o service ./cmd/service

# Build Kafka worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o worker-kafka-consumer ./cmd/worker-order-status-changed

FROM gcr.io/distroless/base-debian12
WORKDIR /

# Copy both binaries from builder
COPY --from=builder /app/service /service-courier
COPY --from=builder /app/worker-kafka-consumer /worker-kafka

# Copy migrations 
COPY --from=builder /app/migrations /migrations

# Default port for HTTP service (overriden in docker-compose)
EXPOSE 8080
USER nonroot:nonroot
# ENTRYPOINT ["/service-courier"]