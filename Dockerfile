# Use the official Golang image as the builder stage
FROM golang:1.23.2-alpine AS builder

# Create and set the working directory
RUN mkdir -p /app
WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project and build the application
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o jenkins-pr-collector

# Use the official Alpine image as the base for the final stage
FROM alpine:3.19.1

# Create and set the working directory
RUN mkdir -p /app
WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /app/jenkins-pr-collector .

# Copy required files
COPY plugins.json .
COPY report.json .
COPY entrypoint.sh .

# Set a non-sensitive environment variable with a default value
ENV START_DATE="2024-08-01"

# Ensure the entrypoint script is executable
RUN chmod +x /app/entrypoint.sh

# Set the entrypoint for the container
ENTRYPOINT ["/app/entrypoint.sh"]
