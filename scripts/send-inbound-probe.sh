#!/usr/bin/env zsh
set -euo pipefail

: "${SERVER_URL:?SERVER_URL is required}"
: "${AUTHENTICATION_API_KEY:?AUTHENTICATION_API_KEY is required}"

INSTANCE_NAME="${1:-connect_b_20260505}"
TARGET_NUMBER="${2:-5519989411105}"
MESSAGE_TEXT="${3:-disparo inbound automatico $(date +%Y-%m-%dT%H:%M:%S)}"

curl -fsS -X POST "${SERVER_URL}/message/sendText/${INSTANCE_NAME}" \
  -H 'Content-Type: application/json' \
  -H "apikey: ${AUTHENTICATION_API_KEY}" \
  --data "{\"number\":\"${TARGET_NUMBER}\",\"text\":\"${MESSAGE_TEXT}\",\"delay\":1000}"
echo
