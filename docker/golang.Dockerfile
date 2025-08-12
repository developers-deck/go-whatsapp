############################
# STEP 1 build executable binary
############################
FROM golang:1.24-alpine3.20 AS builder
RUN apk update && apk add --no-cache gcc musl-dev gcompat
WORKDIR /whatsapp

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Fetch dependencies.
RUN go mod download

# Copy the entire source code
COPY . .

# Build the binary with optimizations
RUN go build -a -ldflags="-w -s" -o /app/whatsapp ./main.go

#############################
## STEP 2 build a smaller image
#############################
FROM alpine:3.20
RUN apk add --no-cache ffmpeg curl
WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/whatsapp /app/whatsapp

# Create necessary directories
RUN mkdir -p /app/storages /app/statics

# Run the binary.
ENTRYPOINT ["/app/whatsapp"]

CMD [ "rest" ]