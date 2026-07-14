# syntax=docker/dockerfile:1.4

################################################################
# Multi-stage Dockerfile for GOLANG-backend-service
#
# Stages:
#   test    -> run unit tests
#   builder -> compile static binary
#   runtime -> minimal distroless production image
#
# Build:
#   docker build -t go-backend-service:latest .
#
# Run:
#   docker run -d -p 8081:8081 --name go-backend-service go-backend-service:latest
################################################################

ARG GO_VERSION=1.21

# -------- Test stage --------
FROM golang:${GO_VERSION}-alpine AS test
WORKDIR /src

# Install CA certs
RUN apk add --no-cache ca-certificates

# Copy module files
COPY go.mod ./

# Copy go.sum if it exists (empty for stdlib-only projects)
COPY go.sum* ./

# Copy all Go source files
COPY *.go ./

# Run tests
RUN go test ./... -v -count=1

# -------- Builder stage --------
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

# Install CA certs and git
RUN apk add --no-cache ca-certificates git

# Copy module files
COPY go.mod ./
COPY go.sum* ./

# Copy all Go source files
COPY *.go ./

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /out/go-backend-service \
    .

# -------- Runtime stage --------
FROM gcr.io/distroless/static:nonroot AS runtime

# Copy binary from builder
COPY --from=builder /out/go-backend-service /usr/local/bin/go-backend-service

# Copy CA certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Run as non-root user
USER nonroot:nonroot

# Expose port
EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/go-backend-service"]
