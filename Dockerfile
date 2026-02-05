FROM golang:1.25-alpine AS builder

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build the application with CGO enabled (required for SQLite)
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/alpha-hygiene-backend ./cmd/app

FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Create data directory for SQLite
RUN mkdir -p /app/data

# Copy the executable from builder stage
COPY --from=builder /app/alpha-hygiene-backend .

# Copy configuration
COPY config/config.yaml config/
COPY .env .

# Copy Swagger files
COPY docs/swagger.json docs/
COPY docs/swagger.yaml docs/

# Expose port
EXPOSE 8080

# Run the application
CMD ["./alpha-hygiene-backend"]
