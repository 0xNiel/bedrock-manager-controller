# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /workspace

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o controller .

# Final stage
FROM gcr.io/distroless/static:nonroot

# Copy the binary from builder stage
COPY --from=builder /workspace/controller /controller

# Use nonroot user
USER 65532:65532

# Set entrypoint
ENTRYPOINT ["/controller"]
