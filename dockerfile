# Use Golang base image to build the application
FROM golang:1.22 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules and download the dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Start a new stage from scratch
FROM alpine:latest

# Install CA certificates for secure HTTPS communication
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /app

# Copy the Pre-built binary from the builder image
COPY --from=builder /app/main /app/main

# Copy environment variables
COPY .env /app/.env

# Ensure the binary is executable
RUN chmod +x /app/main

# Expose port 8080
EXPOSE 8080

# Run the Go app
CMD ["/app/main"]
