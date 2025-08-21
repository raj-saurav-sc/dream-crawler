FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/crawler ./cmd/crawler
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/orchestrator ./cmd/orchestrator
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/indexer ./cmd/indexer
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/content-processor ./cmd/content-processor
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api ./cmd/api

# Final stage - use the same Go image to avoid Alpine package issues
FROM golang:1.23-alpine

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/bin/ ./bin/

# Default command (can be overridden)
CMD ["./bin/crawler"]
