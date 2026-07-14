#!/bin/bash
# ============================================================
# EC2 setup + Docker run script for GOLANG-backend-service
# Run this on a fresh Ubuntu EC2 instance
# Usage: bash setup-ec2.sh
# ============================================================

set -e  # exit on any error

APP_NAME="go-backend-service"
IMAGE_TAG="latest"
PORT=8081
REPO_URL="https://github.com/Manjunath-Kapanaiah/GOLANG-backend-service.git"

echo ""
echo "=================================================="
echo " GOLANG Backend Service - EC2 Setup"
echo "=================================================="
echo ""

# ── Step 1: Update system ──────────────────────────────────
echo "[1/7] Updating system packages..."
sudo apt-get update -y
sudo apt-get upgrade -y

# ── Step 2: Install Docker ─────────────────────────────────
echo "[2/7] Installing Docker..."
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    git

# Add Docker's official GPG key
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
    sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# Add Docker repo
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu \
  $(lsb_release -cs) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get update -y
sudo apt-get install -y docker-ce docker-ce-cli containerd.io

# ── Step 3: Start Docker service ───────────────────────────
echo "[3/7] Starting Docker service..."
sudo systemctl start docker
sudo systemctl enable docker

# Add current user to docker group (avoids sudo for docker commands)
sudo usermod -aG docker "$USER"
echo "Note: Log out and back in for docker group to take effect, OR use 'newgrp docker' now"

# ── Step 4: Clone repo ─────────────────────────────────────
echo "[4/7] Cloning repository..."
if [ -d "$APP_NAME" ]; then
    echo "Repo already exists — pulling latest..."
    cd "$APP_NAME" && git pull origin main
else
    git clone "$REPO_URL"
    cd "$APP_NAME"
fi

# ── Step 5: Build Docker image ─────────────────────────────
echo "[5/7] Building Docker image (this may take a few minutes)..."
echo ""
echo "Building WITHOUT linter stage first (faster and safer for EC2):"
sudo docker build \
    --target runtime \
    -t "$APP_NAME:$IMAGE_TAG" \
    .

echo ""
echo "Image built successfully:"
sudo docker images | grep "$APP_NAME"

# ── Step 6: Run container ──────────────────────────────────
echo "[6/7] Starting container..."

# Stop and remove if already running
sudo docker stop "$APP_NAME" 2>/dev/null || true
sudo docker rm "$APP_NAME" 2>/dev/null || true

sudo docker run -d \
    --name "$APP_NAME" \
    --restart unless-stopped \
    -p "$PORT:$PORT" \
    -e PORT="$PORT" \
    "$APP_NAME:$IMAGE_TAG"

echo "Container started:"
sudo docker ps | grep "$APP_NAME"

# ── Step 7: Smoke test ─────────────────────────────────────
echo "[7/7] Running smoke test..."
sleep 3  # give the service a moment to start

echo ""
echo "Health check:"
curl -s http://localhost:"$PORT"/health | python3 -m json.tool 2>/dev/null || \
    curl -s http://localhost:"$PORT"/health

echo ""
echo "Products list:"
curl -s http://localhost:"$PORT"/products | python3 -m json.tool 2>/dev/null || \
    curl -s http://localhost:"$PORT"/products

echo ""
echo "=================================================="
echo " Setup complete!"
echo " Service running at: http://$(curl -s ifconfig.me):${PORT}"
echo " Health:    GET http://localhost:${PORT}/health"
echo " Products:  GET http://localhost:${PORT}/products"
echo "=================================================="
