# Stage 1: Build the application
# Using golang:1.23-alpine to match the go.mod version (go 1.23.0, toolchain go1.23.9)
FROM golang:1.23-alpine AS builder

# Set necessary environment variables for building a static binary
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Create a non-root user for running the application
# Using -S (system user) and -G (group) for more standard alpine user creation
RUN addgroup -S appgroup && adduser -S -G appgroup appuser

# Set working directory within the builder stage
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
# This leverages Docker layer caching for dependencies if these files haven't changed.
COPY go.mod go.sum ./
RUN go mod download -x # -x for verbose output, useful for debugging
RUN go mod verify

# Copy the rest of the application source code into the /app directory
COPY . .

# Build the application
# The output binary will be named 'vaultify-backend' and placed in /app/bin/
# Using -ldflags="-w -s" to strip debug information and reduce binary size.
# Ensure the main package is correctly specified (e.g., ./cmd/server/main.go or just ./cmd/server)
RUN go build -ldflags="-w -s" -o /app/bin/vaultify-backend ./cmd/server

# Stage 2: Create the final lightweight production image
FROM alpine:latest

# Import the user and group from the builder stage
# This ensures the same UID/GID for the appuser
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Set the user for running the application (non-root)
USER appuser

# Set working directory in the final image
WORKDIR /app

# Copy the built binary from the builder stage to the final image
COPY --from=builder /app/bin/vaultify-backend /app/vaultify-backend

# Copy .env.example for reference.
# Actual .env files or environment variables should be provided at runtime.
# If .env.example contains sensitive defaults, consider not copying it.
COPY .env.example .env.example

# Expose the port the application runs on.
# This is documentation for Docker; the application itself needs to listen on this port,
# typically configured via the PORT environment variable.
EXPOSE 8080

# Command to run the application when the container starts.
# The application binary is the entrypoint.
ENTRYPOINT ["/app/vaultify-backend"]
```
