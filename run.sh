#!/bin/bash
set -e

# Build and run in managed mode (handles Docker containers automatically)
echo "Building benchmark binary..."
go build -o bin/benchmark ./cmd/benchmark

exec ./bin/benchmark -managed "$@"
