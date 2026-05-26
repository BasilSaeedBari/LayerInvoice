# Stage 1: Build static Go binary
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY . .

# Compile optimized static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o layerinvoice .

# Stage 2: Distribute lightweight runner
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy static binary from builder
COPY --from=builder /app/layerinvoice .

# Expose server port
EXPOSE 8080

# Run server
ENTRYPOINT ["./layerinvoice"]
