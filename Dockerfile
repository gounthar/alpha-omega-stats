# Use the official Golang image as the builder stage
FROM golang:1.24.1-alpine AS builder

# Create and set the working directory
RUN mkdir -p /app
WORKDIR /app

# Copy source code first to get dependencies
COPY . .

# Initialize Go modules and get dependencies
RUN go mod init jenkins.io/alpha-omega-stats || true && \
    go mod tidy && \
    go get -d ./... && \
    go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o jenkins-pr-collector

# Use the official Alpine image as the base for the final stage
FROM alpine:3.21.3

# Install required packages
RUN apk add --no-cache \
    bash \
    curl \
    ca-certificates \
    tzdata

# Create and set the working directory
RUN mkdir -p /app
WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /app/jenkins-pr-collector .

# Create empty JSON files if they don't exist in the build context
RUN touch plugins.json report.json

# Copy JSON files if they exist in the build context (will overwrite empty files)
COPY plugins.json* report.json* ./

# Copy entrypoint script
COPY entrypoint.sh .

# Set a non-sensitive environment variable with a default value
ENV START_DATE="2024-08-01"

# Ensure the entrypoint script is executable
RUN chmod +x /app/entrypoint.sh

# Set the entrypoint for the container
ENTRYPOINT ["/app/entrypoint.sh"]
