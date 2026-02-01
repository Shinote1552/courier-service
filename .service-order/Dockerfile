# syntax=docker/dockerfile:1
FROM golang:1.24 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o service ./cmd/service

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/service /service-order
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/service-order"]
