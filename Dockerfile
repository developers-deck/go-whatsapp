# WhatsApp Web Multidevice API - Main Dockerfile
# This file serves as the entry point for Railway and other cloud platforms
# It references the detailed Dockerfile in the docker/ directory

# Use the existing Dockerfile from the docker/ directory
FROM golang:1.24-alpine3.20 AS builder

# Install build dependencies
RUN apk update && apk add --no-cache gcc musl-dev gcompat

# Set working directory
WORKDIR /whatsapp

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Fetch dependencies
RUN go mod download

# Copy the entire source code
COPY . .

# Build the binary with optimizations
RUN go build -a -ldflags="-w -s" -o /app/whatsapp ./main.go

# Build the final image
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ffmpeg curl

# Set working directory
WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/whatsapp /app/whatsapp

# Create necessary directories
RUN mkdir -p /app/storages /app/statics

# Expose port
EXPOSE 3000

# Set environment variables
ENV APP_PORT=3000
ENV APP_DEBUG=false

# Run the binary
ENTRYPOINT ["/app/whatsapp"]

# Default command
CMD ["rest"]
