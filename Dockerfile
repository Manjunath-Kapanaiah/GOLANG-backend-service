# syntax=docker/dockerfile:1.4

################################################################
# Multi-stage Dockerfile for GOLANG-backend-service
#
# Stages:
#   linter  → golangci-lint static analysis (skippable)
#   test    → go test ./...
#   deps    → module download cache
#   builder → compile static binary
#   runtime → minimal distroless image (production)
#
# Build full (all stages):
#   docker build -t go-backend-service:latest .
#
# Build skipping linter (faster, useful on first EC2 run):
#   docker build --build-arg SKIP_LINT=true -t go-backend-service:latest .
#
# Run:
#   docker run -d -p 8081:8081 --name go-service go-backend-service:latest
################################################################

# -------- Common build args --------
ARG GO_VERSION=1.21
ARG APP_NAME=go-backend-service
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG CGO_ENABLED=0

# -------- Linter stage --------
FROM golangci/golangci-lint:v1.59.0 AS linter
WORKDIR /src
COPY . .
# Run linter — if this fails, fix lint errors or build with --target builder to skip
RUN golangci-lint run --timeout=120s ./...

# -------- Test stage --------
FROM golang:${GO_VERSION}-alpine AS test
WORKDIR /src

# Copy module files first (layer caching — only re-downloads if go.mod changes)
COPY go.mod ./
# Copy go.sum only if it exists (handles projects with no external dependencies)
COPY go.m* ./

RUN go env -w GOPROXY=https://proxy.golang.org,direct \
    && go mod download \
    && go mod verify

# Copy source and run tests
COPY . .
RUN go test ./... -v -count=1

# -------- Deps cache stage --------
FROM golang:${GO_VERSION}-alpine AS deps
WORKDIR /src
COPY go.mod ./
COPY go.m* ./
RUN go env -w GOPROXY=https://proxy.golang.org,direct \
    && go mod download \
    && go mod verify

# -------- Builder stage --------
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

# Install CA certs and git (needed for some go tools)
RUN apk add --no-cache ca-certificates git

# Reuse module cache from deps stage
COPY --from=deps /go/pkg/mod /go/pkg/mod
COPY --from=deps /root/.cache /root/.cache

# Copy source
COPY . .

# Build static, stripped binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /out/go-backend-service \
    .

# -------- Runtime stage --------
# distroless/static: no shell, no package manager, minimal attack surface
FROM gcr.io/distroless/static:nonroot AS runtime

# Copy the compiled binary
COPY --from=builder /out/go-backend-service /usr/local/bin/go-backend-service

# Copy CA certificates for HTTPS calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Run as non-root (UID 65532 = nonroot user in distroless)
USER nonroot:nonroot

# Expose service port
EXPOSE 8081

# Health check — Docker will report container status
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/go-backend-service"]

ENTRYPOINT ["/usr/local/bin/go-backend-service"]
