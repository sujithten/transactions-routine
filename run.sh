#!/usr/bin/env bash
set -euo pipefail

echo "Starting Transactions Routine..."
docker compose up --build "$@"
