#!/bin/bash

# Define output directory (no longer needed for host-based binary, but kept for consistency if needed later)
OUTPUT_DIR="bin"
mkdir -p "$OUTPUT_DIR"

echo "Skipping host-based Go binary build for Docker target."

# --- Docker Image Build Section ---
echo "--- Building Docker Image ---"

IMAGE_NAME="beango-local:latest"

echo "Building Docker image: $IMAGE_NAME"
docker build -t "$IMAGE_NAME" .

if [ $? -ne 0 ]; then
  echo "Error building Docker image. Exiting."
  exit 1
fi

echo "Docker image $IMAGE_NAME built successfully."
echo "You can now use this image in your docker-compose.yaml as 'beango-local:latest'."
