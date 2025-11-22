# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY go.mod go.sum ./

# Build the application (no external dependencies needed)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/coordinator

# Final stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Run the binary
ENTRYPOINT ["./main"]
