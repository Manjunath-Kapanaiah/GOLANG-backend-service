# GOLANG Backend Service

A lightweight HTTP backend service written in Go, containerized with a multi-stage Dockerfile and deployable on AWS EC2.

## Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | /health | Service health, version, uptime |
| GET | /products | List all products |
| POST | /products | Create a product |
| GET | /products/{id} | Get product by ID |
| DELETE | /products/{id} | Delete product by ID |

## Project Structure

```
GOLANG-backend-service/
├── main.go           # HTTP server and business logic
├── main_test.go      # Unit tests
├── go.mod            # Go module definition
├── go.sum            # Dependency checksums
├── Dockerfile        # Multi-stage Docker build
├── .dockerignore     # Files excluded from Docker build context
├── setup-ec2.sh      # One-shot EC2 setup script
└── README.md
```

## Run Locally

```bash
go run main.go
# Service starts on http://localhost:8081
```

## Run Tests

```bash
go test ./... -v
```

## Docker Build & Run

Build the image:
```bash
docker build -t go-backend-service:latest .
```

Run the container:
```bash
docker run -d -p 8081:8081 --name go-service go-backend-service:latest
```

Test it:
```bash
curl http://localhost:8081/health
curl http://localhost:8081/products
```

## Deploy on AWS EC2

1. Launch an EC2 instance (Ubuntu 22.04 recommended, t2.micro or above)
2. Open port 8081 in your EC2 Security Group (inbound rule)
3. SSH into your EC2 instance:
   ```bash
   ssh -i your-key.pem ubuntu@<your-ec2-ip>
   ```
4. Run the setup script:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/Manjunath-Kapanaiah/GOLANG-backend-service/main/setup-ec2.sh | bash
   ```
   Or manually:
   ```bash
   git clone https://github.com/Manjunath-Kapanaiah/GOLANG-backend-service.git
   cd GOLANG-backend-service
   bash setup-ec2.sh
   ```
5. Access the service:
   ```
   http://<your-ec2-public-ip>:8081/health
   ```

## Docker Stages

| Stage | Base Image | Purpose |
|---|---|---|
| linter | golangci/golangci-lint:v1.59.0 | Static code analysis |
| test | golang:1.21-alpine | Run unit tests |
| deps | golang:1.21-alpine | Download and cache modules |
| builder | golang:1.21-alpine | Compile static binary |
| runtime | distroless/static:nonroot | Minimal production image |

## Tech Stack

Go 1.21 | Docker | AWS EC2 | distroless
