FROM golang:1.23.2-alpine AS builder

RUN mkdir -p /app
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o jenkins-pr-collector

FROM alpine:3.21.3

RUN mkdir -p /app
WORKDIR /app
COPY --from=builder /app/jenkins-pr-collector .

# Copy required files
COPY plugins.json .
COPY report.json .
COPY entrypoint.sh .

# Default start date for PR collection
# This can be overridden at runtime by setting the START_DATE environment variable
# Format: YYYY-MM-DD (e.g., 2024-08-01)
ENV START_DATE="2024-08-01"

# Ensure the entrypoint script is executable
RUN chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
