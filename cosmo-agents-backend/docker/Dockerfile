FROM 471112636744.dkr.ecr.ap-southeast-1.amazonaws.com/rockship/env:v0.0.1 as certificates
FROM golang:1.25.6-alpine AS builder

WORKDIR /app

ARG TAG

# Install necessary build tools
RUN apk add --no-cache git

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .
COPY --from=certificates backend/cosmo-agents-golang/${TAG}/coze_oauth_private_key.pem ./coze_oauth_private_key.pem
COPY --from=certificates backend/cosmo-agents-golang/${TAG}/.env ./.env

# Build the application
RUN go build -o bin/server ./cmd/server && \
    go build -o bin/worker ./cmd/worker

# Create a smaller image for running the application
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy the binary from the builder stage
COPY --from=builder /app/bin/server .
COPY --from=builder /app/bin/worker .

# Copy runtime files
COPY --from=builder /app/docs ./docs
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/internal/templates ./internal/templates

# Expose the application port
EXPOSE 8080