FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o traffic-generator ./cmd

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/traffic-generator /traffic-generator
CMD ["/traffic-generator"]
