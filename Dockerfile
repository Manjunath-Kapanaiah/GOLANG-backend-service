# syntax=docker/dockerfile:1.4
################################################################
# Multi-stage Dockerfile for Manjunath-Kapanaiah/GOLANG-backend-service
# Stages:
#  - linter  : run golangci-lint (fail build on issues)
#  - test    : run go test ./...
#  - deps    : download/caches go modules
#  - builder : build static, stripped binary
#  - runtime : minimal image (distroless) containing only binary
################################################################

# -------- common args --------
ARG GO_VERSION=1.20
ARG APP_NAME=myapp
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG CGO_ENABLED=0

# -------- Linter stage --------
# Uses the official golangci-lint image to run lint checks.
FROM golangci/golangci-lint:v1.59.0 AS linter
WORKDIR /src
# Copy everything required for linting
COPY . .
# Run linter (will fail the build if issues found)
RUN golangci-lint run ./...

# -------- Test stage --------
FROM golang:${GO_VERSION} AS test
WORKDIR /src
# Copy go.mod/go.sum and download modules first for cache
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org,direct && go mod download
# Copy source and run unit tests
COPY . .
RUN go test ./... -v

# -------- Deps (module cache) --------
FROM golang:${GO_VERSION} AS deps
WORKDIR /src
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org,direct \
    && go mod download

# -------- Builder stage --------
FROM golang:${GO_VERSION} AS builder
WORKDIR /src
# Reuse module cache from deps stage to speed up builds
COPY --from=deps /go/pkg/mod /go/pkg/mod
# Copy source code
COPY . .
# Install CA certs in builder only (so final image can have them copied)
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
# Build static, trimmed binary
ENV CGO_ENABLED=${CGO_ENABLED}
RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/${APP_NAME} .

# -------- Runtime stage --------
FROM gcr.io/distroless/static:nonroot AS runtime
# Copy only the compiled binary and CA certs from the builder stage
COPY --from=builder /out/myapp /usr/local/bin/myapp
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/myapp"]
