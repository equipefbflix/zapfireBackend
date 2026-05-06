#!/usr/bin/env zsh
set -euo pipefail

cd "$(dirname "$0")/.."

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${RABBITMQ_URL:?RABBITMQ_URL is required}"
: "${AUTHENTICATION_API_KEY:?AUTHENTICATION_API_KEY is required}"

exec go run ./cmd/worker
