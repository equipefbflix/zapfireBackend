#!/usr/bin/env zsh
set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"

INSTANCE_NAME="${1:-reactive_b_go_20260515}"
TARGET_NUMBER="${2:-5519989411105}"
MESSAGE_TEXT="${3:-disparo inbound automatico $(date +%Y-%m-%dT%H:%M:%S)}"

tmp_go_file="$(mktemp /tmp/send-inbound-probe.XXXXXX.go)"
trap 'rm -f "$tmp_go_file"' EXIT

cat <<'EOF' >"$tmp_go_file"
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	instanceName := os.Getenv("INSTANCE_NAME")
	databaseURL := os.Getenv("DATABASE_URL")

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	var baseURL string
	var secretName string
	err = pool.QueryRow(context.Background(), `
select es.base_url, coalesce(i.instance_api_key_secret_name, '')
from public.instances i
join public.evolution_servers es on es.id = i.evolution_server_id
where i.instance_name = $1
`, instanceName).Scan(&baseURL, &secretName)
	if err != nil {
		panic(err)
	}

	apiKey := secretName
	if strings.HasPrefix(apiKey, "literal:") {
		apiKey = strings.TrimPrefix(apiKey, "literal:")
	} else {
		apiKey = os.Getenv(apiKey)
	}
	if strings.TrimSpace(apiKey) == "" {
		panic("instance api key is empty")
	}

	fmt.Printf("%s\n%s\n", strings.TrimRight(baseURL, "/"), apiKey)
}
EOF

lookup="$(INSTANCE_NAME="$INSTANCE_NAME" DATABASE_URL="$DATABASE_URL" go run "$tmp_go_file")"

BASE_URL="$(printf '%s\n' "$lookup" | sed -n '1p')"
INSTANCE_API_KEY="$(printf '%s\n' "$lookup" | sed -n '2p')"

curl --connect-timeout 10 --max-time 30 -fsS -X POST "${BASE_URL}/send/text" \
  -H 'Content-Type: application/json' \
  -H "apikey: ${INSTANCE_API_KEY}" \
  --data "{\"number\":\"${TARGET_NUMBER}\",\"text\":\"${MESSAGE_TEXT}\",\"delay\":1000}"
echo
