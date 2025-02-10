# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/plugin main.go

# Final lightweight stage
FROM alpine:latest
LABEL maintainer="ktrust DevOps Team"
LABEL description="Connection Balancer Plugin for Traefik"

WORKDIR /root/
COPY --from=builder /app/plugin .

ENTRYPOINT ["/root/plugin"]
