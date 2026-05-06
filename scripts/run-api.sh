#!/usr/bin/env zsh
set -euo pipefail

cd "$(dirname "$0")/.."

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${SERVER_URL:?SERVER_URL is required}"
: "${AUTHENTICATION_API_KEY:?AUTHENTICATION_API_KEY is required}"

export APP_PORT="${APP_PORT:-8081}"
export APP_PUBLIC_URL="${APP_PUBLIC_URL:-http://127.0.0.1:${APP_PORT}}"
export API_AUTH_ENABLED="${API_AUTH_ENABLED:-true}"
export SUPABASE_URL="${SUPABASE_URL:-https://rxdophybnwoocsdyxyjm.supabase.co}"

exec go run ./cmd/api
