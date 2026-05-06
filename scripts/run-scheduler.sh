#!/usr/bin/env zsh
set -euo pipefail

cd "$(dirname "$0")/.."

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${RABBITMQ_URL:?RABBITMQ_URL is required}"

export SCHEDULER_ENABLED="${SCHEDULER_ENABLED:-true}"
export SCHEDULER_TICK_SECONDS="${SCHEDULER_TICK_SECONDS:-5}"

exec go run ./cmd/scheduler
