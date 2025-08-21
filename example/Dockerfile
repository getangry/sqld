FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files for all modules
COPY go.mod ./
COPY go.sum ./
COPY example/go.mod example/go.sum ./example/
COPY adapters/pgx/go.mod ./adapters/pgx/
COPY adapters/pgx/go.sum ./adapters/pgx/

# Copy source code
COPY . .

# Change to example directory and download dependencies
WORKDIR /app/example
RUN go mod download

# Build the application
RUN go build -o /build/main main.go

FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/main /main

# Expose port
EXPOSE 8080

# Run the application
CMD ["/main"]