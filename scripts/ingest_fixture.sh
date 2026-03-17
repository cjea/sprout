#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-var/sprout.db}"
USER_ID="${USER_ID:-demo}"
FIXTURE_PATH="${FIXTURE_PATH:-fixtures/scotus/24-777_9ol1.pdf}"
FIXTURE_URL="${FIXTURE_URL:-https://www.supremecourt.gov/opinions/25pdf/24-777_9ol1.pdf}"

GOCACHE="${GOCACHE:-$(pwd)/.gocache}" go run ./cmd/sprout ingest-file \
  --db "$DB_PATH" \
  --user "$USER_ID" \
  --file "$FIXTURE_PATH" \
  --url "$FIXTURE_URL"
