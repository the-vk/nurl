# Build stage
FROM golang:latest AS builder

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files (if they exist)
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o nurl .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache --update add ca-certificates

# Create app directory
RUN mkdir -p /app

# Copy binary from build stage
COPY --from=builder /app/nurl /app/nurl

# Set working directory
WORKDIR /app

# Expose port 80 (mapped from 8080)
EXPOSE 80

# Run the application
CMD ["/app/nurl"]